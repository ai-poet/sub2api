package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentTranslationsMigration(t *testing.T) {
	content, err := FS.ReadFile("247_content_translations.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS content_translations (")
	require.Contains(t, sql, "source_hash CHAR(64) NOT NULL,")
	require.Contains(t, sql, "target_lang VARCHAR(8) NOT NULL,")
	require.Contains(t, sql, "translated_text TEXT NOT NULL,")
	require.Contains(t, sql, "manual BOOLEAN NOT NULL DEFAULT FALSE,")
	require.Contains(t, sql, "last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS ux_content_translations_hash_lang ON content_translations (source_hash, target_lang)")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS content_translation_sources (")
	require.Contains(t, sql, "PRIMARY KEY (namespace, source_hash)")
	// 译文永久缓存：迁移里不能有任何清理逻辑
	require.NotContains(t, strings.ToUpper(sql), "DELETE FROM")
}
