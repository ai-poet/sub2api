package admin

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserGroupRateHandler 专属倍率用户管理（fork 本地功能）。
//
// 路由都挂在 /admin/users 下：运维的三个 GET 在 operatorReadScope 直接放行；
// 批量写在 operatorApprovalScope，被截获成审批，管理员批准后以管理员身份重放到这里。
type UserGroupRateHandler struct {
	svc *service.UserGroupRateAdminService
}

// NewUserGroupRateHandler 构造专属倍率管理 handler。
func NewUserGroupRateHandler(svc *service.UserGroupRateAdminService) *UserGroupRateHandler {
	return &UserGroupRateHandler{svc: svc}
}

// UserGroupRateRowDTO 列表行。
type UserGroupRateRowDTO struct {
	UserID              int64     `json:"user_id"`
	UserEmail           string    `json:"user_email"`
	Username            string    `json:"username"`
	UserNotes           string    `json:"user_notes"`
	UserStatus          string    `json:"user_status"`
	GroupID             int64     `json:"group_id"`
	GroupName           string    `json:"group_name"`
	GroupPlatform       string    `json:"group_platform"`
	GroupStatus         string    `json:"group_status"`
	GroupRateMultiplier float64   `json:"group_rate_multiplier"`
	RateMultiplier      float64   `json:"rate_multiplier"`
	RPMOverride         *int      `json:"rpm_override"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// UserGroupRateListResponse 标准分页结构外加去重用户数。
type UserGroupRateListResponse struct {
	response.PaginatedData
	UserCount int64 `json:"user_count"`
}

// UserGroupRateGroupDTO 分组选项。
type UserGroupRateGroupDTO struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	Status           string  `json:"status"`
	SubscriptionType string  `json:"subscription_type"`
	IsExclusive      bool    `json:"is_exclusive"`
	RateMultiplier   float64 `json:"rate_multiplier"`
}

// UserGroupRateUsageDTO 近 30 天用量。
type UserGroupRateUsageDTO struct {
	UserID     int64   `json:"user_id"`
	GroupID    int64   `json:"group_id"`
	Requests   int64   `json:"requests"`
	ActualCost float64 `json:"actual_cost"`
}

// userGroupRateBatchItem rate_multiplier 必须显式给出：数字表示设为该值，null 表示恢复分组默认。
// 漏写字段不当作清除，避免一个拼错的请求把专属倍率清掉。
type userGroupRateBatchItem struct {
	UserID         int64           `json:"user_id"`
	GroupID        int64           `json:"group_id"`
	RateMultiplier json.RawMessage `json:"rate_multiplier"`
}

type userGroupRateBatchRequest struct {
	Items []userGroupRateBatchItem `json:"items"`
}

// List GET /api/v1/admin/users/group-rates
func (h *UserGroupRateHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.UserGroupRateListFilter{
		Page:       page,
		PageSize:   pageSize,
		Search:     c.Query("search"),
		UserStatus: c.Query("user_status"),
		SortBy:     c.Query("sort_by"),
		SortOrder:  c.Query("sort_order"),
	}
	if raw := strings.TrimSpace(c.Query("group_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		filter.GroupID = id
	}

	list, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]UserGroupRateRowDTO, 0, len(list.Items))
	for _, r := range list.Items {
		items = append(items, UserGroupRateRowDTO{
			UserID:              r.UserID,
			UserEmail:           r.UserEmail,
			Username:            r.Username,
			UserNotes:           r.UserNotes,
			UserStatus:          r.UserStatus,
			GroupID:             r.GroupID,
			GroupName:           r.GroupName,
			GroupPlatform:       r.GroupPlatform,
			GroupStatus:         r.GroupStatus,
			GroupRateMultiplier: r.GroupRateMultiplier,
			RateMultiplier:      r.RateMultiplier,
			RPMOverride:         r.RPMOverride,
			UpdatedAt:           r.UpdatedAt,
		})
	}
	pages := int(math.Ceil(float64(list.Total) / float64(list.PageSize)))
	if pages < 1 {
		pages = 1
	}
	response.Success(c, UserGroupRateListResponse{
		PaginatedData: response.PaginatedData{
			Items:    items,
			Total:    list.Total,
			Page:     list.Page,
			PageSize: list.PageSize,
			Pages:    pages,
		},
		UserCount: list.UserCount,
	})
}

// Groups GET /api/v1/admin/users/group-rates/groups
func (h *UserGroupRateHandler) Groups(c *gin.Context) {
	groups, err := h.svc.Groups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]UserGroupRateGroupDTO, 0, len(groups))
	for _, g := range groups {
		out = append(out, UserGroupRateGroupDTO{
			ID:               g.ID,
			Name:             g.Name,
			Platform:         g.Platform,
			Status:           g.Status,
			SubscriptionType: g.SubscriptionType,
			IsExclusive:      g.IsExclusive,
			RateMultiplier:   g.RateMultiplier,
		})
	}
	response.Success(c, out)
}

// Usage GET /api/v1/admin/users/group-rates/usage?user_ids=1,2,3
func (h *UserGroupRateHandler) Usage(c *gin.Context) {
	var ids []int64
	for _, part := range strings.Split(c.Query("user_ids"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid user_ids")
			return
		}
		ids = append(ids, id)
	}
	usage, err := h.svc.Usage(c.Request.Context(), ids)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]UserGroupRateUsageDTO, 0, len(usage))
	for _, u := range usage {
		out = append(out, UserGroupRateUsageDTO{
			UserID:     u.UserID,
			GroupID:    u.GroupID,
			Requests:   u.Requests,
			ActualCost: u.ActualCost,
		})
	}
	response.Success(c, out)
}

// ApplyBatch POST /api/v1/admin/users/batch-group-rates
func (h *UserGroupRateHandler) ApplyBatch(c *gin.Context) {
	var req userGroupRateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	changes := make([]service.UserGroupRateChange, 0, len(req.Items))
	for _, item := range req.Items {
		change := service.UserGroupRateChange{UserID: item.UserID, GroupID: item.GroupID}
		raw := bytes.TrimSpace(item.RateMultiplier)
		switch {
		case len(raw) == 0:
			response.BadRequest(c, "rate_multiplier is required (use null to restore the group default)")
			return
		case bytes.Equal(raw, []byte("null")):
			// 恢复分组默认
		default:
			var rate float64
			if err := json.Unmarshal(raw, &rate); err != nil {
				response.BadRequest(c, "rate_multiplier must be a number or null")
				return
			}
			change.RateMultiplier = &rate
		}
		changes = append(changes, change)
	}

	result, err := h.svc.ApplyBatch(c.Request.Context(), changes)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": result.Updated, "cleared": result.Cleared})
}
