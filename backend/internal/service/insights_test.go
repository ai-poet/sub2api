package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

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

	// 损坏的 JSON 整体回落到默认值，不会半解析出奇怪的开关
	require.Equal(t, def, ParseInsightsConfig(`{"usage_ranking_enabled": true, broken`))

	// 缺失的字段保留默认值，显式 false 生效
	cfg := ParseInsightsConfig(`{"model_ranking_enabled": false, "usage_ranking_limit": 500}`)
	require.False(t, cfg.ModelRankingEnabled)
	require.Equal(t, 50, cfg.UsageRankingLimit)
	require.Equal(t, 3, ParseInsightsConfig(`{"usage_ranking_limit": 1}`).UsageRankingLimit)
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

	saved, err := svc.UpdateInsightsConfig(context.Background(), InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 99})
	require.NoError(t, err)
	require.Equal(t, 50, saved.UsageRankingLimit)
	// 注入 HTML 的公开设置缓存必须刷新，否则开关要等缓存过期才生效
	require.Equal(t, 1, refreshed)

	loaded, err := svc.GetInsightsConfig(context.Background())
	require.NoError(t, err)
	require.True(t, loaded.UsageRankingEnabled)
	require.False(t, loaded.ModelRankingEnabled)
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
}

func (r *fakeInsightsRepo) ModelUsageRanking(context.Context, time.Time, time.Time, int) (*InsightsModelUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modelCalls++
	if r.err != nil {
		return nil, r.err
	}
	return r.modelUsage, nil
}

func (r *fakeInsightsRepo) UserUsageRanking(_ context.Context, _, _ time.Time, metric string, limit int) (*InsightsUserUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userCalls++
	r.lastMetric = metric
	r.lastUserLimit = limit
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

func newTestInsightsService(repo *fakeInsightsRepo, cfg InsightsConfig) (*InsightsService, *insightsClock) {
	clock := &insightsClock{now: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)}
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

	result, err := svc.ModelRanking(context.Background(), InsightsWindow7d)
	require.NoError(t, err)
	require.True(t, result.Enabled)
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

func TestModelRanking_DisabledAndInvalidWindow(t *testing.T) {
	repo := &fakeInsightsRepo{}
	svc, _ := newTestInsightsService(repo, InsightsConfig{ModelRankingEnabled: false, UsageRankingLimit: 20})

	result, err := svc.ModelRanking(context.Background(), InsightsWindow24h)
	require.NoError(t, err)
	require.False(t, result.Enabled)
	require.NotNil(t, result.Items)
	require.Zero(t, repo.modelCalls)

	_, err = svc.ModelRanking(context.Background(), "1y")
	require.ErrorIs(t, err, ErrInsightsInvalidWindow)
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

	result, err := svc.UserRanking(context.Background(), InsightsWindow7d, "", 102)
	require.NoError(t, err)
	require.True(t, result.Enabled)
	require.False(t, result.InsufficientData)
	require.Equal(t, InsightsMetricTokens, result.Metric)
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

	result, err := svc.UserRanking(context.Background(), InsightsWindow24h, InsightsMetricTokens, 0)
	require.NoError(t, err)
	require.True(t, result.InsufficientData)
	require.Empty(t, result.Items)
}

func TestUserRanking_CostMetricNeedsShowCost(t *testing.T) {
	repo := &fakeInsightsRepo{userUsage: &InsightsUserUsage{Rows: usageRows(5), Participants: 5}}
	svc, _ := newTestInsightsService(repo, InsightsConfig{UsageRankingEnabled: true, UsageRankingLimit: 20})

	_, err := svc.UserRanking(context.Background(), InsightsWindow7d, InsightsMetricCost, 0)
	require.ErrorIs(t, err, ErrInsightsInvalidMetric)
	_, err = svc.UserRanking(context.Background(), InsightsWindow7d, "email", 0)
	require.ErrorIs(t, err, ErrInsightsInvalidMetric)
	require.Zero(t, repo.userCalls)

	shown, _ := newTestInsightsService(repo, InsightsConfig{UsageRankingEnabled: true, UsageRankingShowCost: true, UsageRankingLimit: 5})
	result, err := shown.UserRanking(context.Background(), InsightsWindow7d, InsightsMetricCost, 0)
	require.NoError(t, err)
	require.True(t, result.ShowCost)
	require.NotNil(t, result.Items[0].ActualCost)
	require.Equal(t, InsightsMetricCost, repo.lastMetric)
	require.Equal(t, 5, repo.lastUserLimit)
}

func TestUserRanking_DisabledByDefault(t *testing.T) {
	repo := &fakeInsightsRepo{}
	svc, _ := newTestInsightsService(repo, DefaultInsightsConfig())
	result, err := svc.UserRanking(context.Background(), InsightsWindow7d, "", 1)
	require.NoError(t, err)
	require.False(t, result.Enabled)
	require.Zero(t, repo.userCalls)
}

func TestInsightsCache_FreshStaleAndExpired(t *testing.T) {
	repo := &fakeInsightsRepo{modelUsage: &InsightsModelUsage{TotalTokens: 10, Rows: []InsightsModelUsageRow{{Model: "gpt-5.5", Tokens: 10}}}}
	svc, clock := newTestInsightsService(repo, DefaultInsightsConfig())
	ctx := context.Background()

	_, err := svc.ModelRanking(ctx, InsightsWindow24h)
	require.NoError(t, err)
	_, err = svc.ModelRanking(ctx, InsightsWindow24h)
	require.NoError(t, err)
	require.Equal(t, 1, repo.modelCalls, "fresh entries are served from memory")

	// 过期但不超过 3×TTL：先返回旧值，后台刷新一次
	clock.Advance(6 * time.Minute)
	_, err = svc.ModelRanking(ctx, InsightsWindow24h)
	require.NoError(t, err)
	svc.cache.refreshWG.Wait()
	require.Equal(t, 2, repo.modelCalls)

	// 其他档位各自缓存
	_, err = svc.ModelRanking(ctx, InsightsWindow7d)
	require.NoError(t, err)
	require.Equal(t, 3, repo.modelCalls)

	// 超过 3×TTL 同步重查；查询失败时返回错误
	clock.Advance(time.Hour)
	repo.mu.Lock()
	repo.err = errors.New("db down")
	repo.mu.Unlock()
	_, err = svc.ModelRanking(ctx, InsightsWindow24h)
	require.Error(t, err)
}

func TestInsightsCache_StaleValueSurvivesRefreshFailure(t *testing.T) {
	repo := &fakeInsightsRepo{modelUsage: &InsightsModelUsage{TotalTokens: 10, Rows: []InsightsModelUsageRow{{Model: "gpt-5.5", Tokens: 10}}}}
	svc, clock := newTestInsightsService(repo, DefaultInsightsConfig())
	ctx := context.Background()

	_, err := svc.ModelRanking(ctx, InsightsWindow24h)
	require.NoError(t, err)

	repo.mu.Lock()
	repo.err = errors.New("db down")
	repo.mu.Unlock()
	clock.Advance(6 * time.Minute)
	result, err := svc.ModelRanking(ctx, InsightsWindow24h)
	require.NoError(t, err, "stale data is better than an error")
	require.Len(t, result.Items, 1)
	svc.cache.refreshWG.Wait()

	// 失败后一分钟内不再触发后台刷新
	_, _ = svc.ModelRanking(ctx, InsightsWindow24h)
	svc.cache.refreshWG.Wait()
	require.Equal(t, 2, repo.modelCalls)
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
