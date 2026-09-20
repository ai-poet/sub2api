package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const mtokMultiplier = 1_000_000

type modelCatalogAccessService interface {
	GetAvailableGroups(ctx context.Context, userID int64) ([]Group, error)
	GetUserGroupRates(ctx context.Context, userID int64) (map[int64]float64, error)
	// ListPublicGroups 返回可对匿名访客公开展示的分组（活跃 + 非专属 + 非免费订阅）。
	ListPublicGroups(ctx context.Context) ([]Group, error)
}

type modelCatalogModelsService interface {
	GetAvailableModels(ctx context.Context, groupID *int64, platform string) []string
}

type ModelCatalogService struct {
	accessService modelCatalogAccessService
	modelService  modelCatalogModelsService
	billing       *BillingService
	resolver      *ModelPricingResolver
}

func NewModelCatalogService(
	accessService modelCatalogAccessService,
	modelService modelCatalogModelsService,
	billing *BillingService,
	resolver *ModelPricingResolver,
) *ModelCatalogService {
	return &ModelCatalogService{
		accessService: accessService,
		modelService:  modelService,
		billing:       billing,
		resolver:      resolver,
	}
}

type ModelCatalogResponse struct {
	Items   []ModelCatalogItem  `json:"items"`
	Summary ModelCatalogSummary `json:"summary"`
}

type ModelCatalogSummary struct {
	TotalModels       int     `json:"total_models"`
	TokenModels       int     `json:"token_models"`
	NonTokenModels    int     `json:"non_token_models"`
	BestSavingsModel  string  `json:"best_savings_model"`
	MaxSavingsPercent float64 `json:"max_savings_percent"`
}

type ModelCatalogItem struct {
	Model               string                       `json:"model"`
	DisplayName         string                       `json:"display_name"`
	Platform            string                       `json:"platform"`
	BillingMode         string                       `json:"billing_mode"`
	BestGroup           ModelCatalogGroupRef         `json:"best_group"`
	AvailableGroupCount int                          `json:"available_group_count"`
	OfficialPricing     ModelCatalogPricing          `json:"official_pricing"`
	EffectivePricingUSD ModelCatalogPricing          `json:"effective_pricing_usd"`
	Comparison          ModelCatalogComparison       `json:"comparison"`
	PricingDetails      ModelCatalogPricingDetails   `json:"pricing_details"`
	OtherGroups         []ModelCatalogGroupCompanion `json:"other_groups"`

	// ContextWindow 模型上下文窗口（token），客户端用量计的分母。
	// 指针类型且 omitempty：来源不知道时必须整个缺省，不能退化成 0 —— 客户端
	// 靠"字段缺失"来决定不显示百分比，写 0 会被当成一个真实的窗口值。
	ContextWindow *int `json:"context_window,omitempty"`
}

type ModelCatalogGroupRef struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	RateSource     string  `json:"rate_source"`
}

type ModelCatalogPricing struct {
	InputPerMTokUSD      *float64 `json:"input_per_mtok_usd"`
	OutputPerMTokUSD     *float64 `json:"output_per_mtok_usd"`
	CacheWritePerMTokUSD *float64 `json:"cache_write_per_mtok_usd"`
	CacheReadPerMTokUSD  *float64 `json:"cache_read_per_mtok_usd"`
	PerRequestUSD        *float64 `json:"per_request_usd"`
	PerImageUSD          *float64 `json:"per_image_usd"`
	PerSecondUSD         *float64 `json:"per_second_usd"`
	Source               string   `json:"source"`
	HasReference         bool     `json:"has_reference"`
}

