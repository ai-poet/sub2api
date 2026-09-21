package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 工单系统（fork 本地功能，见 docs/TICKETS.md）。
//
// 用户提交工单并在同一线程里与客服往返；唯一的管理员与运维管理员（operator）同权看全部并回复。
// 这里只放领域类型、仓储端口与错误常量；流程见 ticket_service.go，推送见 ticket_notify_service.go。

// 工单状态。open（等客服）→ 客服回复 → replied（等用户）→ 用户回复 → open；任一非关闭态可 close；closed 只能 reopen → open。
const (
	TicketStatusOpen    = "open"
	TicketStatusReplied = "replied"
	TicketStatusClosed  = "closed"
)

// 固定分类。
const (
	TicketCategoryAccount = "account"
	TicketCategoryBilling = "billing"
	TicketCategoryAPI     = "api"
	TicketCategoryOther   = "other"
)

// TicketAuthorRoleUser 消息 / 关闭动作来自工单发起方；客服侧存真实角色（RoleAdmin / RoleOperator）。
const TicketAuthorRoleUser = RoleUser

const (
	TicketTitleMaxRunes  = 200
	TicketBodyMaxRunes   = 5000
	TicketSearchMaxRunes = 100
	// TicketOpenLimitPerUser 每个用户同时最多有多少个未关闭工单。
	TicketOpenLimitPerUser = 5
	TicketListPageSizeMax  = 100
)

// TicketCategories 全部合法分类（顺序即前端展示顺序）。
var TicketCategories = []string{TicketCategoryAccount, TicketCategoryBilling, TicketCategoryAPI, TicketCategoryOther}

// IsValidTicketCategory 报告 v 是否为合法分类。
func IsValidTicketCategory(v string) bool {
	for _, c := range TicketCategories {
		if c == v {
			return true
		}
	}
	return false
}

// IsValidTicketStatus 报告 v 是否为合法状态。
func IsValidTicketStatus(v string) bool {
	switch v {
	case TicketStatusOpen, TicketStatusReplied, TicketStatusClosed:
		return true
	default:
		return false
	}
}

var (
	ErrTicketNotFound        = infraerrors.NotFound("TICKET_NOT_FOUND", "ticket not found")
	ErrTicketClosed          = infraerrors.Conflict("TICKET_CLOSED", "ticket is closed")
	ErrTicketNotClosed       = infraerrors.Conflict("TICKET_NOT_CLOSED", "ticket is not closed")
	ErrTicketOpenLimit       = infraerrors.Conflict("TICKET_OPEN_LIMIT", "too many open tickets")
	ErrTicketInvalidCategory = infraerrors.BadRequest("TICKET_INVALID_CATEGORY", "invalid ticket category")
	ErrTicketTitleInvalid    = infraerrors.BadRequest("TICKET_TITLE_INVALID", "title is required and must be at most 200 characters")
	ErrTicketBodyInvalid     = infraerrors.BadRequest("TICKET_BODY_INVALID", "body is required and must be at most 5000 characters")
	ErrTicketForbidden       = infraerrors.Forbidden("TICKET_FORBIDDEN", "not allowed to act on this ticket")
	ErrTicketUnavailable     = infraerrors.ServiceUnavailable("TICKET_UNAVAILABLE", "ticket service is not available")
)

