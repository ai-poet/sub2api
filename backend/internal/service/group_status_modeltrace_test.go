package service

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	modeltracebank "github.com/Wei-Shaw/sub2api/resources/modeltrace-bank"
	"github.com/stretchr/testify/require"
)

// ---------- 指纹库 ----------

func TestLoadEmbeddedModelTraceBank(t *testing.T) {
	bank, meta, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	require.Len(t, bank.Models, 16)
	require.Equal(t, []string{"gpt", "claude"}, bank.FamilyOrder)
	require.Len(t, meta.SHA256, 64)
	// 与 ModelTrace 网页版 data/unified_bank.json 同一份
	require.Equal(t, "1c2cb74d372f9f0f30d0dabbb7b7a838660d2f769a88d0c8489e4c662e088c21", meta.SHA256)
	require.NotEmpty(t, meta.BuiltAt)
	for key := 1; key <= 3; key++ {
		require.Greater(t, bank.Calibration[key].Beta, 0.0)
	}
	require.Equal(t, "2026-09-23 1c2cb74d", modelTraceBankVersion(meta))
}

func TestModelTraceTargets_AreInEmbeddedBank(t *testing.T) {
	bank, _, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	found := 0
	for _, target := range astraCheckTargets {
		if target.Method != AstraCheckMethodModelTrace {
			require.Empty(t, target.TraceModelID, "target %s", target.ID)
			continue
		}
		found++
		require.Empty(t, target.PackageID, "modeltrace target %s must not point at a meow package", target.ID)
		require.True(t, bank.HasModel(target.TraceModelID), "target %s (%s) is missing from the fingerprint bank", target.ID, target.TraceModelID)
	}
	require.Equal(t, 2, found)
	opus, ok := astraCheckTarget("claude-opus-5.5")
	require.True(t, ok)
	require.Equal(t, AstraCheckMethodModelTrace, opus.Method)
	require.Equal(t, "claude-opus-5-5", opus.TraceModelID)
	opus5, ok := astraCheckTarget("claude-opus-5")
	require.True(t, ok)
	require.Equal(t, PlatformAnthropic, opus5.Platform)
	require.Equal(t, AstraCheckMethodModelTrace, opus5.Method)
	require.Equal(t, "claude-opus-5", opus5.TraceModelID)
	require.Equal(t, "claude-opus-5", opus5.DefaultRequestModel)
	// Fable 5.1 不在 ModelTrace 指纹库里，仍用 meow 基准
	fable, ok := astraCheckTarget("claude-fable-5.1")
	require.True(t, ok)
	require.Equal(t, AstraCheckMethodMeow, fable.Method)
	require.False(t, bank.HasModel("claude-fable-5-1"))

	require.Equal(t, "claude-opus-5.5", modelTraceCandidateID("claude-opus-5-5"))
	require.Equal(t, "claude-opus-5", modelTraceCandidateID("claude-opus-5"))
	require.Equal(t, "Claude Opus 5", AstraModelLabel("claude-opus-5"))
}

func mutateModelTraceBank(t *testing.T, mutate func(doc map[string]any)) []byte {
	t.Helper()
	var doc map[string]any
	require.NoError(t, json.Unmarshal(modeltracebank.Bank, &doc))
	mutate(doc)
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	return raw
}

