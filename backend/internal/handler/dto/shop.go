package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// AdminShopProduct is the admin view of a shop product
type AdminShopProduct struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	Price          float64 `json:"price"`
	Currency       string  `json:"currency"`
	RedeemType     string  `json:"redeem_type"`
	RedeemValue    float64 `json:"redeem_value"`
	GroupID        *int64  `json:"group_id"`
	ValidityDays   int     `json:"validity_days"`
	StockCount     int     `json:"stock_count"`
	IsActive       bool    `json:"is_active"`
	SortOrder      int     `json:"sort_order"`
	CreemProductID string  `json:"creem_product_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// ShopProductStock is the DTO for a stock item
type ShopProductStock struct {
	ID           int64   `json:"id"`
	ProductID    int64   `json:"product_id"`
	RedeemCodeID int64   `json:"redeem_code_id"`
	Status       string  `json:"status"`
	OrderID      *int64  `json:"order_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// ShopOrder is the DTO for an order
type ShopOrder struct {
	ID            int64      `json:"id"`
	OrderNo       string     `json:"order_no"`
	UserID        int64      `json:"user_id"`
	ProductID     int64      `json:"product_id"`
	ProductName   string     `json:"product_name"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	PaymentMethod *string    `json:"payment_method"`
	Status        string     `json:"status"`
	RedeemCodeID  *int64     `json:"redeem_code_id"`
	PaidAt        *time.Time `json:"paid_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// CreateShopProductRequest is the request to create/update a product
type CreateShopProductRequest struct {
	Name           string  `json:"name" binding:"required,max=100"`
	Description    *string `json:"description"`
	Price          float64 `json:"price" binding:"required,gt=0"`
	Currency       string  `json:"currency"`
	RedeemType     string  `json:"redeem_type" binding:"required,oneof=balance subscription"`
	RedeemValue    float64 `json:"redeem_value"`
	GroupID        *int64  `json:"group_id"`
	ValidityDays   int     `json:"validity_days"`
	IsActive       bool    `json:"is_active"`
	SortOrder      int     `json:"sort_order"`
	CreemProductID string  `json:"creem_product_id"`
}

func (r *CreateShopProductRequest) ToService() *service.ShopProduct {
	currency := r.Currency
	if currency == "" {
		currency = "CNY"
	}
	validityDays := r.ValidityDays
	if validityDays == 0 {
		validityDays = 30
	}
	return &service.ShopProduct{
		Name:           r.Name,
		Description:    r.Description,
		Price:          r.Price,
		Currency:       currency,
		RedeemType:     r.RedeemType,
		RedeemValue:    r.RedeemValue,
		GroupID:        r.GroupID,
		ValidityDays:   validityDays,
		IsActive:       r.IsActive,
		SortOrder:      r.SortOrder,
		CreemProductID: r.CreemProductID,
	}
}

func AdminShopProductFromService(p *service.ShopProduct) AdminShopProduct {
	return AdminShopProduct{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Price:          p.Price,
		Currency:       p.Currency,
		RedeemType:     p.RedeemType,
		RedeemValue:    p.RedeemValue,
		GroupID:        p.GroupID,
		ValidityDays:   p.ValidityDays,
		StockCount:     p.StockCount,
		IsActive:       p.IsActive,
		SortOrder:      p.SortOrder,
		CreemProductID: p.CreemProductID,
		CreatedAt:      p.CreatedAt,
	}
}

func ShopProductStockFromService(s *service.ShopProductStock) ShopProductStock {
	return ShopProductStock{
		ID:           s.ID,
		ProductID:    s.ProductID,
		RedeemCodeID: s.RedeemCodeID,
		Status:       s.Status,
		OrderID:      s.OrderID,
		CreatedAt:    s.CreatedAt,
	}
}

func ShopOrderFromService(o *service.ShopOrder) ShopOrder {
	return ShopOrder{
		ID:            o.ID,
		OrderNo:       o.OrderNo,
		UserID:        o.UserID,
		ProductID:     o.ProductID,
		ProductName:   o.ProductName,
		Amount:        o.Amount,
		Currency:      o.Currency,
		PaymentMethod: o.PaymentMethod,
		Status:        o.Status,
		RedeemCodeID:  o.RedeemCodeID,
		PaidAt:        o.PaidAt,
		ExpiresAt:     o.ExpiresAt,
		CreatedAt:     o.CreatedAt,
	}
}
