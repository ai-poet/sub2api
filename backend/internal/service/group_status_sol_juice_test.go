package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------- 归一与分类 ----------

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
		winner         string
		detailContains string
	}{
		{"40", solJuiceStatusPass, "40", "gpt-5.6-sol", ""},
		{"40.0", solJuiceStatusPass, "40", "gpt-5.6-sol", ""},
		{"```\n40\n```", solJuiceStatusPass, "40", "gpt-5.6-sol", ""},
		{"**40**", solJuiceStatusPass, "40", "gpt-5.6-sol", ""},
		{"4000", solJuiceStatusPass, "4000", "gpt-5.6-sol", ""},
		{"40.5", solJuiceStatusPass, "40.5", "gpt-5.6-sol", ""},
		{"32", solJuiceStatusMismatch, "32", "gpt-5.6-terra", "GPT-5.6 Terra"},
		{"48", solJuiceStatusMismatch, "48", "gpt-5.6-luna", "GPT-5.6 Luna"},
		{"96", solJuiceStatusMismatch, "96", "gpt-5.5/5.4", "GPT-5.5 / GPT-5.4"},
		{"64", solJuiceStatusMismatch, "64", "gpt-5.4-mini", "GPT-5.4 mini"},
		{"41", solJuiceStatusInconclusive, "41", "", "unknown juice value"},
		{"I can't provide that", solJuiceStatusInconclusive, "", "", "non-numeric"},
		{"", solJuiceStatusInconclusive, "", "", "empty answer"},
	}
	for _, tc := range cases {
		classification, value, winner, detail := ClassifySolJuiceAnswer(tc.raw)
		require.Equal(t, tc.classification, classification, "raw=%q", tc.raw)
		require.Equal(t, tc.value, value, "raw=%q", tc.raw)
		require.Equal(t, tc.winner, winner, "raw=%q", tc.raw)
		if tc.detailContains == "" {
			require.Empty(t, detail, "raw=%q", tc.raw)
		} else {
			require.Contains(t, detail, tc.detailContains, "raw=%q", tc.raw)
		}
	}
}

// ---------- 作为 gpt-5.6-sol 目标的检测方法 ----------

func juiceSSE(answer string) *http.Response {
	return groupStatusProbeResponse(200, responsesSSEBody(answer, 192))
}

func TestSolJuiceProbe_PassWithHighReasoning(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-5.6-sol"), juiceSSE("40"))

	executions := f.run(t)
	require.Len(t, executions, 1)
	result := executions[0].Result
	require.Equal(t, AstraCheckVerdictMatch, result.Verdict)
	require.Equal(t, "gpt-5.6-sol", result.Winner)
	require.Empty(t, result.Reasons)
	require.Equal(t, solJuiceScoringVersion, result.ScoringVersion)
	require.Equal(t, solJuicePackageID, result.BenchmarkPackageID)
	require.Equal(t, "high", result.Tier)
	require.Equal(t, 1, result.RequestsPlanned)
	require.Equal(t, 1, result.RequestsCompleted)
	require.Equal(t, 1, result.ValidSamples)
	require.Equal(t, 1, result.PlannedSamples)
	require.Equal(t, int64(192), result.ReasoningTokens)
	require.Contains(t, result.ErrorDetail, "juice 40 (expected 40)")
	require.Len(t, result.Cells, 1)
	require.Equal(t, map[string]int{"40": 1}, result.Cells[0].Categories)
	require.Len(t, result.Samples, 1)
	require.Equal(t, "40", result.Samples[0].Category)
	require.Equal(t, AstraCheckStatusPass, executions[0].State.StableStatus)
	require.Contains(t, executions[0].State.Detail, "juice 40")

	// 请求契约沿用原纯 Sol 验证：一条 user 消息、reasoning=high、不带输出上限
	require.Len(t, f.upstream.requests, 1)
	payload := decodeProbeRequestBody(t, f.upstream.requests[0])
	require.Equal(t, "gpt-5.6-sol", payload["model"])
	require.Equal(t, map[string]any{"effort": "high"}, payload["reasoning"])
	require.NotContains(t, payload, "max_output_tokens")
	require.NotContains(t, payload, "instructions")
	input := payload["input"].([]any)
	require.Len(t, input, 1)
	require.Equal(t, solJuicePromptText, input[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"])
}

func TestSolJuiceProbe_OtherFingerprintConfirmedOnSameAccount(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-5.6-sol"), juiceSSE("32"), juiceSSE("32"))

	execution := f.run(t)[0]
	require.True(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 2)
	require.Equal(t, AstraCheckVerdictMismatch, execution.Result.Verdict)
	require.Equal(t, "gpt-5.6-terra", execution.Result.Winner)
	require.Equal(t, *f.repo.results[0].AccountID, *f.repo.results[1].AccountID)
	require.Equal(t, AstraCheckStatusMismatch, execution.State.StableStatus)
	require.NotNil(t, execution.Event)
	require.Equal(t, GroupStatusEventAstraMismatch, execution.Event.EventType)
	require.Equal(t, "gpt-5.6-sol:winner_gpt-5.6-terra", execution.Event.SubStatus)
	require.Contains(t, execution.Event.ErrorDetail, "juice 32")
	require.Equal(t, 1, f.notifier.calls)
}

func TestSolJuiceProbe_NonNumericAnswerIsInsufficientWithoutRetry(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-5.6-sol"), juiceSSE("I can't share that"))

	execution := f.run(t)[0]
	require.Len(t, f.upstream.requests, 1)
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, []string{AstraCheckReasonJuiceNA}, execution.Result.Reasons)
	require.Equal(t, 0, execution.Result.ValidSamples)
	require.Contains(t, execution.Result.ErrorDetail, "non-numeric")
	require.Equal(t, "", execution.State.StableStatus)
	require.Nil(t, execution.Event)
}

func TestSolJuiceProbe_FailsOverToNextAccount(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-5.6-sol"),
		groupStatusProbeResponse(429, `{"error":{"message":"rate limited"}}`),
		juiceSSE("40"),
	)

	execution := f.run(t)[0]
	require.Len(t, f.upstream.requests, 2)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, int64(2), *execution.Result.AccountID)
	require.Contains(t, execution.Result.ErrorDetail, "account 1")
	require.Len(t, execution.Result.Samples, 2)
	require.Equal(t, AstraCheckSampleFailed, execution.Result.Samples[0].Outcome)
}

