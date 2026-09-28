package service

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	astrabenchmark "github.com/Wei-Shaw/sub2api/resources/astra-benchmark"
	"github.com/stretchr/testify/require"
)

// ---------- 合成 v3 基准包 ----------

// syntheticAstraPackage 生成一个小而确定的 v3 包：两道题，每个真实候选 / 参考源各偏好一个答案。
// 包 id 与目标列表里的真实包 id 一致，方便探测测试通过注册表按目标找到它。
func syntheticAstraPackage(t *testing.T, packageID, mode string, models []string, profile string) *AstraBenchmark {
	t.Helper()
	sources := append(append([]string{}, models...), "ref/other")
	letters := []string{"a", "b", "c", "d", "e"}
	numbers := []string{"1", "2", "3", "4", "5"}
	alphaFor := func(categories []string, preferred string) []float64 {
		out := make([]float64, len(categories))
		for i, category := range categories {
			out[i] = 0.2
			if category == preferred {
				out[i] = 20.2
			}
		}
		return out
	}
	cell := func(id string, answers []string) (map[string]any, map[string]any) {
		categories := append(append([]string{}, answers[:len(sources)]...), astraBenchmarkUnseenCategory)
		alpha := map[string]any{}
		for i, source := range sources {
			alpha[source] = alphaFor(categories, answers[i])
		}
		probe := map[string]any{
			"id":         id,
			"normalizer": map[string]any{"id": "exact_trimmed_casefold", "parameters": map[string]any{"max_length": 4096}},
			"cells": []any{map[string]any{
				"id": id, "system": ".", "prompt": "prompt " + id, "history": []any{}, "effort": "low",
				"profile": profile, "parameters": map[string]any{"max_output_tokens": 128},
			}},
		}
		return probe, map[string]any{"categories": categories, "alpha": alpha}
	}
	probe1, fitted1 := cell("c1", letters)
	probe2, fitted2 := cell("c2", numbers)

	modelEntries := []any{}
	fittedModels := []any{}
	thresholds := map[string]any{}
	for _, model := range models {
		modelEntries = append(modelEntries, map[string]any{"id": model, "name": model, "request_model": "vendor/" + model})
		fittedModels = append(fittedModels, model)
		thresholds[model] = 0.5
	}
	modelEntries = append(modelEntries, map[string]any{"id": AstraCheckOtherModel, "name": "other", "request_model": "reference-only:other", "reference_only": true})
	fittedModels = append(fittedModels, AstraCheckOtherModel)
	thresholds[AstraCheckOtherModel] = 0.5

	tier := func(n int) map[string]any {
		return map[string]any{"counts": map[string]any{"c1": n, "c2": n}, "thresholds": thresholds}
	}
	pkg := map[string]any{
		"mode": mode, "id": packageID, "version": "test-1", "schema_version": 1,
		"engine": map[string]any{"scoring_version": astraBenchmarkScoringVersion, "completion_ratio": 0.6, "prior_mass": 1},
		"models": modelEntries,
		"probes": []any{probe1, probe2},
		"tiers":  map[string]any{"low": tier(2), "medium": tier(3), "high": tier(4)},
		"fitted": map[string]any{
			"scoring_version": astraBenchmarkScoringVersion, "prior_mass": 1, "models": fittedModels,
			"sources": sources, "reference_sources": []any{"ref/other"}, "aggregation": astraBenchmarkAggregation,
			"cells": map[string]any{"c1": fitted1, "c2": fitted2},
		},
		"calibration": map[string]any{"status": "calibrated", "tiers": map[string]any{
			"low":    map[string]any{"result": map[string]any{"status": "target_met"}},
			"medium": map[string]any{"result": map[string]any{"status": "target_met"}},
			"high":   map[string]any{"result": map[string]any{"status": "target_met"}},
		}},
	}
	raw, err := json.Marshal(pkg)
	require.NoError(t, err)
	bench, _, err := ParseAstraBenchmark(raw)
	require.NoError(t, err)
	return bench
}

