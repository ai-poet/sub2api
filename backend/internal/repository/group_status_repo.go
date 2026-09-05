package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// 列清单抽成常量，避免 SELECT / RETURNING / scan 三处漂移。
const (
	groupStatusConfigColumns = `id, group_id, enabled, probe_model, probe_prompt, validation_mode, expected_keywords,
		interval_seconds, timeout_seconds, slow_latency_ms, notify_enabled,
		sol_juice_enabled, sol_juice_interval_seconds, sol_juice_model, created_at, updated_at`

	groupStatusStateColumns = `id, group_id, config_id, latest_status, stable_status, response_excerpt, latency_ms, http_code,
		sub_status, error_detail, observed_at, consecutive_down, consecutive_non_down,
		sol_juice_status, sol_juice_stable_status, sol_juice_value, sol_juice_detail, sol_juice_checked_at,
		sol_juice_consecutive_mismatch, sol_juice_input_tokens, sol_juice_output_tokens, sol_juice_reasoning_tokens,
		created_at, updated_at`

	groupStatusSummarySelect = `SELECT c.group_id, c.id, c.enabled, c.probe_model,
		       COALESCE(s.latest_status, ''), COALESCE(s.stable_status, ''), COALESCE(s.response_excerpt, ''),
		       s.latency_ms, s.http_code, COALESCE(s.sub_status, ''), COALESCE(s.error_detail, ''),
		       s.observed_at, COALESCE(s.consecutive_down, 0), COALESCE(s.consecutive_non_down, 0),
		       c.sol_juice_enabled, c.sol_juice_model, c.sol_juice_interval_seconds,
		       COALESCE(s.sol_juice_status, ''), COALESCE(s.sol_juice_stable_status, ''), COALESCE(s.sol_juice_value, ''),
		       COALESCE(s.sol_juice_detail, ''), s.sol_juice_checked_at, COALESCE(s.sol_juice_consecutive_mismatch, 0),
		       COALESCE(s.sol_juice_input_tokens, 0), COALESCE(s.sol_juice_output_tokens, 0), COALESCE(s.sol_juice_reasoning_tokens, 0)
		FROM group_status_configs c
		LEFT JOIN group_status_states s ON s.group_id = c.group_id`

	groupStatusJuiceRecordColumns = `id, group_id, config_id, model, effort, classification, normalized_value, answer_excerpt,
		http_code, latency_ms, input_tokens, output_tokens, reasoning_tokens, error_detail, observed_at, created_at`
)

// aliasColumns 给列清单里的每一列加表别名前缀。
func aliasColumns(alias, columns string) string {
	parts := strings.Split(columns, ",")
	for i, part := range parts {
		parts[i] = alias + "." + strings.TrimSpace(part)
	}
	return strings.Join(parts, ", ")
}

type groupStatusRepository struct {
	db *sql.DB
}

func NewGroupStatusRepository(db *sql.DB) service.GroupStatusRepository {
	return &groupStatusRepository{db: db}
}

func (r *groupStatusRepository) GetConfig(ctx context.Context, groupID int64) (*service.GroupStatusConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+groupStatusConfigColumns+`
		FROM group_status_configs
		WHERE group_id = $1
	`, groupID)
	cfg, err := scanGroupStatusConfig(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrGroupStatusConfigNotFound
		}
		return nil, err
	}
	return cfg, nil
}

