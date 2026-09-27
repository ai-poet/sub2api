//go:build unit

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 运维个人令牌在认证层的越权防线（fork 本地，见 docs/PERSONAL_TOKENS.md「防越权保证」）。
// 全部 adminAuth 路由的「JWT vs 令牌」逐条等价比对见 routes/personal_token_scope_equivalence_test.go。

type personalTokenAuthFixture struct {
	router    *gin.Engine
	auth      *service.AuthService
	users     map[int64]*service.User
	userSvc   *service.UserService
	repo      *testutil.MemoryPersonalTokenRepo
	settings  *testutil.StaticPersonalTokenSettings
	gate      *fakeApprovalGate
	audit     *service.AuditLogService
	auditRepo *capturingAuditRepo
	reached   map[string]int
}

func newPersonalTokenAuthFixture(t *testing.T) *personalTokenAuthFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	users := map[int64]*service.User{
		1: {ID: 1, Email: "admin@example.com", Role: service.RoleAdmin, Status: service.StatusActive, TokenVersion: 1, Concurrency: 3},
		2: {ID: 2, Email: "ops@example.com", Role: service.RoleOperator, Status: service.StatusActive, TokenVersion: 1, Concurrency: 1},
		3: {ID: 3, Email: "user@example.com", Role: service.RoleUser, Status: service.StatusActive, TokenVersion: 1, Concurrency: 1},
	}
	userRepo := &stubUserRepo{
		getByID: func(_ context.Context, id int64) (*service.User, error) {
			u, ok := users[id]
			if !ok {
				return nil, service.ErrUserNotFound
			}
			clone := *u
			return &clone, nil
		},
	}
	userService := service.NewUserService(userRepo, nil, nil, nil)
	repo := testutil.NewMemoryPersonalTokenRepo()
	settings := testutil.NewStaticPersonalTokenSettings(true)
	tokens := service.NewPersonalTokenService(repo, userService, settings)

	auditRepo := &capturingAuditRepo{}
	auditService := service.NewAuditLogService(auditRepo, nil)
	auditService.Start()
	t.Cleanup(auditService.Stop)

	gate := &fakeApprovalGate{}
	f := &personalTokenAuthFixture{
		auth: authService, users: users, userSvc: userService, repo: repo, settings: settings,
		gate: gate, audit: auditService, auditRepo: auditRepo, reached: map[string]int{},
	}

	router := gin.New()
	router.Use(gin.HandlerFunc(NewConsoleAdminAuthMiddleware(authService, userService, nil, auditService, gate, PersonalTokenAuthenticatorOrNil(tokens))))
	echo := func(c *gin.Context) {
		f.reached[c.Request.Method+" "+c.FullPath()]++
		role, _ := GetUserRoleFromContext(c)
		subject, _ := GetAuthSubjectFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"role": role, "actor": subject.UserID,
			"auth_method": c.GetString("auth_method"), "session_id": c.GetString(ContextKeySessionID),
		})
	}
	router.GET("/api/v1/admin/ops/errors/:id", echo)
	router.GET("/api/v1/admin/settings", echo)
	router.GET("/api/v1/admin/accounts", echo)
	router.GET("/api/v1/admin/personal-tokens", echo)
	router.POST("/api/v1/admin/users/:id/balance", echo)
	router.DELETE("/api/v1/admin/users/:id", echo)
	router.POST("/api/v1/admin/approvals/:id/approve", echo)
	router.POST("/api/v1/admin/tickets/:id/messages", echo)
	router.GET("/api/v1/admin/ops/ws/qps", echo)
	// step-up 路由：即使 operator 被放进白名单，令牌也过不了 step-up（开关关闭时也不行）
	router.GET("/api/v1/admin/stepup-probe", stepUpAuth(stubStepUpGrantChecker{granted: true},
		stubStepUpUserReader{user: &service.User{ID: 2, TotpEnabled: true}}, stubStepUpSettingReader{enabled: false}), echo)
	f.router = router
	return f
}

// issue 直接往仓储里放一个属于 userID 的令牌（模拟任何来源的令牌，包括不该存在的 admin 令牌）。
func (f *personalTokenAuthFixture) issue(t *testing.T, userID int64) string {
	t.Helper()
	u, err := f.userSvc.GetByID(context.Background(), userID)
	require.NoError(t, err)
	raw := service.PersonalTokenPrefix + fmt.Sprintf("%064x", 0xbeef0000+userID)
	_, err = f.repo.Upsert(context.Background(), &service.PersonalToken{
		UserID: userID, TokenHash: service.HashPersonalToken(raw), TokenHint: "pat-hint", UserTokenVersion: u.TokenVersion,
	})
	require.NoError(t, err)
	return raw
}

