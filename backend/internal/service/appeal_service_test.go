package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// appealTestStore 内存版申诉会话存储（每个用户只留一个会话，与 Redis 实现同语义）。
type appealTestStore struct {
	mu      sync.Mutex
	records map[string]*AppealSessionRecord
	byUser  map[int64]string
}

func newAppealTestStore() *appealTestStore {
	return &appealTestStore{records: map[string]*AppealSessionRecord{}, byUser: map[int64]string{}}
}

func (s *appealTestStore) Save(_ context.Context, hash string, record *AppealSessionRecord, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.byUser[record.UserID]; ok {
		delete(s.records, old)
	}
	clone := *record
	s.records[hash] = &clone
	s.byUser[record.UserID] = hash
	return nil
}

func (s *appealTestStore) Load(_ context.Context, hash string) (*AppealSessionRecord, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[hash]
	if !ok || s.byUser[r.UserID] != hash {
		return nil, 0, nil
	}
	clone := *r
	return &clone, time.Hour, nil
}

func (s *appealTestStore) Revoke(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.records[hash]; ok && s.byUser[r.UserID] == hash {
		delete(s.byUser, r.UserID)
	}
	delete(s.records, hash)
	return nil
}

func (s *appealTestStore) RevokeUser(_ context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hash, ok := s.byUser[userID]; ok {
		delete(s.records, hash)
		delete(s.byUser, userID)
	}
	return nil
}

func newAppealTestService(user *User) (*AppealService, *appealTestStore, *contentModerationTestUserRepo, *siteMessageTestRepo, *ticketRepoStub) {
	store := newAppealTestStore()
	users := &contentModerationTestUserRepo{user: user}
	messages := &siteMessageTestRepo{}
	tickets := newTicketRepoStub()
	svc := NewAppealService(store, users, nil, NewTicketService(tickets), NewSiteMessageService(messages))
	return svc, store, users, messages, tickets
}

func TestAppealService_IssuesOnlyForDisabledAccounts(t *testing.T) {
	ctx := context.Background()

	svc, _, users, _, _ := newAppealTestService(&User{ID: 5, Email: "u@example.com", Role: RoleUser, Status: StatusActive})
	token, _, err := svc.Issue(ctx, 5)
	require.NoError(t, err)
	require.Empty(t, token, "active accounts never get an appeal token")

	users.user.Status = StatusDisabled
	token, ttl, err := svc.Issue(ctx, 5)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(token, AppealTokenPrefix))
	require.Equal(t, AppealSessionTTL, ttl)

	sess, err := svc.Authenticate(ctx, token)
	require.NoError(t, err)
	require.Equal(t, int64(5), sess.UserID)
	require.Equal(t, "u@example.com", sess.Email)
}

func TestAppealService_RejectsMalformedTokens(t *testing.T) {
	svc, _, _, _, _ := newAppealTestService(&User{ID: 5, Status: StatusDisabled})
	for _, token := range []string{"", "Bearer x", "pat-abc", "apl_" + strings.Repeat("x", 200), "apl_unknown"} {
		_, err := svc.Authenticate(context.Background(), token)
		require.ErrorIs(t, err, ErrAppealTokenInvalid, token)
	}
}

func TestAppealService_RestoredAccountKillsToken(t *testing.T) {
	ctx := context.Background()
	svc, store, users, _, _ := newAppealTestService(&User{ID: 5, Status: StatusDisabled})
	token, _, err := svc.Issue(ctx, 5)
	require.NoError(t, err)

	users.user.Status = StatusActive
	_, err = svc.Authenticate(ctx, token)
	require.ErrorIs(t, err, ErrAppealAccountActive)
	require.Empty(t, store.records, "the session is revoked once the account is active")

	// 再次被封也不能复活旧令牌
	users.user.Status = StatusDisabled
	_, err = svc.Authenticate(ctx, token)
	require.ErrorIs(t, err, ErrAppealTokenInvalid)
}

func TestAppealService_DeletedAccountInvalidatesToken(t *testing.T) {
	ctx := context.Background()
	svc, _, users, _, _ := newAppealTestService(&User{ID: 5, Status: StatusDisabled})
	token, _, err := svc.Issue(ctx, 5)
	require.NoError(t, err)

	users.user = nil
	_, err = svc.Authenticate(ctx, token)
	require.ErrorIs(t, err, ErrAppealTokenInvalid)
}

func TestAppealService_NewSessionReplacesOld(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, _ := newAppealTestService(&User{ID: 5, Status: StatusDisabled})
	first, _, err := svc.Issue(ctx, 5)
	require.NoError(t, err)
	second, _, err := svc.Issue(ctx, 5)
	require.NoError(t, err)

	_, err = svc.Authenticate(ctx, first)
	require.ErrorIs(t, err, ErrAppealTokenInvalid)
	_, err = svc.Authenticate(ctx, second)
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(ctx, second))
	_, err = svc.Authenticate(ctx, second)
	require.ErrorIs(t, err, ErrAppealTokenInvalid)
}

