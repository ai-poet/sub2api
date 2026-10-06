package service

import (
	"context"
	"time"
)

// 扣费失败重试（fork 本地功能）。
//
// 后扣费事务（UsageBillingRepository.Apply）失败时，网关不再把 usage_log 的
// actual_cost 清零静默放过，而是保留应扣金额、把扣费命令入队，由
// UsageBillingRetryService 按退避重放。request_id + api_key_id 与
// usage_billing_dedup 同键，所以重放经去重表天然幂等：原事务其实已提交的情况
// 会被判为 Applied=false 而直接结清，不会重复扣款。

const (
	UsageBillingRetryStatusPending = "pending"
	UsageBillingRetryStatusSettled = "settled"
	UsageBillingRetryStatusFailed  = "failed"
)

// UsageBillingRetry 是一笔待重放 / 已结清 / 已放弃的扣费命令。
type UsageBillingRetry struct {
	ID        int64
	RequestID string
	APIKeyID  int64
	UserID    int64
	AccountID int64
	GroupID   *int64
	Platform  string
	// ActualCost 应扣金额快照（倍率后），只用于展示与告警汇总；重放的是 Command。
	ActualCost  float64
	Command     *UsageBillingCommand
	Status      string
	Attempts    int
	LastError   string
	NextRetryAt time.Time
	SettledAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UsageBillingRetryKey 是 usage_logs 与重试队列共享的幂等键。
type UsageBillingRetryKey struct {
	RequestID string
	APIKeyID  int64
}

// UsageBillingRetrySummary 当前未结清的笔数与金额。
type UsageBillingRetrySummary struct {
	Pending     int64
	Failed      int64
	PendingCost float64
	FailedCost  float64
}

// UsageBillingRetryRepository 重试队列持久化。
type UsageBillingRetryRepository interface {
	// Enqueue 入队；同键已存在时返回 false（不覆盖）。
	Enqueue(ctx context.Context, item *UsageBillingRetry) (bool, error)
	// ClaimDue 领取到期的 pending 行：attempts+1、next_retry_at 推到租约之后，
	// 多实例下用 SKIP LOCKED 互斥。
	ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]*UsageBillingRetry, error)
	MarkSettled(ctx context.Context, id int64, settledAt time.Time) error
	MarkRetry(ctx context.Context, id int64, lastError string, nextRetryAt time.Time) error
	MarkFailed(ctx context.Context, id int64, lastError string) error
	// ListByKeys 按 (request_id, api_key_id) 批量取行，供使用记录页标注。
	ListByKeys(ctx context.Context, keys []UsageBillingRetryKey) ([]*UsageBillingRetry, error)
	// CountCreatedBetween 统计 [start, end) 内入队的失败笔数（告警指标）。
	CountCreatedBetween(ctx context.Context, start, end time.Time) (int64, error)
	Summary(ctx context.Context) (*UsageBillingRetrySummary, error)
}

// BillingRetryEnqueuer 是网关侧的入队钩子（由 UsageBillingRetryService 实现）。
type BillingRetryEnqueuer interface {
	EnqueueFailedBilling(ctx context.Context, requestID string, usageLog *UsageLog, p *postUsageBillingParams, cause error)
}

// BillingFailureSource 供运维告警评估器读取扣费失败指标。
type BillingFailureSource interface {
	CountBillingFailures(ctx context.Context, start, end time.Time) (int64, error)
	UnsettledBillingCount(ctx context.Context) (int64, error)
}

// usageBillingCommandHasEffect 命令是否有任何需要落库的记账效果；没有就不值得重放。
func usageBillingCommandHasEffect(cmd *UsageBillingCommand) bool {
	if cmd == nil {
		return false
	}
	return cmd.BalanceCost > 0 || cmd.SubscriptionCost > 0 || cmd.APIKeyQuotaCost > 0 ||
		cmd.APIKeyRateLimitCost > 0 || cmd.AccountQuotaCost > 0
}