func syntheticGPTPackage(t *testing.T) *AstraBenchmark {
	return syntheticAstraPackage(t, "meow-gpt-other-cap98-efficient", astraBenchmarkModeGPT, []string{"gpt-6-astra", "gpt-6-sol"}, "standard")
}

func observe(cells map[string]map[string]int) []AstraCellObservation {
	out := []AstraCellObservation{}
	for cellID, counts := range cells {
		out = append(out, AstraCellObservation{CellID: cellID, Counts: counts})
	}
	return out
}

func astraMatchByModel(t *testing.T, matches []AstraCheckModelMatch, model string) AstraCheckModelMatch {
	t.Helper()
	for _, match := range matches {
		if match.Model == model {
			return match
		}
	}
	t.Fatalf("model %s not found in matches", model)
	return AstraCheckModelMatch{}
}

func mustParseTimeForTest(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}

// ---------- 归一化与规划 ----------

func TestNormalizeAstraAnswer(t *testing.T) {
	casefold := AstraNormalizer{ID: "exact_trimmed_casefold", MaxLength: 8}
	require.Equal(t, "hello", NormalizeAstraAnswer(casefold, "  Hello \n"))
	require.Equal(t, "五筒", NormalizeAstraAnswer(casefold, "五筒"))
	require.Equal(t, AstraCheckInvalidOutput, NormalizeAstraAnswer(casefold, "   "))
	require.Equal(t, AstraCheckInvalidOutput, NormalizeAstraAnswer(casefold, "123456789"))
	require.Equal(t, "Hello", NormalizeAstraAnswer(AstraNormalizer{ID: "exact_trimmed"}, " Hello "))
}

func TestPlanAstraJobs_RoundRobinFollowsTierCounts(t *testing.T) {
	reg, err := LoadEmbeddedAstraBenchmarks()
	require.NoError(t, err)
	bench := reg.Package("meow-claude-other-cap98-efficient")
	jobs, err := PlanAstraJobs(bench, AstraCheckTierLow)
	require.NoError(t, err)
	require.Len(t, jobs, 48)
	perCell := map[string]int{}
	for _, job := range jobs {
		perCell[job.CellID]++
	}
	require.Equal(t, bench.Tiers[AstraCheckTierLow].Counts, perCell)
	// 前几个任务按题目顺序轮转
	require.Equal(t, bench.Cells[0].ID, jobs[0].CellID)
	require.Equal(t, bench.Cells[1].ID, jobs[1].CellID)

	_, err = PlanAstraJobs(bench, "ultra")
	require.Error(t, err)
}

// ---------- v3 判定 ----------

func TestAstraDirichletLogEvidence(t *testing.T) {
	// α=[1,1]，观测一次第 0 类：lgamma(2) − lgamma(3) + lgamma(2) − lgamma(1) = −ln 2
	require.InDelta(t, -math.Ln2, astraDirichletLogEvidence([]float64{1, 1}, []int{1, 0}), 1e-12)
	require.Equal(t, 0.0, astraDirichletLogEvidence([]float64{1, 1}, []int{0, 0}))
	// 与逐步预测概率之积一致：先 a（1/2），再 a（2/3）
	require.InDelta(t, math.Log(0.5*2.0/3.0), astraDirichletLogEvidence([]float64{1, 1}, []int{2, 0}), 1e-12)
}

func TestScoreAstraCheck_MatchWhenExpectedModelWins(t *testing.T) {
	bench := syntheticGPTPackage(t)
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-astra", observe(map[string]map[string]int{
		"c1": {"a": 2},
		"c2": {"1": 2},
	}))
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictMatch, score.Verdict)
	require.Equal(t, "gpt-6-astra", score.Winner)
	require.Equal(t, "gpt-6-astra", score.Strongest)
	require.Empty(t, score.Reasons)
	require.Equal(t, 4, score.ValidSamples)
	require.Equal(t, 4, score.PlannedSamples)
	astra := astraMatchByModel(t, score.Matches, "gpt-6-astra")
	require.True(t, astra.Passed)
	require.Greater(t, astra.Match, 0.5)
	sol := astraMatchByModel(t, score.Matches, "gpt-6-sol")
	require.False(t, sol.Passed)
	require.Less(t, sol.Match, 0.5)
	require.Equal(t, "GPT-6 Astra", astra.Name)
}

