//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 用户侧（jwtAuth）接口上的运维个人令牌：默认可用，只挡账号安全写操作（fork 本地）。
// 全部用户侧路由的逐条检查见 routes/personal_token_scope_equivalence_test.go。

func newPersonalTokenUserRouter(t *testing.T, f *personalTokenAuthFixture) (*gin.Engine, map[string]int) {
	t.Helper()
	tokens := service.NewPersonalTokenService(f.repo, f.userSvc, f.settings)
	reached := map[string]int{}
	router := gin.New()
	// 与 NewJWTAuthMiddlewareWithPersonalTokens 相同，只是不接活跃时间写入（桩仓储不支持）
	router.Use(jwtAuthWithPersonalTokens(f.auth, f.userSvc, nil, nil, f.audit, PersonalTokenAuthenticatorOrNil(tokens)))
	echo := func(c *gin.Context) {
		reached[c.Request.Method+" "+c.FullPath()]++
		role, _ := GetUserRoleFromContext(c)
		subject, _ := GetAuthSubjectFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"role": role, "actor": subject.UserID,
			"auth_method": c.GetString("auth_method"), "session_id": c.GetString(ContextKeySessionID),
		})
	}
	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/group-status"},
		{http.MethodGet, "/api/v1/group-status/:id/history"},
		{http.MethodGet, "/api/v1/keys"},
		{http.MethodPost, "/api/v1/keys"},
		{http.MethodGet, "/api/v1/user/personal-token"},
		{http.MethodPost, "/api/v1/user/personal-token"},
		{http.MethodDelete, "/api/v1/user/personal-token"},
		{http.MethodPut, "/api/v1/user/password"},
		{http.MethodPost, "/api/v1/user/totp/disable"},
		{http.MethodDelete, "/api/v1/user/passkeys/:id"},
		{http.MethodPost, "/api/v1/user/account-bindings/email"},
		{http.MethodPost, "/api/v1/auth/desktop-session"},
		{http.MethodPost, "/api/v1/auth/revoke-all-sessions"},
	} {
		router.Handle(r.method, r.path, echo)
	}
	return router, reached
}

func TestPersonalTokenOnUserRoutes(t *testing.T) {
	f := newPersonalTokenAuthFixture(t)
	router, reached := newPersonalTokenUserRouter(t, f)
	opsToken := f.issue(t, 2)

	serve := func(method, path, token string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("own_account_reads_and_writes_pass_as_operator_without_session", func(t *testing.T) {
		for _, tc := range []struct{ method, path string }{
			{http.MethodGet, "/api/v1/auth/me"},
			{http.MethodGet, "/api/v1/group-status"},
			{http.MethodGet, "/api/v1/group-status/3/history"},
			{http.MethodGet, "/api/v1/keys"},
			{http.MethodPost, "/api/v1/keys"},
			{http.MethodGet, "/api/v1/user/personal-token"},
		} {
			w := serve(tc.method, tc.path, opsToken)
			require.Equalf(t, http.StatusOK, w.Code, "%s %s: %s", tc.method, tc.path, w.Body.String())
			require.Contains(t, w.Body.String(), `"role":"operator"`)
			require.Contains(t, w.Body.String(), `"actor":2`)
			require.Contains(t, w.Body.String(), `"auth_method":"personal_token"`)
			require.Contains(t, w.Body.String(), `"session_id":""`)
		}
	})

	t.Run("account_security_writes_are_forbidden", func(t *testing.T) {
		for _, tc := range []struct{ method, path, template string }{
			{http.MethodPost, "/api/v1/user/personal-token", "POST /api/v1/user/personal-token"},
			{http.MethodDelete, "/api/v1/user/personal-token", "DELETE /api/v1/user/personal-token"},
			{http.MethodPut, "/api/v1/user/password", "PUT /api/v1/user/password"},
			{http.MethodPost, "/api/v1/user/totp/disable", "POST /api/v1/user/totp/disable"},
			{http.MethodDelete, "/api/v1/user/passkeys/1", "DELETE /api/v1/user/passkeys/:id"},
			{http.MethodPost, "/api/v1/user/account-bindings/email", "POST /api/v1/user/account-bindings/email"},
			{http.MethodPost, "/api/v1/auth/desktop-session", "POST /api/v1/auth/desktop-session"},
			{http.MethodPost, "/api/v1/auth/revoke-all-sessions", "POST /api/v1/auth/revoke-all-sessions"},
		} {
			w := serve(tc.method, tc.path, opsToken)
			require.Equalf(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
			require.Contains(t, w.Body.String(), "PERSONAL_TOKEN_ROUTE_FORBIDDEN")
			require.Zerof(t, reached[tc.template], "handler reached for %s", tc.template)
		}
	})

	t.Run("non_operator_tokens_and_bad_tokens_rejected", func(t *testing.T) {
		for _, token := range []string{f.issue(t, 1), f.issue(t, 3)} {
			w := serve(http.MethodGet, "/api/v1/auth/me", token)
			require.Equal(t, http.StatusUnauthorized, w.Code)
			require.Contains(t, w.Body.String(), "PERSONAL_TOKEN_NOT_ELIGIBLE")
		}
		w := serve(http.MethodGet, "/api/v1/auth/me", "pat-"+strings.Repeat("0", 64))
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "INVALID_TOKEN")

		f.settings.Set(false)
		w = serve(http.MethodGet, "/api/v1/auth/me", opsToken)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "PERSONAL_TOKEN_DISABLED")
		f.settings.Set(true)
	})

	t.Run("without_authenticator_fails_closed", func(t *testing.T) {
		plain := gin.New()
		plain.Use(gin.HandlerFunc(NewJWTAuthMiddleware(f.auth, f.userSvc, nil, nil)))
		plain.GET("/api/v1/auth/me", func(c *gin.Context) { c.Status(http.StatusOK) })
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+opsToken)
		plain.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 被挡下的账号安全操作留审计：凭证类型是令牌，只留掩码
	f.audit.Stop()
	denied := 0
	for _, entry := range f.auditRepo.snapshot() {
		if entry.Action != service.AuditActionAdminScopeDenied {
			continue
		}
		denied++
		require.Equal(t, service.AuditAuthMethodPersonalToken, entry.AuthMethod)
		require.NotContains(t, entry.CredentialMasked, opsToken)
	}
	require.Equal(t, 8, denied)
}

func TestPersonalTokenUserRouteAllows(t *testing.T) {
	require.True(t, PersonalTokenUserRouteAllows(http.MethodGet, "/api/v1/auth/me"))
	require.True(t, PersonalTokenUserRouteAllows(http.MethodGet, "/api/v1/user/totp/status"), "reads are always allowed")
	require.True(t, PersonalTokenUserRouteAllows(http.MethodPost, "/api/v1/keys"))
	require.True(t, PersonalTokenUserRouteAllows(http.MethodPut, "/api/v1/user"), "profile update is not an account security action")
	require.False(t, PersonalTokenUserRouteAllows(http.MethodPost, "/api/v1/user/totp/some-future-route"), "prefix covers future writes")
	require.False(t, PersonalTokenUserRouteAllows(http.MethodPost, "/api/v1/auth/oauth/bind-token"))
	require.False(t, PersonalTokenUserRouteAllows(http.MethodGet, ""), "unmatched routes are denied")
}
