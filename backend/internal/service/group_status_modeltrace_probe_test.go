package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

// ---------- 桩 ----------

type groupStatusModelTraceRepo struct {
	GroupStatusRepository
	state   *GroupStatusState
	results []*GroupStatusModelTraceResult
	events  []*GroupStatusEvent
}

func (r *groupStatusModelTraceRepo) SaveModelTraceRun(_ context.Context, result *GroupStatusModelTraceResult) (*GroupStatusModelTraceRun, *GroupStatusState, *GroupStatusEvent, error) {
	copied := *result
	r.results = append(r.results, &copied)
	runID := int64(len(r.results))
	next, event := ComputeModelTraceTransition(r.state, &copied, runID)
	r.state = next
	if event != nil {
		r.events = append(r.events, event)
	}
	stateCopy := *next
	run := &GroupStatusModelTraceRun{
		ID:            runID,
		GroupID:       copied.GroupID,
		ConfigID:      copied.ConfigID,
		ExpectedModel: copied.ExpectedModel,
		Verdict:       copied.Verdict,
		TopModel:      copied.TopModel,
		AccountID:     copied.AccountID,
	}
	return run, &stateCopy, event, nil
}

// modelTraceFixedChallenges 固定挑战：长度 300（最少 165 个数字），与参考回答的长度相容。
func modelTraceFixedChallenges(n int) []ModelTraceChallenge {
	out := make([]ModelTraceChallenge, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, ModelTraceChallenge{
			ID:            fmt.Sprintf("probe-%d-test", i+1),
			ExpectedCount: 300,
			Prompt:        fmt.Sprintf("challenge %d: 300 个 1 到 355（含端点）的整数", i+1),
		})
	}
	return out
}

// modelTraceReferenceTexts 从 golden 数据里取某个用例的参考回答原文。
func modelTraceReferenceTexts(t *testing.T, caseName string) []string {
	t.Helper()
	golden := loadModelTraceGolden(t)
	for _, tc := range golden.Cases {
		if tc.Name == caseName {
			texts := make([]string, 0, len(tc.Outputs))
			for _, out := range tc.Outputs {
				texts = append(texts, out.Text)
			}
			return texts
		}
	}
	t.Fatalf("golden case %s not found", caseName)
	return nil
}

func modelTraceOpenAISSE(text string) *http.Response {
	delta, _ := json.Marshal(map[string]any{"type": "response.output_text.delta", "delta": text})
	body := "data: " + string(delta) + "\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":300,\"output_tokens\":1200,\"output_tokens_details\":{\"reasoning_tokens\":100}}}}\n\n"
	return groupStatusProbeResponse(200, body)
}

func modelTraceAnthropicSSE(text, stopReason string) *http.Response {
	delta, _ := json.Marshal(map[string]any{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]any{"type": "text_delta", "text": text},
	})
	body := "event: message_start\n" +
		"data: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-opus-5-5\",\"usage\":{\"input_tokens\":250,\"output_tokens\":1}}}\n\n" +
		"data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"data: " + string(delta) + "\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"" + stopReason + "\"},\"usage\":{\"output_tokens\":1100}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	return groupStatusProbeResponse(200, body)
}

func modelTraceOpenAIResponses(texts ...string) []*http.Response {
	out := make([]*http.Response, 0, len(texts))
	for _, text := range texts {
		out = append(out, modelTraceOpenAISSE(text))
	}
	return out
}

type modelTraceFixture struct {
	svc      *GroupStatusProbeService
	group    *Group
	cfg      *GroupStatusConfig
	repo     *groupStatusModelTraceRepo
	upstream *groupStatusProbeHTTPUpstream
	notifier *recordingGroupStatusNotifier
}

