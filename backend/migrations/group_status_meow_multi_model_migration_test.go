package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusMeowMultiModelMigration(t *testing.T) {
	content, err := FS.ReadFile("243_group_status_meow_multi_model.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE group_status_configs ADD COLUMN IF NOT EXISTS astra_check_models JSONB NOT NULL DEFAULT '[]'::jsonb")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS group_status_astra_check_states")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_group_status_astra_check_states_group_model ON group_status_astra_check_states(group_id, expected_model)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS expected_model VARCHAR(255) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_group_status_astra_check_runs_group_model_finished_at")

	// 旧单模型 Astra 与 ModelTrace（含 242 从纯 Sol 迁来的分组）都并入模型列表，ModelTrace 开关随后关闭
	require.Contains(t, sql, "'expected_model', 'gpt-6-astra'")
	require.Contains(t, sql, "WHEN 'claude-opus-5-5' THEN 'claude-opus-5.5'")
	require.Contains(t, sql, "UPDATE group_status_configs SET modeltrace_enabled = FALSE WHERE modeltrace_enabled = TRUE")

	// 旧列 / 旧表刻意保留：SKIP_SETUP 与镜像回滚要求新 schema 兼容旧镜像
	require.NotContains(t, strings.ToUpper(sql), "DROP ")
}
