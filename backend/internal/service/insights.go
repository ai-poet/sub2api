package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// fork：排行榜与首页公开统计（「洞察」）。设计见 CLAUDE.md「排行榜 / 首页公开统计」。

// SettingKeyInsightsConfig 是排行榜与公开统计的后台配置（JSON），三个开关同时进公开设置。
const SettingKeyInsightsConfig = "insights_config"

// 排行榜时间窗只有固定档位，结果才能按档位缓存。
const (
	InsightsWindow24h = "24h"
	InsightsWindow7d  = "7d"
	InsightsWindow30d = "30d"
)

// 用户榜排序指标。cost 只在后台允许显示花费时可用。
const (
	InsightsMetricTokens   = "tokens"
	InsightsMetricRequests = "requests"
	InsightsMetricCost     = "cost"
)

const (
	insightsDefaultUserLimit = 20
	insightsMinUserLimit     = 3
	insightsMaxUserLimit     = 50
	// insightsMinParticipants 是用户榜的最少参与人数，人少时排名几乎等于点名，不公布。
	insightsMinParticipants = 5
	// insightsMaxModelItems 是模型榜展示的最多条目，其余合并为「其他」。
	insightsMaxModelItems = 50
	// insightsModelQueryLimit 是模型榜 SQL 取回的行数上限；过滤掉不公开的模型后再截到 insightsMaxModelItems。
	insightsModelQueryLimit = 200
)

var (
	ErrInsightsInvalidWindow = infraerrors.BadRequest("INSIGHTS_INVALID_WINDOW", "window must be one of 24h, 7d, 30d")
	ErrInsightsInvalidMetric = infraerrors.BadRequest("INSIGHTS_INVALID_METRIC", "metric must be one of tokens, requests, cost")
)

// InsightsConfig 是后台可编辑的排行榜 / 公开统计配置。
type InsightsConfig struct {
	// ModelRankingEnabled 模型热度榜（登录可见），默认开启。
	ModelRankingEnabled bool `json:"model_ranking_enabled"`
	// UsageRankingEnabled 用户用量榜（登录可见，名字脱敏），默认关闭。
	UsageRankingEnabled bool `json:"usage_ranking_enabled"`
	// UsageRankingLimit 用户榜条数，3..50。
	UsageRankingLimit int `json:"usage_ranking_limit"`
	// UsageRankingShowCost 用户榜是否显示花费并允许按花费排序，默认关闭。
	UsageRankingShowCost bool `json:"usage_ranking_show_cost"`
	// PublicStatsEnabled 首页公开统计（累计 / 今日 token、累计请求数），默认关闭。
	PublicStatsEnabled bool `json:"public_stats_enabled"`
}

// DefaultInsightsConfig 返回缺省配置：只开模型榜，用户榜和公开统计都关闭。
func DefaultInsightsConfig() InsightsConfig {
	return InsightsConfig{
		ModelRankingEnabled: true,
		UsageRankingLimit:   insightsDefaultUserLimit,
	}
}

// ParseInsightsConfig 解析设置表里的 JSON。缺失的字段取默认值；JSON 损坏时整体回落到默认值。
func ParseInsightsConfig(raw string) InsightsConfig {
	cfg := DefaultInsightsConfig()
	if strings.TrimSpace(raw) != "" {
		decoded := cfg
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			cfg = decoded
		}
	}
	return normalizeInsightsConfig(cfg)
}

func normalizeInsightsConfig(cfg InsightsConfig) InsightsConfig {
	switch {
	case cfg.UsageRankingLimit == 0:
		cfg.UsageRankingLimit = insightsDefaultUserLimit
	case cfg.UsageRankingLimit < insightsMinUserLimit:
		cfg.UsageRankingLimit = insightsMinUserLimit
	case cfg.UsageRankingLimit > insightsMaxUserLimit:
		cfg.UsageRankingLimit = insightsMaxUserLimit
	}
	return cfg
}

// GetInsightsConfig 读取配置；读不到时按默认值处理。
func (s *SettingService) GetInsightsConfig(ctx context.Context) (InsightsConfig, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultInsightsConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyInsightsConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultInsightsConfig(), nil
		}
		return DefaultInsightsConfig(), err
	}
	return ParseInsightsConfig(raw), nil
}

// UpdateInsightsConfig 保存配置并刷新公开设置缓存（三个开关随 HTML 注入下发）。
func (s *SettingService) UpdateInsightsConfig(ctx context.Context, cfg InsightsConfig) (InsightsConfig, error) {
	cfg = normalizeInsightsConfig(cfg)
	raw, err := json.Marshal(cfg)
	if err != nil {
		return cfg, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyInsightsConfig, string(raw)); err != nil {
		return cfg, err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return cfg, nil
}

// insightsWindowRange 把档位换算成 [start, end)。
func insightsWindowRange(window string, now time.Time) (time.Time, time.Time, error) {
	switch window {
	case InsightsWindow24h:
		return now.Add(-24 * time.Hour), now, nil
	case InsightsWindow7d:
		return now.Add(-7 * 24 * time.Hour), now, nil
	case InsightsWindow30d:
		return now.Add(-30 * 24 * time.Hour), now, nil
	default:
		return time.Time{}, time.Time{}, ErrInsightsInvalidWindow
	}
}

// insightsCacheTTL 是各档位的缓存时长：窗口越长，结果变化越慢，扫表越贵。
func insightsCacheTTL(window string) time.Duration {
	switch window {
	case InsightsWindow7d:
		return 15 * time.Minute
	case InsightsWindow30d:
		return time.Hour
	default:
		return 5 * time.Minute
	}
}

// maskLeaderboardName 生成用户榜显示名：优先用户名，没有就取邮箱 @ 之前的部分，再做脱敏。
// 3 个字符及以上保留首尾，2 个字符保留首字，1 个字符全遮，没有可用名字时返回空串（前端显示「用户」）。
// 按字符而不是字节处理，中文名不会被截成乱码。
func maskLeaderboardName(username, email string) string {
	name := strings.TrimSpace(username)
	if name == "" {
		local, _, _ := strings.Cut(strings.TrimSpace(email), "@")
		name = strings.TrimSpace(local)
	}
	if name == "" {
		return ""
	}
	runes := []rune(name)
	switch {
	case len(runes) >= 3:
		return string(runes[0]) + "***" + string(runes[len(runes)-1])
	case len(runes) == 2:
		return string(runes[0]) + "*"
	default:
		return "*"
	}
}

// leaderboardAvatarSeed 生成头像种子：HMAC 后取前 16 个十六进制字符。
// 同一用户在各档位得到同一个种子（稳定的化名），但从种子反推不出用户 ID。
func leaderboardAvatarSeed(key []byte, userID int64) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("leaderboard-avatar:v1:" + strconv.FormatInt(userID, 10)))
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

// deriveLeaderboardAvatarKey 从 JWT 密钥派生头像种子的专用密钥（独立标签，不与其他用途混用）。
func deriveLeaderboardAvatarKey(secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("sub2api/leaderboard-avatar-key"))
	return mac.Sum(nil)
}
