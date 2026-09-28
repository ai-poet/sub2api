//go:build unit

package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// desktopLoginMemoryStore 内存版一次性登录码存储：Consume 取出即删。
type desktopLoginMemoryStore struct {
	mu      sync.Mutex
	next    int
	records map[string]service.DesktopLoginCode
	ttls    map[string]time.Duration
}

func newDesktopLoginMemoryStore() *desktopLoginMemoryStore {
	return &desktopLoginMemoryStore{
		records: map[string]service.DesktopLoginCode{},
		ttls:    map[string]time.Duration{},
	}
}

func (s *desktopLoginMemoryStore) Store(_ context.Context, record *service.DesktopLoginCode, ttl time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	code := fmt.Sprintf("TEST-%04d", s.next)
	s.records[code] = *record
	s.ttls[code] = ttl
	return code, nil
}

func (s *desktopLoginMemoryStore) Consume(_ context.Context, code string) (*service.DesktopLoginCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[code]
	if !ok {
		return nil, service.ErrDesktopLoginCodeInvalid
	}
	delete(s.records, code)
	return &record, nil
}

func (s *desktopLoginMemoryStore) only(t *testing.T) (string, service.DesktopLoginCode) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.records, 1)
	for code, record := range s.records {
		return code, record
	}
	return "", service.DesktopLoginCode{}
}

// desktopLoginPKCE 生成一对合法的 verifier / S256 challenge。
func desktopLoginPKCE(seed string) (verifier, challenge string) {
	verifier = strings.Repeat(seed, 64)[:64]
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func newDesktopLoginTestService(t *testing.T, user *service.User) (*service.DesktopLoginService, *desktopLoginMemoryStore, *emailBindRefreshTokenCacheStub) {
	t.Helper()
	cache := newEmailBindRefreshTokenCacheStub()
	cfg := &config.Config{
		JWT: config.JWTConfig{Secret: "desktop-login-secret", ExpireHour: 1, RefreshTokenExpireDays: 30},
	}
	authService := service.NewAuthService(nil, newEmailBindUserRepoStub(user), nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	store := newDesktopLoginMemoryStore()
	return service.NewDesktopLoginService(store, authService), store, cache
}

func desktopLoginTestUser(status string) *service.User {
	return &service.User{
		ID:           41,
		Email:        "desk-code@example.com",
		Username:     "desk-code",
		Role:         service.RoleUser,
		Status:       status,
		TokenVersion: 1,
	}
}

func TestDesktopLogin_CodeExchangesForFreshTokensAndKeys(t *testing.T) {
	svc, store, cache := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusActive))
	verifier, challenge := desktopLoginPKCE("abcDEF123-._~")

	issued, err := svc.CreateCode(context.Background(), 41, challenge, "S256", service.DesktopLoginKeys{
		APIKey:       "  sk-general  ",
		ClaudeAPIKey: "sk-claude",
	})
	require.NoError(t, err)
	require.Equal(t, 600, issued.ExpiresIn)

	code, record := store.only(t)
	require.Equal(t, code, issued.Code)
	require.Equal(t, service.DesktopLoginCodeTTL, store.ttls[code])
	require.Equal(t, int64(41), record.UserID)
	require.Equal(t, challenge, record.CodeChallenge)
	require.Equal(t, "sk-general", record.APIKey, "keys are trimmed")
	require.Equal(t, "sk-claude", record.ClaudeAPIKey)
	require.Empty(t, record.CodexAPIKey)
	require.False(t, record.CreatedAt.IsZero())

	// 签码时不签 token：token 只在兑换时生成
	cache.mu.Lock()
	require.Empty(t, cache.tokens)
	cache.mu.Unlock()

	result, err := svc.Exchange(context.Background(), issued.Code, verifier)
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	require.Positive(t, result.ExpiresIn)
	require.Equal(t, int64(41), result.UserID)
	require.Equal(t, service.RoleUser, result.UserRole)
	require.Equal(t, service.DesktopLoginKeys{APIKey: "sk-general", ClaudeAPIKey: "sk-claude"}, result.Keys)

	// 桌面 token 自成一个不绑定指纹的会话家族
	cache.mu.Lock()
	require.Len(t, cache.tokens, 1)
	for _, data := range cache.tokens {
		require.Equal(t, int64(41), data.UserID)
		require.Empty(t, data.BindingHash)
	}
	cache.mu.Unlock()

	// 一次性：再兑换一次失败
	_, err = svc.Exchange(context.Background(), issued.Code, verifier)
	require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)
}

// verifier 错了码也作废：拿到码的人没有第二次机会，正确的 verifier 随后也换不出 token。
func TestDesktopLogin_WrongVerifierBurnsTheCode(t *testing.T) {
	for name, badVerifier := range map[string]string{
		"mismatch":  strings.Repeat("z", 64),
		"too_short": "short",
		"too_long":  strings.Repeat("a", 129),
		"bad_chars": strings.Repeat("a", 60) + "!+/=",
		"empty":     "",
	} {
		t.Run(name, func(t *testing.T) {
			svc, store, cache := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusActive))
			verifier, challenge := desktopLoginPKCE("pkce")
			issued, err := svc.CreateCode(context.Background(), 41, challenge, "S256", service.DesktopLoginKeys{APIKey: "sk-general"})
			require.NoError(t, err)

			_, err = svc.Exchange(context.Background(), issued.Code, badVerifier)
			require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)
			require.Equal(t, "DESKTOP_LOGIN_CODE_INVALID", infraerrors.Reason(err))
			require.Equal(t, 400, infraerrors.Code(err))

			store.mu.Lock()
			require.Empty(t, store.records, "a failed verifier consumes the code")
			store.mu.Unlock()

			_, err = svc.Exchange(context.Background(), issued.Code, verifier)
			require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)

			cache.mu.Lock()
			require.Empty(t, cache.tokens, "no token is minted on any failed attempt")
			cache.mu.Unlock()
		})
	}
}

