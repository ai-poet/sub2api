package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"golang.org/x/sync/singleflight"
)

// fork：排行榜与首页公开统计的服务。
//
// 排行榜没有预聚合表可用，只能在所选日期范围内扫 usage_logs，所以结果按「榜单 + 起止日期 + 指标 + 条数」缓存：
// 新鲜期内直接返回；过期后先返回旧值、后台单飞刷新；没有旧值或旧值太老才同步查询。
// 日期范围由用户选，缓存条目数有上限（按最近使用淘汰），同时进行的查询数也有上限。
// 公开统计只读 usage_dashboard_daily，聚合作业没开或落后太多时宁可不显示，也不回退去扫明细表。

const (
	// insightsQueryTimeout 是单次排行榜查询的上限（含排队等查询名额的时间）；后台刷新与同步加载都用它。
	insightsQueryTimeout = 30 * time.Second
	// insightsMaxCacheEntries 是缓存的最多条目数，超出时淘汰最久没用过的条目。
	insightsMaxCacheEntries = 200
	// insightsMaxConcurrentLoads 是同时进行的排行榜 / 统计查询上限，不同日期范围的冷查询不会一起压到数据库上。
	insightsMaxConcurrentLoads = 4
	// insightsFailureBackoff 是刷新失败后到下次后台刷新的最短间隔。
	insightsFailureBackoff = time.Minute
	// insightsConfigTTL 是配置的内存缓存时长，公开统计接口不必每次都读设置表。
	insightsConfigTTL = 10 * time.Second
	// insightsPublicStatsTTL 是公开统计的缓存时长。
	insightsPublicStatsTTL = time.Minute
	// insightsMaxWatermarkLag 是聚合水位允许落后的最长时间，超过就不对外显示。
	insightsMaxWatermarkLag = 2 * time.Hour
)

// errInsightsCacheType 表示缓存里的值类型不对；每个 key 只存一种类型，出现即为编程错误。
var errInsightsCacheType = errors.New("insights cache: unexpected value type")

// InsightsRepository 是排行榜与公开统计的数据源。
type InsightsRepository interface {
	// ModelUsageRanking 按 token 降序返回 [start, end) 内各模型的用量，以及全部模型的总量。
	ModelUsageRanking(ctx context.Context, start, end time.Time, limit int) (*InsightsModelUsage, error)
	// UserUsageRanking 按 metric 降序返回普通、启用、未删除用户的用量前 limit 名和参与人数。
	UserUsageRanking(ctx context.Context, start, end time.Time, metric string, limit int) (*InsightsUserUsage, error)
	// PublicUsageTotals 汇总预聚合日表：全部保留天数的 token 与请求数，以及 today 当天的 token。
	PublicUsageTotals(ctx context.Context, today time.Time) (*InsightsUsageTotals, error)
	// AggregationWatermark 返回预聚合作业最后一次完成的时间。
	AggregationWatermark(ctx context.Context) (time.Time, error)
}

type InsightsModelUsageRow struct {
	Model    string
	Requests int64
	Tokens   int64
}

type InsightsModelUsage struct {
	Rows        []InsightsModelUsageRow
	TotalTokens int64
}

type InsightsUserUsageRow struct {
	UserID     int64
	Username   string
	Email      string
	Tokens     int64
	Requests   int64
	ActualCost float64
}

type InsightsUserUsage struct {
	Rows         []InsightsUserUsageRow
	Participants int64
}

type InsightsUsageTotals struct {
	TotalTokens   int64
	TotalRequests int64
	TodayTokens   int64
}

// LeaderboardModelItem 是模型榜的一行。
type LeaderboardModelItem struct {
	Rank         int     `json:"rank"`
	Model        string  `json:"model"`
	Platform     string  `json:"platform"`
	Tokens       int64   `json:"tokens"`
	Requests     int64   `json:"requests"`
	SharePercent float64 `json:"share_percent"`
}

// LeaderboardModelsResult 是模型榜接口的响应。
type LeaderboardModelsResult struct {
	Enabled bool `json:"enabled"`
	// StartDate / EndDate 是本次统计的日期范围（YYYY-MM-DD，含首尾，服务器时区）。
	StartDate   string     `json:"start_date"`
	EndDate     string     `json:"end_date"`
	GeneratedAt *time.Time `json:"generated_at,omitempty"`
	TotalTokens int64      `json:"total_tokens"`
	// OtherSharePercent 是榜单之外（含不公开的模型）的 token 占比。
	OtherSharePercent float64                `json:"other_share_percent"`
	Items             []LeaderboardModelItem `json:"items"`
}

