package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------- 分类 ----------

func TestNormalizeSolJuiceAnswer(t *testing.T) {
	cases := map[string]struct {
		value string
		ok    bool
	}{
		"40":                 {"40", true},
		" 40 ":               {"40", true},
		"40.0":               {"40", true},
		"040":                {"40", true},
		"+40":                {"40", true},
		"40.":                {"40", true},
		"**40**":             {"40", true},
		"`40`":               {"40", true},
		"```\n40\n```":       {"40", true},
		"```text\n40\n```":   {"40", true},
		"40.50":              {"40.5", true},
		"4000":               {"4000", true},
		"-0":                 {"0", true},
		"":                   {"", false},
		"forty":              {"", false},
		"The answer is 40":   {"", false},
		"40 tokens":          {"", false},
		"I can't share that": {"", false},
	}
	for raw, expected := range cases {
		value, ok := NormalizeSolJuiceAnswer(raw)
		require.Equal(t, expected.ok, ok, "raw=%q", raw)
		require.Equal(t, expected.value, value, "raw=%q", raw)
	}
}

func TestClassifySolJuiceAnswer(t *testing.T) {
	cases := []struct {
		raw            string
		classification string
		value          string
		detailContains string
	}{
		{"40", SolJuiceStatusPass, "40", ""},
		{"40.0", SolJuiceStatusPass, "40", ""},
		{"```\n40\n```", SolJuiceStatusPass, "40", ""},
		{"**40**", SolJuiceStatusPass, "40", ""},
		{"4000", SolJuiceStatusPass, "4000", ""},
		{"40.5", SolJuiceStatusPass, "40.5", ""},
		{"32", SolJuiceStatusMismatch, "32", "gpt-5.6-terra"},
		{"48", SolJuiceStatusMismatch, "48", "gpt-5.6-luna"},
		{"96", SolJuiceStatusMismatch, "96", "gpt-5.5"},
		{"64", SolJuiceStatusMismatch, "64", "gpt-5.4-mini"},
		{"41", SolJuiceStatusInconclusive, "41", "unknown juice value"},
		{"I can't provide that", SolJuiceStatusInconclusive, "", "non-numeric"},
		{"", SolJuiceStatusInconclusive, "", "empty answer"},
	}
	for _, tc := range cases {
		classification, value, detail := ClassifySolJuiceAnswer(tc.raw)
		require.Equal(t, tc.classification, classification, "raw=%q", tc.raw)
		require.Equal(t, tc.value, value, "raw=%q", tc.raw)
		if tc.detailContains == "" {
			require.Empty(t, detail, "raw=%q", tc.raw)
		} else {
			require.Contains(t, detail, tc.detailContains, "raw=%q", tc.raw)
		}
	}
}

// ---------- 状态机 ----------

func solJuiceResult(classification, value string) *GroupStatusSolJuiceResult {
	httpCode := 200
	latency := int64(1500)
	return &GroupStatusSolJuiceResult{
		GroupID:         7,
		ConfigID:        3,
		Model:           "gpt-5.6-sol",
		Effort:          "high",
		Classification:  classification,
		NormalizedValue: value,
		AnswerExcerpt:   value,
		HTTPCode:        &httpCode,
		LatencyMS:       &latency,
		InputTokens:     52,
		OutputTokens:    210,
		ReasoningTokens: 192,
		ObservedAt:      time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC),
	}
}

