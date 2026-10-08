//go:build unit

package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// sessionHandoffMemoryStore 内存版一次性交接码存储：Consume 取出即删。
type sessionHandoffMemoryStore struct {
	mu      sync.Mutex
	next    int
	records map[string]service.SessionHandoffCode
	ttls    map[string]time.Duration
}

func newSessionHandoffMemoryStore() *sessionHandoffMemoryStore {
	return &sessionHandoffMemoryStore{
		records: map[string]service.SessionHandoffCode{},
		ttls:    map[string]time.Duration{},
	}
}

func (s *sessionHandoffMemoryStore) Store(_ context.Context, record *service.SessionHandoffCode, ttl time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	code := fmt.Sprintf("handoff-%d", s.next)
	s.records[code] = *record
	s.ttls[code] = ttl
	return code, nil
}

func (s *sessionHandoffMemoryStore) Consume(_ context.Context, code string) (*service.SessionHandoffCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[code]
	if !ok {
		return nil, service.ErrSessionHandoffCodeInvalid
	}
	delete(s.records, code)
	return &record, nil
}

func (s *sessionHandoffMemoryStore) put(code string, record service.SessionHandoffCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[code] = record
}

const (
	handoffLoginOrigin = "https://cheaprouter.cc"
	handoffAliasOrigin = "https://cdn.cheaprouter.cc"
)

