//go:build unit

package handler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type desktopLoginHandlerStore struct {
	mu      sync.Mutex
	next    int
	records map[string]service.DesktopLoginCode
}

func (s *desktopLoginHandlerStore) Store(_ context.Context, record *service.DesktopLoginCode, _ time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	code := fmt.Sprintf("K7QM-%04d", s.next)
	s.records[code] = *record
	return code, nil
}

func (s *desktopLoginHandlerStore) Consume(_ context.Context, code string) (*service.DesktopLoginCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[code]
	if !ok {
		return nil, service.ErrDesktopLoginCodeInvalid
	}
	delete(s.records, code)
	return &record, nil
}

type desktopLoginBackendModeStub bool

func (b desktopLoginBackendModeStub) IsBackendModeEnabled(context.Context) bool { return bool(b) }

type desktopLoginHandlerFixture struct {
	router *gin.Engine
	store  *desktopLoginHandlerStore
	h      *DesktopLoginHandler
}

// newDesktopLoginHandlerFixture 挂上两条路由；subjectUserID > 0 时模拟 jwtAuth 写入的登录主体。
func newDesktopLoginHandlerFixture(t *testing.T, role string, subjectUserID int64) *desktopLoginHandlerFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := &userHandlerRepoStub{user: &service.User{
		ID:           41,
		Email:        "desk-code@example.com",
		Username:     "desk-code",
		Role:         role,
		Status:       service.StatusActive,
		TokenVersion: 1,
	}}
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "desktop-login-secret", ExpireHour: 1, RefreshTokenExpireDays: 30}}
	authService := service.NewAuthService(nil, repo, nil, &userHandlerRefreshTokenCacheStub{}, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	store := &desktopLoginHandlerStore{records: map[string]service.DesktopLoginCode{}}
	h := NewDesktopLoginHandler(service.NewDesktopLoginService(store, authService), nil)

	router := gin.New()
	router.POST("/api/v1/auth/desktop-session/code", func(c *gin.Context) {
		if subjectUserID > 0 {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: subjectUserID, Concurrency: 1})
		}
		c.Next()
	}, h.CreateCode)
	router.POST("/api/v1/auth/desktop-session/exchange", h.Exchange)
	return &desktopLoginHandlerFixture{router: router, store: store, h: h}
}

func (f *desktopLoginHandlerFixture) post(path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	return w
}

func desktopLoginHandlerPKCE() (verifier, challenge string) {
	verifier = strings.Repeat("v3rifier-._~", 6)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

type desktopLoginEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Reason  string          `json:"reason"`
	Data    json.RawMessage `json:"data"`
}

func decodeDesktopLoginEnvelope(t *testing.T, w *httptest.ResponseRecorder) desktopLoginEnvelope {
	t.Helper()
	var env desktopLoginEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
	return env
}

func (f *desktopLoginHandlerFixture) issue(t *testing.T, body string) string {
	t.Helper()
	w := f.post("/api/v1/auth/desktop-session/code", body)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var data struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(decodeDesktopLoginEnvelope(t, w).Data, &data))
	return data.Code
}

func TestDesktopLoginHandler_CreateCodeRequiresAuthSubject(t *testing.T) {
	f := newDesktopLoginHandlerFixture(t, service.RoleUser, 0)
	_, challenge := desktopLoginHandlerPKCE()
	w := f.post("/api/v1/auth/desktop-session/code",
		`{"code_challenge":"`+challenge+`","code_challenge_method":"S256","api_key":"sk-general"}`)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, f.store.records)
}