// LeaderboardUserItem 是用户榜的一行。不含用户 ID、邮箱和原始用户名。
type LeaderboardUserItem struct {
	Rank int `json:"rank"`
	// DisplayName 是脱敏后的名字；为空时前端显示「用户」。
	DisplayName string `json:"display_name"`
	// AvatarSeed 是生成头像的化名种子，不可反推用户。
	AvatarSeed string   `json:"avatar_seed"`
	Tokens     int64    `json:"tokens"`
	Requests   int64    `json:"requests"`
	ActualCost *float64 `json:"actual_cost,omitempty"`
	IsSelf     bool     `json:"is_self"`
}

// LeaderboardUsersResult 是用户榜接口的响应。
type LeaderboardUsersResult struct {
	Enabled bool `json:"enabled"`
	// StartDate / EndDate 是本次统计的日期范围（YYYY-MM-DD，含首尾，服务器时区）。
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	// SortBy 是后台设定的排名指标（tokens / requests / cost），前端把它当主指标显示。
	SortBy   string `json:"sort_by"`
	ShowCost bool   `json:"show_cost"`
	// Limit 是后台设定的榜单条数。
	Limit       int        `json:"limit"`
	GeneratedAt *time.Time `json:"generated_at,omitempty"`
	// InsufficientData 为 true 时参与人数太少，不公布榜单。
	InsufficientData bool                  `json:"insufficient_data"`
	Items            []LeaderboardUserItem `json:"items"`
}

// PublicStatsResult 是首页公开统计接口的响应。
type PublicStatsResult struct {
	Enabled bool `json:"enabled"`
	// Available 为 false 时数据暂时拿不到（聚合作业未开启、落后或查询失败），前端隐藏计数。
	Available     bool       `json:"available"`
	TotalTokens   int64      `json:"total_tokens"`
	TodayTokens   int64      `json:"today_tokens"`
	TotalRequests int64      `json:"total_requests"`
	AsOf          *time.Time `json:"as_of,omitempty"`
}

type insightsCatalogReader interface {
	GetPublicCatalog(ctx context.Context) (*ModelCatalogResponse, error)
}

type insightsConfigReader interface {
	GetInsightsConfig(ctx context.Context) (InsightsConfig, error)
}

// InsightsService 提供模型榜、用户榜和首页公开统计。
type InsightsService struct {
	repo                InsightsRepository
	settings            insightsConfigReader
	catalog             insightsCatalogReader
	avatarKey           []byte
	dashboardAggEnabled bool
	cache               *insightsCache
	now                 func() time.Time

	configMu        sync.Mutex
	configCached    InsightsConfig
	configFetchedAt time.Time
}

// NewInsightsService 构造服务。头像种子的密钥从 JWT 密钥派生；没有配置时用进程内随机密钥。
func NewInsightsService(
	repo InsightsRepository,
	settingService *SettingService,
	catalog *ModelCatalogService,
	cfg *config.Config,
) *InsightsService {
	s := &InsightsService{
		repo:  repo,
		cache: newInsightsCache(time.Now),
		now:   time.Now,
	}
	if settingService != nil {
		s.settings = settingService
	}
	if catalog != nil {
		s.catalog = catalog
	}
	if cfg != nil {
		s.dashboardAggEnabled = cfg.DashboardAgg.Enabled
		if strings.TrimSpace(cfg.JWT.Secret) != "" {
			s.avatarKey = deriveLeaderboardAvatarKey(cfg.JWT.Secret)
		}
	}
	if len(s.avatarKey) == 0 {
		s.avatarKey = make([]byte, 32)
		_, _ = rand.Read(s.avatarKey)
	}
	return s
}

