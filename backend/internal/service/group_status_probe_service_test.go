package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type groupStatusProbeAccountRepo struct {
	AccountRepository
	accounts []Account
}

func (r *groupStatusProbeAccountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			return &r.accounts[i], nil
		}
	}
	return nil, ErrAccountNotFound
}

func (r *groupStatusProbeAccountRepo) SetError(ctx context.Context, id int64, errorMsg string) error {
	return nil
}

func (r *groupStatusProbeAccountRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	return r.listByPlatforms(groupID, map[string]struct{}{platform: {}}), nil
}

func (r *groupStatusProbeAccountRepo) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.listByPlatforms(0, map[string]struct{}{platform: {}}), nil
}

func (r *groupStatusProbeAccountRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}

func (r *groupStatusProbeAccountRepo) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
	return r.listByPlatforms(groupID, stringSet(platforms)), nil
}

func (r *groupStatusProbeAccountRepo) ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	return r.listByPlatforms(0, stringSet(platforms)), nil
}

func (r *groupStatusProbeAccountRepo) ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	return r.ListSchedulableByPlatforms(ctx, platforms)
}

func (r *groupStatusProbeAccountRepo) listByPlatforms(groupID int64, platforms map[string]struct{}) []Account {
	result := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if _, ok := platforms[account.Platform]; !ok {
			continue
		}
		if groupID > 0 && !probeTestAccountInGroup(account, groupID) {
			continue
		}
		result = append(result, account)
	}
	return result
}

func probeTestAccountInGroup(account Account, groupID int64) bool {
	for _, ag := range account.AccountGroups {
		if ag.GroupID == groupID {
			return true
		}
	}
	for _, id := range account.GroupIDs {
		if id == groupID {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

type groupStatusProbeGroupRepo struct {
	GroupRepository
	groups map[int64]*Group
}

func (r *groupStatusProbeGroupRepo) GetByID(ctx context.Context, id int64) (*Group, error) {
	return r.get(id)
}

func (r *groupStatusProbeGroupRepo) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return r.get(id)
}

func (r *groupStatusProbeGroupRepo) get(id int64) (*Group, error) {
	if r == nil || r.groups == nil {
		return nil, ErrGroupNotFound
	}
	group, ok := r.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

type groupStatusProbeRepo struct {
	GroupStatusRepository
	lastResult *GroupStatusProbeResult
}

func (r *groupStatusProbeRepo) SaveProbeResult(ctx context.Context, result *GroupStatusProbeResult) (*GroupStatusState, *GroupStatusEvent, error) {
	copied := *result
	r.lastResult = &copied
	return &GroupStatusState{
		GroupID:      result.GroupID,
		ConfigID:     result.ConfigID,
		LatestStatus: result.Status,
		StableStatus: result.Status,
		SubStatus:    result.SubStatus,
		ErrorDetail:  result.ErrorDetail,
		ObservedAt:   &result.ObservedAt,
	}, nil, nil
}

type groupStatusProbeConcurrencyCache struct {
	ConcurrencyCache
	loadMap        map[int64]*AccountLoadInfo
	acquireResults map[int64]bool
	waitCounts     map[int64]int
	released       map[int64]int
}

func (c *groupStatusProbeConcurrencyCache) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
	if c.acquireResults != nil {
		if acquired, ok := c.acquireResults[accountID]; ok {
			return acquired, nil
		}
	}
	return true, nil
}

func (c *groupStatusProbeConcurrencyCache) ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error {
	if c.released == nil {
		c.released = make(map[int64]int)
	}
	c.released[accountID]++
	return nil
}

func (c *groupStatusProbeConcurrencyCache) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	if c.waitCounts != nil {
		return c.waitCounts[accountID], nil
	}
	return 0, nil
}

func (c *groupStatusProbeConcurrencyCache) GetAccountsLoadBatch(ctx context.Context, accounts []AccountWithConcurrency) (map[int64]*AccountLoadInfo, error) {
	result := make(map[int64]*AccountLoadInfo, len(accounts))
	for _, account := range accounts {
		if c.loadMap != nil {
			if load, ok := c.loadMap[account.ID]; ok {
				result[account.ID] = load
				continue
			}
		}
		result[account.ID] = &AccountLoadInfo{AccountID: account.ID, LoadRate: 0}
	}
	return result, nil
}

type groupStatusProbeHTTPUpstream struct {
	responses []*http.Response
	requests  []*http.Request
}

func (u *groupStatusProbeHTTPUpstream) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return nil, errors.New("unexpected Do call")
}

