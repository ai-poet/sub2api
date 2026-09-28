package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewJWTAuthMiddleware 创建 JWT 认证中间件
func NewJWTAuthMiddleware(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
) JWTAuthMiddleware {
	return JWTAuthMiddleware(jwtAuth(authService, userService, userService, settingService, auditService))
}

// NewJWTAuthMiddlewareWithPersonalTokens 生产 wire 使用：用户侧接口同时接受运维个人令牌（fork 本地）。
// 令牌只能调用 PersonalTokenUserRouteAllows 放行的路由（账号安全写操作除外）；tokens 为 nil 时一律 401。
func NewJWTAuthMiddlewareWithPersonalTokens(
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
	tokens PersonalTokenAuthenticator,
) JWTAuthMiddleware {
	return JWTAuthMiddleware(jwtAuthWithPersonalTokens(authService, userService, userService, settingService, auditService, tokens))
}

type jwtUserReader interface {
	GetByID(ctx context.Context, id int64) (*service.User, error)
}

type userActivityToucher interface {
	TouchLastActiveForUser(ctx context.Context, user *service.User)
}

// jwtAuth JWT认证中间件实现
func jwtAuth(
	authService *service.AuthService,
	userService jwtUserReader,
	activityToucher userActivityToucher,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
) gin.HandlerFunc {
	return jwtAuthWithPersonalTokens(authService, userService, activityToucher, settingService, auditService, nil)
}

// jwtAuthWithPersonalTokens JWT 认证；tokens 非空时 pat- 开头的 Bearer 走运维个人令牌入口（fork 本地）。
func jwtAuthWithPersonalTokens(
	authService *service.AuthService,
	userService jwtUserReader,
	activityToucher userActivityToucher,
	settingService *service.SettingService,
	auditService *service.AuditLogService,
	tokens PersonalTokenAuthenticator,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Authorization header中提取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			AbortWithError(c, 401, "UNAUTHORIZED", "Authorization header is required")
			return
		}

		// 验证Bearer scheme
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			AbortWithError(c, 401, "INVALID_AUTH_HEADER", "Authorization header format must be 'Bearer {token}'")
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			AbortWithError(c, 401, "EMPTY_TOKEN", "Token cannot be empty")
			return
		}

		// fork：pat- 前缀的一定不是 JWT（JWT 以 "eyJ" 开头），走运维个人令牌入口
		if strings.HasPrefix(tokenString, service.PersonalTokenPrefix) {
			if !validatePersonalTokenForUser(c, tokenString, tokens, activityToucher, auditService) {
				return
			}
			c.Next()
			return
		}

		// 验证token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				AbortWithError(c, 401, "TOKEN_EXPIRED", "Token has expired")
				return
			}
			AbortWithError(c, 401, "INVALID_TOKEN", "Invalid token")
			return
		}

		// 从数据库获取最新的用户信息
		user, err := userService.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, service.ErrUserNotFound) {
				AbortWithError(c, 401, "USER_NOT_FOUND", "User not found")
			} else {
				AbortWithError(c, 500, "INTERNAL_ERROR", "Failed to load user")
			}
			return
		}

		// 检查用户状态
		if !user.IsActive() {
			AbortWithError(c, 401, "USER_INACTIVE", "User account is not active")
			return
		}

		// Security: Validate TokenVersion to ensure token hasn't been invalidated
		// This check ensures tokens issued before a password change are rejected
		if claims.TokenVersion != user.TokenVersion {
			AbortWithError(c, 401, "TOKEN_REVOKED", "Token has been revoked (password changed)")
			return
		}

		// 会话绑定校验：IP/UA 任一变化即撤销会话（功能可在系统设置中关闭）
		if !enforceSessionBinding(c, authService, settingService, auditService, claims) {
			return
		}

		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      user.ID,
			Concurrency: user.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), user.Role)
		c.Set(ContextKeyAuthEmail, user.Email)
		c.Set(ContextKeySessionID, claims.SessionID)
		if activityToucher != nil {
			activityToucher.TouchLastActiveForUser(c.Request.Context(), user)
		}

		c.Next()
	}
}

// Deprecated: prefer GetAuthSubjectFromContext in auth_subject.go.
