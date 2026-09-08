package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// astraSyntheticPackage 是测试用的最小基准包：2 个模型 × 2 道题，分布与阈值手工可算。
//
// 低档每题 2 次：
//   - 全 Astra 式答案（japan / 3）  → astra 得分 -0.276、sol -2.905 → astra 匹配度 ≈ 0.933 > 0.9
//   - 全 Sol 式答案（brazil / 4）   → sol 匹配度 ≈ 0.955 > 0.9
//   - 混合（japan / 4）            → astra ≈ 0.752，无人越线
const astraSyntheticPackage = `{
  "mode": "gpt",
  "id": "synthetic-astra",
  "version": "test-1",
  "content_sha256": "deadbeef",
  "engine": {"scoring_version": "meow-fingerprint-v2"},
  "models": [
    {"id": "gpt-6-astra", "name": "GPT-6 Astra", "request_model": "gpt-6-astra"},
    {"id": "gpt-5.6-sol", "name": "GPT-5.6 Sol", "request_model": "gpt-5.6-sol"}
  ],
  "probes": [
    {
      "id": "country",
      "family_id": "country_en",
      "normalizer": {"id": "exact_trimmed_casefold", "parameters": {"max_length": 64}},
      "cells": [
        {"id": "country_low", "system": ".", "prompt": "Name a random country. Reply with only the name.", "effort": "low", "parameters": {"max_output_tokens": 128}}
      ]
    },
    {
      "id": "count_r",
      "family_id": "count_r",
      "normalizer": {"id": "b80_exact_3", "parameters": {}},
      "cells": [
        {"id": "strawberry_low", "system": ".", "prompt": "How many r are in strawberry? Reply with a number only.", "effort": "low", "parameters": {"max_output_tokens": 128}}
      ]
    }
  ],
  "tiers": {
    "low": {"counts": {"country_low": 2, "strawberry_low": 2}, "thresholds": {"gpt-6-astra": 0.9, "gpt-5.6-sol": 0.9}},
    "medium": {"counts": 4, "thresholds": {"gpt-6-astra": 0.9, "gpt-5.6-sol": 0.9}},
    "high": {"counts": 6, "thresholds": {"gpt-6-astra": 0.9, "gpt-5.6-sol": 0.9}}
  },
  "fitted": {
    "models": ["gpt-6-astra", "gpt-5.6-sol"],
    "cells": {
      "country_low": {
        "categories": ["japan", "brazil", "__OTHER__"],
        "model_distributions": {
          "gpt-6-astra": {"japan": 0.8, "brazil": 0.1, "__OTHER__": 0.1},
          "gpt-5.6-sol": {"japan": 0.1, "brazil": 0.8, "__OTHER__": 0.1}
        },
        "weight": 1.0,
        "family_id": "country_en",
        "reference_ready": true
      },
      "strawberry_low": {
        "categories": ["exact_3", "other_integer"],
        "model_distributions": {
          "gpt-6-astra": {"exact_3": 0.9, "other_integer": 0.1},
          "gpt-5.6-sol": {"exact_3": 0.3, "other_integer": 0.7}
        },
        "weight": 0.5,
        "family_id": "count_r",
        "reference_ready": true
      }
    }
  },
  "calibration": {"tiers": {"low": {"status": "target_met"}, "medium": {"status": "target_met"}, "high": {"status": "target_met"}}}
}`

func loadSyntheticAstraBenchmark(t *testing.T) (*AstraBenchmark, *AstraBenchmarkMeta) {
	t.Helper()
	bench, meta, err := ParseAstraBenchmark([]byte(astraSyntheticPackage))
	require.NoError(t, err)
	require.NotNil(t, bench)
	require.NotNil(t, meta)
	return bench, meta
}

func astraMatchByModel(t *testing.T, matches []AstraCheckModelMatch, model string) AstraCheckModelMatch {
	t.Helper()
	for _, match := range matches {
		if match.Model == model {
			return match
		}
	}
	t.Fatalf("model %s not found in matches %+v", model, matches)
	return AstraCheckModelMatch{}
}

// ---------- 归一器 ----------