func TestScoreAstraCheck_MismatchPointsToAnotherCandidate(t *testing.T) {
	bench := syntheticGPTPackage(t)
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-astra", observe(map[string]map[string]int{
		"c1": {"b": 2},
		"c2": {"2": 2},
	}))
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictMismatch, score.Verdict)
	require.Equal(t, "gpt-6-sol", score.Winner)
}

func TestScoreAstraCheck_OtherUsesNearestReferenceSource(t *testing.T) {
	bench := syntheticGPTPackage(t)
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-sol", observe(map[string]map[string]int{
		"c1": {"c": 2},
		"c2": {"3": 2},
	}))
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictMismatch, score.Verdict)
	require.Equal(t, AstraCheckOtherModel, score.Winner)
	require.Equal(t, "ref/other", score.NearestReference)
	require.Equal(t, "其他模型", astraMatchByModel(t, score.Matches, AstraCheckOtherModel).Name)
}

func TestScoreAstraCheck_UnknownAnswersFallIntoUnseenAndInvalidDoNotVote(t *testing.T) {
	bench := syntheticGPTPackage(t)
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-astra", observe(map[string]map[string]int{
		"c1": {"zzz": 1, "a": 1, AstraCheckInvalidOutput: 3},
		"c2": {"1": 2},
	}))
	require.NoError(t, err)
	require.Equal(t, 4, score.ValidSamples)
	var c1 AstraCheckCellSummary
	for _, cell := range score.Cells {
		if cell.CellID == "c1" {
			c1 = cell
		}
	}
	require.Equal(t, 2, c1.Valid)
	require.Equal(t, 3, c1.Invalid)
	require.Equal(t, 1, c1.Categories[astraBenchmarkUnseenCategory])
	require.Equal(t, 3, c1.Categories[AstraCheckInvalidOutput])
}

func TestScoreAstraCheck_IncompleteSamplesBlockStrongDirection(t *testing.T) {
	bench := syntheticGPTPackage(t)
	// c2 一个有效答案都没有：每题至少 ceil(0.6×2)=2 条
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-astra", observe(map[string]map[string]int{
		"c1": {"a": 2},
		"c2": {AstraCheckInvalidOutput: 2},
	}))
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Empty(t, score.Winner)
	require.Contains(t, score.Reasons, AstraCheckReasonSamplesIncomplete)
	require.Equal(t, "gpt-6-astra", score.Strongest)
}

func TestScoreAstraCheck_TieIsNoStrongDirection(t *testing.T) {
	bench := syntheticGPTPackage(t)
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-astra", observe(map[string]map[string]int{
		"c1": {"a": 1, "b": 1},
		"c2": {"1": 1, "2": 1},
	}))
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Contains(t, score.Reasons, AstraCheckReasonNoStrongDirection)
}

func TestScoreAstraCheck_NoValidSamples(t *testing.T) {
	bench := syntheticGPTPackage(t)
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-astra", nil)
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Contains(t, score.Reasons, AstraCheckReasonNoValidSamples)
	require.Contains(t, score.Reasons, AstraCheckReasonSamplesIncomplete)
}

func TestScoreAstraCheck_UncalibratedTierNeverPasses(t *testing.T) {
	bench := syntheticGPTPackage(t)
	tier := bench.Tiers[AstraCheckTierLow]
	tier.Calibrated = false
	bench.Tiers[AstraCheckTierLow] = tier
	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, "gpt-6-astra", observe(map[string]map[string]int{
		"c1": {"a": 2},
		"c2": {"1": 2},
	}))
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Contains(t, score.Reasons, AstraCheckReasonUncalibrated)
}

