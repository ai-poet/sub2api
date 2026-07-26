package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
)

// 本文件承载 fork 自研 GitHub OAuth 所需的 service 能力。
// 单独成文件是为了不与上游 auth/setting 的重构互相覆盖。


// pendingOAuthTokenTTL is the validity period for pending OAuth tokens.
const pendingOAuthTokenTTL = 10 * time.Minute

// pendingOAuthPurpose is the purpose claim value for pending OAuth registration tokens.
const pendingOAuthPurpose = "pending_oauth_registration"

type pendingOAuthClaims struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Purpose  string `json:"purpose"`
	jwt.RegisteredClaims
}

// CreatePendingOAuthToken generates a short-lived JWT that carries the OAuth identity
// while waiting for the user to supply an invitation code.
func (s *AuthService) CreatePendingOAuthToken(email, username string) (string, error) {
	now := time.Now()
	claims := &pendingOAuthClaims{
		Email:    email,
		Username: username,
		Purpose:  pendingOAuthPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(pendingOAuthTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// VerifyPendingOAuthToken validates a pending OAuth token and returns the embedded identity.
// Returns ErrInvalidToken when the token is invalid or expired.
func (s *AuthService) VerifyPendingOAuthToken(tokenStr string) (email, username string, err error) {
	if len(tokenStr) > maxTokenLength {
		return "", "", ErrInvalidToken
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	token, parseErr := parser.ParseWithClaims(tokenStr, &pendingOAuthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if parseErr != nil {
		return "", "", ErrInvalidToken
	}
	claims, ok := token.Claims.(*pendingOAuthClaims)
	if !ok || !token.Valid {
		return "", "", ErrInvalidToken
	}
	if claims.Purpose != pendingOAuthPurpose {
		return "", "", ErrInvalidToken
	}
	return claims.Email, claims.Username, nil
}

// GetGitHubOAuthConfig 返回用于登录的"最终生效" GitHub OAuth 配置。
//
// 优先级：
// - 若对应系统设置键存在，则覆盖 config.yaml/env 的值
// - 否则回退到 config.yaml/env 的值
func (s *SettingService) GetGitHubOAuthConfig(ctx context.Context) (config.GitHubOAuthConfig, error) {
	if s == nil || s.cfg == nil {
		return config.GitHubOAuthConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}

	effective := s.cfg.GitHub

	keys := []string{
		SettingKeyGitHubOAuthEnabled,
		SettingKeyGitHubOAuthClientID,
		SettingKeyGitHubOAuthClientSecret,
		SettingKeyGitHubOAuthRedirectURL,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return config.GitHubOAuthConfig{}, fmt.Errorf("get github oauth settings: %w", err)
	}

	if raw, ok := settings[SettingKeyGitHubOAuthEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if v, ok := settings[SettingKeyGitHubOAuthClientID]; ok && strings.TrimSpace(v) != "" {
		effective.ClientID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyGitHubOAuthClientSecret]; ok && strings.TrimSpace(v) != "" {
		effective.ClientSecret = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyGitHubOAuthRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.RedirectURL = strings.TrimSpace(v)
	}

	if !effective.Enabled {
		return config.GitHubOAuthConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "github oauth login is disabled")
	}

	// 基础健壮性校验
	if strings.TrimSpace(effective.ClientID) == "" {
		return config.GitHubOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "github oauth client id not configured")
	}
	if strings.TrimSpace(effective.ClientSecret) == "" {
		return config.GitHubOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "github oauth client secret not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.GitHubOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "github oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		return config.GitHubOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "github oauth frontend redirect url not configured")
	}

	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.GitHubOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "github oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.GitHubOAuthConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "github oauth frontend redirect url invalid")
	}

	return effective, nil
}
