package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type GroupStatusRunnerService struct {
	repo     GroupStatusRepository
	probeSvc *GroupStatusProbeService
	cfg      *config.Config

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup

	// solJuiceRunning 防止一批慢的 high 档 Juice 请求与下一次 tick 重叠
	solJuiceRunning atomic.Bool
	// astraRunning 同理，Astra 指纹一次要跑几十个请求
	astraRunning atomic.Bool
}

func NewGroupStatusRunnerService(
	repo GroupStatusRepository,
	probeSvc *GroupStatusProbeService,
	cfg *config.Config,
) *GroupStatusRunnerService {
	return &GroupStatusRunnerService{
		repo:     repo,
		probeSvc: probeSvc,
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}
}

func (s *GroupStatusRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.loop()
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] started")
	})
}

func (s *GroupStatusRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *GroupStatusRunnerService) loop() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	s.runOnce()
	s.startSolJuiceBatch()
	s.startAstraCheckBatch()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runOnce()
			// Juice / Astra 探测都可能很慢，放到独立 goroutine 里，不拖慢存活探测
			s.startSolJuiceBatch()
			s.startAstraCheckBatch()
		case <-cleanupTicker.C:
			s.cleanupOldRecords()
		}
	}
}

func (s *GroupStatusRunnerService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	configs, err := s.repo.ListDueConfigs(ctx, time.Now(), 100)
	if err != nil {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] list due configs failed: %v", err)
		return
	}
	for _, cfg := range configs {
		if _, err := s.probeSvc.ProbeWithConfig(ctx, cfg); err != nil {
			logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] probe group=%d failed: %v", cfg.GroupID, err)
		}
	}
}

func (s *GroupStatusRunnerService) startSolJuiceBatch() {
	if !s.solJuiceRunning.CompareAndSwap(false, true) {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.solJuiceRunning.Store(false)
		s.runSolJuiceOnce()
	}()
}

// batchContext 给后台批次一个有上限的 ctx，并在 Stop() 时尽快取消。
func (s *GroupStatusRunnerService) batchContext(budget time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	go func() {
		select {
		case <-s.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

func (s *GroupStatusRunnerService) runSolJuiceOnce() {
	ctx, cancel := s.batchContext(10 * time.Minute)
	defer cancel()

	configs, err := s.repo.ListDueSolJuiceConfigs(ctx, time.Now(), 10)
	if err != nil {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] list due sol juice configs failed: %v", err)
		return
	}
	for _, cfg := range configs {
		if ctx.Err() != nil {
			return
		}
		if _, err := s.probeSvc.ProbeSolJuiceWithConfig(ctx, cfg); err != nil {
			logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] sol juice probe group=%d failed: %v", cfg.GroupID, err)
		}
	}
}

func (s *GroupStatusRunnerService) startAstraCheckBatch() {
	if !s.astraRunning.CompareAndSwap(false, true) {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.astraRunning.Store(false)
		s.runAstraCheckOnce()
	}()
}

func (s *GroupStatusRunnerService) runAstraCheckOnce() {
	ctx, cancel := s.batchContext(25 * time.Minute)
	defer cancel()

	configs, err := s.repo.ListDueAstraCheckConfigs(ctx, time.Now(), 3)
	if err != nil {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] list due astra check configs failed: %v", err)
		return
	}
	for _, cfg := range configs {
		if ctx.Err() != nil {
			return
		}
		if _, err := s.probeSvc.ProbeAstraCheckWithConfig(ctx, cfg); err != nil {
			logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] astra check group=%d failed: %v", cfg.GroupID, err)
		}
	}
}

func (s *GroupStatusRunnerService) cleanupOldRecords() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	before := time.Now().AddDate(0, 0, -groupStatusRetentionDays)
	deleted, err := s.repo.DeleteRecordsOlderThan(ctx, before)
	if err != nil {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] cleanup failed: %v", err)
		return
	}
	if deleted > 0 {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] cleaned %d old records", deleted)
	}

	deletedJuice, err := s.repo.DeleteSolJuiceRecordsOlderThan(ctx, before)
	if err != nil {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] sol juice cleanup failed: %v", err)
	} else if deletedJuice > 0 {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] cleaned %d old sol juice records", deletedJuice)
	}

	deletedAstra, err := s.repo.DeleteAstraCheckRunsOlderThan(ctx, before)
	if err != nil {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] astra check cleanup failed: %v", err)
	} else if deletedAstra > 0 {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] cleaned %d old astra check runs", deletedAstra)
	}
}
