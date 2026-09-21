package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------- 桩 ----------

type ticketRepoStub struct {
	mu       sync.Mutex
	seq      int64
	msgSeq   int64
	tickets  map[int64]*SupportTicket
	messages map[int64][]*SupportTicketMessage
}

func newTicketRepoStub() *ticketRepoStub {
	return &ticketRepoStub{tickets: map[int64]*SupportTicket{}, messages: map[int64][]*SupportTicketMessage{}}
}

func (r *ticketRepoStub) Create(_ context.Context, ticket *SupportTicket, first *SupportTicketMessage) (*SupportTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	clone := *ticket
	clone.ID = r.seq
	clone.MessageCount = 0
	if first != nil {
		r.msgSeq++
		first.ID = r.msgSeq
		first.TicketID = clone.ID
		m := *first
		r.messages[clone.ID] = append(r.messages[clone.ID], &m)
		clone.MessageCount = 1
	}
	r.tickets[clone.ID] = &clone
	out := clone
	return &out, nil
}

func (r *ticketRepoStub) GetByID(_ context.Context, id int64) (*SupportTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.tickets[id]
	if !ok {
		return nil, ErrTicketNotFound
	}
	out := *item
	return &out, nil
}

func (r *ticketRepoStub) List(_ context.Context, f *TicketFilter) ([]*SupportTicket, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]*SupportTicket, 0)
	for _, item := range r.tickets {
		if f != nil {
			if f.UserID != nil && item.UserID != *f.UserID {
				continue
			}
			if f.Status != "" && item.Status != f.Status {
				continue
			}
			if f.Category != "" && item.Category != f.Category {
				continue
			}
			if f.Search != "" {
				s := strings.ToLower(f.Search)
				if !strings.Contains(strings.ToLower(item.Title), s) && !strings.Contains(strings.ToLower(item.UserEmail), s) {
					continue
				}
			}
		}
		clone := *item
		all = append(all, &clone)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].LastMessageAt.Equal(all[j].LastMessageAt) {
			return all[i].ID > all[j].ID
		}
		return all[i].LastMessageAt.After(all[j].LastMessageAt)
	})
	total := int64(len(all))
	page, size := 1, 20
	if f != nil {
		if f.Page > 0 {
			page = f.Page
		}
		if f.PageSize > 0 {
			size = f.PageSize
		}
	}
	start := (page - 1) * size
	if start >= len(all) {
		return []*SupportTicket{}, total, nil
	}
	end := start + size
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}

func (r *ticketRepoStub) ListMessages(_ context.Context, ticketID int64) ([]*SupportTicketMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*SupportTicketMessage, 0, len(r.messages[ticketID]))
	for _, m := range r.messages[ticketID] {
		c := *m
		out = append(out, &c)
	}
	return out, nil
}

func (r *ticketRepoStub) AppendMessage(_ context.Context, msg *SupportTicketMessage, newStatus string, userUnread bool, now time.Time) (*SupportTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.tickets[msg.TicketID]
	if !ok {
		return nil, ErrTicketNotFound
	}
	if item.Status == TicketStatusClosed {
		return nil, ErrTicketClosed
	}
	r.msgSeq++
	msg.ID = r.msgSeq
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	m := *msg
	r.messages[msg.TicketID] = append(r.messages[msg.TicketID], &m)
	item.Status = newStatus
	item.UserUnread = userUnread
	item.LastMessageAt = now
	item.MessageCount++
	item.UpdatedAt = now
	out := *item
	return &out, nil
}

func (r *ticketRepoStub) Transition(_ context.Context, id int64, from []string, to string, actor TicketActor, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.tickets[id]
	if !ok {
		return false, nil
	}
	matched := false
	for _, s := range from {
		if item.Status == s {
			matched = true
			break
		}
	}
	if !matched {
		return false, nil
	}
	item.Status = to
	item.UpdatedAt = now
	if to == TicketStatusClosed {
		at := now
		item.ClosedAt = &at
		uid := actor.UserID
		item.ClosedByUserID = &uid
		item.ClosedByRole = actor.Role
	} else {
		item.ClosedAt = nil
		item.ClosedByUserID = nil
		item.ClosedByRole = ""
	}
	return true, nil
}

