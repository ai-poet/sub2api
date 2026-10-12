package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

// ─── 配置 ───

func TestParseInsightsConfig_DefaultsAndClamping(t *testing.T) {
	def := ParseInsightsConfig("")
	require.True(t, def.ModelRankingEnabled)
	require.False(t, def.UsageRankingEnabled)
	require.False(t, def.UsageRankingShowCost)
	require.False(t, def.PublicStatsEnabled)
	require.Equal(t, 20, def.UsageRankingLimit)
	require.Equal(t, InsightsMetricTokens, def.UsageRankingSortBy)

	// 损坏的 JSON 整体回落到默认值，不会半解析出奇怪的开关
	require.Equal(t, def, ParseInsightsConfig(`{"usage_ranking_enabled": true, broken`))

	// 缺失的字段保留默认值，显式 false 生效
	cfg := ParseInsightsConfig(`{"model_ranking_enabled": false, "usage_ranking_limit": 500}`)
	require.False(t, cfg.ModelRankingEnabled)
	require.Equal(t, 50, cfg.UsageRankingLimit)
	require.Equal(t, InsightsMetricTokens, cfg.UsageRankingSortBy, "configs saved before sort_by existed rank by tokens")
	require.Equal(t, 3, ParseInsightsConfig(`{"usage_ranking_limit": 1}`).UsageRankingLimit)

	// 排名指标只认白名单里的值
	require.Equal(t, InsightsMetricRequests, ParseInsightsConfig(`{"usage_ranking_sort_by": "requests"}`).UsageRankingSortBy)
	require.Equal(t, InsightsMetricCost, ParseInsightsConfig(`{"usage_ranking_sort_by": "cost"}`).UsageRankingSortBy)
	require.Equal(t, InsightsMetricTokens, ParseInsightsConfig(`{"usage_ranking_sort_by": "email"}`).UsageRankingSortBy)
}

func TestInsightsConfig_EffectiveSortByNeedsShowCostForCost(t *testing.T) {
	require.Equal(t, InsightsMetricTokens, InsightsConfig{UsageRankingSortBy: InsightsMetricCost}.EffectiveUsageRankingSortBy(),
		"spend is hidden, so it cannot decide the ranking")
	require.Equal(t, InsightsMetricCost, InsightsConfig{UsageRankingSortBy: InsightsMetricCost, UsageRankingShowCost: true}.EffectiveUsageRankingSortBy())
	require.Equal(t, InsightsMetricRequests, InsightsConfig{UsageRankingSortBy: InsightsMetricRequests}.EffectiveUsageRankingSortBy())
	require.Equal(t, InsightsMetricTokens, InsightsConfig{}.EffectiveUsageRankingSortBy())
}

type insightsSettingRepo struct {
	SettingRepository
	mu   sync.Mutex
	data map[string]string
}

func (r *insightsSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.data[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}

func (r *insightsSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = value
	return nil
}

func TestUpdateInsightsConfig_PersistsAndRefreshesPublicSettings(t *testing.T) {
	repo := &insightsSettingRepo{data: map[string]string{}}
	svc := &SettingService{settingRepo: repo}
	refreshed := 0
	svc.SetOnUpdateCallback(func() { refreshed++ })

	missing, err := svc.GetInsightsConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, DefaultInsightsConfig(), missing)

	saved, err := svc.UpdateInsightsConfig(context.Background(), InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 99, UsageRankingSortBy: InsightsMetricCost})
	require.NoError(t, err)
	require.Equal(t, 50, saved.UsageRankingLimit)
	// 不显示花费时也保留管理员选的 cost，重新打开花费后恢复按花费排名
	require.Equal(t, InsightsMetricCost, saved.UsageRankingSortBy)
	// 注入 HTML 的公开设置缓存必须刷新，否则开关要等缓存过期才生效
	require.Equal(t, 1, refreshed)

	loaded, err := svc.GetInsightsConfig(context.Background())
	require.NoError(t, err)
	require.True(t, loaded.UsageRankingEnabled)
	require.False(t, loaded.ModelRankingEnabled)
	require.Equal(t, InsightsMetricCost, loaded.UsageRankingSortBy)
}

// ─── 日期范围 ───

