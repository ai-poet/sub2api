package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PublicPricingHandler 处理落地页的**匿名**定价目录查询。
//
// 与 ModelCatalogHandler 的关键差异：
//  1. 无认证。调用方是未登录访客，因此分组来源是 ListPublicGroups（活跃 + 非专属 +
//     非免费订阅），倍率一律取分组默认值，不存在用户专属倍率。
//  2. 字段白名单。相比登录态目录，额外剥掉分组内部 ID 与 other_groups，避免向匿名
//     调用方泄露内部标识与分组结构，参考 available_channel_handler.go 的既有做法。
//  3. **永不返回 401**。前端共享的 apiClient 在收到任何 401 时会强制跳转 /login，
//     匿名访客在落地页被踢走是不可接受的。因此开关关闭或出错时一律返回 200 + 空列表，
//     让落地页优雅回落到静态文案。
type PublicPricingHandler struct {
	modelCatalogService *service.ModelCatalogService
	settingService      *service.SettingService
}

// NewPublicPricingHandler 创建公开定价 handler。
func NewPublicPricingHandler(
	modelCatalogService *service.ModelCatalogService,
	settingService *service.SettingService,
) *PublicPricingHandler {
	return &PublicPricingHandler{
		modelCatalogService: modelCatalogService,
		settingService:      settingService,
	}
}

// featureEnabled 返回 public-pricing 开关是否启用。默认开启（opt-out）。
func (h *PublicPricingHandler) featureEnabled(c *gin.Context) bool {
	if h.settingService == nil {
		return false
	}
	return h.settingService.GetPublicPricingRuntime(c.Request.Context()).Enabled
}

// publicPricingGroupRef 公开可见的分组引用（去掉内部 ID）。
type publicPricingGroupRef struct {
	Name           string  `json:"name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	RateSource     string  `json:"rate_source"`
}

// publicPricing 公开可见的价格字段。字段与登录态目录一致，单位均为 USD。
type publicPricing struct {
	InputPerMTokUSD      *float64 `json:"input_per_mtok_usd"`
	OutputPerMTokUSD     *float64 `json:"output_per_mtok_usd"`
	CacheWritePerMTokUSD *float64 `json:"cache_write_per_mtok_usd"`
	CacheReadPerMTokUSD  *float64 `json:"cache_read_per_mtok_usd"`
	PerRequestUSD        *float64 `json:"per_request_usd"`
	PerImageUSD          *float64 `json:"per_image_usd"`
	HasReference         bool     `json:"has_reference"`
}

// publicPricingComparison 与官方参考价的对比。
//
// SavingsPercent 只在官方参考价确实存在时下发（见 toPublicItem）——「比官方省 X%」
// 是对外的营销声明，没有参考价时不应凭空造出一个对比数字。
type publicPricingComparison struct {
	SavingsPercent        *float64 `json:"savings_percent"`
	IsCheaperThanOfficial bool     `json:"is_cheaper_than_official"`
}

// publicPricingItem 单个模型的公开定价条目。
type publicPricingItem struct {
	Model               string                  `json:"model"`
	DisplayName         string                  `json:"display_name"`
	Platform            string                  `json:"platform"`
	BillingMode         string                  `json:"billing_mode"`
	Group               publicPricingGroupRef   `json:"group"`
	AvailableGroupCount int                     `json:"available_group_count"`
	OfficialPricing     publicPricing           `json:"official_pricing"`
	EffectivePricingUSD publicPricing           `json:"effective_pricing_usd"`
	Comparison          publicPricingComparison `json:"comparison"`
}

// publicPricingSummary 汇总信息，供落地页做「最高省 X%」这类标题展示。
type publicPricingSummary struct {
	TotalModels       int     `json:"total_models"`
	MaxSavingsPercent float64 `json:"max_savings_percent"`
}

// publicPricingResponse 公开定价目录响应。
type publicPricingResponse struct {
	Enabled bool                 `json:"enabled"`
	Items   []publicPricingItem  `json:"items"`
	Summary publicPricingSummary `json:"summary"`
}

// emptyPublicPricingResponse 构造「无数据」响应。items 恒为非 nil 空数组，
// 让前端可以直接 .length 判断而不必处理 null。
func emptyPublicPricingResponse(enabled bool) publicPricingResponse {
	return publicPricingResponse{
		Enabled: enabled,
		Items:   []publicPricingItem{},
		Summary: publicPricingSummary{},
	}
}

// List 返回公开定价目录。
// GET /api/v1/pricing/public
func (h *PublicPricingHandler) List(c *gin.Context) {
	if !h.featureEnabled(c) {
		response.Success(c, emptyPublicPricingResponse(false))
		return
	}

	catalog, err := h.modelCatalogService.GetPublicCatalog(c.Request.Context())
	if err != nil {
		// 匿名接口不把内部错误暴露成 5xx——落地页宁可回落到静态文案，也不该报错。
		response.Success(c, emptyPublicPricingResponse(true))
		return
	}

	items := make([]publicPricingItem, 0, len(catalog.Items))
	for i := range catalog.Items {
		items = append(items, toPublicPricingItem(catalog.Items[i]))
	}

	response.Success(c, publicPricingResponse{
		Enabled: true,
		Items:   items,
		Summary: publicPricingSummary{
			TotalModels:       catalog.Summary.TotalModels,
			MaxSavingsPercent: catalog.Summary.MaxSavingsPercent,
		},
	})
}

// toPublicPricingItem 将内部目录条目收敛为匿名可见的白名单条目。
func toPublicPricingItem(src service.ModelCatalogItem) publicPricingItem {
	official := toPublicPricing(src.OfficialPricing)

	comparison := publicPricingComparison{
		IsCheaperThanOfficial: src.Comparison.IsCheaperThanOfficial,
	}
	// 只有存在官方参考价时才下发省钱比例，避免对外做无依据的对比声明。
	if official.HasReference {
		comparison.SavingsPercent = src.Comparison.SavingsPercent
	}

	return publicPricingItem{
		Model:       src.Model,
		DisplayName: src.DisplayName,
		Platform:    src.Platform,
		BillingMode: src.BillingMode,
		Group: publicPricingGroupRef{
			Name:           src.BestGroup.Name,
			RateMultiplier: src.BestGroup.RateMultiplier,
			RateSource:     src.BestGroup.RateSource,
		},
		AvailableGroupCount: src.AvailableGroupCount,
		OfficialPricing:     official,
		EffectivePricingUSD: toPublicPricing(src.EffectivePricingUSD),
		Comparison:          comparison,
	}
}

// toPublicPricing 复制价格字段（Source 等内部来源标记不对外暴露）。
func toPublicPricing(src service.ModelCatalogPricing) publicPricing {
	return publicPricing{
		InputPerMTokUSD:      src.InputPerMTokUSD,
		OutputPerMTokUSD:     src.OutputPerMTokUSD,
		CacheWritePerMTokUSD: src.CacheWritePerMTokUSD,
		CacheReadPerMTokUSD:  src.CacheReadPerMTokUSD,
		PerRequestUSD:        src.PerRequestUSD,
		PerImageUSD:          src.PerImageUSD,
		HasReference:         src.HasReference,
	}
}
