package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
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
	ErrOrderNotPending    = infraerrors.Conflict("ORDER_NOT_PENDING", "order is not pending")
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
	Delete(ctx context.Context, id int64) (productID int64, deleted bool, err error)
	// TakeOne atomically takes one available stock item for a product (FIFO)
	TakeOne(ctx context.Context, productID int64, orderID int64) (*ShopProductStock, error)
	CountAvailable(ctx context.Context, productID int64) (int, error)
}

// ShopOrderRepository defines data access for orders
type ShopOrderRepository interface {
	Create(ctx context.Context, o *ShopOrder) error
	Update(ctx context.Context, o *ShopOrder) error
	GetByOrderNo(ctx context.Context, orderNo string) (*ShopOrder, error)
	GetByOrderNoForUpdate(ctx context.Context, orderNo string) (*ShopOrder, error)
	ListByUser(ctx context.Context, userID int64) ([]ShopOrder, error)
	ListByUserAndStatus(ctx context.Context, userID int64, status string) ([]ShopOrder, error)
}

// PaymentCallbackLogRepository defines data access for payment callback logs
type PaymentCallbackLogRepository interface {
	Create(ctx context.Context, log *PaymentCallbackLog) error
	GetByOrderNo(ctx context.Context, orderNo string) ([]PaymentCallbackLog, error)
	UpdateProcessed(ctx context.Context, id int64, processed bool, resultMessage string) error
}

