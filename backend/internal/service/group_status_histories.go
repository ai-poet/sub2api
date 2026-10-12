package service

import (
	"context"
	"time"
)

// fork：模型广场可用率条。用户可见、且开了状态检测的分组一次返回各自的分桶历史，
// 前端不再逐组请求 /group-status/:groupId/history。可见性规则与 ListUserStatuses 一致。

// GroupStatusHistoryOverview 是单个分组在一个时段内的可用率与分桶历史。
type GroupStatusHistoryOverview struct {
	GroupID int64 `json:"group_id"`
	// Availability 是整个时段内非 down 探测的占比（0-100）；TotalCount 为 0 时没有意义。
	Availability float64                    `json:"availability"`
	TotalCount   int                        `json:"total_count"`
	Buckets      []GroupStatusHistoryBucket `json:"buckets"`
}

// groupStatusRecordBatchLister 是批量读取探测结果的仓库能力。
type groupStatusRecordBatchLister interface {
	ListRecordStatusesSince(ctx context.Context, groupIDs []int64, since time.Time) (map[int64][]GroupStatusRecord, error)
}

// ListUserHistories 返回用户可见分组在 period（24h / 7d）内的可用率分桶。
func (s *GroupStatusService) ListUserHistories(ctx context.Context, userID int64, period string) ([]GroupStatusHistoryOverview, error) {
	if err := s.ensureFeatureEnabled(ctx); err != nil {
		return nil, err
	}
	now := time.Now()
	start, _, step, err := GroupStatusPeriodRange(period, now)
	if err != nil {
		return nil, err
	}
	if s.availableGroups == nil {
		return []GroupStatusHistoryOverview{}, nil
	}

	groupIDs, err := s.visibleStatusGroupIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return []GroupStatusHistoryOverview{}, nil
	}

	recordsByGroup, err := s.listRecordStatusesSince(ctx, groupIDs, start)
	if err != nil {
		return nil, err
	}

	out := make([]GroupStatusHistoryOverview, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		records := recordsByGroup[groupID]
		overview := GroupStatusHistoryOverview{
			GroupID:    groupID,
			TotalCount: len(records),
			Buckets:    buildGroupStatusHistory(records, start, now, step),
		}
		if len(records) > 0 {
			available := 0
			for _, record := range records {
				if record.Status != GroupRuntimeStatusDown {
					available++
				}
			}
			overview.Availability = float64(available) / float64(len(records)) * 100
		}
		out = append(out, overview)
	}
	return out, nil
}

// visibleStatusGroupIDs 返回用户可用、且开启了运行状态检测的分组 ID。
func (s *GroupStatusService) visibleStatusGroupIDs(ctx context.Context, userID int64) ([]int64, error) {
	groups, err := s.availableGroups.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}
	candidateIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		candidateIDs = append(candidateIDs, group.ID)
	}
	summaries, err := s.repo.ListSummaries(ctx, candidateIDs)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(summaries))
	for _, summary := range summaries {
		if summary.Enabled {
			ids = append(ids, summary.GroupID)
		}
	}
	return ids, nil
}

// listRecordStatusesSince 优先走批量查询；仓库不支持时逐组读取，结果相同。
func (s *GroupStatusService) listRecordStatusesSince(ctx context.Context, groupIDs []int64, since time.Time) (map[int64][]GroupStatusRecord, error) {
	if lister, ok := s.repo.(groupStatusRecordBatchLister); ok {
		return lister.ListRecordStatusesSince(ctx, groupIDs, since)
	}
	result := make(map[int64][]GroupStatusRecord, len(groupIDs))
	for _, groupID := range groupIDs {
		records, err := s.repo.ListRecordsSince(ctx, groupID, since)
		if err != nil {
			return nil, err
		}
		result[groupID] = records
	}
	return result, nil
}