func TestComputeSolJuiceTransition(t *testing.T) {
	// 首次 pass：稳定 pass，无事件
	state, event := ComputeSolJuiceTransition(nil, solJuiceResult(SolJuiceStatusPass, "40"))
	require.Nil(t, event)
	require.Equal(t, SolJuiceStatusPass, state.SolJuiceStableStatus)
	require.Equal(t, SolJuiceStatusPass, state.SolJuiceStatus)
	require.Equal(t, "40", state.SolJuiceValue)
	require.Equal(t, int64(192), state.SolJuiceReasoningTokens)
	require.NotNil(t, state.SolJuiceCheckedAt)
	require.Equal(t, int64(7), state.GroupID)
	require.Equal(t, int64(3), state.ConfigID)

	// 第 1 次 mismatch：计数 1，稳定不变，无事件
	state, event = ComputeSolJuiceTransition(state, solJuiceResult(SolJuiceStatusMismatch, "32"))
	require.Nil(t, event)
	require.Equal(t, 1, state.SolJuiceConsecutiveMismatch)
	require.Equal(t, SolJuiceStatusPass, state.SolJuiceStableStatus)
	require.Equal(t, SolJuiceStatusMismatch, state.SolJuiceStatus)

	// inconclusive：不动计数与稳定
	state, event = ComputeSolJuiceTransition(state, solJuiceResult(SolJuiceStatusInconclusive, ""))
	require.Nil(t, event)
	require.Equal(t, 1, state.SolJuiceConsecutiveMismatch)
	require.Equal(t, SolJuiceStatusPass, state.SolJuiceStableStatus)
	require.Equal(t, SolJuiceStatusInconclusive, state.SolJuiceStatus)

	// 第 2 次 mismatch：稳定 mismatch + 事件
	state, event = ComputeSolJuiceTransition(state, solJuiceResult(SolJuiceStatusMismatch, "48"))
	require.NotNil(t, event)
	require.Equal(t, GroupStatusEventSolJuiceMismatch, event.EventType)
	require.Equal(t, SolJuiceStatusPass, event.FromStatus)
	require.Equal(t, SolJuiceStatusMismatch, event.ToStatus)
	require.Equal(t, "juice_48", event.SubStatus)
	require.Equal(t, 2, state.SolJuiceConsecutiveMismatch)
	require.Equal(t, SolJuiceStatusMismatch, state.SolJuiceStableStatus)

	// 已经 mismatch 再 mismatch：只计数，不重复发事件
	state, event = ComputeSolJuiceTransition(state, solJuiceResult(SolJuiceStatusMismatch, "48"))
	require.Nil(t, event)
	require.Equal(t, 3, state.SolJuiceConsecutiveMismatch)

	// 恢复：pass 清零并发 recovered
	state, event = ComputeSolJuiceTransition(state, solJuiceResult(SolJuiceStatusPass, "40"))
	require.NotNil(t, event)
	require.Equal(t, GroupStatusEventSolJuiceRecovered, event.EventType)
	require.Equal(t, SolJuiceStatusMismatch, event.FromStatus)
	require.Equal(t, SolJuiceStatusPass, event.ToStatus)
	require.Equal(t, "juice_40", event.SubStatus)
	require.Equal(t, 0, state.SolJuiceConsecutiveMismatch)
	require.Equal(t, SolJuiceStatusPass, state.SolJuiceStableStatus)
}

func TestComputeSolJuiceTransition_FirstTwoMismatchesFromEmpty(t *testing.T) {
	state, event := ComputeSolJuiceTransition(nil, solJuiceResult(SolJuiceStatusMismatch, "32"))
	require.Nil(t, event)
	require.Equal(t, "", state.SolJuiceStableStatus)

	state, event = ComputeSolJuiceTransition(state, solJuiceResult(SolJuiceStatusMismatch, "32"))
	require.NotNil(t, event)
	require.Equal(t, "", event.FromStatus)
	require.Equal(t, SolJuiceStatusMismatch, state.SolJuiceStableStatus)
}

func TestComputeSolJuiceTransition_DoesNotTouchLivenessState(t *testing.T) {
	observed := time.Now()
	prev := &GroupStatusState{
		GroupID:            7,
		ConfigID:           3,
		LatestStatus:       GroupRuntimeStatusUp,
		StableStatus:       GroupRuntimeStatusUp,
		ObservedAt:         &observed,
		ConsecutiveNonDown: 5,
	}
	state, _ := ComputeSolJuiceTransition(prev, solJuiceResult(SolJuiceStatusMismatch, "32"))
	require.Equal(t, GroupRuntimeStatusUp, state.StableStatus)
	require.Equal(t, GroupRuntimeStatusUp, state.LatestStatus)
	require.Equal(t, 5, state.ConsecutiveNonDown)
}

