package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var _ service.SettingRepository = (*stubSettingRepoForPublicSettings)(nil)

type stubSettingRepoForPublicSettings struct {
	values map[string]string
}

func (r *stubSettingRepoForPublicSettings) Get(_ context.Context, key string) (*service.Setting, error) {
	if value, ok := r.values[key]; ok {
		return &service.Setting{Key: key, Value: value}, nil
	}
	return nil, service.ErrSettingNotFound
}

func (r *stubSettingRepoForPublicSettings) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (r *stubSettingRepoForPublicSettings) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *stubSettingRepoForPublicSettings) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (r *stubSettingRepoForPublicSettings) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *stubSettingRepoForPublicSettings) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(r.values))
	for key, value := range r.values {
		result[key] = value
	}
	return result, nil
}

func (r *stubSettingRepoForPublicSettings) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestSettingHandler_GetPublicSettings_IncludesReferralEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settingRepo := &stubSettingRepoForPublicSettings{
		values: map[string]string{
			service.SettingKeyReferralEnabled: "true",
		},
	}

	settingService := service.NewSettingService(settingRepo, nil)
	settingHandler := NewSettingHandler(settingService, "v1.0.0")

	router := gin.New()
	router.GET("/api/v1/settings/public", settingHandler.GetPublicSettings)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			ReferralEnabled bool `json:"referral_enabled"`
		} `json:"data"`
	}

	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.True(t, response.Data.ReferralEnabled)
}
