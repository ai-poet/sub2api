package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	astrabenchmark "github.com/Wei-Shaw/sub2api/resources/astra-benchmark"
)

// mutateAstraPackage 在合成包上做一处改动后重新编码，用于校验失败用例。
func mutateAstraPackage(t *testing.T, mutate func(pkg map[string]any)) []byte {
	t.Helper()
	var pkg map[string]any
	require.NoError(t, json.Unmarshal([]byte(astraSyntheticPackage), &pkg))
	mutate(pkg)
	raw, err := json.Marshal(pkg)
	require.NoError(t, err)
	return raw
}

func TestParseAstraBenchmark_SyntheticPackage(t *testing.T) {
	bench, meta := loadSyntheticAstraBenchmark(t)

	require.Equal(t, "synthetic-astra", bench.PackageID)
	require.Equal(t, "test-1", bench.Version)
	require.Equal(t, "deadbeef", bench.ContentSHA256)
	require.Len(t, bench.BodySHA256, 64)
	require.Equal(t, []string{"gpt-6-astra", "gpt-5.6-sol"}, bench.ModelIDs)
	require.Len(t, bench.Cells, 2)
	require.Equal(t, "country_low", bench.Cells[0].ID)
	require.Equal(t, "country_en", bench.Cells[0].FamilyID)
	require.Equal(t, "exact_trimmed_casefold", bench.Cells[0].Normalizer.ID)
	require.Equal(t, 64, bench.Cells[0].Normalizer.MaxLength)
	require.Equal(t, "low", bench.Cells[0].Effort)
	require.Equal(t, 128, bench.Cells[0].MaxOutputTokens)
	require.Equal(t, "b80_exact_3", bench.Cells[1].Normalizer.ID)
	require.Same(t, &bench.Cells[1], bench.CellIndex["strawberry_low"])

	require.Equal(t, 4, bench.Tiers[AstraCheckTierLow].TotalRequests)
	require.Equal(t, 8, bench.Tiers[AstraCheckTierMedium].TotalRequests)
	require.Equal(t, 12, bench.Tiers[AstraCheckTierHigh].TotalRequests)
	require.True(t, bench.Tiers[AstraCheckTierLow].Calibrated)
	require.InDelta(t, 0.9, bench.Tiers[AstraCheckTierLow].Thresholds["gpt-5.6-sol"], 1e-9)
	require.Equal(t, 4, bench.TierRequests(AstraCheckTierLow))

	require.Equal(t, "GPT-5.6 Sol", bench.modelName("gpt-5.6-sol"))
	require.Equal(t, "GPT-6 Astra", meta.Models[0].Name)
	require.Equal(t, []AstraBenchmarkTierMeta{
		{Tier: "low", Requests: 4, Calibrated: true},
		{Tier: "medium", Requests: 8, Calibrated: true},
		{Tier: "high", Requests: 12, Calibrated: true},
	}, meta.Tiers)
}

