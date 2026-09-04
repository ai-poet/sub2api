//go:build unit

package service_test

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// The desktop's pair must not share the browser's refresh token: refresh
// tokens rotate, so two holders of one token sign each other out. It must
// also carry no browser fingerprint, or the desktop's first renewal would
// revoke the family under session binding.
func TestGenerateDesktopTokenPair_StartsItsOwnUnboundFamily(t *testing.T) {
	user := &service.User{
		ID:           7,
		Email:        "desk@example.com",
		Username:     "desk",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		TokenVersion: 1,
	}
	userRepo := newEmailBindUserRepoStub(user)
	cache := newEmailBindRefreshTokenCacheStub()
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                 "desktop-session-secret",
			ExpireHour:             1,
			RefreshTokenExpireDays: 30,
		},
	}
	svc := service.NewAuthService(nil, userRepo, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil)

	// The browser signs in with the fingerprint the HTTP middleware attaches.
	browserCtx := service.WithSessionBinding(context.Background(), &service.SessionBinding{
		IP:        "203.0.113.5",
		UserAgent: "Mozilla/5.0",
	})
	browserPair, err := svc.GenerateTokenPair(browserCtx, user, "")
	require.NoError(t, err)

	desktopPair, err := svc.GenerateDesktopTokenPair(browserCtx, user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, desktopPair.AccessToken)
	require.NotEmpty(t, desktopPair.RefreshToken)
	require.Positive(t, desktopPair.ExpiresIn)
	require.NotEqual(t, browserPair.RefreshToken, desktopPair.RefreshToken)

	// Two stored refresh tokens: the browser's, bound; the desktop's, not —
	// and in different families.
	cache.mu.Lock()
	var families []string
	var bound, unbound int
	for _, data := range cache.tokens {
		families = append(families, data.FamilyID)
		if data.BindingHash == "" {
			unbound++
		} else {
			bound++
		}
	}
	cache.mu.Unlock()
	require.Len(t, families, 2)
	require.NotEqual(t, families[0], families[1])
	require.Equal(t, 1, bound)
	require.Equal(t, 1, unbound)

	// The desktop renews from its own network identity, and the browser's
	// token is untouched by that renewal.
	desktopCtx := service.WithSessionBinding(context.Background(), &service.SessionBinding{
		IP:        "198.51.100.9",
		UserAgent: "curl/8.0",
	})
	renewed, err := svc.RefreshTokenPair(desktopCtx, desktopPair.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, desktopPair.RefreshToken, renewed.RefreshToken)

	stillBrowser, err := svc.RefreshTokenPair(browserCtx, browserPair.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, stillBrowser.AccessToken)
}

func TestGenerateDesktopTokenPair_RefusesInactiveUsers(t *testing.T) {
	user := &service.User{
		ID:     8,
		Email:  "gone@example.com",
		Role:   service.RoleUser,
		Status: service.StatusDisabled,
	}
	cfg := &config.Config{
		JWT: config.JWTConfig{Secret: "desktop-session-secret", ExpireHour: 1, RefreshTokenExpireDays: 30},
	}
	svc := service.NewAuthService(nil, newEmailBindUserRepoStub(user), nil, newEmailBindRefreshTokenCacheStub(), cfg, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GenerateDesktopTokenPair(context.Background(), user.ID)
	require.ErrorIs(t, err, service.ErrUserNotActive)

	_, err = svc.GenerateDesktopTokenPair(context.Background(), 404)
	require.ErrorIs(t, err, service.ErrUserNotFound)
}