func (u *groupStatusProbeHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	u.requests = append(u.requests, req)
	if len(u.responses) == 0 {
		return nil, fmt.Errorf("no mocked response")
	}
	resp := u.responses[0]
	u.responses = u.responses[1:]
	return resp, nil
}

func groupStatusProbeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newGroupStatusProbeServiceForTest(group *Group, accounts []Account, cache *groupStatusProbeConcurrencyCache, upstream *groupStatusProbeHTTPUpstream, cfg *config.Config) *GroupStatusProbeService {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cache == nil {
		cache = &groupStatusProbeConcurrencyCache{}
	}
	accountRepo := &groupStatusProbeAccountRepo{accounts: accounts}
	groupRepo := &groupStatusProbeGroupRepo{groups: map[int64]*Group{group.ID: group}}
	concurrencySvc := NewConcurrencyService(cache)
	gatewayCache := &stubGatewayCache{}
	gatewaySvc := &GatewayService{
		accountRepo:        accountRepo,
		groupRepo:          groupRepo,
		cache:              gatewayCache,
		cfg:                cfg,
		concurrencyService: concurrencySvc,
	}
	openAIGatewaySvc := &OpenAIGatewayService{
		accountRepo:        accountRepo,
		cache:              gatewayCache,
		cfg:                cfg,
		concurrencyService: concurrencySvc,
	}
	return &GroupStatusProbeService{
		repo:             &groupStatusProbeRepo{},
		groupRepo:        groupRepo,
		accountTestSvc:   NewAccountTestService(accountRepo, nil, nil, upstream, cfg, &TLSFingerprintProfileService{}),
		gatewaySvc:       gatewaySvc,
		openAIGatewaySvc: openAIGatewaySvc,
	}
}

func groupStatusProbeConfig(groupID int64, model string) *GroupStatusConfig {
	return &GroupStatusConfig{
		ID:              101,
		GroupID:         groupID,
		ProbeModel:      model,
		ProbePrompt:     "Please reply ONLINE.",
		ValidationMode:  GroupStatusValidationNonEmpty,
		IntervalSeconds: 60,
		TimeoutSeconds:  5,
		SlowLatencyMS:   15000,
	}
}

func groupStatusProbeAccount(id int64, platform string, groupID int64, priority int, mapping map[string]any) Account {
	return Account{
		ID:          id,
		Platform:    platform,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    priority,
		Credentials: map[string]any{
			"api_key":       "test-key",
			"base_url":      "https://example.com",
			"model_mapping": mapping,
		},
		AccountGroups: []AccountGroup{{GroupID: groupID}},
	}
}

