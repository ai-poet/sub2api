//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// findMediaTier 取出指定档位，找不到直接让用例失败。
func findMediaTier(t *testing.T, tiers []ModelCatalogMediaTier, tier string) ModelCatalogMediaTier {
	t.Helper()
	for _, item := range tiers {
		if item.Tier == tier {
			return item
		}
	}
	require.FailNowf(t, "media tier not found", "tier=%s", tier)
	return ModelCatalogMediaTier{}
}

// 分组只配了 image_price_1k/2k/4k 三列、渠道没有任何 tier 时，目录仍要下发三档
// 分辨率价格。这正是模型广场此前只显示一个价格的原因。
func TestModelCatalogService_GetCatalog_ImageTiersFromGroupColumns(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Image", Platform: PlatformAnthropic, Status: StatusActive,
			RateMultiplier:   0.5,
			SubscriptionType: SubscriptionTypeStandard,
			ImagePrice1K:     testPtrFloat64(0.10),
			ImagePrice2K:     testPtrFloat64(0.20),
			ImagePrice4K:     testPtrFloat64(0.40),
		}},
		map[int64][]string{10: {"img-model"}},
		nil,
		map[string]*LiteLLMModelPricing{"img-model": {OutputCostPerImage: 0.2}},
		[]ChannelModelPricing{{
			Platform:    PlatformAnthropic,
			Models:      []string{"img-model"},
			BillingMode: BillingModeImage,
		}},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	item := result.Items[0]

	require.Equal(t, string(BillingModeImage), item.BillingMode)
	require.Equal(t, MediaPriceUnitImage, item.PricingDetails.MediaUnit)
	require.Len(t, item.PricingDetails.MediaTiers, 3)

	// 三档分别取各自的分组列，并乘上分组倍率。
	for tier, unit := range map[string]float64{"1K": 0.10, "2K": 0.20, "4K": 0.40} {
		entry := findMediaTier(t, item.PricingDetails.MediaTiers, tier)
		require.NotNil(t, entry.EffectiveUSD, "tier %s", tier)
		require.InDelta(t, unit*0.5, *entry.EffectiveUSD, 1e-12, "tier %s", tier)
	}

	// 未指定尺寸时计费落在 2K，主价必须跟着它走，而不是最便宜的 1K。
	require.True(t, findMediaTier(t, item.PricingDetails.MediaTiers, "2K").IsDefaultTier)
	require.False(t, findMediaTier(t, item.PricingDetails.MediaTiers, "1K").IsDefaultTier)
	require.NotNil(t, item.EffectivePricingUSD.PerImageUSD)
	require.InDelta(t, 0.20*0.5, *item.EffectivePricingUSD.PerImageUSD, 1e-12)
}

// 分组三列全空时，档位回落到内置参考价，并沿用计费侧 2K×1.5、4K×2 的缩放。
func TestModelCatalogService_GetCatalog_ImageTiersFallBackToReferencePrice(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Image", Platform: PlatformAnthropic, Status: StatusActive,
			RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard,
		}},
		map[int64][]string{10: {"img-model"}},
		nil,
		map[string]*LiteLLMModelPricing{"img-model": {OutputCostPerImage: 0.2}},
		[]ChannelModelPricing{{
			Platform:    PlatformAnthropic,
			Models:      []string{"img-model"},
			BillingMode: BillingModeImage,
		}},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	tiers := result.Items[0].PricingDetails.MediaTiers

	for tier, want := range map[string]float64{"1K": 0.2, "2K": 0.3, "4K": 0.4} {
		entry := findMediaTier(t, tiers, tier)
		require.NotNil(t, entry.EffectiveUSD, "tier %s", tier)
		require.InDelta(t, want, *entry.EffectiveUSD, 1e-12, "tier %s", tier)
		require.NotNil(t, entry.OfficialUSD, "tier %s", tier)
		require.InDelta(t, want, *entry.OfficialUSD, 1e-12, "tier %s", tier)
	}
}

