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

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runOnce()
			// Juice 探测走 high 档、单条可达两分钟，放到独立 goroutine 里，不拖慢存活探测
			s.startSolJuiceBatch()
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

func (s *GroupStatusRunnerService) runSolJuiceOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	// Stop() 时尽快结束在途的 Juice 批次
	go func() {
		select {
		case <-s.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

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
		return
	}
	if deletedJuice > 0 {
		logger.LegacyPrintf("service.group_status_runner", "[GroupStatusRunner] cleaned %d old sol juice records", deletedJuice)
	}
}