func TestSolJuiceProbe_AllAccountsFailingIsNoAccount(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-5.6-sol"),
		groupStatusProbeResponse(503, `{"error":{"message":"down"}}`),
		groupStatusProbeResponse(503, `{"error":{"message":"down"}}`),
	)

	execution := f.run(t)[0]
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, []string{AstraCheckReasonNoAccount}, execution.Result.Reasons)
	require.Equal(t, 503, *execution.Result.HTTPCode)
	require.Nil(t, execution.Event)
}

func TestSolJuiceProbe_MixedWithMeowTargets(t *testing.T) {
	responses := append([]*http.Response{juiceSSE("40")}, astraLike()...)
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-5.6-sol", "gpt-6-astra"), responses...)

	executions := f.run(t)
	require.Len(t, executions, 2)
	require.Equal(t, solJuiceScoringVersion, executions[0].Result.ScoringVersion)
	require.Equal(t, AstraCheckVerdictMatch, executions[0].Result.Verdict)
	require.Equal(t, astraBenchmarkScoringVersion, executions[1].Result.ScoringVersion)
	require.Equal(t, AstraCheckVerdictMatch, executions[1].Result.Verdict)
	require.Len(t, f.upstream.requests, 5)
}

func TestDecorateAstraCheckSummary_ReportsMethodPerModel(t *testing.T) {
	summary := &GroupStatusSummary{AstraCheckModels: onlyModels("gpt-5.6-sol", "gpt-6-sol")}
	decorateAstraCheckSummary(summary)
	require.Equal(t, AstraCheckMethodSolJuice, summary.AstraCheckStates[0].Method)
	require.Equal(t, AstraCheckMethodMeow, summary.AstraCheckStates[1].Method)
}

// Juice 沿用原纯 Sol 验证的选号：取调度器首选账号，不避让其他并行检测占用的账号。
func TestSolJuiceProbe_UsesSchedulerFirstChoiceEvenWhenBusy(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-5.6-sol"), juiceSSE("40"))
	done := f.svc.astraAccounts.use(1) // 账号 1 正被别的检测占用
	defer done()

	execution := f.run(t)[0]
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, int64(1), *execution.Result.AccountID)
}

// 对照：meow 目标会避开被占用的账号。
func TestAstraCheckProbe_MeowPrefersIdleAccount(t *testing.T) {
	f := newAstraProbeFixture(t, PlatformOpenAI, onlyModels("gpt-6-astra"), astraLike()...)
	done := f.svc.astraAccounts.use(1)
	defer done()

	execution := f.run(t)[0]
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, int64(2), *execution.Result.AccountID)
}

// 结果来源换了（meow 基准 → Juice）：旧来源的稳定结论与计数静默清零，不发事件。
func TestComputeAstraCheckTransition_ResetsWhenSourceChanges(t *testing.T) {
	prev := &GroupStatusAstraCheckState{
		GroupID:             30,
		ExpectedModel:       "gpt-5.6-sol",
		StableStatus:        AstraCheckStatusMismatch,
		ConsecutiveMismatch: 3,
		BenchmarkPackageID:  "meow-gpt-other-cap98",
	}
	juice := astraResult(AstraCheckVerdictMatch, "gpt-5.6-sol", "gpt-5.6-sol")
	juice.BenchmarkPackageID = solJuicePackageID
	next, event := ComputeAstraCheckTransition(prev, juice, 1)
	require.Nil(t, event, "no recovered push for a result from a different source")
	require.Equal(t, AstraCheckStatusPass, next.StableStatus)
	require.Equal(t, 0, next.ConsecutiveMismatch)
	require.Equal(t, solJuicePackageID, next.BenchmarkPackageID)

	// 旧来源留下 1 次不符：新来源第一次不符只算第 1 次，不会直接凑满 2 次变红
	prev = &GroupStatusAstraCheckState{ExpectedModel: "gpt-5.6-sol", StableStatus: AstraCheckStatusPass, ConsecutiveMismatch: 1, BenchmarkPackageID: "meow-gpt-other-cap98"}
	mismatch := astraResult(AstraCheckVerdictMismatch, "gpt-5.6-sol", "gpt-5.6-terra")
	mismatch.BenchmarkPackageID = solJuicePackageID
	next, event = ComputeAstraCheckTransition(prev, mismatch, 2)
	require.Nil(t, event)
	require.Equal(t, 1, next.ConsecutiveMismatch)
	require.Equal(t, "", next.StableStatus)

	// 同一来源（含包版本升级，包 id 不变）照常累计
	same := &GroupStatusAstraCheckState{StableStatus: AstraCheckStatusPass, ConsecutiveMismatch: 1, BenchmarkPackageID: solJuicePackageID}
	next, event = ComputeAstraCheckTransition(same, mismatch, 3)
	require.NotNil(t, event)
	require.Equal(t, AstraCheckStatusMismatch, next.StableStatus)
}