func (r *ticketRepoStub) MarkUserRead(_ context.Context, id, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item, ok := r.tickets[id]; ok && item.UserID == userID {
		item.UserUnread = false
	}
	return nil
}

func (r *ticketRepoStub) CountByStatus(_ context.Context, status string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.tickets {
		if item.Status == status {
			n++
		}
	}
	return n, nil
}

func (r *ticketRepoStub) CountActiveByUser(_ context.Context, userID int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.tickets {
		if item.UserID == userID && item.Status != TicketStatusClosed {
			n++
		}
	}
	return n, nil
}

func (r *ticketRepoStub) CountUserUnread(_ context.Context, userID int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.tickets {
		if item.UserID == userID && item.UserUnread {
			n++
		}
	}
	return n, nil
}

func (r *ticketRepoStub) StaffAttachmentReferencedForUser(_ context.Context, userID int64, key string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	needle := TicketAttachmentURLScheme + key
	for id, item := range r.tickets {
		if item.UserID != userID {
			continue
		}
		for _, m := range r.messages[id] {
			if m.AuthorRole != TicketAuthorRoleUser && strings.Contains(m.Body, needle) {
				return true, nil
			}
		}
	}
	return false, nil
}

type ticketNotifierStub struct {
	mu     sync.Mutex
	events []string
}

func (n *ticketNotifierStub) NotifyTicketCreated(t *SupportTicket, _ *SupportTicketMessage, origin string) {
	n.record("created", t, origin)
}

func (n *ticketNotifierStub) NotifyTicketUserReplied(t *SupportTicket, _ *SupportTicketMessage, origin string) {
	n.record("user_replied", t, origin)
}

func (n *ticketNotifierStub) record(kind string, t *SupportTicket, origin string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, fmt.Sprintf("%s:%d:%s", kind, t.ID, origin))
}

func (n *ticketNotifierStub) snapshot() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.events...)
}

var (
	ticketTestUser     = TicketActor{UserID: 1, Email: "alice@example.com", Role: RoleUser}
	ticketTestOther    = TicketActor{UserID: 2, Email: "bob@example.com", Role: RoleUser}
	ticketTestOperator = TicketActor{UserID: 9, Email: "ops@example.com", Role: RoleOperator}
	ticketTestAdmin    = TicketActor{UserID: 8, Email: "admin@example.com", Role: RoleAdmin}
)

func newTicketServiceForTest() (*TicketService, *ticketRepoStub, *ticketNotifierStub) {
	repo := newTicketRepoStub()
	notifier := &ticketNotifierStub{}
	svc := NewTicketService(repo)
	svc.SetNotifier(notifier)
	base := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	tick := 0
	svc.now = func() time.Time {
		tick++
		return base.Add(time.Duration(tick) * time.Second)
	}
	return svc, repo, notifier
}

func mustCreateTicket(t *testing.T, svc *TicketService, actor TicketActor, title string) *SupportTicket {
	t.Helper()
	created, err := svc.Create(context.Background(), actor, TicketCreateInput{Title: title, Category: TicketCategoryAPI, Body: "body of " + title}, "https://origin.example.com")
	require.NoError(t, err)
	return created
}

// ---------- 用例 ----------