type ModelCatalogComparison struct {
	SavingsPercent        *float64 `json:"savings_percent"`
	IsCheaperThanOfficial bool     `json:"is_cheaper_than_official"`
	DeltaInputPerMTokUSD  *float64 `json:"delta_input_per_mtok_usd"`
	DeltaOutputPerMTokUSD *float64 `json:"delta_output_per_mtok_usd"`
	DeltaPerRequestUSD    *float64 `json:"delta_per_request_usd"`
	DeltaPerImageUSD      *float64 `json:"delta_per_image_usd"`
	DeltaPerSecondUSD     *float64 `json:"delta_per_second_usd"`
}

type ModelCatalogPricingDetails struct {
	SupportsPromptCaching     bool                        `json:"supports_prompt_caching"`
	HasLongContextMultiplier  bool                        `json:"has_long_context_multiplier"`
	LongContextInputThreshold int                         `json:"long_context_input_threshold"`
	Intervals                 []ModelCatalogPriceInterval `json:"intervals"`

	// MediaTiers 是图片/视频模型按分辨率分档的单价。它和 Intervals 是两条不同的
	// 分档轴：Intervals 按上下文 token 区间分，MediaTiers 按分辨率分，min/max_tokens
	// 对后者没有意义。复用 Intervals 会让前端把档位标题渲染成 "≥ —"。
	MediaTiers []ModelCatalogMediaTier `json:"media_tiers"`
	// MediaUnit 是 MediaTiers 的计价单位："image"（每张）、"second"（每秒），
	// 非媒体模型为空串。
	MediaUnit string `json:"media_unit"`
}

// ModelCatalogMediaTier 是单个分辨率档位的展示价。
type ModelCatalogMediaTier struct {
	// Tier 为 "1K"/"2K"/"4K"（图片）或 "480p"/"720p"/"1080p"（视频）。
	Tier         string   `json:"tier"`
	OfficialUSD  *float64 `json:"official_usd"`
	EffectiveUSD *float64 `json:"effective_usd"`
	// IsDefaultTier 标记请求未指定尺寸时计费实际落到的档位。
	IsDefaultTier bool `json:"is_default_tier"`
}

type ModelCatalogPriceInterval struct {
	MinTokens            int      `json:"min_tokens"`
	MaxTokens            *int     `json:"max_tokens"`
	TierLabel            string   `json:"tier_label"`
	InputPerMTokUSD      *float64 `json:"input_per_mtok_usd"`
	OutputPerMTokUSD     *float64 `json:"output_per_mtok_usd"`
	CacheWritePerMTokUSD *float64 `json:"cache_write_per_mtok_usd"`
	CacheReadPerMTokUSD  *float64 `json:"cache_read_per_mtok_usd"`
	PerRequestUSD        *float64 `json:"per_request_usd"`
	PerImageUSD          *float64 `json:"per_image_usd"`
}

type ModelCatalogGroupCompanion struct {
	Group               ModelCatalogGroupRef   `json:"group"`
	EffectivePricingUSD ModelCatalogPricing    `json:"effective_pricing_usd"`
	Comparison          ModelCatalogComparison `json:"comparison"`
}

type modelCatalogEntry struct {
	item           ModelCatalogItem
	modelKey       string
	primarySavings *float64
	primaryInput   *float64
}

func (s *ModelCatalogService) GetCatalog(ctx context.Context, userID int64) (*ModelCatalogResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}

	groups, err := s.accessService.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get available groups: %w", err)
	}

	userRates, err := s.accessService.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user group rates: %w", err)
	}

	return s.buildCatalog(ctx, groups, userRates), nil
}

// GetPublicCatalog 构建面向匿名访客的模型定价目录。
//
// 与 GetCatalog 的区别只有分组来源和倍率来源：这里取「公开分组」（活跃、非专属、
// 非免费订阅，见 ListPublicGroups），且不带任何用户专属倍率——传 nil userRates 后
// resolveCatalogRate 会回落到分组默认倍率并把 RateSource 标为 group_default。
//
// 刻意不加缓存：渠道定价读路径本身走 ChannelService 的进程内缓存（CRUD 后立即重建），
// 分组则直接查库，因此后台调价能即时反映到公开定价页。加缓存会破坏这个实时性。
func (s *ModelCatalogService) GetPublicCatalog(ctx context.Context) (*ModelCatalogResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}

	groups, err := s.accessService.ListPublicGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("list public groups: %w", err)
	}

	return s.buildCatalog(ctx, groups, nil), nil
}

