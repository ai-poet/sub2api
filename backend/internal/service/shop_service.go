package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	creemClient "github.com/Wei-Shaw/sub2api/internal/pkg/creem"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	epayClient "github.com/Wei-Shaw/sub2api/internal/pkg/epay"
)

var (
	ErrProductNotFound    = infraerrors.NotFound("PRODUCT_NOT_FOUND", "product not found")
	ErrProductOutOfStock  = infraerrors.BadRequest("PRODUCT_OUT_OF_STOCK", "product out of stock")
	ErrOrderNotFound      = infraerrors.NotFound("ORDER_NOT_FOUND", "order not found")
	ErrOrderAlreadyPaid   = infraerrors.Conflict("ORDER_ALREADY_PAID", "order already paid")
	ErrEpayNotConfigured  = infraerrors.BadRequest("EPAY_NOT_CONFIGURED", "payment not configured")
	ErrCreemNotConfigured = infraerrors.BadRequest("CREEM_NOT_CONFIGURED", "creem payment not configured")
)

// PaymentChannel represents a payment channel
type PaymentChannel struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Icon     string  `json:"icon"`
	Provider string  `json:"provider"`
	Fee      float64 `json:"fee"`
}

// USDTNetworks maps USDT network channel IDs to their display names
var USDTNetworks = map[string]struct {
	Name string
	Icon string
}{
	"usdt.plasma": {Name: "USDT-Plasma", Icon: "credit-card"},
	"usdt.polygon": {Name: "USDT-Polygon", Icon: "credit-card"},
	"usdt.trc20":  {Name: "USDT-TRC20", Icon: "credit-card"},
	"usdt.erc20":  {Name: "USDT-ERC20", Icon: "credit-card"},
}

// ShopProduct represents a shop product
type ShopProduct struct {
	ID             int64
	Name           string
	Description    *string
	Price          float64
	Currency       string
	RedeemType     string
	RedeemValue    float64
	GroupID        *int64
	ValidityDays   int
	StockCount     int
	IsActive       bool
	SortOrder      int
	CreemProductID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ShopProductStock represents a stock item
type ShopProductStock struct {
	ID           int64
	ProductID    int64
	RedeemCodeID int64
	Status       string
	OrderID      *int64
	CreatedAt    time.Time
}

// ShopOrder represents a shop order
type ShopOrder struct {
	ID            int64
	OrderNo       string
	UserID        int64
	ProductID     int64
	ProductName   string
	Amount        float64
	Currency      string
	PaymentMethod *string
	Status        string
	RedeemCodeID  *int64
	PaidAt        *time.Time
	ExpiresAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ShopProductRepository defines data access for shop products
type ShopProductRepository interface {
	Create(ctx context.Context, p *ShopProduct) error
	Update(ctx context.Context, p *ShopProduct) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*ShopProduct, error)
	List(ctx context.Context, activeOnly bool) ([]ShopProduct, error)
	UpdateStockCount(ctx context.Context, id int64, delta int) error
}

// ShopProductStockRepository defines data access for stock
type ShopProductStockRepository interface {
	CreateBatch(ctx context.Context, stocks []ShopProductStock) error
	ListByProduct(ctx context.Context, productID int64) ([]ShopProductStock, error)
	Delete(ctx context.Context, id int64) error
	// TakeOne atomically takes one available stock item for a product (FIFO)
	TakeOne(ctx context.Context, productID int64, orderID int64) (*ShopProductStock, error)
	CountAvailable(ctx context.Context, productID int64) (int, error)
}

// ShopOrderRepository defines data access for orders
type ShopOrderRepository interface {
	Create(ctx context.Context, o *ShopOrder) error
	Update(ctx context.Context, o *ShopOrder) error
	GetByOrderNo(ctx context.Context, orderNo string) (*ShopOrder, error)
	ListByUser(ctx context.Context, userID int64) ([]ShopOrder, error)
}

// ShopService handles shop business logic
type ShopService struct {
	productRepo ShopProductRepository
	stockRepo   ShopProductStockRepository
	orderRepo   ShopOrderRepository
	redeemSvc   *RedeemService
	settingSvc  *SettingService
	db          *sql.DB
}

func NewShopService(
	productRepo ShopProductRepository,
	stockRepo ShopProductStockRepository,
	orderRepo ShopOrderRepository,
	redeemSvc *RedeemService,
	settingSvc *SettingService,
	db *sql.DB,
) *ShopService {
	return &ShopService{
		productRepo: productRepo,
		stockRepo:   stockRepo,
		orderRepo:   orderRepo,
		redeemSvc:   redeemSvc,
		settingSvc:  settingSvc,
		db:          db,
	}
}

// --- Admin methods ---

func (s *ShopService) CreateProduct(ctx context.Context, p *ShopProduct) (*ShopProduct, error) {
	if err := s.productRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return p, nil
}

func (s *ShopService) UpdateProduct(ctx context.Context, p *ShopProduct) (*ShopProduct, error) {
	if err := s.productRepo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	return p, nil
}

func (s *ShopService) DeleteProduct(ctx context.Context, id int64) error {
	return s.productRepo.Delete(ctx, id)
}

func (s *ShopService) ListAllProducts(ctx context.Context) ([]ShopProduct, error) {
	return s.productRepo.List(ctx, false)
}

// AddStock generates redeem codes and adds them to product stock
func (s *ShopService) AddStock(ctx context.Context, productID int64, count int) (int, error) {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return 0, err
	}

	req := GenerateCodesRequest{
		Count: count,
		Type:  product.RedeemType,
		Value: product.RedeemValue,
	}
	// For subscription type, set group_id and validity_days via notes (handled in GenerateCodes)
	codes, err := s.redeemSvc.GenerateCodesForShop(ctx, req, product.GroupID, product.ValidityDays)
	if err != nil {
		return 0, fmt.Errorf("generate codes: %w", err)
	}

	stocks := make([]ShopProductStock, 0, len(codes))
	for _, c := range codes {
		stocks = append(stocks, ShopProductStock{
			ProductID:    productID,
			RedeemCodeID: c.ID,
			Status:       "available",
		})
	}
	if err := s.stockRepo.CreateBatch(ctx, stocks); err != nil {
		return 0, fmt.Errorf("create stock: %w", err)
	}
	if err := s.productRepo.UpdateStockCount(ctx, productID, len(codes)); err != nil {
		return 0, fmt.Errorf("update stock count: %w", err)
	}
	return len(codes), nil
}

