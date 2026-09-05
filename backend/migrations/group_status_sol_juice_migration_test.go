package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusSolJuiceMigration(t *testing.T) {
	content, err := FS.ReadFile("235_group_status_sol_juice.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE group_status_configs ADD COLUMN IF NOT EXISTS sol_juice_enabled BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS sol_juice_interval_seconds INTEGER NOT NULL DEFAULT 900")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS sol_juice_model VARCHAR(255) NOT NULL DEFAULT 'gpt-5.6-sol'")
	require.Contains(t, sql, "ALTER TABLE group_status_states ADD COLUMN IF NOT EXISTS sol_juice_status VARCHAR(32) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS sol_juice_checked_at TIMESTAMPTZ NULL")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS sol_juice_reasoning_tokens BIGINT NOT NULL DEFAULT 0")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS group_status_juice_records")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_group_status_juice_records_group_observed_at")
}
