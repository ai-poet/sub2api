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
}

type ModelCatalogPricingDetails struct {
	SupportsPromptCaching     bool                        `json:"supports_prompt_caching"`
	HasLongContextMultiplier  bool                        `json:"has_long_context_multiplier"`
	LongContextInputThreshold int                         `json:"long_context_input_threshold"`
	Intervals                 []ModelCatalogPriceInterval `json:"intervals"`
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
	if s == nil || s.accessService == nil || s.modelService == nil || s.billing == nil || s.resolver == nil {
		return nil, fmt.Errorf("model catalog service is not configured")
	}

	groups, err := s.accessService.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get available groups: %w", err)
	}

	userRates, err := s.accessService.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user group rates: %w", err)
	}

	entries := make([]modelCatalogEntry, 0)
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
			entries = append(entries, entry)
			modelKey := strings.ToLower(strings.TrimSpace(model))
			modelBuckets[modelKey] = append(modelBuckets[modelKey], &entries[len(entries)-1])
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
		return compareCatalogEntries(&entries[i], &entries[j])
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
	}, nil
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

	effectivePricing := buildEffectivePricingFromResolved(resolved, officialBase, rateMultiplier, group)
	comparison, primarySavings, primaryInput := buildCatalogComparison(officialPricing, effectivePricing, resolved.Mode)

	rawPricing := s.getRawPricing(model)
	details := buildCatalogPricingDetails(rawPricing, officialBase, resolved, group)

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

	if group.IsSubscriptionType() && group.IsFreeSubscription() {
		if pricing.InputPerMTokUSD != nil || pricing.OutputPerMTokUSD != nil {
			pricing.HasReference = true
		}
		return pricing, basePricing, source
	}

	rawPricing := s.getRawPricing(model)
	if rawPricing != nil && rawPricing.OutputCostPerImage > 0 {
		pricing.PerImageUSD = floatPtr(rawPricing.OutputCostPerImage)
	}

	pricing.HasReference = pricing.InputPerMTokUSD != nil ||
		pricing.OutputPerMTokUSD != nil ||
		pricing.CacheWritePerMTokUSD != nil ||
		pricing.CacheReadPerMTokUSD != nil ||
		pricing.PerImageUSD != nil ||
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

func normalizeCatalogRate(group Group, rate float64) float64 {
	if rate < 0 {
		return 1
	}
	if rate == 0 && !group.IsFreeSubscription() {
		return 1
	}
	return rate
}

func buildEffectivePricingFromResolved(
	resolved *ResolvedPricing,
	officialBase *ModelPricing,
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
		if resolved.DefaultPerRequestPrice > 0 {
			pricing.PerImageUSD = floatPtr(resolved.DefaultPerRequestPrice * rateMultiplier)
		} else if group.GetImagePrice("1K") != nil {
			pricing.PerImageUSD = floatPtr(*group.GetImagePrice("1K") * rateMultiplier)
		} else if officialBase != nil && officialBase.ImageOutputPricePerToken > 0 {
			pricing.PerImageUSD = floatPtr(officialBase.ImageOutputPricePerToken * rateMultiplier)
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

	switch mode {
	case BillingModePerRequest:
		comp.SavingsPercent = savingsPercentPtr(official.PerRequestUSD, effective.PerRequestUSD)
	case BillingModeImage:
		comp.SavingsPercent = savingsPercentPtr(official.PerImageUSD, effective.PerImageUSD)
	default:
		comp.SavingsPercent = savingsPercentPtr(official.InputPerMTokUSD, effective.InputPerMTokUSD)
	}

	comp.IsCheaperThanOfficial = comp.SavingsPercent != nil && *comp.SavingsPercent > 0
	return comp, comp.SavingsPercent, effective.InputPerMTokUSD
}

func buildCatalogPricingDetails(
	rawPricing *LiteLLMModelPricing,
	officialBase *ModelPricing,
	resolved *ResolvedPricing,
	group Group,
) ModelCatalogPricingDetails {
	details := ModelCatalogPricingDetails{}

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