// Config 返回当前配置（10 秒内存缓存）；读取失败时按默认值处理。
func (s *InsightsService) Config(ctx context.Context) InsightsConfig {
	if s.settings == nil {
		return DefaultInsightsConfig()
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if !s.configFetchedAt.IsZero() && s.now().Sub(s.configFetchedAt) < insightsConfigTTL {
		return s.configCached
	}
	cfg, err := s.settings.GetInsightsConfig(ctx)
	if err != nil {
		// 读不到设置时用默认值（用户榜、公开统计都关闭），且不缓存，下次再试。
		return DefaultInsightsConfig()
	}
	s.configCached = cfg
	s.configFetchedAt = s.now()
	return cfg
}

// InvalidateConfig 在后台保存配置后调用，让开关立即生效。
func (s *InsightsService) InvalidateConfig() {
	s.configMu.Lock()
	s.configFetchedAt = time.Time{}
	s.configMu.Unlock()
}

// ModelRanking 返回模型榜。只列公开目录里的模型，专属分组的模型不会出现。
// startDate / endDate 是 YYYY-MM-DD（服务器时区，含首尾），都不传时取今天。
func (s *InsightsService) ModelRanking(ctx context.Context, startDate, endDate string) (*LeaderboardModelsResult, error) {
	dateRange, err := ParseInsightsDateRange(startDate, endDate, s.now())
	if err != nil {
		return nil, err
	}
	if !s.Config(ctx).ModelRankingEnabled {
		return &LeaderboardModelsResult{
			Enabled:   false,
			StartDate: dateRange.StartDate,
			EndDate:   dateRange.EndDate,
			Items:     []LeaderboardModelItem{},
		}, nil
	}
	key := "models:" + dateRange.StartDate + ":" + dateRange.EndDate
	value, err := s.cache.get(ctx, key, insightsRangeCacheTTL(dateRange), func(ctx context.Context) (any, error) {
		return s.loadModelRanking(ctx, dateRange)
	})
	if err != nil {
		return nil, err
	}
	cached, ok := value.(*LeaderboardModelsResult)
	if !ok {
		return nil, errInsightsCacheType
	}
	result := *cached
	result.Items = append([]LeaderboardModelItem(nil), cached.Items...)
	return &result, nil
}

func (s *InsightsService) loadModelRanking(ctx context.Context, dateRange InsightsDateRange) (*LeaderboardModelsResult, error) {
	usage, err := s.repo.ModelUsageRanking(ctx, dateRange.Start, dateRange.End, insightsModelQueryLimit)
	if err != nil {
		return nil, fmt.Errorf("model usage ranking: %w", err)
	}
	platforms, err := s.publicModelPlatforms(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]LeaderboardModelItem, 0, insightsMaxModelItems)
	listedShare := 0.0
	for _, row := range usage.Rows {
		if len(items) >= insightsMaxModelItems {
			break
		}
		platform, ok := platforms[strings.ToLower(strings.TrimSpace(row.Model))]
		if !ok {
			continue
		}
		share := 0.0
		if usage.TotalTokens > 0 {
			share = float64(row.Tokens) / float64(usage.TotalTokens) * 100
		}
		listedShare += share
		items = append(items, LeaderboardModelItem{
			Rank:         len(items) + 1,
			Model:        row.Model,
			Platform:     platform,
			Tokens:       row.Tokens,
			Requests:     row.Requests,
			SharePercent: share,
		})
	}

	other := 0.0
	if usage.TotalTokens > 0 && listedShare < 100 {
		other = 100 - listedShare
	}
	generatedAt := s.now()
	return &LeaderboardModelsResult{
		Enabled:           true,
		StartDate:         dateRange.StartDate,
		EndDate:           dateRange.EndDate,
		GeneratedAt:       &generatedAt,
		TotalTokens:       usage.TotalTokens,
		OtherSharePercent: other,
		Items:             items,
	}, nil
}

// publicModelPlatforms 返回公开目录里的模型（小写）→ 平台。
func (s *InsightsService) publicModelPlatforms(ctx context.Context) (map[string]string, error) {
	if s.catalog == nil {
		return map[string]string{}, nil
	}
	catalog, err := s.catalog.GetPublicCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("public catalog: %w", err)
	}
	platforms := make(map[string]string, len(catalog.Items))
	for _, item := range catalog.Items {
		key := strings.ToLower(strings.TrimSpace(item.Model))
		if _, exists := platforms[key]; !exists {
			platforms[key] = item.Platform
		}
	}
	return platforms, nil
}

type cachedUserRankingRow struct {
	userID      int64
	displayName string
	avatarSeed  string
	tokens      int64
	requests    int64
	actualCost  float64
}

type cachedUserRanking struct {
	rows         []cachedUserRankingRow
	participants int64
	generatedAt  time.Time
}

