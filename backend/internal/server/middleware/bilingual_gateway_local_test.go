package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newBilingualTestContext(path string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return c
}

func TestLocalizeGatewayMessage_GatewayPathsAreBilingual(t *testing.T) {
	want := "API Key 无效 / Invalid API key"
	for _, path := range []string{
		"/v1/messages",
		"/v1/chat/completions",
		"/v1beta/models/gemini:generateContent",
		"/api/v3/contents/generations/tasks",
		"/antigravity/v1/messages",
		"/responses",
	} {
		require.Equal(t, want, localizeGatewayMessage(newBilingualTestContext(path), "Invalid API key"), path)
	}
}

func TestLocalizeGatewayMessage_PanelPathsUntouched(t *testing.T) {
	for _, path := range []string{
		"/api/v1/auth/login",
		"/api/v1/admin/users",
		"/api/internal/anything",
	} {
		require.Equal(t, "User account is not active", localizeGatewayMessage(newBilingualTestContext(path), "User account is not active"), path)
	}
}

func TestLocalizeGatewayMessage_NilSafe(t *testing.T) {
	require.Equal(t, "Invalid API key", localizeGatewayMessage(nil, "Invalid API key"))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	require.Equal(t, "Invalid API key", localizeGatewayMessage(c, "Invalid API key"))
}

func TestGatewayErrorMessage_UsesApplicationMessage(t *testing.T) {
	appErr := infraerrors.TooManyRequests("DAILY_LIMIT_EXCEEDED", "daily usage limit exceeded")
	require.Equal(t, "daily usage limit exceeded", gatewayErrorMessage(appErr))
	require.Equal(t, "daily usage limit exceeded", gatewayErrorMessage(errors.Join(errors.New("wrap"), appErr)))
	require.Equal(t, "plain failure", gatewayErrorMessage(errors.New("plain failure")))
	require.Equal(t, "", gatewayErrorMessage(nil))
}
