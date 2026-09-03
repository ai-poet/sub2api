package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusConfigNotifyEnabledMigration(t *testing.T) {
	content, err := FS.ReadFile("234_group_status_config_notify_enabled.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql,
		"ALTER TABLE group_status_configs ADD COLUMN IF NOT EXISTS notify_enabled BOOLEAN NOT NULL DEFAULT TRUE")
	require.Contains(t, sql, "COMMENT ON COLUMN group_status_configs.notify_enabled")
}
