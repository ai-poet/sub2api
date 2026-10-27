package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultUserConcurrencyMigration(t *testing.T) {
	content, err := FS.ReadFile("249_default_user_concurrency.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN concurrency_is_default BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "SET concurrency_is_default = (concurrency = v_default_concurrency)")
	require.Contains(t, sql, "WHERE key = 'default_concurrency'")
	require.Contains(t, sql, "v_default_concurrency := COALESCE(v_default_concurrency, 5)")
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION enqueue_user_auth_cache_invalidation()")
	require.Contains(t, sql, "AND OLD.concurrency IS NOT DISTINCT FROM NEW.concurrency")
	// The backfill is guarded by the missing-column check so reruns preserve choices.
	require.Contains(t, sql, "column_name = 'concurrency_is_default'")
	require.NotContains(t, sql, "ALTER TABLE users ALTER COLUMN concurrency")
}
