//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 落地页定价区是匿名可见的，而前端共享的 apiClient 收到任何 401 都会强制跳转 /login。
// 因此这个接口在「未认证」这条路径上必须返回 200 + 空列表，绝不能是 401——
// 否则访客一打开首页就被踢到登录页。
func TestPublicPricing_NeverReturns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// settingService 为 nil → featureEnabled 返回 false，走的正是「开关关闭」分支。
	h := &PublicPricingHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/pricing/public", nil)

	h.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotEqual(t, http.StatusUnauthorized, w.Code)
}

// 开关关闭时返回 enabled=false 且 items 为**非 nil 空数组**，
// 让前端可以直接判断 length 而不必处理 null。
func TestPublicPricing_DisabledReturnsEmptyArrayNotNull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PublicPricingHandler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/pricing/public", nil)

	h.List(c)

	var body struct {
		Data publicPricingResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.False(t, body.Data.Enabled)
	require.NotNil(t, body.Data.Items)
	require.Empty(t, body.Data.Items)

	// items 必须序列化成 [] 而不是 null
	raw, err := json.Marshal(emptyPublicPricingResponse(false))
	require.NoError(t, err)
	require.Contains(t, string(raw), `"items":[]`)
}

// 公开 DTO 不得携带分组内部 ID、other_groups 或内部定价来源标记。
func TestPublicPricing_FieldWhitelist(t *testing.T) {
	item := toPublicPricingItem(service.ModelCatalogItem{
		Model:       "claude-sonnet-4-6",
		DisplayName: "claude-sonnet-4-6",
		Platform:    "anthropic",
		BillingMode: "token",
		BestGroup: service.ModelCatalogGroupRef{
			ID:             42,
			Name:           "public-group",
			RateMultiplier: 0.5,
			RateSource:     "group_default",
		},
		OtherGroups: []service.ModelCatalogGroupCompanion{
			{Group: service.ModelCatalogGroupRef{ID: 99, Name: "leaky"}},
		},
		OfficialPricing: service.ModelCatalogPricing{Source: "litellm", HasReference: true},
	})

	raw, err := json.Marshal(item)
	require.NoError(t, err)
	s := string(raw)

	require.Contains(t, s, "public-group")
	require.Contains(t, s, `"rate_multiplier":0.5`)
	// 内部分组 ID / 其它分组 / 定价来源都不应出现在匿名响应里
	require.NotContains(t, s, `"id"`)
	require.NotContains(t, s, "other_groups")
	require.NotContains(t, s, "leaky")
	require.NotContains(t, s, `"source"`)
}

// 没有官方参考价时不得下发 savings_percent——「比官方省 X%」是对外声明，
// 缺少参考价就不应该凭空造一个对比数字出来。
func TestPublicPricing_SavingsSuppressedWithoutOfficialReference(t *testing.T) {
	savings := 42.0

	withRef := toPublicPricingItem(service.ModelCatalogItem{
		OfficialPricing: service.ModelCatalogPricing{HasReference: true},
		Comparison: service.ModelCatalogComparison{
			SavingsPercent:        &savings,
			IsCheaperThanOfficial: true,
		},
	})
	require.NotNil(t, withRef.Comparison.SavingsPercent)
	require.InDelta(t, 42.0, *withRef.Comparison.SavingsPercent, 0.001)

	withoutRef := toPublicPricingItem(service.ModelCatalogItem{
		OfficialPricing: service.ModelCatalogPricing{HasReference: false},
		Comparison: service.ModelCatalogComparison{
			SavingsPercent:        &savings,
			IsCheaperThanOfficial: true,
		},
	})
	require.Nil(t, withoutRef.Comparison.SavingsPercent)
}
