package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 站内信的请求 / 响应结构（fork 本地功能）。
// 用户视图不带发件人身份（只区分 system / staff）；工作人员视图带发件人、来源与审批单号。

const (
	// SiteMessageFromSystem 系统发出（风控通知、账户恢复等）。
	SiteMessageFromSystem = "system"
	// SiteMessageFromStaff 管理员 / 运维发出（用户侧不区分具体是谁）。
	SiteMessageFromStaff = "staff"
)

// SendSiteMessageRequest 管理员 / 运维给单个账号发站内信。
// 不加 binding 标签：长度与非空由 service 校验，返回具体的 SITE_MESSAGE_* 错误码。
type SendSiteMessageRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// SiteMessage 站内信（用户视图）。
type SiteMessage struct {
	ID        int64      `json:"id"`
	Category  string     `json:"category"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	From      string     `json:"from"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// AdminSiteMessage 站内信（工作人员视图）。
type AdminSiteMessage struct {
	SiteMessage
	UserID       int64  `json:"user_id"`
	SourceType   string `json:"source_type"`
	SourceID     string `json:"source_id,omitempty"`
	SenderUserID *int64 `json:"sender_user_id,omitempty"`
	SenderRole   string `json:"sender_role,omitempty"`
	SenderEmail  string `json:"sender_email,omitempty"`
	ApprovalID   *int64 `json:"approval_id,omitempty"`
}

// SiteMessageCountResponse 未读数。
type SiteMessageCountResponse struct {
	Count int64 `json:"count"`
}

// SiteMessageMarkAllResponse 全部已读的更新条数。
type SiteMessageMarkAllResponse struct {
	Updated int64 `json:"updated"`
}

// SiteMessageFromService 用户视图。
func SiteMessageFromService(m *service.SiteMessage) *SiteMessage {
	if m == nil {
		return nil
	}
	from := SiteMessageFromSystem
	if m.SenderUserID != nil {
		from = SiteMessageFromStaff
	}
	return &SiteMessage{
		ID:        m.ID,
		Category:  m.Category,
		Title:     m.Title,
		Content:   m.Content,
		From:      from,
		ReadAt:    m.ReadAt,
		CreatedAt: m.CreatedAt,
	}
}

// AdminSiteMessageFromService 工作人员视图。
func AdminSiteMessageFromService(m *service.SiteMessage) *AdminSiteMessage {
	base := SiteMessageFromService(m)
	if base == nil {
		return nil
	}
	return &AdminSiteMessage{
		SiteMessage:  *base,
		UserID:       m.UserID,
		SourceType:   m.SourceType,
		SourceID:     m.SourceID,
		SenderUserID: m.SenderUserID,
		SenderRole:   m.SenderRole,
		SenderEmail:  m.SenderEmail,
		ApprovalID:   m.ApprovalID,
	}
}
