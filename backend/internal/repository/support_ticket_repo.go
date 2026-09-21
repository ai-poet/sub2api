package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/lib/pq"
)

// 工单仓储（fork 本地功能，原生 SQL，风格同 admin_approval_repo.go）。

const supportTicketColumns = `id, user_id, user_email, title, category, status, user_unread, message_count, last_message_at,
	closed_at, closed_by_user_id, closed_by_role, created_at, updated_at`

const supportTicketMessageColumns = `id, ticket_id, author_user_id, author_email, author_role, body, created_at`

type supportTicketRepository struct {
	db *sql.DB
}

// NewSupportTicketRepository 构造工单仓储。
func NewSupportTicketRepository(db *sql.DB) service.SupportTicketRepository {
	return &supportTicketRepository{db: db}
}

func (r *supportTicketRepository) Create(ctx context.Context, ticket *service.SupportTicket, first *service.SupportTicketMessage) (*service.SupportTicket, error) {
	if ticket == nil {
		return nil, errors.New("ticket is nil")
	}
	now := ticket.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}
	lastMessageAt := ticket.LastMessageAt
	if lastMessageAt.IsZero() {
		lastMessageAt = now
	}
	status := ticket.Status
	if status == "" {
		status = service.TicketStatusOpen
	}
	messageCount := 0
	if first != nil {
		messageCount = 1
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
		INSERT INTO support_tickets (
			user_id, user_email, title, category, status, user_unread, message_count, last_message_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING `+supportTicketColumns,
		ticket.UserID, ticket.UserEmail, ticket.Title, ticket.Category, status, ticket.UserUnread, messageCount, lastMessageAt, now,
	)
	created, err := scanSupportTicket(row)
	if err != nil {
		return nil, err
	}
	if first != nil {
		createdAt := first.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		mrow := tx.QueryRowContext(ctx, `
			INSERT INTO support_ticket_messages (ticket_id, author_user_id, author_email, author_role, body, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at`,
			created.ID, first.AuthorUserID, first.AuthorEmail, first.AuthorRole, first.Body, createdAt,
		)
		if err := mrow.Scan(&first.ID, &first.CreatedAt); err != nil {
			return nil, err
		}
		first.TicketID = created.ID
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *supportTicketRepository) GetByID(ctx context.Context, id int64) (*service.SupportTicket, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+supportTicketColumns+` FROM support_tickets WHERE id = $1`, id)
	out, err := scanSupportTicket(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrTicketNotFound
		}
		return nil, err
	}
	return out, nil
}

func (r *supportTicketRepository) List(ctx context.Context, filter *service.TicketFilter) ([]*service.SupportTicket, int64, error) {
	f := service.TicketFilter{Page: 1, PageSize: 20}
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
	if f.UserID != nil {
		args = append(args, *f.UserID)
		where = append(where, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if category := strings.TrimSpace(f.Category); category != "" {
		args = append(args, category)
		where = append(where, fmt.Sprintf("category = $%d", len(args)))
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		args = append(args, "%"+escapeSupportTicketLike(search)+"%")
		where = append(where, fmt.Sprintf(`(title ILIKE $%d ESCAPE '\' OR user_email ILIKE $%d ESCAPE '\')`, len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM support_tickets WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArgs := append(append([]any{}, args...), f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+supportTicketColumns+`
		FROM support_tickets
		WHERE `+whereSQL+`
		ORDER BY last_message_at DESC, id DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2),
		limitArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.SupportTicket, 0, f.PageSize)
	for rows.Next() {
		item, err := scanSupportTicket(rows)
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

func (r *supportTicketRepository) ListMessages(ctx context.Context, ticketID int64) ([]*service.SupportTicketMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+supportTicketMessageColumns+`
		FROM support_ticket_messages
		WHERE ticket_id = $1
		ORDER BY created_at ASC, id ASC`, ticketID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.SupportTicketMessage, 0, 8)
	for rows.Next() {
		item, err := scanSupportTicketMessage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *supportTicketRepository) AppendMessage(ctx context.Context, msg *service.SupportTicketMessage, newStatus string, userUnread bool, now time.Time) (*service.SupportTicket, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}
	if now.IsZero() {
		now = time.Now()
	}
	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
		UPDATE support_tickets
		SET status = $2, user_unread = $3, last_message_at = $4, message_count = message_count + 1, updated_at = $4
		WHERE id = $1 AND status <> 'closed'
		RETURNING `+supportTicketColumns, msg.TicketID, newStatus, userUnread, now)
	updated, err := scanSupportTicket(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			// 区分"不存在"与"已关闭"，前端据此给不同提示。
			if _, getErr := r.GetByID(ctx, msg.TicketID); getErr != nil {
				return nil, getErr
			}
			return nil, service.ErrTicketClosed
		}
		return nil, err
	}

	mrow := tx.QueryRowContext(ctx, `
		INSERT INTO support_ticket_messages (ticket_id, author_user_id, author_email, author_role, body, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		msg.TicketID, msg.AuthorUserID, msg.AuthorEmail, msg.AuthorRole, msg.Body, createdAt,
	)
	if err := mrow.Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *supportTicketRepository) Transition(ctx context.Context, id int64, from []string, to string, actor service.TicketActor, now time.Time) (bool, error) {
	if len(from) == 0 || strings.TrimSpace(to) == "" {
		return false, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	var (
		closedAt   any
		closedBy   any
		closedRole string
	)
	if to == service.TicketStatusClosed {
		closedAt = now
		if actor.UserID > 0 {
			closedBy = actor.UserID
		}
		closedRole = actor.Role
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE support_tickets
		SET status = $2, closed_at = $3, closed_by_user_id = $4, closed_by_role = $5, updated_at = $6
		WHERE id = $1 AND status = ANY($7)`,
		id, to, closedAt, closedBy, closedRole, now, pq.Array(from))
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *supportTicketRepository) MarkUserRead(ctx context.Context, id, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE support_tickets SET user_unread = FALSE
		WHERE id = $1 AND user_id = $2 AND user_unread`, id, userID)
	return err
}

func (r *supportTicketRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM support_tickets WHERE status = $1`, status).Scan(&total)
	return total, err
}

func (r *supportTicketRepository) CountActiveByUser(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM support_tickets WHERE user_id = $1 AND status <> 'closed'`, userID).Scan(&total)
	return total, err
}

func (r *supportTicketRepository) CountUserUnread(ctx context.Context, userID int64) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM support_tickets WHERE user_id = $1 AND user_unread`, userID).Scan(&total)
	return total, err
}

// StaffAttachmentReferencedForUser 用 strpos 做子串匹配而不是 LIKE，key 里的 '_' / '%' 无需转义。
// 客服生成的 key 固定为 <prefix>staff/<uid>/<yyyymm>/<rand8hex><ext>，不存在一个 key 是另一个 key 前缀的情况。
func (r *supportTicketRepository) StaffAttachmentReferencedForUser(ctx context.Context, userID int64, key string) (bool, error) {
	var referenced bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM support_ticket_messages m
			JOIN support_tickets t ON t.id = m.ticket_id
			WHERE t.user_id = $1 AND m.author_role <> $2 AND strpos(m.body, $3) > 0
		)`, userID, service.TicketAuthorRoleUser, service.TicketAttachmentURLScheme+key).Scan(&referenced)
	return referenced, err
}

// escapeSupportTicketLike 转义 LIKE 通配符（配合 ESCAPE '\'）。
func escapeSupportTicketLike(v string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(v)
}

func scanSupportTicket(row scannable) (*service.SupportTicket, error) {
	var (
		out      service.SupportTicket
		closedAt sql.NullTime
		closedBy sql.NullInt64
	)
	if err := row.Scan(
		&out.ID, &out.UserID, &out.UserEmail, &out.Title, &out.Category, &out.Status, &out.UserUnread, &out.MessageCount, &out.LastMessageAt,
		&closedAt, &closedBy, &out.ClosedByRole, &out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if closedAt.Valid {
		v := closedAt.Time
		out.ClosedAt = &v
	}
	if closedBy.Valid {
		v := closedBy.Int64
		out.ClosedByUserID = &v
	}
	return &out, nil
}

func scanSupportTicketMessage(row scannable) (*service.SupportTicketMessage, error) {
	var out service.SupportTicketMessage
	if err := row.Scan(
		&out.ID, &out.TicketID, &out.AuthorUserID, &out.AuthorEmail, &out.AuthorRole, &out.Body, &out.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}
