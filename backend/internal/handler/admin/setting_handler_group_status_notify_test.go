package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type groupStatusNotifySettingsResponse struct {
	Code int `json:"code"`
	Data struct {
		Enabled           bool   `json:"group_status_notify_serverchan_enabled"`
		UID               string `json:"group_status_notify_serverchan_uid"`
		SendKeyConfigured bool   `json:"group_status_notify_serverchan_sendkey_configured"`
	} `json:"data"`
}

func putSettingsForGroupStatusNotifyTest(t *testing.T, router *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader([]byte(body)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestSettingHandler_UpdateSettings_GroupStatusNotify_RoundTrip(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	recorder := putSettingsForGroupStatusNotifyTest(t, router, `{
		"group_status_notify_serverchan_enabled": true,
		"group_status_notify_serverchan_uid": " 12345 ",
		"group_status_notify_serverchan_sendkey": "sctp_secret_key"
	}`)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response groupStatusNotifySettingsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.True(t, response.Data.Enabled)
	require.Equal(t, "12345", response.Data.UID)
	require.True(t, response.Data.SendKeyConfigured)
	// 密钥绝不能回显
	require.NotContains(t, recorder.Body.String(), "sctp_secret_key")

	require.Equal(t, "true", repo.values[service.SettingKeyGroupStatusNotifyServerChanEnabled])
	require.Equal(t, "12345", repo.values[service.SettingKeyGroupStatusNotifyServerChanUID])
	require.Equal(t, "sctp_secret_key", repo.values[service.SettingKeyGroupStatusNotifyServerChanSendKey])
}

func TestSettingHandler_UpdateSettings_GroupStatusNotify_OmittedFieldsKeepPrevious(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{
		service.SettingKeyGroupStatusNotifyServerChanEnabled: "true",
		service.SettingKeyGroupStatusNotifyServerChanUID:     "12345",
		service.SettingKeyGroupStatusNotifyServerChanSendKey: "sctp_previous",
	}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	recorder := putSettingsForGroupStatusNotifyTest(t, router, `{"site_name": "Sub2API"}`)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response groupStatusNotifySettingsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Data.Enabled)
	require.Equal(t, "12345", response.Data.UID)
	require.True(t, response.Data.SendKeyConfigured)
	require.NotContains(t, recorder.Body.String(), "sctp_previous")

	require.Equal(t, "sctp_previous", repo.values[service.SettingKeyGroupStatusNotifyServerChanSendKey])
}

func TestSettingHandler_UpdateSettings_GroupStatusNotify_EmptySendKeyKeepsPrevious(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{
		service.SettingKeyGroupStatusNotifyServerChanSendKey: "sctp_previous",
	}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	recorder := putSettingsForGroupStatusNotifyTest(t, router, `{
		"group_status_notify_serverchan_enabled": false,
		"group_status_notify_serverchan_uid": "12345",
		"group_status_notify_serverchan_sendkey": ""
	}`)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response groupStatusNotifySettingsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Data.Enabled)
	require.True(t, response.Data.SendKeyConfigured)
	require.Equal(t, "sctp_previous", repo.values[service.SettingKeyGroupStatusNotifyServerChanSendKey])
}

func TestSettingHandler_UpdateSettings_GroupStatusNotify_RejectsInvalidUID(t *testing.T) {
	repo := &settingRepoForPurchaseOpenModeTest{values: map[string]string{}}
	router := setupSettingHandlerForPurchaseOpenModeTest(repo)

	recorder := putSettingsForGroupStatusNotifyTest(t, router, `{"group_status_notify_serverchan_uid": "evil.com"}`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	_, saved := repo.values[service.SettingKeyGroupStatusNotifyServerChanUID]
	require.False(t, saved)
}

// ---------- 测试推送接口 ----------

type fakeGroupStatusNotifyTester struct {
	uid     string
	sendkey string
	calls   int
	err     error
}

func (f *fakeGroupStatusNotifyTester) SendTest(_ context.Context, uid, sendkey string) error {
	f.calls++
	f.uid = uid
	f.sendkey = sendkey
	return f.err
}

func setupGroupStatusNotifyTestRouter(tester groupStatusNotifyTester) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	settingService := service.NewSettingService(&settingRepoForPurchaseOpenModeTest{values: map[string]string{}}, &config.Config{})
	handler := NewSettingHandler(settingService, nil, nil, nil, nil)
	if tester != nil {
		handler.SetGroupStatusNotifyService(tester)
	}
	router.POST("/api/v1/admin/settings/group-status-notify/test", handler.TestGroupStatusNotify)
	return router
}

func postGroupStatusNotifyTest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/settings/group-status-notify/test", bytes.NewReader([]byte(body)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestSettingHandler_TestGroupStatusNotify_PassesThroughAndSucceeds(t *testing.T) {
	tester := &fakeGroupStatusNotifyTester{}
	router := setupGroupStatusNotifyTestRouter(tester)

	recorder := postGroupStatusNotifyTest(router, `{"uid": " 12345 ", "sendkey": " sctp_x "}`)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, tester.calls)
	require.Equal(t, "12345", tester.uid)
	require.Equal(t, "sctp_x", tester.sendkey)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Equal(t, "Test push sent", response.Data.Message)
}

func TestSettingHandler_TestGroupStatusNotify_EmptyBodyUsesSavedConfig(t *testing.T) {
	tester := &fakeGroupStatusNotifyTester{}
	router := setupGroupStatusNotifyTestRouter(tester)

	recorder := postGroupStatusNotifyTest(router, ``)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, tester.calls)
	require.Empty(t, tester.uid)
	require.Empty(t, tester.sendkey)
}

func TestSettingHandler_TestGroupStatusNotify_ReturnsBadRequestOnFailure(t *testing.T) {
	tester := &fakeGroupStatusNotifyTester{err: errors.New("serverchan returned bad key")}
	router := setupGroupStatusNotifyTestRouter(tester)

	recorder := postGroupStatusNotifyTest(router, `{"uid": "12345", "sendkey": "sctp_x"}`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "serverchan returned bad key")
}

func TestSettingHandler_TestGroupStatusNotify_ServiceUnavailableWhenNotWired(t *testing.T) {
	router := setupGroupStatusNotifyTestRouter(nil)

	recorder := postGroupStatusNotifyTest(router, `{}`)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