// developmentObservations 取基准包里某个来源的开发集计数作为「这次运行的回答」（样本内检查）。
func developmentObservations(t *testing.T, file, model string) []AstraCellObservation {
	t.Helper()
	raw, err := astrabenchmark.FS.ReadFile(file)
	require.NoError(t, err)
	var pkg struct {
		Observations map[string]map[string]struct {
			Development struct {
				Counts map[string]int `json:"counts"`
			} `json:"development"`
		} `json:"observations"`
	}
	require.NoError(t, json.Unmarshal(raw, &pkg))
	out := []AstraCellObservation{}
	for cellID, byModel := range pkg.Observations {
		counts := map[string]int{}
		for answer, n := range byModel[model].Development.Counts {
			counts[NormalizeAstraAnswer(AstraNormalizer{ID: "exact_trimmed_casefold", MaxLength: 4096}, answer)] += n
		}
		out = append(out, AstraCellObservation{CellID: cellID, Counts: counts})
	}
	return out
}

func TestScoreAstraCheck_RealPackagesIdentifyTheirOwnDevelopmentAnswers(t *testing.T) {
	reg, err := LoadEmbeddedAstraBenchmarks()
	require.NoError(t, err)
	files := map[string]string{}
	for _, file := range astrabenchmark.Files {
		raw, err := astrabenchmark.FS.ReadFile(file)
		require.NoError(t, err)
		var head struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(raw, &head))
		files[head.ID] = file
	}
	for _, target := range astraCheckTargets {
		t.Run(target.ID, func(t *testing.T) {
			bench := reg.Package(target.PackageID)
			obs := developmentObservations(t, files[target.PackageID], target.ID)
			for _, tier := range astraCheckTiers {
				score, err := ScoreAstraCheck(bench, tier, target.ID, obs)
				require.NoError(t, err)
				require.Equal(t, AstraCheckVerdictMatch, score.Verdict, "tier %s reasons %v strongest %s", tier, score.Reasons, score.Strongest)
			}
			// 用另一个候选的回答核对这个目标：绝不能判成一致；证据最高的是那个候选，
			// 越过它自己的强指向线时判不符（没越线就是 insufficient——强指向线按真实档位校准，不保证每个候选都越线）
			for _, other := range bench.ModelIDs {
				if other == target.ID || other == AstraCheckOtherModel {
					continue
				}
				for _, tier := range astraCheckTiers {
					score, err := ScoreAstraCheck(bench, tier, target.ID, developmentObservations(t, files[target.PackageID], other))
					require.NoError(t, err)
					require.NotEqual(t, AstraCheckVerdictMatch, score.Verdict, "answers of %s, tier %s", other, tier)
					require.Equal(t, other, score.Strongest, "answers of %s, tier %s", other, tier)
					if score.Verdict == AstraCheckVerdictMismatch {
						require.Equal(t, other, score.Winner)
					}
				}
			}
		})
	}
}

// ---------- 状态机 ----------

func astraResult(verdict, expected, winner string) *GroupStatusAstraCheckResult {
	return &GroupStatusAstraCheckResult{
		GroupID:          30,
		ConfigID:         101,
		ExpectedModel:    expected,
		Verdict:          verdict,
		Winner:           winner,
		Strongest:        winner,
		Matches:          []AstraCheckModelMatch{{Model: winner, Match: 0.9, Threshold: 0.5, Passed: winner != ""}},
		ValidSamples:     30,
		PlannedSamples:   32,
		BenchmarkVersion: "4.5.4-test",
		CostUSD:          0.01,
		FinishedAt:       time.Now(),
	}
}

