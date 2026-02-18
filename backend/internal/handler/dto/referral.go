package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// UserReferral 推荐记录 DTO
type UserReferral struct {
	ID                       int64      `json:"id"`
	ReferrerID               int64      `json:"referrer_id"`
	RefereeID                int64      `json:"referee_id"`
	Status                   string     `json:"status"`
	ReferrerBalanceReward    float64    `json:"referrer_balance_reward"`
	ReferrerGroupID          *int64     `json:"referrer_group_id,omitempty"`
	ReferrerSubscriptionDays int        `json:"referrer_subscription_days"`
	ReferrerRewardedAt       *time.Time `json:"referrer_rewarded_at,omitempty"`
	RefereeBalanceReward     float64    `json:"referee_balance_reward"`
	RefereeGroupID           *int64     `json:"referee_group_id,omitempty"`
	RefereeSubscriptionDays  int        `json:"referee_subscription_days"`
	RefereeRewardedAt        *time.Time `json:"referee_rewarded_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`

	// 关联数据
	ReferrerEmail string `json:"referrer_email,omitempty"`
	RefereeEmail  string `json:"referee_email,omitempty"`
}

// ReferralInfo 推荐信息 DTO
type ReferralInfo struct {
	ReferralCode string           `json:"referral_code"`
	ReferralLink string           `json:"referral_link"`
	Stats        ReferralStats    `json:"stats"`
	Rewards      *ReferralSettings `json:"rewards,omitempty"`
}

// ReferralStats 推荐统计 DTO
type ReferralStats struct {
	TotalCount       int     `json:"total_count"`
	RewardedCount    int     `json:"rewarded_count"`
	PendingCount     int     `json:"pending_count"`
	TotalBalanceEarn float64 `json:"total_balance_earn"`
}

// ReferralSettings 推荐配置 DTO
type ReferralSettings struct {
	Enabled                  bool    `json:"enabled"`
	ReferrerBalanceReward    float64 `json:"referrer_balance_reward"`
	ReferrerGroupID          int64   `json:"referrer_group_id"`
	ReferrerSubscriptionDays int     `json:"referrer_subscription_days"`
	RefereeBalanceReward     float64 `json:"referee_balance_reward"`
	RefereeGroupID           int64   `json:"referee_group_id"`
	RefereeSubscriptionDays  int     `json:"referee_subscription_days"`
	MaxPerUser               int     `json:"max_per_user"`
}

// UserReferralFromService 将 service 层 UserReferral 转换为 DTO
func UserReferralFromService(ref *service.UserReferral) *UserReferral {
	if ref == nil {
		return nil
	}
	return &UserReferral{
		ID:                       ref.ID,
		ReferrerID:               ref.ReferrerID,
		RefereeID:                ref.RefereeID,
		Status:                   ref.Status,
		ReferrerBalanceReward:    ref.ReferrerBalanceReward,
		ReferrerGroupID:          ref.ReferrerGroupID,
		ReferrerSubscriptionDays: ref.ReferrerSubscriptionDays,
		ReferrerRewardedAt:       ref.ReferrerRewardedAt,
		RefereeBalanceReward:     ref.RefereeBalanceReward,
		RefereeGroupID:           ref.RefereeGroupID,
		RefereeSubscriptionDays:  ref.RefereeSubscriptionDays,
		RefereeRewardedAt:        ref.RefereeRewardedAt,
		CreatedAt:                ref.CreatedAt,
		UpdatedAt:                ref.UpdatedAt,
		ReferrerEmail:            ref.ReferrerEmail,
		RefereeEmail:             ref.RefereeEmail,
	}
}

// ReferralInfoFromService 将 service 层 ReferralInfo 转换为 DTO
func ReferralInfoFromService(info *service.ReferralInfo) *ReferralInfo {
	if info == nil {
		return nil
	}
	return &ReferralInfo{
		ReferralCode: info.ReferralCode,
		ReferralLink: info.ReferralLink,
		Stats: ReferralStats{
			TotalCount:       info.Stats.TotalCount,
			RewardedCount:    info.Stats.RewardedCount,
			PendingCount:     info.Stats.PendingCount,
			TotalBalanceEarn: info.Stats.TotalBalanceEarn,
		},
		Rewards: ReferralSettingsFromService(info.Rewards),
	}
}

// ReferralSettingsFromService 将 service 层 ReferralSettings 转换为 DTO
func ReferralSettingsFromService(s *service.ReferralSettings) *ReferralSettings {
	if s == nil {
		return nil
	}
	return &ReferralSettings{
		Enabled:                  s.Enabled,
		ReferrerBalanceReward:    s.ReferrerBalanceReward,
		ReferrerGroupID:          s.ReferrerGroupID,
		ReferrerSubscriptionDays: s.ReferrerSubscriptionDays,
		RefereeBalanceReward:     s.RefereeBalanceReward,
		RefereeGroupID:           s.RefereeGroupID,
		RefereeSubscriptionDays:  s.RefereeSubscriptionDays,
		MaxPerUser:               s.MaxPerUser,
	}
}

// ReferralSettingsToService 将 DTO 转换为 service 层 ReferralSettings
func ReferralSettingsToService(s *ReferralSettings) *service.ReferralSettings {
	if s == nil {
		return nil
	}
	return &service.ReferralSettings{
		Enabled:                  s.Enabled,
		ReferrerBalanceReward:    s.ReferrerBalanceReward,
		ReferrerGroupID:          s.ReferrerGroupID,
		ReferrerSubscriptionDays: s.ReferrerSubscriptionDays,
		RefereeBalanceReward:     s.RefereeBalanceReward,
		RefereeGroupID:           s.RefereeGroupID,
		RefereeSubscriptionDays:  s.RefereeSubscriptionDays,
		MaxPerUser:               s.MaxPerUser,
	}
}