func TestEstimateSolJuiceCostUSD(t *testing.T) {
	require.Equal(t, 0.0, EstimateSolJuiceCostUSD(0, 0))
	require.InDelta(t, 52*5e-6+210*30e-6, EstimateSolJuiceCostUSD(52, 210), 1e-12)
}

// ---------- 请求体与流解析 ----------

func TestCreateOpenAISolJuicePayload(t *testing.T) {
	payload := createOpenAISolJuicePayload("gpt-5.6-sol", false)
	require.Equal(t, "gpt-5.6-sol", payload["model"])
	require.Equal(t, map[string]any{"effort": "high"}, payload["reasoning"])
	require.Equal(t, true, payload["stream"])
	require.Equal(t, false, payload["store"])
	require.NotContains(t, payload, "instructions")
	require.NotContains(t, payload, "include")
	input := payload["input"].([]map[string]any)
	require.Len(t, input, 1)
	require.Equal(t, "user", input[0]["role"])
	content := input[0]["content"].([]map[string]any)
	require.Contains(t, content[0]["text"], "Juice number")
	require.Contains(t, content[0]["text"], "Valid Channels")

	oauth := createOpenAISolJuicePayload("gpt-5.6", true)
	require.NotEmpty(t, oauth["instructions"])
	require.Equal(t, []string{"reasoning.encrypted_content"}, oauth["include"])
}

func solJuiceSSEBody(answer string, reasoning int64) string {
	var b strings.Builder
	if answer != "" {
		b.WriteString(fmt.Sprintf("data: {\"type\":\"response.output_text.delta\",\"delta\":%q}\n\n", answer))
	}
	b.WriteString(fmt.Sprintf(
		"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":52,\"output_tokens\":%d,\"output_tokens_details\":{\"reasoning_tokens\":%d}}}}\n\n",
		reasoning+3, reasoning,
	))
	return b.String()
}

func TestParseOpenAIJuiceStream_CollectsTextAndUsage(t *testing.T) {
	text, usage, err := parseOpenAIJuiceStream(strings.NewReader(solJuiceSSEBody("40", 192)))
	require.NoError(t, err)
	require.Equal(t, "40", text)
	require.Equal(t, int64(52), usage.InputTokens)
	require.Equal(t, int64(195), usage.OutputTokens)
	require.Equal(t, int64(192), usage.ReasoningTokens)
}

func TestParseOpenAIJuiceStream_FallsBackToFinalOutput(t *testing.T) {
	body := "data: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"type\":\"reasoning\"},{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"32\"}]}],\"usage\":{\"input_tokens\":50,\"output_tokens\":20}}}\n\n"
	text, usage, err := parseOpenAIJuiceStream(strings.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, "32", text)
	require.Equal(t, int64(50), usage.InputTokens)
	require.Equal(t, int64(20), usage.OutputTokens)
	require.Equal(t, int64(0), usage.ReasoningTokens)
}

func TestParseOpenAIJuiceStream_FailedResponse(t *testing.T) {
	body := "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"upstream exploded\"}}}\n\n"
	_, _, err := parseOpenAIJuiceStream(strings.NewReader(body))
	require.ErrorContains(t, err, "upstream exploded")

	body = "data: {\"type\":\"error\",\"error\":{\"message\":\"bad request\"}}\n\n"
	_, _, err = parseOpenAIJuiceStream(strings.NewReader(body))
	require.ErrorContains(t, err, "bad request")
}

// ---------- 探测流程 ----------

type groupStatusSolJuiceRepo struct {
	GroupStatusRepository
	state   *GroupStatusState
	results []*GroupStatusSolJuiceResult
	events  []*GroupStatusEvent
}

func (r *groupStatusSolJuiceRepo) SaveSolJuiceResult(_ context.Context, result *GroupStatusSolJuiceResult) (*GroupStatusState, *GroupStatusEvent, error) {
	copied := *result
	r.results = append(r.results, &copied)
	next, event := ComputeSolJuiceTransition(r.state, &copied)
	r.state = next
	if event != nil {
		r.events = append(r.events, event)
	}
	stateCopy := *next
	return &stateCopy, event, nil
}