func newModelTraceFixture(t *testing.T, platform, expected string, accounts []Account, responses ...*http.Response) *modelTraceFixture {
	t.Helper()
	group := &Group{ID: 40, Name: "指纹分组", Platform: platform, Status: StatusActive, Hydrated: true}
	for i := range accounts {
		accounts[i].AccountGroups = []AccountGroup{{GroupID: group.ID}}
	}
	upstream := &groupStatusProbeHTTPUpstream{responses: responses}
	svc := newGroupStatusProbeServiceForTest(group, accounts, nil, upstream, nil)
	repo := &groupStatusModelTraceRepo{}
	svc.repo = repo
	notifier := &recordingGroupStatusNotifier{}
	svc.SetTransitionNotifier(notifier)

	// 测试桩按顺序弹出响应且无锁，串行执行；重试退避不等待；挑战固定
	svc.modelTraceConcurrency = 1
	svc.modelTraceSleep = func(context.Context, time.Duration) error { return nil }
	svc.modelTraceChallenges = modelTraceFixedChallenges

	cfg := groupStatusProbeConfig(group.ID, expected)
	cfg.ModelTraceEnabled = true
	cfg.ModelTraceExpectedModel = expected
	cfg.ModelTraceIntervalSeconds = 3600
	return &modelTraceFixture{svc: svc, group: group, cfg: cfg, repo: repo, upstream: upstream, notifier: notifier}
}

func modelTraceOpenAIAccounts(models ...string) []Account {
	mapping := map[string]any{}
	for _, m := range models {
		mapping[m] = m
	}
	a1 := groupStatusProbeAccount(1, PlatformOpenAI, 0, 1, mapping)
	a2 := groupStatusProbeAccount(2, PlatformOpenAI, 0, 2, mapping)
	a1.Credentials["api_key"] = "sk-test"
	a2.Credentials["api_key"] = "sk-test"
	return []Account{a1, a2}
}

// ---------- OpenAI ----------

func TestModelTraceProbe_OpenAIMatch(t *testing.T) {
	sol := modelTraceReferenceTexts(t, "self-gpt-5.6-sol")
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-5.6-sol", modelTraceOpenAIAccounts("gpt-5.6-sol"), modelTraceOpenAIResponses(sol...)...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	result := execution.Result
	require.Equal(t, ModelTraceVerdictMatch, result.Verdict)
	require.Equal(t, ModelTraceOutcomeCompatible, result.Outcome)
	require.Empty(t, result.Reasons)
	require.Equal(t, "gpt-5.6-sol", result.TopModel)
	require.Greater(t, result.TopProbability, 0.99)
	require.NotNil(t, result.ExpectedProbability)
	require.Equal(t, 3, result.ValidOutputs)
	require.Equal(t, 3, result.AttemptsMade)
	require.Equal(t, 6, result.AttemptsPlanned)
	require.Equal(t, 3, result.CalibrationQueries)
	require.Len(t, result.Ranking, 16)
	require.Len(t, result.FamilyProbabilities, 2)
	require.Equal(t, int64(900), result.InputTokens)
	require.Equal(t, int64(3600), result.OutputTokens)
	require.Equal(t, int64(300), result.ReasoningTokens)
	require.Equal(t, PlatformOpenAI, result.Platform)
	require.Len(t, result.BankSHA256, 64)
	require.NotNil(t, result.AccountID)
	require.Equal(t, int64(1), *result.AccountID)
	require.Equal(t, AccountTypeAPIKey, result.AccountType)
	require.Len(t, result.Outputs, 3)
	for i, rec := range result.Outputs {
		require.Equal(t, i+1, rec.Seq)
		require.True(t, rec.Accepted)
		require.Equal(t, 165, rec.MinimumNumbers)
		require.NotEmpty(t, rec.Numbers)
		require.Equal(t, "gpt-5.6-sol", rec.TopModel)
		require.NotEmpty(t, rec.Excerpt)
		require.LessOrEqual(t, len([]rune(rec.Excerpt)), modelTraceExcerptMaxRunes+1)
	}
	require.Equal(t, ModelTraceStatusPass, execution.State.ModelTraceStableStatus)
	require.Nil(t, execution.Event)
	require.Equal(t, 0, f.notifier.calls)
	require.False(t, f.svc.IsModelTraceRunning(f.group.ID))
	require.Nil(t, f.svc.ModelTraceProgress(f.group.ID))

	require.Len(t, f.upstream.requests, 3)
	req := f.upstream.requests[0]
	require.Equal(t, "https://example.com/responses", req.URL.String())
	require.Equal(t, "Bearer sk-test", req.Header.Get("Authorization"))
	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, "gpt-5.6-sol", payload["model"])
	require.Equal(t, true, payload["stream"])
	require.Equal(t, false, payload["store"])
	for _, key := range []string{"reasoning", "temperature", "max_output_tokens", "instructions", "include"} {
		require.NotContains(t, payload, key)
	}
	input := payload["input"].([]any)
	require.Len(t, input, 1)
	user := input[0].(map[string]any)
	require.Equal(t, "user", user["role"])
	require.Equal(t, "challenge 1: 300 个 1 到 355（含端点）的整数", user["content"].([]any)[0].(map[string]any)["text"])
}

