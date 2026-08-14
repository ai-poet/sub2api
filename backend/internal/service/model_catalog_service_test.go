//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type modelCatalogAccessStub struct {
	groups       func(ctx context.Context, userID int64) ([]Group, error)
	rates        func(ctx context.Context, userID int64) (map[int64]float64, error)
	publicGroups func(ctx context.Context) ([]Group, error)
}

func (s *modelCatalogAccessStub) GetAvailableGroups(ctx context.Context, userID int64) ([]Group, error) {
	return s.groups(ctx, userID)
}

func (s *modelCatalogAccessStub) GetUserGroupRates(ctx context.Context, userID int64) (map[int64]float64, error) {
	return s.rates(ctx, userID)
}

// ListPublicGroups 默认复用 groups（userID 0），让既有用例无需改动；
// 需要区分公开分组口径的用例可以单独覆盖 publicGroups。
func (s *modelCatalogAccessStub) ListPublicGroups(ctx context.Context) ([]Group, error) {
	if s.publicGroups != nil {
		return s.publicGroups(ctx)
	}
	return s.groups(ctx, 0)
}

type modelCatalogModelsStub struct {
	byGroup map[int64][]string
}

func (s *modelCatalogModelsStub) GetAvailableModels(_ context.Context, groupID *int64, _ string) []string {
	if groupID == nil {
		return nil
	}
	models := s.byGroup[*groupID]
	cp := make([]string, len(models))
	copy(cp, models)
	return cp
}

func newModelCatalogTestService(t *testing.T, groups []Group, modelsByGroup map[int64][]string, userRates map[int64]float64, pricing map[string]*LiteLLMModelPricing, channelPricing []ChannelModelPricing) *ModelCatalogService {
	t.Helper()

	repo := &mockChannelRepository{
		listAllFn: func(_ context.Context) ([]Channel, error) {
			return []Channel{{
				ID:           1,
				Name:         "catalog-channel",
				Status:       StatusActive,
				GroupIDs:     []int64{10, 20, 30},
				ModelPricing: channelPricing,
			}}, nil
		},
		getGroupPlatformsFn: func(_ context.Context, ids []int64) (map[int64]string, error) {
			out := make(map[int64]string, len(ids))
			for _, id := range ids {
				switch id {
				case 10, 20, 30:
					out[id] = PlatformAnthropic
				default:
					out[id] = PlatformAnthropic
				}
			}
			return out, nil
		},
	}

	billing := &BillingService{
		fallbackPrices: map[string]*ModelPricing{
			"claude-sonnet-4": {
				InputPricePerToken:         3e-6,
				OutputPricePerToken:        15e-6,
				CacheCreationPricePerToken: 3.75e-6,
				CacheReadPricePerToken:     0.3e-6,
			},
		},
		pricingService: &PricingService{
			pricingData: pricing,
		},
	}

	return NewModelCatalogService(
		&modelCatalogAccessStub{
			groups: func(_ context.Context, _ int64) ([]Group, error) { return groups, nil },
			rates:  func(_ context.Context, _ int64) (map[int64]float64, error) { return userRates, nil },
		},
		&modelCatalogModelsStub{byGroup: modelsByGroup},
		billing,
		NewModelPricingResolver(NewChannelService(repo, nil, nil, nil), billing),
	)
}

