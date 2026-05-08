package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSettingHandler_UpdateSettings_GitHubOAuth_OmittedKeepsPreviousValue(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{
		service.SettingKeyGitHubOAuthEnabled:      "true",
		service.SettingKeyGitHubOAuthClientID:     "github-client-id",
		service.SettingKeyGitHubOAuthClientSecret: "github-client-secret",
		service.SettingKeyGitHubOAuthRedirectURL:  "https://example.com/api/v1/auth/oauth/github/callback",
	}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	body := []byte(`{
		"site_name": "Sub2API",
		"default_concurrency": 1,
		"default_balance": 0
	}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			GitHubOAuthEnabled                bool   `json:"github_oauth_enabled"`
			GitHubOAuthClientID               string `json:"github_oauth_client_id"`
			GitHubOAuthClientSecretConfigured bool   `json:"github_oauth_client_secret_configured"`
			GitHubOAuthRedirectURL            string `json:"github_oauth_redirect_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.True(t, response.Data.GitHubOAuthEnabled)
	require.Equal(t, "github-client-id", response.Data.GitHubOAuthClientID)
	require.True(t, response.Data.GitHubOAuthClientSecretConfigured)
	require.Equal(t, "https://example.com/api/v1/auth/oauth/github/callback", response.Data.GitHubOAuthRedirectURL)

	require.Equal(t, "true", repo.values[service.SettingKeyGitHubOAuthEnabled])
	require.Equal(t, "github-client-id", repo.values[service.SettingKeyGitHubOAuthClientID])
	require.Equal(t, "github-client-secret", repo.values[service.SettingKeyGitHubOAuthClientSecret])
	require.Equal(t, "https://example.com/api/v1/auth/oauth/github/callback", repo.values[service.SettingKeyGitHubOAuthRedirectURL])
}