func TestModelTraceProbe_OpenAIOAuthUsesCodexInstructions(t *testing.T) {
	sol := modelTraceReferenceTexts(t, "self-gpt-6-sol")
	accounts := modelTraceOpenAIAccounts("gpt-6-sol")
	accounts[0].Type = AccountTypeOAuth
	accounts[0].Credentials["access_token"] = "oauth-token"
	accounts[0].Credentials["chatgpt_account_id"] = "acct-1"
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-6-sol", accounts, modelTraceOpenAIResponses(sol...)...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Equal(t, ModelTraceVerdictMatch, execution.Result.Verdict)
	req := f.upstream.requests[0]
	require.Equal(t, "chatgpt.com", req.Host)
	require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, openai.CodexBaseInstructionsForModel("gpt-6-sol"), payload["instructions"])
	require.NotContains(t, payload, "reasoning")
}

func TestModelTraceProbe_RequestModelOverride(t *testing.T) {
	astra := modelTraceReferenceTexts(t, "self-gpt-6-astra")
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-6-astra", modelTraceOpenAIAccounts("astra-alias"), modelTraceOpenAIResponses(astra...)...)
	f.cfg.ModelTraceRequestModel = "astra-alias"

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Equal(t, ModelTraceVerdictMatch, execution.Result.Verdict)
	require.Equal(t, "astra-alias", execution.Result.RequestModel)
	require.Equal(t, "gpt-6-astra", execution.Result.ExpectedModel)
	require.Equal(t, "astra-alias", decodeProbeRequestBody(t, f.upstream.requests[0])["model"])
}

func TestModelTraceProbe_MismatchConfirmedOnSameAccount(t *testing.T) {
	terra := modelTraceReferenceTexts(t, "terra-claimed-as-sol")
	responses := append(modelTraceOpenAIResponses(terra...), modelTraceOpenAIResponses(terra...)...)
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-5.6-sol", modelTraceOpenAIAccounts("gpt-5.6-sol"), responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.True(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 6)
	require.Len(t, f.repo.results, 2)
	require.Equal(t, 1, f.repo.results[0].Round)
	require.Equal(t, 2, f.repo.results[1].Round)
	require.Equal(t, *f.repo.results[0].AccountID, *f.repo.results[1].AccountID)
	require.Equal(t, ModelTraceVerdictMismatch, execution.Result.Verdict)
	require.Equal(t, "gpt-5.6-terra", execution.Result.TopModel)
	require.Equal(t, ModelTraceStatusMismatch, execution.State.ModelTraceStableStatus)
	require.Equal(t, 2, execution.State.ModelTraceConsecutiveMismatch)
	require.NotNil(t, execution.Event)
	require.Equal(t, GroupStatusEventModelTraceMismatch, execution.Event.EventType)
	require.Equal(t, "top_gpt-5.6-terra", execution.Event.SubStatus)
	require.Len(t, f.repo.events, 1)
	require.Equal(t, 1, f.notifier.calls)
	require.Same(t, f.group, f.notifier.groups[0])
}

