package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// siteMessageTestRepo 内存版站内信仓储（也被内容审计 / 申诉测试复用）。
type siteMessageTestRepo struct {
	mu       sync.Mutex
	nextID   int64
	messages []*SiteMessage
	// users 非 nil 时只有其中的用户 id 能收信（模拟 deleted_at IS NULL 的过滤）
	users     map[int64]bool
	createErr error
}

func (r *siteMessageTestRepo) Create(_ context.Context, msg *SiteMessage) (*SiteMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.users != nil && !r.users[msg.UserID] {
		return nil, ErrUserNotFound
	}
	r.nextID++
	out := *msg
	out.ID = r.nextID
	if out.CreatedAt.IsZero() {
		out.CreatedAt = time.Now()
	}
	r.messages = append(r.messages, &out)
	clone := out
	return &clone, nil
}

func (r *siteMessageTestRepo) ListByUser(_ context.Context, userID int64, f SiteMessageFilter) ([]*SiteMessage, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []*SiteMessage
	for _, m := range r.messages {
		if m.UserID != userID || (f.UnreadOnly && m.ReadAt != nil) || (f.Category != "" && m.Category != f.Category) {
			continue
		}
		clone := *m
		matched = append(matched, &clone)
	}
	sort.SliceStable(matched, func(i, j int) bool { return matched[i].ID > matched[j].ID })
	total := int64(len(matched))
	start := (f.Page - 1) * f.PageSize
	if start > len(matched) {
		start = len(matched)
	}
	end := start + f.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

func (r *siteMessageTestRepo) CountUnread(_ context.Context, userID int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, m := range r.messages {
		if m.UserID == userID && m.ReadAt == nil {
			n++
		}
	}
	return n, nil
}

func (r *siteMessageTestRepo) MarkRead(_ context.Context, userID, id int64, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.messages {
		if m.ID == id && m.UserID == userID {
			if m.ReadAt == nil {
				at := now
				m.ReadAt = &at
			}
			return nil
		}
	}
	return ErrSiteMessageNotFound
}

func (r *siteMessageTestRepo) MarkAllRead(_ context.Context, userID int64, now time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, m := range r.messages {
		if m.UserID == userID && m.ReadAt == nil {
			at := now
			m.ReadAt = &at
			n++
		}
	}
	return n, nil
}

func (r *siteMessageTestRepo) snapshot() []SiteMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]SiteMessage, 0, len(r.messages))
	for _, m := range r.messages {
		out = append(out, *m)
	}
	return out
}

func TestSiteMessageService_SendFromStaffValidates(t *testing.T) {
	svc := NewSiteMessageService(&siteMessageTestRepo{})
	ctx := context.Background()

	_, err := svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 1, Title: "  ", Content: "x"})
	require.ErrorIs(t, err, ErrSiteMessageTitleInvalid)
	_, err = svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 1, Title: strings.Repeat("标", SiteMessageTitleMaxRunes+1), Content: "x"})
	require.ErrorIs(t, err, ErrSiteMessageTitleInvalid)
	_, err = svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 1, Title: "t", Content: "\r\n "})
	require.ErrorIs(t, err, ErrSiteMessageContentInvalid)
	_, err = svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 1, Title: "t", Content: strings.Repeat("字", SiteMessageContentMaxRunes+1)})
	require.ErrorIs(t, err, ErrSiteMessageContentInvalid)
	_, err = svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 0, Title: "t", Content: "c"})
	require.ErrorIs(t, err, ErrUserNotFound)

	// 按 rune 计数：上限内的中文正好通过
	_, err = svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 1, Title: strings.Repeat("标", SiteMessageTitleMaxRunes), Content: strings.Repeat("字", SiteMessageContentMaxRunes)})
	require.NoError(t, err)
}

func TestSiteMessageService_SendFromStaffForcesAdminCategory(t *testing.T) {
	repo := &siteMessageTestRepo{}
	svc := NewSiteMessageService(repo)

	msg, err := svc.SendFromStaff(context.Background(), SiteMessageSendInput{
		RecipientUserID: 9, Title: "  标题 ", Content: "第一行\r\n第二行", SenderUserID: 1, SenderRole: RoleAdmin,
	})
	require.NoError(t, err)
	require.Equal(t, SiteMessageCategoryAdmin, msg.Category)
	require.Equal(t, SiteMessageSourceAdmin, msg.SourceType)
	require.Equal(t, "标题", msg.Title)
	require.Equal(t, "第一行\n第二行", msg.Content)
	require.NotNil(t, msg.SenderUserID)
	require.Equal(t, int64(1), *msg.SenderUserID)
	require.Nil(t, msg.ApprovalID)
	require.Empty(t, msg.SourceID)
}

