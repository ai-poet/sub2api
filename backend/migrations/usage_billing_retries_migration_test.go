package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 扣费失败重试队列（fork 本地功能）：表结构与默认告警规则的关键语句不能被改掉。
func TestUsageBillingRetriesMigration(t *testing.T) {
	content, err := FS.ReadFile("248_usage_billing_retries.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS usage_billing_retries (")
	require.Contains(t, sql, "request_id VARCHAR(255) NOT NULL,")
	require.Contains(t, sql, "api_key_id BIGINT NOT NULL,")
	require.Contains(t, sql, "command JSONB NOT NULL,")
	require.Contains(t, sql, "status VARCHAR(16) NOT NULL DEFAULT 'pending'")
	require.Contains(t, sql, "attempts INT NOT NULL DEFAULT 0,")
	require.Contains(t, sql, "next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")
	// 与 usage_billing_dedup 同键：重放经去重表幂等
	require.Contains(t, sql, "CONSTRAINT uq_usage_billing_retries_request UNIQUE (request_id, api_key_id)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_usage_billing_retries_due ON usage_billing_retries (next_retry_at) WHERE status = 'pending'")
	// 默认告警规则只在缺席时插入
	require.Contains(t, sql, "metric_type = 'billing_failure_count'")
	require.Contains(t, sql, "WHERE NOT EXISTS (SELECT 1 FROM ops_alert_rules WHERE metric_type = 'billing_failure_count')")
	// users / api_keys 是软删除：不建外键
	require.NotContains(t, sql, "REFERENCES users")
	require.NotContains(t, sql, "REFERENCES api_keys")
}
