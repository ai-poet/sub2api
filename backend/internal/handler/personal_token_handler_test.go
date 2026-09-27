//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type personalTokenHandlerUsers struct {
	users map[int64]*service.User
}

func (u *personalTokenHandlerUsers) GetByID(_ context.Context, id int64) (*service.User, error) {
	user, ok := u.users[id]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func newPersonalTokenHandlerRouter(t *testing.T, userID int64) (*gin.Engine, *testutil.MemoryPersonalTokenRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mk := func(id int64, role string) *service.User {
		u := &service.User{ID: id, Email: role + "@example.com", Role: role, Status: service.StatusActive, TokenVersion: 3, TokenVersionResolved: true}
		require.NoError(t, u.SetPassword("pw-123456"))
		return u
	}
	users := &personalTokenHandlerUsers{users: map[int64]*service.User{
		1: mk(1, service.RoleAdmin),
		2: mk(2, service.RoleOperator),
		3: mk(3, service.RoleUser),
	}}
	repo := testutil.NewMemoryPersonalTokenRepo()
	svc := service.NewPersonalTokenService(repo, users, testutil.NewStaticPersonalTokenSettings(true))
	h := NewPersonalTokenHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID, Concurrency: 1})
		c.Next()
	})
	router.GET("/api/v1/user/personal-token", h.GetStatus)
	router.POST("/api/v1/user/personal-token", h.Generate)
	router.DELETE("/api/v1/user/personal-token", h.Revoke)
	return router, repo
}

func servePersonalToken(router *gin.Engine, method, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/api/v1/user/personal-token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestPersonalTokenHandler_GenerateShowsPlaintextOnce(t *testing.T) {
	router, repo := newPersonalTokenHandlerRouter(t, 2)

	w := servePersonalToken(router, http.MethodPost, `{"password":"pw-123456","expires_in_days":90}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var resp struct {
		Data struct {
			Token string `json:"token"`
			Info  struct {
				Hint      string  `json:"hint"`
				ExpiresAt *string `json:"expires_at"`
			} `json:"info"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, service.IsPersonalTokenFormat(resp.Data.Token))
	require.NotNil(t, resp.Data.Info.ExpiresAt)
	stored, err := repo.GetByUserID(context.Background(), 2)
	require.NoError(t, err)
	require.NotContains(t, w.Body.String(), stored.TokenHash, "hash never leaves the server")

	// 之后的状态查询只有提示串，没有明文、没有哈希
	w = servePersonalToken(router, http.MethodGet, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.NotContains(t, w.Body.String(), resp.Data.Token)
	require.NotContains(t, w.Body.String(), stored.TokenHash)
	require.NotContains(t, w.Body.String(), "token_hash")
	require.Contains(t, w.Body.String(), resp.Data.Info.Hint)
	require.Contains(t, w.Body.String(), `"eligible":true`)

	w = servePersonalToken(router, http.MethodDelete, "")
	require.Equal(t, http.StatusOK, w.Code)
	w = servePersonalToken(router, http.MethodDelete, "")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestPersonalTokenHandler_GenerateRejections(t *testing.T) {
	for _, tc := range []struct {
		name   string
		userID int64
		body   string
		status int
		reason string
	}{
		{name: "admin_forbidden", userID: 1, body: `{"password":"pw-123456","expires_in_days":0}`, status: http.StatusForbidden, reason: "PERSONAL_TOKEN_NOT_ELIGIBLE"},
		{name: "user_forbidden", userID: 3, body: `{"password":"pw-123456","expires_in_days":0}`, status: http.StatusForbidden, reason: "PERSONAL_TOKEN_NOT_ELIGIBLE"},
		{name: "wrong_password", userID: 2, body: `{"password":"nope","expires_in_days":0}`, status: http.StatusBadRequest, reason: "PASSWORD_INCORRECT"},
		{name: "missing_expiry", userID: 2, body: `{"password":"pw-123456"}`, status: http.StatusBadRequest, reason: "PERSONAL_TOKEN_EXPIRY_INVALID"},
		{name: "bad_expiry", userID: 2, body: `{"password":"pw-123456","expires_in_days":3}`, status: http.StatusBadRequest, reason: "PERSONAL_TOKEN_EXPIRY_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, repo := newPersonalTokenHandlerRouter(t, tc.userID)
			w := servePersonalToken(router, http.MethodPost, tc.body)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), tc.reason)
			require.NotContains(t, w.Body.String(), "pat-")
			all, err := repo.List(context.Background())
			require.NoError(t, err)
			require.Empty(t, all)
		})
	}
}