func TestModelCatalogService_GetCatalog_ReturnsGroupModelCards(t *testing.T) {
	pricing := map[string]*LiteLLMModelPricing{
		"claude-sonnet-4": {
			InputCostPerToken:           3e-6,
			OutputCostPerToken:          15e-6,
			CacheCreationInputTokenCost: 3.75e-6,
			CacheReadInputTokenCost:     0.3e-6,
			SupportsPromptCaching:       true,
		},
	}

	svc := newModelCatalogTestService(t,
		[]Group{
			{ID: 10, Name: "Alpha", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 0.8, SubscriptionType: SubscriptionTypeStandard},
			{ID: 20, Name: "Beta", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 1.2, SubscriptionType: SubscriptionTypeStandard},
		},
		map[int64][]string{
			10: {"claude-sonnet-4"},
			20: {"claude-sonnet-4"},
		},
		nil,
		pricing,
		nil,
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	require.Equal(t, 2, result.Summary.TotalModels)
	require.Equal(t, 2, result.Summary.TokenModels)

	require.Equal(t, "Alpha", result.Items[0].BestGroup.Name)
	require.Equal(t, 0.8, result.Items[0].BestGroup.RateMultiplier)
	require.Equal(t, 2, result.Items[0].AvailableGroupCount)
	require.NotNil(t, result.Items[0].Comparison.SavingsPercent)
	require.Greater(t, *result.Items[0].Comparison.SavingsPercent, 0.0)

	require.Equal(t, "Beta", result.Items[1].BestGroup.Name)
	require.Equal(t, 2, result.Items[1].AvailableGroupCount)
	require.Len(t, result.Items[0].OtherGroups, 1)
}

// 公开目录没有「当前用户」，因此即便存在用户专属倍率也必须忽略，
// 一律回落到分组默认倍率（RateSource = group_default）。
func TestModelCatalogService_GetPublicCatalog_IgnoresUserOverrideRate(t *testing.T) {
	pricing := map[string]*LiteLLMModelPricing{
		"claude-sonnet-4": {
			InputCostPerToken:  3e-6,
			OutputCostPerToken: 15e-6,
		},
	}

	svc := newModelCatalogTestService(t,
		[]Group{
			{ID: 10, Name: "Alpha", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 1.4, SubscriptionType: SubscriptionTypeStandard},
		},
		map[int64][]string{
			10: {"claude-sonnet-4"},
		},
		// 用户专属倍率 0.6 存在，但公开目录不应采用它。
		map[int64]float64{10: 0.6},
		pricing,
		nil,
	)

	result, err := svc.GetPublicCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "group_default", result.Items[0].BestGroup.RateSource)
	require.InDelta(t, 1.4, result.Items[0].BestGroup.RateMultiplier, 1e-12)
	// 有效单价 = 官方 $3/Mtok × 分组默认倍率 1.4
	require.NotNil(t, result.Items[0].EffectivePricingUSD.InputPerMTokUSD)
	require.InDelta(t, 4.2, *result.Items[0].EffectivePricingUSD.InputPerMTokUSD, 1e-9)
}

// GetPublicCatalog 走的是 ListPublicGroups 而非 GetAvailableGroups——
// 专属 / 免费订阅分组在那一层就被排除，公开目录只能看到剩下的分组。
func TestModelCatalogService_GetPublicCatalog_UsesPublicGroupSource(t *testing.T) {
	pricing := map[string]*LiteLLMModelPricing{
		"claude-sonnet-4": {InputCostPerToken: 3e-6, OutputCostPerToken: 15e-6},
	}

	svc := newModelCatalogTestService(t,
		[]Group{
			{ID: 10, Name: "PublicGroup", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard},
			{ID: 20, Name: "ExclusiveGroup", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard},
		},
		map[int64][]string{
			10: {"claude-sonnet-4"},
			20: {"claude-sonnet-4"},
		},
		nil,
		pricing,
		nil,
	)
	// 模拟 ListPublicGroups 已剔除专属分组，只放行 ID=10。
	svc.accessService.(*modelCatalogAccessStub).publicGroups = func(_ context.Context) ([]Group, error) {
		return []Group{
			{ID: 10, Name: "PublicGroup", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard},
		}, nil
	}

	result, err := svc.GetPublicCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "PublicGroup", result.Items[0].BestGroup.Name)

	// 对照：登录态目录仍能看到两个分组，证明公开路径的收窄没有影响既有行为。
	authed, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, authed.Items, 2)
}