func newSolJuiceProbeFixture(t *testing.T, responses ...*http.Response) (*GroupStatusProbeService, *Group, *GroupStatusConfig, *groupStatusSolJuiceRepo, *groupStatusProbeHTTPUpstream, *recordingGroupStatusNotifier) {
	t.Helper()
	group := &Group{ID: 30, Name: "Sol 高速", Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}
	accounts := []Account{
		groupStatusProbeAccount(1, PlatformOpenAI, group.ID, 1, map[string]any{"gpt-5.6-sol": "gpt-5.6-sol"}),
		groupStatusProbeAccount(2, PlatformOpenAI, group.ID, 2, map[string]any{"gpt-5.6-sol": "gpt-5.6-sol"}),
	}
	for i := range accounts {
		accounts[i].Credentials["api_key"] = "sk-test"
	}
	upstream := &groupStatusProbeHTTPUpstream{responses: responses}
	svc := newGroupStatusProbeServiceForTest(group, accounts, nil, upstream, nil)
	repo := &groupStatusSolJuiceRepo{}
	svc.repo = repo
	notifier := &recordingGroupStatusNotifier{}
	svc.SetTransitionNotifier(notifier)

	cfg := groupStatusProbeConfig(group.ID, "gpt-5.6-sol")
	cfg.SolJuiceEnabled = true
	cfg.SolJuiceModel = "gpt-5.6-sol"
	cfg.SolJuiceIntervalSeconds = 900
	return svc, group, cfg, repo, upstream, notifier
}

func decodeProbeRequestBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

func TestSolJuiceProbe_PassOnSol40(t *testing.T) {
	svc, group, cfg, repo, upstream, notifier := newSolJuiceProbeFixture(t,
		groupStatusProbeResponse(200, solJuiceSSEBody("40", 192)),
	)

	execution, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	require.Equal(t, SolJuiceStatusPass, execution.Result.Classification)
	require.Equal(t, "40", execution.Result.NormalizedValue)
	require.Equal(t, int64(52), execution.Result.InputTokens)
	require.Equal(t, int64(195), execution.Result.OutputTokens)
	require.Equal(t, int64(192), execution.Result.ReasoningTokens)
	require.Equal(t, "gpt-5.6-sol", execution.Result.Model)
	require.Equal(t, "high", execution.Result.Effort)
	require.NotNil(t, execution.Account)
	require.Equal(t, int64(1), execution.Account.ID)
	require.Equal(t, SolJuiceStatusPass, execution.State.SolJuiceStableStatus)
	require.Nil(t, execution.Event)
	require.Empty(t, repo.events)
	require.Equal(t, 0, notifier.calls)

	require.Len(t, upstream.requests, 1)
	req := upstream.requests[0]
	require.Equal(t, "https://example.com/responses", req.URL.String())
	require.Equal(t, "Bearer sk-test", req.Header.Get("Authorization"))
	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, "gpt-5.6-sol", payload["model"])
	require.Equal(t, map[string]any{"effort": "high"}, payload["reasoning"])
	require.Equal(t, false, payload["store"])
	require.NotContains(t, payload, "instructions")
}

func TestSolJuiceProbe_MismatchConfirmedByImmediateRecheck(t *testing.T) {
	svc, group, cfg, repo, upstream, notifier := newSolJuiceProbeFixture(t,
		groupStatusProbeResponse(200, solJuiceSSEBody("32", 150)),
		groupStatusProbeResponse(200, solJuiceSSEBody("32", 160)),
	)

	execution, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.NoError(t, err)
	require.True(t, execution.Confirmed)
	require.Len(t, upstream.requests, 2)
	require.Len(t, repo.results, 2)
	require.Equal(t, SolJuiceStatusMismatch, execution.State.SolJuiceStableStatus)
	require.Equal(t, 2, execution.State.SolJuiceConsecutiveMismatch)
	require.NotNil(t, execution.Event)
	require.Equal(t, GroupStatusEventSolJuiceMismatch, execution.Event.EventType)
	require.Equal(t, "juice_32", execution.Event.SubStatus)
	require.Contains(t, execution.Event.ErrorDetail, "gpt-5.6-terra")
	require.Len(t, repo.events, 1)
	require.Equal(t, 1, notifier.calls)
	require.Same(t, group, notifier.groups[0])
	require.Equal(t, GroupStatusEventSolJuiceMismatch, notifier.events[0].EventType)
}

