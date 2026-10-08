package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 多域名登录交接（fork 本地功能），配置与背景见 config.SessionHandoffConfig。
//
// 接收方（要拿到会话的域名）在本地生成 PKCE verifier，只把 S256 challenge 交给提供方；
// 提供方（已登录的域名）用当前会话申请一个绑定该 challenge 与接收方 origin 的一次性码，
// 再把浏览器带到接收方，接收方拿码 + verifier 兑换一套新的 token。
// 码只在 Redis 里以 SHA-256 作键出现、一次性（GETDEL），存储里不放 token。
// 与客户端登录码同样是"先消费再校验"：verifier 错了码也就作废了。

const (
	// SessionHandoffCodeTTL 交接码有效期：浏览器跳转一次就会用掉，给足网络抖动即可。
	SessionHandoffCodeTTL = 2 * time.Minute
	// 兜底过期判断的时钟容差（Redis TTL 是主判据）。
	sessionHandoffCodeClockSkew = 30 * time.Second
)

var (
	// ErrSessionHandoffDisabled 未配置 session_handoff。
	ErrSessionHandoffDisabled = infraerrors.NotFound("SESSION_HANDOFF_DISABLED", "session handoff is not configured")
	// ErrSessionHandoffOriginNotAllowed 目标 origin 不在 session_handoff 配置的域名里。
	ErrSessionHandoffOriginNotAllowed = infraerrors.BadRequest("SESSION_HANDOFF_ORIGIN_NOT_ALLOWED", "target_origin is not an allowed handoff origin")
	// ErrSessionHandoffChallengeInvalid code_challenge 不是 43 位 base64url，或 method 不是 S256。
	ErrSessionHandoffChallengeInvalid = infraerrors.BadRequest("SESSION_HANDOFF_CHALLENGE_INVALID", "code_challenge must be an unpadded base64url SHA-256 digest and code_challenge_method must be S256")
	// ErrSessionHandoffCodeInvalid 兑换失败的唯一对外错误：码不存在、过期、已用、verifier 不符、
	// 发起兑换的 origin 与码绑定的不一致，都返回它。
	ErrSessionHandoffCodeInvalid = infraerrors.BadRequest("SESSION_HANDOFF_CODE_INVALID", "handoff code is invalid or expired")
)

// SessionHandoffCode 一个交接码背后的记录。
type SessionHandoffCode struct {
	UserID        int64     `json:"user_id"`
	CodeChallenge string    `json:"code_challenge"`
	TargetOrigin  string    `json:"target_origin"`
	CreatedAt     time.Time `json:"created_at"`
}

// SessionHandoffCodeStore 交接码的一次性存储：Store 生成新码并写入记录，
// Consume 原子地取出并删除；不存在、过期或格式不对时返回 ErrSessionHandoffCodeInvalid。
type SessionHandoffCodeStore interface {
	Store(ctx context.Context, record *SessionHandoffCode, ttl time.Duration) (code string, err error)
	Consume(ctx context.Context, code string) (*SessionHandoffCode, error)
}

// SessionHandoffCodeIssued CreateCode 的结果。
type SessionHandoffCodeIssued struct {
	Code         string
	TargetOrigin string
	ExpiresIn    int // 秒
}

// SessionHandoffExchangeResult Exchange 的结果。
type SessionHandoffExchangeResult struct {
	TokenPair
	UserID   int64
	UserRole string
}

// SessionHandoffService 交接码的签发与兑换。
type SessionHandoffService struct {
	store  SessionHandoffCodeStore
	tokens desktopLoginTokenIssuer
	cfg    config.SessionHandoffConfig
	now    func() time.Time
}

// NewSessionHandoffService 构造交接服务；cfg 为 nil 或未配置时服务处于关闭状态。
func NewSessionHandoffService(store SessionHandoffCodeStore, authService *AuthService, cfg *config.Config) *SessionHandoffService {
	svc := &SessionHandoffService{store: store, now: time.Now}
	// 显式判空，避免 nil 指针包进非 nil 接口
	if authService != nil {
		svc.tokens = authService
	}
	if cfg != nil {
		svc.cfg = cfg.SessionHandoff
	}
	return svc
}

// sessionHandoffLoginOrigin / sessionHandoffAliasOrigins 公开设置里的交接配置；未配置时为空。
func sessionHandoffLoginOrigin(cfg *config.Config) string {
	if cfg == nil || !cfg.SessionHandoff.Enabled() {
		return ""
	}
	return cfg.SessionHandoff.LoginOrigin
}