func TestTicketService_CreateValidation(t *testing.T) {
	svc, _, _ := newTicketServiceForTest()
	ctx := context.Background()

	_, err := svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: "   ", Category: TicketCategoryAPI, Body: "x"}, "")
	require.ErrorIs(t, err, ErrTicketTitleInvalid)

	_, err = svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: strings.Repeat("标", TicketTitleMaxRunes+1), Category: TicketCategoryAPI, Body: "x"}, "")
	require.ErrorIs(t, err, ErrTicketTitleInvalid)

	_, err = svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: "t", Category: "unknown", Body: "x"}, "")
	require.ErrorIs(t, err, ErrTicketInvalidCategory)

	_, err = svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: "t", Category: TicketCategoryAPI, Body: " \n "}, "")
	require.ErrorIs(t, err, ErrTicketBodyInvalid)

	_, err = svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: "t", Category: TicketCategoryAPI, Body: strings.Repeat("字", TicketBodyMaxRunes+1)}, "")
	require.ErrorIs(t, err, ErrTicketBodyInvalid)

	// 恰好 200 / 5000 个 rune 合法（按 rune 而非字节计）
	created, err := svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: strings.Repeat("标", TicketTitleMaxRunes), Category: " API ", Body: strings.Repeat("字", TicketBodyMaxRunes)}, "")
	require.NoError(t, err)
	require.Equal(t, TicketCategoryAPI, created.Category, "分类大小写与空白被归一")

	_, err = svc.Create(ctx, TicketActor{}, TicketCreateInput{Title: "t", Category: TicketCategoryAPI, Body: "x"}, "")
	require.ErrorIs(t, err, ErrTicketForbidden)
}

func TestTicketService_CreateSetsFieldsAndNotifies(t *testing.T) {
	svc, repo, notifier := newTicketServiceForTest()
	created := mustCreateTicket(t, svc, ticketTestUser, "first")

	require.Equal(t, TicketStatusOpen, created.Status)
	require.Equal(t, 1, created.MessageCount)
	require.False(t, created.UserUnread)
	require.Equal(t, "alice@example.com", created.UserEmail)
	require.Equal(t, int64(1), created.UserID)

	msgs, err := repo.ListMessages(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, TicketAuthorRoleUser, msgs[0].AuthorRole)
	require.Equal(t, "body of first", msgs[0].Body)
	require.Equal(t, created.ID, msgs[0].TicketID)

	require.Equal(t, []string{"created:1:https://origin.example.com"}, notifier.snapshot())
}

func TestTicketService_CreateEnforcesOpenLimit(t *testing.T) {
	svc, _, _ := newTicketServiceForTest()
	ctx := context.Background()
	var last *SupportTicket
	for i := 0; i < TicketOpenLimitPerUser; i++ {
		last = mustCreateTicket(t, svc, ticketTestUser, fmt.Sprintf("t%d", i))
	}
	_, err := svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: "one more", Category: TicketCategoryOther, Body: "x"}, "")
	require.ErrorIs(t, err, ErrTicketOpenLimit)

	// 别人的配额独立
	mustCreateTicket(t, svc, ticketTestOther, "other")

	// 关掉一个后又能建
	_, err = svc.CloseAsUser(ctx, last.ID, ticketTestUser)
	require.NoError(t, err)
	_, err = svc.Create(ctx, ticketTestUser, TicketCreateInput{Title: "after close", Category: TicketCategoryOther, Body: "x"}, "")
	require.NoError(t, err)
}

func TestTicketService_GetForUserHidesOthersAndClearsUnread(t *testing.T) {
	svc, repo, _ := newTicketServiceForTest()
	ctx := context.Background()
	created := mustCreateTicket(t, svc, ticketTestUser, "mine")

	_, _, err := svc.GetForUser(ctx, created.ID, ticketTestOther.UserID)
	require.ErrorIs(t, err, ErrTicketNotFound, "他人工单按不存在处理")

	_, _, err = svc.GetForUser(ctx, 999, ticketTestUser.UserID)
	require.ErrorIs(t, err, ErrTicketNotFound)

	_, _, err = svc.ReplyAsStaff(ctx, created.ID, ticketTestOperator, "we are on it")
	require.NoError(t, err)
	unread, err := svc.UnreadCount(ctx, ticketTestUser.UserID)
	require.NoError(t, err)
	require.Equal(t, int64(1), unread)

	ticket, msgs, err := svc.GetForUser(ctx, created.ID, ticketTestUser.UserID)
	require.NoError(t, err)
	require.False(t, ticket.UserUnread, "打开详情即清未读")
	require.Len(t, msgs, 2)
	require.Equal(t, RoleOperator, msgs[1].AuthorRole)

	stored, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.False(t, stored.UserUnread)
	unread, err = svc.UnreadCount(ctx, ticketTestUser.UserID)
	require.NoError(t, err)
	require.Zero(t, unread)
}

