package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type settingRepoForPurchaseOpenModeTest struct {
	values map[string]string
}

func (r *settingRepoForPurchaseOpenModeTest) Get(_ context.Context, key string) (*service.Setting, error) {
	if value, ok := r.values[key]; ok {
		return &service.Setting{Key: key, Value: value}, nil
	}
	return nil, service.ErrSettingNotFound
}

func (r *settingRepoForPurchaseOpenModeTest) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (r *settingRepoForPurchaseOpenModeTest) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *settingRepoForPurchaseOpenModeTest) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (r *settingRepoForPurchaseOpenModeTest) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *settingRepoForPurchaseOpenModeTest) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(r.values))
	for key, value := range r.values {
		result[key] = value
	}
	return result, nil
}

func (r *settingRepoForPurchaseOpenModeTest) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func setupSettingHandlerForPurchaseOpenModeTest(repo service.SettingRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	settingService := service.NewSettingService(repo, &config.Config{
		Default: config.DefaultConfig{
			UserConcurrency: 1,
			UserBalance:     0,
		},
	})
	handler := NewSettingHandler(settingService, nil, nil, nil)
	router.PUT("/api/v1/admin/settings", handler.UpdateSettings)
	return router
}

func TestSettingHandler_UpdateSettings_PurchaseOpenMode_UpdatesToNewWindow(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	body := []byte(`{
		"purchase_subscription_enabled": true,
		"purchase_subscription_url": "https://shop.cyberspirit.io",
		"purchase_subscription_open_mode": "new_window"
	}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			PurchaseSubscriptionOpenMode string `json:"purchase_subscription_open_mode"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Equal(t, "new_window", response.Data.PurchaseSubscriptionOpenMode)
	require.Equal(t, "new_window", repo.values[service.SettingKeyPurchaseSubscriptionOpenMode])
}

func TestSettingHandler_UpdateSettings_PurchaseOpenMode_OmittedKeepsPreviousValue(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{
		service.SettingKeyPurchaseSubscriptionEnabled:  "true",
		service.SettingKeyPurchaseSubscriptionURL:      "https://shop.cyberspirit.io",
		service.SettingKeyPurchaseSubscriptionOpenMode: "new_window",
	}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	body := []byte(`{
		"purchase_subscription_enabled": true,
		"purchase_subscription_url": "https://shop.cyberspirit.io"
	}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			PurchaseSubscriptionOpenMode string `json:"purchase_subscription_open_mode"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Equal(t, "new_window", response.Data.PurchaseSubscriptionOpenMode)
	require.Equal(t, "new_window", repo.values[service.SettingKeyPurchaseSubscriptionOpenMode])
}

func TestSettingHandler_UpdateSettings_PurchaseOpenMode_RejectsInvalidValue(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	body := []byte(`{
		"purchase_subscription_enabled": true,
		"purchase_subscription_url": "https://shop.cyberspirit.io",
		"purchase_subscription_open_mode": "popup"
	}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Purchase Subscription open mode")
}