func sessionHandoffAliasOrigins(cfg *config.Config) []string {
	if cfg == nil || !cfg.SessionHandoff.Enabled() {
		return []string{}
	}
	return append([]string(nil), cfg.SessionHandoff.AliasOrigins...)
}

// Enabled 报告交接是否已配置。
func (s *SessionHandoffService) Enabled() bool {
	return s != nil && s.cfg.Enabled()
}

// CreateCode 为已登录用户签发一个交接码，绑定接收方的 S256 challenge 与 origin。
func (s *SessionHandoffService) CreateCode(
	ctx context.Context,
	userID int64,
	codeChallenge string,
	codeChallengeMethod string,
	targetOrigin string,
) (*SessionHandoffCodeIssued, error) {
	if !s.Enabled() {
		return nil, ErrSessionHandoffDisabled
	}
	if userID <= 0 {
		return nil, ErrUserNotFound
	}
	if codeChallengeMethod != DesktopLoginChallengeMethodS256 || !isDesktopLoginChallenge(codeChallenge) {
		return nil, ErrSessionHandoffChallengeInvalid
	}
	origin, err := config.NormalizeHandoffOrigin(targetOrigin)
	if err != nil || !s.cfg.AllowsOrigin(origin) {
		return nil, ErrSessionHandoffOriginNotAllowed
	}
	if s.store == nil {
		return nil, errors.New("session handoff code store not configured")
	}
	code, err := s.store.Store(ctx, &SessionHandoffCode{
		UserID:        userID,
		CodeChallenge: codeChallenge,
		TargetOrigin:  origin,
		CreatedAt:     s.now().UTC(),
	}, SessionHandoffCodeTTL)
	if err != nil {
		return nil, err
	}
	return &SessionHandoffCodeIssued{
		Code:         code,
		TargetOrigin: origin,
		ExpiresIn:    int(SessionHandoffCodeTTL / time.Second),
	}, nil
}

// Exchange 用交接码 + PKCE verifier 换一对新的 token。
//
// requestOrigin 是兑换请求的 Origin 头：浏览器发出的 POST 都会带上它，
// 带了就必须与码绑定的接收方一致；没带时只靠 PKCE（verifier 只存在于接收方页面里）。
func (s *SessionHandoffService) Exchange(ctx context.Context, code, codeVerifier, requestOrigin string) (*SessionHandoffExchangeResult, error) {
	if !s.Enabled() {
		return nil, ErrSessionHandoffDisabled
	}
	if s.store == nil || s.tokens == nil {
		return nil, errors.New("session handoff service not configured")
	}
	record, err := s.store.Consume(ctx, code)
	if err != nil {
		if errors.Is(err, ErrSessionHandoffCodeInvalid) {
			return nil, ErrSessionHandoffCodeInvalid
		}
		return nil, err
	}
	if record == nil || record.UserID <= 0 || !isDesktopLoginChallenge(record.CodeChallenge) {
		return nil, ErrSessionHandoffCodeInvalid
	}
	if record.CreatedAt.IsZero() || s.now().Sub(record.CreatedAt) > SessionHandoffCodeTTL+sessionHandoffCodeClockSkew {
		return nil, ErrSessionHandoffCodeInvalid
	}
	// 码签发后配置可能被收紧：不再允许的 origin 一律作废。
	if !s.cfg.AllowsOrigin(record.TargetOrigin) {
		return nil, ErrSessionHandoffCodeInvalid
	}
	if requestOrigin != "" {
		origin, err := config.NormalizeHandoffOrigin(requestOrigin)
		if err != nil || origin != record.TargetOrigin {
			return nil, ErrSessionHandoffCodeInvalid
		}
	}
	if !isDesktopLoginVerifier(codeVerifier) {
		return nil, ErrSessionHandoffCodeInvalid
	}
	digest := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(digest[:])
	if subtle.ConstantTimeCompare([]byte(computed), []byte(record.CodeChallenge)) != 1 {
		return nil, ErrSessionHandoffCodeInvalid
	}

	// 每个域名一套独立的 token 家族：两个域名共用同一个 refresh token 会互相轮换掉对方。
	pair, err := s.tokens.GenerateDesktopTokenPairWithUser(ctx, record.UserID)
	if err != nil {
		// 签发码之后账号被删：对兑换方而言就是一个无效码（被禁用的账号照常返回 USER_NOT_ACTIVE）。
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrSessionHandoffCodeInvalid
		}
		return nil, err
	}
	return &SessionHandoffExchangeResult{
		TokenPair: pair.TokenPair,
		UserID:    record.UserID,
		UserRole:  pair.UserRole,
	}, nil
}
