package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 封禁申诉会话（fork 本地功能，见 CLAUDE.md「封禁申诉会话」）。
//
// 被禁用的账号在任意登录方式下验证身份成功后，拿到一个短期的「申诉令牌」：
//   - 不透明随机串（apl_ 前缀），Redis 里只存它的 SHA-256；固定 2 小时有效，不续期；每个用户同时只有一个；
//   - 只通过 X-Appeal-Token 请求头使用，只能访问 /api/v1/appeal/* 白名单里的路由
//     （看站内信、提交 / 查看 / 回复申诉工单），永远换不成 JWT；
//   - 每次请求都重新读库：账号恢复后立即失效（409 APPEAL_ACCOUNT_ACTIVE），账号被删除则无效。

const (
	AppealTokenPrefix = "apl_"
	AppealTokenHeader = "X-Appeal-Token"
	AppealSessionTTL  = 2 * time.Hour
	// appealTokenMaxLen 令牌长度上限（apl_ + 43 字符 base64url），超长直接判无效，不去查 Redis。
	appealTokenMaxLen = 128
)

var (
	ErrAppealTokenInvalid   = infraerrors.Unauthorized("APPEAL_TOKEN_INVALID", "appeal session is invalid or expired")
	ErrAppealAccountActive  = infraerrors.Conflict("APPEAL_ACCOUNT_ACTIVE", "account has been restored, please sign in again")
	ErrAppealUnavailable    = infraerrors.ServiceUnavailable("APPEAL_UNAVAILABLE", "appeal service is not available")
	ErrAppealRouteForbidden = infraerrors.Forbidden("APPEAL_ROUTE_FORBIDDEN", "this action is not available in an appeal session")
)

// AppealSessionRecord Redis 里保存的会话（键是令牌的 SHA-256）。
type AppealSessionRecord struct {
	UserID   int64     `json:"user_id"`
	Email    string    `json:"email"`
	IssuedAt time.Time `json:"issued_at"`
}

// AppealSession 认证通过的申诉会话。
type AppealSession struct {
	UserID    int64
	Email     string
	Role      string
	ExpiresAt time.Time
}

// AppealSessionStore 申诉会话存储（实现见 internal/repository/appeal_session_store.go）。
type AppealSessionStore interface {
	// Save 写入会话并作废该用户之前的会话（每个用户只保留一个）。
	Save(ctx context.Context, tokenHash string, record *AppealSessionRecord, ttl time.Duration) error
	// Load 读取会话；不存在、过期或已被同一用户的新会话替换时返回 (nil, nil)。剩余有效期一并返回。
	Load(ctx context.Context, tokenHash string) (*AppealSessionRecord, time.Duration, error)
	// Revoke 删除一个会话。
	Revoke(ctx context.Context, tokenHash string) error
	// RevokeUser 删除某用户的会话（账号恢复时调用）。
	RevokeUser(ctx context.Context, userID int64) error
}

// UserStatusObserver 账号状态变更的观察者（fork 本地）：管理员修改用户状态、风控中心解封时通知。
type UserStatusObserver interface {
	OnUserStatusChanged(ctx context.Context, userID int64, oldStatus, newStatus string)
}
