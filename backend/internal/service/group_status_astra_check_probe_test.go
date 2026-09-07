package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------- 桩 ----------

type groupStatusAstraRepo struct {
	GroupStatusRepository
	state   *GroupStatusState
	results []*GroupStatusAstraCheckResult
	events  []*GroupStatusEvent
}

func (r *groupStatusAstraRepo) SaveAstraCheckRun(_ context.Context, result *GroupStatusAstraCheckResult) (*GroupStatusAstraCheckRun, *GroupStatusState, *GroupStatusEvent, error) {
	copied := *result
	r.results = append(r.results, &copied)
	runID := int64(len(r.results))
	next, event := ComputeAstraCheckTransition(r.state, &copied, runID)
	r.state = next
	if event != nil {
		r.events = append(r.events, event)
	}
	stateCopy := *next
	run := &GroupStatusAstraCheckRun{
		ID:                runID,
		GroupID:           copied.GroupID,
		ConfigID:          copied.ConfigID,
		Verdict:           copied.Verdict,
		Winner:            copied.Winner,
		RequestsPlanned:   copied.RequestsPlanned,
		RequestsCompleted: copied.RequestsCompleted,
	}
	return run, &stateCopy, event, nil
}

type staticAstraBenchmarkProvider struct {
	bench *AstraBenchmark
	meta  *AstraBenchmarkMeta
}

func (p staticAstraBenchmarkProvider) Active() (*AstraBenchmark, *AstraBenchmarkMeta, error) {
	return p.bench, p.meta, nil
}

func newAstraCheckProbeFixture(t *testing.T, responses ...*http.Response) (*GroupStatusProbeService, *Group, *GroupStatusConfig, *groupStatusAstraRepo, *groupStatusProbeHTTPUpstream, *recordingGroupStatusNotifier) {
	t.Helper()
	group := &Group{ID: 31, Name: "Astra 高速", Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}
	accounts := []Account{
		groupStatusProbeAccount(1, PlatformOpenAI, group.ID, 1, map[string]any{"gpt-6-astra": "gpt-6-astra"}),
		groupStatusProbeAccount(2, PlatformOpenAI, group.ID, 2, map[string]any{"gpt-6-astra": "gpt-6-astra"}),
	}
	for i := range accounts {
		accounts[i].Credentials["api_key"] = "sk-test"
	}
	upstream := &groupStatusProbeHTTPUpstream{responses: responses}
	svc := newGroupStatusProbeServiceForTest(group, accounts, nil, upstream, nil)
	repo := &groupStatusAstraRepo{}
	svc.repo = repo
	notifier := &recordingGroupStatusNotifier{}
	svc.SetTransitionNotifier(notifier)

	// 测试桩按顺序弹出响应且无锁，串行执行；重试退避不等待
	svc.astraConcurrency = 1
	svc.astraSleep = func(context.Context, time.Duration) error { return nil }
	bench, meta := loadSyntheticAstraBenchmark(t)
	svc.SetAstraBenchmarkProvider(staticAstraBenchmarkProvider{bench: bench, meta: meta})

	cfg := groupStatusProbeConfig(group.ID, "gpt-6-astra")
	cfg.AstraCheckEnabled = true
	cfg.AstraCheckRequestModel = "gpt-6-astra"
	cfg.AstraCheckTier = AstraCheckTierLow
	cfg.AstraCheckIntervalSeconds = 3600
	return svc, group, cfg, repo, upstream, notifier
}

func astraSSE(answer string) *http.Response {
	return groupStatusProbeResponse(200, solJuiceSSEBody(answer, 10))
}

// 合成包低档任务顺序：country, strawberry, country, strawberry
func astraLikeResponses() []*http.Response {
	return []*http.Response{astraSSE("Japan"), astraSSE("3"), astraSSE("japan"), astraSSE("3")}
}

