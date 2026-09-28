package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 客户端登录码（fork 本地功能）。
//
// 原生桌面客户端的登录走浏览器桥接页 /auth/paseo：页面在浏览器会话里签发一对
// 独立的桌面 token，再重定向到 http://127.0.0.1:<port>/callback 交给客户端。
// 访问不到 127.0.0.1 的环境（远程桌面、受限网络、浏览器与客户端不在同一台机器）
// 改用一次性登录码：页面换一个短码给用户，用户把它粘贴进客户端，客户端再拿码兑换 token。
//
// 登录码绑定一个 S256 PKCE challenge，verifier 只存在于发起登录的客户端里，
// 所以码被旁人看到也换不出 token。存储里不放 token——token 在兑换时才签发，
// 码在 Redis 里只以 SHA-256 作键出现。

const (
	// DesktopLoginCodeTTL 登录码有效期。
	DesktopLoginCodeTTL = 10 * time.Minute
	// DesktopLoginChallengeMethodS256 唯一接受的 PKCE challenge 方法。
	DesktopLoginChallengeMethodS256 = "S256"

	// base64url（无填充）编码的 SHA-256 摘要恰好 43 个字符。
	desktopLoginChallengeLength = 43
	// RFC 7636 §4.1：code_verifier 长度 43–128。
	desktopLoginVerifierMinLength = 43
	desktopLoginVerifierMaxLength = 128
	// API Key 只做存在性与长度兜底，防止把任意大块数据塞进 Redis。
	desktopLoginAPIKeyMaxLength = 512
	// 兜底过期判断的时钟容差（Redis TTL 是主判据）。
	desktopLoginCodeClockSkew = time.Minute
)

var (
	// ErrDesktopLoginCodeInvalid 兑换失败的唯一对外错误：码不存在、过期、已用、
	// verifier 不匹配或格式错误都返回它，调用方无从区分。
	ErrDesktopLoginCodeInvalid = infraerrors.BadRequest("DESKTOP_LOGIN_CODE_INVALID", "login code is invalid or expired")
	// ErrDesktopLoginChallengeInvalid code_challenge 不是 43 位 base64url，或 method 不是 S256。
	ErrDesktopLoginChallengeInvalid = infraerrors.BadRequest("DESKTOP_LOGIN_CHALLENGE_INVALID", "code_challenge must be an unpadded base64url SHA-256 digest and code_challenge_method must be S256")
	// ErrDesktopLoginAPIKeyInvalid api_key 为空，或任一 key 超长。
	ErrDesktopLoginAPIKeyInvalid = infraerrors.BadRequest("DESKTOP_LOGIN_API_KEY_INVALID", "api_key is required")
)