func (s *ModelCatalogService) ensureConfigured() error {
	if s == nil || s.accessService == nil || s.modelService == nil || s.billing == nil || s.resolver == nil {
		return fmt.Errorf("model catalog service is not configured")
	}
	return nil
}

// buildCatalog 是目录构建的纯计算部分：给定分组集合与倍率覆盖表，产出条目与汇总。
// 不感知「当前用户是谁」，因此登录态目录与公开目录可以共用同一套排序、分桶与对比逻辑。
// userRates 为 nil 时全部回落到分组默认倍率。
func (s *ModelCatalogService) buildCatalog(
	ctx context.Context,
	groups []Group,
	userRates map[int64]float64,
) *ModelCatalogResponse {
	// entries 与 modelBuckets 必须共享同一批 *modelCatalogEntry：
	// 若 buckets 里存 &entries[i]，entries 扩容重分配后这些指针会指向废弃的
	// 旧底层数组，后面写入的 AvailableGroupCount / OtherGroups 全部丢失。
	entries := make([]*modelCatalogEntry, 0)
	modelBuckets := make(map[string][]*modelCatalogEntry)

	for _, group := range groups {
		groupID := group.ID
		models := s.modelService.GetAvailableModels(ctx, &groupID, "")
		if len(models) == 0 {
			continue
		}
		sort.Strings(models)

		for _, model := range models {
			entry, ok := s.buildEntry(ctx, group, userRates, model)
			if !ok {
				continue
			}
			entries = append(entries, &entry)
			modelKey := strings.ToLower(strings.TrimSpace(model))
			modelBuckets[modelKey] = append(modelBuckets[modelKey], &entry)
		}
	}

	for _, bucket := range modelBuckets {
		sort.Slice(bucket, func(i, j int) bool {
			return compareCatalogEntries(bucket[i], bucket[j])
		})
		count := len(bucket)
		for idx, entry := range bucket {
			entry.item.AvailableGroupCount = count
			if idx == 0 {
				continue
			}
		}
		for _, entry := range bucket {
			entry.item.OtherGroups = buildOtherGroups(entry, bucket)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return compareCatalogEntries(entries[i], entries[j])
	})

	items := make([]ModelCatalogItem, 0, len(entries))
	summary := ModelCatalogSummary{}
	bestSavings := -1.0
	for _, entry := range entries {
		items = append(items, entry.item)
		summary.TotalModels++
		if entry.item.BillingMode == string(BillingModeToken) || entry.item.BillingMode == "" {
			summary.TokenModels++
		} else {
			summary.NonTokenModels++
		}
		if entry.primarySavings != nil && *entry.primarySavings > bestSavings {
			bestSavings = *entry.primarySavings
			summary.MaxSavingsPercent = *entry.primarySavings
			summary.BestSavingsModel = fmt.Sprintf("%s / %s", entry.item.BestGroup.Name, entry.item.Model)
		}
	}
	if bestSavings < 0 {
		summary.MaxSavingsPercent = 0
	}

	return &ModelCatalogResponse{
		Items:   items,
		Summary: summary,
	}
}

