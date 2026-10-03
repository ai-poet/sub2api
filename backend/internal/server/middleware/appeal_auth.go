package middleware

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AppealAuthenticator 校验申诉令牌（实现是 *service.AppealService）。
type AppealAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*service.AppealSession, error)
}

const contextKeyAppealSession = "appeal_session"

// AppealAuth 申诉会话认证（fork 本地，见 service/appeal.go）。
//
//   - 先查路由白名单（appeal_scope.go），默认拒绝；
//   - 只读 X-Appeal-Token 请求头，绝不读 Authorization —— 申诉令牌不是 Bearer 凭证，
//     jwtAuth / adminAuth 也永远不认它；
//   - 认证通过后设置与 jwtAuth 相同的 context key（用户 / 角色 / 邮箱），工单、附件、站内信的用户侧
//     handler 可以原样复用；角色固定为 user：申诉人在申诉会话里永远只是工单发起方。
func AppealAuth(auth AppealAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if !AppealScopeAllows(c.Request.Method, c.FullPath()) {
			response.ErrorFrom(c, service.ErrAppealRouteForbidden)
			c.Abort()
			return
		}
		if auth == nil {
			response.ErrorFrom(c, service.ErrAppealUnavailable)
			c.Abort()
			return
		}
		token := strings.TrimSpace(c.GetHeader(service.AppealTokenHeader))
		session, err := auth.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.ErrorFrom(c, err)
			c.Abort()
			return
		}
		c.Set(string(ContextKeyUser), AuthSubject{UserID: session.UserID})
		c.Set(string(ContextKeyUserRole), service.RoleUser)
		c.Set(ContextKeyAuthEmail, session.Email)
		c.Set("auth_method", service.AuditAuthMethodAppealToken)
		c.Set(contextKeyAppealSession, session)
		SetAuditActor(c, session.UserID, session.Email)
		c.Next()
	}
}

// GetAppealSessionFromContext 取出当前申诉会话（只在 /api/v1/appeal/* 上有）。
func GetAppealSessionFromContext(c *gin.Context) (*service.AppealSession, bool) {
	value, ok := c.Get(contextKeyAppealSession)
	if !ok {
		return nil, false
	}
	session, ok := value.(*service.AppealSession)
	return session, ok && session != nil
}
