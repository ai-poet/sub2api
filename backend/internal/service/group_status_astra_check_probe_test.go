package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/stretchr/testify/require"
)

// ---------- 桩 ----------

// groupStatusAstraRepo 按（分组, 预期模型）保存状态，与真实仓储的 ON CONFLICT (group_id, expected_model) 一致。
type groupStatusAstraRepo struct {
	GroupStatusRepository
	// 多模型并行检测时会并发落库
	mu      sync.Mutex
	cfg     *GroupStatusConfig
	states  map[string]*GroupStatusAstraCheckState
	results []*GroupStatusAstraCheckResult
	events  []*GroupStatusEvent
	// onSave 在每次落库时调用（测试借此在运行中途读进度）
	onSave func(result *GroupStatusAstraCheckResult)
}

func (r *groupStatusAstraRepo) GetConfig(_ context.Context, groupID int64) (*GroupStatusConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cfg == nil || r.cfg.GroupID != groupID {
		return nil, ErrGroupStatusConfigNotFound
	}
	return r.cfg, nil
}

func (r *groupStatusAstraRepo) ListAstraCheckStates(_ context.Context, groupIDs []int64) ([]GroupStatusAstraCheckState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	wanted := make(map[int64]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		wanted[id] = struct{}{}
	}
	out := []GroupStatusAstraCheckState{}
	for _, state := range r.states {
		if _, ok := wanted[state.GroupID]; ok {
			out = append(out, *state)
		}
	}
	return out, nil
}