func TestNormalizeAstraAnswer(t *testing.T) {
	cases := []struct {
		name string
		norm AstraNormalizer
		raw  string
		want string
	}{
		{"casefold trims and lowers", AstraNormalizer{ID: "exact_trimmed_casefold"}, "  Japan \n", "japan"},
		{"empty is invalid", AstraNormalizer{ID: "exact_trimmed_casefold"}, "   ", AstraCheckInvalidOutput},
		{"default normalizer casefolds", AstraNormalizer{}, "BRAZIL", "brazil"},
		{"exact_trimmed keeps case", AstraNormalizer{ID: "exact_trimmed"}, " Tokyo ", "Tokyo"},
		{"whitespace collapse", AstraNormalizer{ID: "whitespace_collapse"}, "a   b\n c", "a b c"},
		{"integer canonical", AstraNormalizer{ID: "integer"}, " +007 ", "7"},
		{"integer rejects text", AstraNormalizer{ID: "integer"}, "three", AstraCheckInvalidOutput},
		{"b80 exact 3", AstraNormalizer{ID: "b80_exact_3"}, "3", "exact_3"},
		{"b80 other integer", AstraNormalizer{ID: "b80_exact_3"}, "2", "other_integer"},
		{"b80 non integer invalid", AstraNormalizer{ID: "b80_exact_3"}, "3 r's", AstraCheckInvalidOutput},
		{"behavior label strips punctuation", AstraNormalizer{ID: "behavior_label"}, "\"Blue Jay.\"", "blue jay"},
		{"behavior label rejects digits", AstraNormalizer{ID: "behavior_label"}, "jay 42", AstraCheckInvalidOutput},
		{"fixed enum maps value", AstraNormalizer{ID: "fixed_enum", Values: map[string]string{"Yes": "yes", "No": "no"}}, "YES", "yes"},
		{"fixed enum unknown is other", AstraNormalizer{ID: "fixed_enum", Values: map[string]string{"Yes": "yes"}}, "maybe", AstraCheckOtherCategory},
		{"too long is invalid", AstraNormalizer{ID: "exact_trimmed_casefold", MaxLength: 4}, "japan", AstraCheckInvalidOutput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, NormalizeAstraAnswer(tc.norm, tc.raw))
		})
	}
}

// ---------- 任务规划 ----------

func TestPlanAstraJobs_RoundRobin(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)

	jobs, err := PlanAstraJobs(bench, AstraCheckTierLow)
	require.NoError(t, err)
	require.Equal(t, []AstraJob{
		{CellID: "country_low", Index: 0},
		{CellID: "strawberry_low", Index: 0},
		{CellID: "country_low", Index: 1},
		{CellID: "strawberry_low", Index: 1},
	}, jobs)

	medium, err := PlanAstraJobs(bench, AstraCheckTierMedium)
	require.NoError(t, err)
	require.Len(t, medium, 8)

	_, err = PlanAstraJobs(bench, "ultra")
	require.Error(t, err)
}

// ---------- 判定 ----------

func TestScoreAstraCheck_MatchOnAstraLikeAnswers(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"japan": 2}},
		{CellID: "strawberry_low", Counts: map[string]int{"exact_3": 2}},
	})
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictMatch, score.Verdict)
	require.Equal(t, "gpt-6-astra", score.Winner)
	require.Empty(t, score.Reasons)
	require.Equal(t, 4, score.ValidSamples)
	require.Equal(t, 4, score.PlannedSamples)
	require.Len(t, score.Matches, 2)

	astra := astraMatchByModel(t, score.Matches, "gpt-6-astra")
	require.True(t, astra.Passed)
	require.InDelta(t, 0.933, astra.Match, 0.005)
	require.InDelta(t, 0.9, astra.Threshold, 1e-9)
	require.Equal(t, "GPT-6 Astra", astra.Name)

	sol := astraMatchByModel(t, score.Matches, "gpt-5.6-sol")
	require.False(t, sol.Passed)
	require.InDelta(t, 0.067, sol.Match, 0.005)

	require.Len(t, score.Cells, 2)
	require.Equal(t, "country_en", score.Cells[0].FamilyID)
	require.Equal(t, 2, score.Cells[0].Minimum)
	require.Equal(t, 1.0, score.Cells[0].Weight)
}