func TestParseInsightsDateRange(t *testing.T) {
	loc := timezone.Location()
	now := time.Date(2026, 10, 12, 15, 30, 0, 0, loc)

	// 不传时取服务器时区的今天，结束边界是次日零点
	today, err := ParseInsightsDateRange("", "", now)
	require.NoError(t, err)
	require.Equal(t, "2026-10-12", today.StartDate)
	require.Equal(t, "2026-10-12", today.EndDate)
	require.True(t, today.Start.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, loc)))
	require.True(t, today.End.Equal(time.Date(2026, 10, 13, 0, 0, 0, 0, loc)))

	// 结束日期包含在内
	week, err := ParseInsightsDateRange(" 2026-10-06 ", "2026-10-12", now)
	require.NoError(t, err)
	require.Equal(t, "2026-10-06", week.StartDate)
	require.True(t, week.End.Equal(time.Date(2026, 10, 13, 0, 0, 0, 0, loc)))

	// 只传一个日期按单日处理
	single, err := ParseInsightsDateRange("2026-09-01", "", now)
	require.NoError(t, err)
	require.Equal(t, "2026-09-01", single.EndDate)
	single, err = ParseInsightsDateRange("", "2026-09-02", now)
	require.NoError(t, err)
	require.Equal(t, "2026-09-02", single.StartDate)

	// 90 天（含首尾）是上限
	_, err = ParseInsightsDateRange("2026-07-15", "2026-10-12", now)
	require.NoError(t, err)
	_, err = ParseInsightsDateRange("2026-07-14", "2026-10-12", now)
	require.ErrorIs(t, err, ErrInsightsRangeTooLong)

	_, err = ParseInsightsDateRange("2026-10-12", "2026-10-11", now)
	require.ErrorIs(t, err, ErrInsightsInvalidRange)
	for _, bad := range []string{"2026-13-01", "2026/10/01", "2026-10-1", "today", "2026-10-01T00:00:00"} {
		_, err = ParseInsightsDateRange(bad, "2026-10-12", now)
		require.ErrorIs(t, err, ErrInsightsInvalidDate, bad)
	}
}

func TestInsightsRangeCacheTTL_OpenVersusClosedRanges(t *testing.T) {
	loc := timezone.Location()
	now := time.Date(2026, 10, 12, 15, 0, 0, 0, loc)
	today, err := ParseInsightsDateRange("", "", now)
	require.NoError(t, err)
	yesterday, err := ParseInsightsDateRange("2026-10-11", "2026-10-11", now)
	require.NoError(t, err)

	require.Equal(t, 5*time.Minute, insightsRangeCacheTTL(today)(now))
	require.Equal(t, time.Hour, insightsRangeCacheTTL(yesterday)(now))
	// 昨天 23:58 取的数据范围还没结束，到今天也按 5 分钟过期
	require.Equal(t, 5*time.Minute, insightsRangeCacheTTL(yesterday)(time.Date(2026, 10, 11, 23, 58, 0, 0, loc)))
}

// ─── 隐私 ───

func TestMaskLeaderboardName(t *testing.T) {
	cases := []struct {
		username, email, want string
	}{
		{"alice", "alice@example.com", "a***e"},
		{"", "bob.smith@example.com", "b***h"},
		{"张三丰", "", "张***丰"},
		{"李四", "", "李*"},
		{"x", "", "*"},
		{"  ", "  ", ""},
		{"", "@example.com", ""},
	}
	for _, tc := range cases {
		got := maskLeaderboardName(tc.username, tc.email)
		require.Equal(t, tc.want, got, "%q/%q", tc.username, tc.email)
		require.NotContains(t, got, "@", "domain must never leak")
	}
}

func TestLeaderboardAvatarSeed_StablePseudonym(t *testing.T) {
	key := deriveLeaderboardAvatarKey("secret-a")
	seed := leaderboardAvatarSeed(key, 42)
	require.Len(t, seed, 16)
	require.Equal(t, seed, leaderboardAvatarSeed(key, 42))
	require.NotEqual(t, seed, leaderboardAvatarSeed(key, 43))
	require.NotEqual(t, seed, leaderboardAvatarSeed(deriveLeaderboardAvatarKey("secret-b"), 42))
	require.NotContains(t, seed, "42")
}

// ─── 服务 ───