func solLikeResponses() []*http.Response {
	return []*http.Response{astraSSE("Brazil"), astraSSE("4"), astraSSE("Brazil"), astraSSE("4")}
}

// ---------- 用例 ----------

func TestAstraCheckProbe_MatchOnAstraLikeAnswers(t *testing.T) {
	svc, group, cfg, repo, upstream, notifier := newAstraCheckProbeFixture(t, astraLikeResponses()...)

	execution, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	require.NotNil(t, execution.Result)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, "gpt-6-astra", execution.Result.Winner)
	require.Empty(t, execution.Result.Reasons)
	require.Equal(t, 4, execution.Result.RequestsPlanned)
	require.Equal(t, 4, execution.Result.RequestsCompleted)
	require.Equal(t, 4, execution.Result.ValidSamples)
	require.Equal(t, 4, execution.Result.PlannedSamples)
	require.Equal(t, int64(208), execution.Result.InputTokens)
	require.Equal(t, int64(52), execution.Result.OutputTokens)
	require.Equal(t, int64(40), execution.Result.ReasoningTokens)
	require.Equal(t, "synthetic-astra", execution.Result.BenchmarkPackageID)
	require.Equal(t, "test-1", execution.Result.BenchmarkVersion)
	require.Equal(t, AstraCheckTierLow, execution.Result.Tier)
	require.Equal(t, "", execution.Result.ErrorDetail)
	require.NotNil(t, execution.Account)
	require.Equal(t, int64(1), execution.Account.ID)
	require.NotNil(t, execution.Result.AccountID)
	require.Equal(t, int64(1), *execution.Result.AccountID)
	require.NotNil(t, execution.Run)
	require.Equal(t, int64(1), execution.Run.ID)
	require.Equal(t, AstraCheckStatusPass, execution.State.AstraCheckStableStatus)
	require.Equal(t, AstraCheckVerdictMatch, execution.State.AstraCheckVerdict)
	require.Nil(t, execution.Event)
	require.Empty(t, repo.events)
	require.Equal(t, 0, notifier.calls)
	require.False(t, svc.IsAstraCheckRunning(group.ID))

	require.Len(t, upstream.requests, 4)
	req := upstream.requests[0]
	require.Equal(t, "https://example.com/responses", req.URL.String())
	require.Equal(t, "Bearer sk-test", req.Header.Get("Authorization"))
	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, "gpt-6-astra", payload["model"])
	require.Equal(t, map[string]any{"effort": "low"}, payload["reasoning"])
	require.Equal(t, float64(128), payload["max_output_tokens"])
	require.Equal(t, false, payload["store"])
	require.Equal(t, true, payload["stream"])
	require.NotContains(t, payload, "instructions")
	input, ok := payload["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 2)
	system := input[0].(map[string]any)
	require.Equal(t, "system", system["role"])
	require.Equal(t, ".", system["content"].([]any)[0].(map[string]any)["text"])
	user := input[1].(map[string]any)
	require.Equal(t, "user", user["role"])
	require.Equal(t, "Name a random country. Reply with only the name.", user["content"].([]any)[0].(map[string]any)["text"])

	// 第二个请求是 strawberry 题
	second := decodeProbeRequestBody(t, upstream.requests[1])
	secondInput := second["input"].([]any)
	require.Equal(t, "How many r are in strawberry? Reply with a number only.", secondInput[1].(map[string]any)["content"].([]any)[0].(map[string]any)["text"])
}