func (r *groupStatusRepository) UpsertConfig(ctx context.Context, config *service.GroupStatusConfig) (*service.GroupStatusConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO group_status_configs (
			group_id, enabled, probe_model, probe_prompt, validation_mode, expected_keywords,
			interval_seconds, timeout_seconds, slow_latency_ms, notify_enabled,
			sol_juice_enabled, sol_juice_interval_seconds, sol_juice_model, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		ON CONFLICT (group_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			probe_model = EXCLUDED.probe_model,
			probe_prompt = EXCLUDED.probe_prompt,
			validation_mode = EXCLUDED.validation_mode,
			expected_keywords = EXCLUDED.expected_keywords,
			interval_seconds = EXCLUDED.interval_seconds,
			timeout_seconds = EXCLUDED.timeout_seconds,
			slow_latency_ms = EXCLUDED.slow_latency_ms,
			notify_enabled = EXCLUDED.notify_enabled,
			sol_juice_enabled = EXCLUDED.sol_juice_enabled,
			sol_juice_interval_seconds = EXCLUDED.sol_juice_interval_seconds,
			sol_juice_model = EXCLUDED.sol_juice_model,
			updated_at = NOW()
		RETURNING `+groupStatusConfigColumns+`
	`, config.GroupID, config.Enabled, config.ProbeModel, config.ProbePrompt, config.ValidationMode,
		mustJSON(config.ExpectedKeywords), config.IntervalSeconds, config.TimeoutSeconds, config.SlowLatencyMS,
		config.NotifyEnabled, config.SolJuiceEnabled, config.SolJuiceIntervalSeconds, config.SolJuiceModel)
	return scanGroupStatusConfig(row)
}

func (r *groupStatusRepository) ListDueConfigs(ctx context.Context, now time.Time, limit int) ([]*service.GroupStatusConfig, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+aliasColumns("c", groupStatusConfigColumns)+`
		FROM group_status_configs c
		LEFT JOIN group_status_states s ON s.group_id = c.group_id
		WHERE c.enabled = TRUE
		  AND (
		        s.observed_at IS NULL
		        OR s.observed_at <= ($1::timestamptz - (c.interval_seconds * INTERVAL '1 second'))
		      )
		ORDER BY COALESCE(s.observed_at, to_timestamp(0)) ASC, c.group_id ASC
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*service.GroupStatusConfig
	for rows.Next() {
		cfg, err := scanGroupStatusConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cfg)
	}
	return out, rows.Err()
}

// ListDueSolJuiceConfigs 列出到期的纯 Sol 验证配置：分组仍是 OpenAI 平台、两个开关都开、且距上次验证超过间隔。
func (r *groupStatusRepository) ListDueSolJuiceConfigs(ctx context.Context, now time.Time, limit int) ([]*service.GroupStatusConfig, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+aliasColumns("c", groupStatusConfigColumns)+`
		FROM group_status_configs c
		JOIN groups g ON g.id = c.group_id
		LEFT JOIN group_status_states s ON s.group_id = c.group_id
		WHERE c.enabled = TRUE
		  AND c.sol_juice_enabled = TRUE
		  AND g.platform = $3
		  AND (
		        s.sol_juice_checked_at IS NULL
		        OR s.sol_juice_checked_at <= ($1::timestamptz - (c.sol_juice_interval_seconds * INTERVAL '1 second'))
		      )
		ORDER BY COALESCE(s.sol_juice_checked_at, to_timestamp(0)) ASC, c.group_id ASC
		LIMIT $2
	`, now, limit, service.PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*service.GroupStatusConfig
	for rows.Next() {
		cfg, err := scanGroupStatusConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cfg)
	}
	return out, rows.Err()
}

func (r *groupStatusRepository) GetState(ctx context.Context, groupID int64) (*service.GroupStatusState, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT `+groupStatusStateColumns+`
		FROM group_status_states
		WHERE group_id = $1
	`, groupID)
	state, err := scanGroupStatusState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

func (r *groupStatusRepository) ListSummaries(ctx context.Context, groupIDs []int64) ([]service.GroupStatusSummary, error) {
	if len(groupIDs) == 0 {
		return []service.GroupStatusSummary{}, nil
	}
	rows, err := r.db.QueryContext(ctx, groupStatusSummarySelect+`
		WHERE c.group_id = ANY($1)
		ORDER BY c.group_id ASC
	`, pq.Array(groupIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanGroupStatusSummaries(rows)
}

func (r *groupStatusRepository) ListAllSummaries(ctx context.Context) ([]service.GroupStatusSummary, error) {
	rows, err := r.db.QueryContext(ctx, groupStatusSummarySelect+`
		ORDER BY c.group_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanGroupStatusSummaries(rows)
}

