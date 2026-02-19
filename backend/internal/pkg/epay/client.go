package epay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"time"

	goepay "github.com/Calcium-Ion/go-epay/epay"
)

// Client wraps go-epay client with settings-based config
type Client struct {
	inner *goepay.Client
}

// Config holds epay configuration
type Config struct {
	PID    string
	Key    string
	APIURL string
}

// NewClient creates a new epay client from config
func NewClient(cfg Config) (*Client, error) {
	if cfg.PID == "" || cfg.Key == "" || cfg.APIURL == "" {
		return nil, fmt.Errorf("epay: pid, key and api_url are required")
	}
	inner, err := goepay.NewClient(&goepay.Config{
		PartnerID: cfg.PID,
		Key:       cfg.Key,
	}, cfg.APIURL)
	if err != nil {
		return nil, fmt.Errorf("epay: create client: %w", err)
	}
	return &Client{inner: inner}, nil
}

// PurchaseResult holds the result of a purchase request
type PurchaseResult struct {
	PayURL string
	Params map[string]string
}

// Purchase creates a payment request
func (c *Client) Purchase(ctx context.Context, orderNo, name, payMethod, callbackBase string, amount float64) (*PurchaseResult, error) {
	notifyURL, err := url.Parse(callbackBase + "/api/v1/shop/notify/epay")
	if err != nil {
		return nil, fmt.Errorf("epay: parse notify url: %w", err)
	}
	returnURL, err := url.Parse(callbackBase + "/shop")
	if err != nil {
		return nil, fmt.Errorf("epay: parse return url: %w", err)
	}

	uri, params, err := c.inner.Purchase(&goepay.PurchaseArgs{
		Type:           payMethod,
		ServiceTradeNo: orderNo,
		Name:           name,
		Money:          strconv.FormatFloat(amount, 'f', 2, 64),
		Device:         goepay.PC,
		NotifyUrl:      notifyURL,
		ReturnUrl:      returnURL,
	})
	if err != nil {
		return nil, fmt.Errorf("epay: purchase: %w", err)
	}
	return &PurchaseResult{PayURL: uri.String(), Params: params}, nil
}

// Verify verifies a payment notification
func (c *Client) Verify(params map[string]string) (tradeNo string, ok bool, err error) {
	info, err := c.inner.Verify(params)
	if err != nil {
		return "", false, err
	}
	if !info.VerifyStatus {
		return "", false, nil
	}
	if info.TradeStatus != goepay.StatusTradeSuccess {
		return info.ServiceTradeNo, false, nil
	}
	return info.ServiceTradeNo, true, nil
}

// GenerateOrderNo generates a unique order number with random suffix
func GenerateOrderNo(userID int64) string {
	b := make([]byte, 2)
	rand.Read(b)
	randomSuffix := hex.EncodeToString(b)
	return fmt.Sprintf("SHOP%d%d%s", userID, time.Now().UnixNano(), randomSuffix)
}
