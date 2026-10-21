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
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// fork：排行榜与首页公开统计（「洞察」）。设计见 CLAUDE.md「排行榜 / 首页公开统计」。

// SettingKeyInsightsConfig 是排行榜与公开统计的后台配置（JSON），三个开关同时进公开设置。
const SettingKeyInsightsConfig = "insights_config"

// 用户榜排名指标。cost 只在后台允许显示花费时生效，否则回落到 tokens。
const (
	InsightsMetricTokens   = "tokens"
	InsightsMetricRequests = "requests"
	InsightsMetricCost     = "cost"
)

// insightsDateLayout 是排行榜日期参数的格式（按服务器时区解释）。
const insightsDateLayout = "2006-01-02"

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
	// insightsMaxRangeDays 是排行榜日期范围的最长天数（含首尾），等于 usage_logs 的保留天数。
	insightsMaxRangeDays = 90
)

var (
	ErrInsightsInvalidDate   = infraerrors.BadRequest("INSIGHTS_INVALID_DATE", "start_date and end_date must use YYYY-MM-DD")
	ErrInsightsInvalidRange  = infraerrors.BadRequest("INSIGHTS_INVALID_RANGE", "end_date must not be earlier than start_date")
	ErrInsightsRangeTooLong  = infraerrors.BadRequest("INSIGHTS_RANGE_TOO_LONG", "the date range can span at most 90 days")
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
	// UsageRankingShowCost 用户榜是否显示花费并允许按花费排名，默认关闭。
	UsageRankingShowCost bool `json:"usage_ranking_show_cost"`
	// UsageRankingSortBy 用户榜排名指标：tokens / requests / cost，默认 tokens。
	// cost 只在 UsageRankingShowCost 打开时生效，否则按 tokens 排（存的值保留，重新打开花费后恢复）。
	UsageRankingSortBy string `json:"usage_ranking_sort_by"`
	// PublicStatsEnabled 首页公开统计（累计 / 今日 token、累计请求数），默认关闭。
	PublicStatsEnabled bool `json:"public_stats_enabled"`
}

// DefaultInsightsConfig 返回缺省配置：只开模型榜，用户榜和公开统计都关闭。
func DefaultInsightsConfig() InsightsConfig {
	return InsightsConfig{
		ModelRankingEnabled: true,
		UsageRankingLimit:   insightsDefaultUserLimit,
		UsageRankingSortBy:  InsightsMetricTokens,
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
	switch sortBy := strings.TrimSpace(cfg.UsageRankingSortBy); sortBy {
	case InsightsMetricRequests, InsightsMetricCost:
		cfg.UsageRankingSortBy = sortBy
	default:
		cfg.UsageRankingSortBy = InsightsMetricTokens
	}
	return cfg
}

// EffectiveUsageRankingSortBy 返回用户榜实际使用的排名指标：不显示花费时 cost 回落到 tokens。
func (cfg InsightsConfig) EffectiveUsageRankingSortBy() string {
	switch cfg.UsageRankingSortBy {
	case InsightsMetricRequests:
		return InsightsMetricRequests
	case InsightsMetricCost:
		if cfg.UsageRankingShowCost {
			return InsightsMetricCost
		}
	}
	return InsightsMetricTokens
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

// InsightsDateRange 是排行榜的日期范围：[Start, End)，日期按服务器时区解释，结束日期包含在内。
type InsightsDateRange struct {
	StartDate string
	EndDate   string
	Start     time.Time
	End       time.Time
}

// ParseInsightsDateRange 解析 start_date / end_date（YYYY-MM-DD，服务器时区，与用户侧用量统计同一种解释）。
// 两个都不传时取今天；只传一个时按单日处理。结束日期早于开始日期、或跨度超过 90 天时报错。
func ParseInsightsDateRange(startDate, endDate string, now time.Time) (InsightsDateRange, error) {
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	switch {
	case startDate == "" && endDate == "":
		startDate = timezone.StartOfDay(now).Format(insightsDateLayout)
		endDate = startDate
	case startDate == "":
		startDate = endDate
	case endDate == "":
		endDate = startDate
	}
	start, err := timezone.ParseInLocation(insightsDateLayout, startDate)
	if err != nil {
		return InsightsDateRange{}, ErrInsightsInvalidDate
	}
	endDay, err := timezone.ParseInLocation(insightsDateLayout, endDate)
	if err != nil {
		return InsightsDateRange{}, ErrInsightsInvalidDate
	}
	if endDay.Before(start) {
		return InsightsDateRange{}, ErrInsightsInvalidRange
	}
	// 按日历日计算跨度，不受夏令时 23 / 25 小时的一天影响。
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	endUTC := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, time.UTC)
	if int(endUTC.Sub(startUTC).Hours()/24)+1 > insightsMaxRangeDays {
		return InsightsDateRange{}, ErrInsightsRangeTooLong
	}
	return InsightsDateRange{
		StartDate: start.Format(insightsDateLayout),
		EndDate:   endDay.Format(insightsDateLayout),
		Start:     start,
		End:       endDay.AddDate(0, 0, 1),
	}, nil
}

const (
	// insightsOpenRangeTTL 是还没结束的范围（含今天或未来）的缓存时长。
	insightsOpenRangeTTL = 5 * time.Minute
	// insightsClosedRangeTTL 是已经结束的范围的缓存时长，这些数据不会再变。
	insightsClosedRangeTTL = time.Hour
)

// insightsRangeCacheTTL 按取数时间决定缓存时长：取数时范围还没结束就是 5 分钟，结束后才取的数据缓存 60 分钟。
// 零点前取的「今天」到了第二天仍按 5 分钟过期，不会把缺了最后几分钟的数据留一小时。
func insightsRangeCacheTTL(r InsightsDateRange) func(fetchedAt time.Time) time.Duration {
	return func(fetchedAt time.Time) time.Duration {
		if fetchedAt.Before(r.End) {
			return insightsOpenRangeTTL
		}
		return insightsClosedRangeTTL
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

// leaderboardRealName 是实名视角（管理员 / 运维管理员）的显示名：优先用户名，没有就用完整邮箱。
func leaderboardRealName(username, email string) string {
	if name := strings.TrimSpace(username); name != "" {
		return name
	}
	return strings.TrimSpace(email)
}

// leaderboardAvatarSeed 生成头像种子：HMAC 后取前 16 个十六进制字符。
// 同一用户在各日期范围得到同一个种子（稳定的化名），但从种子反推不出用户 ID。
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
