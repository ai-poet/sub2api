package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ShopHandler struct {
	shopService *service.ShopService
}

func NewShopHandler(shopService *service.ShopService) *ShopHandler {
	return &ShopHandler{shopService: shopService}
}

// ListProducts GET /api/v1/admin/shop/products
func (h *ShopHandler) ListProducts(c *gin.Context) {
	products, err := h.shopService.ListAllProducts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminShopProduct, 0, len(products))
	for i := range products {
		out = append(out, dto.AdminShopProductFromService(&products[i]))
	}
	response.Success(c, out)
}

// CreateProduct POST /api/v1/admin/shop/products
func (h *ShopHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateShopProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p := req.ToService()
	result, err := h.shopService.CreateProduct(c.Request.Context(), p)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.AdminShopProductFromService(result))
}

// UpdateProduct PUT /api/v1/admin/shop/products/:id
func (h *ShopHandler) UpdateProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req dto.CreateShopProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p := req.ToService()
	p.ID = id
	result, err := h.shopService.UpdateProduct(c.Request.Context(), p)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminShopProductFromService(result))
}

// DeleteProduct DELETE /api/v1/admin/shop/products/:id
func (h *ShopHandler) DeleteProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.shopService.DeleteProduct(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// GetStockList GET /api/v1/admin/shop/products/:id/stocks
func (h *ShopHandler) GetStockList(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	stocks, err := h.shopService.GetStockList(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.ShopProductStock, 0, len(stocks))
	for i := range stocks {
		out = append(out, dto.ShopProductStockFromService(&stocks[i]))
	}
	response.Success(c, out)
}

// AddStock POST /api/v1/admin/shop/products/:id/stocks
func (h *ShopHandler) AddStock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req struct {
		Count int `json:"count" binding:"required,min=1,max=1000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	added, err := h.shopService.AddStock(c.Request.Context(), id, req.Count)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"added": added})
}

// DeleteStock DELETE /api/v1/admin/shop/stocks/:id
func (h *ShopHandler) DeleteStock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.shopService.DeleteStock(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
