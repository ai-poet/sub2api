package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 运维管理员个人令牌（fork 本地功能）。任何响应都只带提示串，不带哈希；明文只出现在生成响应里一次。

// PersonalTokenInfo 令牌的展示信息。
type PersonalTokenInfo struct {
	Hint       string     `json:"hint"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	Expired    bool       `json:"expired"`
	LastUsedAt *time.Time `json:"last_used_at"`
	LastUsedIP string     `json:"last_used_ip"`
}

// PersonalTokenStatusResponse GET /api/v1/user/personal-token
type PersonalTokenStatusResponse struct {
	FeatureEnabled bool               `json:"feature_enabled"`
	Eligible       bool               `json:"eligible"`
	Token          *PersonalTokenInfo `json:"token"`
	ExpiryOptions  []int              `json:"expiry_options"`
}

// GeneratePersonalTokenRequest POST /api/v1/user/personal-token
type GeneratePersonalTokenRequest struct {
	Password string `json:"password"`
	// 0 表示永不过期；可选值见 service.PersonalTokenExpiryDays
	ExpiresInDays *int `json:"expires_in_days"`
}

// GeneratePersonalTokenResponse 生成结果：token 为明文，只返回这一次。
type GeneratePersonalTokenResponse struct {
	Token string            `json:"token"`
	Info  PersonalTokenInfo `json:"info"`
}

// AdminPersonalTokenItem 管理员列表中的一行。
type AdminPersonalTokenItem struct {
	UserID     int64      `json:"user_id"`
	UserEmail  string     `json:"user_email"`
	Username   string     `json:"username"`
	UserRole   string     `json:"user_role"`
	State      string     `json:"state"`
	Hint       string     `json:"hint"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	LastUsedIP string     `json:"last_used_ip"`
	CreatedIP  string     `json:"created_ip"`
}

// AdminPersonalTokenListResponse GET /api/v1/admin/personal-tokens
type AdminPersonalTokenListResponse struct {
	FeatureEnabled bool                     `json:"feature_enabled"`
	Items          []AdminPersonalTokenItem `json:"items"`
}

// PersonalTokenInfoFromService 投影令牌展示信息（不含哈希）。
func PersonalTokenInfoFromService(token *service.PersonalToken, now time.Time) *PersonalTokenInfo {
	if token == nil {
		return nil
	}
	return &PersonalTokenInfo{
		Hint:       token.TokenHint,
		CreatedAt:  token.CreatedAt,
		ExpiresAt:  token.ExpiresAt,
		Expired:    token.IsExpired(now),
		LastUsedAt: token.LastUsedAt,
		LastUsedIP: token.LastUsedIP,
	}
}

// AdminPersonalTokenItemFromService 投影管理员列表行（不含哈希）。
func AdminPersonalTokenItemFromService(item service.PersonalTokenOverview) AdminPersonalTokenItem {
	out := AdminPersonalTokenItem{
		UserEmail: item.UserEmail,
		Username:  item.Username,
		UserRole:  item.UserRole,
		State:     item.State,
	}
	if token := item.Token; token != nil {
		out.UserID = token.UserID
		out.Hint = token.TokenHint
		out.CreatedAt = token.CreatedAt
		out.ExpiresAt = token.ExpiresAt
		out.LastUsedAt = token.LastUsedAt
		out.LastUsedIP = token.LastUsedIP
		out.CreatedIP = token.CreatedIP
	}
	return out
}
