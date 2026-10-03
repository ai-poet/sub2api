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

func newSiteMessageRepoMock(t *testing.T) (*siteMessageRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &siteMessageRepository{db: db}, mock
}

func TestSiteMessageRepo_CreateOnlyForLiveUsers(t *testing.T) {
	repo, mock := newSiteMessageRepoMock(t)
	now := time.Now()
	approval := int64(9)
	sender := int64(3)

	mock.ExpectQuery(regexp.QuoteMeta("FROM users u\n\t\tWHERE u.id = $1 AND u.deleted_at IS NULL")).
		WithArgs(int64(5), "admin", "t", "c", "admin", "approval:9", int64(3), "operator", int64(9), now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(11), now))

	out, err := repo.Create(context.Background(), &service.SiteMessage{
		UserID: 5, Category: "admin", Title: "t", Content: "c", SourceType: "admin", SourceID: "approval:9",
		SenderUserID: &sender, SenderRole: "operator", ApprovalID: &approval, CreatedAt: now,
	})
	require.NoError(t, err)
	require.Equal(t, int64(11), out.ID)

	mock.ExpectQuery("INSERT INTO site_messages").WillReturnError(sql.ErrNoRows)
	_, err = repo.Create(context.Background(), &service.SiteMessage{UserID: 6, Title: "t", Content: "c", CreatedAt: now})
	require.ErrorIs(t, err, service.ErrUserNotFound)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSiteMessageRepo_ListAppliesFilters(t *testing.T) {
	repo, mock := newSiteMessageRepoMock(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM site_messages m WHERE m.user_id = $1 AND m.read_at IS NULL AND m.category = $2")).
		WithArgs(int64(5), "security").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("LEFT JOIN users s ON s.id = m.sender_user_id")).
		WithArgs(int64(5), "security", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "category", "title", "content", "source_type", "source_id",
			"sender_user_id", "sender_role", "email", "approval_id", "read_at", "created_at",
		}).AddRow(int64(1), int64(5), "security", "t", "c", "content_moderation", "log:1", nil, "", "", nil, nil, now))

	items, total, err := repo.ListByUser(context.Background(), 5, service.SiteMessageFilter{Page: 1, PageSize: 20, UnreadOnly: true, Category: "security"})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Nil(t, items[0].SenderUserID)
	require.Nil(t, items[0].ReadAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSiteMessageRepo_MarkReadScopedToOwner(t *testing.T) {
	repo, mock := newSiteMessageRepoMock(t)
	now := time.Now()

	// 命中：直接成功
	mock.ExpectExec(regexp.QuoteMeta("WHERE id = $1 AND user_id = $2 AND read_at IS NULL")).
		WithArgs(int64(1), int64(5), now).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.MarkRead(context.Background(), 5, 1, now))

	// 已读：幂等成功
	mock.ExpectExec("UPDATE site_messages").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).WithArgs(int64(1), int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	require.NoError(t, repo.MarkRead(context.Background(), 5, 1, now))

	// 不是自己的：按不存在处理
	mock.ExpectExec("UPDATE site_messages").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).WithArgs(int64(2), int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	require.ErrorIs(t, repo.MarkRead(context.Background(), 5, 2, now), service.ErrSiteMessageNotFound)

	mock.ExpectExec(regexp.QuoteMeta("WHERE user_id = $1 AND read_at IS NULL")).
		WithArgs(int64(5), now).WillReturnResult(sqlmock.NewResult(0, 3))
	updated, err := repo.MarkAllRead(context.Background(), 5, now)
	require.NoError(t, err)
	require.Equal(t, int64(3), updated)

	require.NoError(t, mock.ExpectationsWereMet())
}
