package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// 扣费失败重试队列（fork 本地功能），表见 migrations/248_usage_billing_retries.sql。

const usageBillingRetryErrorMaxLen = 2000

const usageBillingRetryColumns = `id, request_id, api_key_id, user_id, account_id, group_id, platform, actual_cost,
		command, status, attempts, last_error, next_retry_at, settled_at, created_at, updated_at`

type usageBillingRetryRepository struct {
	db *sql.DB
}

func NewUsageBillingRetryRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRetryRepository {
	return &usageBillingRetryRepository{db: sqlDB}
}

func (r *usageBillingRetryRepository) Enqueue(ctx context.Context, item *service.UsageBillingRetry) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("usage billing retry repository db is nil")
	}
	if item == nil || item.Command == nil {
		return false, errors.New("usage billing retry item is nil")
	}
	requestID := strings.TrimSpace(item.RequestID)
	if requestID == "" {
		return false, service.ErrUsageBillingRequestIDRequired
	}
	payload, err := json.Marshal(item.Command)
	if err != nil {
		return false, fmt.Errorf("marshal usage billing command: %w", err)
	}
	status := strings.TrimSpace(item.Status)
	if status == "" {
		status = service.UsageBillingRetryStatusPending
	}
	nextRetryAt := item.NextRetryAt
	if nextRetryAt.IsZero() {
		nextRetryAt = time.Now()
	}
	var groupID sql.NullInt64
	if item.GroupID != nil {
		groupID = sql.NullInt64{Int64: *item.GroupID, Valid: true}
	}
	var id int64
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO usage_billing_retries (
			request_id, api_key_id, user_id, account_id, group_id, platform, actual_cost,
			command, status, attempts, last_error, next_retry_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0, $10, $11, NOW(), NOW())
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`, requestID, item.APIKeyID, item.UserID, item.AccountID, groupID, strings.TrimSpace(item.Platform),
		item.ActualCost, payload, status, truncateUsageBillingRetryError(item.LastError), nextRetryAt).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	item.ID = id
	item.RequestID = requestID
	item.Status = status
	item.NextRetryAt = nextRetryAt
	return true, nil
}

func (r *usageBillingRetryRepository) ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]*service.UsageBillingRetry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing retry repository db is nil")
	}
	if limit <= 0 {
		limit = 100
	}
	if lease <= 0 {
		lease = time.Minute
	}
	rows, err := r.db.QueryContext(ctx, `
		UPDATE usage_billing_retries
		SET attempts = attempts + 1,
			next_retry_at = $2,
			updated_at = NOW()
		WHERE id IN (
			SELECT id FROM usage_billing_retries
			WHERE status = $3 AND next_retry_at <= $1
			ORDER BY next_retry_at, id
			LIMIT $4
			FOR UPDATE SKIP LOCKED
		)
		RETURNING `+usageBillingRetryColumns+`
	`, now, now.Add(lease), service.UsageBillingRetryStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanUsageBillingRetries(rows)
}

func (r *usageBillingRetryRepository) MarkSettled(ctx context.Context, id int64, settledAt time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("usage billing retry repository db is nil")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE usage_billing_retries
		SET status = $2, settled_at = $3, last_error = '', updated_at = NOW()
		WHERE id = $1
	`, id, service.UsageBillingRetryStatusSettled, settledAt)
	return err
}

func (r *usageBillingRetryRepository) MarkRetry(ctx context.Context, id int64, lastError string, nextRetryAt time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("usage billing retry repository db is nil")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE usage_billing_retries
		SET status = $2, last_error = $3, next_retry_at = $4, updated_at = NOW()
		WHERE id = $1
	`, id, service.UsageBillingRetryStatusPending, truncateUsageBillingRetryError(lastError), nextRetryAt)
	return err
}

func (r *usageBillingRetryRepository) MarkFailed(ctx context.Context, id int64, lastError string) error {
	if r == nil || r.db == nil {
		return errors.New("usage billing retry repository db is nil")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE usage_billing_retries
		SET status = $2, last_error = $3, updated_at = NOW()
		WHERE id = $1
	`, id, service.UsageBillingRetryStatusFailed, truncateUsageBillingRetryError(lastError))
	return err
}