func (r *groupStatusRepository) SaveProbeResult(ctx context.Context, result *service.GroupStatusProbeResult) (*service.GroupStatusState, *service.GroupStatusEvent, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO group_status_records (
			group_id, config_id, status, response_excerpt, latency_ms, http_code, sub_status, error_detail, observed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, result.GroupID, result.ConfigID, result.Status, nullIfEmpty(result.ResponseExcerpt), result.LatencyMS,
		result.HTTPCode, result.SubStatus, nullIfEmpty(result.ErrorDetail), result.ObservedAt); err != nil {
		return nil, nil, err
	}

	prev, err := r.getStateForUpdate(ctx, tx, result.GroupID)
	if err != nil {
		return nil, nil, err
	}

	next, event := service.ComputeGroupStatusTransition(prev, result)

	// 只写存活探测自己的列；sol_juice_* 由 SaveSolJuiceResult 维护，这里不能覆盖
	row := tx.QueryRowContext(ctx, `
		INSERT INTO group_status_states (
			group_id, config_id, latest_status, stable_status, response_excerpt, latency_ms, http_code,
			sub_status, error_detail, observed_at, consecutive_down, consecutive_non_down, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		ON CONFLICT (group_id) DO UPDATE SET
			config_id = EXCLUDED.config_id,
			latest_status = EXCLUDED.latest_status,
			stable_status = EXCLUDED.stable_status,
			response_excerpt = EXCLUDED.response_excerpt,
			latency_ms = EXCLUDED.latency_ms,
			http_code = EXCLUDED.http_code,
			sub_status = EXCLUDED.sub_status,
			error_detail = EXCLUDED.error_detail,
			observed_at = EXCLUDED.observed_at,
			consecutive_down = EXCLUDED.consecutive_down,
			consecutive_non_down = EXCLUDED.consecutive_non_down,
			updated_at = NOW()
		RETURNING `+groupStatusStateColumns+`
	`, next.GroupID, next.ConfigID, next.LatestStatus, next.StableStatus, nullIfEmpty(next.ResponseExcerpt),
		next.LatencyMS, next.HTTPCode, next.SubStatus, nullIfEmpty(next.ErrorDetail), next.ObservedAt,
		next.ConsecutiveDown, next.ConsecutiveNonDown)
	savedState, err := scanGroupStatusState(row)
	if err != nil {
		return nil, nil, err
	}

	if event != nil {
		event, err = insertGroupStatusEvent(ctx, tx, event)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	tx = nil
	return savedState, event, nil
}

// SaveSolJuiceResult 落一条 Juice 样本、只更新 group_status_states 的 sol_juice_* 列，稳定结论切换时写事件。
func (r *groupStatusRepository) SaveSolJuiceResult(ctx context.Context, result *service.GroupStatusSolJuiceResult) (*service.GroupStatusState, *service.GroupStatusEvent, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO group_status_juice_records (
			group_id, config_id, model, effort, classification, normalized_value, answer_excerpt,
			http_code, latency_ms, input_tokens, output_tokens, reasoning_tokens, error_detail, observed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
	`, result.GroupID, result.ConfigID, result.Model, result.Effort, result.Classification, result.NormalizedValue,
		nullIfEmpty(result.AnswerExcerpt), result.HTTPCode, result.LatencyMS, result.InputTokens, result.OutputTokens,
		result.ReasoningTokens, nullIfEmpty(result.ErrorDetail), result.ObservedAt); err != nil {
		return nil, nil, err
	}

	prev, err := r.getStateForUpdate(ctx, tx, result.GroupID)
	if err != nil {
		return nil, nil, err
	}

	next, event := service.ComputeSolJuiceTransition(prev, result)

	row := tx.QueryRowContext(ctx, `
		INSERT INTO group_status_states (
			group_id, config_id, sol_juice_status, sol_juice_stable_status, sol_juice_value, sol_juice_detail,
			sol_juice_checked_at, sol_juice_consecutive_mismatch, sol_juice_input_tokens, sol_juice_output_tokens,
			sol_juice_reasoning_tokens, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		ON CONFLICT (group_id) DO UPDATE SET
			sol_juice_status = EXCLUDED.sol_juice_status,
			sol_juice_stable_status = EXCLUDED.sol_juice_stable_status,
			sol_juice_value = EXCLUDED.sol_juice_value,
			sol_juice_detail = EXCLUDED.sol_juice_detail,
			sol_juice_checked_at = EXCLUDED.sol_juice_checked_at,
			sol_juice_consecutive_mismatch = EXCLUDED.sol_juice_consecutive_mismatch,
			sol_juice_input_tokens = EXCLUDED.sol_juice_input_tokens,
			sol_juice_output_tokens = EXCLUDED.sol_juice_output_tokens,
			sol_juice_reasoning_tokens = EXCLUDED.sol_juice_reasoning_tokens,
			updated_at = NOW()
		RETURNING `+groupStatusStateColumns+`
	`, next.GroupID, next.ConfigID, next.SolJuiceStatus, next.SolJuiceStableStatus, next.SolJuiceValue,
		nullIfEmpty(next.SolJuiceDetail), next.SolJuiceCheckedAt, next.SolJuiceConsecutiveMismatch,
		next.SolJuiceInputTokens, next.SolJuiceOutputTokens, next.SolJuiceReasoningTokens)
	savedState, err := scanGroupStatusState(row)
	if err != nil {
		return nil, nil, err
	}

	if event != nil {
		event, err = insertGroupStatusEvent(ctx, tx, event)
		if err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	tx = nil
	return savedState, event, nil
}

func insertGroupStatusEvent(ctx context.Context, tx *sql.Tx, event *service.GroupStatusEvent) (*service.GroupStatusEvent, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO group_status_events (
			group_id, config_id, event_type, from_status, to_status, latency_ms, http_code, sub_status, error_detail, observed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, group_id, config_id, event_type, from_status, to_status, latency_ms, http_code,
		          sub_status, error_detail, observed_at, created_at
	`, event.GroupID, event.ConfigID, event.EventType, event.FromStatus, event.ToStatus, event.LatencyMS,
		event.HTTPCode, event.SubStatus, nullIfEmpty(event.ErrorDetail), event.ObservedAt)
	return scanGroupStatusEvent(row)
}

func (r *groupStatusRepository) ListRecordsSince(ctx context.Context, groupID int64, since time.Time) ([]service.GroupStatusRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, group_id, config_id, status, response_excerpt, latency_ms, http_code, sub_status, error_detail, observed_at, created_at
		FROM group_status_records
		WHERE group_id = $1 AND observed_at >= $2
		ORDER BY observed_at ASC
	`, groupID, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.GroupStatusRecord, 0)
	for rows.Next() {
		record, err := scanGroupStatusRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *record)
	}
	return out, rows.Err()
}

func (r *groupStatusRepository) ListRecentRecords(ctx context.Context, groupID int64, limit int) ([]service.GroupStatusRecord, error) {
	if limit <= 0 {
		limit = 24
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, group_id, config_id, status, response_excerpt, latency_ms, http_code, sub_status, error_detail, observed_at, created_at
		FROM group_status_records
		WHERE group_id = $1
		ORDER BY observed_at DESC
		LIMIT $2
	`, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.GroupStatusRecord, 0)
	for rows.Next() {
		record, err := scanGroupStatusRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *record)
	}
	return out, rows.Err()
}