func TestParseModelTraceBank_RejectsBrokenBanks(t *testing.T) {
	robust := func(doc map[string]any) map[string]any { return doc["robust"].(map[string]any) }
	cases := map[string]func(doc map[string]any){
		"schema": func(doc map[string]any) { doc["schema"] = "something-else" },
		"range": func(doc map[string]any) {
			doc["method"].(map[string]any)["range"] = []any{1, 100}
		},
		"model order": func(doc map[string]any) {
			order := robust(doc)["model_order"].([]any)
			order[0], order[1] = order[1], order[0]
		},
		"centroid rows": func(doc map[string]any) {
			hell := robust(doc)["hellinger"].(map[string]any)
			hell["centroids"] = hell["centroids"].([]any)[:3]
		},
		"feature dimension": func(doc map[string]any) {
			hell := robust(doc)["hellinger"].(map[string]any)
			hell["feature_mean"] = hell["feature_mean"].([]any)[:10]
		},
		"zero scale": func(doc map[string]any) {
			ob := robust(doc)["ordered_blocks"].(map[string]any)
			ob["feature_scale"].([]any)[3] = 0
		},
		"missing calibration": func(doc map[string]any) {
			delete(doc["calibration"].(map[string]any), "3")
		},
		"bad beta": func(doc map[string]any) {
			doc["calibration"].(map[string]any)["2"].(map[string]any)["beta"] = -1
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := ParseModelTraceBank(mutateModelTraceBank(t, mutate))
			require.Error(t, err)
		})
	}
}

// ---------- 挑战 ----------

func TestGenerateModelTraceChallenges(t *testing.T) {
	rng := rand.New(rand.NewChaCha8([32]byte{1, 2, 3}))
	challenges := GenerateModelTraceChallenges(rng, 6)
	require.Len(t, challenges, 6)
	seen := map[int]bool{}
	for i, ch := range challenges {
		require.GreaterOrEqual(t, ch.ExpectedCount, 292)
		require.LessOrEqual(t, ch.ExpectedCount, 332)
		require.False(t, seen[ch.ExpectedCount], "lengths must be distinct")
		seen[ch.ExpectedCount] = true
		require.True(t, strings.HasPrefix(ch.ID, "probe-"+strconv.Itoa(i+1)+"-"))
		require.Len(t, ch.ID, len("probe-")+len(strconv.Itoa(i+1))+1+14)
		require.Contains(t, ch.Prompt, " "+strconv.Itoa(ch.ExpectedCount)+" 个 1 到 355（含端点）的整数。")
		require.Contains(t, ch.Prompt, "禁止调用或借助任何工具")
		require.True(t, strings.HasSuffix(ch.Prompt, "直接从第一个取值开始输出，不要在序列前重复数量、范围或任务说明。"))
	}

	// 固定种子可复现
	again := GenerateModelTraceChallenges(rand.New(rand.NewChaCha8([32]byte{1, 2, 3})), 6)
	require.Equal(t, challenges, again)
	require.Len(t, GenerateModelTraceChallenges(rng, 100), 41)
}

// 措辞必须与 ModelTrace 网页版 challenge-browser.js 逐字一致（指纹只在建库时的提示分布下有效）。
func TestModelTraceChallengePools_MatchWebVersion(t *testing.T) {
	require.Equal(t, []string{"这是一次独立的数值选择记录", "请完成下面的无语义整数选择任务", "执行一次第一反应取值记录", "生成一组不承载语义的整数选择", "进行一轮快速逐项取值"}, modelTraceOpenings)
	require.Equal(t, []string{"为各个位置分别凭第一反应选择", "逐项选择", "每次只决定当前一项，共给出", "分别凭第一反应给出", "逐个直接选择"}, modelTraceActions)
	require.Len(t, modelTraceEndings, 5)
	require.Len(t, modelTraceSeparatorHints, 4)
	// 网页上实际出现过的一条挑战，按同样的片段拼出来必须逐字相同
	web := "执行一次第一反应取值记录。逐个直接选择 303 个 1 到 355（含端点）的整数。每个位置都要单独选择；不要从 1 开始计数，不要连续递增或递减，也不要采用等差、循环、重复区块或其他规则化模式。本任务必须由当前语言模型直接完成：禁止调用或借助任何工具，包括 Python、代码执行器、计算器、搜索、API 和外部随机数生成器；也不要先编写或运行代码。允许某个数字再次出现；每项写出后不要回头排序、去重或替换。只要每个整数边界清楚，格式可自行选择。直接从第一个取值开始输出，不要在序列前重复数量、范围或任务说明。"
	built := fmt.Sprintf("%s。%s %d 个 1 到 355（含端点）的整数。", modelTraceOpenings[2], modelTraceActions[4], 303) +
		"每个位置都要单独选择；不要从 1 开始计数，不要连续递增或递减，也不要采用等差、循环、重复区块或其他规则化模式。" +
		"本任务必须由当前语言模型直接完成：禁止调用或借助任何工具，包括 Python、代码执行器、" +
		"计算器、搜索、API 和外部随机数生成器；也不要先编写或运行代码。" +
		modelTraceEndings[0] + modelTraceSeparatorHints[3] +
		"直接从第一个取值开始输出，不要在序列前重复数量、范围或任务说明。"
	require.Equal(t, web, built)
}

