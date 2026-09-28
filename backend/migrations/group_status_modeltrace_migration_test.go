package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusModelTraceMigration(t *testing.T) {
	content, err := FS.ReadFile("242_group_status_modeltrace.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE group_status_configs ADD COLUMN IF NOT EXISTS modeltrace_enabled BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS modeltrace_expected_model VARCHAR(255) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS modeltrace_request_model VARCHAR(255) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS modeltrace_interval_seconds INTEGER NOT NULL DEFAULT 3600")
	require.Contains(t, sql, "ALTER TABLE group_status_states ADD COLUMN IF NOT EXISTS modeltrace_verdict VARCHAR(32) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS modeltrace_ranking JSONB NOT NULL DEFAULT '[]'::jsonb")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS modeltrace_checked_at TIMESTAMPTZ NULL")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS modeltrace_last_run_id BIGINT NULL")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS group_status_modeltrace_runs")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_group_status_modeltrace_runs_group_finished_at")

	// 已开启 Sol Juice 的 OpenAI 分组迁移到 ModelTrace，并关掉旧开关
	require.Contains(t, sql, "modeltrace_expected_model = 'gpt-5.6-sol'")
	require.Contains(t, sql, "sol_juice_enabled = FALSE WHERE c.sol_juice_enabled = TRUE")
	require.Contains(t, sql, "platform = 'openai'")

	// 旧列 / 旧表刻意保留：SKIP_SETUP 与镜像回滚要求新 schema 兼容旧镜像
	require.NotContains(t, strings.ToUpper(sql), "DROP ")
}
