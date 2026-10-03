package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 站内信（fork 本地功能，见 CLAUDE.md「站内信」）。
//
// 点对点发给单个账号（user / operator / admin），不是广播（广播用公告）。三种来源：
//   - security：内容审计风控通知（违规提醒 / 封禁 / cyber 拦截），见 content_moderation_site_message.go；
//   - admin：管理员在用户管理里手动发送；运维发送经审批重放，发件人记为发起的 operator；
//   - system：系统通知（如申诉后账户恢复）。
//
// 这里只放领域类型、仓储端口与错误常量；流程见 site_message_service.go。

const (
	SiteMessageCategorySecurity = "security"
	SiteMessageCategoryAdmin    = "admin"
	SiteMessageCategorySystem   = "system"
)

// 来源类型（source_type）。
const (
	SiteMessageSourceAdmin                = "admin"
	SiteMessageSourceContentModeration    = "content_moderation"
	SiteMessageSourceContentModerationBan = "content_moderation_ban"
	SiteMessageSourceCyberPolicy          = "cyber_policy"
	SiteMessageSourceCyberPolicyBan       = "cyber_policy_ban"
	SiteMessageSourceAppealRestore        = "appeal_restore"
)

const (
	SiteMessageTitleMaxRunes    = 200
	SiteMessageContentMaxRunes  = 5000
	SiteMessageListPageSizeMax  = 100
	siteMessageSourceMaxRunes   = 32
	siteMessageSourceIDMaxRunes = 128
)

// IsValidSiteMessageCategory 报告 v 是否为合法分类。
func IsValidSiteMessageCategory(v string) bool {
	switch v {
	case SiteMessageCategorySecurity, SiteMessageCategoryAdmin, SiteMessageCategorySystem:
		return true
	default:
		return false
	}
}

var (
	ErrSiteMessageNotFound       = infraerrors.NotFound("SITE_MESSAGE_NOT_FOUND", "site message not found")
	ErrSiteMessageTitleInvalid   = infraerrors.BadRequest("SITE_MESSAGE_TITLE_INVALID", "title is required and must be at most 200 characters")
	ErrSiteMessageContentInvalid = infraerrors.BadRequest("SITE_MESSAGE_CONTENT_INVALID", "content is required and must be at most 5000 characters")
	ErrSiteMessageUnavailable    = infraerrors.ServiceUnavailable("SITE_MESSAGE_UNAVAILABLE", "site message service is not available")
)

// SiteMessage 一条站内信。
type SiteMessage struct {
	ID         int64
	UserID     int64
	Category   string
	Title      string
	Content    string
	SourceType string
	SourceID   string
	// SenderUserID 写信的工作人员（经审批时为发起的 operator）；nil 表示系统发出。
	SenderUserID *int64
	SenderRole   string
	// SenderEmail 只在工作人员视图里填（仓储 LEFT JOIN users），用户侧永远不返回。
	SenderEmail string
	ApprovalID  *int64
	ReadAt      *time.Time
	CreatedAt   time.Time
}

// IsRead 报告是否已读。
func (m *SiteMessage) IsRead() bool {
	return m != nil && m.ReadAt != nil
}

// SiteMessageFilter 列表筛选。Category 为空表示全部。
type SiteMessageFilter struct {
	Page       int
	PageSize   int
	UnreadOnly bool
	Category   string
}

// SiteMessageList 分页结果。
type SiteMessageList struct {
	Items    []*SiteMessage
	Total    int64
	Page     int
	PageSize int
}

// SiteMessageSendInput 工作人员发送站内信的输入。
type SiteMessageSendInput struct {
	RecipientUserID int64
	Title           string
	Content         string
	SenderUserID    int64
	SenderRole      string
	// ApprovalID 非 nil 表示这次发送来自运维写操作审批的重放。
	ApprovalID *int64
}

// SystemSiteMessageInput 系统投递（风控通知、账户恢复等）的输入。
type SystemSiteMessageInput struct {
	UserID     int64
	Category   string
	Title      string
	Content    string
	SourceType string
	SourceID   string
}

// SiteMessageRepository 站内信仓储端口（实现在 internal/repository/site_message_repo.go）。
type SiteMessageRepository interface {
	// Create 收件人存在且未被软删除时写入并回填 ID / CreatedAt；否则返回 ErrUserNotFound。
	Create(ctx context.Context, msg *SiteMessage) (*SiteMessage, error)
	// ListByUser 某用户的站内信（按 created_at, id 倒序），工作人员视图带发件人邮箱。
	ListByUser(ctx context.Context, userID int64, filter SiteMessageFilter) ([]*SiteMessage, int64, error)
	// CountUnread 某用户的未读数。
	CountUnread(ctx context.Context, userID int64) (int64, error)
	// MarkRead 标记已读；不是该用户的消息返回 ErrSiteMessageNotFound，已读则幂等成功。
	MarkRead(ctx context.Context, userID, id int64, now time.Time) error
	// MarkAllRead 标记该用户全部未读为已读，返回更新条数。
	MarkAllRead(ctx context.Context, userID int64, now time.Time) (int64, error)
}