// 分组开了图片独立倍率时，展示价必须用 image_rate_multiplier，而不是文本倍率。
func TestModelCatalogService_GetCatalog_ImageUsesIndependentRateMultiplier(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Image", Platform: PlatformAnthropic, Status: StatusActive,
			RateMultiplier:       2,
			SubscriptionType:     SubscriptionTypeStandard,
			ImageRateIndependent: true,
			ImageRateMultiplier:  0.25,
			ImagePrice2K:         testPtrFloat64(0.40),
		}},
		map[int64][]string{10: {"img-model"}},
		nil,
		map[string]*LiteLLMModelPricing{"img-model": {OutputCostPerImage: 0.2}},
		[]ChannelModelPricing{{
			Platform:    PlatformAnthropic,
			Models:      []string{"img-model"},
			BillingMode: BillingModeImage,
		}},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	item := result.Items[0]

	require.NotNil(t, item.EffectivePricingUSD.PerImageUSD)
	require.InDelta(t, 0.40*0.25, *item.EffectivePricingUSD.PerImageUSD, 1e-12)
	require.InDelta(t, 0.25, item.BestGroup.RateMultiplier, 1e-12)
}

// 回归锁：per_image 是"每张"价，不能把按 token 计的 ImageOutputPricePerToken 写进去。
func TestModelCatalogService_GetCatalog_ImagePriceIsNotPerTokenPrice(t *testing.T) {
	const perImageTokenPrice = 3e-05
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Image", Platform: PlatformAnthropic, Status: StatusActive,
			RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard,
		}},
		map[int64][]string{10: {"img-model"}},
		nil,
		map[string]*LiteLLMModelPricing{
			"img-model": {OutputCostPerImageToken: perImageTokenPrice},
		},
		[]ChannelModelPricing{{
			Platform:    PlatformAnthropic,
			Models:      []string{"img-model"},
			BillingMode: BillingModeImage,
		}},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	item := result.Items[0]

	// 没有每张参考价时不编造官方价。
	require.Nil(t, item.OfficialPricing.PerImageUSD)
	// 实付价回落到内置每张默认价，而不是 per-token 价。
	require.NotNil(t, item.EffectivePricingUSD.PerImageUSD)
	require.Greater(t, *item.EffectivePricingUSD.PerImageUSD, perImageTokenPrice*1000)
	require.InDelta(t, defaultImageGenerationPrice*1.5, *item.EffectivePricingUSD.PerImageUSD, 1e-12)
}

// 视频模型此前整个落进 token 分支，显示四个空价格。
func TestModelCatalogService_GetCatalog_VideoTiersFromGroupColumns(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Video", Platform: PlatformAnthropic, Status: StatusActive,
			RateMultiplier:   1,
			SubscriptionType: SubscriptionTypeStandard,
			VideoPrice480P:   testPtrFloat64(0.05),
			VideoPrice720P:   testPtrFloat64(0.07),
			VideoPrice1080P:  testPtrFloat64(0.25),
		}},
		map[int64][]string{10: {"vid-model"}},
		nil,
		nil,
		[]ChannelModelPricing{{
			Platform:    PlatformAnthropic,
			Models:      []string{"vid-model"},
			BillingMode: BillingModeVideo,
		}},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	item := result.Items[0]

	require.Equal(t, string(BillingModeVideo), item.BillingMode)
	require.Equal(t, MediaPriceUnitSecond, item.PricingDetails.MediaUnit)
	require.Len(t, item.PricingDetails.MediaTiers, 3)
	require.True(t, findMediaTier(t, item.PricingDetails.MediaTiers, "480p").IsDefaultTier)

	for tier, want := range map[string]float64{"480p": 0.05, "720p": 0.07, "1080p": 0.25} {
		entry := findMediaTier(t, item.PricingDetails.MediaTiers, tier)
		require.NotNil(t, entry.EffectiveUSD, "tier %s", tier)
		require.InDelta(t, want, *entry.EffectiveUSD, 1e-12, "tier %s", tier)
	}

	require.NotNil(t, item.EffectivePricingUSD.PerSecondUSD)
	require.InDelta(t, 0.05, *item.EffectivePricingUSD.PerSecondUSD, 1e-12)
}

