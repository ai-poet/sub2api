package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// siteMessageRepoMem 内存版站内信仓储；users 里的 id 才能收信（模拟 deleted_at IS NULL）。
type siteMessageRepoMem struct {
	mu       sync.Mutex
	seq      int64
	users    map[int64]bool
	messages []*service.SiteMessage
}

func (r *siteMessageRepoMem) Create(_ context.Context, msg *service.SiteMessage) (*service.SiteMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.users[msg.UserID] {
		return nil, service.ErrUserNotFound
	}
	r.seq++
	out := *msg
	out.ID = r.seq
	r.messages = append(r.messages, &out)
	clone := out
	return &clone, nil
}

func (r *siteMessageRepoMem) ListByUser(_ context.Context, userID int64, f service.SiteMessageFilter) ([]*service.SiteMessage, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*service.SiteMessage{}
	for _, m := range r.messages {
		if m.UserID == userID {
			clone := *m
			out = append(out, &clone)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, int64(len(out)), nil
}

func (r *siteMessageRepoMem) CountUnread(context.Context, int64) (int64, error) { return 0, nil }
func (r *siteMessageRepoMem) MarkRead(context.Context, int64, int64, time.Time) error {
	return nil
}
func (r *siteMessageRepoMem) MarkAllRead(context.Context, int64, time.Time) (int64, error) {
	return 0, nil
}

func newUserSiteMessageRouter(repo *siteMessageRepoMem, replay *service.ApprovalReplay) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewUserSiteMessageHandler(service.NewSiteMessageService(repo))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, Concurrency: 1})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		if replay != nil {
			c.Request = c.Request.WithContext(service.WithApprovalReplay(c.Request.Context(), replay))
		}
		c.Next()
	})
	router.POST("/api/v1/admin/users/:id/site-messages", h.Send)
	router.GET("/api/v1/admin/users/:id/site-messages", h.List)
	return router
}

func postSiteMessage(t *testing.T, router *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestUserSiteMessageHandler_AdminSends(t *testing.T) {
	repo := &siteMessageRepoMem{users: map[int64]bool{9: true}}
	router := newUserSiteMessageRouter(repo, nil)

	w := postSiteMessage(t, router, "/api/v1/admin/users/9/site-messages", map[string]string{"title": "你好", "content": "**正文**"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID           int64  `json:"id"`
			UserID       int64  `json:"user_id"`
			Category     string `json:"category"`
			From         string `json:"from"`
			SenderUserID *int64 `json:"sender_user_id"`
			SenderRole   string `json:"sender_role"`
			ApprovalID   *int64 `json:"approval_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, int64(9), resp.Data.UserID)
	require.Equal(t, service.SiteMessageCategoryAdmin, resp.Data.Category)
	require.Equal(t, "staff", resp.Data.From)
	require.Equal(t, int64(1), *resp.Data.SenderUserID)
	require.Equal(t, service.RoleAdmin, resp.Data.SenderRole)
	require.Nil(t, resp.Data.ApprovalID)
}

func TestUserSiteMessageHandler_ReplayRecordsOperator(t *testing.T) {
	repo := &siteMessageRepoMem{users: map[int64]bool{9: true}}
	router := newUserSiteMessageRouter(repo, &service.ApprovalReplay{ApprovalID: 33, RequesterUserID: 7, ApproverUserID: 1, ApproverRole: service.RoleAdmin})

	w := postSiteMessage(t, router, "/api/v1/admin/users/9/site-messages", map[string]string{"title": "t", "content": "c"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Len(t, repo.messages, 1)
	stored := repo.messages[0]
	require.Equal(t, int64(7), *stored.SenderUserID, "the operator who asked is the sender, not the approving admin")
	require.Equal(t, service.RoleOperator, stored.SenderRole)
	require.Equal(t, int64(33), *stored.ApprovalID)
	require.Equal(t, "approval:33", stored.SourceID)
}

func TestUserSiteMessageHandler_Errors(t *testing.T) {
	repo := &siteMessageRepoMem{users: map[int64]bool{9: true}}
	router := newUserSiteMessageRouter(repo, nil)

	w := postSiteMessage(t, router, "/api/v1/admin/users/abc/site-messages", map[string]string{"title": "t", "content": "c"})
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = postSiteMessage(t, router, "/api/v1/admin/users/404/site-messages", map[string]string{"title": "t", "content": "c"})
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "USER_NOT_FOUND")

	w = postSiteMessage(t, router, "/api/v1/admin/users/9/site-messages", map[string]string{"title": "", "content": "c"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "SITE_MESSAGE_TITLE_INVALID")

	require.Empty(t, repo.messages)
}

func TestUserSiteMessageHandler_ListHistory(t *testing.T) {
	repo := &siteMessageRepoMem{users: map[int64]bool{9: true}}
	router := newUserSiteMessageRouter(repo, nil)
	postSiteMessage(t, router, "/api/v1/admin/users/9/site-messages", map[string]string{"title": "t1", "content": "c"})
	postSiteMessage(t, router, "/api/v1/admin/users/9/site-messages", map[string]string{"title": "t2", "content": "c"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/9/site-messages?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			Items []struct {
				Title string `json:"title"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, int64(2), resp.Data.Total)
	require.Equal(t, "t2", resp.Data.Items[0].Title)
}