func TestSolJuiceProbe_MismatchThenPassDoesNotAlert(t *testing.T) {
	svc, group, cfg, repo, upstream, notifier := newSolJuiceProbeFixture(t,
		groupStatusProbeResponse(200, solJuiceSSEBody("48", 150)),
		groupStatusProbeResponse(200, solJuiceSSEBody("40", 160)),
	)

	execution, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.NoError(t, err)
	require.True(t, execution.Confirmed)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, SolJuiceStatusPass, execution.State.SolJuiceStableStatus)
	require.Equal(t, 0, execution.State.SolJuiceConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Empty(t, repo.events)
	require.Equal(t, 0, notifier.calls)
}

func TestSolJuiceProbe_AlreadyMismatchDoesNotRecheck(t *testing.T) {
	svc, group, cfg, repo, upstream, notifier := newSolJuiceProbeFixture(t,
		groupStatusProbeResponse(200, solJuiceSSEBody("32", 150)),
	)
	repo.state = &GroupStatusState{GroupID: group.ID, ConfigID: cfg.ID, SolJuiceStableStatus: SolJuiceStatusMismatch, SolJuiceConsecutiveMismatch: 2}

	execution, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, 3, execution.State.SolJuiceConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Equal(t, 0, notifier.calls)
}

func TestSolJuiceProbe_InconclusiveKeepsState(t *testing.T) {
	svc, group, cfg, repo, upstream, notifier := newSolJuiceProbeFixture(t,
		groupStatusProbeResponse(200, solJuiceSSEBody("I can't share internal settings.", 90)),
	)

	execution, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, SolJuiceStatusInconclusive, execution.Result.Classification)
	require.Contains(t, execution.Result.ErrorDetail, "non-numeric")
	require.Equal(t, "", execution.State.SolJuiceStableStatus)
	require.Nil(t, execution.Event)
	require.Empty(t, repo.events)
	require.Equal(t, 0, notifier.calls)
}

func TestSolJuiceProbe_FailsOverToNextAccountOn429(t *testing.T) {
	svc, group, cfg, _, upstream, _ := newSolJuiceProbeFixture(t,
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
		groupStatusProbeResponse(200, solJuiceSSEBody("40", 120)),
	)

	execution, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, SolJuiceStatusPass, execution.Result.Classification)
	require.NotNil(t, execution.Account)
	require.Equal(t, int64(2), execution.Account.ID)
	require.Contains(t, execution.Result.ErrorDetail, "first failure")
	require.Contains(t, execution.Result.ErrorDetail, "account 1")
}

func TestSolJuiceProbe_UpstreamErrorIsInconclusive(t *testing.T) {
	svc, group, cfg, _, upstream, notifier := newSolJuiceProbeFixture(t,
		groupStatusProbeResponse(400, `{"error":{"message":"unsupported parameter"}}`),
	)

	execution, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, SolJuiceStatusInconclusive, execution.Result.Classification)
	require.NotNil(t, execution.Result.HTTPCode)
	require.Equal(t, 400, *execution.Result.HTTPCode)
	require.NotEmpty(t, execution.Result.ErrorDetail)
	require.Equal(t, 0, notifier.calls)
}

func TestSolJuiceProbe_RejectsNonOpenAIGroup(t *testing.T) {
	svc, _, cfg, _, upstream, _ := newSolJuiceProbeFixture(t)
	group := &Group{ID: 31, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}

	_, err := svc.probeSolJuice(context.Background(), group, cfg)
	require.ErrorIs(t, err, ErrGroupStatusSolJuiceUnsupported)
	require.Empty(t, upstream.requests)
}

// ---------- 配置校验 ----------