func TestModelTraceMinimumNumbers(t *testing.T) {
	require.Equal(t, 80, ModelTraceMinimumNumbers(0))
	require.Equal(t, 80, ModelTraceMinimumNumbers(100))
	require.Equal(t, 161, ModelTraceMinimumNumbers(292))
	require.Equal(t, 183, ModelTraceMinimumNumbers(332))
	// 340×0.55 在浮点下是 187.00000000000003，参考实现（JS / Python）的 ceil 结果是 188，这里必须一致
	require.Equal(t, 188, ModelTraceMinimumNumbers(340))
}

func TestParseModelTraceNumbers_EdgeCases(t *testing.T) {
	require.Equal(t, []int{1, 2, 3}, ParseModelTraceNumbers("1,2,3"))
	// 字母分隔切段，取最长段；并列时取第一段
	require.Equal(t, []int{4, 5, 6}, ParseModelTraceNumbers("1 2 x 4 5 6 y 7"))
	require.Equal(t, []int{1, 2}, ParseModelTraceNumbers("1 2 a 3 4"))
	// 越界值丢弃但不切段
	require.Equal(t, []int{10, 20}, ParseModelTraceNumbers("10, 400, 0, 20"))
	require.Empty(t, ParseModelTraceNumbers("没有数字"))
}

// ---------- 判定 ----------

func modelTraceSyntheticAnalysis(used int, entries ...ModelTraceRankEntry) *ModelTraceAnalysis {
	return &ModelTraceAnalysis{UsedOutputs: used, Results: entries}
}

func TestClassifyModelTrace_Thresholds(t *testing.T) {
	bank, _, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	opus := "claude-opus-5-5"
	opus5 := "claude-opus-5"

	cases := []struct {
		name    string
		used    int
		top     ModelTraceRankEntry
		second  ModelTraceRankEntry
		verdict string
		reason  string
	}{
		{"match at 0.5", 3, ModelTraceRankEntry{Model: opus, Probability: 0.5}, ModelTraceRankEntry{Model: opus5, Probability: 0.4}, ModelTraceVerdictMatch, ""},
		{"top expected below 0.5", 3, ModelTraceRankEntry{Model: opus, Probability: 0.49}, ModelTraceRankEntry{Model: opus5, Probability: 0.3}, ModelTraceVerdictInconclusive, ModelTraceReasonAmbiguous},
		{"mismatch at thresholds", 3, ModelTraceRankEntry{Model: opus5, Probability: 0.8}, ModelTraceRankEntry{Model: opus, Probability: 0.15}, ModelTraceVerdictMismatch, ""},
		{"top other below 0.8", 3, ModelTraceRankEntry{Model: opus5, Probability: 0.79}, ModelTraceRankEntry{Model: opus, Probability: 0.1}, ModelTraceVerdictInconclusive, ModelTraceReasonAmbiguous},
		{"expected above 0.15", 3, ModelTraceRankEntry{Model: opus5, Probability: 0.84}, ModelTraceRankEntry{Model: opus, Probability: 0.16}, ModelTraceVerdictInconclusive, ModelTraceReasonAmbiguous},
		{"single output never decides", 1, ModelTraceRankEntry{Model: opus, Probability: 0.99}, ModelTraceRankEntry{Model: opus5, Probability: 0.01}, ModelTraceVerdictInconclusive, ModelTraceReasonInsufficient},
		{"no outputs", 0, ModelTraceRankEntry{Model: opus, Probability: 0.99}, ModelTraceRankEntry{Model: opus5, Probability: 0.01}, ModelTraceVerdictInconclusive, ModelTraceReasonNoValidOutputs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict, reasons := ClassifyModelTrace(modelTraceSyntheticAnalysis(tc.used, tc.top, tc.second), bank, opus)
			require.Equal(t, tc.verdict, verdict)
			if tc.reason == "" {
				require.Empty(t, reasons)
			} else {
				require.Equal(t, []string{tc.reason}, reasons)
			}
		})
	}

	verdict, reasons := ClassifyModelTrace(modelTraceSyntheticAnalysis(3), bank, "claude-fable-5-1")
	require.Equal(t, ModelTraceVerdictInconclusive, verdict)
	require.Equal(t, []string{ModelTraceReasonUnknownExpected}, reasons)
}

