package creem

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	prodAPIURL = "https://api.creem.io/v1"
	testAPIURL = "https://test-api.creem.io/v1"
)

type Client struct {
	apiKey     string
	webhookSecret string
	baseURL    string
	httpClient *http.Client
}

type Config struct {
	APIKey        string
	WebhookSecret string
	TestMode      bool
}

func NewClient(cfg Config) *Client {
	base := prodAPIURL
	if cfg.TestMode {
		base = testAPIURL
	}
	return &Client{
		apiKey:        cfg.APIKey,
		webhookSecret: cfg.WebhookSecret,
		baseURL:       base,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

type CheckoutRequest struct {
	ProductID string            `json:"product_id"`
	RequestID string            `json:"request_id"`
	Customer  struct {
		Email string `json:"email"`
	} `json:"customer"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

func (c *Client) CreateCheckout(ctx context.Context, req CheckoutRequest) (*CheckoutResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/checkouts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("creem: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("creem: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result CheckoutResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("creem: decode response: %w", err)
	}
	return &result, nil
}

// WebhookEvent is the creem webhook payload
type WebhookEvent struct {
	ID        string `json:"id"`
	EventType string `json:"eventType"`
	Object    struct {
		RequestID string `json:"request_id"`
		Order     struct {
			Status string `json:"status"`
		} `json:"order"`
	} `json:"object"`
}

// VerifyWebhook verifies the creem-signature header and returns the parsed event.
// Pass rawBody as the raw request body bytes.
func (c *Client) VerifyWebhook(rawBody []byte, signature string) (*WebhookEvent, error) {
	if c.webhookSecret == "" {
		return nil, fmt.Errorf("creem: webhook secret is required")
	}
	h := hmac.New(sha256.New, []byte(c.webhookSecret))
	h.Write(rawBody)
	expected := hex.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return nil, fmt.Errorf("creem: invalid webhook signature")
	}
	var event WebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return nil, fmt.Errorf("creem: decode webhook: %w", err)
	}
	return &event, nil
}

// GenerateOrderNo generates a unique order reference ID for creem with random suffix
func GenerateOrderNo(userID int64) string {
	b := make([]byte, 4)
	rand.Read(b)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d%d%x", userID, time.Now().UnixNano(), b)))
	return fmt.Sprintf("ref_%s", hex.EncodeToString(h.Sum(nil))[:16])
}
