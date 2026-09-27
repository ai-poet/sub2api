package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersonalTokensMigration(t *testing.T) {
	content, err := FS.ReadFile("241_personal_tokens.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS personal_tokens (")
	// 每人至多一个令牌：重新生成靠 user_id 唯一约束 upsert 覆盖
	require.Contains(t, sql, "user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE")
	// 只存哈希，按哈希唯一查找
	require.Contains(t, sql, "token_hash CHAR(64) NOT NULL UNIQUE")
	require.Contains(t, sql, "user_token_version BIGINT NOT NULL")
	require.Contains(t, sql, "expires_at TIMESTAMPTZ NULL")
	require.NotContains(t, strings.ToLower(sql), "token_plain")
}