type fakeInsightsRepo struct {
	mu            sync.Mutex
	modelUsage    *InsightsModelUsage
	userUsage     *InsightsUserUsage
	totals        *InsightsUsageTotals
	watermark     time.Time
	err           error
	modelCalls    int
	userCalls     int
	totalsCalls   int
	lastUserLimit int
	lastMetric    string
	lastStart     time.Time
	lastEnd       time.Time
}

func (r *fakeInsightsRepo) ModelUsageRanking(_ context.Context, start, end time.Time, _ int) (*InsightsModelUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modelCalls++
	r.lastStart, r.lastEnd = start, end
	if r.err != nil {
		return nil, r.err
	}
	return r.modelUsage, nil
}

func (r *fakeInsightsRepo) UserUsageRanking(_ context.Context, start, end time.Time, metric string, limit int) (*InsightsUserUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userCalls++
	r.lastMetric = metric
	r.lastUserLimit = limit
	r.lastStart, r.lastEnd = start, end
	if r.err != nil {
		return nil, r.err
	}
	return r.userUsage, nil
}

func (r *fakeInsightsRepo) PublicUsageTotals(context.Context, time.Time) (*InsightsUsageTotals, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalsCalls++
	if r.err != nil {
		return nil, r.err
	}
	return r.totals, nil
}

func (r *fakeInsightsRepo) AggregationWatermark(context.Context) (time.Time, error) {
	return r.watermark, nil
}

func (r *fakeInsightsRepo) calls() (models, users int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.modelCalls, r.userCalls
}

type fakeInsightsCatalog struct {
	models map[string]string
}

func (c *fakeInsightsCatalog) GetPublicCatalog(context.Context) (*ModelCatalogResponse, error) {
	items := make([]ModelCatalogItem, 0, len(c.models))
	for model, platform := range c.models {
		items = append(items, ModelCatalogItem{Model: model, Platform: platform})
	}
	return &ModelCatalogResponse{Items: items}, nil
}

type fakeInsightsConfig struct {
	cfg InsightsConfig
}

func (f *fakeInsightsConfig) GetInsightsConfig(context.Context) (InsightsConfig, error) {
	return f.cfg, nil
}

type insightsClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *insightsClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *insightsClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// newTestInsightsService 的时钟停在服务器时区 2026-10-12 12:00，「今天」就是 2026-10-12。
func newTestInsightsService(repo *fakeInsightsRepo, cfg InsightsConfig) (*InsightsService, *insightsClock) {
	clock := &insightsClock{now: time.Date(2026, 10, 12, 12, 0, 0, 0, timezone.Location())}
	svc := &InsightsService{
		repo:                repo,
		settings:            &fakeInsightsConfig{cfg: cfg},
		catalog:             &fakeInsightsCatalog{models: map[string]string{"claude-sonnet-4-6": "anthropic", "gpt-5.5": "openai"}},
		avatarKey:           deriveLeaderboardAvatarKey("test"),
		dashboardAggEnabled: true,
		cache:               newInsightsCache(clock.Now),
		now:                 clock.Now,
	}
	return svc, clock
}

func TestModelRanking_OnlyPublicModelsWithShares(t *testing.T) {
	repo := &fakeInsightsRepo{modelUsage: &InsightsModelUsage{
		TotalTokens: 1000,
		Rows: []InsightsModelUsageRow{
			{Model: "claude-sonnet-4-6", Tokens: 500, Requests: 10},
			{Model: "secret-exclusive-model", Tokens: 300, Requests: 3},
			{Model: "GPT-5.5", Tokens: 100, Requests: 4},
		},
	}}
	svc, _ := newTestInsightsService(repo, DefaultInsightsConfig())

	result, err := svc.ModelRanking(context.Background(), "2026-10-06", "2026-10-12")
	require.NoError(t, err)
	require.True(t, result.Enabled)
	require.Equal(t, "2026-10-06", result.StartDate)
	require.Equal(t, "2026-10-12", result.EndDate)
	require.True(t, repo.lastStart.Equal(time.Date(2026, 10, 6, 0, 0, 0, 0, timezone.Location())))
	require.True(t, repo.lastEnd.Equal(time.Date(2026, 10, 13, 0, 0, 0, 0, timezone.Location())), "the end date is inclusive")
	require.Len(t, result.Items, 2)
	require.Equal(t, "claude-sonnet-4-6", result.Items[0].Model)
	require.Equal(t, "anthropic", result.Items[0].Platform)
	require.Equal(t, 2, result.Items[1].Rank)
	require.InDelta(t, 50.0, result.Items[0].SharePercent, 0.001)
	// 不公开的模型并入「其他」，不单独出现
	require.InDelta(t, 40.0, result.OtherSharePercent, 0.001)
	raw, _ := json.Marshal(result)
	require.NotContains(t, string(raw), "secret-exclusive-model")
}