func TestDesktopLoginHandler_CreateCodeShape(t *testing.T) {
	f := newDesktopLoginHandlerFixture(t, service.RoleUser, 41)
	_, challenge := desktopLoginHandlerPKCE()
	w := f.post("/api/v1/auth/desktop-session/code",
		`{"code_challenge":"`+challenge+`","code_challenge_method":"S256","api_key":"sk-general","claude_api_key":null,"codex_api_key":"sk-codex"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	env := decodeDesktopLoginEnvelope(t, w)
	require.Equal(t, 0, env.Code)
	var data map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &data))
	require.Len(t, data, 2)
	require.Equal(t, "K7QM-0001", data["code"])
	require.EqualValues(t, 600, data["expires_in"])

	record := f.store.records["K7QM-0001"]
	require.Equal(t, int64(41), record.UserID, "the code belongs to the signed-in user")
	require.Equal(t, challenge, record.CodeChallenge)
	require.Equal(t, "sk-general", record.APIKey)
	require.Empty(t, record.ClaudeAPIKey)
	require.Equal(t, "sk-codex", record.CodexAPIKey)
}

func TestDesktopLoginHandler_CreateCodeRejections(t *testing.T) {
	_, challenge := desktopLoginHandlerPKCE()
	for name, tc := range map[string]struct {
		body   string
		reason string
	}{
		"plain_method":   {body: `{"code_challenge":"` + challenge + `","code_challenge_method":"plain","api_key":"sk"}`, reason: "DESKTOP_LOGIN_CHALLENGE_INVALID"},
		"missing_method": {body: `{"code_challenge":"` + challenge + `","api_key":"sk"}`, reason: "DESKTOP_LOGIN_CHALLENGE_INVALID"},
		"short":          {body: `{"code_challenge":"abc","code_challenge_method":"S256","api_key":"sk"}`, reason: "DESKTOP_LOGIN_CHALLENGE_INVALID"},
		"missing_key":    {body: `{"code_challenge":"` + challenge + `","code_challenge_method":"S256"}`, reason: "DESKTOP_LOGIN_API_KEY_INVALID"},
		"blank_key":      {body: `{"code_challenge":"` + challenge + `","code_challenge_method":"S256","api_key":"  "}`, reason: "DESKTOP_LOGIN_API_KEY_INVALID"},
		"malformed_json": {body: `{"code_challenge":`, reason: ""},
	} {
		t.Run(name, func(t *testing.T) {
			f := newDesktopLoginHandlerFixture(t, service.RoleUser, 41)
			w := f.post("/api/v1/auth/desktop-session/code", tc.body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.Equal(t, tc.reason, decodeDesktopLoginEnvelope(t, w).Reason)
			require.Empty(t, f.store.records)
		})
	}
}

func TestDesktopLoginHandler_ExchangeShape(t *testing.T) {
	f := newDesktopLoginHandlerFixture(t, service.RoleUser, 41)
	verifier, challenge := desktopLoginHandlerPKCE()
	code := f.issue(t, `{"code_challenge":"`+challenge+`","code_challenge_method":"S256","api_key":"sk-general","claude_api_key":"sk-claude"}`)

	w := f.post("/api/v1/auth/desktop-session/exchange", `{"code":"`+code+`","code_verifier":"`+verifier+`"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	env := decodeDesktopLoginEnvelope(t, w)
	require.Equal(t, 0, env.Code)
	var data map[string]any
	require.NoError(t, json.Unmarshal(env.Data, &data))
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	require.ElementsMatch(t, []string{"access_token", "refresh_token", "expires_in", "token_type", "api_key", "claude_api_key"}, keys,
		"codex_api_key is omitted when the page left none")
	require.NotEmpty(t, data["access_token"])
	require.NotEmpty(t, data["refresh_token"])
	require.EqualValues(t, 3600, data["expires_in"])
	require.Equal(t, "Bearer", data["token_type"])
	require.Equal(t, "sk-general", data["api_key"])
	require.Equal(t, "sk-claude", data["claude_api_key"])

	// 一次性：同一个码再兑换是 DESKTOP_LOGIN_CODE_INVALID
	w = f.post("/api/v1/auth/desktop-session/exchange", `{"code":"`+code+`","code_verifier":"`+verifier+`"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "DESKTOP_LOGIN_CODE_INVALID", decodeDesktopLoginEnvelope(t, w).Reason)
}

// 码不存在、verifier 不符、verifier 格式错、缺字段、请求体不是 JSON：全是同一个 400。
func TestDesktopLoginHandler_ExchangeFailuresAreIndistinguishable(t *testing.T) {
	verifier, challenge := desktopLoginHandlerPKCE()
	for name, body := range map[string]func(code string) string{
		"unknown_code": func(string) string { return `{"code":"ZZZZ-ZZZZ","code_verifier":"` + verifier + `"}` },
		"wrong_verifier": func(code string) string {
			return `{"code":"` + code + `","code_verifier":"` + strings.Repeat("x", 64) + `"}`
		},
		"short_verifier": func(code string) string { return `{"code":"` + code + `","code_verifier":"abc"}` },
		"no_verifier":    func(code string) string { return `{"code":"` + code + `"}` },
		"no_code":        func(string) string { return `{"code_verifier":"` + verifier + `"}` },
		"malformed_json": func(string) string { return `{"code":` },
	} {
		t.Run(name, func(t *testing.T) {
			f := newDesktopLoginHandlerFixture(t, service.RoleUser, 41)
			code := f.issue(t, `{"code_challenge":"`+challenge+`","code_challenge_method":"S256","api_key":"sk-general"}`)

			w := f.post("/api/v1/auth/desktop-session/exchange", body(code))
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			env := decodeDesktopLoginEnvelope(t, w)
			require.Equal(t, "DESKTOP_LOGIN_CODE_INVALID", env.Reason)
			require.Equal(t, "login code is invalid or expired", env.Message)
			require.NotContains(t, w.Body.String(), "access_token")
		})
	}
}

func TestDesktopLoginHandler_ExchangeHonoursBackendMode(t *testing.T) {
	verifier, challenge := desktopLoginHandlerPKCE()
	for _, tc := range []struct {
		role   string
		status int
	}{
		{role: service.RoleUser, status: http.StatusForbidden},
		{role: service.RoleAdmin, status: http.StatusOK},
		{role: service.RoleOperator, status: http.StatusOK},
	} {
		t.Run(tc.role, func(t *testing.T) {
			f := newDesktopLoginHandlerFixture(t, tc.role, 41)
			f.h.backendMode = desktopLoginBackendModeStub(true)
			code := f.issue(t, `{"code_challenge":"`+challenge+`","code_challenge_method":"S256","api_key":"sk-general"}`)

			w := f.post("/api/v1/auth/desktop-session/exchange", `{"code":"`+code+`","code_verifier":"`+verifier+`"}`)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			if tc.status != http.StatusOK {
				require.NotContains(t, w.Body.String(), "access_token")
				require.NotContains(t, w.Body.String(), "sk-general")
			}
		})
	}
}
