package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// userSiteMessageRepoMem 用户侧 handler 测试用的内存仓储。
type userSiteMessageRepoMem struct {
	mu       sync.Mutex
	messages []*service.SiteMessage
}

func (r *userSiteMessageRepoMem) Create(_ context.Context, msg *service.SiteMessage) (*service.SiteMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := *msg
	out.ID = int64(len(r.messages) + 1)
	r.messages = append(r.messages, &out)
	return &out, nil
}

func (r *userSiteMessageRepoMem) ListByUser(_ context.Context, userID int64, f service.SiteMessageFilter) ([]*service.SiteMessage, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*service.SiteMessage{}
	for _, m := range r.messages {
		if m.UserID == userID && (!f.UnreadOnly || m.ReadAt == nil) {
			clone := *m
			out = append(out, &clone)
		}
	}
	return out, int64(len(out)), nil
}

func (r *userSiteMessageRepoMem) CountUnread(_ context.Context, userID int64) (int64, error) {
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

func (r *userSiteMessageRepoMem) MarkRead(_ context.Context, userID, id int64, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.messages {
		if m.ID == id && m.UserID == userID {
			m.ReadAt = &now
			return nil
		}
	}
	return service.ErrSiteMessageNotFound
}

func (r *userSiteMessageRepoMem) MarkAllRead(_ context.Context, userID int64, now time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, m := range r.messages {
		if m.UserID == userID && m.ReadAt == nil {
			m.ReadAt = &now
			n++
		}
	}
	return n, nil
}

func newSiteMessageUserRouter(repo *userSiteMessageRepoMem, userID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewSiteMessageHandler(service.NewSiteMessageService(repo))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID > 0 {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID, Concurrency: 1})
		}
		c.Next()
	})
	router.GET("/site-messages", h.List)
	router.GET("/site-messages/unread-count", h.UnreadCount)
	router.POST("/site-messages/read-all", h.MarkAllRead)
	router.POST("/site-messages/:id/read", h.MarkRead)
	return router
}

func doSiteMessageRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestSiteMessageHandler_RequiresSubject(t *testing.T) {
	router := newSiteMessageUserRouter(&userSiteMessageRepoMem{}, 0)
	require.Equal(t, http.StatusUnauthorized, doSiteMessageRequest(router, http.MethodGet, "/site-messages").Code)
	require.Equal(t, http.StatusUnauthorized, doSiteMessageRequest(router, http.MethodGet, "/site-messages/unread-count").Code)
}

func TestSiteMessageHandler_InboxFlow(t *testing.T) {
	sender := int64(1)
	repo := &userSiteMessageRepoMem{messages: []*service.SiteMessage{
		{ID: 1, UserID: 5, Category: service.SiteMessageCategorySecurity, Title: "s", Content: "s", CreatedAt: time.Now()},
		{ID: 2, UserID: 5, Category: service.SiteMessageCategoryAdmin, Title: "a", Content: "a", SenderUserID: &sender, SenderEmail: "admin@example.com", CreatedAt: time.Now()},
		{ID: 3, UserID: 6, Category: service.SiteMessageCategoryAdmin, Title: "other", Content: "x", CreatedAt: time.Now()},
	}}
	router := newSiteMessageUserRouter(repo, 5)

	w := doSiteMessageRequest(router, http.MethodGet, "/site-messages/unread-count")
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"count":2}`, string(mustSiteMessageData(t, w)))

	w = doSiteMessageRequest(router, http.MethodGet, "/site-messages?unread_only=1")
	require.Equal(t, http.StatusOK, w.Code)
	require.NotContains(t, w.Body.String(), "admin@example.com", "users never see staff identity")
	require.Contains(t, w.Body.String(), `"from":"staff"`)
	require.Contains(t, w.Body.String(), `"from":"system"`)

	require.Equal(t, http.StatusNotFound, doSiteMessageRequest(router, http.MethodPost, "/site-messages/3/read").Code, "other users' messages look missing")
	require.Equal(t, http.StatusBadRequest, doSiteMessageRequest(router, http.MethodPost, "/site-messages/x/read").Code)
	require.Equal(t, http.StatusOK, doSiteMessageRequest(router, http.MethodPost, "/site-messages/1/read").Code)

	w = doSiteMessageRequest(router, http.MethodPost, "/site-messages/read-all")
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"updated":1}`, string(mustSiteMessageData(t, w)))
}

func mustSiteMessageData(t *testing.T, w *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Data
}
