package service

import (
	"encoding/json"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
	"time"

	modeltracebank "github.com/Wei-Shaw/sub2api/resources/modeltrace-bank"
	"github.com/stretchr/testify/require"
)

// ---------- 指纹库 ----------

func TestLoadEmbeddedModelTraceBank(t *testing.T) {
	bank, meta, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	require.Len(t, bank.Models, 16)
	require.Equal(t, []string{"gpt", "claude"}, bank.FamilyOrder)
	require.Equal(t, "GPT", bank.FamilyNames["gpt"])
	require.Equal(t, "Claude", bank.FamilyNames["claude"])
	require.Len(t, meta.SHA256, 64)
	require.NotEmpty(t, meta.BuiltAt)
	require.Equal(t, 0.25, bank.OrdWeight)
	for key := 1; key <= 3; key++ {
		require.Greater(t, bank.Calibration[key].Beta, 0.0)
	}
}

func TestModelTraceTargets_AreAllInEmbeddedBank(t *testing.T) {
	bank, _, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	for platform, ids := range modelTraceTargetsByPlatform {
		require.NotEmpty(t, ids, platform)
		for _, id := range ids {
			require.True(t, bank.HasModel(id), "target %s (%s) is missing from the fingerprint bank", id, platform)
		}
	}
	require.Equal(t, []string{"gpt-5.6-sol", "gpt-6-sol", "gpt-6-astra"}, modelTraceTargetsByPlatform[PlatformOpenAI])
	require.Equal(t, []string{"claude-opus-5-5"}, modelTraceTargetsByPlatform[PlatformAnthropic])
	require.Equal(t, "gpt-5.6-sol", modelTraceDefaultExpected(PlatformOpenAI))
	require.Equal(t, "claude-opus-5-5", modelTraceDefaultExpected(PlatformAnthropic))
	require.Empty(t, modelTraceDefaultExpected(PlatformGemini))
	require.Empty(t, ModelTraceTargetsForPlatform(PlatformGrok))
	require.Equal(t, "GPT-6 Astra", ModelTraceTargetsForPlatform(PlatformOpenAI)[2].DisplayName)
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
	sol := "gpt-5.6-sol"
	terra := "gpt-5.6-terra"

	cases := []struct {
		name    string
		used    int
		top     ModelTraceRankEntry
		second  ModelTraceRankEntry
		verdict string
		reason  string
	}{
		{"match at 0.5", 3, ModelTraceRankEntry{Model: sol, Probability: 0.5}, ModelTraceRankEntry{Model: terra, Probability: 0.4}, ModelTraceVerdictMatch, ""},
		{"top expected below 0.5", 3, ModelTraceRankEntry{Model: sol, Probability: 0.49}, ModelTraceRankEntry{Model: terra, Probability: 0.3}, ModelTraceVerdictInconclusive, ModelTraceReasonAmbiguous},
		{"mismatch at thresholds", 3, ModelTraceRankEntry{Model: terra, Probability: 0.8}, ModelTraceRankEntry{Model: sol, Probability: 0.15}, ModelTraceVerdictMismatch, ""},
		{"top other below 0.8", 3, ModelTraceRankEntry{Model: terra, Probability: 0.79}, ModelTraceRankEntry{Model: sol, Probability: 0.1}, ModelTraceVerdictInconclusive, ModelTraceReasonAmbiguous},
		{"expected above 0.15", 3, ModelTraceRankEntry{Model: terra, Probability: 0.84}, ModelTraceRankEntry{Model: sol, Probability: 0.16}, ModelTraceVerdictInconclusive, ModelTraceReasonAmbiguous},
		{"single output never decides", 1, ModelTraceRankEntry{Model: sol, Probability: 0.99}, ModelTraceRankEntry{Model: terra, Probability: 0.01}, ModelTraceVerdictInconclusive, ModelTraceReasonInsufficient},
		{"no outputs", 0, ModelTraceRankEntry{Model: sol, Probability: 0.99}, ModelTraceRankEntry{Model: terra, Probability: 0.01}, ModelTraceVerdictInconclusive, ModelTraceReasonNoValidOutputs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict, _, reasons := ClassifyModelTrace(modelTraceSyntheticAnalysis(tc.used, tc.top, tc.second), bank, sol)
			require.Equal(t, tc.verdict, verdict)
			if tc.reason == "" {
				require.Empty(t, reasons)
			} else {
				require.Equal(t, []string{tc.reason}, reasons)
			}
		})
	}

	verdict, _, reasons := ClassifyModelTrace(modelTraceSyntheticAnalysis(3), bank, "claude-fable-5-1")
	require.Equal(t, ModelTraceVerdictInconclusive, verdict)
	require.Equal(t, []string{ModelTraceReasonUnknownExpected}, reasons)
}