// DesktopLoginCode 一个登录码背后的记录。只存兑换所需的东西，不存 token。
type DesktopLoginCode struct {
	UserID        int64     `json:"user_id"`
	CodeChallenge string    `json:"code_challenge"`
	APIKey        string    `json:"api_key"`
	ClaudeAPIKey  string    `json:"claude_api_key,omitempty"`
	CodexAPIKey   string    `json:"codex_api_key,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// DesktopLoginCodeStore 登录码的一次性存储。
//
// Store 生成一个新码并写入记录，返回给用户看的格式化码。
// Consume 原子地取出并删除记录；码不存在、已过期或格式不对时返回 ErrDesktopLoginCodeInvalid。
type DesktopLoginCodeStore interface {
	Store(ctx context.Context, record *DesktopLoginCode, ttl time.Duration) (code string, err error)
	Consume(ctx context.Context, code string) (*DesktopLoginCode, error)
}

// DesktopLoginKeys 页面随登录码一起交给客户端的网关 API Key。
type DesktopLoginKeys struct {
	APIKey       string
	ClaudeAPIKey string
	CodexAPIKey  string
}

// DesktopLoginCodeIssued CreateCode 的结果。
type DesktopLoginCodeIssued struct {
	Code      string
	ExpiresIn int // 秒
}

// DesktopLoginExchangeResult Exchange 的结果：新签发的 token 对、所属用户与页面留下的 key。
type DesktopLoginExchangeResult struct {
	TokenPair
	UserID   int64
	UserRole string
	Keys     DesktopLoginKeys
}

// desktopLoginTokenIssuer 兑换时签发桌面 token 对的能力，由 AuthService 提供。
type desktopLoginTokenIssuer interface {
	GenerateDesktopTokenPairWithUser(ctx context.Context, userID int64) (*TokenPairWithUser, error)
}

// DesktopLoginService 客户端登录码的签发与兑换。
// 独立于 AuthService，保持上游 AuthService 构造函数不变。
type DesktopLoginService struct {
	store  DesktopLoginCodeStore
	tokens desktopLoginTokenIssuer
	now    func() time.Time
}

// NewDesktopLoginService 构造登录码服务。
func NewDesktopLoginService(store DesktopLoginCodeStore, authService *AuthService) *DesktopLoginService {
	svc := &DesktopLoginService{store: store, now: time.Now}
	// 显式判空，避免 nil 指针包进非 nil 接口
	if authService != nil {
		svc.tokens = authService
	}
	return svc
}

// CreateCode 为已登录用户签发一个登录码，绑定客户端给出的 S256 challenge。
func (s *DesktopLoginService) CreateCode(
	ctx context.Context,
	userID int64,
	codeChallenge string,
	codeChallengeMethod string,
	keys DesktopLoginKeys,
) (*DesktopLoginCodeIssued, error) {
	if userID <= 0 {
		return nil, ErrUserNotFound
	}
	if codeChallengeMethod != DesktopLoginChallengeMethodS256 || !isDesktopLoginChallenge(codeChallenge) {
		return nil, ErrDesktopLoginChallengeInvalid
	}
	normalized, err := normalizeDesktopLoginKeys(keys)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		return nil, errors.New("desktop login code store not configured")
	}

	record := &DesktopLoginCode{
		UserID:        userID,
		CodeChallenge: codeChallenge,
		APIKey:        normalized.APIKey,
		ClaudeAPIKey:  normalized.ClaudeAPIKey,
		CodexAPIKey:   normalized.CodexAPIKey,
		CreatedAt:     s.now().UTC(),
	}
	code, err := s.store.Store(ctx, record, DesktopLoginCodeTTL)
	if err != nil {
		return nil, err
	}
	return &DesktopLoginCodeIssued{
		Code:      code,
		ExpiresIn: int(DesktopLoginCodeTTL / time.Second),
	}, nil
}

// Exchange 用登录码 + PKCE verifier 换一对新的桌面 token 与页面留下的 API Key。
//
// 先消费（GetDel）再校验 verifier：verifier 错了码也就作废了，
// 于是拿到码的人没有机会逐个猜 verifier。所有与码相关的失败一律返回
// ErrDesktopLoginCodeInvalid。
func (s *DesktopLoginService) Exchange(ctx context.Context, code string, codeVerifier string) (*DesktopLoginExchangeResult, error) {
	if s.store == nil || s.tokens == nil {
		return nil, errors.New("desktop login service not configured")
	}
	record, err := s.store.Consume(ctx, code)
	if err != nil {
		if errors.Is(err, ErrDesktopLoginCodeInvalid) {
			return nil, ErrDesktopLoginCodeInvalid
		}
		return nil, err
	}
	if record == nil || record.UserID <= 0 || !isDesktopLoginChallenge(record.CodeChallenge) {
		return nil, ErrDesktopLoginCodeInvalid
	}
	// Redis TTL 是主判据；这里兜底，防止 TTL 丢失的记录长期可用。
	if record.CreatedAt.IsZero() || s.now().Sub(record.CreatedAt) > DesktopLoginCodeTTL+desktopLoginCodeClockSkew {
		return nil, ErrDesktopLoginCodeInvalid
	}
	if !isDesktopLoginVerifier(codeVerifier) {
		return nil, ErrDesktopLoginCodeInvalid
	}
	digest := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(digest[:])
	if subtle.ConstantTimeCompare([]byte(computed), []byte(record.CodeChallenge)) != 1 {
		return nil, ErrDesktopLoginCodeInvalid
	}

	pair, err := s.tokens.GenerateDesktopTokenPairWithUser(ctx, record.UserID)
	if err != nil {
		// 签发码之后账号被删：对兑换方而言就是一个无效码。
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrDesktopLoginCodeInvalid
		}
		return nil, err
	}
	return &DesktopLoginExchangeResult{
		TokenPair: pair.TokenPair,
		UserID:    record.UserID,
		UserRole:  pair.UserRole,
		Keys: DesktopLoginKeys{
			APIKey:       record.APIKey,
			ClaudeAPIKey: record.ClaudeAPIKey,
			CodexAPIKey:  record.CodexAPIKey,
		},
	}, nil
}

func normalizeDesktopLoginKeys(keys DesktopLoginKeys) (DesktopLoginKeys, error) {
	out := DesktopLoginKeys{
		APIKey:       strings.TrimSpace(keys.APIKey),
		ClaudeAPIKey: strings.TrimSpace(keys.ClaudeAPIKey),
		CodexAPIKey:  strings.TrimSpace(keys.CodexAPIKey),
	}
	if out.APIKey == "" {
		return DesktopLoginKeys{}, ErrDesktopLoginAPIKeyInvalid
	}
	for _, key := range []string{out.APIKey, out.ClaudeAPIKey, out.CodexAPIKey} {
		if len(key) > desktopLoginAPIKeyMaxLength {
			return DesktopLoginKeys{}, ErrDesktopLoginAPIKeyInvalid
		}
	}
	return out, nil
}

// isDesktopLoginChallenge 43 个 base64url 字符（[A-Za-z0-9_-]），即无填充的 SHA-256 摘要。
func isDesktopLoginChallenge(challenge string) bool {
	if len(challenge) != desktopLoginChallengeLength {
		return false
	}
	for i := 0; i < len(challenge); i++ {
		c := challenge[i]
		if !isASCIIAlnum(c) && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

// isDesktopLoginVerifier RFC 7636 §4.1：43–128 个 unreserved 字符 [A-Za-z0-9-._~]。
func isDesktopLoginVerifier(verifier string) bool {
	if len(verifier) < desktopLoginVerifierMinLength || len(verifier) > desktopLoginVerifierMaxLength {
		return false
	}
	for i := 0; i < len(verifier); i++ {
		c := verifier[i]
		if !isASCIIAlnum(c) && c != '-' && c != '.' && c != '_' && c != '~' {
			return false
		}
	}
	return true
}

func isASCIIAlnum(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}
