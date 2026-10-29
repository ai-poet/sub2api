package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ReferralHandler struct{ referralService *service.ReferralService }

func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referralService: referralService}
}

func (h *ReferralHandler) GetSettings(c *gin.Context) {
	response.Success(c, dto.ReferralSettingsFromService(h.referralService.GetReferralSettings(c.Request.Context())))
}

func (h *ReferralHandler) UpdateSettings(c *gin.Context) {
	var req dto.ReferralSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.referralService.UpdateReferralSettings(c.Request.Context(), &req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.referralService.GetReferralSettings(c.Request.Context()))
}

func affiliateRecordFilter(c *gin.Context, action string) service.AffiliateRecordFilter {
	page, pageSize := response.ParsePagination(c)
	filter := service.AffiliateRecordFilter{Search: strings.TrimSpace(c.Query("search")), Action: action, Page: page, PageSize: pageSize}
	filter.UserID, _ = strconv.ParseInt(c.Query("user_id"), 10, 64)
	if value := c.Query("start_at"); value != "" {
		filter.StartAt = parseAffiliateFilterTime(value, false)
	}
	if value := c.Query("end_at"); value != "" {
		filter.EndAt = parseAffiliateFilterTime(value, true)
	}
	return filter
}

func parseAffiliateFilterTime(value string, end bool) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", value)
		if err == nil && end {
			parsed = parsed.AddDate(0, 0, 1)
		}
	}
	if err != nil {
		return nil
	}
	return &parsed
}

func (h *ReferralHandler) listRelations(c *gin.Context) {
	items, pag, err := h.referralService.ListRelations(c.Request.Context(), affiliateRecordFilter(c, ""))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.PaginatedWithResult(c, items, toResponsePagination(pag))
}

func (h *ReferralHandler) listLedger(c *gin.Context, action string) {
	items, pag, err := h.referralService.ListLedger(c.Request.Context(), affiliateRecordFilter(c, action))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.PaginatedWithResult(c, items, toResponsePagination(pag))
}

func (h *ReferralHandler) ListReferrals(c *gin.Context) { h.listRelations(c) }
func (h *ReferralHandler) ListInvites(c *gin.Context)   { h.listRelations(c) }
func (h *ReferralHandler) ListRebates(c *gin.Context)   { h.listLedger(c, service.AffiliateActionAccrue) }
func (h *ReferralHandler) ListTransfers(c *gin.Context) {
	h.listLedger(c, service.AffiliateActionTransfer)
}

func affiliateUserID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user id")
		return 0, false
	}
	return id, true
}

func (h *ReferralHandler) GetUserOverview(c *gin.Context) {
	id, ok := affiliateUserID(c)
	if !ok {
		return
	}
	detail, err := h.referralService.GetAffiliateDetail(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

type updateAffiliateUserRequest struct {
	AffCode              *string  `json:"aff_code"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent"`
	ClearRebateRate      bool     `json:"clear_rebate_rate"`
}

func (h *ReferralHandler) UpdateUser(c *gin.Context) {
	id, ok := affiliateUserID(c)
	if !ok {
		return
	}
	var req updateAffiliateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.AffCode == nil && req.AffRebateRatePercent == nil && !req.ClearRebateRate {
		response.BadRequest(c, "no affiliate changes supplied")
		return
	}
	if req.AffCode != nil {
		if err := h.referralService.UpdateUserAffCode(c.Request.Context(), id, *req.AffCode); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	if req.ClearRebateRate || req.AffRebateRatePercent != nil {
		if err := h.referralService.SetUserRebateRate(c.Request.Context(), id, req.AffRebateRatePercent); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	h.GetUserOverview(c)
}

func (h *ReferralHandler) ResetUserCode(c *gin.Context) {
	id, ok := affiliateUserID(c)
	if !ok {
		return
	}
	code, err := h.referralService.ResetUserAffCode(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"aff_code": code})
}

type batchAffiliateRateRequest struct {
	UserIDs              []int64  `json:"user_ids" binding:"required"`
	AffRebateRatePercent *float64 `json:"aff_rebate_rate_percent"`
	Clear                bool     `json:"clear"`
}

func (h *ReferralHandler) BatchSetRate(c *gin.Context) {
	var req batchAffiliateRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.UserIDs) > 200 {
		response.BadRequest(c, "at most 200 users per batch")
		return
	}
	if !req.Clear && req.AffRebateRatePercent == nil {
		response.BadRequest(c, "aff_rebate_rate_percent is required")
		return
	}
	for _, id := range req.UserIDs {
		if id <= 0 {
			response.BadRequest(c, "invalid user id")
			return
		}
		if err := h.referralService.SetUserRebateRate(c.Request.Context(), id, req.AffRebateRatePercent); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	response.Success(c, gin.H{"updated": len(req.UserIDs)})
}