func TestModelRanking_DisabledAndInvalidRange(t *testing.T) {
	repo := &fakeInsightsRepo{}
	svc, _ := newTestInsightsService(repo, InsightsConfig{ModelRankingEnabled: false, UsageRankingLimit: 20})

	result, err := svc.ModelRanking(context.Background(), "", "")
	require.NoError(t, err)
	require.False(t, result.Enabled)
	require.Equal(t, "2026-10-12", result.StartDate)
	require.NotNil(t, result.Items)
	require.Zero(t, repo.modelCalls)

	_, err = svc.ModelRanking(context.Background(), "2026-10-12", "2026-10-01")
	require.ErrorIs(t, err, ErrInsightsInvalidRange)
	_, err = svc.ModelRanking(context.Background(), "2026-01-01", "2026-10-12")
	require.ErrorIs(t, err, ErrInsightsRangeTooLong)
}

func usageRows(n int) []InsightsUserUsageRow {
	rows := make([]InsightsUserUsageRow, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, InsightsUserUsageRow{
			UserID:     int64(100 + i),
			Username:   "user-number-" + string(rune('a'+i)),
			Email:      "private" + string(rune('a'+i)) + "@example.com",
			Tokens:     int64(1000 - i),
			Requests:   int64(10 + i),
			ActualCost: 1.5,
		})
	}
	return rows
}

func TestUserRanking_ProjectsWithoutIdentifiers(t *testing.T) {
	repo := &fakeInsightsRepo{userUsage: &InsightsUserUsage{Rows: usageRows(6), Participants: 6}}
	svc, _ := newTestInsightsService(repo, InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 20})

	result, err := svc.UserRanking(context.Background(), "", "", 102)
	require.NoError(t, err)
	require.True(t, result.Enabled)
	require.False(t, result.InsufficientData)
	require.Equal(t, InsightsMetricTokens, result.SortBy)
	require.Equal(t, 20, result.Limit)
	require.Equal(t, "2026-10-12", result.StartDate)
	require.Equal(t, "2026-10-12", result.EndDate)
	require.NotNil(t, result.GeneratedAt)
	require.Len(t, result.Items, 6)
	require.True(t, result.Items[2].IsSelf)
	require.False(t, result.Items[0].IsSelf)
	require.Equal(t, "u***a", result.Items[0].DisplayName)
	require.Nil(t, result.Items[0].ActualCost, "cost stays hidden unless the admin shows it")

	raw, err := json.Marshal(result)
	require.NoError(t, err)
	body := string(raw)
	for _, forbidden := range []string{"user_id", "email", "example.com", "user-number", "actual_cost", `"100"`} {
		require.NotContains(t, body, forbidden)
	}
}

func TestUserRanking_InsufficientParticipants(t *testing.T) {
	repo := &fakeInsightsRepo{userUsage: &InsightsUserUsage{Rows: usageRows(4), Participants: 4}}
	svc, _ := newTestInsightsService(repo, InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 20})

	result, err := svc.UserRanking(context.Background(), "2026-10-01", "2026-10-12", 0)
	require.NoError(t, err)
	require.True(t, result.InsufficientData)
	require.Empty(t, result.Items)
}

