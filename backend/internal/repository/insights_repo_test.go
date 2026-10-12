package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestInsightsRepo_UserRankingFiltersAccountsBeforeRanking(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	start := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	// 管理员 / 运维 / 封禁 / 已删除账号在排名之前就被排除；排序片段来自白名单
	mock.ExpectQuery(`(?s)JOIN users us ON us\.id = uu\.user_id.*us\.role = \$4.*us\.status = \$5.*us\.deleted_at IS NULL.*uu\.requests > 0.*ROW_NUMBER\(\) OVER \(ORDER BY requests DESC, tokens DESC, user_id ASC\)`).
		WithArgs(start, end, 20, service.RoleUser, service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "email", "tokens", "requests", "actual_cost", "participants"}).
			AddRow(int64(7), "alice", "alice@example.com", int64(900), int64(30), 1.25, int64(12)).
			AddRow(int64(9), "", "bob@example.com", int64(800), int64(20), 0.5, int64(12)))

	repo := &insightsRepository{db: db}
	got, err := repo.UserUsageRanking(context.Background(), start, end, service.InsightsMetricRequests, 20)
	require.NoError(t, err)
	require.Len(t, got.Rows, 2)
	require.Equal(t, int64(12), got.Participants)
	require.Equal(t, "bob@example.com", got.Rows[1].Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsightsRepo_UserRankingRejectsUnknownMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := &insightsRepository{db: db}
	_, err = repo.UserUsageRanking(context.Background(), time.Now(), time.Now(), "email; DROP TABLE users", 20)
	require.ErrorIs(t, err, service.ErrInsightsInvalidMetric)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsightsRepo_ModelRankingUsesRequestedModelAndTotal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("COALESCE(NULLIF(TRIM(u.requested_model), ''), u.model) AS model")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 200).
		WillReturnRows(sqlmock.NewRows([]string{"model", "requests", "tokens", "total_tokens"}).
			AddRow("claude-sonnet-4-6", int64(10), int64(500), int64(1000)).
			AddRow("gpt-5.5", int64(4), int64(300), int64(1000)))

	repo := &insightsRepository{db: db}
	got, err := repo.ModelUsageRanking(context.Background(), time.Now().Add(-time.Hour), time.Now(), 200)
	require.NoError(t, err)
	require.Len(t, got.Rows, 2)
	require.Equal(t, int64(1000), got.TotalTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 日期范围按服务器时区解析成 [当天零点, 结束日次日零点)，查询时统一换成 UTC，结束边界不含。
func TestInsightsRepo_DateRangeBoundsAreSentInUTC(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	loc := time.FixedZone("UTC+8", 8*3600)
	start := time.Date(2026, 10, 6, 0, 0, 0, 0, loc)
	end := time.Date(2026, 10, 13, 0, 0, 0, 0, loc)
	mock.ExpectQuery(`(?s)WHERE u\.created_at >= \$1 AND u\.created_at < \$2`).
		WithArgs(time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC), time.Date(2026, 10, 12, 16, 0, 0, 0, time.UTC), 200).
		WillReturnRows(sqlmock.NewRows([]string{"model", "requests", "tokens", "total_tokens"}))
	mock.ExpectQuery(`(?s)WHERE u\.created_at >= \$1 AND u\.created_at < \$2.*ROW_NUMBER\(\) OVER \(ORDER BY tokens DESC, requests DESC, user_id ASC\)`).
		WithArgs(time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC), time.Date(2026, 10, 12, 16, 0, 0, 0, time.UTC), 20, service.RoleUser, service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "email", "tokens", "requests", "actual_cost", "participants"}))

	repo := &insightsRepository{db: db}
	models, err := repo.ModelUsageRanking(context.Background(), start, end, 200)
	require.NoError(t, err)
	require.Empty(t, models.Rows)
	users, err := repo.UserUsageRanking(context.Background(), start, end, service.InsightsMetricTokens, 20)
	require.NoError(t, err)
	require.Empty(t, users.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsightsRepo_PublicTotalsReadOnlyTheDailyAggregate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	today := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)FROM usage_dashboard_daily\s*$`).
		WithArgs(today).
		WillReturnRows(sqlmock.NewRows([]string{"total_tokens", "total_requests", "today_tokens"}).
			AddRow(int64(5000), int64(70), int64(300)))

	repo := &insightsRepository{db: db}
	got, err := repo.PublicUsageTotals(context.Background(), today)
	require.NoError(t, err)
	require.Equal(t, &service.InsightsUsageTotals{TotalTokens: 5000, TotalRequests: 70, TodayTokens: 300}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}
