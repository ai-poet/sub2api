//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fork：扣费失败重试 —— 入队、重放、标注、告警摘要。

type billingRetryRepoStub struct {
	mu         sync.Mutex
	enqueued   []*UsageBillingRetry
	enqueueErr error
	dupKeys    map[UsageBillingRetryKey]bool
	due        []*UsageBillingRetry
	claimCalls int
	settled    []int64
	retried    map[int64]time.Time
	retryErrs  map[int64]string
	failed     map[int64]string
	byKey      map[UsageBillingRetryKey]*UsageBillingRetry
	created    int64
	summary    UsageBillingRetrySummary
}

func newBillingRetryRepoStub() *billingRetryRepoStub {
	return &billingRetryRepoStub{
		dupKeys:   map[UsageBillingRetryKey]bool{},
		retried:   map[int64]time.Time{},
		retryErrs: map[int64]string{},
		failed:    map[int64]string{},
		byKey:     map[UsageBillingRetryKey]*UsageBillingRetry{},
	}
}

func (s *billingRetryRepoStub) Enqueue(_ context.Context, item *UsageBillingRetry) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enqueueErr != nil {
		return false, s.enqueueErr
	}
	key := UsageBillingRetryKey{RequestID: item.RequestID, APIKeyID: item.APIKeyID}
	if s.dupKeys[key] {
		return false, nil
	}
	s.dupKeys[key] = true
	item.ID = int64(len(s.enqueued) + 1)
	s.enqueued = append(s.enqueued, item)
	return true, nil
}

func (s *billingRetryRepoStub) ClaimDue(_ context.Context, _ time.Time, _ time.Duration, _ int) ([]*UsageBillingRetry, error) {
	s.claimCalls++
	items := s.due
	s.due = nil
	for _, item := range items {
		item.Attempts++
	}
	return items, nil
}

func (s *billingRetryRepoStub) MarkSettled(_ context.Context, id int64, _ time.Time) error {
	s.settled = append(s.settled, id)
	return nil
}

func (s *billingRetryRepoStub) MarkRetry(_ context.Context, id int64, lastError string, next time.Time) error {
	s.retried[id] = next
	s.retryErrs[id] = lastError
	return nil
}

func (s *billingRetryRepoStub) MarkFailed(_ context.Context, id int64, lastError string) error {
	s.failed[id] = lastError
	return nil
}

