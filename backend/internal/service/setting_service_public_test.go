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
		"client_download_windows_url": "https://downloads.example.com/windows.exe",
		"client_download_macos_url": "https://downloads.example.com/macos.dmg"
	}`, string(encoded))
}
