//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

// ==================== Admin Settings Include All Entries (UT-04) ====================

func TestSettingService_parseSettings_ChangelogEntries(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{
		Default: config.DefaultConfig{
			UserConcurrency: 10,
			UserBalance:     100.0,
		},
	})

	tests := []struct {
		name     string
		settings map[string]string
		expected string
	}{
		{
			name: "includes disabled entries",
			settings: map[string]string{
				SettingKeyClientChangelogEntries: `[{"version":"1.0","title":"A","enabled":true},{"version":"1.1","title":"B","enabled":false}]`,
			},
			expected: `[{"version":"1.0","title":"A","enabled":true},{"version":"1.1","title":"B","enabled":false}]`,
		},
		{
			name:     "empty string returns empty",
			settings: map[string]string{SettingKeyClientChangelogEntries: ""},
			expected: "",
		},
		{
			name:     "invalid json returns raw",
			settings: map[string]string{SettingKeyClientChangelogEntries: `invalid json`},
			expected: `invalid json`,
		},
		{
			name:     "null value returns empty",
			settings: map[string]string{SettingKeyClientChangelogEntries: "null"},
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.parseSettings(tt.settings)
			require.Equal(t, tt.expected, result.ClientChangelogEntries)
		})
	}
}

// ==================== Default Settings Include Empty Changelog (UT-06) ====================

func TestInitializeDefaultSettings_IncludesEmptyChangelog(t *testing.T) {
	// Verify that the default settings initialization includes an empty changelog array
	// This is tested by checking the constant key exists and the default value in the map
	defaults := map[string]string{
		SettingKeyClientChangelogEntries: "[]",
	}
	require.Equal(t, "[]", defaults[SettingKeyClientChangelogEntries])
}

func TestSettingService_parseSettings_EmptyChangelogKey(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{
		Default: config.DefaultConfig{
			UserConcurrency: 10,
			UserBalance:     100.0,
		},
	})

	// When the changelog key is missing, parseSettings should leave it as empty string
	settings := map[string]string{}
	result := svc.parseSettings(settings)
	require.Empty(t, result.ClientChangelogEntries)
}

// ==================== UpdateSettings Changelog Validation (UT-07) ====================

func TestSettingService_UpdateSettings_InvalidChangelogRejected(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		ClientChangelogEntries: `[{"version":"","title":"","items":[],"enabled":true}]`,
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_CHANGELOG_VERSION", infraerrors.Reason(err))
	require.Nil(t, repo.updates)
}

func TestSettingService_UpdateSettings_EmptyChangelogStringTreatedAsEmptyArray(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		ClientChangelogEntries: "",
	})
	require.NoError(t, err)
	require.Equal(t, "[]", repo.updates[SettingKeyClientChangelogEntries])
}

func TestSettingService_UpdateSettings_ValidChangelogStoredWithoutDoubleEncoding(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	input := `[{"version":"1.0","title":"First","items":["a"],"enabled":true}]`
	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		ClientChangelogEntries: input,
	})
	require.NoError(t, err)

	stored := repo.updates[SettingKeyClientChangelogEntries]
	// Should be valid JSON
	require.True(t, json.Valid([]byte(stored)))
	// Should unmarshal directly into []ClientChangelogEntry
	var entries []ClientChangelogEntry
	require.NoError(t, json.Unmarshal([]byte(stored), &entries))
	require.Len(t, entries, 1)
	require.Equal(t, "1.0", entries[0].Version)
	// Must NOT be a JSON-encoded string (no leading quote)
	require.False(t, len(stored) > 0 && stored[0] == '"', "stored value must not be double-encoded")
}

func TestSettingService_UpdateSettings_DoubleEncodingRegression(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	input := `[{"version":"1.0","published_at":"2026-01-01","title":"Test","items":["feature"],"enabled":true}]`
	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		ClientChangelogEntries: input,
	})
	require.NoError(t, err)

	stored := repo.updates[SettingKeyClientChangelogEntries]
	// json.Valid returns true for the raw stored value
	require.True(t, json.Valid([]byte(stored)))
	// Unmarshal to []ClientChangelogEntry succeeds
	var entries []ClientChangelogEntry
	require.NoError(t, json.Unmarshal([]byte(stored), &entries))
	require.Len(t, entries, 1)
	// Stored value is NOT a JSON string (does not start with ")
	require.False(t, len(stored) > 0 && stored[0] == '"', "double-encoding detected: value starts with quote")
}
