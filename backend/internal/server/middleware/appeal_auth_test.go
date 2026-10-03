//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type appealAuthStub struct {
	tokens map[string]*service.AppealSession
	err    error
}

func (s *appealAuthStub) Authenticate(_ context.Context, token string) (*service.AppealSession, error) {
	if s.err != nil {
		return nil, s.err
	}
	if sess, ok := s.tokens[token]; ok {
		return sess, nil
	}
	return nil, service.ErrAppealTokenInvalid
}

func newAppealAuthRouter(auth AppealAuthenticator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AppealAuth(auth))
	handler := func(c *gin.Context) {
		subject, _ := GetAuthSubjectFromContext(c)
		role, _ := GetUserRoleFromContext(c)
		sess, _ := GetAppealSessionFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":     subject.UserID,
			"role":        role,
			"email":       c.GetString(ContextKeyAuthEmail),
			"auth_method": c.GetString("auth_method"),
			"has_session": sess != nil,
		})
	}
	r.GET("/api/v1/appeal/session", handler)
	r.GET("/api/v1/appeal/tickets/:id", handler)
	// 不在白名单里的路由，即使挂在 appeal 组下也必须拒绝
	r.POST("/api/v1/appeal/tickets/:id/close", handler)
	return r
}

func TestAppealAuth_OnlyAcceptsHeaderToken(t *testing.T) {
	auth := &appealAuthStub{tokens: map[string]*service.AppealSession{
		"apl_ok": {UserID: 5, Email: "u@example.com", Role: service.RoleOperator, ExpiresAt: time.Now().Add(time.Hour)},
	}}
	router := newAppealAuthRouter(auth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/appeal/session", nil)
	req.Header.Set(service.AppealTokenHeader, "apl_ok")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"user_id":5,"role":"user","email":"u@example.com","auth_method":"appeal_token","has_session":true}`, w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	// Authorization 头里的申诉令牌不被认可
	req = httptest.NewRequest(http.MethodGet, "/api/v1/appeal/session", nil)
	req.Header.Set("Authorization", "Bearer apl_ok")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "APPEAL_TOKEN_INVALID")
}

func TestAppealAuth_DefaultDenyScope(t *testing.T) {
	router := newAppealAuthRouter(&appealAuthStub{tokens: map[string]*service.AppealSession{"apl_ok": {UserID: 5}}})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/appeal/tickets/1/close", nil)
	req.Header.Set(service.AppealTokenHeader, "apl_ok")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "APPEAL_ROUTE_FORBIDDEN")
}

func TestAppealAuth_RestoredAccountGets409(t *testing.T) {
	router := newAppealAuthRouter(&appealAuthStub{err: service.ErrAppealAccountActive})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/appeal/session", nil)
	req.Header.Set(service.AppealTokenHeader, "apl_any")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "APPEAL_ACCOUNT_ACTIVE")
}

func TestAppealAuth_NilAuthenticatorFailsClosed(t *testing.T) {
	router := newAppealAuthRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/appeal/session", nil)
	req.Header.Set(service.AppealTokenHeader, "apl_any")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestAppealScopeAllows(t *testing.T) {
	require.True(t, AppealScopeAllows(http.MethodGet, "/api/v1/appeal/session"))
	require.True(t, AppealScopeAllows(http.MethodHead, "/api/v1/appeal/session"))
	require.True(t, AppealScopeAllows("post", "/api/v1/appeal/tickets"))
	require.False(t, AppealScopeAllows(http.MethodDelete, "/api/v1/appeal/tickets/:id"))
	require.False(t, AppealScopeAllows(http.MethodGet, "/api/v1/user/profile"))
	require.False(t, AppealScopeAllows(http.MethodPost, ""))
	require.IsIncreasing(t, AppealScopeRoutes())
}

// 申诉令牌永远换不成普通会话：jwtAuth 把它当作无效 JWT 拒绝。
func TestJWTAuth_RejectsAppealToken(t *testing.T) {
	user := &service.User{ID: 1, Email: "u@example.com", Role: "user", Status: service.StatusDisabled}
	router, _ := newJWTTestEnv(map[int64]*service.User{1: user})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer apl_"+"abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
