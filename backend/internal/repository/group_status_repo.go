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
//
// 以下列从各自的迁移起休眠（保留给旧镜像，SKIP_SETUP 与回滚要求新 schema 兼容旧镜像），这里不再读写：
// 235 的 sol_juice_*、242 的 modeltrace_*、243 起的 astra_check_request_model 与 group_status_states.astra_check_*。
const (
	groupStatusConfigColumns = `id, group_id, enabled, probe_model, probe_prompt, validation_mode, expected_keywords,
		interval_seconds, timeout_seconds, slow_latency_ms, notify_enabled,
		astra_check_enabled, astra_check_models, astra_check_tier, astra_check_interval_seconds,
		created_at, updated_at`

	groupStatusStateColumns = `id, group_id, config_id, latest_status, stable_status, response_excerpt, latency_ms, http_code,
		sub_status, error_detail, observed_at, consecutive_down, consecutive_non_down, total_latency_ms,
		created_at, updated_at`

	groupStatusAstraRunColumns = `id, group_id, config_id, platform, expected_model, round, benchmark_package_id, benchmark_version,
		benchmark_sha256, scoring_version, request_model, tier, account_id, verdict, winner_model, matches, cells, reasons, samples,
		requests_planned, requests_completed, valid_samples, input_tokens, output_tokens, reasoning_tokens, cost_usd,
		latency_ms, http_code, error_detail, started_at, finished_at, created_at`

	groupStatusAstraStateColumns = `id, group_id, config_id, expected_model, verdict, stable_status, winner_model, matches, reasons,
		detail, checked_at, consecutive_mismatch, valid_samples, planned_samples, input_tokens, output_tokens, reasoning_tokens,
		last_cost_usd, last_run_id, benchmark_package_id, benchmark_version, created_at, updated_at`

	groupStatusSummarySelect = `SELECT c.group_id, c.id, c.enabled, c.probe_model,
		       COALESCE(s.latest_status, ''), COALESCE(s.stable_status, ''), COALESCE(s.response_excerpt, ''),
		       s.latency_ms, s.http_code, s.total_latency_ms, COALESCE(s.sub_status, ''), COALESCE(s.error_detail, ''),
		       s.observed_at, COALESCE(s.consecutive_down, 0), COALESCE(s.consecutive_non_down, 0),
		       c.astra_check_enabled, c.astra_check_models, c.astra_check_tier, c.astra_check_interval_seconds
		FROM group_status_configs c
		LEFT JOIN group_status_states s ON s.group_id = c.group_id`
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
			astra_check_enabled, astra_check_models, astra_check_tier, astra_check_interval_seconds,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12::jsonb, $13, $14, NOW(), NOW())
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
			astra_check_enabled = EXCLUDED.astra_check_enabled,
			astra_check_models = EXCLUDED.astra_check_models,
			astra_check_tier = EXCLUDED.astra_check_tier,
			astra_check_interval_seconds = EXCLUDED.astra_check_interval_seconds,
			updated_at = NOW()
		RETURNING `+groupStatusConfigColumns+`
	`, config.GroupID, config.Enabled, config.ProbeModel, config.ProbePrompt, config.ValidationMode,
		mustJSON(config.ExpectedKeywords), config.IntervalSeconds, config.TimeoutSeconds, config.SlowLatencyMS,
		config.NotifyEnabled, config.AstraCheckEnabled, mustJSONArray(config.AstraCheckModels), config.AstraCheckTier,
		config.AstraCheckIntervalSeconds)
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
	summaries, err := scanGroupStatusSummaries(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	return r.attachAstraCheckStates(ctx, summaries)
}

func (r *groupStatusRepository) ListAllSummaries(ctx context.Context) ([]service.GroupStatusSummary, error) {
	rows, err := r.db.QueryContext(ctx, groupStatusSummarySelect+`
		ORDER BY c.group_id ASC
	`)
	if err != nil {
		return nil, err
	}
	summaries, err := scanGroupStatusSummaries(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	return r.attachAstraCheckStates(ctx, summaries)
}

// attachAstraCheckStates 把每个分组各预期模型的指纹状态挂到 summary 上（排序与占位由 service 负责）。
func (r *groupStatusRepository) attachAstraCheckStates(ctx context.Context, summaries []service.GroupStatusSummary) ([]service.GroupStatusSummary, error) {
	if len(summaries) == 0 {
		return summaries, nil
	}
	groupIDs := make([]int64, 0, len(summaries))
	for _, summary := range summaries {
		groupIDs = append(groupIDs, summary.GroupID)
	}
	states, err := r.ListAstraCheckStates(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	byGroup := make(map[int64][]service.GroupStatusAstraCheckState, len(summaries))
	for _, state := range states {
		byGroup[state.GroupID] = append(byGroup[state.GroupID], state)
	}
	for i := range summaries {
		summaries[i].AstraCheckStates = byGroup[summaries[i].GroupID]
	}
	return summaries, nil
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
			group_id, config_id, status, response_excerpt, latency_ms, http_code, sub_status, error_detail, observed_at, total_latency_ms, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	`, result.GroupID, result.ConfigID, result.Status, nullIfEmpty(result.ResponseExcerpt), result.LatencyMS,
		result.HTTPCode, result.SubStatus, nullIfEmpty(result.ErrorDetail), result.ObservedAt, result.TotalLatencyMS); err != nil {
		return nil, nil, err
	}

	prev, err := r.getStateForUpdate(ctx, tx, result.GroupID)
	if err != nil {
		return nil, nil, err
	}

	next, event := service.ComputeGroupStatusTransition(prev, result)

	// 只写存活探测自己的列（指纹验证的状态在 group_status_astra_check_states 表里）
	row := tx.QueryRowContext(ctx, `
		INSERT INTO group_status_states (
			group_id, config_id, latest_status, stable_status, response_excerpt, latency_ms, http_code,
			sub_status, error_detail, observed_at, consecutive_down, consecutive_non_down, total_latency_ms, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		ON CONFLICT (group_id) DO UPDATE SET
			config_id = EXCLUDED.config_id,
			latest_status = EXCLUDED.latest_status,
			stable_status = EXCLUDED.stable_status,
			response_excerpt = EXCLUDED.response_excerpt,
			latency_ms = EXCLUDED.latency_ms,
			total_latency_ms = EXCLUDED.total_latency_ms,
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
		next.ConsecutiveDown, next.ConsecutiveNonDown, next.TotalLatencyMS)
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
		SELECT id, group_id, config_id, status, response_excerpt, latency_ms, total_latency_ms, http_code, sub_status, error_detail, observed_at, created_at
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
		SELECT id, group_id, config_id, status, response_excerpt, latency_ms, total_latency_ms, http_code, sub_status, error_detail, observed_at, created_at
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

// ListDueAstraCheckConfigs 列出到期的指纹验证配置：分组仍是 OpenAI / Anthropic 平台、两个开关都开，
// 且配置的模型里至少有一个距上次检测超过间隔（或从未检测过）。具体跑哪些模型由 service 按状态再筛一遍。
func (r *groupStatusRepository) ListDueAstraCheckConfigs(ctx context.Context, now time.Time, limit int) ([]*service.GroupStatusConfig, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+aliasColumns("c", groupStatusConfigColumns)+`
		FROM group_status_configs c
		JOIN groups g ON g.id = c.group_id
		WHERE c.enabled = TRUE
		  AND c.astra_check_enabled = TRUE
		  AND g.platform = ANY($3)
		  AND (
		        (
		          jsonb_array_length(c.astra_check_models) = 0
		          AND NOT EXISTS (
		              SELECT 1 FROM group_status_astra_check_states st
		              WHERE st.group_id = c.group_id
		                AND st.checked_at > ($1::timestamptz - (c.astra_check_interval_seconds * INTERVAL '1 second'))
		          )
		        )
		        OR EXISTS (
		            SELECT 1
		            FROM jsonb_array_elements(c.astra_check_models) m
		            LEFT JOIN group_status_astra_check_states st
		              ON st.group_id = c.group_id AND st.expected_model = m->>'expected_model'
		            WHERE st.checked_at IS NULL
		               OR st.checked_at <= ($1::timestamptz - (c.astra_check_interval_seconds * INTERVAL '1 second'))
		        )
		      )
		ORDER BY (
		    SELECT MIN(st.checked_at) FROM group_status_astra_check_states st WHERE st.group_id = c.group_id
		) ASC NULLS FIRST, c.group_id ASC
		LIMIT $2
	`, now, limit, pq.Array([]string{service.PlatformOpenAI, service.PlatformAnthropic}))
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

// SaveAstraCheckRun 落一条运行记录、只更新该（分组, 预期模型）的指纹状态，稳定结论切换时写事件。
func (r *groupStatusRepository) SaveAstraCheckRun(ctx context.Context, result *service.GroupStatusAstraCheckResult) (*service.GroupStatusAstraCheckRun, *service.GroupStatusAstraCheckState, *service.GroupStatusEvent, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	round := result.Round
	if round <= 0 {
		round = 1
	}
	row := tx.QueryRowContext(ctx, `
		INSERT INTO group_status_astra_check_runs (
			group_id, config_id, platform, expected_model, round, benchmark_package_id, benchmark_version,
			benchmark_sha256, scoring_version, request_model, tier, account_id, verdict, winner_model,
			matches, cells, reasons, samples, requests_planned, requests_completed, valid_samples,
			input_tokens, output_tokens, reasoning_tokens, cost_usd, latency_ms, http_code, error_detail,
			started_at, finished_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
		        $15::jsonb, $16::jsonb, $17::jsonb, $18::jsonb, $19, $20, $21,
		        $22, $23, $24, $25, $26, $27, $28, $29, $30, NOW())
		RETURNING `+groupStatusAstraRunColumns+`
	`, result.GroupID, result.ConfigID, result.Platform, result.ExpectedModel, round, result.BenchmarkPackageID,
		result.BenchmarkVersion, result.BenchmarkSHA256, result.ScoringVersion, result.RequestModel, result.Tier,
		result.AccountID, result.Verdict, result.Winner,
		mustJSONArray(result.Matches), mustJSONArray(result.Cells), mustJSONArray(result.Reasons), mustJSONArray(result.Samples),
		result.RequestsPlanned, result.RequestsCompleted, result.ValidSamples,
		result.InputTokens, result.OutputTokens, result.ReasoningTokens, result.CostUSD, result.LatencyMS, result.HTTPCode,
		nullIfEmpty(result.ErrorDetail), result.StartedAt, result.FinishedAt)
	run, err := scanGroupStatusAstraCheckRun(row)
	if err != nil {
		return nil, nil, nil, err
	}

	prev, err := r.getAstraCheckStateForUpdate(ctx, tx, result.GroupID, result.ExpectedModel)
	if err != nil {
		return nil, nil, nil, err
	}

	next, event := service.ComputeAstraCheckTransition(prev, result, run.ID)

	row = tx.QueryRowContext(ctx, `
		INSERT INTO group_status_astra_check_states (
			group_id, config_id, expected_model, verdict, stable_status, winner_model, matches, reasons,
			detail, checked_at, consecutive_mismatch, valid_samples, planned_samples, input_tokens,
			output_tokens, reasoning_tokens, last_cost_usd, last_run_id, benchmark_package_id, benchmark_version,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), NOW())
		ON CONFLICT (group_id, expected_model) DO UPDATE SET
			config_id = EXCLUDED.config_id,
			verdict = EXCLUDED.verdict,
			stable_status = EXCLUDED.stable_status,
			winner_model = EXCLUDED.winner_model,
			matches = EXCLUDED.matches,
			reasons = EXCLUDED.reasons,
			detail = EXCLUDED.detail,
			checked_at = EXCLUDED.checked_at,
			consecutive_mismatch = EXCLUDED.consecutive_mismatch,
			valid_samples = EXCLUDED.valid_samples,
			planned_samples = EXCLUDED.planned_samples,
			input_tokens = EXCLUDED.input_tokens,
			output_tokens = EXCLUDED.output_tokens,
			reasoning_tokens = EXCLUDED.reasoning_tokens,
			last_cost_usd = EXCLUDED.last_cost_usd,
			last_run_id = EXCLUDED.last_run_id,
			benchmark_package_id = EXCLUDED.benchmark_package_id,
			benchmark_version = EXCLUDED.benchmark_version,
			updated_at = NOW()
		RETURNING `+groupStatusAstraStateColumns+`
	`, next.GroupID, next.ConfigID, next.ExpectedModel, next.Verdict, next.StableStatus, next.Winner,
		mustJSONArray(next.Matches), mustJSONArray(next.Reasons), nullIfEmpty(next.Detail), next.CheckedAt,
		next.ConsecutiveMismatch, next.ValidSamples, next.PlannedSamples, next.InputTokens, next.OutputTokens,
		next.ReasoningTokens, next.LastCostUSD, next.LastRunID, next.BenchmarkPackageID, next.BenchmarkVersion)
	savedState, err := scanGroupStatusAstraCheckState(row)
	if err != nil {
		return nil, nil, nil, err
	}

	if event != nil {
		event, err = insertGroupStatusEvent(ctx, tx, event)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, nil, err
	}
	tx = nil
	return run, savedState, event, nil
}

// ListLatestAstraCheckRuns 返回分组每个预期模型最近一次运行的完整记录。
func (r *groupStatusRepository) ListLatestAstraCheckRuns(ctx context.Context, groupID int64) ([]service.GroupStatusAstraCheckRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (expected_model) `+groupStatusAstraRunColumns+`
		FROM group_status_astra_check_runs
		WHERE group_id = $1
		ORDER BY expected_model, finished_at DESC, id DESC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.GroupStatusAstraCheckRun, 0)
	for rows.Next() {
		run, err := scanGroupStatusAstraCheckRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *run)
	}
	return out, rows.Err()
}

func (r *groupStatusRepository) DeleteAstraCheckRunsOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM group_status_astra_check_runs WHERE finished_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ListAstraCheckStates 返回这些分组全部预期模型的指纹状态。
func (r *groupStatusRepository) ListAstraCheckStates(ctx context.Context, groupIDs []int64) ([]service.GroupStatusAstraCheckState, error) {
	if len(groupIDs) == 0 {
		return []service.GroupStatusAstraCheckState{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+groupStatusAstraStateColumns+`
		FROM group_status_astra_check_states
		WHERE group_id = ANY($1)
		ORDER BY group_id ASC, expected_model ASC
	`, pq.Array(groupIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.GroupStatusAstraCheckState, 0)
	for rows.Next() {
		state, err := scanGroupStatusAstraCheckState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *state)
	}
	return out, rows.Err()
}

// DeleteAstraCheckStatesExcept 删掉分组里已不在检测列表中的模型状态。
func (r *groupStatusRepository) DeleteAstraCheckStatesExcept(ctx context.Context, groupID int64, keepModels []string) error {
	if keepModels == nil {
		keepModels = []string{}
	}
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM group_status_astra_check_states
		WHERE group_id = $1 AND NOT (expected_model = ANY($2))
	`, groupID, pq.Array(keepModels))
	return err
}

