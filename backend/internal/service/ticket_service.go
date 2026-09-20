package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// TicketService 工单流程（fork 本地功能）。
//
// 用户侧方法只允许操作自己的工单，他人的工单一律按不存在处理（不泄露存在性）；
// 客服侧方法要求 actor 为 admin / operator，两者同权。状态迁移全部在仓储的 WHERE 里原子完成。
type TicketService struct {
	repo SupportTicketRepository

	mu       sync.RWMutex
	notifier TicketNotifier
	now      func() time.Time
}

// NewTicketService 构造工单服务。
func NewTicketService(repo SupportTicketRepository) *TicketService {
	return &TicketService{repo: repo, now: time.Now}
}

// SetNotifier 挂上推送钩子（Server酱³）。
func (s *TicketService) SetNotifier(n TicketNotifier) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifier = n
}

func (s *TicketService) getNotifier() TicketNotifier {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.notifier
}

func (s *TicketService) ready() error {
	if s == nil || s.repo == nil {
		return ErrTicketUnavailable
	}
	return nil
}

// ---------- 校验 ----------

func normalizeTicketTitle(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" || utf8.RuneCountInString(v) > TicketTitleMaxRunes {
		return "", ErrTicketTitleInvalid
	}
	return v, nil
}

func normalizeTicketBody(v string) (string, error) {
	v = strings.TrimSpace(strings.ReplaceAll(v, "\r\n", "\n"))
	if v == "" || utf8.RuneCountInString(v) > TicketBodyMaxRunes {
		return "", ErrTicketBodyInvalid
	}
	return v, nil
}

func normalizeTicketCategory(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if !IsValidTicketCategory(v) {
		return "", ErrTicketInvalidCategory
	}
	return v, nil
}