func TestModelTraceTopRanking_AppendsExpected(t *testing.T) {
	ranking := []ModelTraceRankEntry{{Model: "a"}, {Model: "b"}, {Model: "c"}, {Model: "d"}, {Model: "e"}, {Model: "f"}, {Model: "expected"}}
	top := modelTraceTopRanking(ranking, "expected")
	require.Len(t, top, 6)
	require.Equal(t, "expected", top[5].Model)
	require.Len(t, modelTraceTopRanking(ranking, "a"), 5)
}

// ---------- 作为 Claude Opus 5.5 目标的检测方法 ----------

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

func modelTraceAnthropicResponses(texts []string, stopReason string) []*http.Response {
	out := make([]*http.Response, 0, len(texts))
	for _, text := range texts {
		out = append(out, modelTraceAnthropicSSE(text, stopReason))
	}
	return out
}

func newModelTraceProbeFixture(t *testing.T, responses ...*http.Response) *astraProbeFixture {
	t.Helper()
	return newModelTraceProbeFixtureFor(t, "claude-opus-5.5", responses...)
}

func newModelTraceProbeFixtureFor(t *testing.T, expected string, responses ...*http.Response) *astraProbeFixture {
	t.Helper()
	f := newAstraProbeFixture(t, PlatformAnthropic, onlyModels(expected), responses...)
	f.svc.modelTraceChallengeGen = modelTraceFixedChallenges
	return f
}

