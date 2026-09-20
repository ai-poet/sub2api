//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// ---------- 内存仓储 ----------

type ticketRepoMem struct {
	mu       sync.Mutex
	seq      int64
	msgSeq   int64
	tickets  map[int64]*service.SupportTicket
	messages map[int64][]*service.SupportTicketMessage
}

func newTicketRepoMem() *ticketRepoMem {
	return &ticketRepoMem{tickets: map[int64]*service.SupportTicket{}, messages: map[int64][]*service.SupportTicketMessage{}}
}

func (r *ticketRepoMem) Create(_ context.Context, ticket *service.SupportTicket, first *service.SupportTicketMessage) (*service.SupportTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	clone := *ticket
	clone.ID = r.seq
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

func (r *ticketRepoMem) GetByID(_ context.Context, id int64) (*service.SupportTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.tickets[id]
	if !ok {
		return nil, service.ErrTicketNotFound
	}
	out := *item
	return &out, nil
}

func (r *ticketRepoMem) List(_ context.Context, f *service.TicketFilter) ([]*service.SupportTicket, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*service.SupportTicket, 0)
	for _, item := range r.tickets {
		if f != nil {
			if f.UserID != nil && item.UserID != *f.UserID {
				continue
			}
			if f.Status != "" && item.Status != f.Status {
				continue
			}
		}
		clone := *item
		out = append(out, &clone)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, int64(len(out)), nil
}

func (r *ticketRepoMem) ListMessages(_ context.Context, ticketID int64) ([]*service.SupportTicketMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*service.SupportTicketMessage, 0)
	for _, m := range r.messages[ticketID] {
		c := *m
		out = append(out, &c)
	}
	return out, nil
}

func (r *ticketRepoMem) AppendMessage(_ context.Context, msg *service.SupportTicketMessage, newStatus string, userUnread bool, now time.Time) (*service.SupportTicket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.tickets[msg.TicketID]
	if !ok {
		return nil, service.ErrTicketNotFound
	}
	if item.Status == service.TicketStatusClosed {
		return nil, service.ErrTicketClosed
	}
	r.msgSeq++
	msg.ID = r.msgSeq
	m := *msg
	r.messages[msg.TicketID] = append(r.messages[msg.TicketID], &m)
	item.Status = newStatus
	item.UserUnread = userUnread
	item.LastMessageAt = now
	item.MessageCount++
	out := *item
	return &out, nil
}

func (r *ticketRepoMem) Transition(_ context.Context, id int64, from []string, to string, actor service.TicketActor, now time.Time) (bool, error) {
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
		}
	}
	if !matched {
		return false, nil
	}
	item.Status = to
	if to == service.TicketStatusClosed {
		at := now
		item.ClosedAt = &at
		item.ClosedByRole = actor.Role
	} else {
		item.ClosedAt = nil
		item.ClosedByRole = ""
	}
	return true, nil
}

func (r *ticketRepoMem) MarkUserRead(_ context.Context, id, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item, ok := r.tickets[id]; ok && item.UserID == userID {
		item.UserUnread = false
	}
	return nil
}

func (r *ticketRepoMem) CountByStatus(_ context.Context, status string) (int64, error) {
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

func (r *ticketRepoMem) CountActiveByUser(_ context.Context, userID int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.tickets {
		if item.UserID == userID && item.Status != service.TicketStatusClosed {
			n++
		}
	}
	return n, nil
}

func (r *ticketRepoMem) CountUserUnread(_ context.Context, userID int64) (int64, error) {
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

// ---------- 测试骨架 ----------

func newTicketUserRouter(t *testing.T, svc *service.TicketService, userID int64, email string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := NewTicketHandler(svc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID, Concurrency: 1})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleUser)
		c.Set(middleware.ContextKeyAuthEmail, email)
		c.Next()
	})
	router.GET("/tickets", h.List)
	router.GET("/tickets/unread-count", h.UnreadCount)
	router.POST("/tickets", h.Create)
	router.GET("/tickets/:id", h.Get)
	router.POST("/tickets/:id/messages", h.Reply)
	router.POST("/tickets/:id/close", h.Close)
	router.POST("/tickets/:id/reopen", h.Reopen)
	return router
}

func doTicketReq(router *gin.Engine, method, target string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, target, &buf)
	req.Host = "console.example.com"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeTicketBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return out
}

func ticketPath(id any, suffix string) string {
	return "/tickets/" + strconv.FormatInt(int64(id.(float64)), 10) + suffix
}

