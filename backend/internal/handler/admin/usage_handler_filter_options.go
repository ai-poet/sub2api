package admin

import (
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"

	"github.com/gin-gonic/gin"
)

// 调用日志页筛选项的最小投影接口（fork 本地，运维管理员角色的一部分，见 docs/OPERATOR_ROLE.md）。
//
// operator 无权访问 /admin/groups、/admin/accounts 和 /admin/dashboard/*，而筛选下拉只需要
// id / name / platform 这几个已经出现在日志行里的字段。三个接口都是 GET、无副作用，
// 返回体刻意不带任何配置、凭证、倍率、余额或统计数字，管理员与 operator 看到的内容相同。

// UsageFilterGroup 分组下拉项。
type UsageFilterGroup struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	// 下面三个是分组的非敏感元数据：运维管理员无权访问 /admin/groups，
	// 用户 / 订阅管理页的分组下拉（只显示活跃的标准 / 订阅分组）靠它们过滤。
	Status           string `json:"status"`
	SubscriptionType string `json:"subscription_type"`
	IsExclusive      bool   `json:"is_exclusive"`
}

// UsageFilterAccount 账号下拉项。
type UsageFilterAccount struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

const (
	usageFilterAccountSearchLimit = 20
	usageFilterModelsMaxRangeDays = 366
)

// FilterGroups GET /admin/usage/filter-groups
// 返回全部分组（含已停用，历史日志可能仍引用）的 id / name / platform 与状态 / 订阅类型 / 是否专属，按名称排序。
func (h *UsageHandler) FilterGroups(c *gin.Context) {
	groups, err := h.adminService.GetAllGroupsIncludingInactive(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result := make([]UsageFilterGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, UsageFilterGroup{
			ID:               g.ID,
			Name:             g.Name,
			Platform:         g.Platform,
			Status:           g.Status,
			SubscriptionType: g.SubscriptionType,
			IsExclusive:      g.IsExclusive,
		})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	response.Success(c, result)
}

// SearchAccounts GET /admin/usage/search-accounts?q=
// 按名称搜索账号，只返回 id / name / platform，最多 20 条；空关键字返回空数组。
func (h *UsageHandler) SearchAccounts(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		response.Success(c, []UsageFilterAccount{})
		return
	}
	accounts, _, err := h.adminService.ListAccounts(c.Request.Context(), 1, usageFilterAccountSearchLimit, "", "", "", keyword, 0, "", "name", "asc")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result := make([]UsageFilterAccount, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, UsageFilterAccount{ID: a.ID, Name: a.Name, Platform: a.Platform})
	}
	response.Success(c, result)
}

// FilterModels GET /admin/usage/filter-models?start_date=&end_date=&timezone=
// 返回时间范围内出现过的请求模型名（去重、排序），不带任何统计数字。
// 日期语义与 Stats 一致：闭区间日期，上边界取 end_date 次日 00:00；缺省为最近 24 小时。
func (h *UsageHandler) FilterModels(c *gin.Context) {
	userTZ := c.Query("timezone")
	now := timezone.NowInUserLocation(userTZ)
	startDateStr := strings.TrimSpace(c.Query("start_date"))
	endDateStr := strings.TrimSpace(c.Query("end_date"))

	var startTime, endTime time.Time
	if startDateStr != "" && endDateStr != "" {
		var err error
		startTime, err = timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		endTime, err = timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		endTime = endTime.AddDate(0, 0, 1)
	} else {
		startTime = now.AddDate(0, 0, -1)
		endTime = now
	}
	if !endTime.After(startTime) {
		response.BadRequest(c, "end_date must not be before start_date")
		return
	}
	if endTime.Sub(startTime) > usageFilterModelsMaxRangeDays*24*time.Hour {
		response.BadRequest(c, "Date range too large")
		return
	}

	stats, err := h.usageService.GetModelStatsWithFiltersBySource(c.Request.Context(), startTime, endTime, usagestats.UsageLogFilters{}, usagestats.ModelSourceRequested)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	seen := make(map[string]struct{}, len(stats))
	models := make([]string, 0, len(stats))
	for _, stat := range stats {
		name := strings.TrimSpace(stat.Model)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}
	sort.Strings(models)
	response.Success(c, models)
}
