package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	referralRewardLockPrefix = "referral:reward:"
	referralRewardLockTTL    = 30 * time.Second
)

type referralCache struct {
	rdb *redis.Client
}

func NewReferralCache(rdb *redis.Client) service.ReferralCache {
	return &referralCache{rdb: rdb}
}

func (c *referralCache) AcquireRewardLock(ctx context.Context, refereeID int64) (bool, error) {
	key := fmt.Sprintf("%s%d", referralRewardLockPrefix, refereeID)
	return c.rdb.SetNX(ctx, key, 1, referralRewardLockTTL).Result()
}

func (c *referralCache) ReleaseRewardLock(ctx context.Context, refereeID int64) error {
	key := fmt.Sprintf("%s%d", referralRewardLockPrefix, refereeID)
	return c.rdb.Del(ctx, key).Err()
}
