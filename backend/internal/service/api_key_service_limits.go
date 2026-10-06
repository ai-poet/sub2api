package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// fork：API Key 删除频率限制，补在上游 PR #7752（创建数量 / 频率限制）之上。
//
// 背景：有用户用脚本在 19 小时内创建并删除 1600 个 Key，配合"请求进行中删 Key"让扣费事务
// 回滚免单（上游 PR #7816 已让扣费不再回滚）。上游只限制创建；删除同样按用户、按固定一小时
// 窗口计数（api_key_create.max_deletes_per_user_per_hour，默认 30，0 不限制），admin /
// operator 豁免。计数走 Redis，缓存出错时与创建限流一致：不阻止用户操作。

const apiKeyDeleteCountWindow = time.Hour

var ErrAPIKeyDeleteLimited = infraerrors.TooManyRequests("API_KEY_DELETE_RATE_LIMITED", "too many api keys deleted recently, please try again later")

// APIKeyDeleteCounter 由 Redis 缓存实现；不实现时删除频率限制自动失效。
type APIKeyDeleteCounter interface {
	IncrementDeleteCount(ctx context.Context, userID int64, window time.Duration) (int64, error)
}

// checkAPIKeyDeleteLimit 删除前的频率检查；特权角色豁免（查不到用户时按普通用户处理）。
func (s *APIKeyService) checkAPIKeyDeleteLimit(ctx context.Context, userID int64) error {
	if s == nil || s.cfg == nil {
		return nil
	}
	limit := s.cfg.APIKeyCreate.MaxDeletesPerUserPerHour
	if limit <= 0 {
		return nil
	}
	if s.userRepo != nil {
		if user, err := s.userRepo.GetByID(ctx, userID); err == nil && user != nil && IsPrivilegedRole(user.Role) {
			return nil
		}
	}
	counter, ok := s.cache.(APIKeyDeleteCounter)
	if !ok {
		return nil
	}
	count, err := counter.IncrementDeleteCount(ctx, userID, apiKeyDeleteCountWindow)
	if err != nil {
		logger.LegacyPrintf("service.api_key", "[APIKeyLimits] count deletes for user %d failed: %v", userID, err)
		return nil
	}
	if count > int64(limit) {
		logger.LegacyPrintf("service.api_key", "[APIKeyLimits] user %d exceeded delete rate: %d/%d per hour", userID, count, limit)
		return ErrAPIKeyDeleteLimited
	}
	return nil
}
