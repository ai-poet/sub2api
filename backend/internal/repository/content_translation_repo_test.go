package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newContentTranslationRepoMock(t *testing.T) (*contentTranslationRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &contentTranslationRepository{db: db}, mock
}

var contentTranslationRowColumns = []string{
	"id", "source_hash", "target_lang", "source_text", "translated_text", "model", "manual",
	"created_at", "updated_at", "last_seen_at",
}

func TestContentTranslationRepo_UpsertNeverOverwritesManual(t *testing.T) {
	repo, mock := newContentTranslationRepoMock(t)
	now := time.Now()
	entry := &service.ContentTranslation{SourceHash: "h", TargetLang: "en", SourceText: "原文", TranslatedText: "machine", Model: "m"}

	mock.ExpectQuery(regexp.QuoteMeta("WHERE content_translations.manual = FALSE")).
		WithArgs("h", "en", "原文", "machine", "m").
		WillReturnRows(sqlmock.NewRows(contentTranslationRowColumns).
			AddRow(int64(1), "h", "en", "原文", "machine", "m", false, now, now, now))
	out, err := repo.Upsert(context.Background(), entry)
	require.NoError(t, err)
	require.Equal(t, "machine", out.TranslatedText)

	// 已有人工译文：ON CONFLICT 的 WHERE 不成立，没有 RETURNING 行，读回人工译文
	mock.ExpectQuery(regexp.QuoteMeta("ON CONFLICT (source_hash, target_lang) DO UPDATE")).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE source_hash = $1 AND target_lang = $2")).
		WithArgs("h", "en").
		WillReturnRows(sqlmock.NewRows(contentTranslationRowColumns).
			AddRow(int64(1), "h", "en", "原文", "manual text", "m", true, now, now, now))
	out, err = repo.Upsert(context.Background(), entry)
	require.NoError(t, err)
	require.True(t, out.Manual)
	require.Equal(t, "manual text", out.TranslatedText)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentTranslationRepo_ListFiltersAndEscapesQuery(t *testing.T) {
	repo, mock := newContentTranslationRepoMock(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM content_translations WHERE TRUE AND target_lang = $1 AND (source_text ILIKE $2 OR translated_text ILIKE $2)")).
		WithArgs("ja", `%100\%\_off%`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("ORDER BY updated_at DESC, id DESC")).
		WithArgs("ja", `%100\%\_off%`, 20, 0).
		WillReturnRows(sqlmock.NewRows(contentTranslationRowColumns).
			AddRow(int64(3), "h", "ja", "100%_off", "100%_オフ", "m", false, now, now, now))

	items, total, err := repo.List(context.Background(), service.ContentTranslationFilter{Lang: "ja", Query: "100%_off"})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "100%_オフ", items[0].TranslatedText)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentTranslationRepo_DeleteAndUpdateNotFound(t *testing.T) {
	repo, mock := newContentTranslationRepoMock(t)
	mock.ExpectQuery("DELETE FROM content_translations WHERE id = \\$1").WithArgs(int64(9)).WillReturnError(sql.ErrNoRows)
	_, err := repo.Delete(context.Background(), 9)
	require.ErrorIs(t, err, service.ErrContentTranslationNotFound)

	mock.ExpectQuery("UPDATE content_translations").WithArgs(int64(9), "x").WillReturnError(sql.ErrNoRows)
	_, err = repo.UpdateManual(context.Background(), 9, "x")
	require.ErrorIs(t, err, service.ErrContentTranslationNotFound)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM content_translations WHERE manual = FALSE")).
		WillReturnResult(sqlmock.NewResult(0, 4))
	n, err := repo.DeleteMachine(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(4), n)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentTranslationRepo_ReplaceNamespaceSourcesInOneTransaction(t *testing.T) {
	repo, mock := newContentTranslationRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM content_translation_sources WHERE namespace = $1 AND NOT (source_hash = ANY($2))")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO content_translation_sources")).
		WithArgs("pay", "a", "套餐 A").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO content_translation_sources")).
		WithArgs("pay", "b", "套餐 B").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.ReplaceNamespaceSources(context.Background(), "pay", map[string]string{"b": "套餐 B", "a": "套餐 A"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentTranslationRepo_TouchSeenSkipsEmpty(t *testing.T) {
	repo, mock := newContentTranslationRepoMock(t)
	require.NoError(t, repo.TouchSeen(context.Background(), nil, time.Now()))

	at := time.Now()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE content_translations SET last_seen_at = $2 WHERE source_hash = ANY($1)")).
		WillReturnResult(sqlmock.NewResult(0, 2))
	require.NoError(t, repo.TouchSeen(context.Background(), []string{"a", "b"}, at))
	require.NoError(t, mock.ExpectationsWereMet())
}