func TestTicketService_ReplyStateMachine(t *testing.T) {
	svc, _, notifier := newTicketServiceForTest()
	ctx := context.Background()
	created := mustCreateTicket(t, svc, ticketTestUser, "flow")

	// 用户追加：仍是 open，推送 user_replied
	ticket, msg, err := svc.ReplyAsUser(ctx, created.ID, ticketTestUser, "more details", "http://o")
	require.NoError(t, err)
	require.Equal(t, TicketStatusOpen, ticket.Status)
	require.Equal(t, 2, ticket.MessageCount)
	require.Equal(t, TicketAuthorRoleUser, msg.AuthorRole)
	require.NotZero(t, msg.ID)

	// 客服回复：replied + 用户未读，不推送
	ticket, msg, err = svc.ReplyAsStaff(ctx, created.ID, ticketTestAdmin, "answer")
	require.NoError(t, err)
	require.Equal(t, TicketStatusReplied, ticket.Status)
	require.True(t, ticket.UserUnread)
	require.Equal(t, RoleAdmin, msg.AuthorRole)
	require.Equal(t, "admin@example.com", msg.AuthorEmail)

	// 用户再回：回到 open，未读清零
	ticket, _, err = svc.ReplyAsUser(ctx, created.ID, ticketTestUser, "thanks, but", "http://o")
	require.NoError(t, err)
	require.Equal(t, TicketStatusOpen, ticket.Status)
	require.False(t, ticket.UserUnread)

	// 他人不能回复
	_, _, err = svc.ReplyAsUser(ctx, created.ID, ticketTestOther, "hijack", "")
	require.ErrorIs(t, err, ErrTicketNotFound)

	// 客服关闭
	ticket, err = svc.CloseAsStaff(ctx, created.ID, ticketTestOperator)
	require.NoError(t, err)
	require.Equal(t, TicketStatusClosed, ticket.Status)
	require.NotNil(t, ticket.ClosedAt)
	require.Equal(t, RoleOperator, ticket.ClosedByRole)
	require.Equal(t, int64(9), *ticket.ClosedByUserID)

	// 关闭后不能回复 / 再关闭
	_, _, err = svc.ReplyAsUser(ctx, created.ID, ticketTestUser, "still?", "")
	require.ErrorIs(t, err, ErrTicketClosed)
	_, _, err = svc.ReplyAsStaff(ctx, created.ID, ticketTestAdmin, "still?")
	require.ErrorIs(t, err, ErrTicketClosed)
	_, err = svc.CloseAsUser(ctx, created.ID, ticketTestUser)
	require.ErrorIs(t, err, ErrTicketClosed)

	// 用户重开 → open，closed_* 清空
	ticket, err = svc.ReopenAsUser(ctx, created.ID, ticketTestUser)
	require.NoError(t, err)
	require.Equal(t, TicketStatusOpen, ticket.Status)
	require.Nil(t, ticket.ClosedAt)
	require.Empty(t, ticket.ClosedByRole)
	_, err = svc.ReopenAsStaff(ctx, created.ID, ticketTestAdmin)
	require.ErrorIs(t, err, ErrTicketNotClosed)

	// 用户关闭时记录 closed_by_role=user
	ticket, err = svc.CloseAsUser(ctx, created.ID, ticketTestUser)
	require.NoError(t, err)
	require.Equal(t, TicketAuthorRoleUser, ticket.ClosedByRole)

	// 不存在的工单
	_, err = svc.CloseAsStaff(ctx, 404, ticketTestAdmin)
	require.ErrorIs(t, err, ErrTicketNotFound)
	_, err = svc.ReopenAsUser(ctx, 404, ticketTestUser)
	require.ErrorIs(t, err, ErrTicketNotFound)

	require.Equal(t, []string{
		"created:1:https://origin.example.com",
		"user_replied:1:http://o",
		"user_replied:1:http://o",
	}, notifier.snapshot(), "只有创建与用户回复触发推送")
}

