package middleware

import (
	"context"
	"net/http"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 运维管理员个人令牌在认证层的入口（fork 本地，见 docs/PERSONAL_TOKENS.md）。
//
// 硬约束：令牌只替代「登录」这一步。这里只负责认出持有人，授权一律交给与 JWT 会话共用的
// authorizeConsoleUser（console_scope.go 的默认拒绝白名单 + 审批捕获）。本文件里没有任何
// 自己的放行逻辑；routes/personal_token_scope_equivalence_test.go 对全部 adminAuth 路由逐条
// 比对「operator JWT」与「同一 operator 的令牌」的结果，令牌只要比 JWT 宽一点测试就会失败。

// contextKeyPersonalTokenID 令牌认证通过后写入的令牌 ID（审计用）。
const contextKeyPersonalTokenID = "personal_token_id"

// PersonalTokenAuthenticator 个人令牌认证（由 service.PersonalTokenService 实现）。
type PersonalTokenAuthenticator interface {
	Authenticate(ctx context.Context, raw, clientIP string) (*service.User, *service.PersonalToken, error)
}

// PersonalTokenAuthenticatorOrNil 把可能为 nil 的具体指针归一化为接口，
// 避免 typed-nil 装箱后绕过 validatePersonalTokenForAdmin 里的 nil 判断。
func PersonalTokenAuthenticatorOrNil(svc *service.PersonalTokenService) PersonalTokenAuthenticator {
	if svc == nil {
		return nil
	}
	return svc
}

// validatePersonalTokenForAdmin 认证个人令牌并执行控制台角色门。返回 false 表示请求已被中断。
func validatePersonalTokenForAdmin(
	c *gin.Context,
	raw string,
	tokens PersonalTokenAuthenticator,
	auditService *service.AuditLogService,
	gate service.AdminApprovalGate,
) bool {
	if tokens == nil {
		AbortWithError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token")
		return false
	}
	user, token, err := tokens.Authenticate(c.Request.Context(), raw, SecurityClientIP(c))
	if err != nil {
		abortPersonalTokenError(c, err)
		return false
	}
	// 纵深防御：令牌只可能属于 operator。即使服务层将来放错，也绝不以 admin（或普通用户）身份进入角色门——
	// admin 在角色门里是全权放行的。要把令牌放开给 admin，必须有意识地改这里并重新评估等价测试。
	if user == nil || token == nil || !user.IsActive() || !user.IsOperator() || !service.IsPersonalTokenEligibleRole(user.Role) {
		AbortWithError(c, http.StatusUnauthorized, "PERSONAL_TOKEN_NOT_ELIGIBLE", "Only operators can use personal tokens")
		return false
	}

	c.Set(contextKeyPersonalTokenID, token.ID)
	if !authorizeConsoleUser(c, user, service.AuditAuthMethodPersonalToken, gate, auditService) {
		return false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      user.ID,
		Concurrency: user.Concurrency,
	})
	c.Set(string(ContextKeyUserRole), user.Role)
	c.Set(ContextKeyAuthEmail, user.Email)
	// 刻意不设 ContextKeySessionID：令牌不是会话，不能继承任何会话级授权（如 step-up）。
	SetAuditExtra(c, map[string]any{contextKeyPersonalTokenID: token.ID})
	return true
}

// abortPersonalTokenError 认证失败统一 401（沿用服务层的错误码与文案）；非 4xx 的内部错误不泄露细节。
func abortPersonalTokenError(c *gin.Context, err error) {
	status := infraerrors.Code(err)
	switch {
	case status == http.StatusUnauthorized:
		AbortWithError(c, status, infraerrors.Reason(err), infraerrors.Message(err))
	case status == http.StatusServiceUnavailable:
		AbortWithError(c, status, "PERSONAL_TOKEN_UNAVAILABLE", "Personal token service is not available")
	default:
		AbortWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify token")
	}
}

// consoleAuditAuthMethod 认证层直接落审计时使用的认证方式（authorizeConsoleUser 已写入上下文）。
func consoleAuditAuthMethod(c *gin.Context) string {
	if method := c.GetString("auth_method"); method != "" {
		return method
	}
	return service.AuditAuthMethodJWT
}

// withPersonalTokenAuditExtra 认证层直接落审计时补上令牌 ID（非令牌请求不变）。
func withPersonalTokenAuditExtra(c *gin.Context, extra map[string]any) map[string]any {
	value, ok := c.Get(contextKeyPersonalTokenID)
	if !ok {
		return extra
	}
	id, ok := value.(int64)
	if !ok || id <= 0 {
		return extra
	}
	if extra == nil {
		extra = map[string]any{}
	}
	extra[contextKeyPersonalTokenID] = id
	return extra
}

// validatePersonalTokenForUser 用户侧（jwtAuth）接口的个人令牌入口。返回 false 表示请求已被中断。
//
// 用户侧接口只作用于持有人自己的账号，令牌在这里默认可用；账号安全写操作（改密码、2FA、Passkey、
// 登录方式绑定、令牌自身、会话签发）由 PersonalTokenUserRouteAllows 挡下并留审计。
// 与管理侧一样只认 operator，且不设会话 ID。
func validatePersonalTokenForUser(
	c *gin.Context,
	raw string,
	tokens PersonalTokenAuthenticator,
	activityToucher userActivityToucher,
	auditService *service.AuditLogService,
) bool {
	if tokens == nil {
		AbortWithError(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token")
		return false
	}
	user, token, err := tokens.Authenticate(c.Request.Context(), raw, SecurityClientIP(c))
	if err != nil {
		abortPersonalTokenError(c, err)
		return false
	}
	if user == nil || token == nil || !user.IsActive() || !user.IsOperator() || !service.IsPersonalTokenEligibleRole(user.Role) {
		AbortWithError(c, http.StatusUnauthorized, "PERSONAL_TOKEN_NOT_ELIGIBLE", "Only operators can use personal tokens")
		return false
	}

	c.Set("auth_method", service.AuditAuthMethodPersonalToken)
	c.Set(contextKeyPersonalTokenID, token.ID)
	if !PersonalTokenUserRouteAllows(c.Request.Method, c.FullPath()) {
		// 审计中间件挂在认证之后看不到这里的响应，直接落库（同 operator 管理侧的拒绝）
		recordOperatorScopeDenied(c, auditService, user)
		AbortWithError(c, http.StatusForbidden, "PERSONAL_TOKEN_ROUTE_FORBIDDEN",
			"Personal token cannot perform account security actions; sign in with a browser session")
		return false
	}

	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      user.ID,
		Concurrency: user.Concurrency,
	})
	c.Set(string(ContextKeyUserRole), user.Role)
	c.Set(ContextKeyAuthEmail, user.Email)
	// 刻意不设 ContextKeySessionID：令牌不是会话。
	SetAuditExtra(c, map[string]any{contextKeyPersonalTokenID: token.ID})
	if activityToucher != nil {
		activityToucher.TouchLastActiveForUser(c.Request.Context(), user)
	}
	return true
}