func TestModelCatalogService_GetCatalog_UsesUserOverrideRate(t *testing.T) {
	pricing := map[string]*LiteLLMModelPricing{
		"claude-sonnet-4": {
			InputCostPerToken:           3e-6,
			OutputCostPerToken:          15e-6,
			CacheCreationInputTokenCost: 3.75e-6,
			CacheReadInputTokenCost:     0.3e-6,
		},
	}

	svc := newModelCatalogTestService(t,
		[]Group{
			{ID: 10, Name: "Alpha", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 1.4, SubscriptionType: SubscriptionTypeStandard},
		},
		map[int64][]string{
			10: {"claude-sonnet-4"},
		},
		map[int64]float64{10: 0.6},
		pricing,
		nil,
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "user_override", result.Items[0].BestGroup.RateSource)
	require.InDelta(t, 0.6, result.Items[0].BestGroup.RateMultiplier, 1e-12)
	require.NotNil(t, result.Items[0].EffectivePricingUSD.InputPerMTokUSD)
	require.InDelta(t, 1.8, *result.Items[0].EffectivePricingUSD.InputPerMTokUSD, 1e-9)
}

func TestModelCatalogService_GetCatalog_KeepsTokenIntervalsWithoutFlattening(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{
			{ID: 10, Name: "Tiered", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard},
		},
		map[int64][]string{
			10: {"claude-sonnet-4"},
		},
		nil,
		map[string]*LiteLLMModelPricing{
			"claude-sonnet-4": {
				InputCostPerToken:       3e-6,
				OutputCostPerToken:      15e-6,
				CacheReadInputTokenCost: 0.3e-6,
			},
		},
		[]ChannelModelPricing{{
			Platform:    PlatformAnthropic,
			Models:      []string{"claude-sonnet-4"},
			BillingMode: BillingModeToken,
			Intervals: []PricingInterval{
				{MinTokens: 0, MaxTokens: testPtrInt(128000), InputPrice: testPtrFloat64(2e-6), OutputPrice: testPtrFloat64(10e-6)},
				{MinTokens: 128000, MaxTokens: nil, InputPrice: testPtrFloat64(4e-6), OutputPrice: testPtrFloat64(20e-6)},
			},
		}},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Nil(t, result.Items[0].EffectivePricingUSD.InputPerMTokUSD)
	require.Len(t, result.Items[0].PricingDetails.Intervals, 2)
	require.Nil(t, result.Items[0].Comparison.SavingsPercent)
}

func TestModelCatalogService_GetCatalog_SupportsPerRequestAndImageModes(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{
			{ID: 10, Name: "Prompt", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 0.5, SubscriptionType: SubscriptionTypeStandard},
			{ID: 20, Name: "Image", Platform: PlatformAnthropic, Status: StatusActive, RateMultiplier: 0, SubscriptionType: SubscriptionTypeSubscription},
		},
		map[int64][]string{
			10: {"req-model"},
			20: {"img-model"},
		},
		nil,
		map[string]*LiteLLMModelPricing{
			"img-model": {OutputCostPerImage: 0.2},
		},
		[]ChannelModelPricing{
			{
				Platform:        PlatformAnthropic,
				Models:          []string{"req-model"},
				BillingMode:     BillingModePerRequest,
				PerRequestPrice: testPtrFloat64(0.08),
			},
			{
				Platform:        PlatformAnthropic,
				Models:          []string{"img-model"},
				BillingMode:     BillingModeImage,
				PerRequestPrice: testPtrFloat64(0.12),
			},
		},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 2)

	require.Equal(t, string(BillingModePerRequest), result.Items[0].BillingMode)
	require.NotNil(t, result.Items[0].EffectivePricingUSD.PerRequestUSD)
	require.InDelta(t, 0.04, *result.Items[0].EffectivePricingUSD.PerRequestUSD, 1e-12)

	require.Equal(t, string(BillingModeImage), result.Items[1].BillingMode)
	require.Equal(t, "group_default", result.Items[1].BestGroup.RateSource)
	require.Equal(t, 0.0, result.Items[1].BestGroup.RateMultiplier)
	require.NotNil(t, result.Items[1].EffectivePricingUSD.PerImageUSD)
	require.InDelta(t, 0.0, *result.Items[1].EffectivePricingUSD.PerImageUSD, 1e-12)
	require.NotNil(t, result.Items[1].OfficialPricing.PerImageUSD)
	require.NotNil(t, result.Items[1].Comparison.SavingsPercent)
	require.InDelta(t, 1.0, *result.Items[1].Comparison.SavingsPercent, 1e-12)
}
