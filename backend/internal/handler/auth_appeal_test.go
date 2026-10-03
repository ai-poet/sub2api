//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// appealStoreMem 内存版申诉会话存储。
type appealStoreMem struct {
	mu      sync.Mutex
	records map[string]*service.AppealSessionRecord
}

func newAppealStoreMem() *appealStoreMem {
	return &appealStoreMem{records: map[string]*service.AppealSessionRecord{}}
}

func (s *appealStoreMem) Save(_ context.Context, hash string, record *service.AppealSessionRecord, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.records {
		if v.UserID == record.UserID {
			delete(s.records, k)
		}
	}
	clone := *record
	s.records[hash] = &clone
	return nil
}

func (s *appealStoreMem) Load(_ context.Context, hash string) (*service.AppealSessionRecord, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.records[hash]; ok {
		clone := *r
		return &clone, time.Hour, nil
	}
	return nil, 0, nil
}

func (s *appealStoreMem) Revoke(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, hash)
	return nil
}

func (s *appealStoreMem) RevokeUser(_ context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.records {
		if v.UserID == userID {
			delete(s.records, k)
		}
	}
	return nil
}

func (s *appealStoreMem) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

func newAppealLoginHandler(t *testing.T, status string, withAppeal bool) (*AuthHandler, *appealStoreMem) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	repo := &userHandlerRepoStub{user: &service.User{ID: 41, Email: "banned@example.com", Role: service.RoleUser, Status: status}}
	authService := service.NewAuthService(nil, repo, nil, &userHandlerRefreshTokenCacheStub{}, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	hash, err := authService.HashPassword("correct-password")
	require.NoError(t, err)
	repo.user.PasswordHash = hash

	h := &AuthHandler{cfg: cfg, authService: authService}
	store := newAppealStoreMem()
	if withAppeal {
		h.SetAppealService(service.NewAppealService(store, repo, nil, nil, nil))
	}
	return h, store
}

func doAppealLogin(h *AuthHandler, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"email": "banned@example.com", "password": password})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	return w
}

type appealErrorBody struct {
	Code     int               `json:"code"`
	Reason   string            `json:"reason"`
	Metadata map[string]string `json:"metadata"`
}

func decodeAppealError(t *testing.T, w *httptest.ResponseRecorder) appealErrorBody {
	t.Helper()
	var body appealErrorBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestAppealLogin_DisabledWithCorrectPasswordGetsToken(t *testing.T) {
	h, store := newAppealLoginHandler(t, service.StatusDisabled, true)

	w := doAppealLogin(h, "correct-password")
	require.Equal(t, http.StatusForbidden, w.Code)
	body := decodeAppealError(t, w)
	require.Equal(t, "USER_NOT_ACTIVE", body.Reason)
	require.True(t, strings.HasPrefix(body.Metadata["appeal_token"], service.AppealTokenPrefix))
	require.Equal(t, "7200", body.Metadata["expires_in"])
	require.NotContains(t, w.Body.String(), "access_token")
	require.Equal(t, 1, store.count())
}

func TestAppealLogin_WrongPasswordRevealsNothing(t *testing.T) {
	h, store := newAppealLoginHandler(t, service.StatusDisabled, true)

	w := doAppealLogin(h, "wrong-password")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	body := decodeAppealError(t, w)
	require.Equal(t, "INVALID_CREDENTIALS", body.Reason)
	require.Empty(t, body.Metadata)
	require.Zero(t, store.count())
}

func TestAppealLogin_WithoutAppealServiceBehavesAsBefore(t *testing.T) {
	h, _ := newAppealLoginHandler(t, service.StatusDisabled, false)

	w := doAppealLogin(h, "correct-password")
	require.Equal(t, http.StatusForbidden, w.Code)
	body := decodeAppealError(t, w)
	require.Equal(t, "USER_NOT_ACTIVE", body.Reason)
	require.Empty(t, body.Metadata)
}

func TestAppealLogin_OtherInactiveStatusGetsNoToken(t *testing.T) {
	h, store := newAppealLoginHandler(t, "pending", true)

	w := doAppealLogin(h, "correct-password")
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Empty(t, decodeAppealError(t, w).Metadata)
	require.Zero(t, store.count())
}

func TestAppealOAuthRedirectCarriesOnlyAppealToken(t *testing.T) {
	h, _ := newAppealLoginHandler(t, service.StatusDisabled, true)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/linuxdo/callback", nil)

	handled := h.redirectAppealIfDisabled(c, "/auth/linuxdo/callback", &service.UserDisabledError{UserID: 41, Email: "banned@example.com"})
	require.True(t, handled)
	require.Equal(t, http.StatusFound, w.Code)
	location, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	fragment, err := url.ParseQuery(location.Fragment)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(fragment.Get("appeal_token"), service.AppealTokenPrefix))
	require.Empty(t, fragment.Get("access_token"))
	require.Empty(t, fragment.Get("refresh_token"))

	// 普通错误不受影响
	require.False(t, h.redirectAppealIfDisabled(c, "/auth/linuxdo/callback", service.ErrUserNotActive))
}

func TestAppealPasskeyRejectsInactiveUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &userHandlerRepoStub{user: &service.User{ID: 41, Email: "banned@example.com", Status: service.StatusDisabled}}
	h := &PasskeyHandler{}
	h.SetAppealService(service.NewAppealService(newAppealStoreMem(), repo, nil, nil, nil))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkey/login/finish", nil)
	require.True(t, h.rejectInactivePasskeyUser(c, repo.user))
	require.Equal(t, http.StatusForbidden, w.Code)
	require.True(t, strings.HasPrefix(decodeAppealError(t, w).Metadata["appeal_token"], service.AppealTokenPrefix))

	active := &service.User{ID: 42, Status: service.StatusActive}
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	require.False(t, h.rejectInactivePasskeyUser(c2, active))
}
