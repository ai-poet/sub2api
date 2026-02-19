package service

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

const (
	shopCleanupJobName              = "shop_cleanup"
	shopCleanupLeaderLockKeyDefault = "shop:cleanup:leader"
	shopCleanupLeaderLockTTLDefault = 5 * time.Minute
)

var shopCleanupCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

var shopCleanupReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

// ShopCleanupService periodically cleans up expired shop orders.
//
// - Scheduling: 5-field cron spec (minute hour dom month dow).
// - Multi-instance: best-effort Redis leader lock so only one node runs cleanup.
// - Default: runs every 5 minutes.
type ShopCleanupService struct {
	shopSvc     *ShopService
	redisClient *redis.Client
	cfg         *config.Config

	instanceID string

	cron *cron.Cron

	startOnce sync.Once
	stopOnce  sync.Once

	warnNoRedisOnce sync.Once
}

func NewShopCleanupService(
	shopSvc *ShopService,
	redisClient *redis.Client,
	cfg *config.Config,
) *ShopCleanupService {
	return &ShopCleanupService{
		shopSvc:     shopSvc,
		redisClient: redisClient,
		cfg:         cfg,
		instanceID:  uuid.NewString(),
	}
}

func (s *ShopCleanupService) Start() {
	if s == nil || s.shopSvc == nil {
		return
	}

	s.startOnce.Do(func() {
		// Default: run every 5 minutes
		schedule := "*/5 * * * *"
		if s.cfg != nil && strings.TrimSpace(s.cfg.Shop.CleanupSchedule) != "" {
			schedule = strings.TrimSpace(s.cfg.Shop.CleanupSchedule)
		}

		loc := time.Local
		if s.cfg != nil && strings.TrimSpace(s.cfg.Timezone) != "" {
			if parsed, err := time.LoadLocation(strings.TrimSpace(s.cfg.Timezone)); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(shopCleanupCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc(schedule, func() { s.runScheduled() })
		if err != nil {
			log.Printf("[ShopCleanup] not started (invalid schedule=%q): %v", schedule, err)
			return
		}
		s.cron = c
		s.cron.Start()
		log.Printf("[ShopCleanup] started (schedule=%q tz=%s)", schedule, loc.String())
	})
}

func (s *ShopCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				log.Printf("[ShopCleanup] cron stop timed out")
			}
		}
	})
}

func (s *ShopCleanupService) runScheduled() {
	if s == nil || s.shopSvc == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	release, ok := s.tryAcquireLeaderLock(ctx)
	if !ok {
		return
	}
	if release != nil {
		defer release()
	}

	startedAt := time.Now().UTC()

	count, err := s.shopSvc.CleanupExpiredOrders(ctx)
	if err != nil {
		log.Printf("[ShopCleanup] cleanup failed: %v (duration=%v)", err, time.Since(startedAt))
		return
	}

	if count > 0 {
		log.Printf("[ShopCleanup] cleanup complete: expired=%d duration=%v", count, time.Since(startedAt))
	}
}

func (s *ShopCleanupService) tryAcquireLeaderLock(ctx context.Context) (func(), bool) {
	if s == nil {
		return nil, false
	}
	// In simple run mode, assume single instance.
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		return nil, true
	}

	key := shopCleanupLeaderLockKeyDefault
	ttl := shopCleanupLeaderLockTTLDefault

	// Prefer Redis leader lock when available
	if s.redisClient != nil {
		ok, err := s.redisClient.SetNX(ctx, key, s.instanceID, ttl).Result()
		if err == nil {
			if !ok {
				return nil, false
			}
			return func() {
				_, _ = shopCleanupReleaseScript.Run(ctx, s.redisClient, []string{key}, s.instanceID).Result()
			}, true
		}
		s.warnNoRedisOnce.Do(func() {
			log.Printf("[ShopCleanup] leader lock SetNX failed: %v", err)
		})
	}

	// If no Redis, just run (no distributed lock)
	return nil, true
}