// ---------- 状态机 ----------

func modelTraceResult(verdict, expected, top string) *GroupStatusModelTraceResult {
	p := 0.01
	return &GroupStatusModelTraceResult{
		GroupID:             30,
		ConfigID:            101,
		ExpectedModel:       expected,
		Verdict:             verdict,
		TopModel:            top,
		TopProbability:      0.99,
		ExpectedProbability: &p,
		ValidOutputs:        3,
		AttemptsMade:        3,
		AttemptsPlanned:     6,
		BankSHA256:          "1c2cb74d372f9f0f",
		Ranking: []ModelTraceRankEntry{
			{Model: top, Probability: 0.99}, {Model: "a"}, {Model: "b"}, {Model: "c"}, {Model: "d"}, {Model: "e"}, {Model: expected, Probability: 0.01},
		},
		InputTokens:  900,
		OutputTokens: 3000,
		CostUSD:      0.12,
		FinishedAt:   time.Now(),
	}
}

func TestComputeModelTraceTransition(t *testing.T) {
	sol := "gpt-5.6-sol"

	next, event := ComputeModelTraceTransition(nil, modelTraceResult(ModelTraceVerdictMatch, sol, sol), 7)
	require.Nil(t, event)
	require.Equal(t, ModelTraceStatusPass, next.ModelTraceStableStatus)
	require.Equal(t, sol, next.ModelTraceRunExpectedModel)
	require.Equal(t, int64(7), *next.ModelTraceLastRunID)
	require.Equal(t, 0.12, next.ModelTraceLastCostUSD)

	// 一次 mismatch 只累计
	next, event = ComputeModelTraceTransition(next, modelTraceResult(ModelTraceVerdictMismatch, sol, "gpt-5.6-terra"), 8)
	require.Nil(t, event)
	require.Equal(t, 1, next.ModelTraceConsecutiveMismatch)
	require.Equal(t, ModelTraceStatusPass, next.ModelTraceStableStatus)

	// inconclusive 不清零、不改稳定结论
	next, event = ComputeModelTraceTransition(next, modelTraceResult(ModelTraceVerdictInconclusive, sol, ""), 9)
	require.Nil(t, event)
	require.Equal(t, 1, next.ModelTraceConsecutiveMismatch)
	require.Equal(t, ModelTraceVerdictInconclusive, next.ModelTraceVerdict)

	// 第二次 mismatch 切换并发事件
	next, event = ComputeModelTraceTransition(next, modelTraceResult(ModelTraceVerdictMismatch, sol, "gpt-5.6-terra"), 10)
	require.NotNil(t, event)
	require.Equal(t, GroupStatusEventModelTraceMismatch, event.EventType)
	require.Equal(t, ModelTraceStatusPass, event.FromStatus)
	require.Equal(t, ModelTraceStatusMismatch, event.ToStatus)
	require.Equal(t, "top_gpt-5.6-terra", event.SubStatus)
	require.Contains(t, event.ErrorDetail, "expected GPT-5.6 Sol 1.0%")
	require.Contains(t, event.ErrorDetail, "top GPT-5.6 Terra 99.0%")
	require.Equal(t, ModelTraceStatusMismatch, next.ModelTraceStableStatus)

	// 已是 mismatch 不重复发事件
	next, event = ComputeModelTraceTransition(next, modelTraceResult(ModelTraceVerdictMismatch, sol, "gpt-5.6-terra"), 11)
	require.Nil(t, event)
	require.Equal(t, 3, next.ModelTraceConsecutiveMismatch)

	// 一次 match 即恢复
	next, event = ComputeModelTraceTransition(next, modelTraceResult(ModelTraceVerdictMatch, sol, sol), 12)
	require.NotNil(t, event)
	require.Equal(t, GroupStatusEventModelTraceRecovered, event.EventType)
	require.Equal(t, 0, next.ModelTraceConsecutiveMismatch)
	require.Equal(t, ModelTraceStatusPass, next.ModelTraceStableStatus)
}