func TestAstraCheckProbe_MismatchConfirmedByImmediateRecheck(t *testing.T) {
	responses := append(solLikeResponses(), solLikeResponses()...)
	svc, group, cfg, repo, upstream, notifier := newAstraCheckProbeFixture(t, responses...)

	execution, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.NoError(t, err)
	require.True(t, execution.Confirmed)
	require.Len(t, upstream.requests, 8)
	require.Len(t, repo.results, 2)
	require.Equal(t, AstraCheckVerdictMismatch, execution.Result.Verdict)
	require.Equal(t, "gpt-5.6-sol", execution.Result.Winner)
	require.Equal(t, AstraCheckStatusMismatch, execution.State.AstraCheckStableStatus)
	require.Equal(t, 2, execution.State.AstraCheckConsecutiveMismatch)
	require.Equal(t, "gpt-5.6-sol", execution.State.AstraCheckWinner)
	require.NotNil(t, execution.Event)
	require.Equal(t, GroupStatusEventAstraMismatch, execution.Event.EventType)
	require.Equal(t, "winner_gpt-5.6-sol", execution.Event.SubStatus)
	require.Equal(t, AstraCheckStatusMismatch, execution.Event.ToStatus)
	require.Contains(t, execution.Event.ErrorDetail, "Sol 0.9")
	require.Len(t, repo.events, 1)
	require.Equal(t, 1, notifier.calls)
	require.Same(t, group, notifier.groups[0])
	require.Same(t, cfg, notifier.cfgs[0])
	require.Equal(t, GroupStatusEventAstraMismatch, notifier.events[0].EventType)
}

func TestAstraCheckProbe_MismatchThenMatchDoesNotAlert(t *testing.T) {
	responses := append(solLikeResponses(), astraLikeResponses()...)
	svc, group, cfg, repo, upstream, notifier := newAstraCheckProbeFixture(t, responses...)

	execution, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.NoError(t, err)
	require.True(t, execution.Confirmed)
	require.Len(t, upstream.requests, 8)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, AstraCheckStatusPass, execution.State.AstraCheckStableStatus)
	require.Equal(t, 0, execution.State.AstraCheckConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Empty(t, repo.events)
	require.Equal(t, 0, notifier.calls)
}

func TestAstraCheckProbe_AlreadyMismatchDoesNotRecheck(t *testing.T) {
	svc, group, cfg, _, upstream, notifier := newAstraCheckProbeFixture(t, solLikeResponses()...)
	repo := svc.repo.(*groupStatusAstraRepo)
	repo.state = &GroupStatusState{GroupID: group.ID, ConfigID: cfg.ID, AstraCheckStableStatus: AstraCheckStatusMismatch, AstraCheckConsecutiveMismatch: 2}

	execution, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	require.Len(t, upstream.requests, 4)
	require.Equal(t, 3, execution.State.AstraCheckConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Equal(t, 0, notifier.calls)
}

func TestAstraCheckProbe_InvalidAnswersRetryThenInsufficient(t *testing.T) {
	// 4 个任务 × 3 次尝试，全部空答案
	responses := make([]*http.Response, 0, 12)
	for i := 0; i < 12; i++ {
		responses = append(responses, astraSSE(""))
	}
	svc, group, cfg, repo, upstream, notifier := newAstraCheckProbeFixture(t, responses...)

	execution, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	require.Len(t, upstream.requests, 12)
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, "", execution.Result.Winner)
	require.Contains(t, execution.Result.Reasons, "samples_incomplete")
	require.Equal(t, 4, execution.Result.RequestsCompleted)
	require.Equal(t, 0, execution.Result.ValidSamples)
	require.Equal(t, 4, execution.Result.PlannedSamples)
	require.Len(t, execution.Result.Matches, 2)
	require.Equal(t, "", execution.State.AstraCheckStableStatus)
	require.Equal(t, 0, execution.State.AstraCheckConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Empty(t, repo.events)
	require.Equal(t, 0, notifier.calls)
}

func TestAstraCheckProbe_FailsOverToNextAccountAfterRepeated429(t *testing.T) {
	responses := []*http.Response{
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
	}
	responses = append(responses, astraLikeResponses()...)
	svc, group, cfg, _, upstream, _ := newAstraCheckProbeFixture(t, responses...)

	execution, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Len(t, upstream.requests, 7)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.NotNil(t, execution.Account)
	require.Equal(t, int64(2), execution.Account.ID)
	require.Equal(t, 4, execution.Result.RequestsCompleted)
	require.Contains(t, execution.Result.ErrorDetail, "account 1")
	require.Contains(t, execution.Result.ErrorDetail, "429")
	require.Equal(t, AstraCheckStatusPass, execution.State.AstraCheckStableStatus)
}

func TestAstraCheckProbe_AllAccountsFailingYieldsInsufficient(t *testing.T) {
	responses := make([]*http.Response, 0, 6)
	for i := 0; i < 6; i++ {
		responses = append(responses, groupStatusProbeResponse(503, `{"error":{"message":"down"}}`))
	}
	svc, group, cfg, repo, _, notifier := newAstraCheckProbeFixture(t, responses...)

	execution, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, 0, execution.Result.RequestsCompleted)
	require.NotNil(t, execution.Result.HTTPCode)
	require.Equal(t, 503, *execution.Result.HTTPCode)
	require.Nil(t, execution.Account)
	require.Nil(t, execution.Event)
	require.Empty(t, repo.events)
	require.Equal(t, 0, notifier.calls)
}