func TestModelTraceProbe_AlreadyMismatchDoesNotRecheck(t *testing.T) {
	terra := modelTraceReferenceTexts(t, "terra-claimed-as-sol")
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-5.6-sol", modelTraceOpenAIAccounts("gpt-5.6-sol"), modelTraceOpenAIResponses(terra...)...)
	f.repo.state = &GroupStatusState{
		GroupID:                       f.group.ID,
		ConfigID:                      f.cfg.ID,
		ModelTraceStableStatus:        ModelTraceStatusMismatch,
		ModelTraceConsecutiveMismatch: 2,
		ModelTraceRunExpectedModel:    "gpt-5.6-sol",
	}

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.False(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 3)
	require.Equal(t, 3, execution.State.ModelTraceConsecutiveMismatch)
	require.Nil(t, execution.Event)
	require.Equal(t, 0, f.notifier.calls)
}

func TestModelTraceProbe_RejectedOutputsAreReplaced(t *testing.T) {
	sol := modelTraceReferenceTexts(t, "self-gpt-5.6-sol")
	incomplete := groupStatusProbeResponse(200, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"1, 2, 3\"}\n\n"+
		"data: {\"type\":\"response.incomplete\",\"response\":{\"incomplete_details\":{\"reason\":\"max_output_tokens\"}}}\n\n")
	responses := []*http.Response{
		modelTraceOpenAISSE("I can't produce random numbers."),
		incomplete,
		modelTraceOpenAISSE(sol[0]),
		modelTraceOpenAISSE(sol[1]),
		modelTraceOpenAISSE(sol[2]),
	}
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-5.6-sol", modelTraceOpenAIAccounts("gpt-5.6-sol"), responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	result := execution.Result
	require.Equal(t, ModelTraceVerdictMatch, result.Verdict)
	require.Len(t, f.upstream.requests, 5)
	require.Equal(t, 5, result.AttemptsMade)
	require.Equal(t, 3, result.ValidOutputs)
	require.Equal(t, ModelTraceRejectionTooFewNumbers, result.Outputs[0].Rejection)
	require.Equal(t, ModelTraceRejectionStreamError, result.Outputs[1].Rejection)
	require.Contains(t, result.Outputs[1].Error, "max_output_tokens")
	require.True(t, result.Outputs[2].Accepted)
}

func TestModelTraceProbe_SingleValidOutputIsInconclusive(t *testing.T) {
	sol := modelTraceReferenceTexts(t, "self-gpt-5.6-sol")
	responses := []*http.Response{modelTraceOpenAISSE(sol[0])}
	for i := 0; i < 5; i++ {
		responses = append(responses, modelTraceOpenAISSE("No."))
	}
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-5.6-sol", modelTraceOpenAIAccounts("gpt-5.6-sol"), responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Len(t, f.upstream.requests, 6)
	require.Equal(t, ModelTraceVerdictInconclusive, execution.Result.Verdict)
	require.Equal(t, []string{ModelTraceReasonInsufficient}, execution.Result.Reasons)
	require.Equal(t, 1, execution.Result.ValidOutputs)
	require.Equal(t, "", execution.State.ModelTraceStableStatus)
	require.Nil(t, execution.Event)
}

func TestModelTraceProbe_FailsOverAfterRetriesOn429(t *testing.T) {
	sol := modelTraceReferenceTexts(t, "self-gpt-5.6-sol")
	limited := func() *http.Response {
		return groupStatusProbeResponse(http.StatusTooManyRequests, `{"error":{"message":"rate limited"}}`)
	}
	responses := append([]*http.Response{limited(), limited(), limited()}, modelTraceOpenAIResponses(sol...)...)
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-5.6-sol", modelTraceOpenAIAccounts("gpt-5.6-sol"), responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Len(t, f.upstream.requests, 6)
	require.Equal(t, ModelTraceVerdictMatch, execution.Result.Verdict)
	require.Equal(t, int64(2), execution.Account.ID)
	require.Equal(t, 3, execution.Result.AttemptsMade)
	require.Contains(t, execution.Result.ErrorDetail, "account 1")
	// 失败账号上那次首条挑战也保留在诊断里（同一序号）
	require.Len(t, execution.Result.Outputs, 4)
	require.Equal(t, 3, execution.Result.Outputs[0].Attempts)
}