func (r *groupStatusAstraRepo) SaveAstraCheckRun(_ context.Context, result *GroupStatusAstraCheckResult) (*GroupStatusAstraCheckRun, *GroupStatusAstraCheckState, *GroupStatusEvent, error) {
	if r.onSave != nil {
		r.onSave(result)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *result
	r.results = append(r.results, &copied)
	runID := int64(len(r.results))
	if r.states == nil {
		r.states = map[string]*GroupStatusAstraCheckState{}
	}
	next, event := ComputeAstraCheckTransition(r.states[copied.ExpectedModel], &copied, runID)
	r.states[copied.ExpectedModel] = next
	if event != nil {
		r.events = append(r.events, event)
	}
	stateCopy := *next
	run := &GroupStatusAstraCheckRun{
		ID:                runID,
		GroupID:           copied.GroupID,
		ConfigID:          copied.ConfigID,
		ExpectedModel:     copied.ExpectedModel,
		Round:             copied.Round,
		Verdict:           copied.Verdict,
		Winner:            copied.Winner,
		RequestsPlanned:   copied.RequestsPlanned,
		RequestsCompleted: copied.RequestsCompleted,
	}
	return run, &stateCopy, event, nil
}

type staticAstraRegistryProvider struct {
	reg *AstraBenchmarkRegistry
	err error
}

func (p staticAstraRegistryProvider) Registry() (*AstraBenchmarkRegistry, error) {
	return p.reg, p.err
}

func syntheticAstraRegistry(t *testing.T) *AstraBenchmarkRegistry {
	t.Helper()
	claudePkg := syntheticAstraPackage(t, "meow-claude-other-cap98-efficient", astraBenchmarkModeClaude, []string{"claude-opus-5.5", "claude-fable-5.1"}, "claude-code")
	reg, err := NewAstraBenchmarkRegistry(syntheticGPTPackage(t), claudePkg)
	require.NoError(t, err)
	return reg
}

type astraProbeFixture struct {
	svc      *GroupStatusProbeService
	group    *Group
	cfg      *GroupStatusConfig
	repo     *groupStatusAstraRepo
	upstream *groupStatusProbeHTTPUpstream
	notifier *recordingGroupStatusNotifier
}

func newAstraProbeFixture(t *testing.T, platform string, models []AstraCheckModelConfig, responses ...*http.Response) *astraProbeFixture {
	t.Helper()
	group := &Group{ID: 31, Name: "指纹", Platform: platform, Status: StatusActive, Hydrated: true}
	mapping := map[string]any{}
	for _, m := range models {
		target, _ := astraCheckTarget(m.ExpectedModel)
		requestModel := m.requestModelFor(target)
		mapping[requestModel] = requestModel
	}
	accounts := []Account{
		groupStatusProbeAccount(1, platform, group.ID, 1, mapping),
		groupStatusProbeAccount(2, platform, group.ID, 2, mapping),
	}
	for i := range accounts {
		accounts[i].Credentials["api_key"] = "sk-test"
	}
	upstream := &groupStatusProbeHTTPUpstream{responses: responses}
	svc := newGroupStatusProbeServiceForTest(group, accounts, nil, upstream, nil)
	cfg := groupStatusProbeConfig(group.ID, "probe-model")
	cfg.AstraCheckEnabled = true
	cfg.AstraCheckModels = models
	cfg.AstraCheckTier = AstraCheckTierLow
	cfg.AstraCheckIntervalSeconds = 3600
	repo := &groupStatusAstraRepo{cfg: cfg}
	svc.repo = repo
	notifier := &recordingGroupStatusNotifier{}
	svc.SetTransitionNotifier(notifier)

	// 测试桩按顺序弹出响应且无锁，串行执行；重试退避不等待
	svc.astraConcurrency = 1
	svc.astraModelParallelism = 1
	svc.astraSleep = func(context.Context, time.Duration) error { return nil }
	svc.SetAstraBenchmarkProvider(staticAstraRegistryProvider{reg: syntheticAstraRegistry(t)})
	return &astraProbeFixture{svc: svc, group: group, cfg: cfg, repo: repo, upstream: upstream, notifier: notifier}
}

func (f *astraProbeFixture) run(t *testing.T) []*GroupStatusAstraCheckExecution {
	t.Helper()
	executions, err := f.svc.probeAstraCheck(context.Background(), f.group, f.cfg, f.cfg.AstraCheckModels)
	require.NoError(t, err)
	return executions
}

func astraSSE(answer string) *http.Response {
	return groupStatusProbeResponse(200, responsesSSEBody(answer, 10))
}

// 合成包低档任务顺序：c1, c2, c1, c2；每个真实来源 / 参考源各偏好一个答案（按包内来源顺序）。
func astraAnswers(c1, c2 string) []*http.Response {
	return []*http.Response{astraSSE(c1), astraSSE(c2), astraSSE(c1), astraSSE(c2)}
}

func astraLike() []*http.Response { return astraAnswers("a", "1") }
func solLike() []*http.Response   { return astraAnswers("b", "2") }

func messagesSSE(text, stopReason string) *http.Response {
	events := []string{
		`data: {"type":"message_start","message":{"model":"claude-opus-5-5","usage":{"input_tokens":40,"output_tokens":1}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"hmm"}}`,
	}
	if text != "" {
		events = append(events, fmt.Sprintf(`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":%q}}`, text))
	}
	events = append(events,
		fmt.Sprintf(`data: {"type":"message_delta","delta":{"stop_reason":%q},"usage":{"output_tokens":6}}`, stopReason),
		`data: {"type":"message_stop"}`,
	)
	return groupStatusProbeResponse(200, strings.Join(events, "\n\n")+"\n\n")
}

func onlyModels(ids ...string) []AstraCheckModelConfig {
	out := make([]AstraCheckModelConfig, 0, len(ids))
	for _, id := range ids {
		out = append(out, AstraCheckModelConfig{ExpectedModel: id})
	}
	return out
}

// ---------- 用例 ----------

func TestAstraCheckProbe_RunsEveryConfiguredModel(t *testing.T) {
	responses := append(solLike(), astraLike()...)
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-sol", "gpt-6-astra"), responses...)
	var progress []*AstraCheckProgress
	f.repo.onSave = func(*GroupStatusAstraCheckResult) {
		snaps := f.svc.AstraCheckProgresses(f.group.ID)
		require.Len(t, snaps, 1)
		progress = append(progress, &snaps[0])
	}

	executions := f.run(t)
	require.Len(t, executions, 2)
	require.Len(t, f.upstream.requests, 8)

	sol, astra := executions[0], executions[1]
	require.Equal(t, "gpt-6-sol", sol.Result.ExpectedModel)
	require.Equal(t, AstraCheckVerdictMatch, sol.Result.Verdict)
	require.Equal(t, "gpt-6-sol", sol.Result.Winner)
	require.Equal(t, "gpt-6-astra", astra.Result.ExpectedModel)
	require.Equal(t, AstraCheckVerdictMatch, astra.Result.Verdict)
	require.Equal(t, "gpt-6-astra", astra.Result.Winner)
	for _, execution := range executions {
		require.False(t, execution.Confirmed)
		require.Equal(t, 1, execution.Result.Round)
		require.Equal(t, PlatformOpenAI, execution.Result.Platform)
		require.Equal(t, "meow-gpt-other-cap98-efficient", execution.Result.BenchmarkPackageID)
		require.Equal(t, "test-1", execution.Result.BenchmarkVersion)
		require.Equal(t, astraBenchmarkScoringVersion, execution.Result.ScoringVersion)
		require.Empty(t, execution.Result.Reasons)
		require.Equal(t, 4, execution.Result.RequestsPlanned)
		require.Equal(t, 4, execution.Result.RequestsCompleted)
		require.Equal(t, 4, execution.Result.ValidSamples)
		require.Equal(t, int64(208), execution.Result.InputTokens)
		require.Equal(t, int64(52), execution.Result.OutputTokens)
		require.Equal(t, int64(40), execution.Result.ReasoningTokens)
		require.Equal(t, "", execution.Result.ErrorDetail)
		require.Equal(t, int64(1), *execution.Result.AccountID)
		require.Equal(t, AstraCheckStatusPass, execution.State.StableStatus)
		require.Equal(t, execution.Result.ExpectedModel, execution.State.ExpectedModel)
		require.Nil(t, execution.Event)
		require.Len(t, execution.Result.Samples, 4)
	}
	require.Len(t, f.repo.states, 2)
	require.Empty(t, f.repo.events)
	require.Equal(t, 0, f.notifier.calls)
	require.False(t, f.svc.IsAstraCheckRunning(f.group.ID))
	require.Empty(t, f.svc.AstraCheckProgresses(f.group.ID))

	// 运行中途的进度标明正在检测第几个模型
	require.Len(t, progress, 2)
	require.Equal(t, "gpt-6-sol", progress[0].ExpectedModel)
	require.Equal(t, 1, progress[0].ModelIndex)
	require.Equal(t, 2, progress[0].ModelCount)
	require.Equal(t, "gpt-6-astra", progress[1].ExpectedModel)
	require.Equal(t, 2, progress[1].ModelIndex)
	require.Equal(t, 4, progress[1].Completed)

	// 逐请求样本
	first := sol.Result.Samples[0]
	require.Equal(t, "c1", first.CellID)
	require.Equal(t, "b", first.Answer)
	require.Equal(t, "b", first.Category)
	require.Equal(t, AstraCheckSampleValid, first.Outcome)
	require.Equal(t, "c2", sol.Result.Samples[1].CellID)

	// 请求契约：system 句点 + 题面、low effort、128 输出上限
	payload := decodeProbeRequestBody(t, f.upstream.requests[0])
	require.Equal(t, "gpt-6-sol", payload["model"])
	require.Equal(t, map[string]any{"effort": "low"}, payload["reasoning"])
	require.Equal(t, float64(128), payload["max_output_tokens"])
	require.Equal(t, false, payload["store"])
	require.Equal(t, true, payload["stream"])
	require.NotContains(t, payload, "instructions")
	input := jsonSlice(t, payload["input"])
	require.Len(t, input, 2)
	require.Equal(t, "system", jsonMap(t, input[0])["role"])
	require.Equal(t, ".", jsonMap(t, jsonSlice(t, jsonMap(t, input[0])["content"])[0])["text"])
	require.Equal(t, "prompt c1", jsonMap(t, jsonSlice(t, jsonMap(t, input[1])["content"])[0])["text"])
	require.Equal(t, "prompt c2", jsonMap(t, jsonSlice(t, jsonMap(t, jsonSlice(t, decodeProbeRequestBody(t, f.upstream.requests[1])["input"])[1])["content"])[0])["text"])
	require.Equal(t, "gpt-6-astra", decodeProbeRequestBody(t, f.upstream.requests[4])["model"])
}

func TestAstraCheckProbe_CustomRequestModel(t *testing.T) {
	models := []AstraCheckModelConfig{{ExpectedModel: "gpt-6-astra", RequestModel: "astra-alias"}}
	f := newAstraProbeFixture(t, PlatformOpenAI, models, astraLike()...)
	executions := f.run(t)
	require.Len(t, executions, 1)
	require.Equal(t, "astra-alias", executions[0].Result.RequestModel)
	require.Equal(t, AstraCheckVerdictMatch, executions[0].Result.Verdict)
	require.Equal(t, "astra-alias", decodeProbeRequestBody(t, f.upstream.requests[0])["model"])
}

func TestAstraCheckProbe_MismatchConfirmedOnSameAccount(t *testing.T) {
	responses := append(astraLike(), astraLike()...)
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-sol"), responses...)

	executions := f.run(t)
	require.Len(t, executions, 1)
	execution := executions[0]
	require.True(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 8)
	require.Len(t, f.repo.results, 2)
	require.Equal(t, 1, f.repo.results[0].Round)
	require.Equal(t, 2, f.repo.results[1].Round)
	require.Equal(t, *f.repo.results[0].AccountID, *f.repo.results[1].AccountID)
	require.Equal(t, AstraCheckVerdictMismatch, execution.Result.Verdict)
	require.Equal(t, "gpt-6-astra", execution.Result.Winner)
	require.Equal(t, AstraCheckStatusMismatch, execution.State.StableStatus)
	require.Equal(t, 2, execution.State.ConsecutiveMismatch)
	require.Equal(t, "gpt-6-astra", execution.State.Winner)
	require.NotNil(t, execution.Event)
	require.Equal(t, GroupStatusEventAstraMismatch, execution.Event.EventType)
	require.Equal(t, "gpt-6-sol:winner_gpt-6-astra", execution.Event.SubStatus)
	require.Equal(t, AstraCheckStatusMismatch, execution.Event.ToStatus)
	require.Contains(t, execution.Event.ErrorDetail, "expected GPT-6 Sol")
	require.Contains(t, execution.Event.ErrorDetail, "strongest GPT-6 Astra")
	require.Len(t, f.repo.events, 1)
	require.Equal(t, 1, f.notifier.calls)
	require.Same(t, f.group, f.notifier.groups[0])
	require.Same(t, f.cfg, f.notifier.cfgs[0])
}

func TestAstraCheckProbe_MismatchThenMatchDoesNotAlert(t *testing.T) {
	responses := append(astraLike(), solLike()...)
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-sol"), responses...)

	execution := f.run(t)[0]
	require.True(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 8)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, AstraCheckStatusPass, execution.State.StableStatus)
	require.Equal(t, 0, execution.State.ConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Empty(t, f.repo.events)
	require.Equal(t, 0, f.notifier.calls)
}

func TestAstraCheckProbe_AlreadyMismatchDoesNotRecheck(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-sol"), astraLike()...)
	f.repo.states = map[string]*GroupStatusAstraCheckState{
		"gpt-6-sol": {GroupID: f.group.ID, ConfigID: f.cfg.ID, ExpectedModel: "gpt-6-sol", StableStatus: AstraCheckStatusMismatch, ConsecutiveMismatch: 2},
	}

	execution := f.run(t)[0]
	require.False(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 4)
	require.Equal(t, 3, execution.State.ConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Equal(t, 0, f.notifier.calls)
}

func TestAstraCheckProbe_OneModelMismatchDoesNotTouchAnother(t *testing.T) {
	// sol 已稳定 pass；astra 这次强指向 sol：只动 astra 的状态
	responses := append(solLike(), solLike()...)
	responses = append(responses, solLike()...)
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-sol", "gpt-6-astra"), responses...)
	f.repo.states = map[string]*GroupStatusAstraCheckState{
		"gpt-6-sol": {GroupID: f.group.ID, ExpectedModel: "gpt-6-sol", StableStatus: AstraCheckStatusPass},
	}

	executions := f.run(t)
	require.Len(t, executions, 2)
	require.Equal(t, AstraCheckVerdictMatch, executions[0].Result.Verdict)
	require.True(t, executions[1].Confirmed)
	require.Equal(t, AstraCheckStatusMismatch, f.repo.states["gpt-6-astra"].StableStatus)
	require.Equal(t, AstraCheckStatusPass, f.repo.states["gpt-6-sol"].StableStatus)
	require.Equal(t, 0, f.repo.states["gpt-6-sol"].ConsecutiveMismatch)
	require.Len(t, f.repo.events, 1)
	require.Equal(t, "gpt-6-astra:winner_gpt-6-sol", f.repo.events[0].SubStatus)
}

func TestAstraCheckProbe_InvalidAnswersRetryThenInsufficient(t *testing.T) {
	// 4 个任务 × 3 次尝试，全部空答案
	responses := make([]*http.Response, 0, 12)
	for i := 0; i < 12; i++ {
		responses = append(responses, astraSSE(""))
	}
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-astra"), responses...)

	execution := f.run(t)[0]
	require.False(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 12)
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, "", execution.Result.Winner)
	require.Contains(t, execution.Result.Reasons, AstraCheckReasonSamplesIncomplete)
	require.Contains(t, execution.Result.Reasons, AstraCheckReasonNoValidSamples)
	require.Equal(t, 4, execution.Result.RequestsCompleted)
	require.Equal(t, 0, execution.Result.ValidSamples)
	require.Equal(t, 4, execution.Result.PlannedSamples)
	require.Len(t, execution.Result.Matches, 3)
	require.Equal(t, "", execution.State.StableStatus)
	require.Equal(t, 0, execution.State.ConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Equal(t, 0, f.notifier.calls)

	// 每次尝试都留样本：前两次非最终、第三次最终且为无效
	require.Len(t, execution.Result.Samples, 12)
	require.Equal(t, 3, execution.Result.Samples[2].Attempt)
	require.True(t, execution.Result.Samples[2].Final)
	require.False(t, execution.Result.Samples[1].Final)
	require.Equal(t, AstraCheckSampleInvalid, execution.Result.Samples[2].Outcome)
}

func TestAstraCheckProbe_FailsOverToNextAccountAfterRepeated429(t *testing.T) {
	responses := []*http.Response{
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
	}
	responses = append(responses, astraLike()...)
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-astra"), responses...)

	execution := f.run(t)[0]
	require.Len(t, f.upstream.requests, 7)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, int64(2), execution.Account.ID)
	require.Equal(t, 4, execution.Result.RequestsCompleted)
	require.Contains(t, execution.Result.ErrorDetail, "account 1")
	require.Contains(t, execution.Result.ErrorDetail, "429")
	require.Equal(t, AstraCheckStatusPass, execution.State.StableStatus)

	// 换号前的 3 次 429 也留在样本里，方便回看
	require.Len(t, execution.Result.Samples, 7)
	require.Equal(t, AstraCheckSampleFailed, execution.Result.Samples[0].Outcome)
	require.Equal(t, 429, *execution.Result.Samples[0].HTTPCode)
	require.Equal(t, AstraCheckSampleValid, execution.Result.Samples[3].Outcome)
}

func TestAstraCheckProbe_AllAccountsFailingYieldsInsufficient(t *testing.T) {
	responses := make([]*http.Response, 0, 6)
	for i := 0; i < 6; i++ {
		responses = append(responses, groupStatusProbeResponse(503, `{"error":{"message":"down"}}`))
	}
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-astra"), responses...)

	execution := f.run(t)[0]
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, []string{AstraCheckReasonNoAccount}, execution.Result.Reasons)
	require.Equal(t, 0, execution.Result.RequestsCompleted)
	require.Equal(t, 503, *execution.Result.HTTPCode)
	require.Nil(t, execution.Account)
	require.Nil(t, execution.Event)
	require.Equal(t, 0, f.notifier.calls)
}

