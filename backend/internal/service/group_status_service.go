package service

import (
	"context"
	"errors"
	"sort"
	"time"
)

type GroupStatusAdminView struct {
	Group   *Group             `json:"group"`
	Config  *GroupStatusConfig `json:"config"`
	Summary GroupStatusSummary `json:"summary"`
}

type GroupStatusService struct {
	repo            GroupStatusRepository
	groupRepo       GroupRepository
	settingService  *SettingService
	availableGroups AvailableGroupReader
}

func NewGroupStatusService(
	repo GroupStatusRepository,
	groupRepo GroupRepository,
	settingService *SettingService,
	availableGroups AvailableGroupReader,
) *GroupStatusService {
	return &GroupStatusService{
		repo:            repo,
		groupRepo:       groupRepo,
		settingService:  settingService,
		availableGroups: availableGroups,
	}
}

func (s *GroupStatusService) GetAdminView(ctx context.Context, groupID int64) (*GroupStatusAdminView, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	cfg, err := s.repo.GetConfig(ctx, groupID)
	if err != nil {
		if !errors.Is(err, ErrGroupStatusConfigNotFound) {
			return nil, err
		}
		cfg = DefaultGroupStatusConfig(group)
	}

	summary, err := s.getSummaryForGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if summary.ConfigID == 0 {
		summary.GroupID = groupID
		summary.Enabled = cfg.Enabled
		summary.ProbeModel = cfg.ProbeModel
	}

	return &GroupStatusAdminView{
		Group:   group,
		Config:  cfg,
		Summary: summary,
	}, nil
}

func (s *GroupStatusService) UpdateConfig(ctx context.Context, groupID int64, input *GroupStatusConfigUpsertInput) (*GroupStatusAdminView, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	cfg, err := NormalizeGroupStatusConfig(group, input)
	if err != nil {
		return nil, err
	}
	cfg.GroupID = groupID

	saved, err := s.repo.UpsertConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	view, err := s.GetAdminView(ctx, groupID)
	if err != nil {
		return nil, err
	}
	view.Config = saved
	return view, nil
}

func (s *GroupStatusService) ListAdminSummaries(ctx context.Context) ([]GroupStatusSummary, error) {
	return s.repo.ListAllSummaries(ctx)
}

func (s *GroupStatusService) ListUserStatuses(ctx context.Context, userID int64) ([]GroupStatusListItem, error) {
	if err := s.ensureFeatureEnabled(ctx); err != nil {
		return nil, err
	}
	if s.availableGroups == nil {
		return []GroupStatusListItem{}, nil
	}

	groups, err := s.availableGroups.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []GroupStatusListItem{}, nil
	}

	groupIDs := make([]int64, 0, len(groups))
	groupMap := make(map[int64]Group, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
		groupMap[group.ID] = group
	}

	summaries, err := s.repo.ListSummaries(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	summaryMap := make(map[int64]GroupStatusSummary, len(summaries))
	filteredIDs := make([]int64, 0, len(summaries))
	for _, summary := range summaries {
		if !summary.Enabled {
			continue
		}
		summaryMap[summary.GroupID] = summary
		filteredIDs = append(filteredIDs, summary.GroupID)
	}
	if len(filteredIDs) == 0 {
		return []GroupStatusListItem{}, nil
	}

	availability24, err := s.repo.CalculateAvailability(ctx, filteredIDs, time.Now().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	availability7d, err := s.repo.CalculateAvailability(ctx, filteredIDs, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}

	items := make([]GroupStatusListItem, 0, len(filteredIDs))
	for _, groupID := range filteredIDs {
		group, ok := groupMap[groupID]
		if !ok {
			continue
		}
		summary := summaryMap[groupID]
		items = append(items, GroupStatusListItem{
			Group:          group,
			Summary:        summary,
			Availability24: availability24[groupID],
			Availability7d: availability7d[groupID],
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Group.SortOrder != items[j].Group.SortOrder {
			return items[i].Group.SortOrder < items[j].Group.SortOrder
		}
		return items[i].Group.ID < items[j].Group.ID
	})
	return items, nil
}

func (s *GroupStatusService) GetUserHistory(ctx context.Context, userID, groupID int64, period string) ([]GroupStatusHistoryBucket, error) {
	if err := s.ensureUserGroupAccess(ctx, userID, groupID); err != nil {
		return nil, err
	}

	now := time.Now()
	start, _, step, err := GroupStatusPeriodRange(period, now)
	if err != nil {
		return nil, err
	}
	records, err := s.repo.ListRecordsSince(ctx, groupID, start)
	if err != nil {
		return nil, err
	}
	return buildGroupStatusHistory(records, start, now, step), nil
}

func (s *GroupStatusService) GetUserEvents(ctx context.Context, userID, groupID int64, limit int) ([]GroupStatusEvent, error) {
	if err := s.ensureUserGroupAccess(ctx, userID, groupID); err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, groupID, limit)
}

func (s *GroupStatusService) GetUserRecentRecords(ctx context.Context, userID, groupID int64, limit int) ([]GroupStatusRecord, error) {
	if err := s.ensureUserGroupAccess(ctx, userID, groupID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 24
	}
	if limit > 100 {
		limit = 100
	}
	records, err := s.repo.ListRecentRecords(ctx, groupID, limit)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].ObservedAt.Before(records[j].ObservedAt)
	})
	return records, nil
}

