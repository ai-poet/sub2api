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

// 站内信仓储（fork 本地功能，原生 SQL，风格同 support_ticket_repo.go）。

const siteMessageColumns = `m.id, m.user_id, m.category, m.title, m.content, m.source_type, m.source_id,
	m.sender_user_id, m.sender_role, COALESCE(s.email, ''), m.approval_id, m.read_at, m.created_at`

type siteMessageRepository struct {
	db *sql.DB
}

// NewSiteMessageRepository 构造站内信仓储。
func NewSiteMessageRepository(db *sql.DB) service.SiteMessageRepository {
	return &siteMessageRepository{db: db}
}

// Create 一条语句完成「收件人存在且未删除才写入」，没有先查后写的竞态。
func (r *siteMessageRepository) Create(ctx context.Context, msg *service.SiteMessage) (*service.SiteMessage, error) {
	if msg == nil {
		return nil, errors.New("site message is nil")
	}
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	out := *msg
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO site_messages (
			user_id, category, title, content, source_type, source_id, sender_user_id, sender_role, approval_id, created_at
		)
		SELECT u.id, $2, $3, $4, $5, $6, $7, $8, $9, $10
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL
		RETURNING id, created_at`,
		msg.UserID, msg.Category, msg.Title, msg.Content, msg.SourceType, msg.SourceID,
		nullInt64Ptr(msg.SenderUserID), msg.SenderRole, nullInt64Ptr(msg.ApprovalID), createdAt,
	).Scan(&out.ID, &out.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *siteMessageRepository) ListByUser(ctx context.Context, userID int64, filter service.SiteMessageFilter) ([]*service.SiteMessage, int64, error) {
	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	where := []string{"m.user_id = $1"}
	args := []any{userID}
	if filter.UnreadOnly {
		where = append(where, "m.read_at IS NULL")
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		args = append(args, category)
		where = append(where, fmt.Sprintf("m.category = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_messages m WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+siteMessageColumns+`
		FROM site_messages m
		LEFT JOIN users s ON s.id = m.sender_user_id
		WHERE `+whereSQL+`
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2),
		limitArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.SiteMessage, 0, pageSize)
	for rows.Next() {
		item, err := scanSiteMessage(rows)
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

func (r *siteMessageRepository) CountUnread(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_messages WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&total)
	return total, err
}

func (r *siteMessageRepository) MarkRead(ctx context.Context, userID, id int64, now time.Time) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE site_messages SET read_at = $3
		WHERE id = $1 AND user_id = $2 AND read_at IS NULL`, id, userID, now)
	if err != nil {
		return err
	}
	if affected, err := res.RowsAffected(); err == nil && affected > 0 {
		return nil
	}
	// 没更新到：要么已经读过（幂等成功），要么不是这个用户的消息
	var exists bool
	if err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM site_messages WHERE id = $1 AND user_id = $2)`, id, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return service.ErrSiteMessageNotFound
	}
	return nil
}

func (r *siteMessageRepository) MarkAllRead(ctx context.Context, userID int64, now time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE site_messages SET read_at = $2
		WHERE user_id = $1 AND read_at IS NULL`, userID, now)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanSiteMessage(row scannable) (*service.SiteMessage, error) {
	var (
		out        service.SiteMessage
		senderID   sql.NullInt64
		approvalID sql.NullInt64
		readAt     sql.NullTime
	)
	if err := row.Scan(
		&out.ID, &out.UserID, &out.Category, &out.Title, &out.Content, &out.SourceType, &out.SourceID,
		&senderID, &out.SenderRole, &out.SenderEmail, &approvalID, &readAt, &out.CreatedAt,
	); err != nil {
		return nil, err
	}
	if senderID.Valid {
		v := senderID.Int64
		out.SenderUserID = &v
	}
	if approvalID.Valid {
		v := approvalID.Int64
		out.ApprovalID = &v
	}
	if readAt.Valid {
		v := readAt.Time
		out.ReadAt = &v
	}
	return &out, nil
}
