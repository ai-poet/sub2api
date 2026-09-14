package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 调用日志页图表数据（fork 本地，运维管理员角色的一部分，见 docs/OPERATOR_ROLE.md）。
//
// 管理员的调用日志页图表走 /admin/dashboard/{snapshot-v2,models}，而 dashboard 域对 operator 永久关闭
// （里面还有收入、用户排行等）。这里在 usage 域下提供同样口径的趋势 / 分组 / 模型聚合：
// 筛选参数与 Stats 一致，日期语义与 dashboard 一致；operator 请求时抹掉账号侧成本（account_cost），
// 与调用日志的 operator 投影保持一致。不返回任何用户级明细。

const usageChartsMaxRangeDays = 366

// parseUsageChartFilters 解析与 Stats 相同的筛选参数；非法值直接写 400 并返回 false。
func parseUsageChartFilters(c *gin.Context) (usagestats.UsageLogFilters, bool) {
	var filters usagestats.UsageLogFilters

	parseID := func(name string) (int64, bool) {
		raw := strings.TrimSpace(c.Query(name))
		if raw == "" {
			return 0, true
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid "+name)
			return 0, false
		}
		return id, true
	}
	var ok bool
	if filters.UserID, ok = parseID("user_id"); !ok {
		return filters, false
	}
	if filters.APIKeyID, ok = parseID("api_key_id"); !ok {
		return filters, false
	}
	if filters.AccountID, ok = parseID("account_id"); !ok {
		return filters, false
	}
	if filters.GroupID, ok = parseID("group_id"); !ok {
		return filters, false
	}

	filters.Model = strings.TrimSpace(c.Query("model"))
	filters.ModelFilterSource = usagestats.ModelSourceRequested
	filters.BillingMode = strings.TrimSpace(c.Query("billing_mode"))

	if requestTypeStr := strings.TrimSpace(c.Query("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.BadRequest(c, err.Error())
			return filters, false
		}
		value := int16(parsed)
		filters.RequestType = &value
	} else if streamStr := strings.TrimSpace(c.Query("stream")); streamStr != "" {
		val, err := strconv.ParseBool(streamStr)
		if err != nil {
			response.BadRequest(c, "Invalid stream value, use true or false")
			return filters, false
		}
		filters.Stream = &val
	}

	nativeCompactionV2, err := parseOptionalBoolDashboardFilter(c, "native_compaction_v2")
	if err != nil {
		response.BadRequest(c, "Invalid native_compaction_v2 value, use true or false")
		return filters, false
	}
	filters.NativeCompactionV2 = nativeCompactionV2

	if billingTypeStr := strings.TrimSpace(c.Query("billing_type")); billingTypeStr != "" {
		val, err := strconv.ParseInt(billingTypeStr, 10, 8)
		if err != nil {
			response.BadRequest(c, "Invalid billing_type")
			return filters, false
		}
		bt := int8(val)
		filters.BillingType = &bt
	}

	upstreamModelMismatch, err := parseOptionalBoolDashboardFilter(c, "upstream_model_mismatch")
	if err != nil {
		response.BadRequest(c, "Invalid upstream_model_mismatch value, use true or false")
		return filters, false
	}
	filters.UpstreamModelMismatch = upstreamModelMismatch

	return filters, true
}

// parseUsageChartTimeRange 复用 dashboard 的日期口径（闭区间日期，上边界为 end_date 次日 00:00），
// 并限制最大跨度，避免运维账号发起超大聚合查询。
func parseUsageChartTimeRange(c *gin.Context) (time.Time, time.Time, bool) {
	startTime, endTime := parseTimeRange(c)
	if !endTime.After(startTime) {
		response.BadRequest(c, "end_date must not be before start_date")
		return startTime, endTime, false
	}
	if endTime.Sub(startTime) > usageChartsMaxRangeDays*24*time.Hour {
		response.BadRequest(c, "Date range too large")
		return startTime, endTime, false
	}
	return startTime, endTime, true
}

func usageGroupStatsForViewer(c *gin.Context, stats []usagestats.GroupStat) []usagestats.GroupStat {
	out := make([]usagestats.GroupStat, 0, len(stats))
	operator := middleware.IsOperatorRequest(c)
	for _, s := range stats {
		if operator {
			s.AccountCost = 0
		}
		out = append(out, s)
	}
	return out
}

func usageModelStatsForViewer(c *gin.Context, stats []usagestats.ModelStat) []usagestats.ModelStat {
	out := make([]usagestats.ModelStat, 0, len(stats))
	operator := middleware.IsOperatorRequest(c)
	for _, s := range stats {
		if operator {
			s.AccountCost = 0
		}
		out = append(out, s)
	}
	return out
}

// Charts GET /admin/usage/charts?start_date&end_date&granularity&timezone&<Stats 同款筛选>
// 返回 Token 使用趋势与分组分布，形状与 dashboard snapshot-v2 的 trend / groups 一致。
func (h *UsageHandler) Charts(c *gin.Context) {
	filters, ok := parseUsageChartFilters(c)
	if !ok {
		return
	}
	granularity := strings.TrimSpace(c.DefaultQuery("granularity", "day"))
	if granularity != "day" && granularity != "hour" {
		response.BadRequest(c, "Invalid granularity, use day or hour")
		return
	}
	startTime, endTime, ok := parseUsageChartTimeRange(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	trend, err := h.usageService.GetUsageTrendWithFilters(ctx, startTime, endTime, granularity, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groups, err := h.usageService.GetGroupStatsWithFilters(ctx, startTime, endTime, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if trend == nil {
		trend = []usagestats.TrendDataPoint{}
	}

	response.Success(c, gin.H{
		"trend":       trend,
		"groups":      usageGroupStatsForViewer(c, groups),
		"start_date":  startTime.Format("2006-01-02"),
		"end_date":    endTime.Add(-24 * time.Hour).Format("2006-01-02"),
		"granularity": granularity,
	})
}

// ModelStats GET /admin/usage/model-stats?model_source=requested|upstream|mapping&<Stats 同款筛选>
// 返回模型分布，形状与 dashboard models 一致。与 dashboard 保持一致：model 筛选不作用于模型分布本身。
func (h *UsageHandler) ModelStats(c *gin.Context) {
	filters, ok := parseUsageChartFilters(c)
	if !ok {
		return
	}
	filters.Model = ""

	modelSource := usagestats.ModelSourceRequested
	if raw := strings.TrimSpace(c.Query("model_source")); raw != "" {
		if !usagestats.IsValidModelSource(raw) {
			response.BadRequest(c, "Invalid model_source, use requested/upstream/mapping")
			return
		}
		modelSource = raw
	}
	startTime, endTime, ok := parseUsageChartTimeRange(c)
	if !ok {
		return
	}

	stats, err := h.usageService.GetModelStatsWithFiltersBySource(c.Request.Context(), startTime, endTime, filters, modelSource)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"models":       usageModelStatsForViewer(c, stats),
		"start_date":   startTime.Format("2006-01-02"),
		"end_date":     endTime.Add(-24 * time.Hour).Format("2006-01-02"),
		"model_source": modelSource,
	})
}
