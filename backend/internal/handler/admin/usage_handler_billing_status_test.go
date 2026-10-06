package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fork：扣费失败重试 —— 管理端使用记录按 (request_id, api_key_id) 标注扣费状态，
// 且 actual_cost 保留应扣金额（不再因为扣费失败被清零）。

type adminUsageRepoWithBillingFailure struct {
	service.UsageLogRepository
}

func (s *adminUsageRepoWithBillingFailure) ListWithFilters(_ context.Context, params pagination.PaginationParams, _ usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	return []service.UsageLog{
		{ID: 1, UserID: 4641, APIKeyID: 6483, RequestID: "client:abc", Model: "claude-opus-5-5", TotalCost: 0.324556, ActualCost: 0.0649112},
		{ID: 2, UserID: 4641, APIKeyID: 6484, RequestID: "client:def", Model: "claude-opus-5-5", TotalCost: 0.1, ActualCost: 0.02},
	}, &pagination.PaginationResult{Total: 2, Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

type billingAnnotatorStub struct{ calls int }

func (s *billingAnnotatorStub) Annotate(_ context.Context, logs []service.UsageLog) {
	s.calls++
	for i := range logs {
		if logs[i].RequestID == "client:abc" && logs[i].APIKeyID == 6483 {
			logs[i].BillingStatus = service.UsageBillingRetryStatusPending
			logs[i].BillingError = "api key not found"
			logs[i].BillingAttempts = 2
		}
	}
}

func TestAdminUsageListAnnotatesBillingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(&adminUsageRepoWithBillingFailure{}, nil, nil, nil)
	handler := NewUsageHandler(usageSvc, nil, nil, nil)
	annotator := &billingAnnotatorStub{}
	handler.SetBillingRetryAnnotator(annotator)
	router := gin.New()
	router.GET("/admin/usage", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, annotator.calls)

	var resp struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 2)

	failed := resp.Data.Items[0]
	require.Equal(t, "pending", failed["billing_status"])
	require.Equal(t, "api key not found", failed["billing_error"])
	require.InDelta(t, 2, failed["billing_attempts"], 1e-9)
	require.InDelta(t, 0.0649112, failed["actual_cost"], 1e-12, "应扣金额必须保留")

	settled := resp.Data.Items[1]
	_, hasStatus := settled["billing_status"]
	require.False(t, hasStatus, "即时扣费成功的记录不带 billing_status")
}