func TestAstraCheckProbe_AnthropicAPIKeyUsesClaudeCodeContract(t *testing.T) {
	responses := []*http.Response{
		messagesSSE("b", "max_tokens"), // 第一次被截断：不投票，重试
		messagesSSE(" B ", "end_turn"),
		messagesSSE("2", "end_turn"),
		messagesSSE("b", "end_turn"),
		messagesSSE("2", "end_turn"),
	}
	f := newAstraProbeFixture(t, PlatformAnthropic, onlyModels("claude-fable-5.1"), responses...)

	execution := f.run(t)[0]
	require.Len(t, f.upstream.requests, 5)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, "claude-fable-5.1", execution.Result.Winner)
	require.Equal(t, "claude-fable-5-1", execution.Result.RequestModel)
	require.Equal(t, "meow-claude-other-cap98-efficient", execution.Result.BenchmarkPackageID)
	require.Equal(t, PlatformAnthropic, execution.Result.Platform)
	require.Equal(t, 4, execution.Result.ValidSamples)
	require.Equal(t, int64(200), execution.Result.InputTokens)
	require.Equal(t, int64(30), execution.Result.OutputTokens)
	require.Equal(t, AstraCheckSampleInvalid, execution.Result.Samples[0].Outcome)
	require.Equal(t, "truncated at max_tokens", execution.Result.Samples[0].Error)
	require.Equal(t, "b", execution.Result.Samples[1].Category)

	req := f.upstream.requests[0]
	require.Equal(t, "https://example.com/v1/messages?beta=true", req.URL.String())
	require.Equal(t, "sk-test", req.Header.Get("x-api-key"))
	require.Equal(t, claude.APIKeyBetaHeader, req.Header.Get("anthropic-beta"))
	require.Equal(t, "text/event-stream", req.Header.Get("accept"))
	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, "claude-fable-5-1", payload["model"])
	require.Equal(t, ".", payload["system"])
	require.Equal(t, float64(128), payload["max_tokens"])
	require.Equal(t, true, payload["stream"])
	require.Equal(t, map[string]any{"type": "adaptive"}, payload["thinking"])
	require.Equal(t, map[string]any{"effort": "low"}, payload["output_config"])
	require.NotContains(t, payload, "metadata")
	require.NotContains(t, payload, "temperature")
	messages := jsonSlice(t, payload["messages"])
	require.Len(t, messages, 1)
	require.Equal(t, "prompt c1", jsonMap(t, jsonSlice(t, jsonMap(t, messages[0])["content"])[0])["text"])
}

