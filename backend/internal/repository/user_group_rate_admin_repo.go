package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// userGroupRateAdminRepository 专属倍率用户管理（fork 本地功能）的原生 SQL 实现。
//
// 用户与分组都是软删除，外键 ON DELETE CASCADE 不会触发，表里可能残留已删除用户 / 分组的行，
// 所以每个查询都 join 未删除的 users / groups。只读写 rate_multiplier，rpm_override 原样保留。
type userGroupRateAdminRepository struct {
	db *sql.DB
}

// userGroupRateUsageTimeout 近 30 天用量聚合扫描 usage_logs，给它一个上限，避免拖住连接。
const userGroupRateUsageTimeout = 10 * time.Second

// NewUserGroupRateAdminRepository 创建专属倍率管理仓储。
func NewUserGroupRateAdminRepository(db *sql.DB) service.UserGroupRateAdminRepository {
	return &userGroupRateAdminRepository{db: db}
}

const userGroupRateAdminFrom = `
	FROM user_group_rate_multipliers ugr
	JOIN users u ON u.id = ugr.user_id AND u.deleted_at IS NULL
	JOIN groups g ON g.id = ugr.group_id AND g.deleted_at IS NULL
	WHERE ugr.rate_multiplier IS NOT NULL`

// userGroupRateAdminOrderBy 排序字段白名单 → SQL 表达式；service 已经把非法值归一成 updated_at。
var userGroupRateAdminOrderBy = map[string]string{
	"updated_at":      "ugr.updated_at",
	"rate_multiplier": "ugr.rate_multiplier",
	"discount":        "(ugr.rate_multiplier / NULLIF(g.rate_multiplier, 0))",
	"user_id":         "ugr.user_id",
	"group_name":      "g.name",
}

func buildUserGroupRateAdminWhere(filter service.UserGroupRateListFilter) (string, []any) {
	var where strings.Builder
	args := make([]any, 0, 4)
	if filter.GroupID > 0 {
		args = append(args, filter.GroupID)
		fmt.Fprintf(&where, " AND ugr.group_id = $%d", len(args))
	}
	if filter.UserStatus != "" {
		args = append(args, filter.UserStatus)
		fmt.Fprintf(&where, " AND u.status = $%d", len(args))
	}
	if filter.Search != "" {
		args = append(args, "%"+escapeLikeWildcards(filter.Search)+"%")
		p := len(args)
		fmt.Fprintf(&where, " AND (u.email ILIKE $%d OR u.username ILIKE $%d OR COALESCE(u.notes, '') ILIKE $%d", p, p, p)
		if id, err := strconv.ParseInt(filter.Search, 10, 64); err == nil && id > 0 {
			args = append(args, id)
			fmt.Fprintf(&where, " OR u.id = $%d", len(args))
		}
		where.WriteString(")")
	}
	return where.String(), args
}

