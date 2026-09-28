package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// meow 指纹验证的账号占用登记（本 fork 自有功能）。
//
// 一个分组的多个预期模型、以及定时批次里的多个分组会并行检测。为了既快又不压垮上游账号：
//   - 每个账号一个服务级信号量，所有并行运行在该账号上的在途请求合计不超过它的容量（取账号并发数与服务上限的较小值）；
//   - 选账号时优先挑还没被其他并行运行占用的账号，这样多模型能摊到不同账号上真正并行；都被占用时再共用。

type astraAccountSlot struct {
	capacity int
	ch       chan struct{}
}

type astraAccountRegistry struct {
	mu    sync.Mutex
	slots map[int64]*astraAccountSlot
	users map[int64]int
	// selectMu 串行化「选账号 + 登记占用」，避免两个并行运行同时选中同一个空闲账号
	selectMu sync.Mutex
}

// slot 返回账号的信号量；容量变化（账号并发被改）时换一个新的，旧的由在途请求自然释放。
func (r *astraAccountRegistry) slot(accountID int64, capacity int) *astraAccountSlot {
	if capacity < 1 {
		capacity = 1
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.slots == nil {
		r.slots = make(map[int64]*astraAccountSlot)
	}
	slot := r.slots[accountID]
	if slot == nil || slot.capacity != capacity {
		slot = &astraAccountSlot{capacity: capacity, ch: make(chan struct{}, capacity)}
		r.slots[accountID] = slot
	}
	return slot
}

// acquire 占用账号的一个在途名额，直到 ctx 结束；返回的 release 必须调用一次。
func (r *astraAccountRegistry) acquire(ctx context.Context, accountID int64, capacity int) (func(), error) {
	slot := r.slot(accountID, capacity)
	select {
	case slot.ch <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-slot.ch }) }, nil
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
}

// use 登记一个并行运行正在使用该账号；返回的 done 撤销登记（可重复调用）。
func (r *astraAccountRegistry) use(accountID int64) func() {
	r.mu.Lock()
	if r.users == nil {
		r.users = make(map[int64]int)
	}
	r.users[accountID]++
	r.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.users[accountID] <= 1 {
				delete(r.users, accountID)
				return
			}
			r.users[accountID]--
		})
	}
}

// busy 返回正被其他并行运行使用的账号。
func (r *astraAccountRegistry) busy() map[int64]struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[int64]struct{}, len(r.users))
	for id := range r.users {
		out[id] = struct{}{}
	}
	return out
}

// astraAccountCapacity 是单个账号上所有并行检测的在途请求上限。
func (s *GroupStatusProbeService) astraAccountCapacity(account *Account) int {
	capacity := s.astraCheckConcurrency()
	if account != nil && account.Concurrency > 0 && capacity > account.Concurrency {
		capacity = account.Concurrency
	}
	return capacity
}

// astraCheckModelParallelism 是一次运行里同时检测的模型数上限。
func (s *GroupStatusProbeService) astraCheckModelParallelism() int {
	if s != nil && s.astraModelParallelism > 0 {
		return s.astraModelParallelism
	}
	return groupStatusAstraCheckModelParallelism
}

// selectAstraAccount 选一个账号并登记占用：先排除正被其他并行运行使用的账号，选不到再允许共用。
func (s *GroupStatusProbeService) selectAstraAccount(ctx context.Context, group *Group, cfg *GroupStatusConfig, excludedIDs map[int64]struct{}) (*groupStatusProbeAttempt, func(), error) {
	s.astraAccounts.selectMu.Lock()
	defer s.astraAccounts.selectMu.Unlock()

	busy := s.astraAccounts.busy()
	preferred := make(map[int64]struct{}, len(excludedIDs)+len(busy))
	for id := range excludedIDs {
		preferred[id] = struct{}{}
	}
	extra := 0
	for id := range busy {
		if _, ok := preferred[id]; !ok {
			preferred[id] = struct{}{}
			extra++
		}
	}
	attempt, err := s.selectProbeAttempt(ctx, group, cfg, preferred)
	if (err != nil || attempt == nil || attempt.Account == nil) && extra > 0 {
		fallback := make(map[int64]struct{}, len(excludedIDs))
		for id := range excludedIDs {
			fallback[id] = struct{}{}
		}
		attempt, err = s.selectProbeAttempt(ctx, group, cfg, fallback)
	}
	if err != nil || attempt == nil || attempt.Account == nil {
		return attempt, func() {}, err
	}
	return attempt, s.astraAccounts.use(attempt.Account.ID), nil
}

// astraAccountLock 是一次运行锁定账号的结果；release 撤销占用登记，运行结束时调用。
type astraAccountLock struct {
	account            *Account
	first              *astraSample
	failed             []astraSample // 换号前失败的首个任务，保留以便汇总 HTTP 码与 token
	firstFailureDetail string
	noAccountDetail    string // account 为 nil 时的原因
	release            func()
}

// lockAstraAccount 为一次运行锁定账号：pinned 可用时先用它（复测确认），否则按调度器选（优先未被占用的账号）；
// 在候选账号上同步跑 first，传输 / 上游错误按存活探测的规则换号。拿到结果后整次运行固定在该账号。
func (s *GroupStatusProbeService) lockAstraAccount(ctx context.Context, group *Group, probeCfg *GroupStatusConfig, pinned *Account, progress *astraProgressTracker, first func(*Account) astraSample) *astraAccountLock {
	lock := &astraAccountLock{release: func() {}}
	excludedIDs := make(map[int64]struct{})
	maxAttempts := s.maxProbeAttempts(group)
	tryPinned := pinned != nil && astraAccountCompatible(group, pinned)
	for attemptNo := 0; attemptNo < maxAttempts; attemptNo++ {
		var candidate *Account
		if tryPinned {
			tryPinned = false
			candidate = pinned
			lock.release = s.astraAccounts.use(pinned.ID)
		} else {
			attempt, done, selectErr := s.selectAstraAccount(ctx, group, probeCfg, excludedIDs)
			if selectErr != nil {
				lock.noAccountDetail = "no schedulable account: " + selectErr.Error()
				return lock
			}
			if attempt == nil || attempt.Account == nil {
				done()
				lock.noAccountDetail = "no schedulable account available"
				return lock
			}
			candidate = attempt.Account
			if _, excluded := excludedIDs[candidate.ID]; excluded || !astraAccountCompatible(group, candidate) || attempt.WaitPlan != nil {
				done()
				excludedIDs[candidate.ID] = struct{}{}
				continue
			}
			lock.release = done
		}
		progress.setAccount(candidate.ID)
		sample := first(candidate)
		if sample.TransportFailed {
			failure := &GroupStatusProbeResult{HTTPCode: sample.HTTPCode}
			if s.shouldProbeFailover(candidate, failure, errors.New(sample.ErrDetail)) && attemptNo < maxAttempts-1 {
				if lock.firstFailureDetail == "" {
					lock.firstFailureDetail = fmt.Sprintf("account %d: %s", candidate.ID, truncateProbeText(sample.ErrDetail))
				}
				lock.failed = append(lock.failed, sample)
				excludedIDs[candidate.ID] = struct{}{}
				lock.release()
				lock.release = func() {}
				// 首个任务会在下一个账号上重跑，不算已完成
				progress.rollbackJob(AstraCheckSampleFailed)
				continue
			}
		}
		lock.account = candidate
		lock.first = &sample
		return lock
	}
	lock.noAccountDetail = "failover_exhausted"
	return lock
}
