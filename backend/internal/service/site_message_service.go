package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// SiteMessageService 站内信流程（fork 本地功能）。
//
// 用户侧方法只作用于自己的收件箱，他人的消息一律按不存在处理；
// 工作人员侧方法的权限由路由层（adminAuth + 运维审批）保证，这里只做校验与落库。
type SiteMessageService struct {
	repo SiteMessageRepository
	now  func() time.Time
}

// NewSiteMessageService 构造站内信服务。
func NewSiteMessageService(repo SiteMessageRepository) *SiteMessageService {
	return &SiteMessageService{repo: repo, now: time.Now}
}

func (s *SiteMessageService) ready() error {
	if s == nil || s.repo == nil {
		return ErrSiteMessageUnavailable
	}
	return nil
}

// ---------- 校验 ----------

func normalizeSiteMessageTitle(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" || utf8.RuneCountInString(v) > SiteMessageTitleMaxRunes {
		return "", ErrSiteMessageTitleInvalid
	}
	return v, nil
}

func normalizeSiteMessageContent(v string) (string, error) {
	v = strings.TrimSpace(strings.ReplaceAll(v, "\r\n", "\n"))
	if v == "" || utf8.RuneCountInString(v) > SiteMessageContentMaxRunes {
		return "", ErrSiteMessageContentInvalid
	}
	return v, nil
}

// normalizeSiteMessageFilter 夹紧分页并丢掉非法的 category（当作"全部"）。
func normalizeSiteMessageFilter(filter *SiteMessageFilter) SiteMessageFilter {
	f := SiteMessageFilter{}
	if filter != nil {
		f = *filter
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > SiteMessageListPageSizeMax {
		f.PageSize = SiteMessageListPageSizeMax
	}
	f.Category = strings.ToLower(strings.TrimSpace(f.Category))
	if !IsValidSiteMessageCategory(f.Category) {
		f.Category = ""
	}
	return f
}

// ---------- 用户侧 ----------

// ListMine 当前用户的收件箱。
func (s *SiteMessageService) ListMine(ctx context.Context, userID int64, filter *SiteMessageFilter) (*SiteMessageList, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.list(ctx, userID, filter)
}

// UnreadCount 当前用户的未读数（顶栏角标）。
func (s *SiteMessageService) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if userID <= 0 {
		return 0, nil
	}
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead 标记一条已读；不是自己的消息返回 ErrSiteMessageNotFound。
func (s *SiteMessageService) MarkRead(ctx context.Context, userID, id int64) error {
	if err := s.ready(); err != nil {
		return err
	}
	if userID <= 0 || id <= 0 {
		return ErrSiteMessageNotFound
	}
	return s.repo.MarkRead(ctx, userID, id, s.now())
}

// MarkAllRead 全部标为已读，返回更新条数。
func (s *SiteMessageService) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	if userID <= 0 {
		return 0, nil
	}
	return s.repo.MarkAllRead(ctx, userID, s.now())
}

// ---------- 工作人员侧 ----------

// SendFromStaff 管理员 / 运维（经审批重放）给单个账号发站内信。分类固定为 admin。
func (s *SiteMessageService) SendFromStaff(ctx context.Context, in SiteMessageSendInput) (*SiteMessage, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if in.RecipientUserID <= 0 {
		return nil, ErrUserNotFound
	}
	title, err := normalizeSiteMessageTitle(in.Title)
	if err != nil {
		return nil, err
	}
	content, err := normalizeSiteMessageContent(in.Content)
	if err != nil {
		return nil, err
	}
	msg := &SiteMessage{
		UserID:     in.RecipientUserID,
		Category:   SiteMessageCategoryAdmin,
		Title:      title,
		Content:    content,
		SourceType: SiteMessageSourceAdmin,
		SenderRole: clampRunes(strings.TrimSpace(in.SenderRole), 20),
		CreatedAt:  s.now(),
	}
	if in.SenderUserID > 0 {
		sender := in.SenderUserID
		msg.SenderUserID = &sender
	}
	if in.ApprovalID != nil && *in.ApprovalID > 0 {
		approvalID := *in.ApprovalID
		msg.ApprovalID = &approvalID
		msg.SourceID = fmt.Sprintf("approval:%d", approvalID)
	}
	return s.repo.Create(ctx, msg)
}

// ListForUserAsStaff 工作人员查看某用户收到的站内信（含已读状态与发件人）。
func (s *SiteMessageService) ListForUserAsStaff(ctx context.Context, userID int64, filter *SiteMessageFilter) (*SiteMessageList, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.list(ctx, userID, filter)
}

// ---------- 系统投递 ----------

// DeliverSystemMessage 系统发出的站内信（风控通知、账户恢复等）。收件人已被删除时静默忽略。
// 标题 / 正文由调用方生成，这里只截断到上限而不报错：系统通知不应因为长度丢失。
func (s *SiteMessageService) DeliverSystemMessage(ctx context.Context, in SystemSiteMessageInput) error {
	if err := s.ready(); err != nil {
		return err
	}
	if in.UserID <= 0 {
		return nil
	}
	category := strings.ToLower(strings.TrimSpace(in.Category))
	if !IsValidSiteMessageCategory(category) {
		category = SiteMessageCategorySystem
	}
	title := clampRunes(strings.TrimSpace(in.Title), SiteMessageTitleMaxRunes)
	content := clampRunes(strings.TrimSpace(strings.ReplaceAll(in.Content, "\r\n", "\n")), SiteMessageContentMaxRunes)
	if title == "" || content == "" {
		return nil
	}
	_, err := s.repo.Create(ctx, &SiteMessage{
		UserID:     in.UserID,
		Category:   category,
		Title:      title,
		Content:    content,
		SourceType: clampRunes(strings.TrimSpace(in.SourceType), siteMessageSourceMaxRunes),
		SourceID:   clampRunes(strings.TrimSpace(in.SourceID), siteMessageSourceIDMaxRunes),
		CreatedAt:  s.now(),
	})
	if errors.Is(err, ErrUserNotFound) {
		return nil
	}
	return err
}

func (s *SiteMessageService) list(ctx context.Context, userID int64, filter *SiteMessageFilter) (*SiteMessageList, error) {
	f := normalizeSiteMessageFilter(filter)
	if userID <= 0 {
		return &SiteMessageList{Items: []*SiteMessage{}, Page: f.Page, PageSize: f.PageSize}, nil
	}
	items, total, err := s.repo.ListByUser(ctx, userID, f)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*SiteMessage{}
	}
	return &SiteMessageList{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}