func TestScoreAstraCheck_MismatchPointsToSol(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"brazil": 2}},
		{CellID: "strawberry_low", Counts: map[string]int{"other_integer": 2}},
	})
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictMismatch, score.Verdict)
	require.Equal(t, "gpt-5.6-sol", score.Winner)
	require.Empty(t, score.Reasons)
	require.InDelta(t, 0.955, astraMatchByModel(t, score.Matches, "gpt-5.6-sol").Match, 0.005)
}

// 样本齐全但无人越线：Astra 未达自身阈值，按「软 mismatch」处理，winner 记最接近的非 Astra 模型。
func TestScoreAstraCheck_AstraBelowThresholdIsSoftMismatch(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"japan": 2}},
		{CellID: "strawberry_low", Counts: map[string]int{"other_integer": 2}},
	})
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictMismatch, score.Verdict)
	require.Equal(t, "gpt-5.6-sol", score.Winner)
	require.Equal(t, []string{AstraCheckReasonBelowThreshold}, score.Reasons)
	require.InDelta(t, 0.752, astraMatchByModel(t, score.Matches, "gpt-6-astra").Match, 0.005)
	for _, match := range score.Matches {
		require.False(t, match.Passed)
	}
	require.True(t, astraCheckIsSoftMismatch(score.Reasons))
	require.False(t, astraCheckIsSoftMismatch([]string{"samples_incomplete"}))
}

func TestScoreAstraCheck_IncompleteSamplesBlocksVerdict(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"japan": 1, AstraCheckInvalidOutput: 1}},
		{CellID: "strawberry_low", Counts: map[string]int{"exact_3": 2}},
	})
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Equal(t, []string{"samples_incomplete"}, score.Reasons)
	require.Equal(t, 3, score.ValidSamples)
	require.Equal(t, 1, score.Cells[0].Invalid)
	// 匹配度仍然填满，供 UI 画条
	require.Len(t, score.Matches, 2)
	require.Greater(t, astraMatchByModel(t, score.Matches, "gpt-6-astra").Match, 0.9)
	require.False(t, astraMatchByModel(t, score.Matches, "gpt-6-astra").Passed)
}

func TestScoreAstraCheck_ExceedPlanAndMissingCell(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"japan": 3}},
	})
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Equal(t, []string{"samples_exceed_plan", "samples_incomplete"}, score.Reasons)
	require.Equal(t, 0, score.Cells[1].Valid)
}

func TestScoreAstraCheck_UnknownCategoryFoldsIntoOther(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"france": 2}},
		{CellID: "strawberry_low", Counts: map[string]int{"exact_3": 2}},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]int{AstraCheckOtherCategory: 2}, score.Cells[0].Categories)
	// __OTHER__ 两模型同概率，只剩 strawberry 题区分：astra ln0.9 vs sol ln0.3，×0.5 权重 → 差 0.549，
	// astra 匹配度 ≈ 0.634 达不到 0.9 → 软 mismatch
	require.Equal(t, AstraCheckVerdictMismatch, score.Verdict)
	require.Equal(t, "gpt-5.6-sol", score.Winner)
	require.Equal(t, []string{AstraCheckReasonBelowThreshold}, score.Reasons)
	require.InDelta(t, 0.634, astraMatchByModel(t, score.Matches, "gpt-6-astra").Match, 0.005)
}

func TestScoreAstraCheck_UncalibratedAndMissingBaseline(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)
	tier := bench.Tiers[AstraCheckTierLow]
	tier.Calibrated = false
	bench.Tiers[AstraCheckTierLow] = tier
	fitted := bench.Fitted["strawberry_low"]
	fitted.ReferenceReady = false
	bench.Fitted["strawberry_low"] = fitted

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"japan": 2}},
		{CellID: "strawberry_low", Counts: map[string]int{"exact_3": 2}},
	})
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Equal(t, []string{"baseline_cell_missing", "uncalibrated"}, score.Reasons)
	require.Len(t, score.Cells, 1)
}

func TestScoreAstraCheck_NoWeightedFamily(t *testing.T) {
	bench, _ := loadSyntheticAstraBenchmark(t)
	for id, fitted := range bench.Fitted {
		fitted.Weight = 0
		bench.Fitted[id] = fitted
	}

	score, err := ScoreAstraCheck(bench, AstraCheckTierLow, []AstraCellObservation{
		{CellID: "country_low", Counts: map[string]int{"japan": 2}},
		{CellID: "strawberry_low", Counts: map[string]int{"exact_3": 2}},
	})
	require.NoError(t, err)
	require.Equal(t, AstraCheckVerdictInsufficient, score.Verdict)
	require.Equal(t, []string{"no_weighted_family"}, score.Reasons)
	for _, match := range score.Matches {
		require.InDelta(t, 0.5, match.Match, 1e-9)
	}
}

