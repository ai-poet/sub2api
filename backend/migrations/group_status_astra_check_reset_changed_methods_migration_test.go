package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupStatusAstraCheckResetChangedMethodsMigration(t *testing.T) {
	content, err := FS.ReadFile("245_group_status_astra_check_reset_changed_methods.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "UPDATE group_status_astra_check_states")
	require.Contains(t, sql, "stable_status = ''")
	require.Contains(t, sql, "checked_at = NULL")
	require.Contains(t, sql, "consecutive_mismatch = 0")
	// 只动方法换过的两个目标上旧来源的行，新方法自己的结果不动
	require.Contains(t, sql, "(expected_model = 'gpt-5.6-sol' AND benchmark_package_id <> 'sol-juice')")
	require.Contains(t, sql, "(expected_model = 'claude-opus-5.5' AND benchmark_package_id <> 'modeltrace-bank')")
	require.NotContains(t, strings.ToUpper(sql), "DELETE ")
	require.NotContains(t, strings.ToUpper(sql), "DROP ")
}
