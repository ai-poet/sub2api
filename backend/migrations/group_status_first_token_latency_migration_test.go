package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusFirstTokenLatencyMigration(t *testing.T) {
	content, err := FS.ReadFile("238_group_status_first_token_latency.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE group_status_records ADD COLUMN IF NOT EXISTS total_latency_ms BIGINT NULL")
	require.Contains(t, sql, "ALTER TABLE group_status_states ADD COLUMN IF NOT EXISTS total_latency_ms BIGINT NULL")
}
