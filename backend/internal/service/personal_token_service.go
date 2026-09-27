package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// personalTokenUserReader 读取用户（需返回已归一化 TokenVersion 的实例，由 UserService.GetByID 提供）。
type personalTokenUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

// personalTokenSettingReader 读取总开关（由 SettingService 实现）。
type personalTokenSettingReader interface {
	IsPersonalTokenEnabled(ctx context.Context) bool
}

// PersonalTokenRevoker 吊销某用户的个人令牌（角色离开 operator 时由 admin_user.go 调用）。
type PersonalTokenRevoker interface {
	RevokeForUser(ctx context.Context, userID int64) error
}

// 管理员列表里单个令牌的当前状态。
const (
	PersonalTokenStateActive      = "active"
	PersonalTokenStateExpired     = "expired"
	PersonalTokenStateRevoked     = "revoked"      // 签发后改过密码 / 邮箱
	PersonalTokenStateNotEligible = "not_eligible" // 用户已不是 operator
	PersonalTokenStateUserInvalid = "user_inactive"
	PersonalTokenStateUserMissing = "user_missing"
)

// PersonalTokenService 个人令牌的生成、查询、吊销与认证（fork 本地功能）。
type PersonalTokenService struct {
	repo     PersonalTokenRepository
	users    personalTokenUserReader
	settings personalTokenSettingReader
	now      func() time.Time
}

// NewPersonalTokenService 构造个人令牌服务。
func NewPersonalTokenService(repo PersonalTokenRepository, users personalTokenUserReader, settings personalTokenSettingReader) *PersonalTokenService {
	return &PersonalTokenService{repo: repo, users: users, settings: settings, now: time.Now}
}

// PersonalTokenStatus 用户自己看到的令牌状态。
type PersonalTokenStatus struct {
	FeatureEnabled bool
	// Eligible 当前账号是否可以持有令牌（operator 且已激活）。
	Eligible bool
	Token    *PersonalToken
	Expired  bool
}

// PersonalTokenGenerateInput 生成请求。
type PersonalTokenGenerateInput struct {
	UserID        int64
	Password      string
	ExpiresInDays int
	ClientIP      string
}

// PersonalTokenIssued 生成结果：Token 是明文，只在这里出现一次。
type PersonalTokenIssued struct {
	Token  string
	Record *PersonalToken
}

// PersonalTokenOverview 管理员列表中的一行。
type PersonalTokenOverview struct {
	Token     *PersonalToken
	UserEmail string
	Username  string
	UserRole  string
	State     string
}

// Enabled 报告管理员是否打开了个人令牌功能（默认关闭）。
func (s *PersonalTokenService) Enabled(ctx context.Context) bool {
	return s != nil && s.settings != nil && s.settings.IsPersonalTokenEnabled(ctx)
}

func (s *PersonalTokenService) ready() error {
	if s == nil || s.repo == nil || s.users == nil {
		return ErrPersonalTokenUnavailable
	}
	return nil
}

// Status 返回用户自己的令牌状态（功能关闭时也返回，便于前端提示与吊销）。
func (s *PersonalTokenService) Status(ctx context.Context, userID int64) (*PersonalTokenStatus, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &PersonalTokenStatus{
		FeatureEnabled: s.Enabled(ctx),
		Eligible:       user.IsActive() && IsPersonalTokenEligibleRole(user.Role),
	}
	token, err := s.repo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, ErrPersonalTokenNotFound) {
		return nil, err
	}
	if token != nil {
		out.Token = token
		out.Expired = token.IsExpired(s.now())
	}
	return out, nil
}

// Generate 为用户生成（或覆盖）个人令牌。要求：功能已开启、operator 且已激活、当前密码正确。
func (s *PersonalTokenService) Generate(ctx context.Context, in PersonalTokenGenerateInput) (*PersonalTokenIssued, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if !s.Enabled(ctx) {
		return nil, ErrPersonalTokenDisabled
	}
	if !IsValidPersonalTokenExpiryDays(in.ExpiresInDays) {
		return nil, ErrPersonalTokenExpiryInvalid
	}
	user, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, ErrUserNotActive
	}
	if !IsPersonalTokenEligibleRole(user.Role) {
		return nil, ErrPersonalTokenNotEligible
	}
	if in.Password == "" {
		return nil, ErrPasswordRequired
	}
	if !user.CheckPassword(in.Password) {
		return nil, ErrPasswordIncorrect
	}

	raw, err := newPersonalTokenPlaintext()
	if err != nil {
		return nil, err
	}
	now := s.now()
	record := &PersonalToken{
		UserID:           user.ID,
		TokenHash:        HashPersonalToken(raw),
		TokenHint:        personalTokenHint(raw),
		UserTokenVersion: user.TokenVersion,
		CreatedIP:        truncatePersonalTokenIP(in.ClientIP),
		CreatedAt:        now,
	}
	if in.ExpiresInDays > 0 {
		expiresAt := now.Add(time.Duration(in.ExpiresInDays) * 24 * time.Hour)
		record.ExpiresAt = &expiresAt
	}
	saved, err := s.repo.Upsert(ctx, record)
	if err != nil {
		return nil, err
	}
	return &PersonalTokenIssued{Token: raw, Record: saved}, nil
}