func TestModelTraceProbe_OpusMatch(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	f := newModelTraceProbeFixture(t, modelTraceAnthropicResponses(opus, "end_turn")...)

	execution := f.run(t)[0]
	result := execution.Result
	require.Equal(t, AstraCheckVerdictMatch, result.Verdict)
	require.Equal(t, "claude-opus-5.5", result.Winner)
	require.Equal(t, "claude-opus-5.5", result.Strongest)
	require.Empty(t, result.Reasons)
	require.Equal(t, modelTraceScoringVersion, result.ScoringVersion)
	require.Equal(t, modelTracePackageID, result.BenchmarkPackageID)
	require.Equal(t, "2026-09-23 1c2cb74d", result.BenchmarkVersion)
	require.Equal(t, 6, result.RequestsPlanned)
	require.Equal(t, 3, result.RequestsCompleted)
	require.Equal(t, 3, result.ValidSamples)
	require.Equal(t, 3, result.PlannedSamples)
	require.Equal(t, int64(750), result.InputTokens)
	require.Equal(t, int64(3300), result.OutputTokens)
	require.Contains(t, result.ErrorDetail, "outputs 3/3")
	require.Len(t, f.upstream.requests, 3)

	// 候选匹配：第一名是预期模型（用目标 id），概率越过 50% 的一致线
	require.NotEmpty(t, result.Matches)
	require.Equal(t, "claude-opus-5.5", result.Matches[0].Model)
	require.Equal(t, "Claude Opus 5.5", result.Matches[0].Name)
	require.True(t, result.Matches[0].Passed)
	require.Equal(t, modelTraceMatchMinProbability, result.Matches[0].Threshold)
	require.Greater(t, result.Matches[0].Match, 0.5)

	require.Len(t, result.Samples, 3)
	require.Equal(t, "challenge-1", result.Samples[0].CellID)
	require.Equal(t, AstraCheckSampleValid, result.Samples[0].Outcome)
	require.Equal(t, AstraCheckStatusPass, execution.State.StableStatus)
	require.Contains(t, execution.State.Detail, "expected Claude Opus 5.5")

	// 请求贴近建库环境：一条 user 消息、max_tokens 4096，不带 system / temperature / thinking
	req := f.upstream.requests[0]
	require.Equal(t, "sk-test", req.Header.Get("x-api-key"))
	require.Equal(t, claude.APIKeyBetaHeader, req.Header.Get("anthropic-beta"))
	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, "claude-opus-5-5", payload["model"])
	require.Equal(t, float64(groupStatusModelTraceAnthropicMaxTokens), payload["max_tokens"])
	for _, key := range []string{"system", "temperature", "thinking", "metadata", "output_config"} {
		require.NotContains(t, payload, key)
	}
	messages := payload["messages"].([]any)
	require.Len(t, messages, 1)
	require.Equal(t, "challenge 1: 300 个 1 到 355（含端点）的整数", messages[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"])
}

func TestModelTraceProbe_Opus5Match(t *testing.T) {
	opus5 := modelTraceReferenceTexts(t, "self-claude-opus-5")
	f := newModelTraceProbeFixtureFor(t, "claude-opus-5", modelTraceAnthropicResponses(opus5, "end_turn")...)

	execution := f.run(t)[0]
	result := execution.Result
	require.Equal(t, AstraCheckVerdictMatch, result.Verdict)
	require.Equal(t, "claude-opus-5", result.Winner)
	require.Equal(t, modelTracePackageID, result.BenchmarkPackageID)
	require.Equal(t, 3, result.ValidSamples)
	require.Equal(t, "Claude Opus 5", result.Matches[0].Name)
	require.True(t, result.Matches[0].Passed)
	require.Equal(t, AstraCheckStatusPass, execution.State.StableStatus)
	require.Len(t, f.upstream.requests, 3)
	require.Equal(t, "claude-opus-5", decodeProbeRequestBody(t, f.upstream.requests[0])["model"])
}

// Opus 5 与 Opus 5.5 在指纹库里分得开：互换答案会被判成对方，而不是证据不足。
func TestModelTraceProbe_Opus5AndOpus55TellEachOtherApart(t *testing.T) {
	cases := []struct {
		expected, reference, winner string
	}{
		{expected: "claude-opus-5", reference: "self-claude-opus-5-5", winner: "claude-opus-5.5"},
		{expected: "claude-opus-5.5", reference: "self-claude-opus-5", winner: "claude-opus-5"},
	}
	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			texts := modelTraceReferenceTexts(t, tc.reference)
			responses := append(modelTraceAnthropicResponses(texts, "end_turn"), modelTraceAnthropicResponses(texts, "end_turn")...)
			f := newModelTraceProbeFixtureFor(t, tc.expected, responses...)

			execution := f.run(t)[0]
			require.True(t, execution.Confirmed)
			require.Equal(t, AstraCheckVerdictMismatch, execution.Result.Verdict)
			require.Equal(t, tc.winner, execution.Result.Winner)
			require.Equal(t, AstraCheckStatusMismatch, execution.State.StableStatus)
			require.NotNil(t, execution.Event)
			require.Equal(t, tc.expected+":winner_"+tc.winner, execution.Event.SubStatus)
		})
	}
}