func (r *groupStatusRepository) ListRecentSolJuiceRecords(ctx context.Context, groupID int64, limit int) ([]service.GroupStatusJuiceRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+groupStatusJuiceRecordColumns+`
		FROM group_status_juice_records
		WHERE group_id = $1
		ORDER BY observed_at DESC
		LIMIT $2
	`, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.GroupStatusJuiceRecord, 0)
	for rows.Next() {
		record, err := scanGroupStatusJuiceRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *record)
	}
	return out, rows.Err()
}

func (r *groupStatusRepository) ListEvents(ctx context.Context, groupID int64, limit int) ([]service.GroupStatusEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, group_id, config_id, event_type, from_status, to_status, latency_ms, http_code, sub_status, error_detail, observed_at, created_at
		FROM group_status_events
		WHERE group_id = $1
		ORDER BY observed_at DESC
		LIMIT $2
	`, groupID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.GroupStatusEvent, 0)
	for rows.Next() {
		event, err := scanGroupStatusEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *event)
	}
	return out, rows.Err()
}

func (r *groupStatusRepository) CalculateAvailability(ctx context.Context, groupIDs []int64, since time.Time) (map[int64]float64, error) {
	result := make(map[int64]float64, len(groupIDs))
	if len(groupIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT group_id,
		       COALESCE(AVG(CASE WHEN status <> 'down' THEN 1.0 ELSE 0.0 END) * 100.0, 0.0) AS availability
		FROM group_status_records
		WHERE group_id = ANY($1) AND observed_at >= $2
		GROUP BY group_id
	`, pq.Array(groupIDs), since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var groupID int64
		var availability float64
		if err := rows.Scan(&groupID, &availability); err != nil {
			return nil, err
		}
		result[groupID] = availability
	}
	return result, rows.Err()
}

