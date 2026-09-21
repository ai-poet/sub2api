package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
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

func (r *ticketRepoMem) StaffAttachmentReferencedForUser(_ context.Context, userID int64, key string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	needle := service.TicketAttachmentURLScheme + key
	for id, item := range r.tickets {
		if item.UserID != userID {
			continue
		}
		for _, m := range r.messages[id] {
			if m.AuthorRole != service.TicketAuthorRoleUser && strings.Contains(m.Body, needle) {
				return true, nil
			}
		}
	}
	return false, nil
}

// ---------- 测试骨架 ----------

type ticketTestIdentity struct {
	userID int64
	email  string
	role   string
}

func newTicketTestRouter(t *testing.T, repo *ticketRepoMem, identity ticketTestIdentity) (*gin.Engine, *service.TicketService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := service.NewTicketService(repo)
	h := NewTicketHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: identity.userID, Concurrency: 1})
		c.Set(string(middleware.ContextKeyUserRole), identity.role)
		c.Set(middleware.ContextKeyAuthEmail, identity.email)
		c.Next()
	})
	router.GET("/admin/tickets", h.List)
	router.GET("/admin/tickets/open-count", h.OpenCount)
	router.GET("/admin/tickets/:id", h.Get)
	router.POST("/admin/tickets/:id/messages", h.Reply)
	router.POST("/admin/tickets/:id/close", h.Close)
	router.POST("/admin/tickets/:id/reopen", h.Reopen)
	return router, svc
}

func seedTicket(t *testing.T, svc *service.TicketService, userID int64, email, title string) *service.SupportTicket {
	t.Helper()
	created, err := svc.Create(context.Background(), service.TicketActor{UserID: userID, Email: email, Role: service.RoleUser},
		service.TicketCreateInput{Title: title, Category: service.TicketCategoryBilling, Body: "body of " + title}, "")
	require.NoError(t, err)
	return created
}

func doTicket(router *gin.Engine, method, target string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, target, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeTicketJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), rec.Body.String())
	return out
}

func TestAdminTicketHandler_OperatorSeesAllAndRepliesDirectly(t *testing.T) {
	repo := newTicketRepoMem()
	router, svc := newTicketTestRouter(t, repo, ticketTestIdentity{userID: 9, email: "ops@example.com", role: service.RoleOperator})
	seedTicket(t, svc, 1, "alice@example.com", "alpha")
	second := seedTicket(t, svc, 2, "bob@example.com", "beta")

	rec := doTicket(router, http.MethodGet, "/admin/tickets", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := decodeTicketJSON(t, rec)["data"].(map[string]any)
	require.EqualValues(t, 2, data["total"])
	items := data["items"].([]any)
	first := items[0].(map[string]any)
	require.Equal(t, "bob@example.com", first["user"].(map[string]any)["email"], "客服视图带发起人")

	rec = doTicket(router, http.MethodGet, "/admin/tickets/open-count", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.EqualValues(t, 2, decodeTicketJSON(t, rec)["data"].(map[string]any)["count"])

	rec = doTicket(router, http.MethodPost, "/admin/tickets/"+itoa(second.ID)+"/messages", map[string]any{"body": "we are looking into it"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	reply := decodeTicketJSON(t, rec)["data"].(map[string]any)
	require.Equal(t, service.TicketStatusReplied, reply["ticket"].(map[string]any)["status"])
	require.Equal(t, true, reply["ticket"].(map[string]any)["user_unread"])
	msg := reply["message"].(map[string]any)
	require.Equal(t, service.RoleOperator, msg["author_role"])
	require.Equal(t, "ops@example.com", msg["author_email"])

	rec = doTicket(router, http.MethodGet, "/admin/tickets/"+itoa(second.ID), nil)
	require.Equal(t, http.StatusOK, rec.Code)
	detail := decodeTicketJSON(t, rec)["data"].(map[string]any)
	msgs := detail["messages"].([]any)
	require.Len(t, msgs, 2)
	require.Equal(t, service.RoleUser, msgs[0].(map[string]any)["author_role"])
	require.Equal(t, "bob@example.com", detail["ticket"].(map[string]any)["user"].(map[string]any)["email"], "detail belongs to bob")

	rec = doTicket(router, http.MethodGet, "/admin/tickets/open-count", nil)
	require.EqualValues(t, 1, decodeTicketJSON(t, rec)["data"].(map[string]any)["count"], "回复后待处理数减一")
}

func TestAdminTicketHandler_CloseReopenAndConflicts(t *testing.T) {
	repo := newTicketRepoMem()
	router, svc := newTicketTestRouter(t, repo, ticketTestIdentity{userID: 8, email: "admin@example.com", role: service.RoleAdmin})
	ticket := seedTicket(t, svc, 1, "alice@example.com", "close me")
	base := "/admin/tickets/" + itoa(ticket.ID)

	rec := doTicket(router, http.MethodPost, base+"/close", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	closed := decodeTicketJSON(t, rec)["data"].(map[string]any)
	require.Equal(t, service.TicketStatusClosed, closed["status"])
	require.Equal(t, service.RoleAdmin, closed["closed_by_role"])

	rec = doTicket(router, http.MethodPost, base+"/messages", map[string]any{"body": "too late"})
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "TICKET_CLOSED", decodeTicketJSON(t, rec)["reason"])

	rec = doTicket(router, http.MethodPost, base+"/close", nil)
	require.Equal(t, http.StatusConflict, rec.Code)

	rec = doTicket(router, http.MethodPost, base+"/reopen", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.TicketStatusOpen, decodeTicketJSON(t, rec)["data"].(map[string]any)["status"])

	rec = doTicket(router, http.MethodPost, base+"/reopen", nil)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "TICKET_NOT_CLOSED", decodeTicketJSON(t, rec)["reason"])

	rec = doTicket(router, http.MethodPost, base+"/messages", map[string]any{"body": ""})
	require.Equal(t, http.StatusBadRequest, rec.Code, "空正文被 binding 拦下")
}

func TestAdminTicketHandler_BadIDsAndNotFound(t *testing.T) {
	repo := newTicketRepoMem()
	router, _ := newTicketTestRouter(t, repo, ticketTestIdentity{userID: 8, email: "admin@example.com", role: service.RoleAdmin})

	rec := doTicket(router, http.MethodGet, "/admin/tickets/abc", nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doTicket(router, http.MethodGet, "/admin/tickets/999", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
	rec = doTicket(router, http.MethodPost, "/admin/tickets/999/close", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

	// 非 staff 角色即使打到 handler 也拿不到数据（认证层本来就会拦）
	userRouter, _ := newTicketTestRouter(t, repo, ticketTestIdentity{userID: 1, email: "alice@example.com", role: service.RoleUser})
	rec = doTicket(userRouter, http.MethodGet, "/admin/tickets", nil)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