func TestParseAstraBenchmark_ValidationFailures(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(pkg map[string]any)
		wantErr string
	}{
		{"wrong mode", func(pkg map[string]any) { pkg["mode"] = "claude" }, "mode must be"},
		{"wrong scoring version", func(pkg map[string]any) {
			pkg["engine"] = map[string]any{"scoring_version": "meow-fingerprint-v1"}
		}, "scoring_version"},
		{"missing astra model", func(pkg map[string]any) {
			pkg["models"] = []any{map[string]any{"id": "gpt-5.6-sol", "name": "Sol"}}
		}, "does not contain gpt-6-astra"},
		{"cell without fitted distribution", func(pkg map[string]any) {
			cells := pkg["fitted"].(map[string]any)["cells"].(map[string]any)
			delete(cells, "strawberry_low")
		}, "has no fitted distribution"},
		{"fitted cell lacks a model", func(pkg map[string]any) {
			cell := pkg["fitted"].(map[string]any)["cells"].(map[string]any)["country_low"].(map[string]any)
			delete(cell["model_distributions"].(map[string]any), "gpt-5.6-sol")
		}, "lacks distribution for gpt-5.6-sol"},
		{"all weights zero", func(pkg map[string]any) {
			cells := pkg["fitted"].(map[string]any)["cells"].(map[string]any)
			for _, raw := range cells {
				raw.(map[string]any)["weight"] = 0
			}
		}, "positive weight"},
		{"tier missing", func(pkg map[string]any) { delete(pkg["tiers"].(map[string]any), "high") }, "tier high is missing"},
		{"tier plans zero", func(pkg map[string]any) {
			pkg["tiers"].(map[string]any)["low"].(map[string]any)["counts"] = 0
		}, "plans zero requests"},
		{"no probes", func(pkg map[string]any) { pkg["probes"] = []any{} }, "no cells"},
		{"empty package", nil, "empty package"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw []byte
			if tc.mutate != nil {
				raw = mutateAstraPackage(t, tc.mutate)
			}
			_, _, err := ParseAstraBenchmark(raw)
			require.Error(t, err)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestParseAstraBenchmark_MissingThresholdMarksTierUncalibrated(t *testing.T) {
	raw := mutateAstraPackage(t, func(pkg map[string]any) {
		low := pkg["tiers"].(map[string]any)["low"].(map[string]any)
		low["thresholds"] = map[string]any{"gpt-6-astra": 0.9}
	})
	bench, meta, err := ParseAstraBenchmark(raw)
	require.NoError(t, err)
	require.False(t, bench.Tiers[AstraCheckTierLow].Calibrated)
	require.True(t, bench.Tiers[AstraCheckTierMedium].Calibrated)
	require.False(t, meta.Tiers[0].Calibrated)
}

func TestParseAstraBenchmark_CalibrationStatusNotMet(t *testing.T) {
	raw := mutateAstraPackage(t, func(pkg map[string]any) {
		pkg["calibration"] = map[string]any{"tiers": map[string]any{"low": map[string]any{"status": "pending"}}}
	})
	bench, _, err := ParseAstraBenchmark(raw)
	require.NoError(t, err)
	require.False(t, bench.Tiers[AstraCheckTierLow].Calibrated)
	require.False(t, bench.Tiers[AstraCheckTierMedium].Calibrated)
}

func TestParseAstraBenchmark_FallbackIDAndVersion(t *testing.T) {
	raw := mutateAstraPackage(t, func(pkg map[string]any) {
		delete(pkg, "id")
		delete(pkg, "version")
	})
	bench, _, err := ParseAstraBenchmark(raw)
	require.NoError(t, err)
	require.Equal(t, "astra-benchmark-"+bench.BodySHA256[:12], bench.PackageID)
	require.Equal(t, bench.BodySHA256[:12], bench.Version)
}

// 内置包冒烟：能解析、四个模型、低/中/高各 20/50/100 请求、三档已校准。
func TestLoadEmbeddedAstraBenchmark_Smoke(t *testing.T) {
	require.NotEmpty(t, astrabenchmark.Package)

	bench, meta, err := LoadEmbeddedAstraBenchmark()
	require.NoError(t, err)
	require.NotNil(t, bench)
	require.NotNil(t, meta)

	require.Equal(t, "meow-gpt-baseline", bench.PackageID)
	require.Equal(t, "4.5.0-rc4", bench.Version)
	require.Equal(t, astraBenchmarkScoringVersion, bench.ScoringVersion)
	require.Equal(t, []string{"gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"}, bench.ModelIDs)
	require.Len(t, bench.Cells, 5)
	for _, cell := range bench.Cells {
		require.NotEmpty(t, cell.Prompt, cell.ID)
		require.Equal(t, "low", cell.Effort, cell.ID)
		require.Equal(t, 128, cell.MaxOutputTokens, cell.ID)
		fitted, ok := bench.Fitted[cell.ID]
		require.True(t, ok, cell.ID)
		require.True(t, fitted.ReferenceReady, cell.ID)
		require.Greater(t, fitted.Weight, 0.0, cell.ID)
		for _, model := range bench.ModelIDs {
			require.NotEmpty(t, fitted.Distributions[model], "%s/%s", cell.ID, model)
		}
	}

	require.Equal(t, 20, bench.TierRequests(AstraCheckTierLow))
	require.Equal(t, 50, bench.TierRequests(AstraCheckTierMedium))
	require.Equal(t, 100, bench.TierRequests(AstraCheckTierHigh))
	for _, tier := range astraCheckTiers {
		require.True(t, bench.Tiers[tier].Calibrated, tier)
		require.Len(t, bench.Tiers[tier].Thresholds, 4, tier)
	}
	require.Equal(t, "4.5.0-rc4", meta.Version)
	require.Len(t, meta.Models, 4)

	// 二次加载返回同一实例
	again, _, err := LoadEmbeddedAstraBenchmark()
	require.NoError(t, err)
	require.Same(t, bench, again)
}

func TestDecorateAstraCheckSummary_FillsCostAndBenchmarkMeta(t *testing.T) {
	summary := &GroupStatusSummary{AstraCheckInputTokens: 1000, AstraCheckOutputTokens: 100}
	decorateAstraCheckSummary(summary)
	require.InDelta(t, 0.015, summary.AstraCheckLastCostUSD, 1e-12)
	require.Equal(t, "4.5.0-rc4", summary.AstraCheckBenchmarkVersion)
	require.Len(t, summary.AstraCheckBenchmarkModels, 4)
	require.Len(t, summary.AstraCheckBenchmarkTiers, 3)
	require.Equal(t, 20, summary.AstraCheckBenchmarkTiers[0].Requests)
}
