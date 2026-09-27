package handler

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PersonalTokenHandler 运维管理员个人令牌的自助管理接口（fork 本地功能）。
//
// 这些路由挂在 jwtAuth 之后（routes/user.go），jwtAuth 只认 JWT：个人令牌本身调不到这里，
// 所以令牌不能生成、覆盖或吊销令牌。只能操作当前登录用户自己的令牌。
type PersonalTokenHandler struct {
	svc *service.PersonalTokenService
}

// NewPersonalTokenHandler 构造用户侧个人令牌 handler。
func NewPersonalTokenHandler(svc *service.PersonalTokenService) *PersonalTokenHandler {
	return &PersonalTokenHandler{svc: svc}
}

// GetStatus GET /api/v1/user/personal-token
func (h *PersonalTokenHandler) GetStatus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.svc.Status(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.PersonalTokenStatusResponse{
		FeatureEnabled: status.FeatureEnabled,
		Eligible:       status.Eligible,
		Token:          dto.PersonalTokenInfoFromService(status.Token, time.Now()),
		ExpiryOptions:  append([]int(nil), service.PersonalTokenExpiryDays...),
	})
}

// Generate POST /api/v1/user/personal-token
// 需要当前密码；覆盖已有令牌。响应里的 token 是明文，只返回这一次。
func (h *PersonalTokenHandler) Generate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var body dto.GeneratePersonalTokenRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	if body.ExpiresInDays == nil {
		response.ErrorFrom(c, service.ErrPersonalTokenExpiryInvalid)
		return
	}
	issued, err := h.svc.Generate(c.Request.Context(), service.PersonalTokenGenerateInput{
		UserID:        subject.UserID,
		Password:      body.Password,
		ExpiresInDays: *body.ExpiresInDays,
		ClientIP:      middleware2.SecurityClientIP(c),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// 明文凭证不允许被任何中间层缓存
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	response.Success(c, dto.GeneratePersonalTokenResponse{
		Token: issued.Token,
		Info:  *dto.PersonalTokenInfoFromService(issued.Record, time.Now()),
	})
}

// Revoke DELETE /api/v1/user/personal-token
func (h *PersonalTokenHandler) Revoke(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := h.svc.Revoke(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"revoked": true})
}