func (s *GroupStatusService) ensureFeatureEnabled(ctx context.Context) error {
	if s.settingService == nil {
		return nil
	}
	enabled, err := s.settingService.IsGroupStatusEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrGroupStatusFeatureClosed
	}
	return nil
}

func (s *GroupStatusService) ensureUserGroupAccess(ctx context.Context, userID, groupID int64) error {
	if err := s.ensureFeatureEnabled(ctx); err != nil {
		return err
	}
	if s.availableGroups == nil {
		return ErrGroupStatusForbidden
	}
	groups, err := s.availableGroups.GetAvailableGroups(ctx, userID)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.ID == groupID {
			summaries, err := s.repo.ListSummaries(ctx, []int64{groupID})
			if err != nil {
				return err
			}
			if len(summaries) == 0 || !summaries[0].Enabled {
				return ErrGroupStatusForbidden
			}
			return nil
		}
	}
	return ErrGroupStatusForbidden
}

func (s *GroupStatusService) getSummaryForGroup(ctx context.Context, groupID int64) (GroupStatusSummary, error) {
	summaries, err := s.repo.ListSummaries(ctx, []int64{groupID})
	if err != nil {
		return GroupStatusSummary{}, err
	}
	if len(summaries) == 0 {
		return GroupStatusSummary{}, nil
	}
	return summaries[0], nil
}

func buildGroupStatusHistory(records []GroupStatusRecord, start, end time.Time, step time.Duration) []GroupStatusHistoryBucket {
	if step <= 0 {
		return []GroupStatusHistoryBucket{}
	}
	bucketCount := int(end.Sub(start) / step)
	if bucketCount <= 0 {
		bucketCount = 1
	}
	buckets := make([]GroupStatusHistoryBucket, 0, bucketCount)
	latencySums := make([]float64, 0, bucketCount)
	latencyCounts := make([]int, 0, bucketCount)
	cursor := start
	for cursor.Before(end) {
		bucketEnd := cursor.Add(step)
		buckets = append(buckets, GroupStatusHistoryBucket{
			BucketStart: cursor,
			BucketEnd:   bucketEnd,
		})
		latencySums = append(latencySums, 0)
		latencyCounts = append(latencyCounts, 0)
		cursor = bucketEnd
		if len(buckets) > 64 {
			break
		}
	}

	for _, record := range records {
		if record.ObservedAt.Before(start) {
			continue
		}
		index := int(record.ObservedAt.Sub(start) / step)
		if index < 0 || index >= len(buckets) {
			continue
		}
		bucket := &buckets[index]
		bucket.TotalCount++
		if record.Status == GroupRuntimeStatusDown {
			bucket.DownCount++
		}
		if record.LatencyMS != nil {
			latencySums[index] += float64(*record.LatencyMS)
			latencyCounts[index]++
		}
		bucket.LatestStatus = record.Status
	}

	for i := range buckets {
		if buckets[i].TotalCount == 0 {
			continue
		}
		available := buckets[i].TotalCount - buckets[i].DownCount
		buckets[i].Availability = (float64(available) / float64(buckets[i].TotalCount)) * 100
		if latencyCounts[i] > 0 {
			value := latencySums[i] / float64(latencyCounts[i])
			buckets[i].AvgLatencyMS = &value
		}
	}
	return buckets
}