// token 模型不应该出现媒体档位。
func TestModelCatalogService_GetCatalog_TokenModelHasNoMediaTiers(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Text", Platform: PlatformAnthropic, Status: StatusActive,
			RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard,
		}},
		map[int64][]string{10: {"claude-sonnet-4"}},
		nil,
		map[string]*LiteLLMModelPricing{
			"claude-sonnet-4": {InputCostPerToken: 3e-6, OutputCostPerToken: 15e-6},
		},
		nil,
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Empty(t, result.Items[0].PricingDetails.MediaTiers)
	require.Equal(t, "", result.Items[0].PricingDetails.MediaUnit)
	require.Nil(t, result.Items[0].OfficialPricing.PerImageUSD)
}

// 没有显式定价卡的图片模型：扣费侧默认按张计费（只有定价卡显式 token 模式才按
// token 计），目录此前却回落到 LiteLLM token 参考价显示每百万 token 价格，
// 与账单对不上。gpt-image-2 必须按张展示，价格与 getDefaultImagePrice 同源。
func TestModelCatalogService_GetCatalog_ImageModelWithoutCardShowsPerImagePrice(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Image", Platform: PlatformOpenAI, Status: StatusActive,
			RateMultiplier: 0.2, SubscriptionType: SubscriptionTypeStandard,
		}},
		map[int64][]string{10: {"gpt-image-2"}},
		nil,
		map[string]*LiteLLMModelPricing{
			"gpt-image-2": {
				InputCostPerToken:  5e-6,
				OutputCostPerToken: 10e-6,
				OutputCostPerImage: 0.1,
			},
		},
		nil,
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	item := result.Items[0]

	require.Equal(t, string(BillingModeImage), item.BillingMode)
	require.Equal(t, MediaPriceUnitImage, item.PricingDetails.MediaUnit)
	require.Len(t, item.PricingDetails.MediaTiers, 3)
	require.True(t, findMediaTier(t, item.PricingDetails.MediaTiers, "2K").IsDefaultTier)

	// 主展示价 = 默认档位（2K = 1.5 倍基准）× 分组倍率，与扣费口径一致。
	require.NotNil(t, item.EffectivePricingUSD.PerImageUSD)
	require.InDelta(t, 0.1*1.5*0.2, *item.EffectivePricingUSD.PerImageUSD, 1e-12)
	require.NotNil(t, item.OfficialPricing.PerImageUSD)
	require.InDelta(t, 0.1*1.5, *item.OfficialPricing.PerImageUSD, 1e-12)

	// 图片模式下不再把 LiteLLM token 价当主价展示。
	require.Nil(t, item.EffectivePricingUSD.InputPerMTokUSD)
	require.Nil(t, item.EffectivePricingUSD.OutputPerMTokUSD)
}

// 显式配置 token 模式的定价卡优先：管理员明确要求按 token 计费时，
// 目录与扣费都保持 token 口径，不做隐式切换。
func TestModelCatalogService_GetCatalog_ImageModelWithExplicitTokenCardStaysToken(t *testing.T) {
	svc := newModelCatalogTestService(t,
		[]Group{{
			ID: 10, Name: "Image", Platform: PlatformAnthropic, Status: StatusActive,
			RateMultiplier: 1, SubscriptionType: SubscriptionTypeStandard,
		}},
		map[int64][]string{10: {"gpt-image-2"}},
		nil,
		map[string]*LiteLLMModelPricing{
			"gpt-image-2": {
				InputCostPerToken:  5e-6,
				OutputCostPerToken: 10e-6,
				OutputCostPerImage: 0.1,
			},
		},
		[]ChannelModelPricing{{
			Platform:    PlatformAnthropic,
			Models:      []string{"gpt-image-2"},
			BillingMode: BillingModeToken,
		}},
	)

	result, err := svc.GetCatalog(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	item := result.Items[0]

	require.Equal(t, string(BillingModeToken), item.BillingMode)
	require.Empty(t, item.PricingDetails.MediaTiers)
	require.NotNil(t, item.EffectivePricingUSD.InputPerMTokUSD)
}