func (s *ModelCatalogService) buildEntry(
	ctx context.Context,
	group Group,
	userRates map[int64]float64,
	model string,
) (modelCatalogEntry, bool) {
	rateMultiplier, rateSource := resolveCatalogRate(group, userRates)

	officialPricing, officialBase, officialSource := s.buildOfficialPricing(model, group)
	resolved := s.resolver.Resolve(ctx, PricingInput{
		Model:   model,
		GroupID: &group.ID,
	})
	if resolved == nil {
		return modelCatalogEntry{}, false
	}

	// 图片/视频可以配独立倍率，扣费时用的是它而不是分组文本倍率；展示价必须同源。
	rateMultiplier = resolveCatalogModeRate(resolved.Mode, group, rateMultiplier)

	effectivePricing := s.buildEffectivePricingFromResolved(model, resolved, rateMultiplier, group)
	comparison, primarySavings, primaryInput := buildCatalogComparison(officialPricing, effectivePricing, resolved.Mode)

	rawPricing := s.getRawPricing(model)
	details := s.buildCatalogPricingDetails(model, rawPricing, officialBase, resolved, rateMultiplier, group)

	item := ModelCatalogItem{
		Model:       model,
		DisplayName: model,
		Platform:    group.Platform,
		BillingMode: normalizedCatalogBillingMode(resolved.Mode),
		BestGroup: ModelCatalogGroupRef{
			ID:             group.ID,
			Name:           group.Name,
			RateMultiplier: rateMultiplier,
			RateSource:     rateSource,
		},
		AvailableGroupCount: 1,
		OfficialPricing:     officialPricing.withSource(officialSource),
		EffectivePricingUSD: effectivePricing,
		Comparison:          comparison,
		PricingDetails:      details,
		OtherGroups:         []ModelCatalogGroupCompanion{},
		ContextWindow:       catalogContextWindow(rawPricing),
	}

	return modelCatalogEntry{
		item:           item,
		modelKey:       strings.ToLower(strings.TrimSpace(model)),
		primarySavings: primarySavings,
		primaryInput:   primaryInput,
	}, true
}

func (s *ModelCatalogService) buildOfficialPricing(model string, group Group) (ModelCatalogPricing, *ModelPricing, string) {
	basePricing, source := s.resolveOfficialBasePricing(model)
	pricing := ModelCatalogPricing{}

	if basePricing != nil {
		pricing.InputPerMTokUSD = mtokPtr(basePricing.InputPricePerToken)
		pricing.OutputPerMTokUSD = mtokPtr(basePricing.OutputPricePerToken)
		pricing.CacheWritePerMTokUSD = mtokPtr(basePricing.CacheCreationPricePerToken)
		pricing.CacheReadPerMTokUSD = mtokPtr(basePricing.CacheReadPricePerToken)
	}

	// 参考价按"未指定尺寸时落账的档位"取，与 EffectivePricingUSD 同档，否则
	// savings 会拿 1K 参考价去比 2K 实付价。必须在免费订阅分支之前赋值：免费分组
	// 的图片模型同样要显示参考价，否则"省了多少"无从算起。
	pricing.PerImageUSD = s.officialImageTierPrice(model, defaultImageBillingTier())
	pricing.PerSecondUSD = s.officialVideoTierPrice(model, defaultVideoBillingResolution())

	if group.IsSubscriptionType() && group.IsFreeSubscription() {
		if pricing.InputPerMTokUSD != nil || pricing.OutputPerMTokUSD != nil ||
			pricing.PerImageUSD != nil || pricing.PerSecondUSD != nil {
			pricing.HasReference = true
		}
		return pricing, basePricing, source
	}

	pricing.HasReference = pricing.InputPerMTokUSD != nil ||
		pricing.OutputPerMTokUSD != nil ||
		pricing.CacheWritePerMTokUSD != nil ||
		pricing.CacheReadPerMTokUSD != nil ||
		pricing.PerImageUSD != nil ||
		pricing.PerSecondUSD != nil ||
		pricing.PerRequestUSD != nil

	return pricing, basePricing, source
}

func (s *ModelCatalogService) resolveOfficialBasePricing(model string) (*ModelPricing, string) {
	if s.billing == nil {
		return nil, "none"
	}
	if raw := s.getRawPricing(model); raw != nil {
		pricing, err := s.billing.GetModelPricing(model)
		if err == nil {
			return pricing, PricingSourceLiteLLM
		}
	}
	if fallback := s.billing.getFallbackPricing(strings.ToLower(strings.TrimSpace(model))); fallback != nil {
		return s.billing.applyModelSpecificPricingPolicy(model, fallback), PricingSourceFallback
	}
	return nil, "none"
}

