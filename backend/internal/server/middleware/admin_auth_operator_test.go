//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// capturingAuditRepo 收集异步写入的审计记录（Stop() 会排空队列并落盘）。
type capturingAuditRepo struct {
	mu   sync.Mutex
	logs []*service.AuditLog
}

func (r *capturingAuditRepo) BatchInsert(_ context.Context, logs []*service.AuditLog) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, logs...)
	return int64(len(logs)), nil
}

func (r *capturingAuditRepo) Insert(_ context.Context, log *service.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, log)
	return nil
}

func (r *capturingAuditRepo) List(context.Context, *service.AuditLogFilter) (*service.AuditLogList, error) {
	return &service.AuditLogList{}, nil
}

func (r *capturingAuditRepo) GetByID(context.Context, int64) (*service.AuditLog, error) {
	return nil, nil
}

func (r *capturingAuditRepo) Count(context.Context) (int64, error) { return 0, nil }

func (r *capturingAuditRepo) TruncateAll(context.Context) error { return nil }

func (r *capturingAuditRepo) DeleteBefore(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func (r *capturingAuditRepo) snapshot() []*service.AuditLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*service.AuditLog(nil), r.logs...)
}

type operatorAuthFixture struct {
	router    *gin.Engine
	auth      *service.AuthService
	audit     *service.AuditLogService
	auditRepo *capturingAuditRepo
	users     map[int64]*service.User
}

func newOperatorAuthFixture(t *testing.T) *operatorAuthFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)

	users := map[int64]*service.User{
		1: {ID: 1, Email: "admin@example.com", Role: service.RoleAdmin, Status: service.StatusActive, TokenVersion: 1, Concurrency: 1},
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

	auditRepo := &capturingAuditRepo{}
	auditService := service.NewAuditLogService(auditRepo, nil)
	auditService.Start()
	t.Cleanup(auditService.Stop)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(authService, userService, nil, auditService)))
	echoRole := func(c *gin.Context) {
		role, _ := GetUserRoleFromContext(c)
		c.JSON(http.StatusOK, gin.H{"role": role})
	}
	// 用模板注册，保证中间件里 c.FullPath() 拿到的是路由模板而不是空串。
	router.GET("/api/v1/admin/ops/errors/:id", echoRole)
	router.GET("/api/v1/admin/ops/ws/qps", echoRole)
	router.GET("/api/v1/admin/usage", echoRole)
	router.GET("/api/v1/admin/settings", echoRole)
	router.GET("/api/v1/admin/accounts", echoRole)
	router.POST("/api/v1/admin/users/:id/balance", echoRole)
	router.PUT("/api/v1/admin/ops/errors/:id/resolve", echoRole)
	router.POST("/api/v1/admin/compliance/accept", echoRole)
	router.GET("/api/internal/pay/auth/admin", echoRole)

	return &operatorAuthFixture{router: router, auth: authService, audit: auditService, auditRepo: auditRepo, users: users}
}

func (f *operatorAuthFixture) tokenFor(t *testing.T, userID int64) string {
	t.Helper()
	u := f.users[userID]
	token, err := f.auth.GenerateToken(context.Background(), &service.User{ID: u.ID, Email: u.Email, Role: u.Role, TokenVersion: u.TokenVersion})
	require.NoError(t, err)
	return token
}

func (f *operatorAuthFixture) do(t *testing.T, userID int64, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+f.tokenFor(t, userID))
	f.router.ServeHTTP(w, req)
	return w
}

func TestAdminAuthOperatorScope(t *testing.T) {
	f := newOperatorAuthFixture(t)

	t.Run("operator_allowed_on_whitelisted_get", func(t *testing.T) {
		w := f.do(t, 2, http.MethodGet, "/api/v1/admin/ops/errors/42?q=x")
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"role":"operator"`)
	})

	t.Run("operator_has_no_write_access_at_all", func(t *testing.T) {
		w := f.do(t, 2, http.MethodPost, "/api/v1/admin/compliance/accept")
		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("operator_denied_outside_scope", func(t *testing.T) {
		for _, tc := range []struct{ method, path string }{
			{http.MethodPost, "/api/v1/admin/users/7/balance"},
			{http.MethodGet, "/api/v1/admin/settings"},
			{http.MethodGet, "/api/v1/admin/accounts"},
			{http.MethodPut, "/api/v1/admin/ops/errors/42/resolve"},
			{http.MethodGet, "/api/internal/pay/auth/admin"},
		} {
			w := f.do(t, 2, tc.method, tc.path)
			require.Equalf(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
			require.Contains(t, w.Body.String(), "FORBIDDEN")
		}
	})

	t.Run("admin_unaffected", func(t *testing.T) {
		for _, tc := range []struct{ method, path string }{
			{http.MethodGet, "/api/v1/admin/settings"},
			{http.MethodPost, "/api/v1/admin/users/7/balance"},
			{http.MethodGet, "/api/v1/admin/usage"},
		} {
			w := f.do(t, 1, tc.method, tc.path)
			require.Equalf(t, http.StatusOK, w.Code, "%s %s", tc.method, tc.path)
			require.Contains(t, w.Body.String(), `"role":"admin"`)
		}
	})

	t.Run("regular_user_denied_even_on_whitelisted_route", func(t *testing.T) {
		w := f.do(t, 3, http.MethodGet, "/api/v1/admin/usage")
		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("operator_websocket_handshake_follows_scope", func(t *testing.T) {
		token := f.tokenFor(t, 2)
		for path, want := range map[string]int{
			"/api/v1/admin/ops/ws/qps": http.StatusOK,
			"/api/v1/admin/settings":   http.StatusForbidden,
		} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
			f.router.ServeHTTP(w, req)
			require.Equalf(t, want, w.Code, path)
		}
	})

	// 排空异步队列后检查审计：只有 operator 的拒绝被记录，普通用户与 admin 不产生 scope.denied。
	f.audit.Stop()
	logs := f.auditRepo.snapshot()
	require.NotEmpty(t, logs)
	for _, entry := range logs {
		require.Equal(t, service.AuditActionAdminScopeDenied, entry.Action)
		require.Equal(t, service.RoleOperator, entry.ActorRole)
		require.Equal(t, "ops@example.com", entry.ActorEmail)
		require.NotNil(t, entry.ActorUserID)
		require.Equal(t, int64(2), *entry.ActorUserID)
		require.Equal(t, service.AuditAuthMethodJWT, entry.AuthMethod)
		require.Equal(t, http.StatusForbidden, entry.StatusCode)
		require.Empty(t, entry.RequestBody)
	}
	// 1 条写请求 + 5 条普通拒绝 + 1 条 WebSocket 握手拒绝。
	require.Len(t, logs, 7)

	paths := map[string]bool{}
	for _, entry := range logs {
		paths[entry.Method+" "+entry.Path] = true
	}
	require.True(t, paths["POST /api/v1/admin/users/:id/balance"], "path is recorded as the route template")
	require.True(t, paths["GET /api/internal/pay/auth/admin"], "adminAuth reuse outside /admin is denied and audited too")
}
