//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fork：扣费失败重试 —— 告警评估器的 billing_failure_count / billing_unsettled_count 指标。

type billingFailureSourceStub struct {
	failures  int64
	unsettled int64
	err       error
	gotStart  time.Time
	gotEnd    time.Time
}

func (s *billingFailureSourceStub) CountBillingFailures(_ context.Context, start, end time.Time) (int64, error) {
	s.gotStart, s.gotEnd = start, end
	return s.failures, s.err
}

func (s *billingFailureSourceStub) UnsettledBillingCount(context.Context) (int64, error) {
	return s.unsettled, s.err
}

func TestComputeRuleMetricBillingFailureIndicators(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	start, end := now.Add(-5*time.Minute), now
	src := &billingFailureSourceStub{failures: 3, unsettled: 7}

	svc := &OpsAlertEvaluatorService{opsRepo: &stubOpsRepo{}}
	svc.SetBillingFailureSource(src)

	val, ok := svc.computeRuleMetric(context.Background(), &OpsAlertRule{MetricType: "billing_failure_count"}, nil, start, end, "", nil)
	require.True(t, ok)
	require.InDelta(t, 3.0, val, 1e-9)
	require.Equal(t, start, src.gotStart)
	require.Equal(t, end, src.gotEnd)

	val, ok = svc.computeRuleMetric(context.Background(), &OpsAlertRule{MetricType: "billing_unsettled_count"}, nil, start, end, "", nil)
	require.True(t, ok)
	require.InDelta(t, 7.0, val, 1e-9)

	// 数据源出错：指标不可用，规则跳过而不是误报 0
	src.err = errors.New("db down")
	_, ok = svc.computeRuleMetric(context.Background(), &OpsAlertRule{MetricType: "billing_failure_count"}, nil, start, end, "", nil)
	require.False(t, ok)

	// 未注入数据源：同样不可用
	bare := &OpsAlertEvaluatorService{opsRepo: &stubOpsRepo{}}
	_, ok = bare.computeRuleMetric(context.Background(), &OpsAlertRule{MetricType: "billing_unsettled_count"}, nil, start, end, "", nil)
	require.False(t, ok)
}