func TestTicketService_StaffGuards(t *testing.T) {
	svc, _, _ := newTicketServiceForTest()
	ctx := context.Background()
	created := mustCreateTicket(t, svc, ticketTestUser, "guard")

	_, _, err := svc.ReplyAsStaff(ctx, created.ID, ticketTestOther, "not staff")
	require.ErrorIs(t, err, ErrTicketForbidden)
	_, err = svc.ListAll(ctx, ticketTestOther, nil)
	require.ErrorIs(t, err, ErrTicketForbidden)
	_, _, err = svc.GetForStaff(ctx, created.ID, ticketTestOther)
	require.ErrorIs(t, err, ErrTicketForbidden)
	_, err = svc.CloseAsStaff(ctx, created.ID, ticketTestOther)
	require.ErrorIs(t, err, ErrTicketForbidden)
	_, err = svc.ReopenAsStaff(ctx, created.ID, ticketTestOther)
	require.ErrorIs(t, err, ErrTicketForbidden)

	// operator 与 admin 同权
	ticket, msgs, err := svc.GetForStaff(ctx, created.ID, ticketTestOperator)
	require.NoError(t, err)
	require.Equal(t, created.ID, ticket.ID)
	require.Len(t, msgs, 1)
	_, _, err = svc.ReplyAsStaff(ctx, created.ID, ticketTestOperator, "operator here")
	require.NoError(t, err)

	// 客服回复的空正文一样被拒
	_, _, err = svc.ReplyAsStaff(ctx, created.ID, ticketTestAdmin, "   ")
	require.ErrorIs(t, err, ErrTicketBodyInvalid)
}

func TestTicketService_ListScopesAndFilters(t *testing.T) {
	svc, _, _ := newTicketServiceForTest()
	ctx := context.Background()
	a1 := mustCreateTicket(t, svc, ticketTestUser, "alpha billing question")
	mustCreateTicket(t, svc, ticketTestUser, "beta")
	b1 := mustCreateTicket(t, svc, ticketTestOther, "gamma")

	mine, err := svc.ListMine(ctx, ticketTestUser.UserID, &TicketFilter{Search: "gamma", PageSize: 500})
	require.NoError(t, err)
	require.Equal(t, int64(2), mine.Total, "用户侧忽略搜索且只看自己的")
	require.Equal(t, TicketListPageSizeMax, mine.PageSize, "分页上限被夹紧")
	for _, item := range mine.Items {
		require.Equal(t, ticketTestUser.UserID, item.UserID)
	}

	all, err := svc.ListAll(ctx, ticketTestOperator, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), all.Total)
	require.Equal(t, b1.ID, all.Items[0].ID, "按最后更新时间倒序")

	byEmail, err := svc.ListAll(ctx, ticketTestAdmin, &TicketFilter{Search: "BOB@"})
	require.NoError(t, err)
	require.Equal(t, int64(1), byEmail.Total)
	require.Equal(t, b1.ID, byEmail.Items[0].ID)

	byTitle, err := svc.ListAll(ctx, ticketTestAdmin, &TicketFilter{Search: "alpha"})
	require.NoError(t, err)
	require.Equal(t, int64(1), byTitle.Total)
	require.Equal(t, a1.ID, byTitle.Items[0].ID)

	_, err = svc.CloseAsStaff(ctx, a1.ID, ticketTestAdmin)
	require.NoError(t, err)
	closed, err := svc.ListAll(ctx, ticketTestAdmin, &TicketFilter{Status: "CLOSED"})
	require.NoError(t, err)
	require.Equal(t, int64(1), closed.Total)
	bogus, err := svc.ListAll(ctx, ticketTestAdmin, &TicketFilter{Status: "nope", Category: "nope"})
	require.NoError(t, err)
	require.Equal(t, int64(3), bogus.Total, "非法筛选值当作全部")

	open, err := svc.OpenCount(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), open)
}

func TestTicketService_NilSafety(t *testing.T) {
	var nilSvc *TicketService
	_, err := nilSvc.Create(context.Background(), ticketTestUser, TicketCreateInput{}, "")
	require.ErrorIs(t, err, ErrTicketUnavailable)
	nilSvc.SetNotifier(nil)

	svc := NewTicketService(nil)
	_, err = svc.OpenCount(context.Background())
	require.True(t, errors.Is(err, ErrTicketUnavailable))
}
