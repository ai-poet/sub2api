package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// fork：排行榜与首页公开统计的查询。排行榜按固定时间窗扫 usage_logs（走 created_at 索引），
// 结果由 service 层按档位缓存；公开统计只读预聚合日表。

type insightsRepository struct {
	db *sql.DB
}

func NewInsightsRepository(db *sql.DB) service.InsightsRepository {
	return &insightsRepository{db: db}
}

// insightsTokensExpr 是一条日志的总 token 数。
const insightsTokensExpr = "(u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens)"

// insightsModelExpr 是用户请求的模型名（与用户侧用量统计一致，取 requested_model）。
const insightsModelExpr = "COALESCE(NULLIF(TRIM(u.requested_model), ''), u.model)"

// insightsUserOrderBy 是用户榜各指标的排序片段；只从这张白名单取，设置值不会拼进 SQL。
var insightsUserOrderBy = map[string]string{
	service.InsightsMetricTokens:   "tokens DESC, requests DESC, user_id ASC",
	service.InsightsMetricRequests: "requests DESC, tokens DESC, user_id ASC",
	service.InsightsMetricCost:     "actual_cost DESC, tokens DESC, user_id ASC",
}

// insightsUserMetricColumn 是用户榜各指标的列，用于排除该指标为 0 的用户。
var insightsUserMetricColumn = map[string]string{
	service.InsightsMetricTokens:   "tokens",
	service.InsightsMetricRequests: "requests",
	service.InsightsMetricCost:     "actual_cost",
}

func (r *insightsRepository) ModelUsageRanking(ctx context.Context, start, end time.Time, limit int) (result *service.InsightsModelUsage, err error) {
	if limit <= 0 {
		limit = 50
	}
	query := fmt.Sprintf(`
		WITH model_usage AS (
			SELECT
				%s AS model,
				COUNT(*) AS requests,
				COALESCE(SUM(%s), 0) AS tokens
			FROM usage_logs u
			WHERE u.created_at >= $1 AND u.created_at < $2
				AND (%s > 0 OR u.actual_cost > 0)
			GROUP BY 1
		)
		SELECT model, requests, tokens, COALESCE(SUM(tokens) OVER (), 0) AS total_tokens
		FROM model_usage
		ORDER BY tokens DESC, requests DESC, model ASC
		LIMIT $3
	`, insightsModelExpr, insightsTokensExpr, insightsTokensExpr)

	rows, err := r.db.QueryContext(ctx, query, start.UTC(), end.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()

	usage := &service.InsightsModelUsage{Rows: []service.InsightsModelUsageRow{}}
	for rows.Next() {
		var row service.InsightsModelUsageRow
		if err = rows.Scan(&row.Model, &row.Requests, &row.Tokens, &usage.TotalTokens); err != nil {
			return nil, err
		}
		usage.Rows = append(usage.Rows, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return usage, nil
}

func (r *insightsRepository) UserUsageRanking(ctx context.Context, start, end time.Time, metric string, limit int) (result *service.InsightsUserUsage, err error) {
	orderBy, ok := insightsUserOrderBy[metric]
	if !ok {
		return nil, service.ErrInsightsInvalidMetric
	}
	metricColumn := insightsUserMetricColumn[metric]
	if limit <= 0 {
		limit = 20
	}

	// 先按用户聚合，再只保留普通、启用、未删除的账号：管理员、运维和封禁账号不占名次。
	query := fmt.Sprintf(`
		WITH user_usage AS (
			SELECT
				u.user_id,
				COALESCE(SUM(%s), 0) AS tokens,
				COUNT(*) AS requests,
				COALESCE(SUM(u.actual_cost), 0) AS actual_cost
			FROM usage_logs u
			WHERE u.created_at >= $1 AND u.created_at < $2
			GROUP BY u.user_id
		),
		eligible AS (
			SELECT
				uu.user_id,
				COALESCE(us.username, '') AS username,
				COALESCE(us.email, '') AS email,
				uu.tokens,
				uu.requests,
				uu.actual_cost
			FROM user_usage uu
			JOIN users us ON us.id = uu.user_id
			WHERE us.role = $4
				AND us.status = $5
				AND us.deleted_at IS NULL
				AND uu.%s > 0
		),
		ranked AS (
			SELECT
				user_id, username, email, tokens, requests, actual_cost,
				COUNT(*) OVER () AS participants,
				ROW_NUMBER() OVER (ORDER BY %s) AS position
			FROM eligible
		)
		SELECT user_id, username, email, tokens, requests, actual_cost, participants
		FROM ranked
		WHERE position <= $3
		ORDER BY position
	`, insightsTokensExpr, metricColumn, orderBy)

	rows, err := r.db.QueryContext(ctx, query, start.UTC(), end.UTC(), limit, service.RoleUser, service.StatusActive)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()

	usage := &service.InsightsUserUsage{Rows: []service.InsightsUserUsageRow{}}
	for rows.Next() {
		var row service.InsightsUserUsageRow
		if err = rows.Scan(&row.UserID, &row.Username, &row.Email, &row.Tokens, &row.Requests, &row.ActualCost, &usage.Participants); err != nil {
			return nil, err
		}
		usage.Rows = append(usage.Rows, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return usage, nil
}

func (r *insightsRepository) PublicUsageTotals(ctx context.Context, today time.Time) (*service.InsightsUsageTotals, error) {
	totals := &service.InsightsUsageTotals{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS total_tokens,
			COALESCE(SUM(total_requests), 0) AS total_requests,
			COALESCE(SUM(CASE WHEN bucket_date = $1::date
				THEN input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens
				ELSE 0 END), 0) AS today_tokens
		FROM usage_dashboard_daily
	`, today).Scan(&totals.TotalTokens, &totals.TotalRequests, &totals.TodayTokens)
	if err != nil {
		return nil, err
	}
	return totals, nil
}

func (r *insightsRepository) AggregationWatermark(ctx context.Context) (time.Time, error) {
	var ts time.Time
	err := r.db.QueryRowContext(ctx, `SELECT last_aggregated_at FROM usage_dashboard_aggregation_watermark WHERE id = 1`).Scan(&ts)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Unix(0, 0).UTC(), nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return ts.UTC(), nil
}