func TestTicketHandler_CreateListGetOwnOnly(t *testing.T) {
	svc := service.NewTicketService(newTicketRepoMem())
	alice := newTicketUserRouter(t, svc, 1, "alice@example.com")
	bob := newTicketUserRouter(t, svc, 2, "bob@example.com")

	rec := doTicketReq(alice, http.MethodPost, "/tickets", map[string]any{"title": "Cannot call API", "category": "api", "body": "**429** all the time"})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	created := decodeTicketBody(t, rec)["data"].(map[string]any)
	require.Equal(t, service.TicketStatusOpen, created["status"])
	require.EqualValues(t, 1, created["message_count"])
	_, hasUser := created["user"]
	require.False(t, hasUser, "用户视图不带发起人引用")

	rec = doTicketReq(alice, http.MethodGet, "/tickets", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	list := decodeTicketBody(t, rec)["data"].(map[string]any)
	require.EqualValues(t, 1, list["total"])

	rec = doTicketReq(bob, http.MethodGet, "/tickets", nil)
	require.EqualValues(t, 0, decodeTicketBody(t, rec)["data"].(map[string]any)["total"], "只看自己的")

	rec = doTicketReq(alice, http.MethodGet, ticketPath(created["id"], ""), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	detail := decodeTicketBody(t, rec)["data"].(map[string]any)
	msgs := detail["messages"].([]any)
	require.Len(t, msgs, 1)
	first := msgs[0].(map[string]any)
	require.Equal(t, service.RoleUser, first["author_role"])
	_, hasEmail := first["author_email"]
	require.False(t, hasEmail, "用户视图不带邮箱")

	rec = doTicketReq(bob, http.MethodGet, ticketPath(created["id"], ""), nil)
	require.Equal(t, http.StatusNotFound, rec.Code, "他人工单按不存在处理")
	rec = doTicketReq(bob, http.MethodPost, ticketPath(created["id"], "/messages"), map[string]any{"body": "hijack"})
	require.Equal(t, http.StatusNotFound, rec.Code)
	rec = doTicketReq(bob, http.MethodPost, ticketPath(created["id"], "/close"), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTicketHandler_ReplyCloseReopenAndUnread(t *testing.T) {
	svc := service.NewTicketService(newTicketRepoMem())
	alice := newTicketUserRouter(t, svc, 1, "alice@example.com")

	rec := doTicketReq(alice, http.MethodPost, "/tickets", map[string]any{"title": "billing", "category": "billing", "body": "charged twice"})
	require.Equal(t, http.StatusCreated, rec.Code)
	id := decodeTicketBody(t, rec)["data"].(map[string]any)["id"]

	// 客服回复 → 用户未读 1；打开详情后归零，客服消息归一为 staff 且无邮箱
	_, _, err := svc.ReplyAsStaff(context.Background(), int64(id.(float64)), service.TicketActor{UserID: 9, Email: "ops@example.com", Role: service.RoleOperator}, "refunded")
	require.NoError(t, err)
	rec = doTicketReq(alice, http.MethodGet, "/tickets/unread-count", nil)
	require.EqualValues(t, 1, decodeTicketBody(t, rec)["data"].(map[string]any)["count"])

	rec = doTicketReq(alice, http.MethodGet, ticketPath(id, ""), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	detail := decodeTicketBody(t, rec)["data"].(map[string]any)
	require.Equal(t, false, detail["ticket"].(map[string]any)["user_unread"])
	staffMsg := detail["messages"].([]any)[1].(map[string]any)
	require.Equal(t, "staff", staffMsg["author_role"])
	_, hasEmail := staffMsg["author_email"]
	require.False(t, hasEmail)
	require.NotContains(t, rec.Body.String(), "ops@example.com")

	rec = doTicketReq(alice, http.MethodGet, "/tickets/unread-count", nil)
	require.EqualValues(t, 0, decodeTicketBody(t, rec)["data"].(map[string]any)["count"])

	// 用户回复 → open
	rec = doTicketReq(alice, http.MethodPost, ticketPath(id, "/messages"), map[string]any{"body": "thanks"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	reply := decodeTicketBody(t, rec)["data"].(map[string]any)
	require.Equal(t, service.TicketStatusOpen, reply["ticket"].(map[string]any)["status"])
	require.Equal(t, service.RoleUser, reply["message"].(map[string]any)["author_role"])

	// 关闭 → 回复 409 → 重开 → 再重开 409
	rec = doTicketReq(alice, http.MethodPost, ticketPath(id, "/close"), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.TicketStatusClosed, decodeTicketBody(t, rec)["data"].(map[string]any)["status"])
	rec = doTicketReq(alice, http.MethodPost, ticketPath(id, "/messages"), map[string]any{"body": "again"})
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "TICKET_CLOSED", decodeTicketBody(t, rec)["reason"])
	rec = doTicketReq(alice, http.MethodPost, ticketPath(id, "/reopen"), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doTicketReq(alice, http.MethodPost, ticketPath(id, "/reopen"), nil)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "TICKET_NOT_CLOSED", decodeTicketBody(t, rec)["reason"])
}

func TestTicketHandler_CreateValidationAndLimit(t *testing.T) {
	svc := service.NewTicketService(newTicketRepoMem())
	alice := newTicketUserRouter(t, svc, 1, "alice@example.com")

	rec := doTicketReq(alice, http.MethodPost, "/tickets", map[string]any{"title": "x", "category": "weird", "body": "y"})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doTicketReq(alice, http.MethodPost, "/tickets", map[string]any{"title": "", "category": "api", "body": "y"})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doTicketReq(alice, http.MethodGet, "/tickets/abc", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	for i := 0; i < service.TicketOpenLimitPerUser; i++ {
		rec = doTicketReq(alice, http.MethodPost, "/tickets", map[string]any{"title": "t", "category": "other", "body": "y"})
		require.Equal(t, http.StatusCreated, rec.Code)
	}
	rec = doTicketReq(alice, http.MethodPost, "/tickets", map[string]any{"title": "t", "category": "other", "body": "y"})
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "TICKET_OPEN_LIMIT", decodeTicketBody(t, rec)["reason"])
}