func TestModelTraceProbe_AllAccountsFailIsInconclusive(t *testing.T) {
	accounts := modelTraceOpenAIAccounts("gpt-5.6-sol")[:1]
	responses := []*http.Response{
		groupStatusProbeResponse(500, "boom"),
		groupStatusProbeResponse(500, "boom"),
		groupStatusProbeResponse(500, "boom"),
	}
	f := newModelTraceFixture(t, PlatformOpenAI, "gpt-5.6-sol", accounts, responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Equal(t, ModelTraceVerdictInconclusive, execution.Result.Verdict)
	require.Equal(t, []string{ModelTraceReasonNoAccount}, execution.Result.Reasons)
	require.NotNil(t, execution.Result.HTTPCode)
	require.Equal(t, 500, *execution.Result.HTTPCode)
	require.Equal(t, 0, f.notifier.calls)
}

// ---------- Anthropic ----------

func modelTraceAnthropicAccount(id int64, priority int) Account {
	a := groupStatusProbeAccount(id, PlatformAnthropic, 0, priority, map[string]any{"claude-opus-5-5": "claude-opus-5-5"})
	a.Credentials["api_key"] = "sk-ant-test"
	return a
}

func TestModelTraceProbe_AnthropicAPIKeyMatch(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	responses := []*http.Response{}
	for _, text := range opus {
		responses = append(responses, modelTraceAnthropicSSE(text, "end_turn"))
	}
	f := newModelTraceFixture(t, PlatformAnthropic, "claude-opus-5-5", []Account{modelTraceAnthropicAccount(1, 1)}, responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	result := execution.Result
	require.Equal(t, ModelTraceVerdictMatch, result.Verdict)
	require.Equal(t, "claude-opus-5-5", result.TopModel)
	require.Equal(t, int64(750), result.InputTokens)
	require.Equal(t, int64(3300), result.OutputTokens)
	require.Equal(t, "end_turn", result.Outputs[0].StopReason)
	require.False(t, result.Outputs[0].ThinkingPresent)

	req := f.upstream.requests[0]
	require.Equal(t, "https://example.com/v1/messages?beta=true", req.URL.String())
	require.Equal(t, "sk-ant-test", req.Header.Get("x-api-key"))
	require.Equal(t, claude.APIKeyBetaHeader, req.Header.Get("anthropic-beta"))
	require.Equal(t, "2023-06-01", req.Header.Get("anthropic-version"))
	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, "claude-opus-5-5", payload["model"])
	require.Equal(t, float64(groupStatusModelTraceAnthropicMaxTokens), payload["max_tokens"])
	require.Equal(t, true, payload["stream"])
	for _, key := range []string{"system", "temperature", "thinking", "metadata", "output_config"} {
		require.NotContains(t, payload, key)
	}
	messages := payload["messages"].([]any)
	require.Len(t, messages, 1)
	content := messages[0].(map[string]any)["content"].([]any)[0].(map[string]any)
	require.Equal(t, "text", content["type"])
	require.NotContains(t, content, "cache_control")
}

func TestModelTraceProbe_AnthropicOAuthCarriesClaudeCodeIdentity(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	responses := []*http.Response{}
	for _, text := range opus {
		responses = append(responses, modelTraceAnthropicSSE(text, "end_turn"))
	}
	account := modelTraceAnthropicAccount(1, 1)
	account.Type = AccountTypeOAuth
	account.Credentials["access_token"] = "oauth-token"
	f := newModelTraceFixture(t, PlatformAnthropic, "claude-opus-5-5", []Account{account}, responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Equal(t, ModelTraceVerdictMatch, execution.Result.Verdict)

	req := f.upstream.requests[0]
	require.Equal(t, testClaudeAPIURL, req.URL.String())
	require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
	require.Contains(t, req.Header.Get("anthropic-beta"), claude.BetaOAuth)
	payload := decodeProbeRequestBody(t, req)
	system := payload["system"].([]any)
	require.Len(t, system, 1)
	require.Equal(t, claudeCodeSystemPrompt, system[0].(map[string]any)["text"])
	require.NotEmpty(t, payload["metadata"].(map[string]any)["user_id"])
	require.NotContains(t, payload, "temperature")
	require.NotContains(t, payload, "thinking")
}

func TestModelTraceProbe_AnthropicMaxTokensIsRejected(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	responses := []*http.Response{modelTraceAnthropicSSE(opus[0], "max_tokens")}
	for _, text := range opus {
		responses = append(responses, modelTraceAnthropicSSE(text, "end_turn"))
	}
	f := newModelTraceFixture(t, PlatformAnthropic, "claude-opus-5-5", []Account{modelTraceAnthropicAccount(1, 1)}, responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Len(t, f.upstream.requests, 4)
	require.Equal(t, ModelTraceRejectionMaxTokens, execution.Result.Outputs[0].Rejection)
	require.False(t, execution.Result.Outputs[0].Accepted)
	require.Equal(t, ModelTraceVerdictMatch, execution.Result.Verdict)
}

func TestModelTraceProbe_AnthropicCrossFamilyMismatch(t *testing.T) {
	astra := modelTraceReferenceTexts(t, "astra-claimed-as-opus-5-5")
	responses := []*http.Response{}
	for round := 0; round < 2; round++ {
		for _, text := range astra {
			responses = append(responses, modelTraceAnthropicSSE(text, "end_turn"))
		}
	}
	f := newModelTraceFixture(t, PlatformAnthropic, "claude-opus-5-5", []Account{modelTraceAnthropicAccount(1, 1)}, responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.True(t, execution.Confirmed)
	require.Equal(t, ModelTraceStatusMismatch, execution.State.ModelTraceStableStatus)
	require.Equal(t, "gpt-6-astra", execution.Result.TopModel)
	require.Equal(t, 1, f.notifier.calls)
	require.Equal(t, "gpt", execution.Result.FamilyProbabilities[0].Family)
	require.Greater(t, execution.Result.FamilyProbabilities[0].Probability, 0.99)
}

func TestModelTraceProbe_SkipsBedrockAccounts(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	responses := []*http.Response{}
	for _, text := range opus {
		responses = append(responses, modelTraceAnthropicSSE(text, "end_turn"))
	}
	bedrock := modelTraceAnthropicAccount(1, 1)
	bedrock.Type = AccountTypeBedrock
	f := newModelTraceFixture(t, PlatformAnthropic, "claude-opus-5-5", []Account{bedrock, modelTraceAnthropicAccount(2, 2)}, responses...)

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Equal(t, ModelTraceVerdictMatch, execution.Result.Verdict)
	require.Equal(t, int64(2), execution.Account.ID)
	require.Len(t, f.upstream.requests, 3)
}

// ---------- 配置与平台 ----------

func TestModelTraceProbe_TargetNotAllowedMakesNoRequests(t *testing.T) {
	f := newModelTraceFixture(t, PlatformAnthropic, "gpt-6-sol", []Account{modelTraceAnthropicAccount(1, 1)})

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Empty(t, f.upstream.requests)
	require.Equal(t, ModelTraceVerdictInconclusive, execution.Result.Verdict)
	require.Equal(t, []string{ModelTraceReasonTargetNotAllowed}, execution.Result.Reasons)
	require.NotNil(t, execution.State.ModelTraceCheckedAt, "the run is still saved so the runner is throttled")
}

func TestModelTraceProbe_RejectsUnsupportedPlatform(t *testing.T) {
	f := newModelTraceFixture(t, PlatformGemini, "gpt-5.6-sol", nil)
	_, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.ErrorIs(t, err, ErrGroupStatusModelTraceUnsupported)
	require.Empty(t, f.upstream.requests)
}

func TestModelTraceProbe_DefaultsExpectedModelByPlatform(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	responses := []*http.Response{}
	for _, text := range opus {
		responses = append(responses, modelTraceAnthropicSSE(text, "end_turn"))
	}
	f := newModelTraceFixture(t, PlatformAnthropic, "claude-opus-5-5", []Account{modelTraceAnthropicAccount(1, 1)}, responses...)
	f.cfg.ModelTraceExpectedModel = ""

	execution, err := f.svc.probeModelTrace(context.Background(), f.group, f.cfg)
	require.NoError(t, err)
	require.Equal(t, "claude-opus-5-5", execution.Result.ExpectedModel)
	require.True(t, strings.HasPrefix(execution.State.ModelTraceDetail, "expected Claude Opus 5.5"))
}
