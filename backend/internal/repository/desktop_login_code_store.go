package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// 客户端登录码的 Redis 存储（fork 本地功能）。
//
// 码是 8 个字符，取自去掉易混字符（0/O、1/I/L）的 31 字符表，展示为 XXXX-XXXX。
// Redis 里只以码的 SHA-256 作键，库里看不到码本身；写入用 SET NX + TTL，
// 取出用 GETDEL，保证一个码只能兑换一次。
const (
	desktopLoginCodePrefix   = "desktop_login_code:"
	desktopLoginCodeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	desktopLoginCodeLength   = 8
	// 输入上限：码加分隔符与空白不会超过这个长度，超过的直接判无效。
	desktopLoginCodeMaxInput = 64
	// 码空间约 8.5e11，撞上未过期码的概率可以忽略；仍按 NX 重试几次兜底。
	desktopLoginCodeStoreAttempts = 5
)

type desktopLoginCodeStore struct {
	redis *redis.Client
}

// NewDesktopLoginCodeStore 构造登录码存储。
func NewDesktopLoginCodeStore(redisClient *redis.Client) service.DesktopLoginCodeStore {
	return &desktopLoginCodeStore{redis: redisClient}
}

func (s *desktopLoginCodeStore) Store(
	ctx context.Context,
	record *service.DesktopLoginCode,
	ttl time.Duration,
) (string, error) {
	if record == nil || ttl <= 0 {
		return "", fmt.Errorf("invalid desktop login code record")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("encode desktop login code: %w", err)
	}
	for attempt := 0; attempt < desktopLoginCodeStoreAttempts; attempt++ {
		code, err := randomDesktopLoginCode()
		if err != nil {
			return "", fmt.Errorf("generate desktop login code: %w", err)
		}
		stored, err := s.redis.SetNX(ctx, desktopLoginCodeKey(code), payload, ttl).Result()
		if err != nil {
			return "", fmt.Errorf("store desktop login code: %w", err)
		}
		if stored {
			return formatDesktopLoginCode(code), nil
		}
	}
	return "", fmt.Errorf("store desktop login code: no free code after %d attempts", desktopLoginCodeStoreAttempts)
}

func (s *desktopLoginCodeStore) Consume(ctx context.Context, code string) (*service.DesktopLoginCode, error) {
	normalized, ok := normalizeDesktopLoginCode(code)
	if !ok {
		return nil, service.ErrDesktopLoginCodeInvalid
	}
	payload, err := s.redis.GetDel(ctx, desktopLoginCodeKey(normalized)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, service.ErrDesktopLoginCodeInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("consume desktop login code: %w", err)
	}
	var record service.DesktopLoginCode
	if err = json.Unmarshal(payload, &record); err != nil {
		return nil, service.ErrDesktopLoginCodeInvalid
	}
	return &record, nil
}

// desktopLoginCodeKey 键里只有规范化后码的 SHA-256。
func desktopLoginCodeKey(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return desktopLoginCodePrefix + hex.EncodeToString(sum[:])
}

// randomDesktopLoginCode 用 crypto/rand 从字符表里均匀抽 8 个字符（拒绝采样，无取模偏差）。
func randomDesktopLoginCode() (string, error) {
	const alphabetSize = len(desktopLoginCodeAlphabet)
	// 256 以内 alphabetSize 的最大整倍数；落在其上的字节丢弃重抽。
	const limit = 256 - 256%alphabetSize
	out := make([]byte, 0, desktopLoginCodeLength)
	buf := make([]byte, desktopLoginCodeLength*2)
	for len(out) < desktopLoginCodeLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if int(b) >= limit {
				continue
			}
			out = append(out, desktopLoginCodeAlphabet[int(b)%alphabetSize])
			if len(out) == desktopLoginCodeLength {
				break
			}
		}
	}
	return string(out), nil
}

// formatDesktopLoginCode 把 8 位码排成 XXXX-XXXX 给人看。
func formatDesktopLoginCode(normalized string) string {
	half := desktopLoginCodeLength / 2
	return normalized[:half] + "-" + normalized[half:]
}

// normalizeDesktopLoginCode 转大写并去掉字母数字以外的一切（空格、连字符等），
// 结果必须恰好 8 个字符且都在字符表内。只处理 ASCII，避免 Unicode 大小写折叠
// 把别的字符变成字符表里的字母。
func normalizeDesktopLoginCode(code string) (string, bool) {
	if len(code) > desktopLoginCodeMaxInput {
		return "", false
	}
	var b strings.Builder
	b.Grow(desktopLoginCodeLength)
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			_ = b.WriteByte(c)
		}
	}
	normalized := b.String()
	if len(normalized) != desktopLoginCodeLength {
		return "", false
	}
	for i := 0; i < len(normalized); i++ {
		if strings.IndexByte(desktopLoginCodeAlphabet, normalized[i]) < 0 {
			return "", false
		}
	}
	return normalized, true
}
