//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSettingService_parseSettings_ClientChangelogGitHubRepo(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{
		Default: config.DefaultConfig{
			UserConcurrency: 10,
			UserBalance:     100.0,
		},
	})

	result := svc.parseSettings(map[string]string{
		SettingKeyClientChangelogGitHubRepo: "  ai-poet/agent-client  ",
	})
	require.Equal(t, "ai-poet/agent-client", result.ClientChangelogGitHubRepo)

	result = svc.parseSettings(map[string]string{})
	require.Empty(t, result.ClientChangelogGitHubRepo)
}

func TestSettingService_UpdateSettings_NormalizesClientChangelogGitHubRepo(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		ClientChangelogGitHubRepo: "https://github.com/ai-poet/agent-client.git",
	})
	require.NoError(t, err)
	require.Equal(t, "ai-poet/agent-client", repo.updates[SettingKeyClientChangelogGitHubRepo])
}

func TestSettingService_UpdateSettings_EmptyClientChangelogGitHubRepoStoredEmpty(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{})
	require.NoError(t, err)
	value, ok := repo.updates[SettingKeyClientChangelogGitHubRepo]
	require.True(t, ok)
	require.Empty(t, value)
}

func TestSettingService_UpdateSettings_InvalidClientChangelogGitHubRepoRejected(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		ClientChangelogGitHubRepo: "not a repo",
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_CLIENT_CHANGELOG_REPO", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}