func (r *groupStatusRepository) DeleteRecordsOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM group_status_records WHERE observed_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *groupStatusRepository) DeleteSolJuiceRecordsOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM group_status_juice_records WHERE observed_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *groupStatusRepository) getStateForUpdate(ctx context.Context, tx *sql.Tx, groupID int64) (*service.GroupStatusState, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT `+groupStatusStateColumns+`
		FROM group_status_states
		WHERE group_id = $1
		FOR UPDATE
	`, groupID)
	state, err := scanGroupStatusState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

func scanGroupStatusConfig(row scannable) (*service.GroupStatusConfig, error) {
	var keywordsRaw []byte
	cfg := &service.GroupStatusConfig{}
	if err := row.Scan(
		&cfg.ID, &cfg.GroupID, &cfg.Enabled, &cfg.ProbeModel, &cfg.ProbePrompt, &cfg.ValidationMode, &keywordsRaw,
		&cfg.IntervalSeconds, &cfg.TimeoutSeconds, &cfg.SlowLatencyMS, &cfg.NotifyEnabled,
		&cfg.SolJuiceEnabled, &cfg.SolJuiceIntervalSeconds, &cfg.SolJuiceModel, &cfg.CreatedAt, &cfg.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(keywordsRaw) > 0 {
		if err := json.Unmarshal(keywordsRaw, &cfg.ExpectedKeywords); err != nil {
			return nil, err
		}
	}
	if cfg.ExpectedKeywords == nil {
		cfg.ExpectedKeywords = []string{}
	}
	return cfg, nil
}

func scanGroupStatusState(row scannable) (*service.GroupStatusState, error) {
	state := &service.GroupStatusState{}
	var responseExcerpt sql.NullString
	var latency sql.NullInt64
	var httpCode sql.NullInt64
	var subStatus sql.NullString
	var errorDetail sql.NullString
	var observedAt sql.NullTime
	var solJuiceDetail sql.NullString
	var solJuiceCheckedAt sql.NullTime
	if err := row.Scan(
		&state.ID, &state.GroupID, &state.ConfigID, &state.LatestStatus, &state.StableStatus, &responseExcerpt,
		&latency, &httpCode, &subStatus, &errorDetail, &observedAt, &state.ConsecutiveDown,
		&state.ConsecutiveNonDown,
		&state.SolJuiceStatus, &state.SolJuiceStableStatus, &state.SolJuiceValue, &solJuiceDetail, &solJuiceCheckedAt,
		&state.SolJuiceConsecutiveMismatch, &state.SolJuiceInputTokens, &state.SolJuiceOutputTokens, &state.SolJuiceReasoningTokens,
		&state.CreatedAt, &state.UpdatedAt,
	); err != nil {
		return nil, err
	}
	state.ResponseExcerpt = responseExcerpt.String
	if latency.Valid {
		v := latency.Int64
		state.LatencyMS = &v
	}
	if httpCode.Valid {
		v := int(httpCode.Int64)
		state.HTTPCode = &v
	}
	state.SubStatus = subStatus.String
	state.ErrorDetail = errorDetail.String
	if observedAt.Valid {
		v := observedAt.Time
		state.ObservedAt = &v
	}
	state.SolJuiceDetail = solJuiceDetail.String
	if solJuiceCheckedAt.Valid {
		v := solJuiceCheckedAt.Time
		state.SolJuiceCheckedAt = &v
	}
	return state, nil
}

func scanGroupStatusRecord(row scannable) (*service.GroupStatusRecord, error) {
	record := &service.GroupStatusRecord{}
	var responseExcerpt sql.NullString
	var latency sql.NullInt64
	var httpCode sql.NullInt64
	var subStatus sql.NullString
	var errorDetail sql.NullString
	if err := row.Scan(
		&record.ID, &record.GroupID, &record.ConfigID, &record.Status, &responseExcerpt, &latency, &httpCode,
		&subStatus, &errorDetail, &record.ObservedAt, &record.CreatedAt,
	); err != nil {
		return nil, err
	}
	record.ResponseExcerpt = responseExcerpt.String
	if latency.Valid {
		v := latency.Int64
		record.LatencyMS = &v
	}
	if httpCode.Valid {
		v := int(httpCode.Int64)
		record.HTTPCode = &v
	}
	record.SubStatus = subStatus.String
	record.ErrorDetail = errorDetail.String
	return record, nil
}

func scanGroupStatusJuiceRecord(row scannable) (*service.GroupStatusJuiceRecord, error) {
	record := &service.GroupStatusJuiceRecord{}
	var answerExcerpt sql.NullString
	var httpCode sql.NullInt64
	var latency sql.NullInt64
	var errorDetail sql.NullString
	if err := row.Scan(
		&record.ID, &record.GroupID, &record.ConfigID, &record.Model, &record.Effort, &record.Classification,
		&record.NormalizedValue, &answerExcerpt, &httpCode, &latency, &record.InputTokens, &record.OutputTokens,
		&record.ReasoningTokens, &errorDetail, &record.ObservedAt, &record.CreatedAt,
	); err != nil {
		return nil, err
	}
	record.AnswerExcerpt = answerExcerpt.String
	if httpCode.Valid {
		v := int(httpCode.Int64)
		record.HTTPCode = &v
	}
	if latency.Valid {
		v := latency.Int64
		record.LatencyMS = &v
	}
	record.ErrorDetail = errorDetail.String
	return record, nil
}

func scanGroupStatusEvent(row scannable) (*service.GroupStatusEvent, error) {
	event := &service.GroupStatusEvent{}
	var latency sql.NullInt64
	var httpCode sql.NullInt64
	var subStatus sql.NullString
	var errorDetail sql.NullString
	if err := row.Scan(
		&event.ID, &event.GroupID, &event.ConfigID, &event.EventType, &event.FromStatus, &event.ToStatus,
		&latency, &httpCode, &subStatus, &errorDetail, &event.ObservedAt, &event.CreatedAt,
	); err != nil {
		return nil, err
	}
	if latency.Valid {
		v := latency.Int64
		event.LatencyMS = &v
	}
	if httpCode.Valid {
		v := int(httpCode.Int64)
		event.HTTPCode = &v
	}
	event.SubStatus = subStatus.String
	event.ErrorDetail = errorDetail.String
	return event, nil
}

func scanGroupStatusSummaries(rows *sql.Rows) ([]service.GroupStatusSummary, error) {
	var out []service.GroupStatusSummary
	for rows.Next() {
		item, err := scanGroupStatusSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func scanGroupStatusSummary(row scannable) (*service.GroupStatusSummary, error) {
	item := &service.GroupStatusSummary{}
	var latency sql.NullInt64
	var httpCode sql.NullInt64
	var observedAt sql.NullTime
	var solJuiceCheckedAt sql.NullTime
	if err := row.Scan(
		&item.GroupID, &item.ConfigID, &item.Enabled, &item.ProbeModel,
		&item.LatestStatus, &item.StableStatus, &item.ResponseExcerpt, &latency, &httpCode,
		&item.SubStatus, &item.ErrorDetail, &observedAt, &item.ConsecutiveDown, &item.ConsecutiveNonDown,
		&item.SolJuiceEnabled, &item.SolJuiceModel, &item.SolJuiceIntervalSeconds,
		&item.SolJuiceStatus, &item.SolJuiceStableStatus, &item.SolJuiceValue,
		&item.SolJuiceDetail, &solJuiceCheckedAt, &item.SolJuiceConsecutiveMismatch,
		&item.SolJuiceInputTokens, &item.SolJuiceOutputTokens, &item.SolJuiceReasoningTokens,
	); err != nil {
		return nil, err
	}
	if latency.Valid {
		v := latency.Int64
		item.LatencyMS = &v
	}
	if httpCode.Valid {
		v := int(httpCode.Int64)
		item.HTTPCode = &v
	}
	if observedAt.Valid {
		v := observedAt.Time
		item.ObservedAt = &v
	}
	if solJuiceCheckedAt.Valid {
		v := solJuiceCheckedAt.Time
		item.SolJuiceCheckedAt = &v
	}
	return item, nil
}

func mustJSON(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}