func TestUserRanking_SortByComesFromTheAdminSetting(t *testing.T) {
	repo := &fakeInsightsRepo{userUsage: &InsightsUserUsage{Rows: usageRows(5), Participants: 5}}

	// cost 没有打开花费显示时回落到 tokens，花费也不出现在响应里
	hidden, _ := newTestInsightsService(repo, InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 20, UsageRankingSortBy: InsightsMetricCost})
	result, err := hidden.UserRanking(context.Background(), "", "", 0)
	require.NoError(t, err)
	require.Equal(t, InsightsMetricTokens, result.SortBy)
	require.False(t, result.ShowCost)
	require.Equal(t, InsightsMetricTokens, repo.lastMetric)
	require.Nil(t, result.Items[0].ActualCost)

	shown, _ := newTestInsightsService(repo, InsightsConfig{UsageRankingEnabled: true, UsageRankingShowCost: true, UsageRankingLimit: 5, UsageRankingSortBy: InsightsMetricCost})
	result, err = shown.UserRanking(context.Background(), "", "", 0)
	require.NoError(t, err)
	require.Equal(t, InsightsMetricCost, result.SortBy)
	require.True(t, result.ShowCost)
	require.Equal(t, 5, result.Limit)
	require.NotNil(t, result.Items[0].ActualCost)
	require.Equal(t, InsightsMetricCost, repo.lastMetric)
	require.Equal(t, 5, repo.lastUserLimit)

	requests, _ := newTestInsightsService(repo, InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 20, UsageRankingSortBy: InsightsMetricRequests})
	result, err = requests.UserRanking(context.Background(), "", "", 0)
	require.NoError(t, err)
	require.Equal(t, InsightsMetricRequests, result.SortBy)
	require.Equal(t, InsightsMetricRequests, repo.lastMetric)
}

func TestUserRanking_DisabledByDefault(t *testing.T) {
	repo := &fakeInsightsRepo{}
	svc, _ := newTestInsightsService(repo, DefaultInsightsConfig())
	result, err := svc.UserRanking(context.Background(), "2026-10-10", "2026-10-12", 1)
	require.NoError(t, err)
	require.False(t, result.Enabled)
	require.Equal(t, "2026-10-10", result.StartDate)
	require.Equal(t, "2026-10-12", result.EndDate)
	require.Empty(t, result.Items)
	require.Zero(t, repo.userCalls)

	_, err = svc.UserRanking(context.Background(), "2026-10-12", "2026-10-10", 1)
	require.ErrorIs(t, err, ErrInsightsInvalidRange, "bad ranges are rejected even while the board is off")
}

func TestInsightsCache_KeysPerRangeSortByAndLimit(t *testing.T) {
	repo := &fakeInsightsRepo{
		modelUsage: &InsightsModelUsage{TotalTokens: 10, Rows: []InsightsModelUsageRow{{Model: "gpt-5.5", Tokens: 10}}},
		userUsage:  &InsightsUserUsage{Rows: usageRows(5), Participants: 5},
	}
	ctx := context.Background()
	svc, _ := newTestInsightsService(repo, InsightsConfig{ModelRankingEnabled: true, UsageRankingEnabled: true, UsageRankingLimit: 20})

	_, err := svc.ModelRanking(ctx, "", "")
	require.NoError(t, err)
	_, err = svc.ModelRanking(ctx, "2026-10-12", "2026-10-12")
	require.NoError(t, err)
	models, _ := repo.calls()
	require.Equal(t, 1, models, "an omitted range and the explicit today share one entry")
	_, err = svc.ModelRanking(ctx, "2026-10-11", "2026-10-12")
	require.NoError(t, err)
	models, _ = repo.calls()
	require.Equal(t, 2, models, "each range is cached on its own")

	_, err = svc.UserRanking(ctx, "", "", 1)
	require.NoError(t, err)
	_, err = svc.UserRanking(ctx, "", "", 2)
	require.NoError(t, err)
	_, users := repo.calls()
	require.Equal(t, 1, users, "the viewer does not split the cache")

	// 管理员改了排名指标或条数，换一个缓存条目
	svc.settings = &fakeInsightsConfig{cfg: InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 20, UsageRankingSortBy: InsightsMetricRequests}}
	svc.InvalidateConfig()
	_, err = svc.UserRanking(ctx, "", "", 1)
	require.NoError(t, err)
	svc.settings = &fakeInsightsConfig{cfg: InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 10, UsageRankingSortBy: InsightsMetricRequests}}
	svc.InvalidateConfig()
	_, err = svc.UserRanking(ctx, "", "", 1)
	require.NoError(t, err)
	_, users = repo.calls()
	require.Equal(t, 3, users)
}

