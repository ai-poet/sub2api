package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// fork：模型广场可用率条。一次读出多个分组的探测结果，只取画分桶需要的列
// （不读 response_excerpt / error_detail 这类长文本），走 idx_group_status_records_group_observed_at。

// ListRecordStatusesSince 返回各分组自 since 起按时间升序的探测结果，键为分组 ID。
func (r *groupStatusRepository) ListRecordStatusesSince(ctx context.Context, groupIDs []int64, since time.Time) (map[int64][]service.GroupStatusRecord, error) {
	result := make(map[int64][]service.GroupStatusRecord, len(groupIDs))
	if len(groupIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT group_id, status, latency_ms, observed_at
		FROM group_status_records
		WHERE group_id = ANY($1) AND observed_at >= $2
		ORDER BY group_id, observed_at ASC
	`, pq.Array(groupIDs), since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var record service.GroupStatusRecord
		if err := rows.Scan(&record.GroupID, &record.Status, &record.LatencyMS, &record.ObservedAt); err != nil {
			return nil, err
		}
		result[record.GroupID] = append(result[record.GroupID], record)
	}
	return result, rows.Err()
}