func TestComputeAstraCheckTransition_TruthTable(t *testing.T) {
	sol := "gpt-6-sol"
	next, event := ComputeAstraCheckTransition(nil, astraResult(AstraCheckVerdictMatch, sol, sol), 7)
	require.Nil(t, event)
	require.Equal(t, AstraCheckStatusPass, next.StableStatus)
	require.Equal(t, sol, next.ExpectedModel)
	require.Equal(t, int64(7), *next.LastRunID)
	require.Equal(t, 0.01, next.LastCostUSD)
	require.Equal(t, "4.5.4-test", next.BenchmarkVersion)

	next, event = ComputeAstraCheckTransition(next, astraResult(AstraCheckVerdictMismatch, sol, "gpt-6-astra"), 8)
	require.Nil(t, event)
	require.Equal(t, 1, next.ConsecutiveMismatch)

	next, event = ComputeAstraCheckTransition(next, astraResult(AstraCheckVerdictInsufficient, sol, ""), 9)
	require.Nil(t, event)
	require.Equal(t, 1, next.ConsecutiveMismatch)
	require.Equal(t, AstraCheckStatusPass, next.StableStatus)

	next, event = ComputeAstraCheckTransition(next, astraResult(AstraCheckVerdictMismatch, sol, "gpt-6-astra"), 10)
	require.NotNil(t, event)
	require.Equal(t, GroupStatusEventAstraMismatch, event.EventType)
	require.Equal(t, "gpt-6-sol:winner_gpt-6-astra", event.SubStatus)
	require.Contains(t, event.ErrorDetail, "expected GPT-6 Sol")
	require.Equal(t, AstraCheckStatusMismatch, next.StableStatus)

	next, event = ComputeAstraCheckTransition(next, astraResult(AstraCheckVerdictMismatch, sol, "gpt-6-astra"), 11)
	require.Nil(t, event)
	require.Equal(t, 3, next.ConsecutiveMismatch)

	next, event = ComputeAstraCheckTransition(next, astraResult(AstraCheckVerdictMatch, sol, sol), 12)
	require.NotNil(t, event)
	require.Equal(t, GroupStatusEventAstraRecovered, event.EventType)
	require.Equal(t, 0, next.ConsecutiveMismatch)
}

func TestAstraCheckEventModels(t *testing.T) {
	event := &GroupStatusEvent{SubStatus: astraCheckEventSubStatus("claude-opus-5.5", AstraCheckOtherModel)}
	expected, winner := astraCheckEventModels(event)
	require.Equal(t, "claude-opus-5.5", expected)
	require.Equal(t, AstraCheckOtherModel, winner)
	require.Equal(t, "Claude Opus 5.5", astraExpectedFromEvent(event))
	require.Equal(t, "其他模型", astraWinnerFromEvent(event))

	unknown := &GroupStatusEvent{SubStatus: astraCheckEventSubStatus("gpt-6-sol", "")}
	require.Equal(t, "?", astraWinnerFromEvent(unknown))
	require.Equal(t, "?", astraWinnerFromEvent(&GroupStatusEvent{SubStatus: "winner_gpt-6-astra"}))
	require.Equal(t, "?", astraExpectedFromEvent(nil))
}

// ---------- 配置 ----------

func astraBoolPtr(v bool) *bool { return &v }