func (s *ShopService) GetStockList(ctx context.Context, productID int64) ([]ShopProductStock, error) {
	return s.stockRepo.ListByProduct(ctx, productID)
}

func (s *ShopService) DeleteStock(ctx context.Context, stockID int64) error {
	return s.stockRepo.Delete(ctx, stockID)
}

// --- User methods ---

func (s *ShopService) GetActiveProducts(ctx context.Context) ([]ShopProduct, error) {
	return s.productRepo.List(ctx, true)
}

// CreateOrder creates an order and returns the order + pay URL
func (s *ShopService) CreateOrder(ctx context.Context, userID int64, productID int64, paymentMethod string) (*ShopOrder, string, error) {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, "", err
	}
	if !product.IsActive {
		return nil, "", ErrProductNotFound
	}
	if product.StockCount <= 0 {
		return nil, "", ErrProductOutOfStock
	}

	settings, err := s.settingSvc.GetAllSettings(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get settings: %w", err)
	}

	expiresAt := time.Now().Add(30 * time.Minute)
	order := &ShopOrder{
		UserID:        userID,
		ProductID:     productID,
		ProductName:   product.Name,
		Amount:        product.Price,
		Currency:      product.Currency,
		PaymentMethod: &paymentMethod,
		Status:        "pending",
		ExpiresAt:     &expiresAt,
	}

	var payURL string
	if paymentMethod == "creem" {
		// Creem 支付
		if settings.CreemAPIKey == "" {
			return nil, "", ErrCreemNotConfigured
		}
		order.OrderNo = creemClient.GenerateOrderNo(userID)
		if err := s.orderRepo.Create(ctx, order); err != nil {
			return nil, "", fmt.Errorf("create order: %w", err)
		}
		client := creemClient.NewClient(creemClient.Config{
			APIKey:        settings.CreemAPIKey,
			WebhookSecret: settings.CreemWebhookSecret,
			TestMode:      settings.CreemTestMode,
		})
		resp, err := client.CreateCheckout(ctx, creemClient.CheckoutRequest{
			ProductID: product.CreemProductID,
			RequestID: order.OrderNo,
		})
		if err != nil {
			return nil, "", fmt.Errorf("creem checkout: %w", err)
		}
		payURL = resp.CheckoutURL
	} else {
		// 易支付
		if settings.EpayPID == "" || settings.EpayKey == "" || settings.EpayAPIURL == "" {
			return nil, "", ErrEpayNotConfigured
		}
		order.OrderNo = epayClient.GenerateOrderNo(userID)
		if err := s.orderRepo.Create(ctx, order); err != nil {
			return nil, "", fmt.Errorf("create order: %w", err)
		}
		client, err := epayClient.NewClient(epayClient.Config{
			PID:    settings.EpayPID,
			Key:    settings.EpayKey,
			APIURL: settings.EpayAPIURL,
		})
		if err != nil {
			return nil, "", fmt.Errorf("create epay client: %w", err)
		}
		result, err := client.Purchase(ctx, order.OrderNo, product.Name, paymentMethod, settings.APIBaseURL, product.Price)
		if err != nil {
			return nil, "", fmt.Errorf("create payment: %w", err)
		}
		payURL = result.PayURL
	}

	return order, payURL, nil
}

func (s *ShopService) QueryOrder(ctx context.Context, orderNo string) (*ShopOrder, error) {
	return s.orderRepo.GetByOrderNo(ctx, orderNo)
}