func (r *userGroupRateAdminRepository) List(ctx context.Context, filter service.UserGroupRateListFilter) (*service.UserGroupRateList, error) {
	where, args := buildUserGroupRateAdminWhere(filter)
	out := &service.UserGroupRateList{Items: []service.UserGroupRateRow{}, Page: filter.Page, PageSize: filter.PageSize}

	if err := scanSingleRow(ctx, r.db,
		"SELECT COUNT(*), COUNT(DISTINCT ugr.user_id)"+userGroupRateAdminFrom+where,
		args, &out.Total, &out.UserCount); err != nil {
		return nil, err
	}
	if out.Total == 0 {
		return out, nil
	}

	orderExpr, ok := userGroupRateAdminOrderBy[filter.SortBy]
	if !ok {
		orderExpr = userGroupRateAdminOrderBy["updated_at"]
	}
	direction := "DESC NULLS LAST"
	if filter.SortOrder == "asc" {
		direction = "ASC NULLS LAST"
	}
	offset := (filter.Page - 1) * filter.PageSize
	pageArgs := append(append([]any{}, args...), filter.PageSize, offset)
	query := `
		SELECT ugr.user_id, u.email, COALESCE(u.username, ''), COALESCE(u.notes, ''), u.status,
		       ugr.group_id, g.name, g.platform, g.status, g.rate_multiplier,
		       ugr.rate_multiplier, ugr.rpm_override, ugr.updated_at` +
		userGroupRateAdminFrom + where +
		fmt.Sprintf(" ORDER BY %s %s, ugr.user_id ASC, ugr.group_id ASC LIMIT $%d OFFSET $%d",
			orderExpr, direction, len(pageArgs)-1, len(pageArgs))

	rows, err := r.db.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var row service.UserGroupRateRow
		var rpm sql.NullInt32
		if err := rows.Scan(
			&row.UserID, &row.UserEmail, &row.Username, &row.UserNotes, &row.UserStatus,
			&row.GroupID, &row.GroupName, &row.GroupPlatform, &row.GroupStatus, &row.GroupRateMultiplier,
			&row.RateMultiplier, &rpm, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if rpm.Valid {
			v := int(rpm.Int32)
			row.RPMOverride = &v
		}
		out.Items = append(out.Items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userGroupRateAdminRepository) ListGroups(ctx context.Context) ([]service.UserGroupRateGroupOption, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, platform, status, subscription_type, is_exclusive, rate_multiplier
		FROM groups
		WHERE deleted_at IS NULL
		ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.UserGroupRateGroupOption{}
	for rows.Next() {
		var g service.UserGroupRateGroupOption
		if err := rows.Scan(&g.ID, &g.Name, &g.Platform, &g.Status, &g.SubscriptionType, &g.IsExclusive, &g.RateMultiplier); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userGroupRateAdminRepository) Usage(ctx context.Context, userIDs []int64, since time.Time) ([]service.UserGroupRateUsage, error) {
	out := []service.UserGroupRateUsage{}
	if len(userIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, userGroupRateUsageTimeout)
	defer cancel()
	// 走 idx_usage_logs_user_created(user_id, created_at)。
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, group_id, COUNT(*), COALESCE(SUM(actual_cost), 0)
		FROM usage_logs
		WHERE user_id = ANY($1) AND created_at >= $2 AND group_id IS NOT NULL
		GROUP BY user_id, group_id`, pq.Array(userIDs), since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var u service.UserGroupRateUsage
		if err := rows.Scan(&u.UserID, &u.GroupID, &u.Requests, &u.ActualCost); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userGroupRateAdminRepository) ExistingUserIDs(ctx context.Context, ids []int64) ([]int64, error) {
	return r.existingIDs(ctx, `SELECT id FROM users WHERE id = ANY($1) AND deleted_at IS NULL`, ids)
}

func (r *userGroupRateAdminRepository) ExistingGroupIDs(ctx context.Context, ids []int64) ([]int64, error) {
	return r.existingIDs(ctx, `SELECT id FROM groups WHERE id = ANY($1) AND deleted_at IS NULL`, ids)
}

func (r *userGroupRateAdminRepository) existingIDs(ctx context.Context, query string, ids []int64) ([]int64, error) {
	out := []int64{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ApplyBatch 一个事务内：清除的行把 rate_multiplier 置 NULL（rpm_override 也为 NULL 时整行删除），
// 其余行只 upsert rate_multiplier。service 已保证 (user_id, group_id) 不重复。
func (r *userGroupRateAdminRepository) ApplyBatch(ctx context.Context, changes []service.UserGroupRateChange) error {
	var clearUsers, clearGroups, setUsers, setGroups []int64
	var setRates []float64
	for _, c := range changes {
		if c.RateMultiplier == nil {
			clearUsers = append(clearUsers, c.UserID)
			clearGroups = append(clearGroups, c.GroupID)
			continue
		}
		setUsers = append(setUsers, c.UserID)
		setGroups = append(setGroups, c.GroupID)
		setRates = append(setRates, *c.RateMultiplier)
	}
	now := time.Now()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if len(clearUsers) > 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE user_group_rate_multipliers ugr
			SET rate_multiplier = NULL, updated_at = $3
			FROM unnest($1::bigint[], $2::bigint[]) AS data(user_id, group_id)
			WHERE ugr.user_id = data.user_id AND ugr.group_id = data.group_id`,
			pq.Array(clearUsers), pq.Array(clearGroups), now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM user_group_rate_multipliers ugr
			USING unnest($1::bigint[], $2::bigint[]) AS data(user_id, group_id)
			WHERE ugr.user_id = data.user_id AND ugr.group_id = data.group_id
			  AND ugr.rate_multiplier IS NULL AND ugr.rpm_override IS NULL`,
			pq.Array(clearUsers), pq.Array(clearGroups)); err != nil {
			return err
		}
	}
	if len(setUsers) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
			SELECT data.user_id, data.group_id, data.rate_multiplier, $4::timestamptz, $4::timestamptz
			FROM unnest($1::bigint[], $2::bigint[], $3::double precision[]) AS data(user_id, group_id, rate_multiplier)
			ON CONFLICT (user_id, group_id)
			DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier, updated_at = EXCLUDED.updated_at`,
			pq.Array(setUsers), pq.Array(setGroups), pq.Array(setRates), now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
