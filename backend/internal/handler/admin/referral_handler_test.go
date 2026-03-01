package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestReferralHandler_UpdateSettings_RejectNegativeValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewReferralHandler(nil)
	router := gin.New()
	router.PUT("/api/v1/admin/referral/settings", handler.UpdateSettings)

	body := []byte(`{
		"enabled": true,
		"referrer_balance_reward": -1,
		"referrer_group_id": 0,
		"referrer_subscription_days": 0,
		"referee_balance_reward": 0,
		"referee_group_id": 0,
		"referee_subscription_days": 0,
		"max_per_user": 0
	}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/referral/settings", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Invalid request")
}