// catalogContextWindow 取模型的上下文窗口，未知时返回 nil。
//
// 只信 max_input_tokens：同一条目里的 long_context_input_token_threshold 是
// 计价分档的边界，不是容量，两者数值相近但含义无关，混用会给出看似合理的错值。
func catalogContextWindow(pricing *LiteLLMModelPricing) *int {
	if pricing == nil || pricing.MaxInputTokens <= 0 {
		return nil
	}
	window := pricing.MaxInputTokens
	return &window
}

func (s *ModelCatalogService) getRawPricing(model string) *LiteLLMModelPricing {
	if s == nil || s.billing == nil || s.billing.pricingService == nil {
		return nil
	}
	return s.billing.pricingService.GetModelPricing(model)
}

func resolveCatalogRate(group Group, userRates map[int64]float64) (float64, string) {
	if userRates != nil {
		if custom, ok := userRates[group.ID]; ok {
			return normalizeCatalogRate(group, custom), "user_override"
		}
	}
	return normalizeCatalogRate(group, group.RateMultiplier), "group_default"
}

// resolveCatalogModeRate 把分组倍率收敛成该计费模式实际生效的倍率。图片/视频
// 分组可以开独立倍率（ImageRateIndependent / VideoRateIndependent），此时扣费
// 走的是 Image/VideoRateMultiplier，目录若仍用文本倍率就会和账单对不上。
func resolveCatalogModeRate(mode BillingMode, group Group, rateMultiplier float64) float64 {
	switch mode {
	case BillingModeImage:
		return resolveImageRateMultiplierForGroup(&group, rateMultiplier)
	case BillingModeVideo:
		return resolveVideoRateMultiplierForGroup(&group, rateMultiplier)
	default:
		return rateMultiplier
	}
}

func normalizeCatalogRate(group Group, rate float64) float64 {
	if rate < 0 {
		return 1
	}
	if rate == 0 && !group.IsFreeSubscription() {
		return 1
	}
	return rate
}

func (s *ModelCatalogService) buildEffectivePricingFromResolved(
	model string,
	resolved *ResolvedPricing,
	rateMultiplier float64,
	group Group,
) ModelCatalogPricing {
	if resolved == nil {
		return ModelCatalogPricing{Source: "none"}
	}
	mode := resolved.Mode
	if mode == "" {
		mode = BillingModeToken
	}
	pricing := ModelCatalogPricing{
		Source: "effective",
	}

	switch mode {
	case BillingModePerRequest:
		if resolved.DefaultPerRequestPrice > 0 {
			pricing.PerRequestUSD = floatPtr(resolved.DefaultPerRequestPrice * rateMultiplier)
		}
	case BillingModeImage:
		// 主价取"未指定尺寸时实际落账的档位"（2K），而不是最便宜的 1K。
		if price := s.imageTierUnitPrice(model, defaultImageBillingTier(), resolved, group); price != nil {
			pricing.PerImageUSD = floatPtr(*price * rateMultiplier)
		}
	case BillingModeVideo:
		// 视频单价是每秒价，总价 = 每秒价 × 时长 × 条数。
		if price := s.videoTierUnitPrice(model, defaultVideoBillingResolution(), resolved, group); price != nil {
			pricing.PerSecondUSD = floatPtr(*price * rateMultiplier)
		}
	default:
		if len(resolved.Intervals) > 0 {
			return pricing
		}
		base := resolved.BasePricing
		if base == nil {
			return pricing
		}
		pricing.InputPerMTokUSD = mtokPtr(base.InputPricePerToken * rateMultiplier)
		pricing.OutputPerMTokUSD = mtokPtr(base.OutputPricePerToken * rateMultiplier)
		pricing.CacheWritePerMTokUSD = mtokPtr(base.CacheCreationPricePerToken * rateMultiplier)
		pricing.CacheReadPerMTokUSD = mtokPtr(base.CacheReadPricePerToken * rateMultiplier)
	}

	return pricing
}

