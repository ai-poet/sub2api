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

func newUsageBillingRetryRepoMock(t *testing.T) (*usageBillingRetryRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &usageBillingRetryRepository{db: db}, mock
}

func usageBillingRetryRowColumns() []string {
	return []string{
		"id", "request_id", "api_key_id", "user_id", "account_id", "group_id", "platform", "actual_cost",
		"command", "status", "attempts", "last_error", "next_retry_at", "settled_at", "created_at", "updated_at",
	}
}

func TestUsageBillingRetryRepo_EnqueueIsIdempotent(t *testing.T) {
	repo, mock := newUsageBillingRetryRepoMock(t)
	gid := int64(1)
	next := time.Date(2026, 10, 6, 8, 0, 30, 0, time.UTC)
	cmd := &service.UsageBillingCommand{RequestID: "client:abc", APIKeyID: 6483, UserID: 4641, AccountID: 11, BalanceCost: 0.0649}

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO usage_billing_retries (")).
		WithArgs("client:abc", int64(6483), int64(4641), int64(11), sql.NullInt64{Int64: 1, Valid: true}, "anthropic",
			0.0649, sqlmock.AnyArg(), "pending", "api key not found", next).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(5)))
	item := &service.UsageBillingRetry{
		RequestID: " client:abc ", APIKeyID: 6483, UserID: 4641, AccountID: 11, GroupID: &gid, Platform: "anthropic",
		ActualCost: 0.0649, Command: cmd, LastError: "api key not found", NextRetryAt: next,
	}
	queued, err := repo.Enqueue(context.Background(), item)
	require.NoError(t, err)
	require.True(t, queued)
	require.Equal(t, int64(5), item.ID)
	require.Equal(t, "client:abc", item.RequestID)
	require.Equal(t, service.UsageBillingRetryStatusPending, item.Status)

	// 同键已存在：ON CONFLICT DO NOTHING 不返回行 → 不是错误，也不覆盖
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO usage_billing_retries (")).WillReturnError(sql.ErrNoRows)
	queued, err = repo.Enqueue(context.Background(), &service.UsageBillingRetry{RequestID: "client:abc", APIKeyID: 6483, Command: cmd})
	require.NoError(t, err)
	require.False(t, queued)

	_, err = repo.Enqueue(context.Background(), &service.UsageBillingRetry{RequestID: "  ", Command: cmd})
	require.ErrorIs(t, err, service.ErrUsageBillingRequestIDRequired)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageBillingRetryRepo_ClaimDueDecodesCommand(t *testing.T) {
	repo, mock := newUsageBillingRetryRepoMock(t)
	now := time.Date(2026, 10, 6, 8, 1, 0, 0, time.UTC)
	created := now.Add(-time.Minute)

	mock.ExpectQuery(`UPDATE usage_billing_retries\s+SET attempts = attempts \+ 1,[\s\S]*FOR UPDATE SKIP LOCKED`).
		WithArgs(now, now.Add(2*time.Minute), "pending", 50).
		WillReturnRows(sqlmock.NewRows(usageBillingRetryRowColumns()).AddRow(
			int64(7), "client:abc", int64(6483), int64(4641), int64(11), int64(1), "anthropic", 0.0649,
			[]byte(`{"RequestID":"client:abc","APIKeyID":6483,"UserID":4641,"AccountID":11,"RequestFingerprint":"fp","BalanceCost":0.0649,"APIKeyQuotaCost":0.0649}`),
			"pending", 1, "api key not found", now.Add(2*time.Minute), nil, created, created,
		))

	items, err := repo.ClaimDue(context.Background(), now, 2*time.Minute, 50)
	require.NoError(t, err)
	require.Len(t, items, 1)
	item := items[0]
	require.Equal(t, int64(7), item.ID)
	require.Equal(t, 1, item.Attempts)
	require.NotNil(t, item.GroupID)
	require.Equal(t, int64(1), *item.GroupID)
	require.Nil(t, item.SettledAt)
	require.NotNil(t, item.Command)
	require.Equal(t, "fp", item.Command.RequestFingerprint)
	require.InDelta(t, 0.0649, item.Command.BalanceCost, 1e-12)
	require.InDelta(t, 0.0649, item.Command.APIKeyQuotaCost, 1e-12)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageBillingRetryRepo_ListByKeysFiltersAPIKey(t *testing.T) {
	repo, mock := newUsageBillingRetryRepoMock(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE request_id = ANY($1)")).
		WillReturnRows(sqlmock.NewRows(usageBillingRetryRowColumns()).
			AddRow(int64(1), "client:abc", int64(6483), int64(4641), int64(11), nil, "", 0.1, []byte(`{}`), "pending", 2, "boom", now, nil, now, now).
			AddRow(int64(2), "client:abc", int64(9999), int64(4641), int64(11), nil, "", 0.1, []byte(`{}`), "failed", 24, "boom", now, nil, now, now))

	items, err := repo.ListByKeys(context.Background(), []service.UsageBillingRetryKey{
		{RequestID: "client:abc", APIKeyID: 6483},
		{RequestID: "client:abc", APIKeyID: 6483}, // 重复键只查一次
		{RequestID: "", APIKeyID: 1},
	})
	require.NoError(t, err)
	require.Len(t, items, 1, "同 request_id 但 api_key_id 不同的行必须被过滤掉")
	require.Equal(t, int64(6483), items[0].APIKeyID)
	require.Nil(t, items[0].GroupID)
	require.NoError(t, mock.ExpectationsWereMet())

	items, err = repo.ListByKeys(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, items)
}

func TestUsageBillingRetryRepo_SummaryAndCounts(t *testing.T) {
	repo, mock := newUsageBillingRetryRepoMock(t)
	start := time.Date(2026, 10, 6, 7, 55, 0, 0, time.UTC)
	end := start.Add(5 * time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE status IN ($1, $2)")).
		WithArgs("pending", "failed").
		WillReturnRows(sqlmock.NewRows([]string{"status", "count", "sum"}).
			AddRow("pending", int64(3), 1.5).
			AddRow("failed", int64(1), 0.25))
	summary, err := repo.Summary(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(3), summary.Pending)
	require.InDelta(t, 1.5, summary.PendingCost, 1e-12)
	require.Equal(t, int64(1), summary.Failed)
	require.InDelta(t, 0.25, summary.FailedCost, 1e-12)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE created_at >= $1 AND created_at < $2")).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(4)))
	n, err := repo.CountCreatedBetween(context.Background(), start, end)
	require.NoError(t, err)
	require.Equal(t, int64(4), n)

	mock.ExpectExec(regexp.QuoteMeta("SET status = $2, settled_at = $3, last_error = '', updated_at = NOW()")).
		WithArgs(int64(7), "settled", end).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.MarkSettled(context.Background(), 7, end))

	mock.ExpectExec(regexp.QuoteMeta("SET status = $2, last_error = $3, next_retry_at = $4, updated_at = NOW()")).
		WithArgs(int64(7), "pending", "pq: canceling statement due to user request", end).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.MarkRetry(context.Background(), 7, "pq: canceling statement due to user request", end))

	mock.ExpectExec(regexp.QuoteMeta("SET status = $2, last_error = $3, updated_at = NOW()")).
		WithArgs(int64(7), "failed", "usage billing request fingerprint conflict").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.MarkFailed(context.Background(), 7, "usage billing request fingerprint conflict"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTruncateUsageBillingRetryError(t *testing.T) {
	require.Equal(t, "", truncateUsageBillingRetryError("   "))
	long := make([]rune, usageBillingRetryErrorMaxLen+10)
	for i := range long {
		long[i] = '错'
	}
	out := truncateUsageBillingRetryError(string(long))
	require.Equal(t, usageBillingRetryErrorMaxLen, len([]rune(out)))
}
