package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// 封禁申诉会话的 Redis 存储（fork 本地功能，见 service/appeal.go）。
//
// appeal_session:<sha256(token)> → 会话 JSON；appeal_session_user:<uid> → 当前有效会话的 sha256。
// Redis 里没有令牌原文。每个用户只保留一个会话：新签发时删掉旧的，Load 也要求用户指针指向自己，
// 所以旧令牌在新会话签发后立即失效。
const (
	appealSessionKeyPrefix     = "appeal_session:"
	appealSessionUserKeyPrefix = "appeal_session_user:"
)

type appealSessionStore struct {
	redis *redis.Client
}

// NewAppealSessionStore 构造申诉会话存储。
func NewAppealSessionStore(redisClient *redis.Client) service.AppealSessionStore {
	return &appealSessionStore{redis: redisClient}
}

func appealSessionKey(hash string) string {
	return appealSessionKeyPrefix + hash
}

func appealSessionUserKey(userID int64) string {
	return appealSessionUserKeyPrefix + strconv.FormatInt(userID, 10)
}

func (s *appealSessionStore) Save(ctx context.Context, tokenHash string, record *service.AppealSessionRecord, ttl time.Duration) error {
	if record == nil || record.UserID <= 0 || tokenHash == "" || ttl <= 0 {
		return fmt.Errorf("invalid appeal session record")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode appeal session: %w", err)
	}
	userKey := appealSessionUserKey(record.UserID)
	previous, err := s.redis.Get(ctx, userKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	_, err = s.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		if previous != "" && previous != tokenHash {
			pipe.Del(ctx, appealSessionKey(previous))
		}
		pipe.Set(ctx, appealSessionKey(tokenHash), payload, ttl)
		pipe.Set(ctx, userKey, tokenHash, ttl)
		return nil
	})
	return err
}

func (s *appealSessionStore) Load(ctx context.Context, tokenHash string) (*service.AppealSessionRecord, time.Duration, error) {
	if tokenHash == "" {
		return nil, 0, nil
	}
	key := appealSessionKey(tokenHash)
	raw, err := s.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var record service.AppealSessionRecord
	if err := json.Unmarshal(raw, &record); err != nil || record.UserID <= 0 {
		return nil, 0, nil
	}
	current, err := s.redis.Get(ctx, appealSessionUserKey(record.UserID)).Result()
	if errors.Is(err, redis.Nil) || (err == nil && current != tokenHash) {
		// 已被同一用户的新会话替换，或用户的会话已被整体吊销
		_ = s.redis.Del(ctx, key).Err()
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	ttl, err := s.redis.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = 0
	}
	return &record, ttl, nil
}

func (s *appealSessionStore) Revoke(ctx context.Context, tokenHash string) error {
	if tokenHash == "" {
		return nil
	}
	key := appealSessionKey(tokenHash)
	raw, err := s.redis.Get(ctx, key).Bytes()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	keys := []string{key}
	var record service.AppealSessionRecord
	if len(raw) > 0 && json.Unmarshal(raw, &record) == nil && record.UserID > 0 {
		userKey := appealSessionUserKey(record.UserID)
		if current, err := s.redis.Get(ctx, userKey).Result(); err == nil && current == tokenHash {
			keys = append(keys, userKey)
		}
	}
	return s.redis.Del(ctx, keys...).Err()
}

func (s *appealSessionStore) RevokeUser(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return nil
	}
	userKey := appealSessionUserKey(userID)
	current, err := s.redis.Get(ctx, userKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	keys := []string{userKey}
	if current != "" {
		keys = append(keys, appealSessionKey(current))
	}
	return s.redis.Del(ctx, keys...).Err()
}
