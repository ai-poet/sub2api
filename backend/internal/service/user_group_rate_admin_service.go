package service

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// UserGroupRateAdminService 专属倍率用户管理（fork 本地功能）。
//
// 权限由路由层保证：管理员直接调用；运维的批量写被 middleware/console_scope.go 的
// operatorApprovalScope 截获成审批，批准后以管理员身份重放到这里。这里只做校验与落库。
//
// 计费侧在进程内缓存用户专属倍率（gateway.user_group_rate_cache_ttl_seconds，默认 30 秒），
// 与用户管理里改倍率一样不主动失效，改动在一个缓存周期内生效。
type UserGroupRateAdminService struct {
	repo UserGroupRateAdminRepository
	now  func() time.Time
}

// NewUserGroupRateAdminService 构造专属倍率管理服务。
func NewUserGroupRateAdminService(repo UserGroupRateAdminRepository) *UserGroupRateAdminService {
	return &UserGroupRateAdminService{repo: repo, now: time.Now}
}

func (s *UserGroupRateAdminService) ready() error {
	if s == nil || s.repo == nil {
		return ErrUserGroupRateAdminUnavailable
	}
	return nil
}

// userGroupRateSortFields 列表允许的排序字段。
var userGroupRateSortFields = map[string]struct{}{
	"updated_at":      {},
	"rate_multiplier": {},
	"discount":        {},
	"user_id":         {},
	"group_name":      {},
}

func normalizeUserGroupRateListFilter(filter UserGroupRateListFilter) UserGroupRateListFilter {
	f := filter
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > UserGroupRateBatchMaxItems {
		f.PageSize = UserGroupRateBatchMaxItems
	}
	f.Search = strings.TrimSpace(f.Search)
	f.UserStatus = strings.TrimSpace(f.UserStatus)
	if f.GroupID < 0 {
		f.GroupID = 0
	}
	f.SortBy = strings.TrimSpace(f.SortBy)
	if _, ok := userGroupRateSortFields[f.SortBy]; !ok {
		f.SortBy = "updated_at"
	}
	if strings.EqualFold(strings.TrimSpace(f.SortOrder), "asc") {
		f.SortOrder = "asc"
	} else {
		f.SortOrder = "desc"
	}
	return f
}

// List 分页列出所有设置了专属倍率的（用户, 分组）。
func (s *UserGroupRateAdminService) List(ctx context.Context, filter UserGroupRateListFilter) (*UserGroupRateList, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, normalizeUserGroupRateListFilter(filter))
}

// Groups 返回全部未删除的分组及其默认倍率。
func (s *UserGroupRateAdminService) Groups(ctx context.Context) ([]UserGroupRateGroupOption, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	return s.repo.ListGroups(ctx)
}

// Usage 返回给定用户近 30 天按分组汇总的请求数与实际扣费。
func (s *UserGroupRateAdminService) Usage(ctx context.Context, userIDs []int64) ([]UserGroupRateUsage, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	ids := uniquePositiveIDs(userIDs)
	if len(ids) == 0 {
		return []UserGroupRateUsage{}, nil
	}
	if len(ids) > UserGroupRateUsageMaxUsers {
		return nil, ErrUserGroupRateUsageTooMany
	}
	return s.repo.Usage(ctx, ids, s.now().Add(-UserGroupRateUsageWindow))
}

// ApplyBatch 校验并在一个事务里写入一批调整。
func (s *UserGroupRateAdminService) ApplyBatch(ctx context.Context, changes []UserGroupRateChange) (*UserGroupRateBatchResult, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	normalized, err := normalizeUserGroupRateChanges(changes)
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, 0, len(normalized))
	groupIDs := make([]int64, 0, len(normalized))
	for _, c := range normalized {
		userIDs = append(userIDs, c.UserID)
		groupIDs = append(groupIDs, c.GroupID)
	}
	userIDs = uniquePositiveIDs(userIDs)
	groupIDs = uniquePositiveIDs(groupIDs)

	existingUsers, err := s.repo.ExistingUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	if missing := missingIDs(userIDs, existingUsers); len(missing) > 0 {
		return nil, ErrUserGroupRateUserNotFound.WithMetadata(map[string]string{"ids": joinIDs(missing)})
	}
	existingGroups, err := s.repo.ExistingGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	if missing := missingIDs(groupIDs, existingGroups); len(missing) > 0 {
		return nil, ErrUserGroupRateGroupNotFound.WithMetadata(map[string]string{"ids": joinIDs(missing)})
	}

	if err := s.repo.ApplyBatch(ctx, normalized); err != nil {
		return nil, err
	}
	result := &UserGroupRateBatchResult{}
	for _, c := range normalized {
		if c.RateMultiplier == nil {
			result.Cleared++
		} else {
			result.Updated++
		}
	}
	return result, nil
}

// normalizeUserGroupRateChanges 校验数量、id、重复和倍率范围，并把倍率舍入到 DECIMAL(10,4) 的精度。
func normalizeUserGroupRateChanges(changes []UserGroupRateChange) ([]UserGroupRateChange, error) {
	if len(changes) == 0 {
		return nil, ErrUserGroupRateBatchEmpty
	}
	if len(changes) > UserGroupRateBatchMaxItems {
		return nil, ErrUserGroupRateBatchTooLarge
	}
	type pair struct{ user, group int64 }
	seen := make(map[pair]struct{}, len(changes))
	out := make([]UserGroupRateChange, 0, len(changes))
	for _, c := range changes {
		if c.UserID <= 0 || c.GroupID <= 0 {
			return nil, ErrUserGroupRateInvalidID
		}
		key := pair{c.UserID, c.GroupID}
		if _, dup := seen[key]; dup {
			return nil, ErrUserGroupRateDuplicatePair
		}
		seen[key] = struct{}{}
		item := UserGroupRateChange{UserID: c.UserID, GroupID: c.GroupID}
		if c.RateMultiplier != nil {
			rate := *c.RateMultiplier
			if math.IsNaN(rate) || math.IsInf(rate, 0) {
				return nil, ErrUserGroupRateInvalidRate
			}
			rate = math.Round(rate*10000) / 10000
			if rate <= 0 || rate >= userGroupRateMax {
				return nil, ErrUserGroupRateInvalidRate
			}
			item.RateMultiplier = &rate
		}
		out = append(out, item)
	}
	return out, nil
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func missingIDs(want, have []int64) []int64 {
	present := make(map[int64]struct{}, len(have))
	for _, id := range have {
		present[id] = struct{}{}
	}
	var missing []int64
	for _, id := range want {
		if _, ok := present[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	return missing
}

func joinIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}
