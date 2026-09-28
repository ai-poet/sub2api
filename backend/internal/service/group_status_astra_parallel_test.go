package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

// astraRoutingUpstream 按请求里的模型与题目作答（与请求到达顺序无关），并统计每个账号的并发在途数。
type astraRoutingUpstream struct {
	mu          sync.Mutex
	answers     map[string][2]string // 请求模型 → c1 / c2 的答案
	delay       time.Duration
	inFlight    map[int64]int
	maxInFlight map[int64]int
	accounts    map[string]map[int64]int // 请求模型 → 账号 → 请求数
}

func newAstraRoutingUpstream(delay time.Duration, answers map[string][2]string) *astraRoutingUpstream {
	return &astraRoutingUpstream{
		answers:     answers,
		delay:       delay,
		inFlight:    map[int64]int{},
		maxInFlight: map[int64]int{},
		accounts:    map[string]map[int64]int{},
	}
}

func (u *astraRoutingUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return nil, errors.New("unexpected Do call")
}

func (u *astraRoutingUpstream) DoWithTLS(req *http.Request, _ string, accountID int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	raw, _ := io.ReadAll(req.Body)
	body := string(raw)

	u.mu.Lock()
	u.inFlight[accountID]++
	if u.inFlight[accountID] > u.maxInFlight[accountID] {
		u.maxInFlight[accountID] = u.inFlight[accountID]
	}
	u.mu.Unlock()

	time.Sleep(u.delay)

	u.mu.Lock()
	defer u.mu.Unlock()
	u.inFlight[accountID]--
	for model, answers := range u.answers {
		if !strings.Contains(body, `"model":"`+model+`"`) {
			continue
		}
		if u.accounts[model] == nil {
			u.accounts[model] = map[int64]int{}
		}
		u.accounts[model][accountID]++
		answer := answers[0]
		if strings.Contains(body, "prompt c2") {
			answer = answers[1]
		}
		return groupStatusProbeResponse(200, responsesSSEBody(answer, 10)), nil
	}
	return groupStatusProbeResponse(400, `{"error":{"message":"unknown model"}}`), nil
}

func (u *astraRoutingUpstream) accountsFor(model string) []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	out := []int64{}
	for id := range u.accounts[model] {
		out = append(out, id)
	}
	return out
}

func newParallelAstraFixture(t *testing.T, accountConcurrency []int, upstream *astraRoutingUpstream) *astraProbeFixture {
	t.Helper()
	models := onlyModels("gpt-6-sol", "gpt-6-astra")
	f := newAstraProbeFixture(t, PlatformOpenAI, models)
	accounts := make([]Account, 0, len(accountConcurrency))
	for i, concurrency := range accountConcurrency {
		account := groupStatusProbeAccount(int64(i+1), PlatformOpenAI, f.group.ID, i+1, map[string]any{"gpt-6-sol": "gpt-6-sol", "gpt-6-astra": "gpt-6-astra"})
		account.Concurrency = concurrency
		account.Credentials["api_key"] = "sk-test"
		accounts = append(accounts, account)
	}
	svc := newGroupStatusProbeServiceForTest(f.group, accounts, nil, nil, nil)
	svc.accountTestSvc.httpUpstream = upstream
	svc.repo = f.repo
	svc.SetTransitionNotifier(f.notifier)
	svc.astraSleep = func(context.Context, time.Duration) error { return nil }
	svc.SetAstraBenchmarkProvider(staticAstraRegistryProvider{reg: syntheticAstraRegistry(t)})
	f.svc = svc
	return f
}

func TestAstraCheckProbe_ParallelModelsSpreadAcrossAccounts(t *testing.T) {
	upstream := newAstraRoutingUpstream(15*time.Millisecond, map[string][2]string{
		"gpt-6-sol":   {"b", "2"},
		"gpt-6-astra": {"a", "1"},
	})
	f := newParallelAstraFixture(t, []int{4, 4}, upstream)

	executions := f.run(t)
	require.Len(t, executions, 2)
	require.Equal(t, "gpt-6-sol", executions[0].Result.ExpectedModel)
	require.Equal(t, AstraCheckVerdictMatch, executions[0].Result.Verdict)
	require.Equal(t, "gpt-6-astra", executions[1].Result.ExpectedModel)
	require.Equal(t, AstraCheckVerdictMatch, executions[1].Result.Verdict)

	// 两个模型各自锁定一个账号，且摊到了不同账号上
	solAccounts := upstream.accountsFor("gpt-6-sol")
	astraAccounts := upstream.accountsFor("gpt-6-astra")
	require.Len(t, solAccounts, 1)
	require.Len(t, astraAccounts, 1)
	require.NotEqual(t, solAccounts[0], astraAccounts[0])
	require.Equal(t, solAccounts[0], *executions[0].Result.AccountID)

	require.False(t, f.svc.IsAstraCheckRunning(f.group.ID))
	require.Empty(t, f.svc.AstraCheckProgresses(f.group.ID))
	require.Empty(t, f.svc.astraAccounts.busy())
}

func TestAstraCheckProbe_ParallelModelsShareOneAccountWithinItsConcurrency(t *testing.T) {
	upstream := newAstraRoutingUpstream(15*time.Millisecond, map[string][2]string{
		"gpt-6-sol":   {"b", "2"},
		"gpt-6-astra": {"a", "1"},
	})
	f := newParallelAstraFixture(t, []int{2}, upstream)

	executions := f.run(t)
	require.Len(t, executions, 2)
	for _, execution := range executions {
		require.Equal(t, AstraCheckVerdictMatch, execution.Result.Verdict)
		require.Equal(t, int64(1), *execution.Result.AccountID)
	}
	// 两个模型共用唯一的账号：合计在途不超过账号并发 2
	require.LessOrEqual(t, upstream.maxInFlight[1], 2)
	require.Equal(t, []int64{1}, upstream.accountsFor("gpt-6-sol"))
	require.Equal(t, []int64{1}, upstream.accountsFor("gpt-6-astra"))
}

func TestAstraAccountRegistry_CapacityAndBusy(t *testing.T) {
	var reg astraAccountRegistry

	releaseA, err := reg.acquire(context.Background(), 7, 1)
	require.NoError(t, err)
	// 名额已满：在 ctx 结束前拿不到
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = reg.acquire(ctx, 7, 1)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	releaseA()
	releaseA() // 重复释放无副作用
	releaseB, err := reg.acquire(context.Background(), 7, 1)
	require.NoError(t, err)
	releaseB()

	doneA := reg.use(3)
	doneB := reg.use(3)
	require.Contains(t, reg.busy(), int64(3))
	doneA()
	doneA()
	require.Contains(t, reg.busy(), int64(3))
	doneB()
	require.Empty(t, reg.busy())
}

func TestAstraAccountCapacity(t *testing.T) {
	svc := &GroupStatusProbeService{}
	require.Equal(t, groupStatusAstraCheckDefaultConcurrency, svc.astraAccountCapacity(&Account{}))
	require.Equal(t, 3, svc.astraAccountCapacity(&Account{Concurrency: 3}))
	require.Equal(t, groupStatusAstraCheckDefaultConcurrency, svc.astraAccountCapacity(&Account{Concurrency: 100}))
	require.Equal(t, groupStatusAstraCheckModelParallelism, svc.astraCheckModelParallelism())
}