func (r *usageBillingRetryRepository) ListByKeys(ctx context.Context, keys []service.UsageBillingRetryKey) ([]*service.UsageBillingRetry, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing retry repository db is nil")
	}
	if len(keys) == 0 {
		return nil, nil
	}
	wanted := make(map[service.UsageBillingRetryKey]struct{}, len(keys))
	requestIDs := make([]string, 0, len(keys))
	for _, key := range keys {
		key.RequestID = strings.TrimSpace(key.RequestID)
		if key.RequestID == "" {
			continue
		}
		if _, dup := wanted[key]; dup {
			continue
		}
		wanted[key] = struct{}{}
		requestIDs = append(requestIDs, key.RequestID)
	}
	if len(requestIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+usageBillingRetryColumns+`
		FROM usage_billing_retries
		WHERE request_id = ANY($1)
	`, pq.Array(requestIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanUsageBillingRetries(rows)
	if err != nil {
		return nil, err
	}
	out := items[:0]
	for _, item := range items {
		if _, ok := wanted[service.UsageBillingRetryKey{RequestID: item.RequestID, APIKeyID: item.APIKeyID}]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *usageBillingRetryRepository) CountCreatedBetween(ctx context.Context, start, end time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("usage billing retry repository db is nil")
	}
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM usage_billing_retries
		WHERE created_at >= $1 AND created_at < $2
	`, start, end).Scan(&n)
	return n, err
}

func (r *usageBillingRetryRepository) Summary(ctx context.Context) (*service.UsageBillingRetrySummary, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing retry repository db is nil")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*), COALESCE(SUM(actual_cost), 0)
		FROM usage_billing_retries
		WHERE status IN ($1, $2)
		GROUP BY status
	`, service.UsageBillingRetryStatusPending, service.UsageBillingRetryStatusFailed)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	summary := &service.UsageBillingRetrySummary{}
	for rows.Next() {
		var status string
		var count int64
		var cost float64
		if err := rows.Scan(&status, &count, &cost); err != nil {
			return nil, err
		}
		switch status {
		case service.UsageBillingRetryStatusPending:
			summary.Pending, summary.PendingCost = count, cost
		case service.UsageBillingRetryStatusFailed:
			summary.Failed, summary.FailedCost = count, cost
		}
	}
	return summary, rows.Err()
}

func scanUsageBillingRetries(rows *sql.Rows) ([]*service.UsageBillingRetry, error) {
	var items []*service.UsageBillingRetry
	for rows.Next() {
		var (
			item        service.UsageBillingRetry
			groupID     sql.NullInt64
			payload     []byte
			settledAt   sql.NullTime
			nextRetryAt time.Time
		)
		if err := rows.Scan(
			&item.ID, &item.RequestID, &item.APIKeyID, &item.UserID, &item.AccountID, &groupID, &item.Platform, &item.ActualCost,
			&payload, &item.Status, &item.Attempts, &item.LastError, &nextRetryAt, &settledAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if groupID.Valid {
			gid := groupID.Int64
			item.GroupID = &gid
		}
		if settledAt.Valid {
			at := settledAt.Time
			item.SettledAt = &at
		}
		item.NextRetryAt = nextRetryAt
		cmd := &service.UsageBillingCommand{}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, cmd); err != nil {
				return nil, fmt.Errorf("decode usage billing command (retry %d): %w", item.ID, err)
			}
		}
		item.Command = cmd
		items = append(items, &item)
	}
	return items, rows.Err()
}

func truncateUsageBillingRetryError(msg string) string {
	msg = strings.TrimSpace(msg)
	runes := []rune(msg)
	if len(runes) <= usageBillingRetryErrorMaxLen {
		return msg
	}
	return string(runes[:usageBillingRetryErrorMaxLen])
}