// ---------- 状态机 ----------

func astraResultWithVerdict(verdict, winner string) *GroupStatusAstraCheckResult {
	latency := int64(1500)
	return &GroupStatusAstraCheckResult{
		GroupID:          30,
		ConfigID:         101,
		BenchmarkVersion: "test-1",
		Verdict:          verdict,
		Winner:           winner,
		Matches: []AstraCheckModelMatch{
			{Model: "gpt-6-astra", Match: 0.1, Threshold: 0.9},
			{Model: "gpt-5.6-sol", Match: 0.9, Threshold: 0.9},
		},
		ValidSamples:   4,
		PlannedSamples: 4,
		LatencyMS:      &latency,
		StartedAt:      time.Now().Add(-2 * time.Second),
		FinishedAt:     time.Now(),
	}
}

func TestComputeAstraCheckTransition_TruthTable(t *testing.T) {
	sol := "gpt-5.6-sol"
	cases := []struct {
		name          string
		prev          *GroupStatusState
		verdict       string
		winner        string
		wantStable    string
		wantConsec    int
		wantEventType string
		wantSubStatus string
	}{
		{"fresh match → pass, no event", nil, AstraCheckVerdictMatch, "gpt-6-astra", AstraCheckStatusPass, 0, "", ""},
		{"fresh mismatch → count 1, no event", nil, AstraCheckVerdictMismatch, sol, "", 1, "", ""},
		{"second mismatch → stable mismatch + event", &GroupStatusState{AstraCheckConsecutiveMismatch: 1}, AstraCheckVerdictMismatch, sol, AstraCheckStatusMismatch, 2, GroupStatusEventAstraMismatch, "winner_gpt-5.6-sol"},
		{"already mismatch → count grows, no event", &GroupStatusState{AstraCheckStableStatus: AstraCheckStatusMismatch, AstraCheckConsecutiveMismatch: 2}, AstraCheckVerdictMismatch, sol, AstraCheckStatusMismatch, 3, "", ""},
		{"mismatch → match recovers with event", &GroupStatusState{AstraCheckStableStatus: AstraCheckStatusMismatch, AstraCheckConsecutiveMismatch: 2}, AstraCheckVerdictMatch, "gpt-6-astra", AstraCheckStatusPass, 0, GroupStatusEventAstraRecovered, "winner_gpt-6-astra"},
		{"pass → insufficient keeps pass", &GroupStatusState{AstraCheckStableStatus: AstraCheckStatusPass}, AstraCheckVerdictInsufficient, "", AstraCheckStatusPass, 0, "", ""},
		{"count 1 → insufficient keeps count", &GroupStatusState{AstraCheckConsecutiveMismatch: 1}, AstraCheckVerdictInsufficient, "", "", 1, "", ""},
		{"pass → single mismatch keeps pass", &GroupStatusState{AstraCheckStableStatus: AstraCheckStatusPass}, AstraCheckVerdictMismatch, sol, AstraCheckStatusPass, 1, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := astraResultWithVerdict(tc.verdict, tc.winner)
			next, event := ComputeAstraCheckTransition(tc.prev, result, 77)
			require.Equal(t, tc.wantStable, next.AstraCheckStableStatus)
			require.Equal(t, tc.wantConsec, next.AstraCheckConsecutiveMismatch)
			require.Equal(t, tc.verdict, next.AstraCheckVerdict)
			require.Equal(t, tc.winner, next.AstraCheckWinner)
			require.NotNil(t, next.AstraCheckCheckedAt)
			require.NotNil(t, next.AstraCheckLastRunID)
			require.Equal(t, int64(77), *next.AstraCheckLastRunID)
			require.Len(t, next.AstraCheckMatches, 2)
			require.Contains(t, next.AstraCheckDetail, "valid 4/4")
			require.Contains(t, next.AstraCheckDetail, "benchmark test-1")
			if tc.wantEventType == "" {
				require.Nil(t, event)
				return
			}
			require.NotNil(t, event)
			require.Equal(t, tc.wantEventType, event.EventType)
			require.Equal(t, tc.wantSubStatus, event.SubStatus)
			require.Equal(t, int64(30), event.GroupID)
			require.Equal(t, int64(101), event.ConfigID)
			require.Equal(t, next.AstraCheckStableStatus, event.ToStatus)
		})
	}
}