func TestDesktopLogin_UnknownCodeIsTheSameError(t *testing.T) {
	svc, _, _ := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusActive))
	verifier, _ := desktopLoginPKCE("pkce")

	for _, code := range []string{"", "NOPE-NOPE", "TEST-0001"} {
		_, err := svc.Exchange(context.Background(), code, verifier)
		require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid, "code=%q", code)
	}
}

func TestDesktopLogin_RejectsBadChallenge(t *testing.T) {
	_, good := desktopLoginPKCE("pkce")
	for name, tc := range map[string]struct {
		challenge string
		method    string
	}{
		"plain_method":   {challenge: good, method: "plain"},
		"empty_method":   {challenge: good, method: ""},
		"lowercase_s256": {challenge: good, method: "s256"},
		"empty":          {challenge: "", method: "S256"},
		"too_short":      {challenge: good[:42], method: "S256"},
		"too_long":       {challenge: good + "A", method: "S256"},
		"padded":         {challenge: good[:42] + "=", method: "S256"},
		"std_alphabet":   {challenge: good[:41] + "+/", method: "S256"},
	} {
		t.Run(name, func(t *testing.T) {
			svc, store, _ := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusActive))
			_, err := svc.CreateCode(context.Background(), 41, tc.challenge, tc.method, service.DesktopLoginKeys{APIKey: "sk-general"})
			require.ErrorIs(t, err, service.ErrDesktopLoginChallengeInvalid)
			require.Equal(t, "DESKTOP_LOGIN_CHALLENGE_INVALID", infraerrors.Reason(err))
			require.Equal(t, 400, infraerrors.Code(err))
			store.mu.Lock()
			require.Empty(t, store.records)
			store.mu.Unlock()
		})
	}
}

func TestDesktopLogin_RejectsMissingOrOversizedAPIKey(t *testing.T) {
	_, challenge := desktopLoginPKCE("pkce")
	for name, keys := range map[string]service.DesktopLoginKeys{
		"empty":            {},
		"blank":            {APIKey: "   ", ClaudeAPIKey: "sk-claude"},
		"oversized":        {APIKey: strings.Repeat("k", 513)},
		"oversized_codex":  {APIKey: "sk-general", CodexAPIKey: strings.Repeat("k", 513)},
		"oversized_claude": {APIKey: "sk-general", ClaudeAPIKey: strings.Repeat("k", 513)},
	} {
		t.Run(name, func(t *testing.T) {
			svc, store, _ := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusActive))
			_, err := svc.CreateCode(context.Background(), 41, challenge, "S256", keys)
			require.ErrorIs(t, err, service.ErrDesktopLoginAPIKeyInvalid)
			require.Equal(t, "DESKTOP_LOGIN_API_KEY_INVALID", infraerrors.Reason(err))
			require.Equal(t, 400, infraerrors.Code(err))
			store.mu.Lock()
			require.Empty(t, store.records)
			store.mu.Unlock()
		})
	}
}

func TestDesktopLogin_InactiveUserCannotExchange(t *testing.T) {
	svc, _, cache := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusDisabled))
	verifier, challenge := desktopLoginPKCE("pkce")
	issued, err := svc.CreateCode(context.Background(), 41, challenge, "S256", service.DesktopLoginKeys{APIKey: "sk-general"})
	require.NoError(t, err)

	_, err = svc.Exchange(context.Background(), issued.Code, verifier)
	require.ErrorIs(t, err, service.ErrUserNotActive)
	cache.mu.Lock()
	require.Empty(t, cache.tokens)
	cache.mu.Unlock()
}

// 签码后账号被删：对兑换方而言只是一个无效码，不暴露账号状态。
func TestDesktopLogin_DeletedUserIsAnInvalidCode(t *testing.T) {
	svc, _, _ := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusActive))
	verifier, challenge := desktopLoginPKCE("pkce")
	issued, err := svc.CreateCode(context.Background(), 404, challenge, "S256", service.DesktopLoginKeys{APIKey: "sk-general"})
	require.NoError(t, err)

	_, err = svc.Exchange(context.Background(), issued.Code, verifier)
	require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)
}

// Redis TTL 是主判据；TTL 丢失时，记录自带的签发时间仍然让过期码失效。
func TestDesktopLogin_StaleRecordIsRejectedEvenIfStoreKeptIt(t *testing.T) {
	svc, store, _ := newDesktopLoginTestService(t, desktopLoginTestUser(service.StatusActive))
	verifier, challenge := desktopLoginPKCE("pkce")
	store.mu.Lock()
	store.records["STALE-0001"] = service.DesktopLoginCode{
		UserID:        41,
		CodeChallenge: challenge,
		APIKey:        "sk-general",
		CreatedAt:     time.Now().Add(-service.DesktopLoginCodeTTL - 2*time.Minute),
	}
	store.mu.Unlock()

	_, err := svc.Exchange(context.Background(), "STALE-0001", verifier)
	require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)
}