func TestComputeModelTraceTransition_ExpectedChangeResetsSilently(t *testing.T) {
	prev := &GroupStatusState{
		GroupID:                       30,
		ModelTraceStableStatus:        ModelTraceStatusMismatch,
		ModelTraceConsecutiveMismatch: 4,
		ModelTraceRunExpectedModel:    "gpt-6-sol",
	}
	// 改成 5.6-sol 后第一次 match：不发恢复事件
	next, event := ComputeModelTraceTransition(prev, modelTraceResult(ModelTraceVerdictMatch, "gpt-5.6-sol", "gpt-5.6-sol"), 1)
	require.Nil(t, event)
	require.Equal(t, ModelTraceStatusPass, next.ModelTraceStableStatus)
	require.Equal(t, "gpt-5.6-sol", next.ModelTraceRunExpectedModel)

	// 改预期后第一次 mismatch 只算第一次
	next, event = ComputeModelTraceTransition(prev, modelTraceResult(ModelTraceVerdictMismatch, "gpt-6-astra", "gpt-6-sol"), 2)
	require.Nil(t, event)
	require.Equal(t, 1, next.ModelTraceConsecutiveMismatch)
	require.Equal(t, "", next.ModelTraceStableStatus)
}

func TestComputeModelTraceTransition_DoesNotTouchOtherProbes(t *testing.T) {
	prev := &GroupStatusState{
		GroupID:                30,
		StableStatus:           GroupRuntimeStatusDown,
		ConsecutiveDown:        3,
		AstraCheckStableStatus: AstraCheckStatusMismatch,
		AstraCheckWinner:       "gpt-5.6-sol",
	}
	next, _ := ComputeModelTraceTransition(prev, modelTraceResult(ModelTraceVerdictMatch, "gpt-5.6-sol", "gpt-5.6-sol"), 1)
	require.Equal(t, GroupRuntimeStatusDown, next.StableStatus)
	require.Equal(t, 3, next.ConsecutiveDown)
	require.Equal(t, AstraCheckStatusMismatch, next.AstraCheckStableStatus)
	require.Equal(t, "gpt-5.6-sol", next.AstraCheckWinner)
	require.Equal(t, "", prev.ModelTraceStableStatus)
}

func TestModelTraceTopRanking_AppendsExpected(t *testing.T) {
	result := modelTraceResult(ModelTraceVerdictMismatch, "gpt-5.6-sol", "gpt-5.6-terra")
	ranking := modelTraceTopRanking(result.Ranking, result.ExpectedModel)
	require.Len(t, ranking, 6)
	require.Equal(t, "gpt-5.6-terra", ranking[0].Model)
	require.Equal(t, "gpt-5.6-sol", ranking[5].Model)

	inTop := modelTraceTopRanking(result.Ranking, "a")
	require.Len(t, inTop, 5)
}

func TestModelTraceDetailText_HasNoAccountOrPrompt(t *testing.T) {
	result := modelTraceResult(ModelTraceVerdictMismatch, "gpt-5.6-sol", "gpt-5.6-terra")
	accountID := int64(4242)
	result.AccountID = &accountID
	result.Outputs = []ModelTraceOutputRecord{{Prompt: "这是一次独立的数值选择记录"}}
	detail := modelTraceDetailText(result)
	require.NotContains(t, detail, "4242")
	require.NotContains(t, detail, "数值选择")
	require.Contains(t, detail, "outputs 3/3 (attempts 3/6)")
	require.Contains(t, detail, "bank 1c2cb74d")
}

// ---------- 配置 ----------

