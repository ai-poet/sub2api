package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 工单系统的响应结构（fork 本地功能）。
// 用户视图不带客服邮箱、不带发起人引用；客服视图（admin / operator 同权）带完整信息。

// TicketMessageAuthorStaff 用户视图里客服消息的归一角色（不区分 admin / operator）。
const TicketMessageAuthorStaff = "staff"

// SupportTicketUserRef 工单发起人引用（客服视图）。
type SupportTicketUserRef struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

// SupportTicket 工单（用户视图）。
type SupportTicket struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Category      string     `json:"category"`
	Status        string     `json:"status"`
	UserUnread    bool       `json:"user_unread"`
	MessageCount  int        `json:"message_count"`
	LastMessageAt time.Time  `json:"last_message_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AdminSupportTicket 工单（客服视图）：多发起人引用与关闭方角色。
type AdminSupportTicket struct {
	SupportTicket
	User         SupportTicketUserRef `json:"user"`
	ClosedByRole string               `json:"closed_by_role,omitempty"`
}

// SupportTicketMessage 工单消息。用户视图里 AuthorRole 归一为 user | staff 且不带客服邮箱。
type SupportTicketMessage struct {
	ID          int64     `json:"id"`
	TicketID    int64     `json:"ticket_id"`
	AuthorRole  string    `json:"author_role"`
	AuthorEmail string    `json:"author_email,omitempty"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}

// SupportTicketDetail 工单 + 全部消息（用户视图）。
type SupportTicketDetail struct {
	Ticket   *SupportTicket         `json:"ticket"`
	Messages []SupportTicketMessage `json:"messages"`
}

// AdminSupportTicketDetail 工单 + 全部消息（客服视图）。
type AdminSupportTicketDetail struct {
	Ticket   *AdminSupportTicket    `json:"ticket"`
	Messages []SupportTicketMessage `json:"messages"`
}

// TicketReplyResponse 用户回复的响应。
type TicketReplyResponse struct {
	Ticket  *SupportTicket        `json:"ticket"`
	Message *SupportTicketMessage `json:"message"`
}

// AdminTicketReplyResponse 客服回复的响应。
type AdminTicketReplyResponse struct {
	Ticket  *AdminSupportTicket   `json:"ticket"`
	Message *SupportTicketMessage `json:"message"`
}

// TicketCountResponse 角标数量。
type TicketCountResponse struct {
	Count int64 `json:"count"`
}

// CreateTicketRequest 创建工单的请求体（service 会再校验一次）。
type CreateTicketRequest struct {
	Title    string `json:"title" binding:"required,max=200"`
	Category string `json:"category" binding:"required,oneof=account billing api other"`
	Body     string `json:"body" binding:"required,max=5000"`
}

// ReplyTicketRequest 回复工单的请求体。
type ReplyTicketRequest struct {
	Body string `json:"body" binding:"required,max=5000"`
}

// SupportTicketFromService 用户视图。
func SupportTicketFromService(t *service.SupportTicket) *SupportTicket {
	if t == nil {
		return nil
	}
	return &SupportTicket{
		ID:            t.ID,
		Title:         t.Title,
		Category:      t.Category,
		Status:        t.Status,
		UserUnread:    t.UserUnread,
		MessageCount:  t.MessageCount,
		LastMessageAt: t.LastMessageAt,
		ClosedAt:      t.ClosedAt,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

// AdminSupportTicketFromService 客服视图。
func AdminSupportTicketFromService(t *service.SupportTicket) *AdminSupportTicket {
	base := SupportTicketFromService(t)
	if base == nil {
		return nil
	}
	return &AdminSupportTicket{
		SupportTicket: *base,
		User:          SupportTicketUserRef{ID: t.UserID, Email: t.UserEmail},
		ClosedByRole:  t.ClosedByRole,
	}
}

// SupportTicketMessageForUser 用户视图：客服（admin / operator）折叠为 staff，且不带客服邮箱。
func SupportTicketMessageForUser(m *service.SupportTicketMessage) SupportTicketMessage {
	if m == nil {
		return SupportTicketMessage{}
	}
	out := SupportTicketMessage{
		ID:        m.ID,
		TicketID:  m.TicketID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
	}
	if service.IsPrivilegedRole(m.AuthorRole) {
		out.AuthorRole = TicketMessageAuthorStaff
	} else {
		out.AuthorRole = service.TicketAuthorRoleUser
	}
	return out
}

// SupportTicketMessageForStaff 客服视图：真实角色 + 作者邮箱。
func SupportTicketMessageForStaff(m *service.SupportTicketMessage) SupportTicketMessage {
	if m == nil {
		return SupportTicketMessage{}
	}
	return SupportTicketMessage{
		ID:          m.ID,
		TicketID:    m.TicketID,
		AuthorRole:  m.AuthorRole,
		AuthorEmail: m.AuthorEmail,
		Body:        m.Body,
		CreatedAt:   m.CreatedAt,
	}
}

// SupportTicketMessagesForUser / ForStaff 批量映射（nil 安全，永远返回非 nil 切片）。
func SupportTicketMessagesForUser(items []*service.SupportTicketMessage) []SupportTicketMessage {
	out := make([]SupportTicketMessage, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, SupportTicketMessageForUser(item))
	}
	return out
}

func SupportTicketMessagesForStaff(items []*service.SupportTicketMessage) []SupportTicketMessage {
	out := make([]SupportTicketMessage, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, SupportTicketMessageForStaff(item))
	}
	return out
}
