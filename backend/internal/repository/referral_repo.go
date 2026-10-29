package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type referralRepository struct{ client *dbent.Client }

type affiliateSingleRow struct {
	rows *sql.Rows
	err  error
}

func (r *affiliateSingleRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	defer r.rows.Close()
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return r.rows.Scan(dest...)
}

func affiliateQueryRow(ctx context.Context, client *dbent.Client, query string, args ...any) *affiliateSingleRow {
	rows, err := client.QueryContext(ctx, query, args...)
	return &affiliateSingleRow{rows: rows, err: err}
}

func NewReferralRepository(client *dbent.Client) service.AffiliateRepository {
	return &referralRepository{client: client}
}

func (r *referralRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}
	if dbent.TxFromContext(ctx) != nil {
		return fn(ctx)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		if errors.Is(err, dbent.ErrTxStarted) {
			return fn(ctx)
		}
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(dbent.NewTxContext(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit()
}

const affiliateProfileSelect = `
SELECT ua.user_id, ua.aff_code, ua.inviter_id, ua.aff_count,
       ua.aff_quota::float8, ua.aff_frozen_quota::float8, ua.aff_history_quota::float8,
       ua.aff_rebate_rate_percent::float8, ua.aff_code_custom, ua.created_at, ua.updated_at,
       COALESCE(u.email, ''), COALESCE(u.username, '')
FROM user_affiliates ua JOIN users u ON u.id = ua.user_id`

func scanAffiliateProfile(row interface{ Scan(...any) error }) (*service.AffiliateSummary, error) {
	var p service.AffiliateSummary
	if err := row.Scan(&p.UserID, &p.AffCode, &p.InviterID, &p.AffCount, &p.AffQuota, &p.AffFrozenQuota, &p.AffHistoryQuota, &p.AffRebateRatePercent, &p.AffCodeCustom, &p.CreatedAt, &p.UpdatedAt, &p.Email, &p.Username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAffiliateNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *referralRepository) getByUserID(ctx context.Context, userID int64) (*service.AffiliateSummary, error) {
	return scanAffiliateProfile(affiliateQueryRow(ctx, clientFromContext(ctx, r.client), affiliateProfileSelect+" WHERE ua.user_id=$1", userID))
}

func (r *referralRepository) EnsureUserAffiliate(ctx context.Context, userID int64) (*service.AffiliateSummary, error) {
	if profile, err := r.getByUserID(ctx, userID); err == nil {
		return profile, nil
	} else if !errors.Is(err, service.ErrAffiliateNotFound) {
		return nil, err
	}
	for attempt := 0; attempt < 12; attempt++ {
		code, err := service.NewAffiliateCode()
		if err != nil {
			return nil, err
		}
		_, err = clientFromContext(ctx, r.client).ExecContext(ctx, `
INSERT INTO user_affiliates (user_id, aff_code)
SELECT id, $2 FROM users WHERE id=$1 AND deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING`, userID, code)
		if err != nil {
			if isUniqueConstraintViolation(err) {
				continue
			}
			return nil, err
		}
		profile, err := r.getByUserID(ctx, userID)
		if err == nil {
			return profile, nil
		}
		if !errors.Is(err, service.ErrAffiliateNotFound) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("generate unique affiliate code after retries")
}

func (r *referralRepository) GetAffiliateByCode(ctx context.Context, code string) (*service.AffiliateSummary, error) {
	return scanAffiliateProfile(affiliateQueryRow(ctx, clientFromContext(ctx, r.client), affiliateProfileSelect+" WHERE ua.aff_code=$1 AND u.deleted_at IS NULL", code))
}

func (r *referralRepository) BindInviter(ctx context.Context, userID, inviterID int64) (bool, error) {
	bound := false
	err := r.RunInTx(ctx, func(txCtx context.Context) error {
		client := clientFromContext(txCtx, r.client)
		result, err := client.ExecContext(txCtx, `UPDATE user_affiliates SET inviter_id=$1, updated_at=NOW() WHERE user_id=$2 AND inviter_id IS NULL`, inviterID, userID)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		if _, err := client.ExecContext(txCtx, `UPDATE user_affiliates SET aff_count=aff_count+1, updated_at=NOW() WHERE user_id=$1`, inviterID); err != nil {
			return err
		}
		bound = true
		return nil
	})
	return bound, err
}

func (r *referralRepository) WithLockedInviter(ctx context.Context, inviterID int64, fn func(context.Context) error) error {
	return r.RunInTx(ctx, func(txCtx context.Context) error {
		var id int64
		err := affiliateQueryRow(txCtx, clientFromContext(txCtx, r.client), `SELECT user_id FROM user_affiliates WHERE user_id=$1 FOR UPDATE`, inviterID).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrAffiliateNotFound
		}
		if err != nil {
			return err
		}
		return fn(txCtx)
	})
}

func (r *referralRepository) GetAccruedRebateFromInvitee(ctx context.Context, inviterID, inviteeID int64) (float64, error) {
	var amount float64
	err := affiliateQueryRow(ctx, clientFromContext(ctx, r.client), `SELECT COALESCE(SUM(amount),0)::float8 FROM user_affiliate_ledger WHERE user_id=$1 AND source_user_id=$2 AND action='accrue'`, inviterID, inviteeID).Scan(&amount)
	return amount, err
}

func (r *referralRepository) AccrueQuota(ctx context.Context, inviterID, inviteeID int64, amount float64, freezeHours int, sourceOrderRef *string) (bool, error) {
	client := clientFromContext(ctx, r.client)
	if sourceOrderRef != nil {
		var exists bool
		if err := affiliateQueryRow(ctx, client, `SELECT EXISTS(SELECT 1 FROM user_affiliate_ledger WHERE action='accrue' AND source_order_ref=$1)`, *sourceOrderRef).Scan(&exists); err != nil {
			return false, err
		}
		if exists {
			return false, nil
		}
	}
	quotaColumn := "aff_quota"
	var frozenUntil any
	if freezeHours > 0 {
		quotaColumn = "aff_frozen_quota"
		frozenUntil = time.Now().Add(time.Duration(freezeHours) * time.Hour)
	}
	// quotaColumn is selected exclusively from the two constants above.
	query := fmt.Sprintf(`
WITH updated AS (
    UPDATE user_affiliates SET %s = %s + $1, aff_history_quota=aff_history_quota+$1, updated_at=NOW()
    WHERE user_id=$2
    RETURNING aff_quota::float8, aff_frozen_quota::float8, aff_history_quota::float8
)
INSERT INTO user_affiliate_ledger
    (user_id, action, amount, source_user_id, source_order_ref, frozen_until,
     aff_quota_after, aff_frozen_quota_after, aff_history_quota_after)
SELECT $2, 'accrue', $1, $3, $4, $5, updated.aff_quota, updated.aff_frozen_quota, updated.aff_history_quota
FROM updated RETURNING id`, quotaColumn, quotaColumn)
	var id int64
	if err := affiliateQueryRow(ctx, client, query, amount, inviterID, inviteeID, sourceOrderRef, frozenUntil).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, service.ErrAffiliateNotFound
		}
		return false, err
	}
	return true, nil
}

func (r *referralRepository) ThawFrozenQuota(ctx context.Context, userID int64) (float64, error) {
	thawed := 0.0
	err := r.RunInTx(ctx, func(txCtx context.Context) error {
		client := clientFromContext(txCtx, r.client)
		var locked int64
		if err := affiliateQueryRow(txCtx, client, `SELECT user_id FROM user_affiliates WHERE user_id=$1 FOR UPDATE`, userID).Scan(&locked); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if err := affiliateQueryRow(txCtx, client, `SELECT COALESCE(SUM(amount),0)::float8 FROM user_affiliate_ledger WHERE user_id=$1 AND action='accrue' AND frozen_until IS NOT NULL AND frozen_until<=NOW()`, userID).Scan(&thawed); err != nil {
			return err
		}
		if thawed <= 0 {
			return nil
		}
		if _, err := client.ExecContext(txCtx, `UPDATE user_affiliates SET aff_quota=aff_quota+$1, aff_frozen_quota=aff_frozen_quota-$1, updated_at=NOW() WHERE user_id=$2 AND aff_frozen_quota>=$1`, thawed, userID); err != nil {
			return err
		}
		_, err := client.ExecContext(txCtx, `UPDATE user_affiliate_ledger SET frozen_until=NULL, updated_at=NOW() WHERE user_id=$1 AND action='accrue' AND frozen_until IS NOT NULL AND frozen_until<=NOW()`, userID)
		return err
	})
	return thawed, err
}

const affiliateLedgerSelect = `
SELECT l.id, l.user_id, l.action, l.amount::float8, l.source_user_id, l.source_order_ref,
       l.frozen_until, l.balance_after::float8, l.aff_quota_after::float8,
       l.aff_frozen_quota_after::float8, l.aff_history_quota_after::float8, l.created_at,
       COALESCE(u.email,''), COALESCE(u.username,''), COALESCE(su.email,''), COALESCE(su.username,'')
FROM user_affiliate_ledger l
JOIN users u ON u.id=l.user_id
LEFT JOIN users su ON su.id=l.source_user_id`

func scanAffiliateLedger(row interface{ Scan(...any) error }) (*service.AffiliateLedgerEntry, error) {
	var e service.AffiliateLedgerEntry
	if err := row.Scan(&e.ID, &e.UserID, &e.Action, &e.Amount, &e.SourceUserID, &e.SourceOrderRef, &e.FrozenUntil, &e.BalanceAfter, &e.AffQuotaAfter, &e.AffFrozenQuotaAfter, &e.AffHistoryQuotaAfter, &e.CreatedAt, &e.Email, &e.Username, &e.SourceEmail, &e.SourceUsername); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *referralRepository) TransferQuotaToBalance(ctx context.Context, userID int64) (*service.AffiliateLedgerEntry, error) {
	var id int64
	err := r.RunInTx(ctx, func(txCtx context.Context) error {
		client := clientFromContext(txCtx, r.client)
		err := affiliateQueryRow(txCtx, client, `
WITH claimed AS (
    SELECT aff_quota::float8 AS amount FROM user_affiliates WHERE user_id=$1 AND aff_quota>0 FOR UPDATE
), cleared AS (
    UPDATE user_affiliates ua SET aff_quota=0, updated_at=NOW() FROM claimed c WHERE ua.user_id=$1
    RETURNING c.amount, ua.aff_frozen_quota::float8, ua.aff_history_quota::float8
), credited AS (
    UPDATE users u SET balance=balance+c.amount, total_recharged=total_recharged+c.amount, updated_at=NOW()
    FROM cleared c WHERE u.id=$1 AND u.deleted_at IS NULL RETURNING u.balance::float8, c.amount, c.aff_frozen_quota, c.aff_history_quota
)
INSERT INTO user_affiliate_ledger
    (user_id, action, amount, balance_after, aff_quota_after, aff_frozen_quota_after, aff_history_quota_after)
SELECT $1, 'transfer', amount, balance, 0, aff_frozen_quota, aff_history_quota FROM credited RETURNING id`, userID).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrAffiliateQuotaEmpty
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	entry, err := scanAffiliateLedger(affiliateQueryRow(ctx, clientFromContext(ctx, r.client), affiliateLedgerSelect+" WHERE l.id=$1", id))
	return entry, err
}

func maskEmail(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return "***"
	}
	visible := 2
	if at <= 2 {
		visible = 1
	}
	return email[:visible] + "***" + email[at:]
}

func (r *referralRepository) ListInvitees(ctx context.Context, inviterID int64, limit int) ([]service.AffiliateInvitee, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := clientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT ua.user_id, COALESCE(u.email,''), COALESCE(u.username,''), ua.created_at,
       COALESCE((SELECT SUM(l.amount) FROM user_affiliate_ledger l WHERE l.user_id=$1 AND l.source_user_id=ua.user_id AND l.action='accrue'),0)::float8
FROM user_affiliates ua JOIN users u ON u.id=ua.user_id
WHERE ua.inviter_id=$1 ORDER BY ua.created_at DESC LIMIT $2`, inviterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.AffiliateInvitee, 0)
	for rows.Next() {
		var item service.AffiliateInvitee
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &item.CreatedAt, &item.TotalRebate); err != nil {
			return nil, err
		}
		item.Email = maskEmail(item.Email)
		items = append(items, item)
	}
	return items, rows.Err()
}

func affiliateFilterWhere(filter service.AffiliateRecordFilter, alias string) (string, []any) {
	where := make([]string, 0, 5)
	args := make([]any, 0, 5)
	add := func(condition string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if filter.UserID > 0 {
		add(alias+".user_id=$%d", filter.UserID)
	}
	if filter.Action != "" {
		add(alias+".action=$%d", filter.Action)
	}
	if filter.StartAt != nil {
		add(alias+".created_at>=$%d", *filter.StartAt)
	}
	if filter.EndAt != nil {
		add(alias+".created_at<$%d", *filter.EndAt)
	}
	if len(where) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

func (r *referralRepository) ListLedger(ctx context.Context, filter service.AffiliateRecordFilter) ([]service.AffiliateLedgerEntry, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: filter.Page, PageSize: filter.PageSize}
	where, args := affiliateFilterWhere(filter, "l")
	client := clientFromContext(ctx, r.client)
	var total int64
	if err := affiliateQueryRow(ctx, client, "SELECT COUNT(*) FROM user_affiliate_ledger l"+where, args...).Scan(&total); err != nil {
		return nil, nil, err
	}
	offset := params.Offset()
	args = append(args, params.Limit(), offset)
	rows, err := client.QueryContext(ctx, affiliateLedgerSelect+where+fmt.Sprintf(" ORDER BY l.created_at DESC, l.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := make([]service.AffiliateLedgerEntry, 0)
	for rows.Next() {
		entry, err := scanAffiliateLedger(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, *entry)
	}
	return items, paginationResultFromTotal(total, params), rows.Err()
}

func (r *referralRepository) ListRelations(ctx context.Context, filter service.AffiliateRecordFilter) ([]service.AffiliateSummary, *pagination.PaginationResult, error) {
	params := pagination.PaginationParams{Page: filter.Page, PageSize: filter.PageSize}
	where := " WHERE ua.inviter_id IS NOT NULL"
	args := make([]any, 0, 4)
	if filter.Search != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%")
		where += fmt.Sprintf(" AND (LOWER(u.email) LIKE $%d OR LOWER(u.username) LIKE $%d OR LOWER(iu.email) LIKE $%d OR ua.aff_code ILIKE $%d)", len(args), len(args), len(args), len(args))
	}
	client := clientFromContext(ctx, r.client)
	var total int64
	countQuery := "SELECT COUNT(*) FROM user_affiliates ua JOIN users u ON u.id=ua.user_id JOIN users iu ON iu.id=ua.inviter_id" + where
	if err := affiliateQueryRow(ctx, client, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, err
	}
	args = append(args, params.Limit(), params.Offset())
	query := affiliateProfileSelect + " JOIN users iu ON iu.id=ua.inviter_id" + where + fmt.Sprintf(" ORDER BY ua.created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := make([]service.AffiliateSummary, 0)
	for rows.Next() {
		// The base profile select has twelve columns; read inviter identity separately below.
		var p service.AffiliateSummary
		if err := rows.Scan(&p.UserID, &p.AffCode, &p.InviterID, &p.AffCount, &p.AffQuota, &p.AffFrozenQuota, &p.AffHistoryQuota, &p.AffRebateRatePercent, &p.AffCodeCustom, &p.CreatedAt, &p.UpdatedAt, &p.Email, &p.Username); err != nil {
			return nil, nil, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	// Fill inviter labels in one batch; relation lists are at most one page.
	for i := range items {
		if items[i].InviterID == nil {
			continue
		}
		_ = affiliateQueryRow(ctx, client, `SELECT COALESCE(email,''),COALESCE(username,'') FROM users WHERE id=$1`, *items[i].InviterID).Scan(&items[i].InviterEmail, &items[i].InviterUsername)
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *referralRepository) UpdateUserAffCode(ctx context.Context, userID int64, code string) error {
	_, err := clientFromContext(ctx, r.client).ExecContext(ctx, `UPDATE user_affiliates SET aff_code=$1, aff_code_custom=TRUE, updated_at=NOW() WHERE user_id=$2`, code, userID)
	if isUniqueConstraintViolation(err) {
		return service.ErrAffiliateCodeTaken
	}
	return err
}

func (r *referralRepository) ResetUserAffCode(ctx context.Context, userID int64) (string, error) {
	for attempt := 0; attempt < 12; attempt++ {
		code, err := service.NewAffiliateCode()
		if err != nil {
			return "", err
		}
		_, err = clientFromContext(ctx, r.client).ExecContext(ctx, `UPDATE user_affiliates SET aff_code=$1, aff_code_custom=FALSE, updated_at=NOW() WHERE user_id=$2`, code, userID)
		if isUniqueConstraintViolation(err) {
			continue
		}
		return code, err
	}
	return "", fmt.Errorf("generate unique affiliate code after retries")
}

func (r *referralRepository) SetUserRebateRate(ctx context.Context, userID int64, rate *float64) error {
	_, err := clientFromContext(ctx, r.client).ExecContext(ctx, `UPDATE user_affiliates SET aff_rebate_rate_percent=$1, updated_at=NOW() WHERE user_id=$2`, rate, userID)
	return err
}