// UserRanking 返回用户榜。排名指标和条数来自后台设置（用户不能切换）；viewerID 只用来标记「你」，不影响缓存的榜单。
// startDate / endDate 是 YYYY-MM-DD（服务器时区，含首尾），都不传时取今天。
func (s *InsightsService) UserRanking(ctx context.Context, startDate, endDate string, viewerID int64) (*LeaderboardUsersResult, error) {
	dateRange, err := ParseInsightsDateRange(startDate, endDate, s.now())
	if err != nil {
		return nil, err
	}
	cfg := s.Config(ctx)
	sortBy := cfg.EffectiveUsageRankingSortBy()
	limit := cfg.UsageRankingLimit
	result := &LeaderboardUsersResult{
		Enabled:   cfg.UsageRankingEnabled,
		StartDate: dateRange.StartDate,
		EndDate:   dateRange.EndDate,
		SortBy:    sortBy,
		ShowCost:  cfg.UsageRankingShowCost,
		Limit:     limit,
		Items:     []LeaderboardUserItem{},
	}
	if !cfg.UsageRankingEnabled {
		return result, nil
	}

	key := fmt.Sprintf("users:%s:%s:%s:%d", dateRange.StartDate, dateRange.EndDate, sortBy, limit)
	value, err := s.cache.get(ctx, key, insightsRangeCacheTTL(dateRange), func(ctx context.Context) (any, error) {
		return s.loadUserRanking(ctx, dateRange, sortBy, limit)
	})
	if err != nil {
		return nil, err
	}
	cached, ok := value.(*cachedUserRanking)
	if !ok {
		return nil, errInsightsCacheType
	}

	generatedAt := cached.generatedAt
	result.GeneratedAt = &generatedAt
	if cached.participants < insightsMinParticipants {
		result.InsufficientData = true
		return result, nil
	}
	for i, row := range cached.rows {
		item := LeaderboardUserItem{
			Rank:        i + 1,
			DisplayName: row.displayName,
			AvatarSeed:  row.avatarSeed,
			Tokens:      row.tokens,
			Requests:    row.requests,
			IsSelf:      viewerID > 0 && row.userID == viewerID,
		}
		if cfg.UsageRankingShowCost {
			cost := row.actualCost
			item.ActualCost = &cost
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (s *InsightsService) loadUserRanking(ctx context.Context, dateRange InsightsDateRange, sortBy string, limit int) (*cachedUserRanking, error) {
	usage, err := s.repo.UserUsageRanking(ctx, dateRange.Start, dateRange.End, sortBy, limit)
	if err != nil {
		return nil, fmt.Errorf("user usage ranking: %w", err)
	}
	rows := make([]cachedUserRankingRow, 0, len(usage.Rows))
	for _, row := range usage.Rows {
		rows = append(rows, cachedUserRankingRow{
			userID:      row.UserID,
			displayName: maskLeaderboardName(row.Username, row.Email),
			avatarSeed:  leaderboardAvatarSeed(s.avatarKey, row.UserID),
			tokens:      row.Tokens,
			requests:    row.Requests,
			actualCost:  row.ActualCost,
		})
	}
	return &cachedUserRanking{rows: rows, participants: usage.Participants, generatedAt: s.now()}, nil
}

// PublicStats 返回首页公开统计。匿名接口用，任何错误都折算成 Available=false，从不返回错误。
func (s *InsightsService) PublicStats(ctx context.Context) *PublicStatsResult {
	if !s.Config(ctx).PublicStatsEnabled {
		return &PublicStatsResult{Enabled: false}
	}
	if !s.dashboardAggEnabled {
		return &PublicStatsResult{Enabled: true, Available: false}
	}
	value, err := s.cache.get(ctx, "public-stats", insightsFixedTTL(insightsPublicStatsTTL), func(ctx context.Context) (any, error) {
		return s.loadPublicStats(ctx)
	})
	if err != nil {
		return &PublicStatsResult{Enabled: true, Available: false}
	}
	cached, ok := value.(*PublicStatsResult)
	if !ok {
		return &PublicStatsResult{Enabled: true, Available: false}
	}
	result := *cached
	return &result
}

func (s *InsightsService) loadPublicStats(ctx context.Context) (*PublicStatsResult, error) {
	watermark, err := s.repo.AggregationWatermark(ctx)
	if err != nil {
		return nil, fmt.Errorf("aggregation watermark: %w", err)
	}
	if s.now().Sub(watermark) > insightsMaxWatermarkLag {
		// 聚合作业落后太多，数字会明显偏小，不对外显示。
		return &PublicStatsResult{Enabled: true, Available: false}, nil
	}
	totals, err := s.repo.PublicUsageTotals(ctx, timezone.Today())
	if err != nil {
		return nil, fmt.Errorf("public usage totals: %w", err)
	}
	asOf := watermark
	return &PublicStatsResult{
		Enabled:       true,
		Available:     true,
		TotalTokens:   totals.TotalTokens,
		TodayTokens:   totals.TodayTokens,
		TotalRequests: totals.TotalRequests,
		AsOf:          &asOf,
	}, nil
}

// insightsFixedTTL 返回与取数时间无关的固定缓存时长。
func insightsFixedTTL(ttl time.Duration) func(time.Time) time.Duration {
	return func(time.Time) time.Duration { return ttl }
}

// insightsCache 是按 key 缓存、过期后先返回旧值再后台刷新的小缓存。
// 条目数超过 maxEntries 时淘汰最久没用过的条目；同时进行的加载不超过 loadSlots 的容量。
type insightsCache struct {
	mu         sync.Mutex
	entries    map[string]*insightsCacheEntry
	maxEntries int
	loadSlots  chan struct{}
	group      singleflight.Group
	now        func() time.Time
	// refreshWG 让测试能等后台刷新结束。
	refreshWG sync.WaitGroup
}

type insightsCacheEntry struct {
	value      any
	fetchedAt  time.Time
	failedAt   time.Time
	lastUsed   time.Time
	refreshing bool
}

func newInsightsCache(now func() time.Time) *insightsCache {
	return &insightsCache{
		entries:    map[string]*insightsCacheEntry{},
		maxEntries: insightsMaxCacheEntries,
		loadSlots:  make(chan struct{}, insightsMaxConcurrentLoads),
		now:        now,
	}
}

// get 返回 key 的值：新鲜直接返回；过期但不超过 3×ttl 时先返回旧值并在后台刷新；
// 没有值或值太老就同步加载。同一 key 的并发加载只查一次。ttlFor 按条目的取数时间给出缓存时长。
func (c *insightsCache) get(ctx context.Context, key string, ttlFor func(fetchedAt time.Time) time.Duration, load func(context.Context) (any, error)) (any, error) {
	now := c.now()
	c.mu.Lock()
	if entry := c.entries[key]; entry != nil {
		entry.lastUsed = now
		if entry.value != nil {
			ttl := ttlFor(entry.fetchedAt)
			age := now.Sub(entry.fetchedAt)
			if age < ttl {
				value := entry.value
				c.mu.Unlock()
				return value, nil
			}
			if age < 3*ttl {
				value := entry.value
				startRefresh := !entry.refreshing && now.Sub(entry.failedAt) >= insightsFailureBackoff
				if startRefresh {
					entry.refreshing = true
					c.refreshWG.Add(1)
				}
				c.mu.Unlock()
				if startRefresh {
					go c.refresh(key, load)
				}
				return value, nil
			}
		}
	}
	c.mu.Unlock()

	// 请求取消不应中断共享的加载：其他等待者和后续缓存都依赖这次结果。
	loadCtx := context.WithoutCancel(ctx)
	value, err, _ := c.group.Do(key, func() (any, error) {
		return c.loadAndStore(loadCtx, key, load)
	})
	return value, err
}

func (c *insightsCache) refresh(key string, load func(context.Context) (any, error)) {
	defer c.refreshWG.Done()
	_, _, _ = c.group.Do(key, func() (any, error) {
		return c.loadAndStore(context.Background(), key, load)
	})
	c.mu.Lock()
	if entry := c.entries[key]; entry != nil {
		entry.refreshing = false
	}
	c.mu.Unlock()
}

func (c *insightsCache) loadAndStore(ctx context.Context, key string, load func(context.Context) (any, error)) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, insightsQueryTimeout)
	defer cancel()
	value, err := c.runLoad(ctx, load)

	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.entries[key]
	if entry == nil {
		c.evictLocked()
		entry = &insightsCacheEntry{lastUsed: c.now()}
		c.entries[key] = entry
	}
	if err != nil {
		entry.failedAt = c.now()
		return nil, err
	}
	entry.value = value
	entry.fetchedAt = c.now()
	entry.failedAt = time.Time{}
	return value, nil
}

// runLoad 占一个查询名额再加载；等名额的时间也算在 ctx 的超时里。
func (c *insightsCache) runLoad(ctx context.Context, load func(context.Context) (any, error)) (any, error) {
	if c.loadSlots != nil {
		select {
		case c.loadSlots <- struct{}{}:
			defer func() { <-c.loadSlots }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return load(ctx)
}

// evictLocked 为新条目腾位置：条目数到上限时淘汰最久没用过的条目（正在后台刷新的跳过）。调用方持有 c.mu。
func (c *insightsCache) evictLocked() {
	if c.maxEntries <= 0 {
		return
	}
	for len(c.entries) >= c.maxEntries {
		oldestKey := ""
		var oldest time.Time
		for key, entry := range c.entries {
			if entry.refreshing {
				continue
			}
			if oldestKey == "" || entry.lastUsed.Before(oldest) {
				oldestKey, oldest = key, entry.lastUsed
			}
		}
		if oldestKey == "" {
			return
		}
		delete(c.entries, oldestKey)
	}
}
