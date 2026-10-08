package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func contentModerationKeyStatuses() []service.ContentModerationAPIKeyStatus {
	return []service.ContentModerationAPIKeyStatus{
		{Index: 0, KeyHash: "hash-a", Masked: "sk-...aaaa", Status: "healthy", SuccessCount: 3, Configured: true},
		{Index: 1, KeyHash: "hash-b", Masked: "sk-...bbbb", Status: "frozen", FailureCount: 2, LastError: "429", Configured: true},
	}
}

func TestContentModerationConfigForViewer_OperatorLosesKeyIdentifiers(t *testing.T) {
	cfg := &service.ContentModerationConfigView{
		Engine:           "openai",
		Enabled:          true,
		BaseURL:          "https://api.openai.com",
		APIKeyConfigured: true,
		APIKeyMasked:     "sk-...aaaa",
		APIKeyCount:      2,
		APIKeyMasks:      []string{"sk-...aaaa", "sk-...bbbb"},
		APIKeyStatuses:   contentModerationKeyStatuses(),
		BlockedKeywords:  []string{"kw"},
		EngineConfigs: map[string]*service.ContentModerationConfigView{
			"typesafe": {Engine: "typesafe", APIKeyMasked: "ts-...cccc", APIKeyMasks: []string{"ts-...cccc"}, APIKeyStatuses: contentModerationKeyStatuses()},
			"empty":    nil,
		},
	}

	out := contentModerationConfigForViewer(opsTestContext(t, service.RoleOperator), cfg)

	require.NotSame(t, cfg, out)
	require.Empty(t, out.APIKeyMasked)
	require.Nil(t, out.APIKeyMasks)
	require.Len(t, out.APIKeyStatuses, 2)
	for _, status := range out.APIKeyStatuses {
		require.Empty(t, status.KeyHash)
		require.Empty(t, status.Masked)
	}
	require.Equal(t, "frozen", out.APIKeyStatuses[1].Status, "health data stays visible")
	require.Equal(t, 2, out.APIKeyStatuses[1].FailureCount)
	require.Equal(t, 2, out.APIKeyCount)
	require.True(t, out.APIKeyConfigured)
	require.Equal(t, []string{"kw"}, out.BlockedKeywords)

	typesafe := out.EngineConfigs["typesafe"]
	require.NotNil(t, typesafe)
	require.Empty(t, typesafe.APIKeyMasked)
	require.Nil(t, typesafe.APIKeyMasks)
	for _, status := range typesafe.APIKeyStatuses {
		require.Empty(t, status.KeyHash)
		require.Empty(t, status.Masked)
	}
	require.Contains(t, out.EngineConfigs, "empty")
	require.Nil(t, out.EngineConfigs["empty"])

	// 原始对象未被修改
	require.Equal(t, "sk-...aaaa", cfg.APIKeyMasked)
	require.Equal(t, "hash-a", cfg.APIKeyStatuses[0].KeyHash)
	require.Equal(t, "ts-...cccc", cfg.EngineConfigs["typesafe"].APIKeyMasked)
	require.Equal(t, "hash-b", cfg.EngineConfigs["typesafe"].APIKeyStatuses[1].KeyHash)
}

func TestContentModerationConfigForViewer_AdminUnchanged(t *testing.T) {
	cfg := &service.ContentModerationConfigView{APIKeyMasked: "sk-...aaaa", APIKeyStatuses: contentModerationKeyStatuses()}
	require.Same(t, cfg, contentModerationConfigForViewer(opsTestContext(t, service.RoleAdmin), cfg))
	require.Nil(t, contentModerationConfigForViewer(opsTestContext(t, service.RoleOperator), nil))
}

func TestContentModerationStatusForViewer(t *testing.T) {
	status := &service.ContentModerationRuntimeStatus{
		Enabled:        true,
		QueueLength:    5,
		APIKeyStatuses: contentModerationKeyStatuses(),
		PreBlockAPIKeyLoads: []service.ContentModerationAPIKeyLoad{
			{Index: 0, KeyHash: "hash-a", Masked: "sk-...aaaa", Status: "healthy", Total: 9},
		},
	}

	require.Same(t, status, contentModerationStatusForViewer(opsTestContext(t, service.RoleAdmin), status))

	out := contentModerationStatusForViewer(opsTestContext(t, service.RoleOperator), status)
	require.NotSame(t, status, out)
	require.Equal(t, 5, out.QueueLength)
	require.Len(t, out.PreBlockAPIKeyLoads, 1)
	require.Empty(t, out.PreBlockAPIKeyLoads[0].KeyHash)
	require.Empty(t, out.PreBlockAPIKeyLoads[0].Masked)
	require.Equal(t, int64(9), out.PreBlockAPIKeyLoads[0].Total)
	for _, s := range out.APIKeyStatuses {
		require.Empty(t, s.KeyHash)
		require.Empty(t, s.Masked)
	}

	require.Equal(t, "hash-a", status.PreBlockAPIKeyLoads[0].KeyHash, "original untouched")
	require.Equal(t, "sk-...bbbb", status.APIKeyStatuses[1].Masked, "original untouched")
	require.Nil(t, contentModerationStatusForViewer(opsTestContext(t, service.RoleOperator), nil))
}