func TestInsightsCache_FreshStaleAndExpired(t *testing.T) {
	repo := &fakeInsightsRepo{modelUsage: &InsightsModelUsage{TotalTokens: 10, Rows: []InsightsModelUsageRow{{Model: "gpt-5.5", Tokens: 10}}}}
	svc, clock := newTestInsightsService(repo, DefaultInsightsConfig())
	ctx := context.Background()

	_, err := svc.ModelRanking(ctx, "", "")
	require.NoError(t, err)
	_, err = svc.ModelRanking(ctx, "", "")
	require.NoError(t, err)
	require.Equal(t, 1, repo.modelCalls, "fresh entries are served from memory")

	// 含今天的范围 5 分钟过期；过期但不超过 3×TTL：先返回旧值，后台刷新一次
	clock.Advance(6 * time.Minute)
	_, err = svc.ModelRanking(ctx, "", "")
	require.NoError(t, err)
	svc.cache.refreshWG.Wait()
	require.Equal(t, 2, repo.modelCalls)

	// 已结束的范围缓存 60 分钟
	_, err = svc.ModelRanking(ctx, "2026-10-01", "2026-10-07")
	require.NoError(t, err)
	require.Equal(t, 3, repo.modelCalls)
	clock.Advance(30 * time.Minute)
	_, err = svc.ModelRanking(ctx, "2026-10-01", "2026-10-07")
	require.NoError(t, err)
	svc.cache.refreshWG.Wait()
	require.Equal(t, 3, repo.modelCalls, "a past range is still fresh after 30 minutes")
	clock.Advance(31 * time.Minute)
	_, err = svc.ModelRanking(ctx, "2026-10-01", "2026-10-07")
	require.NoError(t, err)
	svc.cache.refreshWG.Wait()
	require.Equal(t, 4, repo.modelCalls)

	// 超过 3×TTL 同步重查；查询失败时返回错误
	clock.Advance(time.Hour)
	repo.mu.Lock()
	repo.err = errors.New("db down")
	repo.mu.Unlock()
	_, err = svc.ModelRanking(ctx, "", "")
	require.Error(t, err)
}

func TestInsightsCache_TodayFetchedBeforeMidnightKeepsTheShortTTL(t *testing.T) {
	repo := &fakeInsightsRepo{modelUsage: &InsightsModelUsage{TotalTokens: 10, Rows: []InsightsModelUsageRow{{Model: "gpt-5.5", Tokens: 10}}}}
	svc, clock := newTestInsightsService(repo, DefaultInsightsConfig())
	ctx := context.Background()
	clock.Advance(11*time.Hour + 58*time.Minute) // 23:58

	_, err := svc.ModelRanking(ctx, "2026-10-12", "2026-10-12")
	require.NoError(t, err)
	clock.Advance(6 * time.Minute) // 次日 00:04，这个范围已经结束
	_, err = svc.ModelRanking(ctx, "2026-10-12", "2026-10-12")
	require.NoError(t, err)
	svc.cache.refreshWG.Wait()
	require.Equal(t, 2, repo.modelCalls, "data fetched while the day was open must not be kept for an hour")
}

func TestInsightsCache_StaleValueSurvivesRefreshFailure(t *testing.T) {
	repo := &fakeInsightsRepo{modelUsage: &InsightsModelUsage{TotalTokens: 10, Rows: []InsightsModelUsageRow{{Model: "gpt-5.5", Tokens: 10}}}}
	svc, clock := newTestInsightsService(repo, DefaultInsightsConfig())
	ctx := context.Background()

	_, err := svc.ModelRanking(ctx, "", "")
	require.NoError(t, err)

	repo.mu.Lock()
	repo.err = errors.New("db down")
	repo.mu.Unlock()
	clock.Advance(6 * time.Minute)
	result, err := svc.ModelRanking(ctx, "", "")
	require.NoError(t, err, "stale data is better than an error")
	require.Len(t, result.Items, 1)
	svc.cache.refreshWG.Wait()

	// 失败后一分钟内不再触发后台刷新
	_, _ = svc.ModelRanking(ctx, "", "")
	svc.cache.refreshWG.Wait()
	require.Equal(t, 2, repo.modelCalls)
}

