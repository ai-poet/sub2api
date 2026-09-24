package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestInternalPayAuthMiddlewareAcceptsConfiguredSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	router := gin.New()
	router.Use(internalPayAuthMiddleware(strings.Repeat("a", 32)))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(internalPayTokenHeader, deriveInternalPayToken(strings.Repeat("a", 32)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestInternalPayAuthMiddlewareAcceptsEnvSecretFallback(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("b", 32))

	router := gin.New()
	router.Use(internalPayAuthMiddleware(strings.Repeat("a", 32)))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(internalPayTokenHeader, deriveInternalPayToken(strings.Repeat("b", 32)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestInternalPayNeutralSessionBindingClearsFingerprint(t *testing.T) {
	router := gin.New()
	// Stand-in for the global SessionBindingContext: sub2apipay's own IP/UA.
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(service.WithSessionBinding(c.Request.Context(), &service.SessionBinding{
			IP:        "127.0.0.1",
			UserAgent: "node",
		}))
		c.Next()
	})
	router.Use(internalPayNeutralSessionBinding())
	var hash string
	router.GET("/test", func(c *gin.Context) {
		hash = service.SessionBindingFromContext(c.Request.Context()).Hash()
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, hash)
}

func TestInternalPayAuthMiddlewareRejectsUnknownSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("b", 32))

	router := gin.New()
	router.Use(internalPayAuthMiddleware(strings.Repeat("a", 32)))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(internalPayTokenHeader, deriveInternalPayToken(strings.Repeat("c", 32)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
