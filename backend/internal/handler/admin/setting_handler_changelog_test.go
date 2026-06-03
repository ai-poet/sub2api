//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSettingHandler_UpdateSettings_Changelog_SavesAndReads(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	entries := []service.ClientChangelogEntry{
		{Version: "1.0", PublishedAt: "2026-01-01", Title: "Initial", Items: []string{"Feature X"}, Enabled: true},
		{Version: "0.9", PublishedAt: "2025-12-01", Title: "Beta", Items: []string{"Beta Feature"}, Enabled: false},
	}
	body, _ := json.Marshal(map[string]any{
		"client_changelog_entries": entries,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			ClientChangelogEntries []service.ClientChangelogEntry `json:"client_changelog_entries"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Len(t, response.Data.ClientChangelogEntries, 2)

	// Verify DB stores single-layer JSON (not double-encoded)
	stored := repo.values[service.SettingKeyClientChangelogEntries]
	require.True(t, json.Valid([]byte(stored)))
	require.False(t, len(stored) > 0 && stored[0] == '"', "stored value must not be double-encoded")

	var storedEntries []service.ClientChangelogEntry
	require.NoError(t, json.Unmarshal([]byte(stored), &storedEntries))
	require.Len(t, storedEntries, 2)
	require.Equal(t, "1.0", storedEntries[0].Version)
}

func TestSettingHandler_UpdateSettings_Changelog_RejectsTooManyEntries(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	entries := make([]service.ClientChangelogEntry, service.MaxChangelogEntries+1)
	for i := range entries {
		entries[i] = service.ClientChangelogEntry{
			Version: "1.0", PublishedAt: "2026-01-01", Title: "Entry", Items: []string{"item"}, Enabled: true,
		}
	}
	body, _ := json.Marshal(map[string]any{
		"client_changelog_entries": entries,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "INVALID_CHANGELOG_TOO_MANY_ENTRIES")
}

func TestSettingHandler_UpdateSettings_Changelog_RejectsVersionTooLong(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	entries := []service.ClientChangelogEntry{
		{Version: strings.Repeat("v", service.MaxChangelogVersion+1), PublishedAt: "2026-01-01", Title: "A", Items: []string{"item"}, Enabled: true},
	}
	body, _ := json.Marshal(map[string]any{
		"client_changelog_entries": entries,
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "INVALID_CHANGELOG_VERSION_TOO_LONG")
}