func (f *personalTokenAuthFixture) do(method, path string, header map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var body *strings.Reader
	if method == http.MethodPost {
		body = strings.NewReader(`{"balance":1,"operation":"add"}`)
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	f.router.ServeHTTP(w, req)
	return w
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func TestPersonalTokenAdminAuth(t *testing.T) {
	f := newPersonalTokenAuthFixture(t)
	opsToken := f.issue(t, 2)

	t.Run("allowlisted_read_passes_as_operator_without_session", func(t *testing.T) {
		w := f.do(http.MethodGet, "/api/v1/admin/ops/errors/42", bearer(opsToken))
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"role":"operator"`)
		require.Contains(t, w.Body.String(), `"auth_method":"personal_token"`)
		require.Contains(t, w.Body.String(), `"session_id":""`, "a token is not a session")
	})

	t.Run("allowlisted_write_passes", func(t *testing.T) {
		w := f.do(http.MethodPost, "/api/v1/admin/tickets/9/messages", bearer(opsToken))
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"role":"operator"`)
	})

	t.Run("outside_allowlist_is_403_and_never_reaches_handler", func(t *testing.T) {
		for _, tc := range []struct{ method, path, template string }{
			{http.MethodGet, "/api/v1/admin/settings", "GET /api/v1/admin/settings"},
			{http.MethodGet, "/api/v1/admin/accounts", "GET /api/v1/admin/accounts"},
			{http.MethodGet, "/api/v1/admin/personal-tokens", "GET /api/v1/admin/personal-tokens"},
			{http.MethodPost, "/api/v1/admin/approvals/5/approve", "POST /api/v1/admin/approvals/:id/approve"},
		} {
			w := f.do(tc.method, tc.path, bearer(opsToken))
			require.Equalf(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
			require.Zerof(t, f.reached[tc.template], "handler reached for %s", tc.template)
		}
	})

	t.Run("approval_scope_write_is_queued_not_executed", func(t *testing.T) {
		w := f.do(http.MethodPost, "/api/v1/admin/users/7/balance", bearer(opsToken))
		require.Equal(t, http.StatusAccepted, w.Code)
		require.Zero(t, f.reached["POST /api/v1/admin/users/:id/balance"])
		in := f.gate.last()
		require.NotNil(t, in)
		require.Equal(t, int64(2), in.RequesterUserID)
	})

	t.Run("refused_action_stays_refused", func(t *testing.T) {
		w := f.do(http.MethodDelete, "/api/v1/admin/users/7", bearer(opsToken))
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Contains(t, w.Body.String(), "OPERATOR_ACTION_FORBIDDEN")
		require.Zero(t, f.reached["DELETE /api/v1/admin/users/:id"])
	})

	t.Run("step_up_route_rejects_token_even_with_switch_off", func(t *testing.T) {
		// 把 step-up 探针临时当作 operator 可达：直接测 step-up 这一层（路由本身不在白名单时认证层就会先 403）
		c, rec := newStepUpTestContext(t)
		c.Set("auth_method", service.AuditAuthMethodPersonalToken)
		c.Set(string(ContextKeyUser), AuthSubject{UserID: 2})
		require.False(t, enforceStepUp(c, stubStepUpGrantChecker{granted: true},
			stubStepUpUserReader{user: &service.User{ID: 2, TotpEnabled: true}}, stubStepUpSettingReader{enabled: false}))
		require.Equal(t, http.StatusForbidden, rec.Code)

		w := f.do(http.MethodGet, "/api/v1/admin/stepup-probe", bearer(opsToken))
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Zero(t, f.reached["GET /api/v1/admin/stepup-probe"])
	})

	t.Run("token_never_acts_as_admin", func(t *testing.T) {
		adminToken := f.issue(t, 1)
		for _, path := range []string{"/api/v1/admin/settings", "/api/v1/admin/ops/errors/1"} {
			w := f.do(http.MethodGet, path, bearer(adminToken))
			require.Equal(t, http.StatusUnauthorized, w.Code, path)
			require.Contains(t, w.Body.String(), "PERSONAL_TOKEN_NOT_ELIGIBLE")
			require.NotContains(t, w.Body.String(), `"role":"admin"`)
		}
		// 运维被提升为 admin 后，旧令牌同样不能以 admin 身份通过
		f.users[2].Role = service.RoleAdmin
		w := f.do(http.MethodGet, "/api/v1/admin/settings", bearer(opsToken))
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Zero(t, f.reached["GET /api/v1/admin/settings"])
		f.users[2].Role = service.RoleOperator
	})

	t.Run("regular_user_token_rejected", func(t *testing.T) {
		userToken := f.issue(t, 3)
		w := f.do(http.MethodGet, "/api/v1/admin/ops/errors/1", bearer(userToken))
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("password_change_and_switch_off_revoke_immediately", func(t *testing.T) {
		f.users[2].PasswordHash = "changed"
		w := f.do(http.MethodGet, "/api/v1/admin/ops/errors/1", bearer(opsToken))
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "TOKEN_REVOKED")
		f.users[2].PasswordHash = ""

		f.settings.Set(false)
		w = f.do(http.MethodGet, "/api/v1/admin/ops/errors/1", bearer(opsToken))
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "PERSONAL_TOKEN_DISABLED")
		f.settings.Set(true)

		w = f.do(http.MethodGet, "/api/v1/admin/ops/errors/1", bearer(opsToken))
		require.Equal(t, http.StatusOK, w.Code, "re-enabling restores the token")
	})

	t.Run("token_only_accepted_in_authorization_header", func(t *testing.T) {
		// （x-api-key 头走全局 Admin API Key 的常量时间比对，令牌不可能等于那把 key；该分支不经过令牌代码。）
		// WebSocket 子协议只认 JWT
		w := f.do(http.MethodGet, "/api/v1/admin/ops/ws/qps", map[string]string{
			"Upgrade": "websocket", "Connection": "Upgrade", "Sec-WebSocket-Protocol": "sub2api-admin, jwt." + opsToken,
		})
		require.Equal(t, http.StatusUnauthorized, w.Code)
		// query string 不是凭证
		w = f.do(http.MethodGet, "/api/v1/admin/ops/errors/1?token="+opsToken, nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		// 不认识的 pat- 串
		w = f.do(http.MethodGet, "/api/v1/admin/ops/errors/1", bearer("pat-"+strings.Repeat("0", 64)))
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "INVALID_TOKEN")
	})

	t.Run("jwt_path_unchanged", func(t *testing.T) {
		u := f.users[2]
		jwt, err := f.auth.GenerateToken(context.Background(), &service.User{ID: u.ID, Email: u.Email, Role: u.Role, TokenVersion: u.TokenVersion})
		require.NoError(t, err)
		w := f.do(http.MethodGet, "/api/v1/admin/ops/errors/1", bearer(jwt))
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"auth_method":"jwt"`)
	})

	// 认证层直接落的审计：拒绝 / 入队记录的是真实凭证类型与令牌 ID，凭证只留掩码
	f.audit.Stop()
	logs := f.auditRepo.snapshot()
	require.NotEmpty(t, logs)
	sawDenied, sawQueued := false, false
	for _, entry := range logs {
		if entry.ActorUserID == nil || *entry.ActorUserID != 2 {
			continue
		}
		require.Equal(t, service.AuditAuthMethodPersonalToken, entry.AuthMethod)
		require.NotContains(t, entry.CredentialMasked, opsToken, "full token must never be stored")
		require.NotEmpty(t, entry.CredentialMasked)
		require.EqualValues(t, f.mustTokenID(t, 2), entry.Extra["personal_token_id"])
		switch entry.Action {
		case service.AuditActionAdminScopeDenied:
			sawDenied = true
		case service.AuditActionAdminApprovalRequested:
			sawQueued = true
		}
	}
	require.True(t, sawDenied)
	require.True(t, sawQueued)
}

func (f *personalTokenAuthFixture) mustTokenID(t *testing.T, userID int64) int64 {
	t.Helper()
	token, err := f.repo.GetByUserID(context.Background(), userID)
	require.NoError(t, err)
	return token.ID
}

func TestPersonalTokenWithoutAuthenticatorFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	userService := service.NewUserService(&stubUserRepo{getByID: func(context.Context, int64) (*service.User, error) {
		return nil, service.ErrUserNotFound
	}}, nil, nil, nil)
	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(authService, userService, nil, nil)))
	router.GET("/api/v1/admin/ops/errors/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ops/errors/1", nil)
	req.Header.Set("Authorization", "Bearer pat-"+strings.Repeat("a", 64))
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "INVALID_TOKEN")
}