func TestSiteMessageService_SendFromStaffRecordsApproval(t *testing.T) {
	repo := &siteMessageTestRepo{}
	svc := NewSiteMessageService(repo)
	approvalID := int64(42)

	msg, err := svc.SendFromStaff(context.Background(), SiteMessageSendInput{
		RecipientUserID: 9, Title: "t", Content: "c", SenderUserID: 7, SenderRole: RoleOperator, ApprovalID: &approvalID,
	})
	require.NoError(t, err)
	require.Equal(t, RoleOperator, msg.SenderRole)
	require.Equal(t, int64(7), *msg.SenderUserID)
	require.Equal(t, int64(42), *msg.ApprovalID)
	require.Equal(t, "approval:42", msg.SourceID)
}

func TestSiteMessageService_SendToDeletedUserIsNotFound(t *testing.T) {
	svc := NewSiteMessageService(&siteMessageTestRepo{users: map[int64]bool{1: true}})
	_, err := svc.SendFromStaff(context.Background(), SiteMessageSendInput{RecipientUserID: 2, Title: "t", Content: "c"})
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestSiteMessageService_UserInboxIsolation(t *testing.T) {
	repo := &siteMessageTestRepo{}
	svc := NewSiteMessageService(repo)
	ctx := context.Background()

	mine, err := svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 1, Title: "a", Content: "a"})
	require.NoError(t, err)
	other, err := svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 2, Title: "b", Content: "b"})
	require.NoError(t, err)

	count, err := svc.UnreadCount(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	// 别人的消息按不存在处理
	require.ErrorIs(t, svc.MarkRead(ctx, 1, other.ID), ErrSiteMessageNotFound)
	require.NoError(t, svc.MarkRead(ctx, 1, mine.ID))
	require.NoError(t, svc.MarkRead(ctx, 1, mine.ID), "marking twice is idempotent")

	count, err = svc.UnreadCount(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, count)

	list, err := svc.ListMine(ctx, 1, nil)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, mine.ID, list.Items[0].ID)
}

func TestSiteMessageService_MarkAllReadAndFilters(t *testing.T) {
	repo := &siteMessageTestRepo{}
	svc := NewSiteMessageService(repo)
	ctx := context.Background()

	require.NoError(t, svc.DeliverSystemMessage(ctx, SystemSiteMessageInput{UserID: 1, Category: SiteMessageCategorySecurity, Title: "s", Content: "s"}))
	_, err := svc.SendFromStaff(ctx, SiteMessageSendInput{RecipientUserID: 1, Title: "a", Content: "a"})
	require.NoError(t, err)

	list, err := svc.ListMine(ctx, 1, &SiteMessageFilter{Category: "SECURITY", UnreadOnly: true})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, SiteMessageCategorySecurity, list.Items[0].Category)

	list, err = svc.ListMine(ctx, 1, &SiteMessageFilter{Category: "bogus", PageSize: 1000})
	require.NoError(t, err)
	require.Len(t, list.Items, 2, "unknown category means all")
	require.Equal(t, SiteMessageListPageSizeMax, list.PageSize)

	updated, err := svc.MarkAllRead(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(2), updated)
	list, err = svc.ListMine(ctx, 1, &SiteMessageFilter{UnreadOnly: true})
	require.NoError(t, err)
	require.Empty(t, list.Items)
}

func TestSiteMessageService_DeliverSystemMessage(t *testing.T) {
	repo := &siteMessageTestRepo{users: map[int64]bool{1: true}}
	svc := NewSiteMessageService(repo)
	ctx := context.Background()

	require.NoError(t, svc.DeliverSystemMessage(ctx, SystemSiteMessageInput{UserID: 0, Title: "t", Content: "c"}), "no recipient is a no-op")
	require.NoError(t, svc.DeliverSystemMessage(ctx, SystemSiteMessageInput{UserID: 2, Title: "t", Content: "c"}), "deleted user is ignored")
	require.NoError(t, svc.DeliverSystemMessage(ctx, SystemSiteMessageInput{
		UserID: 1, Category: "weird", Title: strings.Repeat("t", 500), Content: "c", SourceType: "x", SourceID: strings.Repeat("s", 300),
	}))

	msgs := repo.snapshot()
	require.Len(t, msgs, 1)
	require.Equal(t, SiteMessageCategorySystem, msgs[0].Category, "unknown category falls back to system")
	require.Len(t, []rune(msgs[0].Title), SiteMessageTitleMaxRunes)
	require.Len(t, []rune(msgs[0].SourceID), siteMessageSourceIDMaxRunes)
	require.Nil(t, msgs[0].SenderUserID)

	repo.createErr = errors.New("db down")
	require.Error(t, svc.DeliverSystemMessage(ctx, SystemSiteMessageInput{UserID: 1, Title: "t", Content: "c"}))
}

func TestSiteMessageService_NilRepoIsUnavailable(t *testing.T) {
	svc := NewSiteMessageService(nil)
	_, err := svc.UnreadCount(context.Background(), 1)
	require.ErrorIs(t, err, ErrSiteMessageUnavailable)
}