func TestGroupStatusProbe_SelectsLowerPriorityAccountSupportingProbeModel(t *testing.T) {
	group := &Group{ID: 10, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "claude-sonnet-4-5")
	accounts := []Account{
		groupStatusProbeAccount(1, PlatformAnthropic, group.ID, 1, map[string]any{"claude-haiku-4-5": "claude-haiku-4-5"}),
		groupStatusProbeAccount(2, PlatformAnthropic, group.ID, 2, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"}),
	}

	svc := newGroupStatusProbeServiceForTest(group, accounts, nil, nil, nil)
	attempt, err := svc.selectProbeAttempt(context.Background(), group, cfg, nil)
	require.NoError(t, err)
	require.NotNil(t, attempt)
	require.Equal(t, int64(2), attempt.Account.ID)
}

func TestGroupStatusProbe_SkipsModelRateLimitedAccount(t *testing.T) {
	group := &Group{ID: 11, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "claude-sonnet-4-5")
	resetAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	limited := groupStatusProbeAccount(1, PlatformAnthropic, group.ID, 1, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"})
	limited.Extra = map[string]any{
		modelRateLimitsKey: map[string]any{
			"claude-sonnet-4-5": map[string]any{"rate_limit_reset_at": resetAt},
		},
	}
	available := groupStatusProbeAccount(2, PlatformAnthropic, group.ID, 2, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"})

	svc := newGroupStatusProbeServiceForTest(group, []Account{limited, available}, nil, nil, nil)
	attempt, err := svc.selectProbeAttempt(context.Background(), group, cfg, nil)
	require.NoError(t, err)
	require.NotNil(t, attempt)
	require.Equal(t, int64(2), attempt.Account.ID)
}

func TestGroupStatusProbe_QueueingDoesNotCallUpstream(t *testing.T) {
	group := &Group{ID: 12, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "claude-sonnet-4-5")
	account := groupStatusProbeAccount(1, PlatformAnthropic, group.ID, 1, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"})
	cache := &groupStatusProbeConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			1: {AccountID: 1, CurrentConcurrency: 1, LoadRate: 100},
		},
		acquireResults: map[int64]bool{1: false},
	}
	upstream := &groupStatusProbeHTTPUpstream{}
	svc := newGroupStatusProbeServiceForTest(group, []Account{account}, cache, upstream, nil)

	execution, err := svc.executeProbe(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Equal(t, GroupRuntimeStatusDegraded, execution.Result.Status)
	require.Equal(t, "queueing", execution.Result.SubStatus)
	require.Empty(t, upstream.requests)
}

func TestGroupStatusProbe_FailoverRecoveredRecordsDegraded(t *testing.T) {
	group := &Group{ID: 13, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "claude-sonnet-4-5")
	accounts := []Account{
		groupStatusProbeAccount(1, PlatformAnthropic, group.ID, 1, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"}),
		groupStatusProbeAccount(2, PlatformAnthropic, group.ID, 2, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"}),
	}
	upstream := &groupStatusProbeHTTPUpstream{
		responses: []*http.Response{
			groupStatusProbeResponse(http.StatusTooManyRequests, "rate limited"),
			groupStatusProbeResponse(http.StatusOK, `data: {"type":"content_block_delta","delta":{"text":"ONLINE"}}
data: {"type":"message_stop"}

`),
		},
	}
	svc := newGroupStatusProbeServiceForTest(group, accounts, nil, upstream, nil)

	execution, err := svc.executeProbe(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Equal(t, int64(2), execution.Account.ID)
	require.Equal(t, GroupRuntimeStatusDegraded, execution.Result.Status)
	require.Equal(t, "failover_recovered", execution.Result.SubStatus)
	require.Contains(t, execution.Result.ErrorDetail, "account 1")
	require.Len(t, upstream.requests, 2)
}

func TestGroupStatusProbe_AllCandidatesFailDown(t *testing.T) {
	group := &Group{ID: 14, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "claude-sonnet-4-5")
	cfg.SlowLatencyMS = int64(time.Hour / time.Millisecond)
	cfg.TimeoutSeconds = 5
	account := groupStatusProbeAccount(1, PlatformAnthropic, group.ID, 1, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"})
	upstream := &groupStatusProbeHTTPUpstream{
		responses: []*http.Response{groupStatusProbeResponse(http.StatusTooManyRequests, "rate limited")},
	}
	svc := newGroupStatusProbeServiceForTest(group, []Account{account}, nil, upstream, nil)

	execution, err := svc.executeProbe(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Equal(t, GroupRuntimeStatusDown, execution.Result.Status)
	require.Equal(t, "failover_exhausted", execution.Result.SubStatus)
	require.Len(t, upstream.requests, 1)
}

func TestGroupStatusProbe_OpenAIUsesOpenAISchedulerRules(t *testing.T) {
	group := &Group{ID: 15, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "gpt-5.2")
	accounts := []Account{
		groupStatusProbeAccount(1, PlatformOpenAI, group.ID, 1, map[string]any{"gpt-4.1": "gpt-4.1"}),
		groupStatusProbeAccount(2, PlatformOpenAI, group.ID, 2, map[string]any{"gpt-5.2": "gpt-5.2"}),
	}
	for i := range accounts {
		accounts[i].Credentials["api_key"] = "sk-test"
	}

	svc := newGroupStatusProbeServiceForTest(group, accounts, nil, nil, nil)
	attempt, err := svc.selectProbeAttempt(context.Background(), group, cfg, nil)
	require.NoError(t, err)
	require.NotNil(t, attempt)
	require.Equal(t, int64(2), attempt.Account.ID)
}

func TestGroupStatusProbe_GeminiMixedSchedulingUsesAntigravityAccount(t *testing.T) {
	group := &Group{ID: 16, Platform: PlatformGemini, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "gemini-3-flash")
	nativeGemini := groupStatusProbeAccount(1, PlatformGemini, group.ID, 1, map[string]any{"gemini-2.5-pro": "gemini-2.5-pro"})
	antigravity := groupStatusProbeAccount(2, PlatformAntigravity, group.ID, 1, map[string]any{"gemini-3-flash": "gemini-3-flash"})
	antigravity.Extra = map[string]any{"mixed_scheduling": true}

	svc := newGroupStatusProbeServiceForTest(group, []Account{nativeGemini, antigravity}, nil, nil, nil)
	attempt, err := svc.selectProbeAttempt(context.Background(), group, cfg, nil)
	require.NoError(t, err)
	require.NotNil(t, attempt)
	require.Equal(t, int64(2), attempt.Account.ID)
	require.Equal(t, PlatformAntigravity, attempt.Account.Platform)
}
