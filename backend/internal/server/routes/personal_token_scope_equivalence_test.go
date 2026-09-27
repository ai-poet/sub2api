//go:build unit

package routes

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/testutil"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// 运维个人令牌「防越权保证」的兜底测试（fork 本地，见 docs/PERSONAL_TOKENS.md）。
//
// 1. 全路由等价：对每一条经 adminAuth 保护的已注册路由（admin / pages / pay 桥接，今后上游新增的也自动纳入），
//    分别用「operator 的 JWT」和「同一 operator 的个人令牌」请求一次，状态码与 handler 是否被执行必须完全一致；
//    凡是执行到 handler 的，身份必须是 operator。令牌只要在任何一条路由上比 JWT 宽，这里就会失败。
// 2. 用户侧全拒绝：所有挂 jwtAuth 的用户 / 认证路由（改密、2FA、建网关 Key、令牌自身的管理接口……）
//    用令牌请求一律 401，handler 一次都不会被执行。

// equivalenceUserRepo 只实现 GetByID；其余方法未实现（被调用即 panic，说明认证路径越界）。
type equivalenceUserRepo struct {
	service.UserRepository
	users map[int64]*service.User
}

func (r *equivalenceUserRepo) GetByID(_ context.Context, id int64) (*service.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}

func (r *equivalenceUserRepo) GetUserAvatar(context.Context, int64) (*service.UserAvatar, error) {
	return nil, nil
}

type equivalenceGate struct{}