func TestNormalizeGroupStatusConfig_AstraCheckModels(t *testing.T) {
	base := func() *GroupStatusConfigUpsertInput {
		return &GroupStatusConfigUpsertInput{
			Enabled:        true,
			ProbeModel:     "gpt-6-sol",
			ProbePrompt:    "ping",
			ValidationMode: GroupStatusValidationNonEmpty,
		}
	}
	openAI := &Group{ID: 7, Platform: PlatformOpenAI}
	anthropic := &Group{ID: 8, Platform: PlatformAnthropic}
	gemini := &Group{ID: 9, Platform: PlatformGemini}

	cfg, err := NormalizeGroupStatusConfig(openAI, base())
	require.NoError(t, err)
	require.False(t, cfg.AstraCheckEnabled)
	require.Equal(t, []AstraCheckModelConfig{{ExpectedModel: "gpt-5.6-sol"}}, cfg.AstraCheckModels)
	require.Equal(t, AstraCheckTierLow, cfg.AstraCheckTier)
	require.Equal(t, 3600, cfg.AstraCheckIntervalSeconds)

	cfg, err = NormalizeGroupStatusConfig(anthropic, base())
	require.NoError(t, err)
	require.Equal(t, []AstraCheckModelConfig{{ExpectedModel: "claude-opus-5.5"}}, cfg.AstraCheckModels)

	on := base()
	on.AstraCheckEnabled = astraBoolPtr(true)
	on.AstraCheckModels = []AstraCheckModelConfig{
		{ExpectedModel: " gpt-6-sol "},
		{ExpectedModel: "gpt-6-astra", RequestModel: " astra-alias "},
		{ExpectedModel: "gpt-6-sol"},
		{ExpectedModel: ""},
	}
	on.AstraCheckTier = "Medium"
	on.AstraCheckIntervalSeconds = 1800
	cfg, err = NormalizeGroupStatusConfig(openAI, on)
	require.NoError(t, err)
	require.True(t, cfg.AstraCheckEnabled)
	require.Equal(t, []AstraCheckModelConfig{{ExpectedModel: "gpt-6-sol"}, {ExpectedModel: "gpt-6-astra", RequestModel: "astra-alias"}}, cfg.AstraCheckModels)
	require.Equal(t, AstraCheckTierMedium, cfg.AstraCheckTier)
	require.Equal(t, 1800, cfg.AstraCheckIntervalSeconds)

	claudeOn := base()
	claudeOn.AstraCheckEnabled = astraBoolPtr(true)
	claudeOn.AstraCheckModels = []AstraCheckModelConfig{{ExpectedModel: "claude-opus-5.5"}, {ExpectedModel: "claude-fable-5.1"}}
	cfg, err = NormalizeGroupStatusConfig(anthropic, claudeOn)
	require.NoError(t, err)
	require.Len(t, cfg.AstraCheckModels, 2)

	// Claude 目标不能配给 GPT 分组
	_, err = NormalizeGroupStatusConfig(openAI, claudeOn)
	require.ErrorIs(t, err, ErrGroupStatusAstraCheckTargetInvalid)

	// 开启时模型列表不能为空
	empty := base()
	empty.AstraCheckEnabled = astraBoolPtr(true)
	empty.AstraCheckModels = []AstraCheckModelConfig{}
	_, err = NormalizeGroupStatusConfig(openAI, empty)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)

	geminiOn := base()
	geminiOn.AstraCheckEnabled = astraBoolPtr(true)
	_, err = NormalizeGroupStatusConfig(gemini, geminiOn)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)

	badTier := base()
	badTier.AstraCheckTier = "ultra"
	_, err = NormalizeGroupStatusConfig(openAI, badTier)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)

	tooFast := base()
	tooFast.AstraCheckIntervalSeconds = 600
	_, err = NormalizeGroupStatusConfig(openAI, tooFast)
	require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
}

func TestAstraCheckModelsToRun(t *testing.T) {
	group := &Group{ID: 1, Platform: PlatformOpenAI}
	cfg := &GroupStatusConfig{AstraCheckModels: []AstraCheckModelConfig{{ExpectedModel: "gpt-6-sol"}, {ExpectedModel: "gpt-6-astra"}}}
	models, err := astraCheckModelsToRun(group, cfg, "")
	require.NoError(t, err)
	require.Len(t, models, 2)

	models, err = astraCheckModelsToRun(group, cfg, "gpt-6-astra")
	require.NoError(t, err)
	require.Equal(t, []AstraCheckModelConfig{{ExpectedModel: "gpt-6-astra"}}, models)

	_, err = astraCheckModelsToRun(group, cfg, "gpt-5.6-sol")
	require.ErrorIs(t, err, ErrGroupStatusAstraCheckTargetInvalid)

	models, err = astraCheckModelsToRun(group, &GroupStatusConfig{}, "")
	require.NoError(t, err)
	require.Equal(t, []AstraCheckModelConfig{{ExpectedModel: "gpt-5.6-sol"}}, models)

	target, _ := astraCheckTarget("claude-opus-5.5")
	require.Equal(t, "claude-opus-5-5", AstraCheckModelConfig{ExpectedModel: "claude-opus-5.5"}.requestModelFor(target))
	require.Equal(t, "opus-alias", AstraCheckModelConfig{ExpectedModel: "claude-opus-5.5", RequestModel: "opus-alias"}.requestModelFor(target))
}
