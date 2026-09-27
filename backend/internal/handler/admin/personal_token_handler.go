package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PersonalTokenHandler 管理员查看 / 吊销运维管理员个人令牌（fork 本地功能）。
// 路由挂 AdminOnly()，且不在 operator 白名单里（routes/console_scope_coverage_test.go 的
// operatorForbiddenPrefixes 钉死）：运维永远不能管理别人的令牌。响应只含提示串。
type PersonalTokenHandler struct {
	svc *service.PersonalTokenService
}

// NewPersonalTokenHandler 构造管理员侧个人令牌 handler。
func NewPersonalTokenHandler(svc *service.PersonalTokenService) *PersonalTokenHandler {
	return &PersonalTokenHandler{svc: svc}
}

// List GET /api/v1/admin/personal-tokens
func (h *PersonalTokenHandler) List(c *gin.Context) {
	items, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := dto.AdminPersonalTokenListResponse{
		FeatureEnabled: h.svc.Enabled(c.Request.Context()),
		Items:          make([]dto.AdminPersonalTokenItem, 0, len(items)),
	}
	for _, item := range items {
		out.Items = append(out.Items, dto.AdminPersonalTokenItemFromService(item))
	}
	response.Success(c, out)
}

// Revoke DELETE /api/v1/admin/personal-tokens/:user_id
func (h *PersonalTokenHandler) Revoke(c *gin.Context) {
	userID, err := strconv.ParseInt(strings.TrimSpace(c.Param("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user id")
		return
	}
	if err := h.svc.Revoke(c.Request.Context(), userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"revoked": true})
}