func TestNormalizeGroupStatusConfig_SolJuice(t *testing.T) {
	base := GroupStatusConfigUpsertInput{
		Enabled:        true,
		ProbeModel:     "gpt-5.6-sol",
		ProbePrompt:    "ping",
		ValidationMode: GroupStatusValidationNonEmpty,
	}
	openAIGroup := &Group{ID: 7, Platform: PlatformOpenAI}
	anthropicGroup := &Group{ID: 8, Platform: PlatformAnthropic}

	cfg, err := NormalizeGroupStatusConfig(openAIGroup, &base)
	require.NoError(t, err)
	require.False(t, cfg.SolJuiceEnabled)
	require.Equal(t, 900, cfg.SolJuiceIntervalSeconds)
	require.Equal(t, "gpt-5.6-sol", cfg.SolJuiceModel)

	enabled := true
	on := base
	on.SolJuiceEnabled = &enabled
	on.SolJuiceIntervalSeconds = 1800
	on.SolJuiceModel = " gpt-5.6 "
	cfg, err = NormalizeGroupStatusConfig(openAIGroup, &on)
	require.NoError(t, err)
	require.True(t, cfg.SolJuiceEnabled)
	require.Equal(t, 1800, cfg.SolJuiceIntervalSeconds)
	require.Equal(t, "gpt-5.6", cfg.SolJuiceModel)

	_, err = NormalizeGroupStatusConfig(anthropicGroup, &on)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)

	tooFast := on
	tooFast.SolJuiceIntervalSeconds = 100
	_, err = NormalizeGroupStatusConfig(openAIGroup, &tooFast)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
}

// ---------- 推送 ----------

func TestBuildGroupStatusNotifyMessage_SolJuiceEvents(t *testing.T) {
	group := &Group{ID: 7, Name: "Sol 高速", Platform: PlatformOpenAI}
	mismatch := &GroupStatusEvent{
		EventType:   GroupStatusEventSolJuiceMismatch,
		FromStatus:  SolJuiceStatusPass,
		ToStatus:    SolJuiceStatusMismatch,
		SubStatus:   "juice_32",
		ErrorDetail: "juice 32 matches gpt-5.6-terra fingerprint, not gpt-5.6-sol",
		ObservedAt:  time.Now(),
	}
	title, desp := buildGroupStatusNotifyMessage("My Gateway", group, mismatch)
	require.Equal(t, "[My Gateway] 分组「Sol 高速」疑似非 Sol（Juice 指纹 32）", title)
	require.Contains(t, desp, "**状态**：Sol 验证通过 → 非 Sol")
	require.Contains(t, desp, "gpt-5.6-terra")

	recovered := &GroupStatusEvent{
		EventType:  GroupStatusEventSolJuiceRecovered,
		FromStatus: SolJuiceStatusMismatch,
		ToStatus:   SolJuiceStatusPass,
		SubStatus:  "juice_40",
		ObservedAt: time.Now(),
	}
	title, desp = buildGroupStatusNotifyMessage("My Gateway", group, recovered)
	require.Equal(t, "[My Gateway] 分组「Sol 高速」Sol 验证已恢复", title)
	require.Contains(t, desp, "**状态**：非 Sol → Sol 验证通过")
}

func TestGroupStatusNotifyService_NotifyTransition_AcceptsSolJuiceEvents(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", "sctp_secret"))

	cfg := &GroupStatusConfig{GroupID: 7, NotifyEnabled: true}
	svc.NotifyTransition(&Group{ID: 7, Name: "G7"}, cfg, &GroupStatusEvent{
		EventType:  GroupStatusEventSolJuiceMismatch,
		FromStatus: SolJuiceStatusPass,
		ToStatus:   SolJuiceStatusMismatch,
		SubStatus:  "juice_48",
		ObservedAt: time.Now(),
	})
	require.Eventually(t, func() bool { return ts.count() == 1 }, 5*time.Second, 10*time.Millisecond)
	require.Contains(t, ts.request(0).Title, "疑似非 Sol（Juice 指纹 48）")

	// 分组关闭推送时同样静默
	silent := &GroupStatusConfig{GroupID: 7, NotifyEnabled: false}
	svc.NotifyTransition(&Group{ID: 7, Name: "G7"}, silent, &GroupStatusEvent{EventType: GroupStatusEventSolJuiceRecovered, ObservedAt: time.Now()})
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 1, ts.count())
}
