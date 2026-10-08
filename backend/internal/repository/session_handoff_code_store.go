package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// 多域名登录交接码的 Redis 存储（fork 本地功能）。
//
// 交接码只在浏览器跳转里传递、从不需要人工输入，所以直接用 32 字节随机数的 base64url
// （43 个字符）。Redis 里只以码的 SHA-256 作键；写入 SET NX + TTL，取出 GETDEL，一次性。
const (
	sessionHandoffCodePrefix      = "session_handoff_code:"
	sessionHandoffCodeBytes       = 32
	sessionHandoffCodeLength      = 43 // base64url(32 字节) 无填充
	sessionHandoffCodeStoreTrials = 3
)

type sessionHandoffCodeStore struct {
	redis *redis.Client
}

// NewSessionHandoffCodeStore 构造交接码存储。
func NewSessionHandoffCodeStore(redisClient *redis.Client) service.SessionHandoffCodeStore {
	return &sessionHandoffCodeStore{redis: redisClient}
}

func (s *sessionHandoffCodeStore) Store(ctx context.Context, record *service.SessionHandoffCode, ttl time.Duration) (string, error) {
	if record == nil || ttl <= 0 {
		return "", fmt.Errorf("invalid session handoff code record")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("encode session handoff code: %w", err)
	}
	for attempt := 0; attempt < sessionHandoffCodeStoreTrials; attempt++ {
		buf := make([]byte, sessionHandoffCodeBytes)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate session handoff code: %w", err)
		}
		code := base64.RawURLEncoding.EncodeToString(buf)
		stored, err := s.redis.SetNX(ctx, sessionHandoffCodeKey(code), payload, ttl).Result()
		if err != nil {
			return "", fmt.Errorf("store session handoff code: %w", err)
		}
		if stored {
			return code, nil
		}
	}
	return "", fmt.Errorf("store session handoff code: no free code after %d attempts", sessionHandoffCodeStoreTrials)
}

func (s *sessionHandoffCodeStore) Consume(ctx context.Context, code string) (*service.SessionHandoffCode, error) {
	if !isSessionHandoffCode(code) {
		return nil, service.ErrSessionHandoffCodeInvalid
	}
	payload, err := s.redis.GetDel(ctx, sessionHandoffCodeKey(code)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, service.ErrSessionHandoffCodeInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("consume session handoff code: %w", err)
	}
	var record service.SessionHandoffCode
	if err = json.Unmarshal(payload, &record); err != nil {
		return nil, service.ErrSessionHandoffCodeInvalid
	}
	return &record, nil
}

func sessionHandoffCodeKey(code string) string {
	sum := sha256.Sum256([]byte(code))
	return sessionHandoffCodePrefix + hex.EncodeToString(sum[:])
}

// isSessionHandoffCode 恰好 43 个 base64url 字符。
func isSessionHandoffCode(code string) bool {
	if len(code) != sessionHandoffCodeLength {
		return false
	}
	for i := 0; i < len(code); i++ {
		if !isBase64URLChar(code[i]) {
			return false
		}
	}
	return true
}

func isBase64URLChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
}
