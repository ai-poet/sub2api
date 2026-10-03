package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"
)

// AppealService 封禁申诉会话（fork 本地功能，见 appeal.go）。
type AppealService struct {
	store        AppealSessionStore
	users        UserRepository
	settings     *SettingService
	tickets      *TicketService
	siteMessages *SiteMessageService
	now          func() time.Time
}

// NewAppealService 构造申诉服务。settings / tickets / siteMessages 可为 nil（对应功能降级）。
func NewAppealService(store AppealSessionStore, users UserRepository, settings *SettingService, tickets *TicketService, siteMessages *SiteMessageService) *AppealService {
	return &AppealService{store: store, users: users, settings: settings, tickets: tickets, siteMessages: siteMessages, now: time.Now}
}

func (s *AppealService) ready() error {
	if s == nil || s.store == nil || s.users == nil {
		return ErrAppealUnavailable
	}
	return nil
}

// AppealTokenHash 令牌在 Redis 里的键值（SHA-256 十六进制）。
func AppealTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newAppealToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return AppealTokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

// Issue 给已验证身份、且当前确实被禁用的账号签发申诉令牌。
// 不满足条件（账号不是 disabled、后端模式、服务不可用）时返回空令牌、nil 错误，
// 调用方照常返回原来的 403。
func (s *AppealService) Issue(ctx context.Context, userID int64) (string, time.Duration, error) {
	if s.ready() != nil || userID <= 0 {
		return "", 0, nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil || user.Status != StatusDisabled {
		return "", 0, nil
	}
	// 后端模式下面板只对管理端开放，申诉会话（角色固定为 user）会被 BackendModeUserGuard 拦下，干脆不签发；
	// 后端模式里被禁用的账号由管理员直接在用户管理里恢复。
	if s.settings != nil && s.settings.IsBackendModeEnabled(ctx) {
		return "", 0, nil
	}
	token, err := newAppealToken()
	if err != nil {
		return "", 0, err
	}
	record := &AppealSessionRecord{UserID: user.ID, Email: user.Email, IssuedAt: s.now()}
	if err := s.store.Save(ctx, AppealTokenHash(token), record, AppealSessionTTL); err != nil {
		return "", 0, err
	}
	return token, AppealSessionTTL, nil
}

// Authenticate 校验申诉令牌，并实时确认账号仍处于禁用状态。
func (s *AppealService) Authenticate(ctx context.Context, token string) (*AppealSession, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	token = strings.TrimSpace(token)
	if token == "" || len(token) > appealTokenMaxLen || !strings.HasPrefix(token, AppealTokenPrefix) {
		return nil, ErrAppealTokenInvalid
	}
	hash := AppealTokenHash(token)
	record, ttl, err := s.store.Load(ctx, hash)
	if err != nil {
		return nil, err
	}
	if record == nil || record.UserID <= 0 {
		return nil, ErrAppealTokenInvalid
	}
	user, err := s.users.GetByID(ctx, record.UserID)
	if err != nil || user == nil {
		_ = s.store.Revoke(ctx, hash)
		if err == nil || errors.Is(err, ErrUserNotFound) {
			return nil, ErrAppealTokenInvalid
		}
		return nil, err
	}
	switch user.Status {
	case StatusDisabled:
	case StatusActive:
		_ = s.store.RevokeUser(ctx, user.ID)
		return nil, ErrAppealAccountActive
	default:
		return nil, ErrAppealTokenInvalid
	}
	role := user.Role
	if role == "" {
		role = RoleUser
	}
	return &AppealSession{UserID: user.ID, Email: user.Email, Role: role, ExpiresAt: s.now().Add(ttl)}, nil
}

// Revoke 作废令牌（申诉页「退出」）。
func (s *AppealService) Revoke(ctx context.Context, token string) error {
	if err := s.ready(); err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.store.Revoke(ctx, AppealTokenHash(token))
}

// CreateTicket 在申诉会话里提交申诉工单。
func (s *AppealService) CreateTicket(ctx context.Context, sess *AppealSession, title, body, requestOrigin string) (*SupportTicket, error) {
	if sess == nil || sess.UserID <= 0 {
		return nil, ErrAppealTokenInvalid
	}
	if s == nil || s.tickets == nil {
		return nil, ErrTicketUnavailable
	}
	return s.tickets.CreateAppeal(ctx, TicketActor{UserID: sess.UserID, Email: sess.Email, Role: RoleUser}, title, body, requestOrigin)
}

// HasActiveAppealTicket 报告该用户是否已有未关闭的申诉工单（申诉页决定显示表单还是线程）。
func (s *AppealService) HasActiveAppealTicket(ctx context.Context, userID int64) (bool, error) {
	if s == nil || s.tickets == nil || s.tickets.repo == nil || userID <= 0 {
		return false, nil
	}
	n, err := s.tickets.repo.CountActiveByUserCategory(ctx, userID, TicketCategoryAppeal)
	return n > 0, err
}

// OnUserStatusChanged 账号从禁用恢复为启用时：作废申诉会话，并发一条「账户已恢复」站内信。
func (s *AppealService) OnUserStatusChanged(ctx context.Context, userID int64, oldStatus, newStatus string) {
	if s == nil || userID <= 0 || oldStatus != StatusDisabled || newStatus != StatusActive {
		return
	}
	if s.store != nil {
		if err := s.store.RevokeUser(ctx, userID); err != nil {
			slog.Warn("appeal.revoke_on_restore_failed", "user_id", userID, "error", err)
		}
	}
	if s.siteMessages != nil {
		err := s.siteMessages.DeliverSystemMessage(ctx, SystemSiteMessageInput{
			UserID:     userID,
			Category:   SiteMessageCategorySystem,
			Title:      appealRestoredTitle,
			Content:    appealRestoredContent,
			SourceType: SiteMessageSourceAppealRestore,
		})
		if err != nil {
			slog.Warn("appeal.restore_site_message_failed", "user_id", userID, "error", err)
		}
	}
}

const (
	appealRestoredTitle   = "账户已恢复 / Account restored"
	appealRestoredContent = "您的账户已恢复正常使用，API Key 与网站登录均已可用。请遵守平台使用规范，再次触发风控可能导致账户被重新禁用。" +
		"\n\n---\n\n" +
		"Your account has been restored. Your API keys and website sign-in work again. Please follow the platform's usage policy; further violations may disable the account again."
)
