package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 专属倍率用户管理（fork 本地功能）：跨用户列出 / 调整 user_group_rate_multipliers 里的专属计费倍率。
//
// 只读写 rate_multiplier 一列；同一张表上的 rpm_override 由分组的 RPM 覆盖弹窗维护，这里从不触碰。
// 用户与分组都是软删除，外键级联不会触发，所有查询都要过滤掉已删除的用户和分组。

const (
	// UserGroupRateBatchMaxItems 单次批量调整的上限：运维提交时整批是一条审批，
	// 200 项的请求体（约 11 KiB）仍在审批详情 16 KiB 的展示上限内，管理员能看到完整列表。
	UserGroupRateBatchMaxItems = 200
	// UserGroupRateUsageMaxUsers 近 30 天用量一次最多查询的用户数。
	UserGroupRateUsageMaxUsers = 200
	// UserGroupRateUsageWindow 用量统计窗口。
	UserGroupRateUsageWindow = 30 * 24 * time.Hour
	// userGroupRateMax 是 DECIMAL(10,4) 能存下的上界（不含）。
	userGroupRateMax = 1e6
)

var (
	ErrUserGroupRateBatchEmpty       = infraerrors.BadRequest("USER_GROUP_RATE_BATCH_EMPTY", "at least one item is required")
	ErrUserGroupRateBatchTooLarge    = infraerrors.BadRequest("USER_GROUP_RATE_BATCH_TOO_LARGE", "too many items in one batch")
	ErrUserGroupRateDuplicatePair    = infraerrors.BadRequest("USER_GROUP_RATE_DUPLICATE_PAIR", "each user and group pair may appear only once")
	ErrUserGroupRateInvalidRate      = infraerrors.BadRequest("USER_GROUP_RATE_INVALID_RATE", "rate_multiplier must be greater than 0 and less than 1000000")
	ErrUserGroupRateInvalidID        = infraerrors.BadRequest("USER_GROUP_RATE_INVALID_ID", "user_id and group_id must be positive")
	ErrUserGroupRateUserNotFound     = infraerrors.BadRequest("USER_GROUP_RATE_USER_NOT_FOUND", "some users do not exist")
	ErrUserGroupRateGroupNotFound    = infraerrors.BadRequest("USER_GROUP_RATE_GROUP_NOT_FOUND", "some groups do not exist")
	ErrUserGroupRateUsageTooMany     = infraerrors.BadRequest("USER_GROUP_RATE_USAGE_TOO_MANY_USERS", "too many user_ids")
	ErrUserGroupRateAdminUnavailable = infraerrors.ServiceUnavailable("USER_GROUP_RATE_UNAVAILABLE", "user group rate service is not available")
)

// UserGroupRateRow 列表里的一行：某个用户在某个分组的专属倍率。
type UserGroupRateRow struct {
	UserID              int64
	UserEmail           string
	Username            string
	UserNotes           string
	UserStatus          string
	GroupID             int64
	GroupName           string
	GroupPlatform       string
	GroupStatus         string
	GroupRateMultiplier float64
	RateMultiplier      float64
	RPMOverride         *int
	UpdatedAt           time.Time
}

// UserGroupRateListFilter 列表筛选与分页。SortBy 只接受白名单内的值，其余回退到 updated_at。
type UserGroupRateListFilter struct {
	Page       int
	PageSize   int
	Search     string
	GroupID    int64
	UserStatus string
	SortBy     string
	SortOrder  string
}

// UserGroupRateList 列表结果。UserCount 是当前筛选下的去重用户数。
type UserGroupRateList struct {
	Items     []UserGroupRateRow
	Total     int64
	UserCount int64
	Page      int
	PageSize  int
}

// UserGroupRateGroupOption 分组选项（筛选下拉、新增弹窗、折扣计算用）。
type UserGroupRateGroupOption struct {
	ID               int64
	Name             string
	Platform         string
	Status           string
	SubscriptionType string
	IsExclusive      bool
	RateMultiplier   float64
}

// UserGroupRateUsage 某个用户在某个分组近 30 天的用量。
type UserGroupRateUsage struct {
	UserID     int64
	GroupID    int64
	Requests   int64
	ActualCost float64
}

// UserGroupRateChange 一项调整：RateMultiplier 为 nil 表示清除专属倍率（恢复分组默认）。
type UserGroupRateChange struct {
	UserID         int64
	GroupID        int64
	RateMultiplier *float64
}

// UserGroupRateBatchResult 批量调整结果。
type UserGroupRateBatchResult struct {
	Updated int
	Cleared int
}

// UserGroupRateAdminRepository 专属倍率管理的数据访问。与上游的 UserGroupRateRepository 分开，
// 避免上游接口和它的测试桩跟着变。
type UserGroupRateAdminRepository interface {
	List(ctx context.Context, filter UserGroupRateListFilter) (*UserGroupRateList, error)
	ListGroups(ctx context.Context) ([]UserGroupRateGroupOption, error)
	Usage(ctx context.Context, userIDs []int64, since time.Time) ([]UserGroupRateUsage, error)
	// ExistingUserIDs / ExistingGroupIDs 返回给定 id 里存在且未删除的那些。
	ExistingUserIDs(ctx context.Context, ids []int64) ([]int64, error)
	ExistingGroupIDs(ctx context.Context, ids []int64) ([]int64, error)
	// ApplyBatch 在一个事务里写入全部调整，只改 rate_multiplier，不动 rpm_override。
	ApplyBatch(ctx context.Context, changes []UserGroupRateChange) error
}