func TestAstraCheckProbe_AnthropicOAuthRequest(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformAnthropic, onlyModels("claude-fable-5.1"))
	reg := syntheticAstraRegistry(t)
	cell := reg.Package("meow-claude-other-cap98-efficient").CellIndex["c1"]
	account := &Account{ID: 9, Platform: PlatformAnthropic, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "oauth-token"}}

	req, err := f.svc.buildAnthropicMessagesProbeRequest(context.Background(), account, "claude-opus-5-5", anthropicProbeHeadersClaudeCode, func(modelID string, isOAuth bool) (map[string]any, error) {
		return createAnthropicAstraCheckPayload(modelID, cell, isOAuth)
	})
	require.NoError(t, err)
	require.Equal(t, testClaudeAPIURL, req.URL.String())
	require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
	require.Equal(t, claude.DefaultBetaHeader, req.Header.Get("anthropic-beta"))

	payload := decodeProbeRequestBody(t, req)
	system := jsonSlice(t, payload["system"])
	require.Len(t, system, 2)
	require.Equal(t, claudeCodeSystemPrompt, jsonMap(t, system[0])["text"])
	require.Equal(t, ".", jsonMap(t, system[1])["text"])
	require.NotEmpty(t, jsonMap(t, payload["metadata"])["user_id"])
	require.Equal(t, map[string]any{"type": "adaptive"}, payload["thinking"])

	// 非 claude-code 契约的题不带 thinking / output_config
	plain, err := createAnthropicAstraCheckPayload("m", &AstraBenchmarkCell{ID: "x", Prompt: "hi"}, false)
	require.NoError(t, err)
	require.NotContains(t, plain, "thinking")
	require.NotContains(t, plain, "output_config")
	require.Equal(t, ".", plain["system"])
	require.Equal(t, 128, plain["max_tokens"])
}

