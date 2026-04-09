package service

import (
	"context"
	"sync"
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

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runOnce()
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
}
