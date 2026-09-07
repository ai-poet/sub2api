package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusAstraCheckMigration(t *testing.T) {
	content, err := FS.ReadFile("236_group_status_astra_check.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE group_status_configs ADD COLUMN IF NOT EXISTS astra_check_enabled BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS astra_check_request_model VARCHAR(255) NOT NULL DEFAULT 'gpt-6-astra'")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS astra_check_tier VARCHAR(16) NOT NULL DEFAULT 'low'")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS astra_check_interval_seconds INTEGER NOT NULL DEFAULT 3600")
	require.Contains(t, sql, "ALTER TABLE group_status_states ADD COLUMN IF NOT EXISTS astra_check_verdict VARCHAR(32) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS astra_check_matches JSONB NOT NULL DEFAULT '[]'::jsonb")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS astra_check_last_run_id BIGINT NULL")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS group_status_astra_check_runs")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_group_status_astra_check_runs_group_finished_at")
}