// PaymentCallbackLog represents a payment callback log entry
type PaymentCallbackLog struct {
	ID            int64
	OrderNo       string
	Provider      string // epay, creem
	RawData       map[string]string
	Signature     *string
	Verified      bool
	Processed     bool
	ResultMessage *string
	ClientIP      *string
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

// ShopService handles shop business logic
type ShopService struct {
	productRepo ShopProductRepository
	stockRepo   ShopProductStockRepository
	orderRepo   ShopOrderRepository
	callbackLog PaymentCallbackLogRepository
	redeemSvc   *RedeemService
	settingSvc  *SettingService
	db          *sql.DB
}

func NewShopService(
	productRepo ShopProductRepository,
	stockRepo ShopProductStockRepository,
	orderRepo ShopOrderRepository,
	callbackLog PaymentCallbackLogRepository,
	redeemSvc *RedeemService,
	settingSvc *SettingService,
	db *sql.DB,
) *ShopService {
	return &ShopService{
		productRepo: productRepo,
		stockRepo:   stockRepo,
		orderRepo:   orderRepo,
		callbackLog: callbackLog,
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
	productID, deleted, err := s.stockRepo.Delete(ctx, stockID)
	if err != nil {
		return err
	}
	if !deleted {
		return infraerrors.NotFound("STOCK_NOT_FOUND", "stock not found or already sold")
	}
	if err := s.productRepo.UpdateStockCount(ctx, productID, -1); err != nil {
		return fmt.Errorf("update stock count: %w", err)
	}
	return nil
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

	// Always use real-time stock to prevent stale stock_count from allowing unpaid orders.
	availableCount, err := s.stockRepo.CountAvailable(ctx, productID)
	if err != nil {
		return nil, "", fmt.Errorf("count available stock: %w", err)
	}
	if availableCount <= 0 {
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
		// Creem requires API key + webhook secret, otherwise callback cannot be verified safely.
		if settings.CreemAPIKey == "" || settings.CreemWebhookSecret == "" {
			return nil, "", ErrCreemNotConfigured
		}
		if strings.TrimSpace(product.CreemProductID) == "" {
			return nil, "", infraerrors.BadRequest("CREEM_PRODUCT_NOT_CONFIGURED", "creem product id is not configured")
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

// QueryUserOrder returns an order only when it belongs to the specified user.
func (s *ShopService) QueryUserOrder(ctx context.Context, userID int64, orderNo string) (*ShopOrder, error) {
	order, err := s.orderRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	// Hide cross-user order existence.
	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

// fulfillOrder atomically takes stock, redeems code, and marks order as paid.
func (s *ShopService) fulfillOrder(ctx context.Context, orderNo string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := context.WithValue(ctx, ShopSQLTxKey{}, tx)

	// Lock order row first so concurrent callbacks become idempotent.
	order, err := s.orderRepo.GetByOrderNoForUpdate(txCtx, orderNo)
	if err != nil {
		return fmt.Errorf("lock order: %w", err)
	}
	if order.Status == "paid" {
		return nil
	}
	if order.Status != "pending" {
		return ErrOrderNotPending
	}

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
func (s *ShopService) HandlePaymentNotify(ctx context.Context, params map[string]string, clientIP string) error {
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
		// Log failed verification
		s.logCallback(ctx, orderNo, "epay", params, nil, false, false, err.Error(), clientIP)
		return fmt.Errorf("verify: %w", err)
	}
	if !ok {
		// Log invalid signature
		s.logCallback(ctx, orderNo, "epay", params, nil, false, false, "invalid signature", clientIP)
		return nil
	}

	// Log successful verification
	s.logCallback(ctx, orderNo, "epay", params, nil, true, false, "", clientIP)

	if err := s.fulfillOrder(ctx, orderNo); err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			s.updateCallbackResult(ctx, orderNo, false, "order not found")
			return nil
		}
		if errors.Is(err, ErrOrderNotPending) {
			s.updateCallbackResult(ctx, orderNo, false, "order is not pending")
			return nil
		}
		s.updateCallbackResult(ctx, orderNo, false, err.Error())
		return err
	}

	s.updateCallbackResult(ctx, orderNo, true, "order fulfilled")
	return nil
}

// HandleCreemWebhook processes creem payment webhook
func (s *ShopService) HandleCreemWebhook(ctx context.Context, rawBody []byte, signature string, clientIP string) error {
	settings, err := s.settingSvc.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	if settings.CreemWebhookSecret == "" {
		s.logCallback(ctx, "", "creem", map[string]string{"raw": string(rawBody)}, &signature, false, false, "webhook secret not configured", clientIP)
		return ErrCreemNotConfigured
	}
	client := creemClient.NewClient(creemClient.Config{
		APIKey:        settings.CreemAPIKey,
		WebhookSecret: settings.CreemWebhookSecret,
		TestMode:      settings.CreemTestMode,
	})
	event, err := client.VerifyWebhook(rawBody, signature)
	if err != nil {
		// Log failed verification
		s.logCallback(ctx, "", "creem", map[string]string{"raw": string(rawBody)}, &signature, false, false, err.Error(), clientIP)
		return fmt.Errorf("verify webhook: %w", err)
	}

	orderNo := event.Object.RequestID
	orderStatus := event.Object.Order.Status

	// Log successful verification
	s.logCallback(ctx, orderNo, "creem", map[string]string{"raw": string(rawBody)}, &signature, true, false, "", clientIP)

	if orderStatus != "paid" {
		return nil
	}
	if err := s.fulfillOrder(ctx, orderNo); err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			s.updateCallbackResult(ctx, orderNo, false, "order not found")
			return nil
		}
		if errors.Is(err, ErrOrderNotPending) {
			s.updateCallbackResult(ctx, orderNo, false, "order is not pending")
			return nil
		}
		s.updateCallbackResult(ctx, orderNo, false, err.Error())
		return err
	}

	s.updateCallbackResult(ctx, orderNo, true, "order fulfilled")
	return nil
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

	// Check if Creem is configured (must include webhook secret for secure callback verification)
	if settings.CreemAPIKey != "" && settings.CreemWebhookSecret != "" {
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

// CleanupExpiredOrders marks expired pending orders as cancelled
// This should be called periodically by a scheduler
func (s *ShopService) CleanupExpiredOrders(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE shop_orders 
		SET status = 'expired', updated_at = NOW() 
		WHERE status = 'pending' AND expires_at IS NOT NULL AND expires_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired orders: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("[ShopService] cleaned up %d expired orders", affected)
	}
	return affected, nil
}

// GetUserOrders returns orders for a user with optional status filter
func (s *ShopService) GetUserOrders(ctx context.Context, userID int64, status string) ([]ShopOrder, error) {
	if status != "" {
		return s.orderRepo.ListByUserAndStatus(ctx, userID, status)
	}
	return s.orderRepo.ListByUser(ctx, userID)
}

// CancelOrder cancels a pending order (user-initiated)
func (s *ShopService) CancelOrder(ctx context.Context, userID int64, orderNo string) error {
	order, err := s.orderRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return err
	}
	if order.UserID != userID {
		return infraerrors.Forbidden("ORDER_NOT_OWNER", "order does not belong to user")
	}
	if order.Status != "pending" {
		return infraerrors.BadRequest("ORDER_NOT_PENDING", "only pending orders can be cancelled")
	}
	order.Status = "cancelled"
	return s.orderRepo.Update(ctx, order)
}

// generateRandomSuffix generates a random hex suffix for order numbers
func generateRandomSuffix() string {
	b := make([]byte, 2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// logCallback records a payment callback log entry
func (s *ShopService) logCallback(ctx context.Context, orderNo, provider string, rawData map[string]string, signature *string, verified bool, processed bool, resultMessage string, clientIP string) {
	if s.callbackLog == nil {
		return
	}

	entry := &PaymentCallbackLog{
		OrderNo:       orderNo,
		Provider:      provider,
		RawData:       rawData,
		Signature:     signature,
		Verified:      verified,
		Processed:     processed,
		ResultMessage: &resultMessage,
		ClientIP:      &clientIP,
		CreatedAt:     time.Now(),
	}

	if err := s.callbackLog.Create(ctx, entry); err != nil {
		log.Printf("[ShopService] failed to log callback: %v", err)
	}
}

// updateCallbackResult updates the processing result of a callback log
func (s *ShopService) updateCallbackResult(ctx context.Context, orderNo string, processed bool, resultMessage string) {
	if s.callbackLog == nil {
		return
	}

	logs, err := s.callbackLog.GetByOrderNo(ctx, orderNo)
	if err != nil || len(logs) == 0 {
		return
	}

	// Update the most recent log entry
	latestLog := logs[0]
	if err := s.callbackLog.UpdateProcessed(ctx, latestLog.ID, processed, resultMessage); err != nil {
		log.Printf("[ShopService] failed to update callback result: %v", err)
	}
}
