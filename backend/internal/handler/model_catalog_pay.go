package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Prices and tiers only. Account details, other groups and price sources never
// cross the bridge into the for-sale plan response.
type payCatalogModel struct {
	Model          string               `json:"model"`
	DisplayName    string               `json:"display_name"`
	BillingMode    string               `json:"billing_mode"`
	Pricing        publicPricing        `json:"effective_pricing_usd"`
	PricingDetails publicPricingDetails `json:"pricing_details"`
}

// ListPayGroup is registered exclusively behind internalPayAuthMiddleware.
func (h *ModelCatalogHandler) ListPayGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	group, err := h.payGroupReader.GetGroup(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if group == nil || group.Status != service.StatusActive {
		response.ErrorFrom(c, service.ErrGroupNotFound)
		return
	}
	catalog, err := h.modelCatalogService.GetPayGroupCatalog(c.Request.Context(), *group)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	models := make([]payCatalogModel, 0, len(catalog.Items))
	for _, item := range catalog.Items {
		models = append(models, payCatalogModel{
			Model: item.Model, DisplayName: item.DisplayName, BillingMode: item.BillingMode,
			Pricing:        toPublicPricing(item.EffectivePricingUSD),
			PricingDetails: toPublicPricingDetails(item.PricingDetails),
		})
	}
	response.Success(c, gin.H{"models": models})
}