// SupportTicket 一个工单。
type SupportTicket struct {
	ID        int64
	UserID    int64
	UserEmail string
	Title     string
	Category  string
	Status    string
	// UserUnread 客服回复后为 true，用户拉取详情时清零。
	UserUnread    bool
	MessageCount  int
	LastMessageAt time.Time

	ClosedAt       *time.Time
	ClosedByUserID *int64
	ClosedByRole   string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsClosed 报告工单是否已关闭。
func (t *SupportTicket) IsClosed() bool {
	return t != nil && t.Status == TicketStatusClosed
}

// SupportTicketMessage 工单线程里的一条消息。
type SupportTicketMessage struct {
	ID           int64
	TicketID     int64
	AuthorUserID int64
	AuthorEmail  string
	// AuthorRole TicketAuthorRoleUser（发起方）| RoleAdmin | RoleOperator
	AuthorRole string
	Body       string
	CreatedAt  time.Time
}

// TicketActor 当前操作者（从 gin context 组装）。Role 为 RoleUser / RoleAdmin / RoleOperator。
type TicketActor struct {
	UserID int64
	Email  string
	Role   string
}

// IsStaff 报告操作者是否为客服（admin 或 operator）。
func (a TicketActor) IsStaff() bool {
	return IsPrivilegedRole(a.Role)
}

// TicketFilter 列表筛选。Status / Category 为空表示全部；Search 匹配标题与用户邮箱；UserID 非 nil 时只看该用户。
type TicketFilter struct {
	Page     int
	PageSize int
	Status   string
	Category string
	Search   string
	UserID   *int64
}

// TicketList 分页结果。
type TicketList struct {
	Items    []*SupportTicket
	Total    int64
	Page     int
	PageSize int
}

// TicketCreateInput 创建工单的输入。
type TicketCreateInput struct {
	Title    string
	Category string
	Body     string
}

// SupportTicketRepository 工单仓储端口（实现在 internal/repository/support_ticket_repo.go）。
type SupportTicketRepository interface {
	// Create 在一个事务里写工单 + 首条消息（message_count=1）；first 的 ID / TicketID / CreatedAt 会被回填。
	Create(ctx context.Context, ticket *SupportTicket, first *SupportTicketMessage) (*SupportTicket, error)
	// GetByID 不存在时返回 ErrTicketNotFound。
	GetByID(ctx context.Context, id int64) (*SupportTicket, error)
	List(ctx context.Context, filter *TicketFilter) ([]*SupportTicket, int64, error)
	// ListMessages 按 created_at, id 升序返回工单全部消息。
	ListMessages(ctx context.Context, ticketID int64) ([]*SupportTicketMessage, error)
	// AppendMessage 一个事务：UPDATE 工单（status / user_unread / last_message_at / message_count+1，
	// 且 WHERE status <> 'closed'）+ INSERT 消息；工单已关闭返回 ErrTicketClosed，不存在返回 ErrTicketNotFound。
	// msg 的 ID / CreatedAt 会被回填。
	AppendMessage(ctx context.Context, msg *SupportTicketMessage, newStatus string, userUnread bool, now time.Time) (*SupportTicket, error)
	// Transition 原子状态迁移：WHERE id = $1 AND status = ANY(from)；to 为 closed 时写 closed_*，否则清空；返回是否命中。
	Transition(ctx context.Context, id int64, from []string, to string, actor TicketActor, now time.Time) (bool, error)
	// MarkUserRead 清掉用户侧未读标记。
	MarkUserRead(ctx context.Context, id, userID int64) error
	// CountByStatus 全站某状态的工单数（客服角标用 open）。
	CountByStatus(ctx context.Context, status string) (int64, error)
	// CountActiveByUser 某用户未关闭的工单数（配额）。
	CountActiveByUser(ctx context.Context, userID int64) (int64, error)
	// CountUserUnread 某用户有未读客服回复的工单数（用户角标）。
	CountUserUnread(ctx context.Context, userID int64) (int64, error)
	// StaffAttachmentReferencedForUser 报告 key 是否被 userID 拥有的某个工单里的一条客服消息
	// （author_role 不是 user）正文以 ticket-attachment://<key> 引用。只读，是用户侧读取
	// 客服附件的授权依据：只认客服写的消息，用户自己在正文里塞的 key 不算。
	StaffAttachmentReferencedForUser(ctx context.Context, userID int64, key string) (bool, error)
}

// TicketNotifier 新工单 / 用户回复后的推送钩子（Server酱³ 实现见 ticket_notify_service.go）。
// fallbackOrigin 是请求的 scheme://host，站点未配置 frontend_url 时用于拼工单页链接。
type TicketNotifier interface {
	NotifyTicketCreated(ticket *SupportTicket, first *SupportTicketMessage, fallbackOrigin string)
	NotifyTicketUserReplied(ticket *SupportTicket, msg *SupportTicketMessage, fallbackOrigin string)
}