func TestAstraCheckProbe_TargetOfAnotherPlatformSendsNothing(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("claude-opus-5.5"))

	execution := f.run(t)[0]
	require.Empty(t, f.upstream.requests)
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, []string{AstraCheckReasonTargetNotAllowed}, execution.Result.Reasons)
	require.Nil(t, execution.Account)
}

func TestAstraCheckProbe_BenchmarkErrorSendsNothing(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-astra"))
	f.svc.SetAstraBenchmarkProvider(staticAstraRegistryProvider{err: ErrGroupStatusAstraBenchmarkInvalid})

	execution := f.run(t)[0]
	require.Empty(t, f.upstream.requests)
	require.Equal(t, []string{AstraCheckReasonBenchmarkInvalid}, execution.Result.Reasons)
}

func TestAstraCheckProbe_RejectsUnsupportedPlatform(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-astra"))
	gemini := &Group{ID: 32, Name: "Gemini", Platform: PlatformGemini, Status: StatusActive, Hydrated: true}

	_, err := f.svc.probeAstraCheck(context.Background(), gemini, f.cfg, f.cfg.AstraCheckModels)
	require.ErrorIs(t, err, ErrGroupStatusAstraCheckUnsupported)
	require.Empty(t, f.upstream.requests)
}

func TestAstraCheckProbe_RejectsWhenAlreadyRunning(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-astra"), astraLike()...)
	require.True(t, f.svc.markAstraCheckRunning(f.group.ID))
	require.True(t, f.svc.IsAstraCheckRunning(f.group.ID))

	_, err := f.svc.probeAstraCheck(context.Background(), f.group, f.cfg, f.cfg.AstraCheckModels)
	require.ErrorIs(t, err, ErrGroupStatusAstraCheckRunning)
	require.Empty(t, f.upstream.requests)
	require.ErrorIs(t, f.svc.StartAstraCheckAsync(f.group.ID, ""), ErrGroupStatusAstraCheckRunning)

	f.svc.clearAstraCheckRunning(f.group.ID)
	require.False(t, f.svc.IsAstraCheckRunning(f.group.ID))
}

