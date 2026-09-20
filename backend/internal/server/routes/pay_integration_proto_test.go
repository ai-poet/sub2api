package routes

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPayProxyForwardedProtoPrefersUpstreamHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/pay/api/orders", nil)
	require.Equal(t, "http", payProxyForwardedProto(req), "直连且无 TLS 时回退 http")

	req.Header.Set("X-Forwarded-Proto", "https")
	require.Equal(t, "https", payProxyForwardedProto(req), "边缘终止 TLS 后必须沿用上游的 https")

	req.Header.Set("X-Forwarded-Proto", " HTTPS , http")
	require.Equal(t, "https", payProxyForwardedProto(req), "多级代理取第一个值，大小写不敏感")

	req.Header.Set("X-Forwarded-Proto", "gopher")
	require.Equal(t, "http", payProxyForwardedProto(req), "非法值按无头处理")

	req.TLS = &tls.ConnectionState{}
	require.Equal(t, "https", payProxyForwardedProto(req), "无头时才看 req.TLS")

	req.Header.Set("X-Forwarded-Proto", "http")
	require.Equal(t, "http", payProxyForwardedProto(req), "上游明确说 http 时不按 TLS 覆盖")
}

// 端到端：经网关反代到支付服务的请求，X-Forwarded-Proto 必须是边缘设置的 https 而不是被降级成 http。
func TestPayProxyKeepsUpstreamForwardedProto(t *testing.T) {
	var seen http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	t.Setenv("SUB2APIPAY_INTERNAL_URL", upstream.URL)

	r := gin.New()
	registerPayProxyRoutes(r)
	// ReverseProxy 需要真实的 ResponseWriter（gin 1.9 会调用 CloseNotify），用真实监听而不是 Recorder
	gateway := httptest.NewServer(r)
	defer gateway.Close()

	req, err := http.NewRequest(http.MethodGet, gateway.URL+"/pay/api/orders/abc", nil)
	require.NoError(t, err)
	req.Host = "pay.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NotNil(t, seen)
	require.Equal(t, "https", seen.Get("X-Forwarded-Proto"))
	require.Equal(t, "pay.example.com", seen.Get("X-Forwarded-Host"))
	require.Equal(t, "/pay", seen.Get("X-Forwarded-Prefix"))
}