func buildCatalogComparison(
	official ModelCatalogPricing,
	effective ModelCatalogPricing,
	mode BillingMode,
) (ModelCatalogComparison, *float64, *float64) {
	comp := ModelCatalogComparison{}

	comp.DeltaInputPerMTokUSD = deltaPtr(official.InputPerMTokUSD, effective.InputPerMTokUSD)
	comp.DeltaOutputPerMTokUSD = deltaPtr(official.OutputPerMTokUSD, effective.OutputPerMTokUSD)
	comp.DeltaPerRequestUSD = deltaPtr(official.PerRequestUSD, effective.PerRequestUSD)
	comp.DeltaPerImageUSD = deltaPtr(official.PerImageUSD, effective.PerImageUSD)
	comp.DeltaPerSecondUSD = deltaPtr(official.PerSecondUSD, effective.PerSecondUSD)

	switch mode {
	case BillingModePerRequest:
		comp.SavingsPercent = savingsPercentPtr(official.PerRequestUSD, effective.PerRequestUSD)
	case BillingModeImage:
		comp.SavingsPercent = savingsPercentPtr(official.PerImageUSD, effective.PerImageUSD)
	case BillingModeVideo:
		comp.SavingsPercent = savingsPercentPtr(official.PerSecondUSD, effective.PerSecondUSD)
	default:
		comp.SavingsPercent = savingsPercentPtr(official.InputPerMTokUSD, effective.InputPerMTokUSD)
	}

	comp.IsCheaperThanOfficial = comp.SavingsPercent != nil && *comp.SavingsPercent > 0
	return comp, comp.SavingsPercent, effective.InputPerMTokUSD
}

func (s *ModelCatalogService) buildCatalogPricingDetails(
	model string,
	rawPricing *LiteLLMModelPricing,
	officialBase *ModelPricing,
	resolved *ResolvedPricing,
	rateMultiplier float64,
	group Group,
) ModelCatalogPricingDetails {
	details := ModelCatalogPricingDetails{
		MediaTiers: []ModelCatalogMediaTier{},
	}

	if rawPricing != nil {
		details.SupportsPromptCaching = rawPricing.SupportsPromptCaching
	} else if officialBase != nil {
		details.SupportsPromptCaching = officialBase.CacheCreationPricePerToken > 0 || officialBase.CacheReadPricePerToken > 0
	}

	if officialBase != nil {
		details.LongContextInputThreshold = officialBase.LongContextInputThreshold
		details.HasLongContextMultiplier = officialBase.LongContextInputThreshold > 0 &&
			(officialBase.LongContextInputMultiplier > 1 || officialBase.LongContextOutputMultiplier > 1)
	}

	if resolved != nil {
		switch resolved.Mode {
		case BillingModePerRequest:
			details.Intervals = buildIntervals(resolved.RequestTiers, resolved.Mode, group)
		case BillingModeImage:
			details.Intervals = buildIntervals(resolved.RequestTiers, resolved.Mode, group)
			details.MediaTiers = s.buildImageMediaTiers(model, resolved, rateMultiplier, group)
			details.MediaUnit = MediaPriceUnitImage
		case BillingModeVideo:
			details.Intervals = buildIntervals(resolved.RequestTiers, resolved.Mode, group)
			details.MediaTiers = s.buildVideoMediaTiers(model, resolved, rateMultiplier, group)
			details.MediaUnit = MediaPriceUnitSecond
		default:
			details.Intervals = buildIntervals(resolved.Intervals, resolved.Mode, group)
		}
	}

	return details
}