func TestAstraCheckProbe_GroupNowChecksOneModel(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-sol", "gpt-6-astra"), astraLike()...)

	executions, err := f.svc.ProbeAstraCheckGroupNow(context.Background(), f.group.ID, "gpt-6-astra")
	require.NoError(t, err)
	require.Len(t, executions, 1)
	require.Equal(t, "gpt-6-astra", executions[0].Result.ExpectedModel)
	require.Len(t, f.upstream.requests, 4)

	_, err = f.svc.ProbeAstraCheckGroupNow(context.Background(), f.group.ID, "gpt-5.6-sol")
	require.ErrorIs(t, err, ErrGroupStatusAstraCheckTargetInvalid)
	require.ErrorIs(t, f.svc.StartAstraCheckAsync(f.group.ID, "gpt-5.6-sol"), ErrGroupStatusAstraCheckTargetInvalid)
}

func TestAstraCheckProbe_WithConfigRunsOnlyDueModels(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-sol", "gpt-6-astra"), astraLike()...)
	recent := time.Now().Add(-10 * time.Minute)
	f.repo.states = map[string]*GroupStatusAstraCheckState{
		"gpt-6-sol": {GroupID: f.group.ID, ExpectedModel: "gpt-6-sol", StableStatus: AstraCheckStatusPass, CheckedAt: &recent},
	}

	executions, err := f.svc.ProbeAstraCheckWithConfig(context.Background(), f.cfg)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	require.Equal(t, "gpt-6-astra", executions[0].Result.ExpectedModel)
	require.Len(t, f.upstream.requests, 4)

	// 两个都没到期：什么都不做
	executions, err = f.svc.ProbeAstraCheckWithConfig(context.Background(), f.cfg)
	require.NoError(t, err)
	require.Empty(t, executions)
	require.Len(t, f.upstream.requests, 4)

	_, err = f.svc.ProbeAstraCheckWithConfig(context.Background(), nil)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
}

