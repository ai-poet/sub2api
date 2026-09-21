package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGitHubOAuthStatePayload_JSONRoundTrip(t *testing.T) {
	p := gitHubOAuthStatePayload{
		N: "nonce-value",
		R: "/auth/paseo?endpoint=https%3A%2F%2Fexample.com",
		A: "REF123",
	}
	raw, err := json.Marshal(p)
	require.NoError(t, err)

	var decoded gitHubOAuthStatePayload
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, p.N, decoded.N)
	require.Equal(t, p.R, decoded.R)
	require.Equal(t, p.A, decoded.A)
}

// 老版本的 state cookie 只有 n / r：解码后推荐码为空，而不是解不开；没有码时也不写 a。
func TestGitHubOAuthStatePayload_LegacyWithoutAffiliateCode(t *testing.T) {
	var decoded gitHubOAuthStatePayload
	require.NoError(t, json.Unmarshal([]byte(`{"n":"nonce-value","r":"/dashboard"}`), &decoded))
	require.Equal(t, "nonce-value", decoded.N)
	require.Equal(t, "/dashboard", decoded.R)
	require.Empty(t, decoded.A)

	raw, err := json.Marshal(gitHubOAuthStatePayload{N: "n", R: "/r"})
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"a"`)
}

// settingSvc 置空走 config 回退（同 configureLinuxDoOAuthTestHandler 的手法），
// GitHubOAuthStart 只需要 client id / secret 与回调地址就能生成跳转。
func newGitHubOAuthStartTestHandler(t *testing.T) *AuthHandler {
	t.Helper()
	handler, client := newOAuthPendingFlowTestHandler(t, false)
	t.Cleanup(func() { _ = client.Close() })
	handler.settingSvc = nil
	handler.cfg = &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "test-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   7,
		},
		GitHub: config.GitHubOAuthConfig{
			Enabled:      true,
			ClientID:     "github-client",
			ClientSecret: "github-secret",
			RedirectURL:  "https://api.example.com/api/v1/auth/oauth/github/callback",
		},
	}
	return handler
}

func startGitHubOAuth(t *testing.T, handler *AuthHandler, target string) (*httptest.ResponseRecorder, gitHubOAuthStatePayload) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)

	handler.GitHubOAuthStart(c)

	require.Equal(t, http.StatusFound, recorder.Code, recorder.Body.String())
	cookie := findCookie(recorder.Result().Cookies(), gitHubOAuthStateCookieName)
	require.NotNil(t, cookie)
	var payload gitHubOAuthStatePayload
	require.NoError(t, json.Unmarshal([]byte(decodeCookieValueForTest(t, cookie.Value)), &payload))
	return recorder, payload
}

// 邀请链接 /register?ref=<码> 来的 GitHub 注册：前端把码放在 start 的 aff_code 上，
// 后端必须随 state cookie 带到回调，否则建号时 bindOAuthAffiliate 拿到的是空串。
func TestGitHubOAuthStartStoresAffiliateCodeInStateCookie(t *testing.T) {
	handler := newGitHubOAuthStartTestHandler(t)

	recorder, payload := startGitHubOAuth(t, handler, "/api/v1/auth/oauth/github/start?redirect=/dashboard&aff_code=REF123")

	require.Equal(t, "REF123", payload.A)
	require.Equal(t, "/dashboard", payload.R)
	require.NotEmpty(t, payload.N)

	location, err := url.Parse(recorder.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "github.com", location.Host)
	require.Equal(t, payload.N, location.Query().Get("state"))
}

func TestGitHubOAuthStartAcceptsAffAliasAndCapsLength(t *testing.T) {
	handler := newGitHubOAuthStartTestHandler(t)

	_, payload := startGitHubOAuth(t, handler, "/api/v1/auth/oauth/github/start?aff=REF456")
	require.Equal(t, "REF456", payload.A)

	_, payload = startGitHubOAuth(t, handler, "/api/v1/auth/oauth/github/start?aff_code="+strings.Repeat("x", 100))
	require.Equal(t, strings.Repeat("x", oauthAffiliateCodeMaxLen), payload.A)
}

func TestGitHubOAuthStartLeavesAffiliateCodeEmptyWhenAbsent(t *testing.T) {
	handler := newGitHubOAuthStartTestHandler(t)

	recorder, payload := startGitHubOAuth(t, handler, "/api/v1/auth/oauth/github/start?redirect=/dashboard")
	require.Empty(t, payload.A)

	cookie := findCookie(recorder.Result().Cookies(), gitHubOAuthStateCookieName)
	require.NotNil(t, cookie)
	require.NotContains(t, decodeCookieValueForTest(t, cookie.Value), `"a"`)
}
