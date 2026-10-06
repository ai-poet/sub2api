//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fork：扣费失败重试 —— 后扣费失败时日志保留应扣金额，并把命令交给入队钩子。

type billingRetryEnqueuerStub struct {
	calls     int
	requestID string
	usageLog  *UsageLog
	params    *postUsageBillingParams
	cause     error
}

func (s *billingRetryEnqueuerStub) EnqueueFailedBilling(_ context.Context, requestID string, usageLog *UsageLog, p *postUsageBillingParams, cause error) {
	s.calls++
	s.requestID = requestID
	s.usageLog = usageLog
	s.params = p
	s.cause = cause
}

func TestGatewayServiceRecordUsage_BillingFailureKeepsActualCostAndEnqueuesRetry(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{err: ErrAPIKeyNotFound}
	userRepo := &openAIRecordUsageUserRepoStub{}
	subRepo := &openAIRecordUsageSubRepoStub{}
	svc := newGatewayRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, userRepo, subRepo)
	enqueuer := &billingRetryEnqueuerStub{}
	svc.SetBillingRetryEnqueuer(enqueuer)

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gateway_deleted_key",
			Usage:     ClaudeUsage{InputTokens: 10, OutputTokens: 6},
			Model:     "claude-sonnet-4",
			Duration:  time.Second,
		},
		APIKey:        &APIKey{ID: 6483, Quota: 0.01},
		User:          &User{ID: 4641},
		Account:       &Account{ID: 11},
		APIKeyService: &openAIRecordUsageAPIKeyQuotaStub{},
	})

	require.ErrorIs(t, err, ErrAPIKeyNotFound)
	require.Equal(t, 1, billingRepo.calls)
	require.Equal(t, 1, usageRepo.calls, "扣费失败也要写用量日志")
	require.NotNil(t, usageRepo.lastLog)
	require.Greater(t, usageRepo.lastLog.TotalCost, 0.0)
	require.InDelta(t, usageRepo.lastLog.TotalCost*1.1, usageRepo.lastLog.ActualCost, 1e-9, "应扣金额不再被清零")

	require.Equal(t, 1, enqueuer.calls)
	require.Equal(t, usageRepo.lastLog.RequestID, enqueuer.requestID)
	require.Same(t, usageRepo.lastLog, enqueuer.usageLog)
	require.ErrorIs(t, enqueuer.cause, ErrAPIKeyNotFound)
	require.NotNil(t, enqueuer.params)
	require.InDelta(t, usageRepo.lastLog.ActualCost, enqueuer.params.Cost.ActualCost, 1e-12)
	require.Equal(t, 0, userRepo.deductCalls, "统一扣费仓储在位时不会走旧的直接扣款路径")
}

// 钩子未注入时行为与上游一致（只差日志不清零）。
func TestGatewayServiceRecordUsage_BillingFailureWithoutEnqueuerStillLogs(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{err: ErrAPIKeyNotFound}
	svc := newGatewayRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result:  &ForwardResult{RequestID: "gateway_no_hook", Usage: ClaudeUsage{InputTokens: 10, OutputTokens: 6}, Model: "claude-sonnet-4", Duration: time.Second},
		APIKey:  &APIKey{ID: 1},
		User:    &User{ID: 2},
		Account: &Account{ID: 3},
	})
	require.ErrorIs(t, err, ErrAPIKeyNotFound)
	require.Equal(t, 1, usageRepo.calls)
	require.Greater(t, usageRepo.lastLog.ActualCost, 0.0)
}
