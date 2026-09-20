package service

// 模型广场的图片/视频档位展示。
//
// 这里刻意不自己推导单价公式：每个档位的价格都按 GatewayService.calculateImageCost /
// calculateOpenAIVideoCost 的优先级顺序解析，最终兜底直接调用计费侧的
// getImageUnitPrice / getVideoUnitPrice。展示价与账单必须同源，否则用户看到的
// 和扣掉的对不上。

const (
	// MediaPriceUnitImage 表示 MediaTiers 的单价是"每张图片"。
	MediaPriceUnitImage = "image"
	// MediaPriceUnitSecond 表示 MediaTiers 的单价是"每秒"（视频按秒计费）。
	MediaPriceUnitSecond = "second"
)

// imageBillingTiers / videoBillingResolutions 是展示顺序，从低到高。
var (
	imageBillingTiers       = []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K}
	videoBillingResolutions = []string{
		VideoBillingResolution480P,
		VideoBillingResolution720P,
		VideoBillingResolution1080P,
	}
)

// defaultImageBillingTier 是请求未带尺寸时计费实际落到的档位（2K）。
func defaultImageBillingTier() string {
	return NormalizeImageBillingTierOrDefault("")
}

// defaultVideoBillingResolution 是请求未带分辨率时计费实际落到的档位（480p）。
func defaultVideoBillingResolution() string {
	return NormalizeVideoBillingResolutionOrDefault("")
}

// resolvedRequestTierPrice 取渠道/分组定价卡里该档位的按次价，没有匹配档位时
// 退回卡片上的默认按次价。口径同 ModelPricingResolver.GetRequestTierPrice。
func resolvedRequestTierPrice(resolved *ResolvedPricing, tierLabel string) *float64 {
	if resolved == nil {
		return nil
	}
	for _, tier := range resolved.RequestTiers {
		if tier.TierLabel == tierLabel && tier.PerRequestPrice != nil {
			return copyFloatPtr(tier.PerRequestPrice)
		}
	}
	if resolved.DefaultPerRequestPrice > 0 {
		return floatPtr(resolved.DefaultPerRequestPrice)
	}
	return nil
}

// imageTierUnitPrice 解析单张图片在指定档位的单价（未乘倍率）。
// 优先级镜像 GatewayService.calculateImageCost：
// 分组定价卡 → 分组 image_price_* 列 → 渠道定价卡 → 内置参考价。
func (s *ModelCatalogService) imageTierUnitPrice(model, tier string, resolved *ResolvedPricing, group Group) *float64 {
	fromCard := resolvedRequestTierPrice(resolved, tier)
	if fromCard != nil && resolved.Source == PricingSourceGroup {
		return fromCard
	}
	if price := group.GetImagePrice(tier); price != nil {
		return copyFloatPtr(price)
	}
	if fromCard != nil {
		return fromCard
	}
	if s == nil || s.billing == nil {
		return nil
	}
	return floatPtr(s.billing.getImageUnitPrice(model, tier, nil))
}

// videoTierUnitPrice 解析视频在指定分辨率的每秒单价（未乘倍率）。
// 优先级镜像 OpenAIGatewayService 的视频计费路径。
func (s *ModelCatalogService) videoTierUnitPrice(model, resolution string, resolved *ResolvedPricing, group Group) *float64 {
	fromCard := resolvedRequestTierPrice(resolved, resolution)
	if fromCard != nil && resolved.Source == PricingSourceGroup {
		return fromCard
	}
	if price := group.GetVideoPriceForModel(model, resolution); price != nil {
		return copyFloatPtr(price)
	}
	if fromCard != nil {
		return fromCard
	}
	if s == nil || s.billing == nil {
		return nil
	}
	return floatPtr(s.billing.getVideoUnitPrice(model, resolution, nil))
}

// officialImageTierPrice 是该档位的参考价（不含分组配置、不乘倍率）。
// 只有存在真实参考价时才返回值：否则 getDefaultImagePrice 会兜底到硬编码的
// defaultImageGenerationPrice，把"没有参考价"渲染成一个看似权威的数字。
func (s *ModelCatalogService) officialImageTierPrice(model, tier string) *float64 {
	if s == nil || s.billing == nil || !s.hasImageReferencePrice(model) {
		return nil
	}
	return floatPtr(s.billing.getDefaultImagePrice(model, tier))
}

func (s *ModelCatalogService) hasImageReferencePrice(model string) bool {
	if _, ok := getDefaultGrokImagineImagePrice(model, ImageBillingSize1K); ok {
		return true
	}
	raw := s.getRawPricing(model)
	return raw != nil && raw.OutputCostPerImage > 0
}

// officialVideoTierPrice 同上。内置定价表不含视频价，所以参考价目前只有
// Grok Imagine 系列有。
func (s *ModelCatalogService) officialVideoTierPrice(model, resolution string) *float64 {
	if s == nil || s.billing == nil {
		return nil
	}
	price, ok := getDefaultGrokImagineVideoPrice(model, resolution)
	if !ok {
		return nil
	}
	return floatPtr(price)
}

func (s *ModelCatalogService) buildImageMediaTiers(
	model string,
	resolved *ResolvedPricing,
	rateMultiplier float64,
	group Group,
) []ModelCatalogMediaTier {
	defaultTier := defaultImageBillingTier()
	tiers := make([]ModelCatalogMediaTier, 0, len(imageBillingTiers))
	for _, tier := range imageBillingTiers {
		entry := ModelCatalogMediaTier{
			Tier:          tier,
			OfficialUSD:   s.officialImageTierPrice(model, tier),
			IsDefaultTier: tier == defaultTier,
		}
		if unit := s.imageTierUnitPrice(model, tier, resolved, group); unit != nil {
			entry.EffectiveUSD = floatPtr(*unit * rateMultiplier)
		}
		tiers = append(tiers, entry)
	}
	return tiers
}

func (s *ModelCatalogService) buildVideoMediaTiers(
	model string,
	resolved *ResolvedPricing,
	rateMultiplier float64,
	group Group,
) []ModelCatalogMediaTier {
	defaultResolution := defaultVideoBillingResolution()
	tiers := make([]ModelCatalogMediaTier, 0, len(videoBillingResolutions))
	for _, resolution := range videoBillingResolutions {
		entry := ModelCatalogMediaTier{
			Tier:          resolution,
			OfficialUSD:   s.officialVideoTierPrice(model, resolution),
			IsDefaultTier: resolution == defaultResolution,
		}
		if unit := s.videoTierUnitPrice(model, resolution, resolved, group); unit != nil {
			entry.EffectiveUSD = floatPtr(*unit * rateMultiplier)
		}
		tiers = append(tiers, entry)
	}
	return tiers
}
