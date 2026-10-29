package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ReferralHandler 用户侧推荐 Handler
type ReferralHandler struct {
	referralService *service.ReferralService
}

// NewReferralHandler 创建推荐 Handler
func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{
		referralService: referralService,
	}
}

// GetInfo 获取推荐信息
// GET /api/v1/referral/info
func (h *ReferralHandler) GetInfo(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	info, err := h.referralService.GetReferralInfo(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.ReferralInfoFromService(info))
}

// GetHistory 获取推荐历史
// GET /api/v1/referral/history
func (h *ReferralHandler) GetHistory(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{Page: page, PageSize: pageSize}

	refs, pag, err := h.referralService.GetReferralHistory(c.Request.Context(), subject.UserID, params)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.PaginatedWithResult(c, refs, referralPagToResponse(pag))
}

// GetAffiliate exposes the upstream-compatible affiliate detail shape.
// GET /api/v1/user/aff
func (h *ReferralHandler) GetAffiliate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok { response.Unauthorized(c, "User not authenticated"); return }
	detail, err := h.referralService.GetAffiliateDetail(c.Request.Context(), subject.UserID)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, detail)
}

// TransferAffiliate exposes the upstream-compatible transfer response.
// POST /api/v1/user/aff/transfer
func (h *ReferralHandler) TransferAffiliate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok { response.Unauthorized(c, "User not authenticated"); return }
	entry, err := h.referralService.TransferQuotaToBalance(c.Request.Context(), subject.UserID)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, gin.H{"transferred_quota": entry.Amount, "balance_after": entry.BalanceAfter, "balance": entry.BalanceAfter})
}

// Transfer moves all available rebate into the user's main balance.
// POST /api/v1/referral/transfer
func (h *ReferralHandler) Transfer(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	entry, err := h.referralService.TransferQuotaToBalance(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, entry)
}

func referralPagToResponse(p *pagination.PaginationResult) *response.PaginationResult {
	if p == nil {
		return nil
	}
	return &response.PaginationResult{
		Total:    p.Total,
		Page:     p.Page,
		PageSize: p.PageSize,
		Pages:    p.Pages,
	}
}
