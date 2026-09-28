package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusAstraCheckDefaultMediumMigration(t *testing.T) {
	content, err := FS.ReadFile("244_group_status_astra_check_default_medium.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE group_status_configs ALTER COLUMN astra_check_tier SET DEFAULT 'medium'")
	// 只改默认值，不改已保存分组的档位
	require.NotContains(t, strings.ToUpper(sql), "UPDATE ")
}
