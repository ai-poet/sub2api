//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingPublicRepoStub struct {
	values map[string]string
}

func (s *settingPublicRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingPublicRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingPublicRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingPublicRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingPublicRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingPublicRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingPublicRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetPublicSettings_ExposesRegistrationEmailSuffixWhitelist(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","@invalid_domain",""]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar"}, settings.RegistrationEmailSuffixWhitelist)
}

func TestSettingService_GetPublicSettings_ExposesClientDownloadURLs(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyClientDownloadWindowsURL: " https://downloads.example.com/cheaprouter-win.exe ",
			SettingKeyClientDownloadMacOSURL:   "https://downloads.example.com/cheaprouter-mac.dmg",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "https://downloads.example.com/cheaprouter-win.exe", settings.ClientDownloadWindowsURL)
	require.Equal(t, "https://downloads.example.com/cheaprouter-mac.dmg", settings.ClientDownloadMacOSURL)
}

func TestSettingService_GetPublicSettingsForInjection_IncludesClientDownloadURLs(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyClientDownloadWindowsURL: "https://downloads.example.com/windows.exe",
			SettingKeyClientDownloadMacOSURL:   "https://downloads.example.com/macos.dmg",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	payload, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"registration_enabled": false,
		"email_verify_enabled": false,
		"registration_email_suffix_whitelist": [],
		"promo_code_enabled": true,
		"password_reset_enabled": false,
		"invitation_code_enabled": false,
		"totp_enabled": false,
		"turnstile_enabled": false,
		"site_name": "Sub2API",
		"site_subtitle": "Subscription to API Conversion Platform",
		"hide_ccs_import_button": false,
		"purchase_subscription_enabled": false,
		"purchase_subscription_open_mode": "iframe",
		"custom_menu_items": [],
		"custom_endpoints": [],
		"group_status_enabled": false,
		"linuxdo_oauth_enabled": false,
		"github_oauth_enabled": false,
		"referral_enabled": false,
		"backend_mode_enabled": false,
		"client_changelog_entries": [],
		"client_download_windows_url": "https://downloads.example.com/windows.exe",
		"client_download_macos_url": "https://downloads.example.com/macos.dmg"
	}`, string(encoded))
}

// ==================== Changelog Filtering Tests (UT-01) ====================

func TestSettingService_GetPublicSettings_ChangelogFiltering(t *testing.T) {
	tests := []struct {
		name     string
		rawJSON  string
		expected []ClientChangelogEntry
	}{
		{
			name:    "filters disabled entries",
			rawJSON: `[{"version":"1.0","title":"First","items":["a"],"enabled":true},{"version":"1.1","title":"Second","items":["b"],"enabled":false}]`,
			expected: []ClientChangelogEntry{
				{Version: "1.0", Title: "First", Items: []string{"a"}, Enabled: true},
			},
		},
		{
			name:    "filters blank items",
			rawJSON: `[{"version":"1.0","title":"First","items":["","  ","a","\n"],"enabled":true}]`,
			expected: []ClientChangelogEntry{
				{Version: "1.0", Title: "First", Items: []string{"a"}, Enabled: true},
			},
		},
		{
			name:     "empty array",
			rawJSON:  `[]`,
			expected: []ClientChangelogEntry{},
		},
		{
			name:     "empty string",
			rawJSON:  "",
			expected: []ClientChangelogEntry{},
		},
		{
			name:     "invalid json",
			rawJSON:  `invalid json`,
			expected: []ClientChangelogEntry{},
		},
		{
			name:    "preserves items arrays",
			rawJSON: `[{"version":"1.0","title":"A","items":[],"enabled":true},{"version":"1.1","title":"B","items":["x"],"enabled":true}]`,
			expected: []ClientChangelogEntry{
				{Version: "1.0", Title: "A", Items: []string{}, Enabled: true},
				{Version: "1.1", Title: "B", Items: []string{"x"}, Enabled: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &settingPublicRepoStub{
				values: map[string]string{
					SettingKeyClientChangelogEntries: tt.rawJSON,
				},
			}
			svc := NewSettingService(repo, &config.Config{})

			settings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.expected, settings.ClientChangelogEntries)
		})
	}
}

// ==================== Changelog Sorting Tests (UT-02) ====================

func TestSettingService_GetPublicSettings_ChangelogSorting(t *testing.T) {
	tests := []struct {
		name     string
		input    []ClientChangelogEntry
		expected []ClientChangelogEntry
	}{
		{
			name: "descending by date",
			input: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: []string{}, Enabled: true},
				{Version: "1.1", PublishedAt: "2026-03-01", Title: "B", Items: []string{}, Enabled: true},
				{Version: "1.2", PublishedAt: "2026-02-01", Title: "C", Items: []string{}, Enabled: true},
			},
			expected: []ClientChangelogEntry{
				{Version: "1.1", PublishedAt: "2026-03-01", Title: "B", Items: []string{}, Enabled: true},
				{Version: "1.2", PublishedAt: "2026-02-01", Title: "C", Items: []string{}, Enabled: true},
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: []string{}, Enabled: true},
			},
		},
		{
			name: "empty dates last",
			input: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: []string{}, Enabled: true},
				{Version: "1.1", PublishedAt: "", Title: "B", Items: []string{}, Enabled: true},
				{Version: "1.2", PublishedAt: "2026-02-01", Title: "C", Items: []string{}, Enabled: true},
			},
			expected: []ClientChangelogEntry{
				{Version: "1.2", PublishedAt: "2026-02-01", Title: "C", Items: []string{}, Enabled: true},
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: []string{}, Enabled: true},
				{Version: "1.1", PublishedAt: "", Title: "B", Items: []string{}, Enabled: true},
			},
		},
		{
			name: "stable order for empty dates",
			input: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "", Title: "A", Items: []string{}, Enabled: true},
				{Version: "1.1", PublishedAt: "", Title: "B", Items: []string{}, Enabled: true},
			},
			expected: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "", Title: "A", Items: []string{}, Enabled: true},
				{Version: "1.1", PublishedAt: "", Title: "B", Items: []string{}, Enabled: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON, err := json.Marshal(tt.input)
			require.NoError(t, err)

			repo := &settingPublicRepoStub{
				values: map[string]string{
					SettingKeyClientChangelogEntries: string(rawJSON),
				},
			}
			svc := NewSettingService(repo, &config.Config{})

			settings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.expected, settings.ClientChangelogEntries)
		})
	}
}

// ==================== HTML Injection Tests (UT-05) ====================

func TestSettingService_GetPublicSettingsForInjection_ChangelogFiltering(t *testing.T) {
	tests := []struct {
		name                   string
		rawJSON                string
		expectedVersions       []string
		expectedItemCount      int
		expectedFirstItemValue string
	}{
		{
			name:             "only enabled entries",
			rawJSON:          `[{"version":"1.0","title":"A","enabled":true},{"version":"1.1","title":"B","enabled":false}]`,
			expectedVersions: []string{"1.0"},
		},
		{
			name:             "empty array",
			rawJSON:          `[]`,
			expectedVersions: []string{},
		},
		{
			name:             "empty string",
			rawJSON:          "",
			expectedVersions: []string{},
		},
		{
			name:                   "filters blank items in injection",
			rawJSON:                `[{"version":"1.0","title":"A","items":["","  ","x"],"enabled":true}]`,
			expectedVersions:       []string{"1.0"},
			expectedItemCount:      1,
			expectedFirstItemValue: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &settingPublicRepoStub{
				values: map[string]string{
					SettingKeyClientChangelogEntries: tt.rawJSON,
				},
			}
			svc := NewSettingService(repo, &config.Config{})

			payload, err := svc.GetPublicSettingsForInjection(context.Background())
			require.NoError(t, err)

			// Use JSON marshal/unmarshal to extract the changelog entries from the anonymous struct
			encoded, err := json.Marshal(payload)
			require.NoError(t, err)

			var result struct {
				ClientChangelogEntries []ClientChangelogEntry `json:"client_changelog_entries"`
			}
			err = json.Unmarshal(encoded, &result)
			require.NoError(t, err)

			versions := make([]string, len(result.ClientChangelogEntries))
			for i, e := range result.ClientChangelogEntries {
				versions[i] = e.Version
			}
			require.Equal(t, tt.expectedVersions, versions)

			if tt.expectedItemCount > 0 {
				require.Len(t, result.ClientChangelogEntries[0].Items, tt.expectedItemCount)
				require.Equal(t, tt.expectedFirstItemValue, result.ClientChangelogEntries[0].Items[0])
			}
		})
	}
}