func TestInsightsCache_EvictsLeastRecentlyUsedBeyondCapacity(t *testing.T) {
	clock := &insightsClock{now: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)}
	cache := newInsightsCache(clock.Now)
	cache.maxEntries = 3
	ttl := insightsFixedTTL(time.Hour)
	loads := map[string]int{}
	get := func(key string) {
		_, err := cache.get(context.Background(), key, ttl, func(context.Context) (any, error) {
			loads[key]++
			return key, nil
		})
		require.NoError(t, err)
		clock.Advance(time.Second)
	}

	get("a")
	get("b")
	get("c")
	get("a") // a 最近用过，b 成了最久没用的
	get("d")
	require.Len(t, cache.entries, 3)
	require.NotContains(t, cache.entries, "b")
	get("a")
	require.Equal(t, 1, loads["a"], "a recently used entry survives eviction")
	get("b")
	require.Equal(t, 2, loads["b"], "an evicted entry is loaded again")

	// 用户随便挑日期也撑不爆缓存
	big := newInsightsCache(clock.Now)
	for i := 0; i < insightsMaxCacheEntries+50; i++ {
		_, err := big.get(context.Background(), fmt.Sprintf("models:%d", i), ttl, func(context.Context) (any, error) { return i, nil })
		require.NoError(t, err)
	}
	require.Len(t, big.entries, insightsMaxCacheEntries)
}

func TestInsightsCache_LimitsConcurrentLoads(t *testing.T) {
	cache := newInsightsCache(time.Now)
	ttl := insightsFixedTTL(time.Hour)
	release := make(chan struct{})
	var mu sync.Mutex
	inFlight, peak := 0, 0
	var wg sync.WaitGroup
	for i := 0; i < insightsMaxConcurrentLoads*2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = cache.get(context.Background(), fmt.Sprintf("k%d", i), ttl, func(context.Context) (any, error) {
				mu.Lock()
				inFlight++
				if inFlight > peak {
					peak = inFlight
				}
				mu.Unlock()
				<-release
				mu.Lock()
				inFlight--
				mu.Unlock()
				return i, nil
			})
		}(i)
	}
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return inFlight == insightsMaxConcurrentLoads
	}, 2*time.Second, 5*time.Millisecond)
	close(release)
	wg.Wait()
	require.Equal(t, insightsMaxConcurrentLoads, peak)
}

func TestPublicStats(t *testing.T) {
	ctx := context.Background()
	repo := &fakeInsightsRepo{totals: &InsightsUsageTotals{TotalTokens: 1_200_000_000, TodayTokens: 3_000_000, TotalRequests: 42}}

	off, _ := newTestInsightsService(repo, DefaultInsightsConfig())
	require.Equal(t, &PublicStatsResult{Enabled: false}, off.PublicStats(ctx))

	on, clock := newTestInsightsService(repo, InsightsConfig{PublicStatsEnabled: true, UsageRankingLimit: 20})
	repo.watermark = clock.Now().Add(-10 * time.Minute)
	got := on.PublicStats(ctx)
	require.True(t, got.Available)
	require.Equal(t, int64(1_200_000_000), got.TotalTokens)
	require.Equal(t, int64(3_000_000), got.TodayTokens)
	require.NotNil(t, got.AsOf)

	// 聚合作业关闭：开关打开也不显示，且不去查表
	noAgg, _ := newTestInsightsService(repo, InsightsConfig{PublicStatsEnabled: true, UsageRankingLimit: 20})
	noAgg.dashboardAggEnabled = false
	calls := repo.totalsCalls
	require.False(t, noAgg.PublicStats(ctx).Available)
	require.Equal(t, calls, repo.totalsCalls)

	// 水位落后两小时以上：不显示
	stale, staleClock := newTestInsightsService(repo, InsightsConfig{PublicStatsEnabled: true, UsageRankingLimit: 20})
	repo.watermark = staleClock.Now().Add(-3 * time.Hour)
	require.False(t, stale.PublicStats(ctx).Available)

	// 查询失败：不报错，只是不可用
	failingRepo := &fakeInsightsRepo{err: errors.New("boom")}
	failing, failingClock := newTestInsightsService(failingRepo, InsightsConfig{PublicStatsEnabled: true, UsageRankingLimit: 20})
	failingRepo.watermark = failingClock.Now()
	result := failing.PublicStats(ctx)
	require.True(t, result.Enabled)
	require.False(t, result.Available)

	raw, _ := json.Marshal(got)
	require.False(t, strings.Contains(string(raw), "user"), "public stats never carry user counts")
}
