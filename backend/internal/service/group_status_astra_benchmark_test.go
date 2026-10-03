package service

import (
	"encoding/json"
	"testing"

	astrabenchmark "github.com/Wei-Shaw/sub2api/resources/astra-benchmark"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbeddedAstraBenchmarks_AllPackagesParse(t *testing.T) {
	reg, err := LoadEmbeddedAstraBenchmarks()
	require.NoError(t, err)
	require.Len(t, reg.Packages, len(astrabenchmark.Files))

	gpt6 := reg.Package("meow-gpt-other-cap98-efficient")
	require.NotNil(t, gpt6)
	require.Equal(t, "4.5.4-predictive.20260924.2", gpt6.Version)
	require.Equal(t, []string{"gpt-6-astra", "gpt-6-sol", "gpt-5.6-terra", "gpt-6-luna", AstraCheckOtherModel}, gpt6.ModelIDs)
	require.Equal(t, 32, gpt6.TierRequests(AstraCheckTierLow))
	require.Equal(t, 64, gpt6.TierRequests(AstraCheckTierMedium))
	require.Equal(t, 128, gpt6.TierRequests(AstraCheckTierHigh))
	require.Equal(t, "98f8d12c83100352addf44db15d8b57aa183338a4fb5f83ba30ffdbc06d78612", gpt6.BodySHA256)

	// GPT-5.6 Sol 改用 Juice 读数，不再内置只为它服务的 4.5.3 包
	require.Nil(t, reg.Package("meow-gpt-other-cap98"))

	claude := reg.Package("meow-claude-other-cap98-efficient")
	require.NotNil(t, claude)
	require.Equal(t, astraBenchmarkModeClaude, claude.Mode)
	require.True(t, claude.HasModel("claude-opus-5.5"))
	require.True(t, claude.HasModel("claude-fable-5.1"))
	require.Equal(t, 48, claude.TierRequests(AstraCheckTierLow))
	for _, cell := range claude.Cells {
		require.Equal(t, "claude-code", cell.Profile)
	}

	for _, pkg := range reg.Packages {
		require.Equal(t, astraBenchmarkScoringVersion, pkg.ScoringVersion)
		require.InDelta(t, 0.6, pkg.CompletionRatio, 1e-12)
		require.NotEmpty(t, pkg.ReferenceSources)
		for _, tier := range astraCheckTiers {
			require.True(t, pkg.Tiers[tier].Calibrated, "%s %s", pkg.PackageID, tier)
		}
	}
	require.Len(t, reg.Metas(), len(astrabenchmark.Files))
}

func TestAstraCheckTargets_AreInTheirBenchmarkPackages(t *testing.T) {
	reg, err := LoadEmbeddedAstraBenchmarks()
	require.NoError(t, err)
	for _, target := range astraCheckTargets {
		if target.Method == AstraCheckMethodSolJuice || target.Method == AstraCheckMethodModelTrace {
			require.Empty(t, target.PackageID, "%s target %s must not point at a meow package", target.Method, target.ID)
			continue
		}
		require.Equal(t, AstraCheckMethodMeow, target.Method, "target %s", target.ID)
		pkg := reg.Package(target.PackageID)
		require.NotNil(t, pkg, "target %s points at a missing package %s", target.ID, target.PackageID)
		require.True(t, pkg.HasModel(target.ID), "package %s does not contain target %s", target.PackageID, target.ID)
		require.NotEmpty(t, target.DefaultRequestModel)
	}
	ids := func(platform string) []string {
		var out []string
		for _, target := range AstraCheckTargetsForPlatform(platform) {
			out = append(out, target.ID)
		}
		return out
	}
	require.Equal(t, []string{"gpt-5.6-sol", "gpt-6-sol", "gpt-6-astra", "gpt-6.1-sol"}, ids(PlatformOpenAI))
	require.Equal(t, []string{"claude-opus-5.5", "claude-opus-5", "claude-fable-5.1"}, ids(PlatformAnthropic))
	require.Empty(t, ids(PlatformGemini))
	gpt56, ok := astraCheckTarget("gpt-5.6-sol")
	require.True(t, ok)
	require.Equal(t, AstraCheckMethodSolJuice, gpt56.Method)
	require.True(t, astraCheckTargetAllowed(PlatformAnthropic, "claude-opus-5.5"))
	require.False(t, astraCheckTargetAllowed(PlatformOpenAI, "claude-opus-5.5"))
	require.True(t, astraCheckTargetAllowed(PlatformAnthropic, "claude-opus-5"))
	require.False(t, astraCheckTargetAllowed(PlatformOpenAI, "claude-opus-5"))
}

func mutateEmbeddedAstraPackage(t *testing.T, mutate func(pkg map[string]any)) []byte {
	t.Helper()
	raw, err := astrabenchmark.FS.ReadFile(astrabenchmark.Files[0])
	require.NoError(t, err)
	var pkg map[string]any
	require.NoError(t, json.Unmarshal(raw, &pkg))
	mutate(pkg)
	out, err := json.Marshal(pkg)
	require.NoError(t, err)
	return out
}

func TestParseAstraBenchmark_RejectsBrokenPackages(t *testing.T) {
	fitted := func(pkg map[string]any) map[string]any { return pkg["fitted"].(map[string]any) }
	firstCell := func(pkg map[string]any) map[string]any {
		for _, cell := range fitted(pkg)["cells"].(map[string]any) {
			return cell.(map[string]any)
		}
		return nil
	}
	cases := map[string]func(pkg map[string]any){
		"scoring version": func(pkg map[string]any) { pkg["engine"].(map[string]any)["scoring_version"] = "meow-fingerprint-v2" },
		"mode":            func(pkg map[string]any) { pkg["mode"] = "chat" },
		"aggregation":     func(pkg map[string]any) { fitted(pkg)["aggregation"] = "mixture" },
		"model order": func(pkg map[string]any) {
			models := fitted(pkg)["models"].([]any)
			models[0], models[1] = models[1], models[0]
		},
		"missing alpha": func(pkg map[string]any) {
			delete(firstCell(pkg)["alpha"].(map[string]any), "gpt-6-sol")
		},
		"unseen category": func(pkg map[string]any) {
			cell := firstCell(pkg)
			categories := cell["categories"].([]any)
			for i, category := range categories {
				if category == astraBenchmarkUnseenCategory {
					categories[i] = "zzz"
				}
			}
		},
		"non-positive alpha": func(pkg map[string]any) {
			alpha := firstCell(pkg)["alpha"].(map[string]any)["gpt-6-astra"].([]any)
			alpha[0] = 0.0
		},
		"missing threshold": func(pkg map[string]any) {
			delete(pkg["tiers"].(map[string]any)["low"].(map[string]any)["thresholds"].(map[string]any), "gpt-6-sol")
		},
		"completion ratio": func(pkg map[string]any) { pkg["engine"].(map[string]any)["completion_ratio"] = 0.0 },
		"reference sources": func(pkg map[string]any) {
			fitted(pkg)["reference_sources"] = []any{}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := ParseAstraBenchmark(mutateEmbeddedAstraPackage(t, mutate))
			require.Error(t, err)
		})
	}
}