func newSessionHandoffTestService(t *testing.T, user *service.User, handoff config.SessionHandoffConfig) (*service.SessionHandoffService, *sessionHandoffMemoryStore, *emailBindRefreshTokenCacheStub) {
	t.Helper()
	cache := newEmailBindRefreshTokenCacheStub()
	cfg := &config.Config{
		JWT:            config.JWTConfig{Secret: "session-handoff-secret", ExpireHour: 1, RefreshTokenExpireDays: 30},
		SessionHandoff: handoff,
	}
	authService := service.NewAuthService(nil, newEmailBindUserRepoStub(user), nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	store := newSessionHandoffMemoryStore()
	return service.NewSessionHandoffService(store, authService, cfg), store, cache
}

func enabledHandoffConfig() config.SessionHandoffConfig {
	return config.SessionHandoffConfig{LoginOrigin: handoffLoginOrigin, AliasOrigins: []string{handoffAliasOrigin}}
}

func TestSessionHandoff_CodeExchangesForAFreshUnboundFamily(t *testing.T) {
	svc, store, cache := newSessionHandoffTestService(t, desktopLoginTestUser(service.StatusActive), enabledHandoffConfig())
	verifier, challenge := desktopLoginPKCE("handoff-123")

	issued, err := svc.CreateCode(context.Background(), 41, challenge, "S256", "HTTPS://CDN.cheaprouter.cc:443/")
	require.NoError(t, err)
	require.Equal(t, handoffAliasOrigin, issued.TargetOrigin, "origin is normalized")
	require.Equal(t, 120, issued.ExpiresIn)
	require.Equal(t, service.SessionHandoffCodeTTL, store.ttls[issued.Code])

	// 签码时不签 token
	cache.mu.Lock()
	require.Empty(t, cache.tokens)
	cache.mu.Unlock()

	result, err := svc.Exchange(context.Background(), issued.Code, verifier, handoffAliasOrigin)
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	require.Equal(t, int64(41), result.UserID)
	require.Equal(t, service.RoleUser, result.UserRole)

	cache.mu.Lock()
	require.Len(t, cache.tokens, 1)
	for _, data := range cache.tokens {
		require.Empty(t, data.BindingHash, "handoff tokens start an unbound family")
	}
	cache.mu.Unlock()

	_, err = svc.Exchange(context.Background(), issued.Code, verifier, handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid, "a code works once")
}

func TestSessionHandoff_LoginOriginCanReceiveToo(t *testing.T) {
	svc, _, _ := newSessionHandoffTestService(t, desktopLoginTestUser(service.StatusActive), enabledHandoffConfig())
	verifier, challenge := desktopLoginPKCE("pull-back")

	issued, err := svc.CreateCode(context.Background(), 41, challenge, "S256", handoffLoginOrigin)
	require.NoError(t, err)
	result, err := svc.Exchange(context.Background(), issued.Code, verifier, "")
	require.NoError(t, err, "a request without an Origin header relies on PKCE alone")
	require.NotEmpty(t, result.AccessToken)
}

func TestSessionHandoff_DisabledWithoutConfig(t *testing.T) {
	svc, _, _ := newSessionHandoffTestService(t, desktopLoginTestUser(service.StatusActive), config.SessionHandoffConfig{})
	_, challenge := desktopLoginPKCE("disabled")

	require.False(t, svc.Enabled())
	_, err := svc.CreateCode(context.Background(), 41, challenge, "S256", handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffDisabled)
	_, err = svc.Exchange(context.Background(), "anything", "verifier", handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffDisabled)
}

func TestSessionHandoff_RejectsUnlistedOriginsAndBadChallenges(t *testing.T) {
	svc, store, _ := newSessionHandoffTestService(t, desktopLoginTestUser(service.StatusActive), enabledHandoffConfig())
	_, challenge := desktopLoginPKCE("reject")

	for _, origin := range []string{"https://evil.example", "https://cdn.cheaprouter.cc.evil.example", "http://cdn.cheaprouter.cc", "https://cdn.cheaprouter.cc/path", "javascript:alert(1)", ""} {
		_, err := svc.CreateCode(context.Background(), 41, challenge, "S256", origin)
		require.ErrorIsf(t, err, service.ErrSessionHandoffOriginNotAllowed, "origin %q", origin)
	}
	_, err := svc.CreateCode(context.Background(), 41, challenge, "plain", handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffChallengeInvalid)
	_, err = svc.CreateCode(context.Background(), 41, "short", "S256", handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffChallengeInvalid)
	require.Empty(t, store.records)
}

func TestSessionHandoff_WrongVerifierOrOriginBurnsTheCode(t *testing.T) {
	svc, _, _ := newSessionHandoffTestService(t, desktopLoginTestUser(service.StatusActive), enabledHandoffConfig())
	verifier, challenge := desktopLoginPKCE("burn")
	otherVerifier, _ := desktopLoginPKCE("other")

	issued, err := svc.CreateCode(context.Background(), 41, challenge, "S256", handoffAliasOrigin)
	require.NoError(t, err)
	_, err = svc.Exchange(context.Background(), issued.Code, otherVerifier, handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid)
	_, err = svc.Exchange(context.Background(), issued.Code, verifier, handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid, "the wrong verifier consumed the code")

	issued, err = svc.CreateCode(context.Background(), 41, challenge, "S256", handoffAliasOrigin)
	require.NoError(t, err)
	_, err = svc.Exchange(context.Background(), issued.Code, verifier, handoffLoginOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid, "exchanging on another origin than the code was issued for")
	_, err = svc.Exchange(context.Background(), issued.Code, verifier, handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid)
}

func TestSessionHandoff_StaleOrNoLongerAllowedRecordsAreRejected(t *testing.T) {
	svc, store, _ := newSessionHandoffTestService(t, desktopLoginTestUser(service.StatusActive), enabledHandoffConfig())
	verifier, challenge := desktopLoginPKCE("stale")

	store.put("stale", service.SessionHandoffCode{UserID: 41, CodeChallenge: challenge, TargetOrigin: handoffAliasOrigin, CreatedAt: time.Now().Add(-10 * time.Minute)})
	_, err := svc.Exchange(context.Background(), "stale", verifier, handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid)

	store.put("removed-alias", service.SessionHandoffCode{UserID: 41, CodeChallenge: challenge, TargetOrigin: "https://old.cheaprouter.cc", CreatedAt: time.Now()})
	_, err = svc.Exchange(context.Background(), "removed-alias", verifier, "https://old.cheaprouter.cc")
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid)
}

func TestSessionHandoff_InactiveUserCannotExchange(t *testing.T) {
	svc, _, _ := newSessionHandoffTestService(t, desktopLoginTestUser(service.StatusDisabled), enabledHandoffConfig())
	verifier, challenge := desktopLoginPKCE("inactive")

	issued, err := svc.CreateCode(context.Background(), 41, challenge, "S256", handoffAliasOrigin)
	require.NoError(t, err)
	_, err = svc.Exchange(context.Background(), issued.Code, verifier, handoffAliasOrigin)
	require.ErrorIs(t, err, service.ErrUserNotActive)
}