func TestAstraCheckProbe_RejectsNonOpenAIGroup(t *testing.T) {
	svc, _, cfg, _, upstream, _ := newAstraCheckProbeFixture(t)
	group := &Group{ID: 32, Name: "Claude", Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}

	_, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.ErrorIs(t, err, ErrGroupStatusAstraCheckUnsupported)
	require.Empty(t, upstream.requests)
}

func TestAstraCheckProbe_RejectsWhenAlreadyRunning(t *testing.T) {
	svc, group, cfg, _, upstream, _ := newAstraCheckProbeFixture(t, astraLikeResponses()...)
	require.True(t, svc.markAstraCheckRunning(group.ID))
	require.True(t, svc.IsAstraCheckRunning(group.ID))

	_, err := svc.probeAstraCheck(context.Background(), group, cfg)
	require.ErrorIs(t, err, ErrGroupStatusAstraCheckRunning)
	require.Empty(t, upstream.requests)
	require.ErrorIs(t, svc.StartAstraCheckAsync(group.ID), ErrGroupStatusAstraCheckRunning)

	svc.clearAstraCheckRunning(group.ID)
	require.False(t, svc.IsAstraCheckRunning(group.ID))
}

func TestAstraCheckProbe_ProbeWithConfigLoadsGroup(t *testing.T) {
	svc, _, cfg, _, upstream, _ := newAstraCheckProbeFixture(t, astraLikeResponses()...)

	execution, err := svc.ProbeAstraCheckWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Len(t, upstream.requests, 4)

	_, err = svc.ProbeAstraCheckWithConfig(context.Background(), nil)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
}

func TestCreateOpenAIAstraCheckPayload_OAuthUsesInstructions(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)
	cell := bench.CellIndex["country_low"]

	payload := createOpenAIAstraCheckPayload("gpt-6-astra", cell, true)
	require.Equal(t, ".", payload["instructions"])
	require.Equal(t, []string{"reasoning.encrypted_content"}, payload["include"])
	input := payload["input"].([]map[string]any)
	require.Len(t, input, 1)
	require.Equal(t, "user", input[0]["role"])

	apiKey := createOpenAIAstraCheckPayload("gpt-6-astra", cell, false)
	require.NotContains(t, apiKey, "instructions")
	require.Len(t, apiKey["input"].([]map[string]any), 2)

	// 缺省字段回退：空 system → "."，空 effort → low，0 上限 → 128
	bare := createOpenAIAstraCheckPayload("gpt-6-astra", &AstraBenchmarkCell{ID: "x", Prompt: "hi"}, true)
	require.Equal(t, ".", bare["instructions"])
	require.Equal(t, map[string]any{"effort": "low"}, bare["reasoning"])
	require.Equal(t, 128, bare["max_output_tokens"])
}