func TestAppealService_OneActiveAppealTicket(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, tickets := newAppealTestService(&User{ID: 5, Email: "u@example.com", Status: StatusDisabled})
	sess := &AppealSession{UserID: 5, Email: "u@example.com", Role: RoleOperator}

	ticket, err := svc.CreateTicket(ctx, sess, "请求解封", "我没有违规", "")
	require.NoError(t, err)
	require.Equal(t, TicketCategoryAppeal, ticket.Category)
	require.Equal(t, int64(5), ticket.UserID)

	_, err = svc.CreateTicket(ctx, sess, "再来一次", "正文", "")
	require.ErrorIs(t, err, ErrTicketAppealActive)

	active, err := svc.HasActiveAppealTicket(ctx, 5)
	require.NoError(t, err)
	require.True(t, active)

	// 关闭后可以重新提交
	tickets.mu.Lock()
	tickets.tickets[ticket.ID].Status = TicketStatusClosed
	tickets.mu.Unlock()
	_, err = svc.CreateTicket(ctx, sess, "重新申诉", "补充说明", "")
	require.NoError(t, err)
}

func TestTicketService_NormalCreateRejectsAppealCategory(t *testing.T) {
	svc := NewTicketService(newTicketRepoStub())
	_, err := svc.Create(context.Background(), TicketActor{UserID: 1, Role: RoleUser}, TicketCreateInput{
		Title: "t", Category: TicketCategoryAppeal, Body: "b",
	}, "")
	require.ErrorIs(t, err, ErrTicketInvalidCategory)
}

func TestAppealService_RestoreRevokesAndNotifies(t *testing.T) {
	ctx := context.Background()
	svc, store, _, messages, _ := newAppealTestService(&User{ID: 5, Status: StatusDisabled})
	_, _, err := svc.Issue(ctx, 5)
	require.NoError(t, err)

	svc.OnUserStatusChanged(ctx, 5, StatusActive, StatusDisabled) // 封禁：不发消息
	require.Empty(t, messages.snapshot())
	require.Len(t, store.records, 1)

	svc.OnUserStatusChanged(ctx, 5, StatusDisabled, StatusActive)
	require.Empty(t, store.records)
	msgs := messages.snapshot()
	require.Len(t, msgs, 1)
	require.Equal(t, SiteMessageCategorySystem, msgs[0].Category)
	require.Equal(t, SiteMessageSourceAppealRestore, msgs[0].SourceType)
	require.Equal(t, "账户已恢复 / Account restored", msgs[0].Title)
}

func TestAdminService_NotifiesStatusObserver(t *testing.T) {
	observer := &recordingStatusObserver{}
	svc := &adminServiceImpl{}
	svc.SetUserStatusObserver(observer)

	svc.notifyUserStatusChanged(context.Background(), 5, StatusDisabled, StatusDisabled)
	require.Empty(t, observer.calls, "no change, no notification")
	svc.notifyUserStatusChanged(context.Background(), 5, StatusDisabled, StatusActive)
	require.Equal(t, []string{"5:disabled->active"}, observer.calls)
}

func TestContentModerationUnbanNotifiesStatusObserver(t *testing.T) {
	observer := &recordingStatusObserver{}
	userRepo := &contentModerationTestUserRepo{user: &User{ID: 5, Status: StatusDisabled}}
	svc := NewContentModerationService(nil, nil, nil, nil, userRepo, nil, nil, nil)
	svc.SetUserStatusObserver(observer)

	_, err := svc.UnbanUser(context.Background(), 5)
	require.NoError(t, err)
	require.Equal(t, []string{"5:disabled->active"}, observer.calls)

	// 已经是 active：不重复通知
	_, err = svc.UnbanUser(context.Background(), 5)
	require.NoError(t, err)
	require.Len(t, observer.calls, 1)
}

type recordingStatusObserver struct {
	mu    sync.Mutex
	calls []string
}

func (o *recordingStatusObserver) OnUserStatusChanged(_ context.Context, userID int64, oldStatus, newStatus string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls = append(o.calls, strconv.FormatInt(userID, 10)+":"+oldStatus+"->"+newStatus)
}

func TestUserDisabledErrorUnwrapsToNotActive(t *testing.T) {
	err := userNotActiveError(&User{ID: 9, Email: "x@example.com", Status: StatusDisabled})
	require.ErrorIs(t, err, ErrUserNotActive)
	disabled, ok := AsUserDisabledError(err)
	require.True(t, ok)
	require.Equal(t, int64(9), disabled.UserID)

	require.Same(t, ErrUserNotActive, userNotActiveError(&User{ID: 9, Status: "pending"}))
	_, ok = AsUserDisabledError(ErrUserNotActive)
	require.False(t, ok)
}