func TestComputeAstraCheckTransition_DoesNotTouchOtherProbes(t *testing.T) {
	checkedAt := time.Now().Add(-time.Hour)
	prev := &GroupStatusState{
		GroupID:                     30,
		ConfigID:                    101,
		LatestStatus:                GroupRuntimeStatusDown,
		StableStatus:                GroupRuntimeStatusDown,
		ConsecutiveDown:             3,
		SolJuiceStatus:              SolJuiceStatusMismatch,
		SolJuiceStableStatus:        SolJuiceStatusMismatch,
		SolJuiceValue:               "32",
		SolJuiceCheckedAt:           &checkedAt,
		SolJuiceConsecutiveMismatch: 2,
	}

	next, _ := ComputeAstraCheckTransition(prev, astraResultWithVerdict(AstraCheckVerdictMatch, "gpt-6-astra"), 1)
	require.Equal(t, GroupRuntimeStatusDown, next.LatestStatus)
	require.Equal(t, GroupRuntimeStatusDown, next.StableStatus)
	require.Equal(t, 3, next.ConsecutiveDown)
	require.Equal(t, SolJuiceStatusMismatch, next.SolJuiceStableStatus)
	require.Equal(t, "32", next.SolJuiceValue)
	require.Equal(t, 2, next.SolJuiceConsecutiveMismatch)
	require.Same(t, &checkedAt, next.SolJuiceCheckedAt)
	require.Equal(t, AstraCheckStatusPass, next.AstraCheckStableStatus)
	// 原状态不被原地修改
	require.Equal(t, "", prev.AstraCheckStableStatus)
}

// ---------- 成本与配置 ----------

func TestEstimateAstraCheckCostUSD(t *testing.T) {
	require.Equal(t, 0.0, EstimateAstraCheckCostUSD(0, 0))
	require.InDelta(t, 0.0002, EstimateAstraCheckCostUSD(20, 0), 1e-12)
	require.InDelta(t, 0.005, EstimateAstraCheckCostUSD(0, 100), 1e-12)
	// 低档 20 请求，约 1200 输入 + 400 输出（含推理）→ 0.012 + 0.02
	require.InDelta(t, 0.032, EstimateAstraCheckCostUSD(1200, 400), 1e-12)
}

func astraBoolPtr(v bool) *bool { return &v }

func TestNormalizeGroupStatusConfig_AstraCheckRules(t *testing.T) {
	openAI := &Group{ID: 30, Platform: PlatformOpenAI}
	base := func() *GroupStatusConfigUpsertInput {
		return &GroupStatusConfigUpsertInput{
			Enabled:        true,
			ProbeModel:     "gpt-6-astra",
			ProbePrompt:    "Please reply ONLINE.",
			ValidationMode: GroupStatusValidationNonEmpty,
		}
	}

	t.Run("defaults", func(t *testing.T) {
		cfg, err := NormalizeGroupStatusConfig(openAI, base())
		require.NoError(t, err)
		require.False(t, cfg.AstraCheckEnabled)
		require.Equal(t, "gpt-6-astra", cfg.AstraCheckRequestModel)
		require.Equal(t, AstraCheckTierLow, cfg.AstraCheckTier)
		require.Equal(t, 3600, cfg.AstraCheckIntervalSeconds)
	})

	t.Run("enabled with custom values", func(t *testing.T) {
		input := base()
		input.AstraCheckEnabled = astraBoolPtr(true)
		input.AstraCheckRequestModel = " gpt-6-astra-alias "
		input.AstraCheckTier = "Medium"
		input.AstraCheckIntervalSeconds = 1800
		cfg, err := NormalizeGroupStatusConfig(openAI, input)
		require.NoError(t, err)
		require.True(t, cfg.AstraCheckEnabled)
		require.Equal(t, "gpt-6-astra-alias", cfg.AstraCheckRequestModel)
		require.Equal(t, AstraCheckTierMedium, cfg.AstraCheckTier)
		require.Equal(t, 1800, cfg.AstraCheckIntervalSeconds)
		// Sol Juice 默认不受影响
		require.False(t, cfg.SolJuiceEnabled)
		require.Equal(t, "gpt-5.6-sol", cfg.SolJuiceModel)
	})

	t.Run("rejected on non-openai group", func(t *testing.T) {
		input := base()
		input.AstraCheckEnabled = astraBoolPtr(true)
		_, err := NormalizeGroupStatusConfig(&Group{ID: 31, Platform: PlatformAnthropic}, input)
		require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
		require.ErrorContains(t, err, "astra_check")
	})

	t.Run("rejects unknown tier", func(t *testing.T) {
		input := base()
		input.AstraCheckTier = "ultra"
		_, err := NormalizeGroupStatusConfig(openAI, input)
		require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
		require.ErrorContains(t, err, "astra_check_tier")
	})

	t.Run("rejects interval below minimum", func(t *testing.T) {
		input := base()
		input.AstraCheckIntervalSeconds = 600
		_, err := NormalizeGroupStatusConfig(openAI, input)
		require.ErrorIs(t, err, ErrGroupStatusInvalidConfig)
		require.ErrorContains(t, err, "astra_check_interval_seconds")
	})
}

