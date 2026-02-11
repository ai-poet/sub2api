package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)


// ReferralHandler 管理侧推荐 Handler
type ReferralHandler struct {
	referralService *service.ReferralService
}

// NewReferralHandler 创建管理侧推荐 Handler
func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{
		referralService: referralService,
	}
}

// GetSettings 获取推荐配置
// GET /api/v1/admin/referral/settings
func (h *ReferralHandler) GetSettings(c *gin.Context) {
	settings := h.referralService.GetReferralSettings(c.Request.Context())
	response.Success(c, dto.ReferralSettingsFromService(settings))
}

// UpdateSettings 更新推荐配置
// PUT /api/v1/admin/referral/settings
func (h *ReferralHandler) UpdateSettings(c *gin.Context) {
	var req dto.ReferralSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.referralService.UpdateReferralSettings(c.Request.Context(), dto.ReferralSettingsToService(&req)); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "settings updated"})
}

// ListReferrals 查看所有推荐记录
// GET /api/v1/admin/referral/list
func (h *ReferralHandler) ListReferrals(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	refs, pag, err := h.referralService.GetAllReferrals(c.Request.Context(), params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserReferral, 0, len(refs))
	for i := range refs {
		out = append(out, *dto.UserReferralFromService(&refs[i]))
	}

	response.PaginatedWithResult(c, out, toResponsePagination(pag))
}
