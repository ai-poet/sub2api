package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
