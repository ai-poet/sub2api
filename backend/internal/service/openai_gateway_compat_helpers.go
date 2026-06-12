package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type httpUpstreamProfileKey struct{}

const HTTPUpstreamProfileOpenAI = "openai"

func WithHTTPUpstreamProfile(ctx context.Context, profile string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, httpUpstreamProfileKey{}, profile)
}

func HTTPUpstreamProfileFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(httpUpstreamProfileKey{}).(string)
	return v
}

func detachUpstreamContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.Background(), func() {}
	}
	return context.WithoutCancel(ctx), func() {}
}

func (s *OpenAIGatewayService) readUpstreamErrorBody(resp *http.Response) []byte {
	if resp == nil || resp.Body == nil {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return body
}

func (s *OpenAIGatewayService) handleOpenAIAccountUpstreamError(ctx context.Context, account *Account, statusCode int, headers http.Header, body []byte, _ ...string) {
	if s == nil || s.rateLimitService == nil || account == nil {
		return
	}
	s.rateLimitService.HandleUpstreamError(ctx, account, statusCode, headers, body)
}

func (a *Account) IsPoolModeRetryableStatus(statusCode int) bool {
	return isPoolModeRetryableStatus(statusCode)
}

type OpenAIFastBlockedError struct {
	Message string
}

func (e *OpenAIFastBlockedError) Error() string { return e.Message }

func (s *OpenAIGatewayService) applyOpenAIFastPolicyToBody(_ context.Context, _ *Account, _ string, body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	rawTier := normalizedOpenAIServiceTierValue(strings.TrimSpace(extractJSONServiceTier(body)))
	if rawTier == "" {
		return body, nil
	}
	return body, nil
}

func extractJSONServiceTier(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if tier := extractOpenAIServiceTierFromBody(body); tier != nil {
		return *tier
	}
	return ""
}

func writeOpenAIFastPolicyBlockedResponse(c *gin.Context, err *OpenAIFastBlockedError) {
	if c == nil || err == nil {
		return
	}
	MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"type":    "permission_error",
			"message": err.Message,
		},
	})
}

func buildOpenAIEndpointURL(base string, endpoint string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	if normalized == "" {
		return strings.TrimSpace(endpoint)
	}
	endpoint = "/" + strings.TrimLeft(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(normalized, endpoint) {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + strings.TrimPrefix(endpoint, "/v1")
	}
	return normalized + endpoint
}

func isContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
