package handler

import (
	"io"

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

// ListProducts GET /api/v1/shop/products
func (h *ShopHandler) ListProducts(c *gin.Context) {
	products, err := h.shopService.GetActiveProducts(c.Request.Context())
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

// GetPaymentChannels GET /api/v1/shop/channels
func (h *ShopHandler) GetPaymentChannels(c *gin.Context) {
	channels, err := h.shopService.GetPaymentChannels(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, channels)
}

// CreateOrder POST /api/v1/shop/orders
func (h *ShopHandler) CreateOrder(c *gin.Context) {
	var req struct {
		ProductID     int64  `json:"product_id" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := c.GetInt64("user_id")
	order, payURL, err := h.shopService.CreateOrder(c.Request.Context(), userID, req.ProductID, req.PaymentMethod)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"order":   dto.ShopOrderFromService(order),
		"pay_url": payURL,
	})
}

// QueryOrder GET /api/v1/shop/orders/:orderNo
func (h *ShopHandler) QueryOrder(c *gin.Context) {
	orderNo := c.Param("orderNo")
	userID := c.GetInt64("user_id")
	order, err := h.shopService.QueryUserOrder(c.Request.Context(), userID, orderNo)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ShopOrderFromService(order))
}

// CreemNotify POST /api/v1/shop/notify/creem
func (h *ShopHandler) CreemNotify(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(400)
		return
	}
	signature := c.GetHeader("creem-signature")
	clientIP := c.ClientIP()
	if err := h.shopService.HandleCreemWebhook(c.Request.Context(), rawBody, signature, clientIP); err != nil {
		c.Status(400)
		return
	}
	c.Status(200)
}
func (h *ShopHandler) EpayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	params := make(map[string]string)
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	if len(params) == 0 {
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}
	clientIP := c.ClientIP()
	if err := h.shopService.HandlePaymentNotify(c.Request.Context(), params, clientIP); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	_, _ = c.Writer.Write([]byte("success"))
}

// ListUserOrders GET /api/v1/shop/my-orders
func (h *ShopHandler) ListUserOrders(c *gin.Context) {
	userID := c.GetInt64("user_id")
	status := c.Query("status")
	orders, err := h.shopService.GetUserOrders(c.Request.Context(), userID, status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.ShopOrder, 0, len(orders))
	for i := range orders {
		out = append(out, dto.ShopOrderFromService(&orders[i]))
	}
	response.Success(c, out)
}

// CancelOrderByUser POST /api/v1/shop/my-orders/:orderNo/cancel
func (h *ShopHandler) CancelOrderByUser(c *gin.Context) {
	userID := c.GetInt64("user_id")
	orderNo := c.Param("orderNo")
	if err := h.shopService.CancelOrder(c.Request.Context(), userID, orderNo); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "order cancelled"})
}
