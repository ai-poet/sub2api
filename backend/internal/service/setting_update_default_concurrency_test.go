//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type defaultConcurrencySettingRepoStub struct {
	*settingUpdateRepoStub
	followerUpdates map[string]string
	followerCalls   int
}

func (s *defaultConcurrencySettingRepoStub) SetMultipleWithDefaultConcurrency(_ context.Context, updates map[string]string) error {
	s.followerCalls++
	s.followerUpdates = make(map[string]string, len(updates))
	for key, value := range updates {
		s.followerUpdates[key] = value
	}
	return s.setMultipleErr
}

func TestSettingService_PersistSettingsUpdatesUsesDefaultConcurrencyWriter(t *testing.T) {
	repo := &defaultConcurrencySettingRepoStub{settingUpdateRepoStub: &settingUpdateRepoStub{}}
	svc := NewSettingService(repo, &config.Config{})
	updates := map[string]string{SettingKeyDefaultConcurrency: "8", "site_name": "Sub2API"}

	require.NoError(t, svc.persistSettingsUpdates(context.Background(), updates))
	require.Equal(t, 1, repo.followerCalls)
	require.Equal(t, updates, repo.followerUpdates)
	require.Nil(t, repo.updates, "the plain SetMultiple fallback must not be used")
}

func TestSettingService_DefaultConcurrencyAutoFollowRoundTrip(t *testing.T) {
	svc := NewSettingService(&settingUpdateRepoStub{}, &config.Config{})
	parsed := svc.parseSettings(map[string]string{SettingKeyDefaultConcurrencyAutoFollow: "true"})
	require.True(t, parsed.DefaultConcurrencyAutoFollow)

	updates, err := svc.buildSystemSettingsUpdates(context.Background(), &SystemSettings{
		DefaultConcurrency:           5,
		DefaultConcurrencyAutoFollow: true,
	})
	require.NoError(t, err)
	require.Equal(t, "true", updates[SettingKeyDefaultConcurrencyAutoFollow])
}