func TestDecorateAstraCheckSummary_OrdersStatesByConfigWithPlaceholders(t *testing.T) {
	checkedAt := mustParseTimeForTest(t, "2026-09-28T10:00:00Z")
	summary := &GroupStatusSummary{
		GroupID:  7,
		ConfigID: 3,
		AstraCheckModels: []AstraCheckModelConfig{
			{ExpectedModel: "gpt-6-sol"},
			{ExpectedModel: "gpt-6-astra"},
		},
		AstraCheckStates: []GroupStatusAstraCheckState{
			{GroupID: 7, ExpectedModel: "gpt-6-astra", Verdict: AstraCheckVerdictMatch, CheckedAt: &checkedAt},
			{GroupID: 7, ExpectedModel: "gpt-5.6-sol", Verdict: AstraCheckVerdictMismatch}, // 已移出配置
		},
	}
	decorateAstraCheckSummary(summary)
	require.Len(t, summary.AstraCheckStates, 2)
	require.Equal(t, "gpt-6-sol", summary.AstraCheckStates[0].ExpectedModel)
	require.Equal(t, "GPT-6 Sol", summary.AstraCheckStates[0].DisplayName)
	require.Nil(t, summary.AstraCheckStates[0].CheckedAt)
	require.NotNil(t, summary.AstraCheckStates[0].Matches)
	require.NotNil(t, summary.AstraCheckStates[0].Reasons)
	require.Equal(t, "gpt-6-astra", summary.AstraCheckStates[1].ExpectedModel)
	require.Equal(t, AstraCheckVerdictMatch, summary.AstraCheckStates[1].Verdict)
	require.Len(t, summary.AstraCheckBenchmarks, len(astrabenchmark.Files))
}