func buildIntervals(intervals []PricingInterval, mode BillingMode, group Group) []ModelCatalogPriceInterval {
	if len(intervals) == 0 {
		return []ModelCatalogPriceInterval{}
	}
	out := make([]ModelCatalogPriceInterval, 0, len(intervals))
	for _, iv := range intervals {
		item := ModelCatalogPriceInterval{
			MinTokens: iv.MinTokens,
			MaxTokens: iv.MaxTokens,
			TierLabel: iv.TierLabel,
		}
		switch mode {
		case BillingModePerRequest:
			item.PerRequestUSD = copyFloatPtr(iv.PerRequestPrice)
		case BillingModeImage:
			if iv.PerRequestPrice != nil {
				item.PerImageUSD = copyFloatPtr(iv.PerRequestPrice)
			} else if groupPrice := group.GetImagePrice(iv.TierLabel); groupPrice != nil {
				item.PerImageUSD = copyFloatPtr(groupPrice)
			}
		default:
			item.InputPerMTokUSD = mtokPtrValue(iv.InputPrice)
			item.OutputPerMTokUSD = mtokPtrValue(iv.OutputPrice)
			item.CacheWritePerMTokUSD = mtokPtrValue(iv.CacheWritePrice)
			item.CacheReadPerMTokUSD = mtokPtrValue(iv.CacheReadPrice)
		}
		out = append(out, item)
	}
	return out
}

func buildOtherGroups(current *modelCatalogEntry, bucket []*modelCatalogEntry) []ModelCatalogGroupCompanion {
	others := make([]ModelCatalogGroupCompanion, 0, 2)
	for _, candidate := range bucket {
		if candidate == current {
			continue
		}
		others = append(others, ModelCatalogGroupCompanion{
			Group:               candidate.item.BestGroup,
			EffectivePricingUSD: candidate.item.EffectivePricingUSD,
			Comparison:          candidate.item.Comparison,
		})
		if len(others) == 2 {
			break
		}
	}
	return others
}

func compareCatalogEntries(a, b *modelCatalogEntry) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	if a.primarySavings != nil && b.primarySavings != nil && *a.primarySavings != *b.primarySavings {
		return *a.primarySavings > *b.primarySavings
	}
	if a.primarySavings != nil && b.primarySavings == nil {
		return true
	}
	if a.primarySavings == nil && b.primarySavings != nil {
		return false
	}
	if a.primaryInput != nil && b.primaryInput != nil && *a.primaryInput != *b.primaryInput {
		return *a.primaryInput < *b.primaryInput
	}
	if a.primaryInput != nil && b.primaryInput == nil {
		return true
	}
	if a.primaryInput == nil && b.primaryInput != nil {
		return false
	}
	if a.item.BestGroup.Name != b.item.BestGroup.Name {
		return a.item.BestGroup.Name < b.item.BestGroup.Name
	}
	return a.item.Model < b.item.Model
}

func normalizedCatalogBillingMode(mode BillingMode) string {
	if mode == "" {
		return string(BillingModeToken)
	}
	return string(mode)
}

func mtokPtr(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	converted := value * mtokMultiplier
	return &converted
}

func mtokPtrValue(value *float64) *float64 {
	if value == nil || *value <= 0 {
		return nil
	}
	converted := *value * mtokMultiplier
	return &converted
}

func floatPtr(value float64) *float64 {
	return &value
}

func copyFloatPtr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func deltaPtr(official, effective *float64) *float64 {
	if official == nil || effective == nil {
		return nil
	}
	delta := *official - *effective
	return &delta
}

func savingsPercentPtr(official, effective *float64) *float64 {
	if official == nil || effective == nil || *official <= 0 {
		return nil
	}
	value := 1 - (*effective / *official)
	return &value
}

func (p ModelCatalogPricing) withSource(source string) ModelCatalogPricing {
	p.Source = source
	if source == "" {
		p.Source = "none"
	}
	return p
}