func TestCreateOpenAIAstraCheckPayload_OAuthUsesInstructions(t *testing.T) {
	cell := syntheticGPTPackage(t).CellIndex["c1"]

	payload := createOpenAIAstraCheckPayload("gpt-6-astra", cell, true)
	require.Equal(t, ".", payload["instructions"])
	require.Equal(t, []string{"reasoning.encrypted_content"}, payload["include"])
	input, ok := payload["input"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, input, 1)
	require.Equal(t, "user", input[0]["role"])

	apiKey := createOpenAIAstraCheckPayload("gpt-6-astra", cell, false)
	require.NotContains(t, apiKey, "instructions")
	apiKeyInput, ok := apiKey["input"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, apiKeyInput, 2)

	// 缺省字段回退：空 system → "."，空 effort → low，0 上限 → 128
	bare := createOpenAIAstraCheckPayload("gpt-6-astra", &AstraBenchmarkCell{ID: "x", Prompt: "hi"}, true)
	require.Equal(t, ".", bare["instructions"])
	require.Equal(t, map[string]any{"effort": "low"}, bare["reasoning"])
	require.Equal(t, 128, bare["max_output_tokens"])
}

func TestAstraProgressTracker_CountsAndSnapshot(t *testing.T) {
	tracker := newAstraProgressTracker(2, "gpt-6-sol", 1, 3)
	tracker.setPlanned(3)
	tracker.setAccount(7)
	tracker.setPhase(AstraCheckPhaseRunning)

	code := 200
	tracker.requestStarted()
	tracker.record(AstraCheckSampleRecord{CellID: "a", Attempt: 1, Answer: "  Japan  ", Category: "japan", Outcome: AstraCheckSampleValid, Final: true, HTTPCode: &code}, true)
	tracker.requestStarted()
	tracker.record(AstraCheckSampleRecord{CellID: "b", Attempt: 1, Category: AstraCheckInvalidOutput, Outcome: AstraCheckSampleInvalid, Final: false}, true)
	tracker.requestStarted()
	snap := tracker.snapshot()
	require.Equal(t, "gpt-6-sol", snap.ExpectedModel)
	require.Equal(t, 1, snap.ModelIndex)
	require.Equal(t, 3, snap.ModelCount)
	require.Equal(t, 2, snap.Round)
	require.Equal(t, AstraCheckPhaseRunning, snap.Phase)
	require.Equal(t, int64(7), *snap.AccountID)
	require.Equal(t, 3, snap.Planned)
	require.Equal(t, 1, snap.Completed)
	require.Equal(t, 1, snap.Valid)
	require.Equal(t, 0, snap.Invalid)
	require.Equal(t, 3, snap.Requests)
	require.Equal(t, 1, snap.InFlight)
	require.Len(t, snap.Samples, 2)
	require.Equal(t, "Japan", snap.Samples[0].Answer)
	require.Equal(t, 1, snap.Samples[0].Seq)
	require.Equal(t, 2, snap.Samples[1].Seq)

	tracker.record(AstraCheckSampleRecord{CellID: "b", Attempt: 2, Outcome: AstraCheckSampleFailed, Final: true}, true)
	tracker.rollbackJob(AstraCheckSampleFailed)
	snap = tracker.snapshot()
	require.Equal(t, 1, snap.Completed)
	require.Equal(t, 0, snap.Failed)
	require.Equal(t, 0, snap.InFlight)
	require.Len(t, tracker.allSamples(), 3)

	var nilTracker *astraProgressTracker
	require.Nil(t, nilTracker.snapshot())
	require.Empty(t, nilTracker.allSamples())
	nilTracker.record(AstraCheckSampleRecord{}, true) // 不 panic
}

// 刚开始、还没有样本时，快照与汇总里的数组也必须是 []，不能是 null（前端直接读 .length）。
func TestAstraCheckJSON_ArraysNeverNull(t *testing.T) {
	fresh := newAstraProgressTracker(1, "gpt-6-astra", 1, 1).snapshot()
	require.NotNil(t, fresh.Samples)
	raw, err := json.Marshal(fresh)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"samples":[]`)

	summary := &GroupStatusSummary{}
	decorateAstraCheckSummary(summary)
	raw, err = json.Marshal(summary)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"astra_check_models":[]`)
	require.Contains(t, string(raw), `"astra_check_states":[]`)
	require.NotContains(t, string(raw), `"astra_check_benchmarks":null`)

	summary = &GroupStatusSummary{AstraCheckModels: onlyModels("gpt-6-astra")}
	decorateAstraCheckSummary(summary)
	raw, err = json.Marshal(summary.AstraCheckStates)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"matches":[]`)
	require.Contains(t, string(raw), `"reasons":[]`)
}