// normalizeTicketFilter 夹紧分页并丢掉非法的 status / category（当作"全部"）。
func normalizeTicketFilter(filter *TicketFilter) TicketFilter {
	f := TicketFilter{}
	if filter != nil {
		f = *filter
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > TicketListPageSizeMax {
		f.PageSize = TicketListPageSizeMax
	}
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	if !IsValidTicketStatus(f.Status) {
		f.Status = ""
	}
	f.Category = strings.ToLower(strings.TrimSpace(f.Category))
	if !IsValidTicketCategory(f.Category) {
		f.Category = ""
	}
	f.Search = clampRunes(strings.TrimSpace(f.Search), TicketSearchMaxRunes)
	return f
}

func ticketActorEmail(actor TicketActor) string {
	return clampRunes(strings.TrimSpace(actor.Email), 255)
}

// ---------- 用户侧 ----------

// Create 创建工单：校验标题 / 正文 / 分类，检查未关闭工单配额，落库后异步推送。
func (s *TicketService) Create(ctx context.Context, actor TicketActor, in TicketCreateInput, requestOrigin string) (*SupportTicket, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if actor.UserID <= 0 {
		return nil, ErrTicketForbidden
	}
	title, err := normalizeTicketTitle(in.Title)
	if err != nil {
		return nil, err
	}
	category, err := normalizeTicketCategory(in.Category)
	if err != nil {
		return nil, err
	}
	body, err := normalizeTicketBody(in.Body)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.CountActiveByUser(ctx, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("count active tickets: %w", err)
	}
	if active >= TicketOpenLimitPerUser {
		return nil, ErrTicketOpenLimit
	}

	now := s.now()
	email := ticketActorEmail(actor)
	ticket := &SupportTicket{
		UserID:        actor.UserID,
		UserEmail:     email,
		Title:         title,
		Category:      category,
		Status:        TicketStatusOpen,
		MessageCount:  1,
		LastMessageAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	first := &SupportTicketMessage{
		AuthorUserID: actor.UserID,
		AuthorEmail:  email,
		AuthorRole:   TicketAuthorRoleUser,
		Body:         body,
		CreatedAt:    now,
	}
	created, err := s.repo.Create(ctx, ticket, first)
	if err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	if n := s.getNotifier(); n != nil {
		n.NotifyTicketCreated(created, first, requestOrigin)
	}
	return created, nil
}

// ListMine 用户自己的工单列表（强制按 userID 过滤，不支持搜索）。
func (s *TicketService) ListMine(ctx context.Context, userID int64, filter *TicketFilter) (*TicketList, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	f := normalizeTicketFilter(filter)
	uid := userID
	f.UserID = &uid
	f.Search = ""
	return s.list(ctx, f)
}

// GetForUser 读取自己的工单与全部消息；非本人按不存在处理；有未读客服回复时顺带清零。
func (s *TicketService) GetForUser(ctx context.Context, id, userID int64) (*SupportTicket, []*SupportTicketMessage, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	ticket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if ticket.UserID != userID {
		return nil, nil, ErrTicketNotFound
	}
	msgs, err := s.repo.ListMessages(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list ticket messages: %w", err)
	}
	if ticket.UserUnread {
		if err := s.repo.MarkUserRead(ctx, id, userID); err != nil {
			return nil, nil, fmt.Errorf("mark ticket read: %w", err)
		}
		ticket.UserUnread = false
	}
	return ticket, msgs, nil
}

// ReplyAsUser 用户追加回复：工单回到 open（等客服），并推送提醒客服。
func (s *TicketService) ReplyAsUser(ctx context.Context, id int64, actor TicketActor, body, requestOrigin string) (*SupportTicket, *SupportTicketMessage, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	body, err := normalizeTicketBody(body)
	if err != nil {
		return nil, nil, err
	}
	ticket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if ticket.UserID != actor.UserID {
		return nil, nil, ErrTicketNotFound
	}
	now := s.now()
	msg := &SupportTicketMessage{
		TicketID:     id,
		AuthorUserID: actor.UserID,
		AuthorEmail:  ticketActorEmail(actor),
		AuthorRole:   TicketAuthorRoleUser,
		Body:         body,
		CreatedAt:    now,
	}
	updated, err := s.repo.AppendMessage(ctx, msg, TicketStatusOpen, false, now)
	if err != nil {
		return nil, nil, err
	}
	if n := s.getNotifier(); n != nil {
		n.NotifyTicketUserReplied(updated, msg, requestOrigin)
	}
	return updated, msg, nil
}

// CloseAsUser 用户关闭自己的工单。
func (s *TicketService) CloseAsUser(ctx context.Context, id int64, actor TicketActor) (*SupportTicket, error) {
	actor.Role = TicketAuthorRoleUser
	return s.transition(ctx, id, actor, true, []string{TicketStatusOpen, TicketStatusReplied}, TicketStatusClosed, ErrTicketClosed)
}

// ReopenAsUser 用户重开自己已关闭的工单（回到 open，等客服）。
func (s *TicketService) ReopenAsUser(ctx context.Context, id int64, actor TicketActor) (*SupportTicket, error) {
	actor.Role = TicketAuthorRoleUser
	return s.transition(ctx, id, actor, true, []string{TicketStatusClosed}, TicketStatusOpen, ErrTicketNotClosed)
}

// UnreadCount 用户有未读客服回复的工单数（侧边栏角标）。
func (s *TicketService) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	return s.repo.CountUserUnread(ctx, userID)
}

// ---------- 客服侧（admin / operator 同权） ----------

// ListAll 全部工单，支持状态 / 分类 / 搜索。
func (s *TicketService) ListAll(ctx context.Context, actor TicketActor, filter *TicketFilter) (*TicketList, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if !actor.IsStaff() {
		return nil, ErrTicketForbidden
	}
	return s.list(ctx, normalizeTicketFilter(filter))
}

// GetForStaff 读取任意工单与全部消息，不动未读标记。
func (s *TicketService) GetForStaff(ctx context.Context, id int64, actor TicketActor) (*SupportTicket, []*SupportTicketMessage, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	if !actor.IsStaff() {
		return nil, nil, ErrTicketForbidden
	}
	ticket, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	msgs, err := s.repo.ListMessages(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list ticket messages: %w", err)
	}
	return ticket, msgs, nil
}

// ReplyAsStaff 客服回复：工单转为 replied（等用户）并给用户打未读标记；不推送。
func (s *TicketService) ReplyAsStaff(ctx context.Context, id int64, actor TicketActor, body string) (*SupportTicket, *SupportTicketMessage, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	if !actor.IsStaff() {
		return nil, nil, ErrTicketForbidden
	}
	body, err := normalizeTicketBody(body)
	if err != nil {
		return nil, nil, err
	}
	now := s.now()
	msg := &SupportTicketMessage{
		TicketID:     id,
		AuthorUserID: actor.UserID,
		AuthorEmail:  ticketActorEmail(actor),
		AuthorRole:   actor.Role,
		Body:         body,
		CreatedAt:    now,
	}
	updated, err := s.repo.AppendMessage(ctx, msg, TicketStatusReplied, true, now)
	if err != nil {
		return nil, nil, err
	}
	return updated, msg, nil
}

// CloseAsStaff 客服关闭任意工单。
func (s *TicketService) CloseAsStaff(ctx context.Context, id int64, actor TicketActor) (*SupportTicket, error) {
	if !actor.IsStaff() {
		return nil, ErrTicketForbidden
	}
	return s.transition(ctx, id, actor, false, []string{TicketStatusOpen, TicketStatusReplied}, TicketStatusClosed, ErrTicketClosed)
}

// ReopenAsStaff 客服重开任意已关闭工单。
func (s *TicketService) ReopenAsStaff(ctx context.Context, id int64, actor TicketActor) (*SupportTicket, error) {
	if !actor.IsStaff() {
		return nil, ErrTicketForbidden
	}
	return s.transition(ctx, id, actor, false, []string{TicketStatusClosed}, TicketStatusOpen, ErrTicketNotClosed)
}

// OpenCount 全站待处理（open）工单数（客服侧角标）。
func (s *TicketService) OpenCount(ctx context.Context) (int64, error) {
	if err := s.ready(); err != nil {
		return 0, err
	}
	return s.repo.CountByStatus(ctx, TicketStatusOpen)
}

// ---------- 内部 ----------

func (s *TicketService) list(ctx context.Context, f TicketFilter) (*TicketList, error) {
	items, total, err := s.repo.List(ctx, &f)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	if items == nil {
		items = []*SupportTicket{}
	}
	return &TicketList{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

// transition 执行状态迁移；ownerOnly 时先确认工单属于 actor；未命中（状态不对）返回 missErr，不存在返回 ErrTicketNotFound。
func (s *TicketService) transition(ctx context.Context, id int64, actor TicketActor, ownerOnly bool, from []string, to string, missErr error) (*SupportTicket, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if ownerOnly {
		ticket, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if ticket.UserID != actor.UserID {
			return nil, ErrTicketNotFound
		}
	}
	ok, err := s.repo.Transition(ctx, id, from, to, actor, s.now())
	if err != nil {
		return nil, fmt.Errorf("transition ticket: %w", err)
	}
	if !ok {
		if _, err := s.repo.GetByID(ctx, id); err != nil {
			return nil, err
		}
		return nil, missErr
	}
	return s.repo.GetByID(ctx, id)
}
