package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 运维管理员个人令牌（fork 本地功能，见 docs/PERSONAL_TOKENS.md）。
//
// operator 在个人资料页自助生成一个令牌，脚本用 Authorization: Bearer pat-... 调管理 API。
// 令牌只替代「登录」这一步：认证通过后与 operator 的 JWT 会话走同一个角色门
// （middleware.authorizeConsoleUser / console_scope.go），可达面完全一致，不放大任何权限。
// 这里只放领域类型、仓储端口、格式工具与错误；流程见 personal_token_service.go。

const (
	// PersonalTokenPrefix 令牌前缀；JWT 以 "eyJ" 开头，不会撞前缀。
	PersonalTokenPrefix = "pat-"
	// personalTokenRandomBytes 随机部分字节数（hex 后 64 个字符）。
	personalTokenRandomBytes = 32
	// PersonalTokenTouchInterval 最后使用时间的最小写入间隔，避免每个请求都写库。
	PersonalTokenTouchInterval = time.Minute
)

// PersonalTokenExpiryDays 生成时允许的有效期（天）；0 表示永不过期。
var PersonalTokenExpiryDays = []int{0, 30, 90, 180, 365}

// personalTokenEligibleRoles 可以持有个人令牌的角色。以后要放开给 admin 只改这一处；
// admin 已有全局 Admin API Key，这里刻意只放 operator。
var personalTokenEligibleRoles = map[string]struct{}{
	RoleOperator: {},
}

// IsPersonalTokenEligibleRole 报告 role 是否可以持有 / 使用个人令牌。
func IsPersonalTokenEligibleRole(role string) bool {
	_, ok := personalTokenEligibleRoles[role]
	return ok
}

// IsValidPersonalTokenExpiryDays 报告 days 是否为允许的有效期。
func IsValidPersonalTokenExpiryDays(days int) bool {
	for _, d := range PersonalTokenExpiryDays {
		if d == days {
			return true
		}
	}
	return false
}

var (
	// 生成 / 管理侧（用户会话调用）
	ErrPersonalTokenDisabled      = infraerrors.Forbidden("PERSONAL_TOKEN_DISABLED", "personal tokens are disabled by the administrator")
	ErrPersonalTokenNotEligible   = infraerrors.Forbidden("PERSONAL_TOKEN_NOT_ELIGIBLE", "only operators can use personal tokens")
	ErrPersonalTokenNotFound      = infraerrors.NotFound("PERSONAL_TOKEN_NOT_FOUND", "personal token not found")
	ErrPersonalTokenExpiryInvalid = infraerrors.BadRequest("PERSONAL_TOKEN_EXPIRY_INVALID", "expires_in_days must be one of 0, 30, 90, 180, 365")
	ErrPersonalTokenUnavailable   = infraerrors.ServiceUnavailable("PERSONAL_TOKEN_UNAVAILABLE", "personal token service is not available")

	// 认证侧（脚本调用管理 API）：一律 401，文案与 JWT 分支对齐，不泄露令牌是否存在
	ErrPersonalTokenAuthInvalid     = infraerrors.Unauthorized("INVALID_TOKEN", "Invalid token")
	ErrPersonalTokenAuthExpired     = infraerrors.Unauthorized("TOKEN_EXPIRED", "Token has expired")
	ErrPersonalTokenAuthRevoked     = infraerrors.Unauthorized("TOKEN_REVOKED", "Token has been revoked (password or email changed)")
	ErrPersonalTokenAuthDisabled    = infraerrors.Unauthorized("PERSONAL_TOKEN_DISABLED", "Personal tokens are disabled by the administrator")
	ErrPersonalTokenAuthNotEligible = infraerrors.Unauthorized("PERSONAL_TOKEN_NOT_ELIGIBLE", "Only operators can use personal tokens")
	ErrPersonalTokenAuthUserInvalid = infraerrors.Unauthorized("USER_INACTIVE", "User account is not active")
)

// PersonalToken 一条个人令牌记录。明文从不落库，只有 TokenHash。
type PersonalToken struct {
	ID               int64
	UserID           int64
	TokenHash        string
	TokenHint        string
	UserTokenVersion int64
	ExpiresAt        *time.Time
	LastUsedAt       *time.Time
	LastUsedIP       string
	CreatedIP        string
	CreatedAt        time.Time
}

// IsExpired 报告令牌在 now 时刻是否已过期；ExpiresAt 为空表示永不过期。
func (t *PersonalToken) IsExpired(now time.Time) bool {
	return t != nil && t.ExpiresAt != nil && !now.Before(*t.ExpiresAt)
}

// PersonalTokenRepository 个人令牌持久化端口（原生 SQL 实现见 repository/personal_token_repo.go）。
type PersonalTokenRepository interface {
	// Upsert 按 user_id 插入或整行覆盖（每人至多一个），返回落库后的记录。
	Upsert(ctx context.Context, token *PersonalToken) (*PersonalToken, error)
	// GetByHash 按哈希查找；不存在返回 ErrPersonalTokenNotFound。
	GetByHash(ctx context.Context, tokenHash string) (*PersonalToken, error)
	// GetByUserID 查找某用户的令牌；不存在返回 ErrPersonalTokenNotFound。
	GetByUserID(ctx context.Context, userID int64) (*PersonalToken, error)
	// DeleteByUserID 删除某用户的令牌，返回是否删到了记录。
	DeleteByUserID(ctx context.Context, userID int64) (bool, error)
	// List 全部令牌（按创建时间倒序）。
	List(ctx context.Context) ([]*PersonalToken, error)
	// TouchLastUsed 更新最后使用时间 / IP；距上次写入不足 minInterval 时不写。
	TouchLastUsed(ctx context.Context, id int64, ip string, now time.Time, minInterval time.Duration) error
}

// IsPersonalTokenFormat 报告 raw 是否形如 pat-<64 位小写 hex>。
// 只做格式判断，不代表令牌有效；adminAuth 用它决定走令牌分支还是 JWT 分支。
func IsPersonalTokenFormat(raw string) bool {
	if !strings.HasPrefix(raw, PersonalTokenPrefix) {
		return false
	}
	body := raw[len(PersonalTokenPrefix):]
	if len(body) != personalTokenRandomBytes*2 {
		return false
	}
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

// HashPersonalToken 返回令牌明文的 SHA-256（hex），即库里存的 token_hash。
func HashPersonalToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// personalTokenHint 列表展示用的提示串：pat-1a2b3c...9f3c。
func personalTokenHint(raw string) string {
	if len(raw) <= len(PersonalTokenPrefix)+10 {
		return PersonalTokenPrefix + "..."
	}
	return raw[:len(PersonalTokenPrefix)+6] + "..." + raw[len(raw)-4:]
}