// fulfillOrder atomically takes stock, redeems code, and marks order as paid.
// Must be called within a transaction context or will use its own transaction.
func (s *ShopService) fulfillOrder(ctx context.Context, order *ShopOrder) error {
	if order.Status == "paid" {
		return nil // idempotent
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := context.WithValue(ctx, ShopSQLTxKey{}, tx)

	stock, err := s.stockRepo.TakeOne(txCtx, order.ProductID, order.ID)
	if err != nil {
		return fmt.Errorf("take stock: %w", err)
	}

	redeemCode, err := s.redeemSvc.GetByID(ctx, stock.RedeemCodeID)
	if err != nil {
		return fmt.Errorf("get redeem code: %w", err)
	}
	if _, err := s.redeemSvc.Redeem(ctx, order.UserID, redeemCode.Code); err != nil {
		return fmt.Errorf("redeem: %w", err)
	}

	now := time.Now()
	order.Status = "paid"
	order.PaidAt = &now
	order.RedeemCodeID = &stock.RedeemCodeID
	if err := s.orderRepo.Update(txCtx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if err := s.productRepo.UpdateStockCount(txCtx, order.ProductID, -1); err != nil {
		return fmt.Errorf("update stock count: %w", err)
	}
	return tx.Commit()
}

// HandlePaymentNotify processes epay payment callback
func (s *ShopService) HandlePaymentNotify(ctx context.Context, params map[string]string) error {
	settings, err := s.settingSvc.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	client, err := epayClient.NewClient(epayClient.Config{
		PID:    settings.EpayPID,
		Key:    settings.EpayKey,
		APIURL: settings.EpayAPIURL,
	})
	if err != nil {
		return fmt.Errorf("create epay client: %w", err)
	}
	orderNo, ok, err := client.Verify(params)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	if !ok {
		return nil
	}
	order, err := s.orderRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil
		}
		return fmt.Errorf("get order: %w", err)
	}
	return s.fulfillOrder(ctx, order)
}

// HandleCreemWebhook processes creem payment webhook
func (s *ShopService) HandleCreemWebhook(ctx context.Context, rawBody []byte, signature string) error {
	settings, err := s.settingSvc.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	client := creemClient.NewClient(creemClient.Config{
		APIKey:        settings.CreemAPIKey,
		WebhookSecret: settings.CreemWebhookSecret,
		TestMode:      settings.CreemTestMode,
	})
	event, err := client.VerifyWebhook(rawBody, signature)
	if err != nil {
		return fmt.Errorf("verify webhook: %w", err)
	}
	if event.Object.Order.Status != "paid" {
		return nil
	}
	order, err := s.orderRepo.GetByOrderNo(ctx, event.Object.RequestID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil
		}
		return fmt.Errorf("get order: %w", err)
	}
	return s.fulfillOrder(ctx, order)
}

// GetUserOrders returns orders for a user
func (s *ShopService) GetUserOrders(ctx context.Context, userID int64) ([]ShopOrder, error) {
	return s.orderRepo.ListByUser(ctx, userID)
}

// RedeemTypeBalance and RedeemTypeSubscription constants
const (
	ShopRedeemTypeBalance      = domain.RedeemTypeBalance
	ShopRedeemTypeSubscription = domain.RedeemTypeSubscription
)

// ShopSQLTxKey is the context key for passing *sql.Tx to shop repositories
type ShopSQLTxKey struct{}

// GetPaymentChannels returns available payment channels based on configuration
func (s *ShopService) GetPaymentChannels(ctx context.Context) ([]PaymentChannel, error) {
	settings, err := s.settingSvc.GetAllSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	var channels []PaymentChannel

	// Check if Creem is configured
	if settings.CreemAPIKey != "" {
		channels = append(channels, PaymentChannel{
			ID:       "creem",
			Name:     "Creem",
			Icon:     "credit-card",
			Provider: "creem",
			Fee:      0,
		})
	}

	// Check if Epay is configured
	if settings.EpayPID != "" && settings.EpayKey != "" && settings.EpayAPIURL != "" {
		// Parse enabled channels from configuration
		enabledChannels := settings.EpayChannels
		if enabledChannels == "" {
			enabledChannels = "alipay,wxpay" // Default channels
		}

		channelList := strings.Split(enabledChannels, ",")
		for _, ch := range channelList {
			ch = strings.TrimSpace(ch)
			if ch == "" {
				continue
			}

			switch ch {
			case "alipay":
				channels = append(channels, PaymentChannel{
					ID:       "alipay",
					Name:     "支付宝",
					Icon:     "wallet",
					Provider: "epay",
					Fee:      0,
				})
			case "wxpay":
				channels = append(channels, PaymentChannel{
					ID:       "wxpay",
					Name:     "微信支付",
					Icon:     "credit-card",
					Provider: "epay",
					Fee:      0,
				})
			default:
				// Check if it's a USDT network
				if network, ok := USDTNetworks[ch]; ok {
					channels = append(channels, PaymentChannel{
						ID:       ch,
						Name:     network.Name,
						Icon:     network.Icon,
						Provider: "epay",
						Fee:      0,
					})
				}
			}
		}
	}

	return channels, nil
}
