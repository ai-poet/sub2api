package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func sampleUsageLogWithRelations() *service.UsageLog {
	now := time.Date(2026, 9, 12, 8, 0, 0, 0, time.UTC)
	ip := "203.0.113.42"
	session := "sess-1"
	cost := 0.5
	multiplier := 1.2
	return &service.UsageLog{
		ID:                    1,
		UserID:                7,
		APIKeyID:              70,
		RequestID:             "req_1",
		Model:                 "claude-sonnet-5",
		IPAddress:             &ip,
		SessionID:             &session,
		AccountStatsCost:      &cost,
		AccountRateMultiplier: &multiplier,
		CreatedAt:             now,
		User: &service.User{
			ID:            7,
			Email:         "alice@example.com",
			Username:      "alice",
			Role:          service.RoleUser,
			Status:        service.StatusActive,
			Balance:       123.45,
			FrozenBalance: 6.7,
			Concurrency:   5,
		},
		APIKey: &service.APIKey{
			ID:          70,
			UserID:      7,
			Key:         "sk-live-secret",
			Name:        "prod",
			Status:      service.StatusActive,
			IPWhitelist: []string{"1.1.1.1"},
		},
		Account: &service.Account{ID: 3, Name: "acct-a"},
	}
}

func marshalOperatorToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

func TestUsageLogFromServiceOperator_RedactsSecretsBalanceAndIP(t *testing.T) {
	t.Parallel()

	out := marshalOperatorToMap(t, UsageLogFromServiceOperator(sampleUsageLogWithRelations()))

	require.Equal(t, "203.0.113.x", out["ip_address"])
	_, hasSession := out["session_id"]
	require.False(t, hasSession)
	_, hasAccountCost := out["account_stats_cost"]
	require.False(t, hasAccountCost)
	require.Nil(t, out["account_rate_multiplier"])

	user, ok := out["user"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "alice@example.com", user["email"])
	require.Equal(t, "alice", user["username"])
	for _, forbidden := range []string{"balance", "frozen_balance", "concurrency", "rpm_limit", "allowed_groups", "total_recharged"} {
		_, present := user[forbidden]
		require.Falsef(t, present, "user.%s must not be exposed to operator", forbidden)
	}

	apiKey, ok := out["api_key"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "prod", apiKey["name"])
	for _, forbidden := range []string{"key", "ip_whitelist", "ip_blacklist", "last_used_ip", "quota", "user"} {
		_, present := apiKey[forbidden]
		require.Falsef(t, present, "api_key.%s must not be exposed to operator", forbidden)
	}

	// 账号引用只保留 id / name，排障需要知道是哪个上游账号。
	account, ok := out["account"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "acct-a", account["name"])
}

func TestUsageLogFromServiceOperator_UnparseableIPIsDropped(t *testing.T) {
	t.Parallel()

	log := sampleUsageLogWithRelations()
	weird := "not-an-ip"
	log.IPAddress = &weird
	out := marshalOperatorToMap(t, UsageLogFromServiceOperator(log))
	_, present := out["ip_address"]
	require.False(t, present)
}

func TestUsageLogFromServiceAdmin_ListDoesNotCarryPlaintextKey(t *testing.T) {
	t.Parallel()

	out := marshalOperatorToMap(t, UsageLogFromServiceAdmin(sampleUsageLogWithRelations()))
	apiKey, ok := out["api_key"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "", apiKey["key"], "admin usage list must not embed the plaintext key")
	require.Equal(t, "prod", apiKey["name"])
	require.Nil(t, apiKey["ip_whitelist"])
	require.Nil(t, apiKey["user"])
	// 管理员仍然看到完整 IP 与用户余额（未变更的既有行为）。
	require.Equal(t, "203.0.113.42", out["ip_address"])
	user := out["user"].(map[string]any)
	require.Equal(t, 123.45, user["balance"])
}