func TestAstraWinnerFromEvent(t *testing.T) {
	require.Equal(t, "?", astraWinnerFromEvent(nil))
	require.Equal(t, "?", astraWinnerFromEvent(&GroupStatusEvent{SubStatus: "winner_unknown"}))
	require.Equal(t, "?", astraWinnerFromEvent(&GroupStatusEvent{SubStatus: "juice_32"}))
	require.Equal(t, "Sol", astraWinnerFromEvent(&GroupStatusEvent{SubStatus: "winner_gpt-5.6-sol"}))
	require.Equal(t, "Luna", astraWinnerFromEvent(&GroupStatusEvent{SubStatus: "closest_gpt-5.6-luna"}))
	require.Equal(t, "Astra", astraWinnerFromEvent(&GroupStatusEvent{SubStatus: "winner_gpt-6-astra"}))
	require.Equal(t, "gpt-7-x", astraWinnerFromEvent(&GroupStatusEvent{SubStatus: "winner_gpt-7-x"}))

	require.Equal(t, "强指向 Sol", astraEventPointerText(&GroupStatusEvent{SubStatus: "winner_gpt-5.6-sol"}))
	require.Equal(t, "最接近 Luna，Astra 未达自身阈值", astraEventPointerText(&GroupStatusEvent{SubStatus: "closest_gpt-5.6-luna"}))
}

// 软 mismatch 与强指向走同一状态机：两次连续才变红，事件 sub_status 用 closest_ 前缀区分。
func TestComputeAstraCheckTransition_SoftMismatchUsesClosestPrefix(t *testing.T) {
	result := astraResultWithVerdict(AstraCheckVerdictMismatch, "gpt-5.6-luna")
	result.Reasons = []string{AstraCheckReasonBelowThreshold}

	next, event := ComputeAstraCheckTransition(&GroupStatusState{AstraCheckStableStatus: AstraCheckStatusPass}, result, 1)
	require.Nil(t, event)
	require.Equal(t, AstraCheckStatusPass, next.AstraCheckStableStatus)
	require.Equal(t, 1, next.AstraCheckConsecutiveMismatch)
	require.Equal(t, AstraCheckVerdictMismatch, next.AstraCheckVerdict)
	require.Equal(t, "gpt-5.6-luna", next.AstraCheckWinner)
	require.Equal(t, []string{AstraCheckReasonBelowThreshold}, next.AstraCheckReasons)

	next, event = ComputeAstraCheckTransition(next, result, 2)
	require.NotNil(t, event)
	require.Equal(t, GroupStatusEventAstraMismatch, event.EventType)
	require.Equal(t, "closest_gpt-5.6-luna", event.SubStatus)
	require.Equal(t, AstraCheckStatusMismatch, next.AstraCheckStableStatus)
	require.Contains(t, event.ErrorDetail, AstraCheckReasonBelowThreshold)
}