// Revoke 吊销用户自己的令牌；没有令牌时返回 ErrPersonalTokenNotFound。
// 不要求功能开启、不要求仍是 operator：任何人都应该能删掉自己名下的凭证。
func (s *PersonalTokenService) Revoke(ctx context.Context, userID int64) error {
	if err := s.ready(); err != nil {
		return err
	}
	deleted, err := s.repo.DeleteByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrPersonalTokenNotFound
	}
	return nil
}

// RevokeForUser 幂等吊销（管理员操作 / 角色变更钩子）；没有令牌不算错误。
func (s *PersonalTokenService) RevokeForUser(ctx context.Context, userID int64) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.repo.DeleteByUserID(ctx, userID)
	return err
}

// ListAll 管理员查看全部令牌及其当前状态（只含提示串，不含哈希以外的任何凭证信息）。
func (s *PersonalTokenService) ListAll(ctx context.Context) ([]PersonalTokenOverview, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	tokens, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]PersonalTokenOverview, 0, len(tokens))
	for _, token := range tokens {
		item := PersonalTokenOverview{Token: token}
		user, userErr := s.users.GetByID(ctx, token.UserID)
		switch {
		case userErr != nil && errors.Is(userErr, ErrUserNotFound):
			item.State = PersonalTokenStateUserMissing
		case userErr != nil:
			return nil, userErr
		default:
			item.UserEmail = user.Email
			item.Username = user.Username
			item.UserRole = user.Role
			item.State = personalTokenState(token, user, now)
		}
		out = append(out, item)
	}
	return out, nil
}

// Authenticate 校验脚本带来的令牌并返回持有人。任何一步不满足都返回 401 类错误：
// 格式 → 总开关 → 哈希查找 → 过期 → 用户存在且激活 → 角色仍是 operator → 密码 / 邮箱指纹未变。
// 通过后按节流更新最后使用时间。角色门（白名单 / 审批 / 拒绝）不在这里，由中间件与 JWT 共用。
func (s *PersonalTokenService) Authenticate(ctx context.Context, raw, clientIP string) (*User, *PersonalToken, error) {
	if err := s.ready(); err != nil {
		return nil, nil, err
	}
	if !IsPersonalTokenFormat(raw) {
		return nil, nil, ErrPersonalTokenAuthInvalid
	}
	if !s.Enabled(ctx) {
		return nil, nil, ErrPersonalTokenAuthDisabled
	}
	token, err := s.repo.GetByHash(ctx, HashPersonalToken(raw))
	if err != nil {
		if errors.Is(err, ErrPersonalTokenNotFound) {
			return nil, nil, ErrPersonalTokenAuthInvalid
		}
		return nil, nil, err
	}
	now := s.now()
	if token.IsExpired(now) {
		return nil, nil, ErrPersonalTokenAuthExpired
	}
	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrPersonalTokenAuthInvalid
		}
		return nil, nil, err
	}
	switch personalTokenState(token, user, now) {
	case PersonalTokenStateActive:
	case PersonalTokenStateUserInvalid:
		return nil, nil, ErrPersonalTokenAuthUserInvalid
	case PersonalTokenStateNotEligible:
		return nil, nil, ErrPersonalTokenAuthNotEligible
	case PersonalTokenStateRevoked:
		return nil, nil, ErrPersonalTokenAuthRevoked
	default:
		return nil, nil, ErrPersonalTokenAuthInvalid
	}

	if err := s.repo.TouchLastUsed(ctx, token.ID, truncatePersonalTokenIP(clientIP), now, PersonalTokenTouchInterval); err != nil {
		// 最后使用时间只是展示信息，写失败不影响本次认证
		slog.Warn("personal token: touch last used failed", "token_id", token.ID, "error", err)
	}
	return user, token, nil
}

// personalTokenState 由令牌与其持有人推导当前状态；只有 active 才允许认证通过。
func personalTokenState(token *PersonalToken, user *User, now time.Time) string {
	switch {
	case token.IsExpired(now):
		return PersonalTokenStateExpired
	case user == nil:
		return PersonalTokenStateUserMissing
	case !user.IsActive():
		return PersonalTokenStateUserInvalid
	case !IsPersonalTokenEligibleRole(user.Role):
		return PersonalTokenStateNotEligible
	case token.UserTokenVersion != user.TokenVersion:
		return PersonalTokenStateRevoked
	default:
		return PersonalTokenStateActive
	}
}

func newPersonalTokenPlaintext() (string, error) {
	buf := make([]byte, personalTokenRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate personal token: %w", err)
	}
	return PersonalTokenPrefix + hex.EncodeToString(buf), nil
}

func truncatePersonalTokenIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if len(ip) > 64 {
		return ip[:64]
	}
	return ip
}
