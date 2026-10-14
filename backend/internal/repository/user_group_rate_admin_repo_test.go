package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func newUserGroupRateAdminRepoMock(t *testing.T) (*userGroupRateAdminRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &userGroupRateAdminRepository{db: db}, mock
}

func TestUserGroupRateAdminRepo_ListAppliesFiltersAndSkipsDeleted(t *testing.T) {
	repo, mock := newUserGroupRateAdminRepoMock(t)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*), COUNT(DISTINCT ugr.user_id)")).
		WithArgs(int64(10), "active", "%42%", int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "users"}).AddRow(int64(3), int64(2)))
	mock.ExpectQuery(`(?s)JOIN users u ON u.id = ugr.user_id AND u.deleted_at IS NULL.*JOIN groups g ON g.id = ugr.group_id AND g.deleted_at IS NULL.*WHERE ugr.rate_multiplier IS NOT NULL AND ugr.group_id = \$1 AND u.status = \$2 AND \(u.email ILIKE \$3 OR u.username ILIKE \$3 OR COALESCE\(u.notes, ''\) ILIKE \$3 OR u.id = \$4\) ORDER BY \(ugr.rate_multiplier / NULLIF\(g.rate_multiplier, 0\)\) ASC NULLS LAST, ugr.user_id ASC, ugr.group_id ASC LIMIT \$5 OFFSET \$6`).
		WithArgs(int64(10), "active", "%42%", int64(42), 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "email", "username", "notes", "status",
			"group_id", "name", "platform", "group_status", "group_rate",
			"rate", "rpm", "updated_at",
		}).
			AddRow(int64(42), "a@b.c", "alice", "vip", "active", int64(10), "Claude", "anthropic", "active", 1.0, 0.8, nil, now).
			AddRow(int64(43), "d@e.f", "", "", "active", int64(10), "Claude", "anthropic", "active", 1.0, 0.7, int64(5), now))

	out, err := repo.List(context.Background(), service.UserGroupRateListFilter{
		Page: 2, PageSize: 20, Search: "42", GroupID: 10, UserStatus: "active", SortBy: "discount", SortOrder: "asc",
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), out.Total)
	require.Equal(t, int64(2), out.UserCount)
	require.Len(t, out.Items, 2)
	require.Nil(t, out.Items[0].RPMOverride)
	require.Equal(t, 5, *out.Items[1].RPMOverride)
	require.InDelta(t, 0.8, out.Items[0].RateMultiplier, 1e-12)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildUserGroupRateAdminWhere_EscapesLikeAndSkipsIDForText(t *testing.T) {
	where, args := buildUserGroupRateAdminWhere(service.UserGroupRateListFilter{Search: "a_b%"})
	require.Equal(t, []any{`%a\_b\%%`}, args)
	require.NotContains(t, where, "u.id =")
}

func TestUserGroupRateAdminRepo_ListEmptySkipsPageQuery(t *testing.T) {
	repo, mock := newUserGroupRateAdminRepoMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*), COUNT(DISTINCT ugr.user_id)")).
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{"count", "users"}).AddRow(int64(0), int64(0)))

	out, err := repo.List(context.Background(), service.UserGroupRateListFilter{Page: 1, PageSize: 20, SortBy: "updated_at", SortOrder: "desc"})
	require.NoError(t, err)
	require.Empty(t, out.Items)
	require.NotNil(t, out.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserGroupRateAdminRepo_UsageAggregatesByUserAndGroup(t *testing.T) {
	repo, mock := newUserGroupRateAdminRepoMock(t)
	since := time.Now().Add(-30 * 24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE user_id = ANY($1) AND created_at >= $2 AND group_id IS NOT NULL\n\t\tGROUP BY user_id, group_id")).
		WithArgs(pq.Array([]int64{1, 2}), since).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id", "count", "sum"}).
			AddRow(int64(1), int64(10), int64(12), 3.5))

	out, err := repo.Usage(context.Background(), []int64{1, 2}, since)
	require.NoError(t, err)
	require.Equal(t, []service.UserGroupRateUsage{{UserID: 1, GroupID: 10, Requests: 12, ActualCost: 3.5}}, out)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserGroupRateAdminRepo_ApplyBatchInOneTransaction(t *testing.T) {
	repo, mock := newUserGroupRateAdminRepoMock(t)
	r1, r2 := 0.8, 1.2

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET rate_multiplier = NULL, updated_at = $3")).
		WithArgs(pq.Array([]int64{2}), pq.Array([]int64{11}), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("AND ugr.rate_multiplier IS NULL AND ugr.rpm_override IS NULL")).
		WithArgs(pq.Array([]int64{2}), pq.Array([]int64{11})).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier, updated_at = EXCLUDED.updated_at")).
		WithArgs(pq.Array([]int64{1, 3}), pq.Array([]int64{10, 10}), pq.Array([]float64{r1, r2}), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err := repo.ApplyBatch(context.Background(), []service.UserGroupRateChange{
		{UserID: 1, GroupID: 10, RateMultiplier: &r1},
		{UserID: 2, GroupID: 11},
		{UserID: 3, GroupID: 10, RateMultiplier: &r2},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserGroupRateAdminRepo_ApplyBatchRollsBackOnError(t *testing.T) {
	repo, mock := newUserGroupRateAdminRepoMock(t)
	r1 := 0.8

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO user_group_rate_multipliers").WillReturnError(errors.New("boom"))
	mock.ExpectRollback()

	err := repo.ApplyBatch(context.Background(), []service.UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: &r1}})
	require.EqualError(t, err, "boom")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserGroupRateAdminRepo_ExistingIDsSkipDeleted(t *testing.T) {
	repo, mock := newUserGroupRateAdminRepoMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM users WHERE id = ANY($1) AND deleted_at IS NULL")).
		WithArgs(pq.Array([]int64{1, 2})).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM groups WHERE id = ANY($1) AND deleted_at IS NULL")).
		WithArgs(pq.Array([]int64{10})).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

	users, err := repo.ExistingUserIDs(context.Background(), []int64{1, 2})
	require.NoError(t, err)
	require.Equal(t, []int64{1}, users)
	groups, err := repo.ExistingGroupIDs(context.Background(), []int64{10})
	require.NoError(t, err)
	require.Equal(t, []int64{10}, groups)
	require.NoError(t, mock.ExpectationsWereMet())
}