func TestNormalizeGroupStatusConfig_ModelTrace(t *testing.T) {
	base := func() *GroupStatusConfigUpsertInput {
		return &GroupStatusConfigUpsertInput{
			Enabled:        true,
			ProbeModel:     "gpt-5.6-sol",
			ProbePrompt:    "ping",
			ValidationMode: GroupStatusValidationNonEmpty,
		}
	}
	enabled := true
	openAI := &Group{ID: 7, Platform: PlatformOpenAI}
	anthropic := &Group{ID: 8, Platform: PlatformAnthropic}
	gemini := &Group{ID: 9, Platform: PlatformGemini}

	cfg, err := NormalizeGroupStatusConfig(openAI, base())
	require.NoError(t, err)
	require.False(t, cfg.ModelTraceEnabled)
	require.Equal(t, "gpt-5.6-sol", cfg.ModelTraceExpectedModel)
	require.Equal(t, 3600, cfg.ModelTraceIntervalSeconds)

	cfg, err = NormalizeGroupStatusConfig(anthropic, base())
	require.NoError(t, err)
	require.Equal(t, "claude-opus-5-5", cfg.ModelTraceExpectedModel)

	on := base()
	on.ModelTraceEnabled = &enabled
	on.ModelTraceExpectedModel = " gpt-6-astra "
	on.ModelTraceRequestModel = " astra-alias "
	on.ModelTraceIntervalSeconds = 1800
	cfg, err = NormalizeGroupStatusConfig(openAI, on)
	require.NoError(t, err)
	require.True(t, cfg.ModelTraceEnabled)
	require.Equal(t, "gpt-6-astra", cfg.ModelTraceExpectedModel)
	require.Equal(t, "astra-alias", cfg.ModelTraceRequestModel)
	require.Equal(t, 1800, cfg.ModelTraceIntervalSeconds)

	// Claude 目标不能配给 GPT 分组
	wrong := base()
	wrong.ModelTraceExpectedModel = "claude-opus-5-5"
	_, err = NormalizeGroupStatusConfig(openAI, wrong)
	require.ErrorIs(t, err, ErrGroupStatusModelTraceTargetInvalid)

	// 不支持的平台不能开启
	geminiOn := base()
	geminiOn.ModelTraceEnabled = &enabled
	_, err = NormalizeGroupStatusConfig(gemini, geminiOn)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)

	tooFast := base()
	tooFast.ModelTraceIntervalSeconds = 600
	_, err = NormalizeGroupStatusConfig(openAI, tooFast)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
}

// ---------- 推送 ----------

func TestBuildGroupStatusNotifyMessage_ModelTraceEvents(t *testing.T) {
	group := &Group{ID: 7, Name: "Sol 高速", Platform: PlatformOpenAI}
	mismatch := &GroupStatusEvent{
		EventType:   GroupStatusEventModelTraceMismatch,
		FromStatus:  ModelTraceStatusPass,
		ToStatus:    ModelTraceStatusMismatch,
		SubStatus:   "top_gpt-5.6-terra",
		ErrorDetail: "expected GPT-5.6 Sol 0.0% · top GPT-5.6 Terra 100.0%",
		ObservedAt:  time.Now(),
	}
	title, desp := buildGroupStatusNotifyMessage("My Gateway", group, mismatch)
	require.Equal(t, "[My Gateway] 分组「Sol 高速」ModelTrace 指纹不符（强指向 GPT-5.6 Terra）", title)
	require.Contains(t, desp, "**状态**：指纹一致 → 指纹不符")
	require.Contains(t, desp, "GPT-5.6 Terra")

	recovered := &GroupStatusEvent{
		EventType:  GroupStatusEventModelTraceRecovered,
		FromStatus: ModelTraceStatusMismatch,
		ToStatus:   ModelTraceStatusPass,
		SubStatus:  "top_gpt-5.6-sol",
		ObservedAt: time.Now(),
	}
	title, desp = buildGroupStatusNotifyMessage("My Gateway", group, recovered)
	require.Equal(t, "[My Gateway] 分组「Sol 高速」ModelTrace 指纹验证已恢复", title)
	require.Contains(t, desp, "**状态**：指纹不符 → 指纹一致")
}

func TestGroupStatusNotifyService_NotifyTransition_AcceptsModelTraceEvents(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", "sctp_secret"))

	cfg := &GroupStatusConfig{GroupID: 7, NotifyEnabled: true}
	svc.NotifyTransition(&Group{ID: 7, Name: "G7"}, cfg, &GroupStatusEvent{
		EventType:  GroupStatusEventModelTraceMismatch,
		FromStatus: ModelTraceStatusPass,
		ToStatus:   ModelTraceStatusMismatch,
		SubStatus:  "top_claude-opus-5",
		ObservedAt: time.Now(),
	})
	require.Eventually(t, func() bool { return ts.count() == 1 }, 5*time.Second, 10*time.Millisecond)
	require.Contains(t, ts.request(0).Title, "ModelTrace 指纹不符（强指向 Claude Opus 5）")

	// 分组关闭推送时同样静默
	silent := &GroupStatusConfig{GroupID: 7, NotifyEnabled: false}
	svc.NotifyTransition(&Group{ID: 7, Name: "G7"}, silent, &GroupStatusEvent{EventType: GroupStatusEventModelTraceRecovered, ObservedAt: time.Now()})
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 1, ts.count())
}