func (r *groupStatusRepository) getAstraCheckStateForUpdate(ctx context.Context, tx *sql.Tx, groupID int64, expectedModel string) (*service.GroupStatusAstraCheckState, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT `+groupStatusAstraStateColumns+`
		FROM group_status_astra_check_states
		WHERE group_id = $1 AND expected_model = $2
		FOR UPDATE
	`, groupID, expectedModel)
	state, err := scanGroupStatusAstraCheckState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

func scanGroupStatusAstraCheckRun(row scannable) (*service.GroupStatusAstraCheckRun, error) {
	run := &service.GroupStatusAstraCheckRun{}
	var accountID sql.NullInt64
	var matchesRaw, cellsRaw, reasonsRaw, samplesRaw []byte
	var latency sql.NullInt64
	var httpCode sql.NullInt64
	var errorDetail sql.NullString
	if err := row.Scan(
		&run.ID, &run.GroupID, &run.ConfigID, &run.Platform, &run.ExpectedModel, &run.Round, &run.BenchmarkPackageID,
		&run.BenchmarkVersion, &run.BenchmarkSHA256, &run.ScoringVersion, &run.RequestModel, &run.Tier, &accountID,
		&run.Verdict, &run.Winner, &matchesRaw, &cellsRaw, &reasonsRaw, &samplesRaw,
		&run.RequestsPlanned, &run.RequestsCompleted, &run.ValidSamples, &run.InputTokens, &run.OutputTokens,
		&run.ReasoningTokens, &run.CostUSD, &latency, &httpCode, &errorDetail, &run.StartedAt, &run.FinishedAt, &run.CreatedAt,
	); err != nil {
		return nil, err
	}
	if len(samplesRaw) > 0 {
		_ = json.Unmarshal(samplesRaw, &run.Samples)
	}
	if run.Samples == nil {
		run.Samples = []service.AstraCheckSampleRecord{}
	}
	if accountID.Valid {
		v := accountID.Int64
		run.AccountID = &v
	}
	run.Matches = decodeAstraMatches(matchesRaw)
	if len(cellsRaw) > 0 {
		_ = json.Unmarshal(cellsRaw, &run.Cells)
	}
	if run.Cells == nil {
		run.Cells = []service.AstraCheckCellSummary{}
	}
	run.Reasons = decodeJSONStrings(reasonsRaw)
	if latency.Valid {
		v := latency.Int64
		run.LatencyMS = &v
	}
	if httpCode.Valid {
		v := int(httpCode.Int64)
		run.HTTPCode = &v
	}
	run.ErrorDetail = errorDetail.String
	return run, nil
}

func scanGroupStatusAstraCheckState(row scannable) (*service.GroupStatusAstraCheckState, error) {
	state := &service.GroupStatusAstraCheckState{}
	var matchesRaw, reasonsRaw []byte
	var detail sql.NullString
	var checkedAt sql.NullTime
	var lastRunID sql.NullInt64
	if err := row.Scan(
		&state.ID, &state.GroupID, &state.ConfigID, &state.ExpectedModel, &state.Verdict, &state.StableStatus,
		&state.Winner, &matchesRaw, &reasonsRaw, &detail, &checkedAt, &state.ConsecutiveMismatch,
		&state.ValidSamples, &state.PlannedSamples, &state.InputTokens, &state.OutputTokens, &state.ReasoningTokens,
		&state.LastCostUSD, &lastRunID, &state.BenchmarkPackageID, &state.BenchmarkVersion, &state.CreatedAt, &state.UpdatedAt,
	); err != nil {
		return nil, err
	}
	state.Matches = decodeAstraMatches(matchesRaw)
	state.Reasons = decodeJSONStrings(reasonsRaw)
	state.Detail = detail.String
	if checkedAt.Valid {
		v := checkedAt.Time
		state.CheckedAt = &v
	}
	if lastRunID.Valid {
		v := lastRunID.Int64
		state.LastRunID = &v
	}
	return state, nil
}

func decodeAstraMatches(raw []byte) []service.AstraCheckModelMatch {
	out := []service.AstraCheckModelMatch{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []service.AstraCheckModelMatch{}
	}
	return out
}

func decodeJSONStrings(raw []byte) []string {
	out := []string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []string{}
	}
	return out
}

// mustJSONArray 把切片编码成 JSON 数组；nil 也输出 []，避免 jsonb 列存成 null。
func mustJSONArray(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return []byte("[]")
	}
	return raw
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
	var keywordsRaw, astraModelsRaw []byte
	cfg := &service.GroupStatusConfig{}
	if err := row.Scan(
		&cfg.ID, &cfg.GroupID, &cfg.Enabled, &cfg.ProbeModel, &cfg.ProbePrompt, &cfg.ValidationMode, &keywordsRaw,
		&cfg.IntervalSeconds, &cfg.TimeoutSeconds, &cfg.SlowLatencyMS, &cfg.NotifyEnabled,
		&cfg.AstraCheckEnabled, &astraModelsRaw, &cfg.AstraCheckTier, &cfg.AstraCheckIntervalSeconds,
		&cfg.CreatedAt, &cfg.UpdatedAt,
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
	cfg.AstraCheckModels = decodeAstraCheckModels(astraModelsRaw)
	return cfg, nil
}

func decodeAstraCheckModels(raw []byte) []service.AstraCheckModelConfig {
	out := []service.AstraCheckModelConfig{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []service.AstraCheckModelConfig{}
	}
	return out
}

func scanGroupStatusState(row scannable) (*service.GroupStatusState, error) {
	state := &service.GroupStatusState{}
	var responseExcerpt sql.NullString
	var latency sql.NullInt64
	var httpCode sql.NullInt64
	var subStatus sql.NullString
	var errorDetail sql.NullString
	var observedAt sql.NullTime
	var totalLatency sql.NullInt64
	if err := row.Scan(
		&state.ID, &state.GroupID, &state.ConfigID, &state.LatestStatus, &state.StableStatus, &responseExcerpt,
		&latency, &httpCode, &subStatus, &errorDetail, &observedAt, &state.ConsecutiveDown,
		&state.ConsecutiveNonDown, &totalLatency,
		&state.CreatedAt, &state.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if totalLatency.Valid {
		v := totalLatency.Int64
		state.TotalLatencyMS = &v
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
	return state, nil
}

func scanGroupStatusRecord(row scannable) (*service.GroupStatusRecord, error) {
	record := &service.GroupStatusRecord{}
	var responseExcerpt sql.NullString
	var latency sql.NullInt64
	var totalLatency sql.NullInt64
	var httpCode sql.NullInt64
	var subStatus sql.NullString
	var errorDetail sql.NullString
	if err := row.Scan(
		&record.ID, &record.GroupID, &record.ConfigID, &record.Status, &responseExcerpt, &latency, &totalLatency, &httpCode,
		&subStatus, &errorDetail, &record.ObservedAt, &record.CreatedAt,
	); err != nil {
		return nil, err
	}
	record.ResponseExcerpt = responseExcerpt.String
	if latency.Valid {
		v := latency.Int64
		record.LatencyMS = &v
	}
	if totalLatency.Valid {
		v := totalLatency.Int64
		record.TotalLatencyMS = &v
	}
	if httpCode.Valid {
		v := int(httpCode.Int64)
		record.HTTPCode = &v
	}
	record.SubStatus = subStatus.String
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
	var totalLatency sql.NullInt64
	var astraModelsRaw []byte
	if err := row.Scan(
		&item.GroupID, &item.ConfigID, &item.Enabled, &item.ProbeModel,
		&item.LatestStatus, &item.StableStatus, &item.ResponseExcerpt, &latency, &httpCode, &totalLatency,
		&item.SubStatus, &item.ErrorDetail, &observedAt, &item.ConsecutiveDown, &item.ConsecutiveNonDown,
		&item.AstraCheckEnabled, &astraModelsRaw, &item.AstraCheckTier, &item.AstraCheckIntervalSeconds,
	); err != nil {
		return nil, err
	}
	if totalLatency.Valid {
		v := totalLatency.Int64
		item.TotalLatencyMS = &v
	}
	item.AstraCheckModels = decodeAstraCheckModels(astraModelsRaw)
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