func (s *billingRetryRepoStub) ListByKeys(_ context.Context, keys []UsageBillingRetryKey) ([]*UsageBillingRetry, error) {
	var out []*UsageBillingRetry
	for _, key := range keys {
		if item, ok := s.byKey[key]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *billingRetryRepoStub) CountCreatedBetween(context.Context, time.Time, time.Time) (int64, error) {
	return s.created, nil
}

func (s *billingRetryRepoStub) Summary(context.Context) (*UsageBillingRetrySummary, error) {
	summary := s.summary
	return &summary, nil
}

type billingApplyOutcome struct {
	result *UsageBillingApplyResult
	err    error
}

type billingApplyStub struct {
	outcomes []billingApplyOutcome
	calls    []*UsageBillingCommand
}

func (s *billingApplyStub) Apply(_ context.Context, cmd *UsageBillingCommand) (*UsageBillingApplyResult, error) {
	s.calls = append(s.calls, cloneUsageBillingCommand(cmd))
	if len(s.outcomes) == 0 {
		return &UsageBillingApplyResult{Applied: true}, nil
	}
	next := s.outcomes[0]
	s.outcomes = s.outcomes[1:]
	return next.result, next.err
}

func (s *billingApplyStub) ReserveBatchImageBalance(context.Context, *BatchImageBalanceHoldCommand) (*BatchImageBalanceHoldResult, error) {
	return nil, nil
}

func (s *billingApplyStub) CaptureBatchImageBalance(context.Context, *BatchImageBalanceHoldCommand) (*BatchImageBalanceHoldResult, error) {
	return nil, nil
}

func (s *billingApplyStub) ReleaseBatchImageBalance(context.Context, *BatchImageBalanceHoldCommand) (*BatchImageBalanceHoldResult, error) {
	return nil, nil
}

type billingNotifySettingRepoStub struct {
	SettingRepository
	values map[string]string
}

func (s *billingNotifySettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if v, ok := s.values[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}

func newBillingRetryServiceForTest(repo *billingRetryRepoStub, apply *billingApplyStub, now time.Time) *UsageBillingRetryService {
	svc := NewUsageBillingRetryService(repo, apply, nil, nil, nil, nil, nil)
	svc.now = func() time.Time { return now }
	return svc
}

func billingRetryParamsForTest() (*UsageLog, *postUsageBillingParams) {
	gid := int64(1)
	usageLog := &UsageLog{
		RequestID: "client:abc", APIKeyID: 6483, UserID: 4641, AccountID: 11, Model: "claude-opus-5-5",
		InputTokens: 4, OutputTokens: 101, CacheReadTokens: 694075, GroupID: &gid,
		TotalCost: 0.324556, ActualCost: 0.0649112,
	}
	p := &postUsageBillingParams{
		Cost:                  &CostBreakdown{TotalCost: 0.324556, ActualCost: 0.0649112},
		User:                  &User{ID: 4641},
		APIKey:                &APIKey{ID: 6483, Quota: 0.01, GroupID: &gid},
		Account:               &Account{ID: 11, Type: AccountTypeOAuth},
		Platform:              "anthropic",
		AccountRateMultiplier: 1,
		APIKeyService:         &openAIRecordUsageAPIKeyQuotaStub{},
	}
	return usageLog, p
}

func TestUsageBillingRetryService_EnqueueFailedBillingStoresCommand(t *testing.T) {
	now := time.Date(2026, 10, 6, 7, 48, 44, 0, time.UTC)
	repo := newBillingRetryRepoStub()
	svc := newBillingRetryServiceForTest(repo, &billingApplyStub{}, now)
	usageLog, p := billingRetryParamsForTest()

	svc.EnqueueFailedBilling(context.Background(), "client:abc", usageLog, p, ErrAPIKeyNotFound)

	require.Len(t, repo.enqueued, 1)
	item := repo.enqueued[0]
	require.Equal(t, "client:abc", item.RequestID)
	require.Equal(t, int64(6483), item.APIKeyID)
	require.Equal(t, int64(4641), item.UserID)
	require.Equal(t, int64(11), item.AccountID)
	require.NotNil(t, item.GroupID)
	require.Equal(t, int64(1), *item.GroupID)
	require.Equal(t, "anthropic", item.Platform)
	require.InDelta(t, 0.0649112, item.ActualCost, 1e-12)
	require.Equal(t, UsageBillingRetryStatusPending, item.Status)
	require.Equal(t, now.Add(usageBillingRetryBaseBackoff), item.NextRetryAt)
	require.Contains(t, item.LastError, "api key not found")
	require.NotNil(t, item.Command)
	require.InDelta(t, 0.0649112, item.Command.BalanceCost, 1e-12)
	require.InDelta(t, 0.0649112, item.Command.APIKeyQuotaCost, 1e-12)
	require.NotEmpty(t, item.Command.RequestFingerprint, "指纹随命令固化，重放按原键去重")
	require.Equal(t, int64(1), svc.newFailures.Load())

	// 同键再次入队：不重复、不重复计数
	svc.EnqueueFailedBilling(context.Background(), "client:abc", usageLog, p, ErrAPIKeyNotFound)
	require.Len(t, repo.enqueued, 1)
	require.Equal(t, int64(1), svc.newFailures.Load())
}

func TestUsageBillingRetryService_EnqueueSkipsWhenNothingToCharge(t *testing.T) {
	now := time.Now()
	repo := newBillingRetryRepoStub()
	svc := newBillingRetryServiceForTest(repo, &billingApplyStub{}, now)

	usageLog, p := billingRetryParamsForTest()
	p.Cost = &CostBreakdown{TotalCost: 0.3, ActualCost: 0}
	svc.EnqueueFailedBilling(context.Background(), "client:free", usageLog, p, errors.New("boom"))
	require.Empty(t, repo.enqueued, "应扣金额为 0 没有可补的钱")

	usageLog, p = billingRetryParamsForTest()
	p.SimpleModeKeyRateLimitOnly = true
	svc.EnqueueFailedBilling(context.Background(), "client:simple", usageLog, p, errors.New("boom"))
	require.Empty(t, repo.enqueued, "简单模式只记窗口用量")

	usageLog, p = billingRetryParamsForTest()
	svc.EnqueueFailedBilling(context.Background(), "client:nil", usageLog, nil, errors.New("boom"))
	require.Empty(t, repo.enqueued)

	// 落库失败：只记日志，不 panic，不计入待推送
	repo.enqueueErr = errors.New("db down")
	usageLog, p = billingRetryParamsForTest()
	svc.EnqueueFailedBilling(context.Background(), "client:abc", usageLog, p, errors.New("boom"))
	require.Zero(t, svc.newFailures.Load())
}

func dueBillingRetry(id int64, attempts int, cmd *UsageBillingCommand) *UsageBillingRetry {
	return &UsageBillingRetry{
		ID: id, RequestID: cmd.RequestID, APIKeyID: cmd.APIKeyID, UserID: cmd.UserID, AccountID: cmd.AccountID,
		Attempts: attempts, Status: UsageBillingRetryStatusPending, Command: cmd,
	}
}

func TestUsageBillingRetryService_ReplaySettlesOnSuccess(t *testing.T) {
	now := time.Now()
	repo := newBillingRetryRepoStub()
	apply := &billingApplyStub{}
	svc := newBillingRetryServiceForTest(repo, apply, now)
	repo.due = []*UsageBillingRetry{dueBillingRetry(7, 0, &UsageBillingCommand{RequestID: "client:abc", APIKeyID: 6483, UserID: 4641, AccountID: 11, BalanceCost: 0.05})}

	svc.runOnce()

	require.Equal(t, 1, repo.claimCalls)
	require.Equal(t, []int64{7}, repo.settled)
	require.Len(t, apply.calls, 1)
	require.Equal(t, "client:abc", apply.calls[0].RequestID)
	require.Empty(t, repo.retried)
	require.Empty(t, repo.failed)
}

func TestUsageBillingRetryService_ReplayBacksOffOnTransientError(t *testing.T) {
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	repo := newBillingRetryRepoStub()
	apply := &billingApplyStub{outcomes: []billingApplyOutcome{
		{err: errors.New("pq: canceling statement due to user request")},
		{err: errors.New("pq: canceling statement due to user request")},
	}}
	svc := newBillingRetryServiceForTest(repo, apply, now)
	cmd := &UsageBillingCommand{RequestID: "client:abc", APIKeyID: 6483, UserID: 4641, BalanceCost: 0.05}
	repo.due = []*UsageBillingRetry{
		dueBillingRetry(7, 0, cmd), // 领取后第 1 次
		dueBillingRetry(8, 2, cmd), // 领取后第 3 次
	}

	svc.runOnce()

	require.Empty(t, repo.settled)
	require.Empty(t, repo.failed)
	require.Equal(t, now.Add(30*time.Second), repo.retried[7])
	require.Equal(t, now.Add(2*time.Minute), repo.retried[8])
	require.Contains(t, repo.retryErrs[7], "canceling statement")
}

func TestUsageBillingRetryService_ReplayAbandonsPermanentErrors(t *testing.T) {
	now := time.Now()
	repo := newBillingRetryRepoStub()
	apply := &billingApplyStub{outcomes: []billingApplyOutcome{
		{err: ErrUsageBillingRequestConflict},
		{err: errors.New("still down")},
		{err: ErrUserNotFound},
	}}
	svc := newBillingRetryServiceForTest(repo, apply, now)
	cmd := &UsageBillingCommand{RequestID: "client:abc", APIKeyID: 6483, UserID: 4641, BalanceCost: 0.05}
	repo.due = []*UsageBillingRetry{
		dueBillingRetry(7, 0, cmd),                              // 指纹冲突：不可重试
		dueBillingRetry(8, usageBillingRetryMaxAttempts-1, cmd), // 领取后达到上限
		dueBillingRetry(9, 0, cmd),                              // 用户已删除：不可重试
	}

	svc.runOnce()

	require.Empty(t, repo.settled)
	require.Empty(t, repo.retried)
	require.Contains(t, repo.failed[7], "fingerprint conflict")
	require.Contains(t, repo.failed[8], "still down")
	require.Contains(t, repo.failed[9], "user not found")
	require.Equal(t, int64(3), svc.abandoned.Load())
}

func TestUsageBillingRetryService_ReplayDropsAccountQuotaWhenAccountGone(t *testing.T) {
	now := time.Now()
	repo := newBillingRetryRepoStub()
	apply := &billingApplyStub{outcomes: []billingApplyOutcome{
		{err: ErrAccountNotFound},
		{result: &UsageBillingApplyResult{Applied: true}},
	}}
	svc := newBillingRetryServiceForTest(repo, apply, now)
	repo.due = []*UsageBillingRetry{dueBillingRetry(7, 0, &UsageBillingCommand{
		RequestID: "client:abc", APIKeyID: 6483, UserID: 4641, AccountID: 11, AccountType: AccountTypeAPIKey,
		BalanceCost: 0.05, AccountQuotaCost: 0.3,
	})}

	svc.runOnce()

	require.Equal(t, []int64{7}, repo.settled)
	require.Len(t, apply.calls, 2)
	require.InDelta(t, 0.3, apply.calls[0].AccountQuotaCost, 1e-12)
	require.Zero(t, apply.calls[1].AccountQuotaCost, "账号已删：去掉账号配额再扣一次用户的钱")
	require.InDelta(t, 0.05, apply.calls[1].BalanceCost, 1e-12)

	// 本来就没有账号配额项：账号不存在属于不可重试
	repo2 := newBillingRetryRepoStub()
	apply2 := &billingApplyStub{outcomes: []billingApplyOutcome{{err: ErrAccountNotFound}}}
	svc2 := newBillingRetryServiceForTest(repo2, apply2, now)
	repo2.due = []*UsageBillingRetry{dueBillingRetry(1, 0, &UsageBillingCommand{RequestID: "client:x", APIKeyID: 1, UserID: 2, BalanceCost: 0.05})}
	svc2.runOnce()
	require.Len(t, apply2.calls, 1)
	require.Contains(t, repo2.failed[1], "account not found")
}

func TestUsageBillingRetryService_ReplayDedupHitSettlesWithoutCharge(t *testing.T) {
	now := time.Now()
	repo := newBillingRetryRepoStub()
	apply := &billingApplyStub{outcomes: []billingApplyOutcome{{result: &UsageBillingApplyResult{Applied: false}}}}
	svc := newBillingRetryServiceForTest(repo, apply, now)
	repo.due = []*UsageBillingRetry{dueBillingRetry(7, 0, &UsageBillingCommand{RequestID: "client:abc", APIKeyID: 6483, UserID: 4641, BalanceCost: 0.05})}

	svc.runOnce()

	require.Equal(t, []int64{7}, repo.settled, "原事务其实已提交：去重命中即结清，不再扣一次")
	require.Len(t, apply.calls, 1)
}

func TestUsageBillingRetryService_AnnotateMarksLogs(t *testing.T) {
	repo := newBillingRetryRepoStub()
	repo.byKey[UsageBillingRetryKey{RequestID: "client:abc", APIKeyID: 6483}] = &UsageBillingRetry{
		RequestID: "client:abc", APIKeyID: 6483, Status: UsageBillingRetryStatusFailed, Attempts: 24, LastError: "boom",
	}
	svc := newBillingRetryServiceForTest(repo, &billingApplyStub{}, time.Now())

	logs := []UsageLog{
		{RequestID: "client:abc", APIKeyID: 6483},
		{RequestID: "client:abc", APIKeyID: 1}, // 同 request_id 不同 Key：不标注
		{RequestID: "", APIKeyID: 6483},
	}
	svc.Annotate(context.Background(), logs)

	require.Equal(t, UsageBillingRetryStatusFailed, logs[0].BillingStatus)
	require.Equal(t, 24, logs[0].BillingAttempts)
	require.Equal(t, "boom", logs[0].BillingError)
	require.Empty(t, logs[1].BillingStatus)
	require.Empty(t, logs[2].BillingStatus)
}

func TestUsageBillingRetryService_AlertSourceReadsRepo(t *testing.T) {
	repo := newBillingRetryRepoStub()
	repo.created = 5
	repo.summary = UsageBillingRetrySummary{Pending: 2, Failed: 3}
	svc := newBillingRetryServiceForTest(repo, &billingApplyStub{}, time.Now())

	n, err := svc.CountBillingFailures(context.Background(), time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	require.Equal(t, int64(5), n)
	unsettled, err := svc.UnsettledBillingCount(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(5), unsettled)
}

func TestUsageBillingRetryBackoff(t *testing.T) {
	cases := map[int]time.Duration{
		0: 30 * time.Second, 1: 30 * time.Second, 2: time.Minute, 3: 2 * time.Minute,
		7: 32 * time.Minute, 8: time.Hour, 20: time.Hour,
	}
	for attempt, want := range cases {
		require.Equal(t, want, usageBillingRetryBackoff(attempt), "attempt %d", attempt)
	}
}

func TestBuildBillingFailureNotifyMessage(t *testing.T) {
	at := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	summary := &UsageBillingRetrySummary{Pending: 4, PendingCost: 0.5, Failed: 1, FailedCost: 0.25}

	title, desp := buildBillingFailureNotifyMessage("CheapRouter", "https://example.com/", 3, 0, summary, at)
	require.Equal(t, "【CheapRouter】扣费失败 3 笔，待补扣 4 笔", title)
	require.Contains(t, desp, "新增扣费失败：3 笔")
	require.Contains(t, desp, "待补扣：4 笔，合计 $0.5000")
	require.Contains(t, desp, "已放弃（需人工处理）：1 笔，合计 $0.2500")
	require.Contains(t, desp, "https://example.com/admin/usage")

	title, desp = buildBillingFailureNotifyMessage("CheapRouter", "", 0, 2, summary, at)
	require.Equal(t, "【CheapRouter】扣费重试放弃 2 笔，待人工处理", title)
	require.NotContains(t, desp, "/admin/usage")
}

func TestUsageBillingRetryService_NotifyRespectsSwitchAndMinGap(t *testing.T) {
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	repo := newBillingRetryRepoStub()
	repo.summary = UsageBillingRetrySummary{Pending: 1, PendingCost: 0.06}
	svc := newBillingRetryServiceForTest(repo, &billingApplyStub{}, now)
	settings := &billingNotifySettingRepoStub{values: map[string]string{
		SettingKeyBillingFailureNotifyServerChanEnabled: "true",
		SettingKeyGroupStatusNotifyServerChanUID:        "12345",
		SettingKeyGroupStatusNotifyServerChanSendKey:    "sk-test",
		SettingKeySiteName:                              "CheapRouter",
	}}
	svc.settingRepo = settings
	delivered := make(chan string, 4)
	svc.notifyDeliver = func(_ context.Context, cfg billingFailureNotifyConfig, title, _ string) error {
		require.Equal(t, "12345", cfg.UID)
		require.Equal(t, "sk-test", cfg.SendKey)
		delivered <- title
		return nil
	}

	svc.newFailures.Store(2)
	svc.maybeNotify()
	select {
	case title := <-delivered:
		require.Contains(t, title, "扣费失败 2 笔")
	case <-time.After(3 * time.Second):
		t.Fatal("expected a push")
	}
	require.Zero(t, svc.newFailures.Load())

	// 最小间隔内再来一笔：不推，计数保留到下次
	svc.newFailures.Store(1)
	svc.maybeNotify()
	select {
	case <-delivered:
		t.Fatal("second push inside the min gap")
	case <-time.After(200 * time.Millisecond):
	}
	require.Equal(t, int64(1), svc.newFailures.Load())

	// 间隔过去后补推
	svc.now = func() time.Time { return now.Add(usageBillingRetryNotifyMinGap + time.Second) }
	svc.maybeNotify()
	select {
	case title := <-delivered:
		require.Contains(t, title, "扣费失败 1 笔")
	case <-time.After(3 * time.Second):
		t.Fatal("expected a push after the gap")
	}

	// 开关关闭：直接投递也静默
	settings.values[SettingKeyBillingFailureNotifyServerChanEnabled] = "false"
	require.NoError(t, svc.deliverNotify(1, 0))
	select {
	case <-delivered:
		t.Fatal("push while disabled")
	case <-time.After(100 * time.Millisecond):
	}
}
