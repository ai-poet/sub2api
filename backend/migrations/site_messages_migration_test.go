package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSiteMessagesMigration(t *testing.T) {
	content, err := FS.ReadFile("246_site_messages.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS site_messages (")
	require.Contains(t, sql, "user_id BIGINT NOT NULL,")
	require.Contains(t, sql, "category VARCHAR(20) NOT NULL DEFAULT 'admin'")
	require.Contains(t, sql, "title VARCHAR(200) NOT NULL")
	require.Contains(t, sql, "content TEXT NOT NULL")
	require.Contains(t, sql, "source_id VARCHAR(128) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "sender_user_id BIGINT NULL")
	require.Contains(t, sql, "approval_id BIGINT NULL")
	require.Contains(t, sql, "read_at TIMESTAMPTZ NULL")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_site_messages_user_created ON site_messages (user_id, created_at DESC, id DESC)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_site_messages_user_unread ON site_messages (user_id, created_at DESC) WHERE read_at IS NULL")
	// users 是软删除：不建外键
	require.NotContains(t, sql, "REFERENCES users")
}
