package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 运维管理员写操作审批队列的仓储（原生 SQL，风格同 group_status_repo.go）。

const adminApprovalColumns = `id, status, action, method, route_template, request_path, request_query, content_type,
	request_body_enc, request_body_redacted, request_body_sha256,
	target_type, target_id, target_summary,
	requester_user_id, requester_email, requester_ip, request_id,
	decided_by_user_id, decided_by_email, decided_at, decision_reason,
	executed_at, result_status_code, result_body, result_error,
	notified_at, expires_at, created_at, updated_at`

type adminApprovalRepository struct {
	db *sql.DB
}

// NewAdminApprovalRepository 构造审批仓储。
func NewAdminApprovalRepository(db *sql.DB) service.AdminApprovalRepository {
	return &adminApprovalRepository{db: db}
}

func (r *adminApprovalRepository) Create(ctx context.Context, req *service.AdminApprovalRequest) (*service.AdminApprovalRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_approval_requests (
			status, action, method, route_template, request_path, request_query, content_type,
			request_body_enc, request_body_redacted, request_body_sha256,
			target_type, target_id, target_summary,
			requester_user_id, requester_email, requester_ip, request_id,
			expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW())
		RETURNING `+adminApprovalColumns,
		req.Status, req.Action, req.Method, req.RouteTemplate, req.RequestPath, req.RequestQuery, req.ContentType,
		req.RequestBodyEnc, req.RequestBodyRedacted, req.RequestBodySHA256,
		req.TargetType, nullInt64Ptr(req.TargetID), req.TargetSummary,
		req.RequesterUserID, req.RequesterEmail, req.RequesterIP, req.RequestID,
		req.ExpiresAt,
	)
	return scanAdminApproval(row)
}

func (r *adminApprovalRepository) GetByID(ctx context.Context, id int64) (*service.AdminApprovalRequest, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+adminApprovalColumns+` FROM admin_approval_requests WHERE id = $1`, id)
	out, err := scanAdminApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrApprovalNotFound
		}
		return nil, err
	}
	return out, nil
}

func (r *adminApprovalRepository) List(ctx context.Context, filter *service.AdminApprovalFilter) ([]*service.AdminApprovalRequest, int64, error) {
	f := service.AdminApprovalFilter{Page: 1, PageSize: 20}
	if filter != nil {
		f = *filter
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}

	where := []string{"1=1"}
	args := []any{}
	switch strings.TrimSpace(f.Status) {
	case "":
	case service.ApprovalStatusFilterProcessed:
		where = append(where, "status NOT IN ('pending', 'executing')")
	default:
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.RequesterUserID != nil {
		args = append(args, *f.RequesterUserID)
		where = append(where, fmt.Sprintf("requester_user_id = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_approval_requests WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArgs := append(append([]any{}, args...), f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+adminApprovalColumns+`
		FROM admin_approval_requests
		WHERE `+whereSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2),
		limitArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.AdminApprovalRequest, 0, f.PageSize)
	for rows.Next() {
		item, err := scanAdminApproval(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *adminApprovalRepository) CountPending(ctx context.Context, now time.Time, requesterUserID *int64) (int64, error) {
	var total int64
	if requesterUserID != nil {
		err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM admin_approval_requests
			WHERE status = 'pending' AND expires_at > $1 AND requester_user_id = $2`, now, *requesterUserID).Scan(&total)
		return total, err
	}
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM admin_approval_requests
		WHERE status = 'pending' AND expires_at > $1`, now).Scan(&total)
	return total, err
}

func (r *adminApprovalRepository) TransitionToExecuting(ctx context.Context, id, approverID int64, approverEmail string, now time.Time) (*service.AdminApprovalRequest, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE admin_approval_requests
		SET status = 'executing', decided_by_user_id = $2, decided_by_email = $3, decided_at = $4, updated_at = $4
		WHERE id = $1 AND status = 'pending' AND expires_at > $4
		RETURNING `+adminApprovalColumns, id, approverID, approverEmail, now)
	out, err := scanAdminApproval(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 区分"不存在"与"状态不对"，前端据此给不同提示。
			if _, getErr := r.GetByID(ctx, id); getErr != nil {
				return nil, getErr
			}
			return nil, service.ErrApprovalNotPending
		}
		return nil, err
	}
	return out, nil
}

func (r *adminApprovalRepository) FinishExecution(ctx context.Context, id int64, status string, statusCode int, body, errText string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE admin_approval_requests
		SET status = $2, executed_at = $3, result_status_code = $4, result_body = $5, result_error = $6, updated_at = $3
		WHERE id = $1 AND status = 'executing'`, id, status, now, statusCode, body, errText)
	return err
}

func (r *adminApprovalRepository) Decide(ctx context.Context, id int64, toStatus string, actorID int64, actorEmail, reason string, now time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE admin_approval_requests
		SET status = $2, decided_by_user_id = $3, decided_by_email = $4, decision_reason = $5, decided_at = $6, updated_at = $6
		WHERE id = $1 AND status = 'pending' AND expires_at > $6`, id, toStatus, actorID, actorEmail, reason, now)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *adminApprovalRepository) MarkNotified(ctx context.Context, id int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE admin_approval_requests SET notified_at = $2 WHERE id = $1 AND notified_at IS NULL`, id, now)
	return err
}

func (r *adminApprovalRepository) ExpirePending(ctx context.Context, now time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE admin_approval_requests SET status = 'expired', updated_at = $1
		WHERE status = 'pending' AND expires_at <= $1`, now)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *adminApprovalRepository) FailStuckExecuting(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE admin_approval_requests
		SET status = 'failed', result_error = 'execution_timeout', updated_at = NOW()
		WHERE status = 'executing' AND decided_at IS NOT NULL AND decided_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanAdminApproval(row scannable) (*service.AdminApprovalRequest, error) {
	var (
		out              service.AdminApprovalRequest
		targetID         sql.NullInt64
		decidedBy        sql.NullInt64
		decidedAt        sql.NullTime
		executedAt       sql.NullTime
		resultStatusCode sql.NullInt64
		notifiedAt       sql.NullTime
	)
	if err := row.Scan(
		&out.ID, &out.Status, &out.Action, &out.Method, &out.RouteTemplate, &out.RequestPath, &out.RequestQuery, &out.ContentType,
		&out.RequestBodyEnc, &out.RequestBodyRedacted, &out.RequestBodySHA256,
		&out.TargetType, &targetID, &out.TargetSummary,
		&out.RequesterUserID, &out.RequesterEmail, &out.RequesterIP, &out.RequestID,
		&decidedBy, &out.DecidedByEmail, &decidedAt, &out.DecisionReason,
		&executedAt, &resultStatusCode, &out.ResultBody, &out.ResultError,
		&notifiedAt, &out.ExpiresAt, &out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if targetID.Valid {
		v := targetID.Int64
		out.TargetID = &v
	}
	if decidedBy.Valid {
		v := decidedBy.Int64
		out.DecidedByUserID = &v
	}
	if decidedAt.Valid {
		v := decidedAt.Time
		out.DecidedAt = &v
	}
	if executedAt.Valid {
		v := executedAt.Time
		out.ExecutedAt = &v
	}
	if resultStatusCode.Valid {
		v := int(resultStatusCode.Int64)
		out.ResultStatusCode = &v
	}
	if notifiedAt.Valid {
		v := notifiedAt.Time
		out.NotifiedAt = &v
	}
	return &out, nil
}