func (equivalenceGate) Capture(_ context.Context, in *service.AdminApprovalCaptureInput) (*service.AdminApprovalRequest, error) {
	return &service.AdminApprovalRequest{ID: 1, Status: service.ApprovalStatusPending, Action: in.Action, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type equivalenceFixture struct {
	auth    *service.AuthService
	userSvc *service.UserService
	tokens  *service.PersonalTokenService
	jwt     string
	pat     string
}

func newEquivalenceFixture(t *testing.T) *equivalenceFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	operator := &service.User{ID: 2, Email: "ops@example.com", Role: service.RoleOperator, Status: service.StatusActive, TokenVersion: 1, Concurrency: 1}
	userSvc := service.NewUserService(&equivalenceUserRepo{users: map[int64]*service.User{2: operator}}, nil, nil, nil)

	repo := testutil.NewMemoryPersonalTokenRepo()
	tokens := service.NewPersonalTokenService(repo, userSvc, testutil.NewStaticPersonalTokenSettings(true))

	loaded, err := userSvc.GetByID(context.Background(), 2)
	require.NoError(t, err)
	pat := service.PersonalTokenPrefix + fmt.Sprintf("%064x", 0xc0ffee)
	_, err = repo.Upsert(context.Background(), &service.PersonalToken{
		UserID: 2, TokenHash: service.HashPersonalToken(pat), TokenHint: "pat-hint", UserTokenVersion: loaded.TokenVersion,
	})
	require.NoError(t, err)

	jwt, err := authService.GenerateToken(context.Background(), &service.User{ID: 2, Email: operator.Email, Role: operator.Role, TokenVersion: operator.TokenVersion})
	require.NoError(t, err)
	return &equivalenceFixture{auth: authService, userSvc: userSvc, tokens: tokens, jwt: jwt, pat: pat}
}

// concretePath 把路由模板填成可请求的路径（:param → 1，*path → x）。
func concretePath(template string) string {
	segs := strings.Split(template, "/")
	for i, seg := range segs {
		switch {
		case strings.HasPrefix(seg, ":"):
			segs[i] = "1"
		case strings.HasPrefix(seg, "*"):
			segs[i] = "x"
		}
	}
	return strings.Join(segs, "/")
}

type routeOutcome struct {
	status     int
	reached    bool
	role       string
	authMethod string
	sessionID  string
}

func TestPersonalTokenScopeEquivalentToOperatorJWT(t *testing.T) {
	f := newEquivalenceFixture(t)
	registered := registerAllAdminAuthRoutesForTest(t).Routes()
	require.NotEmpty(t, registered)

	adminAuth := servermiddleware.NewConsoleAdminAuthMiddleware(f.auth, f.userSvc, nil, nil, equivalenceGate{},
		servermiddleware.PersonalTokenAuthenticatorOrNil(f.tokens))

	var current *routeOutcome
	engine := gin.New()
	engine.Use(gin.HandlerFunc(adminAuth))
	stub := func(c *gin.Context) {
		current.reached = true
		current.role, _ = servermiddleware.GetUserRoleFromContext(c)
		current.authMethod = c.GetString("auth_method")
		current.sessionID = c.GetString(servermiddleware.ContextKeySessionID)
		c.Status(http.StatusNoContent)
	}
	for _, r := range registered {
		engine.Handle(r.Method, r.Path, stub)
	}

	request := func(method, path, token string) routeOutcome {
		out := routeOutcome{}
		current = &out
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		engine.ServeHTTP(w, req)
		out.status = w.Code
		return out
	}

	reached, queued := 0, 0
	for _, r := range registered {
		path := concretePath(r.Path)
		viaJWT := request(r.Method, path, f.jwt)
		viaPAT := request(r.Method, path, f.pat)

		key := r.Method + " " + r.Path
		require.Equalf(t, viaJWT.status, viaPAT.status, "status differs between operator JWT and personal token on %s", key)
		require.Equalf(t, viaJWT.reached, viaPAT.reached, "handler reachability differs on %s", key)
		if viaPAT.reached {
			reached++
			require.Equalf(t, service.RoleOperator, viaPAT.role, "personal token must act as operator on %s", key)
			require.Equalf(t, service.AuditAuthMethodPersonalToken, viaPAT.authMethod, key)
			require.Emptyf(t, viaPAT.sessionID, "personal token must not carry a session on %s", key)
		}
		if viaPAT.status == http.StatusAccepted {
			queued++
		}
	}

	// 自检：比对确实覆盖了白名单与审批范围（否则上面的等价只是「全部 401」的空转）
	require.Equal(t, len(servermiddleware.OperatorScopeRoutes()), reached, "every allowlisted route is reachable, nothing else is")
	require.Equal(t, len(servermiddleware.OperatorApprovalScopeRoutes()), queued, "every approval-scope write is queued, nothing else is")
}

func TestPersonalTokenRejectedOnAllJWTOnlyRoutes(t *testing.T) {
	f := newEquivalenceFixture(t)
	// 公开路由的限流器在标记遍里会被触发：给它一个内存 Redis，避免 fail-close 等待拖慢测试
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	passAudit := servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() })

	build := func(jwtAuth servermiddleware.JWTAuthMiddleware) *gin.Engine {
		engine := gin.New()
		// 公开路由的零值 handler 会 panic：静默恢复成 599（只可能出现在不挂 jwtAuth 的路由上）
		engine.Use(gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) { c.AbortWithStatus(599) }))
		v1 := engine.Group("/api/v1")
		h := &handler.Handlers{Admin: &handler.AdminHandlers{}, Auth: &handler.AuthHandler{}, Setting: &handler.SettingHandler{}}
		RegisterAuthRoutes(v1, h, jwtAuth, passAudit, rdb, nil, nil)
		RegisterUserRoutes(v1, h, jwtAuth, passAudit, nil, nil)
		return engine
	}
	send := func(engine *gin.Engine, method, path string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+f.pat)
		engine.ServeHTTP(w, req)
		return w.Code
	}

	// 第一遍：用标记中间件找出所有挂 jwtAuth 的路由
	protected := map[string]string{}
	marker := build(servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
		protected[c.Request.Method+" "+c.FullPath()] = c.Request.URL.Path
		c.AbortWithStatus(http.StatusTeapot)
	}))
	for _, r := range marker.Routes() {
		send(marker, r.Method, concretePath(r.Path))
	}
	require.Contains(t, protected, "POST /api/v1/user/personal-token", "token management must be JWT-only")
	require.Contains(t, protected, "DELETE /api/v1/user/personal-token")
	require.Contains(t, protected, "PUT /api/v1/user/password")
	require.Contains(t, protected, "POST /api/v1/user/totp/disable")
	require.Contains(t, protected, "POST /api/v1/keys")
	require.Contains(t, protected, "GET /api/v1/auth/me")
	require.Greater(t, len(protected), 40)

	// 第二遍：真实 jwtAuth，令牌逐条请求，一律 401（handler 被执行会 panic → 599，或返回别的状态）
	real := build(servermiddleware.NewJWTAuthMiddleware(f.auth, f.userSvc, nil, nil))
	for key, path := range protected {
		method, _, _ := strings.Cut(key, " ")
		require.Equalf(t, http.StatusUnauthorized, send(real, method, path), "personal token must be rejected on %s", key)
	}
}