func TestModelTraceProbe_CrossFamilyMismatchConfirmedOnSameAccount(t *testing.T) {
	astra := modelTraceReferenceTexts(t, "astra-claimed-as-opus-5-5")
	responses := append(modelTraceAnthropicResponses(astra, "end_turn"), modelTraceAnthropicResponses(astra, "end_turn")...)
	f := newModelTraceProbeFixture(t, responses...)

	execution := f.run(t)[0]
	require.True(t, execution.Confirmed)
	require.Len(t, f.upstream.requests, 6)
	require.Equal(t, AstraCheckVerdictMismatch, execution.Result.Verdict)
	require.Equal(t, "gpt-6-astra", execution.Result.Winner)
	require.Equal(t, *f.repo.results[0].AccountID, *f.repo.results[1].AccountID)
	require.Equal(t, AstraCheckStatusMismatch, execution.State.StableStatus)
	require.NotNil(t, execution.Event)
	require.Equal(t, "claude-opus-5.5:winner_gpt-6-astra", execution.Event.SubStatus)
	require.Equal(t, 1, f.notifier.calls)
}

func TestModelTraceProbe_RejectedOutputIsReplacedByNextChallenge(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	responses := append([]*http.Response{modelTraceAnthropicSSE(opus[0], "max_tokens")}, modelTraceAnthropicResponses(opus, "end_turn")...)
	f := newModelTraceProbeFixture(t, responses...)

	execution := f.run(t)[0]
	require.Len(t, f.upstream.requests, 4)
	require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
	require.Equal(t, 3, execution.Result.ValidSamples)
	require.Equal(t, 4, execution.Result.RequestsCompleted)
	require.Equal(t, AstraCheckSampleInvalid, execution.Result.Samples[0].Outcome)
	require.Contains(t, execution.Result.Samples[0].Error, ModelTraceRejectionMaxTokens)
}

func TestModelTraceProbe_SingleValidOutputIsInsufficient(t *testing.T) {
	opus := modelTraceReferenceTexts(t, "self-claude-opus-5-5")
	responses := []*http.Response{modelTraceAnthropicSSE(opus[0], "end_turn")}
	for i := 0; i < 5; i++ {
		responses = append(responses, modelTraceAnthropicSSE("我不能完成这个任务。", "end_turn"))
	}
	f := newModelTraceProbeFixture(t, responses...)

	execution := f.run(t)[0]
	require.Len(t, f.upstream.requests, 6)
	require.Equal(t, AstraCheckVerdictInsufficient, execution.Result.Verdict)
	require.Equal(t, []string{ModelTraceReasonInsufficient}, execution.Result.Reasons)
	require.Equal(t, 1, execution.Result.ValidSamples)
	require.Nil(t, execution.Event)
}

func TestModelTraceProbe_OAuthCarriesClaudeCodeIdentity(t *testing.T) {
	f := newModelTraceProbeFixture(t)
	account := &Account{ID: 9, Platform: PlatformAnthropic, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "oauth-token"}}

	req, err := f.svc.buildAnthropicMessagesProbeRequest(t.Context(), account, "claude-opus-5-5", anthropicProbeHeadersClaudeCode, func(modelID string, isOAuth bool) (map[string]any, error) {
		return createAnthropicModelTracePayload(modelID, "prompt", isOAuth)
	})
	require.NoError(t, err)
	require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
	require.Contains(t, req.Header.Get("anthropic-beta"), claude.BetaOAuth)
	payload := decodeProbeRequestBody(t, req)
	system := payload["system"].([]any)
	require.Len(t, system, 1)
	require.Equal(t, claudeCodeSystemPrompt, system[0].(map[string]any)["text"])
	require.NotEmpty(t, payload["metadata"].(map[string]any)["user_id"])
	require.NotContains(t, payload, "thinking")
}

func TestModelTraceProbe_FableStaysOnMeow(t *testing.T) {
	summary := &GroupStatusSummary{AstraCheckModels: onlyModels("claude-opus-5.5", "claude-opus-5", "claude-fable-5.1")}
	decorateAstraCheckSummary(summary)
	require.Equal(t, AstraCheckMethodModelTrace, summary.AstraCheckStates[0].Method)
	require.Equal(t, AstraCheckMethodModelTrace, summary.AstraCheckStates[1].Method)
	require.Equal(t, AstraCheckMethodMeow, summary.AstraCheckStates[2].Method)
}
