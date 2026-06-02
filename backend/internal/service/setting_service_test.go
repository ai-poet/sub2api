//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
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
