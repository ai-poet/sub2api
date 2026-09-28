package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// 这些用例钉住 meow 指纹验证（多模型）三处容易漂移的地方：列清单常量、INSERT 占位符个数、scanner 目标个数。
// sqlmock 在列数与 Scan 目标数不一致时直接报错，所以只要这里的行按常量的列名构造，漂移就会让用例失败。

func columnNames(columns string) []string {
	parts := strings.Split(columns, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

func anyArgs(n int) []driver.Value {
	out := make([]driver.Value, n)
	for i := range out {
		out[i] = sqlmock.AnyArg()
	}
	return out
}

func rowFor(t *testing.T, columns string, values map[string]driver.Value) *sqlmock.Rows {
	t.Helper()
	names := columnNames(columns)
	require.Len(t, values, len(names), "test row must provide exactly one value per column")
	row := make([]driver.Value, len(names))
	for i, name := range names {
		value, ok := values[name]
		require.True(t, ok, "missing test value for column %s", name)
		row[i] = value
	}
	return sqlmock.NewRows(names).AddRow(row...)
}

func astraConfigRow(t *testing.T, now time.Time, models string) *sqlmock.Rows {
	return rowFor(t, groupStatusConfigColumns, map[string]driver.Value{
		"id": int64(3), "group_id": int64(7), "enabled": true, "probe_model": "gpt-6-sol", "probe_prompt": "ping",
		"validation_mode": "non_empty", "expected_keywords": []byte(`[]`), "interval_seconds": int64(60),
		"timeout_seconds": int64(30), "slow_latency_ms": int64(15000), "notify_enabled": true,
		"astra_check_enabled": true, "astra_check_models": []byte(models), "astra_check_tier": "medium",
		"astra_check_interval_seconds": int64(1800), "created_at": now, "updated_at": now,
	})
}

func astraStateValues(now time.Time, model string, mismatches int64) map[string]driver.Value {
	return map[string]driver.Value{
		"id": int64(11), "group_id": int64(7), "config_id": int64(3), "expected_model": model, "verdict": "mismatch",
		"stable_status": "pass", "winner_model": "gpt-6-astra",
		"matches": []byte(`[{"model":"gpt-6-astra","name":"GPT-6 Astra","score":-3.2,"match":0.97,"threshold":0.56,"passed":true}]`),
		"reasons": []byte(`[]`), "detail": "expected GPT-6 Sol", "checked_at": now, "consecutive_mismatch": mismatches,
		"valid_samples": int64(32), "planned_samples": int64(32), "input_tokens": int64(1600), "output_tokens": int64(320),
		"reasoning_tokens": int64(64), "last_cost_usd": 0.0123, "last_run_id": int64(41),
		"benchmark_package_id": "meow-gpt-other-cap98-efficient", "benchmark_version": "4.5.4-predictive.20260924.2",
		"created_at": now, "updated_at": now,
	}
}

func TestGroupStatusRepositoryUpsertConfigRoundTripsAstraModels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Now()
	models := `[{"expected_model":"gpt-6-sol","request_model":""},{"expected_model":"gpt-6-astra","request_model":"astra-alias"}]`

	args := anyArgs(14)
	args[11] = []byte(models) // $12::jsonb astra_check_models
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO group_status_configs")).
		WithArgs(args...).
		WillReturnRows(astraConfigRow(t, now, models))

	repo := NewGroupStatusRepository(db)
	cfg, err := repo.UpsertConfig(context.Background(), &service.GroupStatusConfig{
		GroupID: 7, Enabled: true, ProbeModel: "gpt-6-sol", ProbePrompt: "ping", ValidationMode: "non_empty",
		IntervalSeconds: 60, TimeoutSeconds: 30, SlowLatencyMS: 15000, NotifyEnabled: true,
		AstraCheckEnabled: true, AstraCheckTier: "medium", AstraCheckIntervalSeconds: 1800,
		AstraCheckModels: []service.AstraCheckModelConfig{
			{ExpectedModel: "gpt-6-sol"},
			{ExpectedModel: "gpt-6-astra", RequestModel: "astra-alias"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []service.AstraCheckModelConfig{
		{ExpectedModel: "gpt-6-sol"},
		{ExpectedModel: "gpt-6-astra", RequestModel: "astra-alias"},
	}, cfg.AstraCheckModels)
	require.Equal(t, "medium", cfg.AstraCheckTier)
	require.Equal(t, 1800, cfg.AstraCheckIntervalSeconds)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupStatusRepositorySaveAstraCheckRunUpsertsPerModelState(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Now()
	accountID := int64(5)

	result := &service.GroupStatusAstraCheckResult{
		GroupID: 7, ConfigID: 3, Platform: service.PlatformOpenAI, ExpectedModel: "gpt-6-sol", Round: 1,
		BenchmarkPackageID: "meow-gpt-other-cap98-efficient", BenchmarkVersion: "4.5.4-predictive.20260924.2",
		ScoringVersion: "meow-fingerprint-v3-predictive", RequestModel: "gpt-6-sol", Tier: "low", AccountID: &accountID,
		Verdict: service.AstraCheckVerdictMismatch, Winner: "gpt-6-astra", Strongest: "gpt-6-astra",
		Matches:        []service.AstraCheckModelMatch{{Model: "gpt-6-astra", Match: 0.97, Threshold: 0.56, Passed: true}},
		ValidSamples:   32,
		PlannedSamples: 32,
		CostUSD:        0.0123,
		StartedAt:      now.Add(-time.Minute),
		FinishedAt:     now,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO group_status_astra_check_runs")).
		WithArgs(anyArgs(30)...).
		WillReturnRows(rowFor(t, groupStatusAstraRunColumns, map[string]driver.Value{
			"id": int64(41), "group_id": int64(7), "config_id": int64(3), "platform": "openai", "expected_model": "gpt-6-sol",
			"round": int64(1), "benchmark_package_id": "meow-gpt-other-cap98-efficient",
			"benchmark_version": "4.5.4-predictive.20260924.2", "benchmark_sha256": "98f8d12c",
			"scoring_version": "meow-fingerprint-v3-predictive", "request_model": "gpt-6-sol", "tier": "low",
			"account_id": int64(5), "verdict": "mismatch", "winner_model": "gpt-6-astra",
			"matches": []byte(`[{"model":"gpt-6-astra","match":0.97}]`),
			"cells":   []byte(`[{"cell_id":"gpt__screen023","planned":1,"valid":1,"categories":{"43":1}}]`),
			"reasons": []byte(`[]`), "samples": []byte(`[{"seq":1,"cell_id":"gpt__screen023","answer":"43"}]`),
			"requests_planned": int64(32), "requests_completed": int64(32), "valid_samples": int64(32),
			"input_tokens": int64(1600), "output_tokens": int64(320), "reasoning_tokens": int64(64), "cost_usd": 0.0123,
			"latency_ms": int64(60000), "http_code": nil, "error_detail": nil,
			"started_at": now.Add(-time.Minute), "finished_at": now, "created_at": now,
		}))
	mock.ExpectQuery(regexp.QuoteMeta("FROM group_status_astra_check_states")).
		WithArgs(int64(7), "gpt-6-sol").
		WillReturnRows(sqlmock.NewRows(columnNames(groupStatusAstraStateColumns)))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO group_status_astra_check_states")).
		WithArgs(anyArgs(20)...).
		WillReturnRows(rowFor(t, groupStatusAstraStateColumns, astraStateValues(now, "gpt-6-sol", 1)))
	mock.ExpectCommit()

	repo := NewGroupStatusRepository(db)
	run, state, event, err := repo.SaveAstraCheckRun(context.Background(), result)
	require.NoError(t, err)
	require.Nil(t, event) // 第一次不符只计数，不切换稳定结论

	require.Equal(t, int64(41), run.ID)
	require.Equal(t, "gpt-6-sol", run.ExpectedModel)
	require.Equal(t, "openai", run.Platform)
	require.Equal(t, 1, run.Round)
	require.Equal(t, "meow-fingerprint-v3-predictive", run.ScoringVersion)
	require.Equal(t, 0.0123, run.CostUSD)
	require.Equal(t, int64(5), *run.AccountID)
	require.Nil(t, run.HTTPCode)
	require.Len(t, run.Cells, 1)
	require.Equal(t, 1, run.Cells[0].Categories["43"])
	require.Len(t, run.Samples, 1)
	require.Equal(t, []string{}, run.Reasons)

	require.Equal(t, "gpt-6-sol", state.ExpectedModel)
	require.Equal(t, 1, state.ConsecutiveMismatch)
	require.Equal(t, "gpt-6-astra", state.Winner)
	require.Len(t, state.Matches, 1)
	require.Equal(t, 0.97, state.Matches[0].Match)
	require.Equal(t, int64(41), *state.LastRunID)
	require.NotNil(t, state.CheckedAt)
	require.Equal(t, "4.5.4-predictive.20260924.2", state.BenchmarkVersion)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupStatusRepositorySaveAstraCheckRunWritesEventOnSecondMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Now()

	result := &service.GroupStatusAstraCheckResult{
		GroupID: 7, ConfigID: 3, Platform: service.PlatformOpenAI, ExpectedModel: "gpt-6-sol", Round: 2,
		Verdict: service.AstraCheckVerdictMismatch, Winner: "gpt-6-astra", FinishedAt: now,
	}

	mock.ExpectBegin()
	runValues := map[string]driver.Value{}
	for _, name := range columnNames(groupStatusAstraRunColumns) {
		runValues[name] = ""
	}
	for name, value := range map[string]driver.Value{
		"id": int64(42), "group_id": int64(7), "config_id": int64(3), "round": int64(2), "account_id": nil,
		"matches": []byte(`[]`), "cells": []byte(`[]`), "reasons": []byte(`[]`), "samples": []byte(`[]`),
		"requests_planned": int64(0), "requests_completed": int64(0), "valid_samples": int64(0),
		"input_tokens": int64(0), "output_tokens": int64(0), "reasoning_tokens": int64(0), "cost_usd": 0.0,
		"latency_ms": nil, "http_code": nil, "error_detail": nil, "started_at": now, "finished_at": now, "created_at": now,
	} {
		runValues[name] = value
	}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO group_status_astra_check_runs")).
		WithArgs(anyArgs(30)...).
		WillReturnRows(rowFor(t, groupStatusAstraRunColumns, runValues))
	mock.ExpectQuery(regexp.QuoteMeta("FROM group_status_astra_check_states")).
		WithArgs(int64(7), "gpt-6-sol").
		WillReturnRows(rowFor(t, groupStatusAstraStateColumns, astraStateValues(now, "gpt-6-sol", 1)))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO group_status_astra_check_states")).
		WithArgs(anyArgs(20)...).
		WillReturnRows(rowFor(t, groupStatusAstraStateColumns, astraStateValues(now, "gpt-6-sol", 2)))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO group_status_events")).
		WithArgs(int64(7), int64(3), service.GroupStatusEventAstraMismatch, "pass", service.AstraCheckStatusMismatch,
			sqlmock.AnyArg(), sqlmock.AnyArg(), "gpt-6-sol:winner_gpt-6-astra", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "group_id", "config_id", "event_type", "from_status", "to_status",
			"latency_ms", "http_code", "sub_status", "error_detail", "observed_at", "created_at"}).
			AddRow(int64(9), int64(7), int64(3), service.GroupStatusEventAstraMismatch, "pass", "mismatch",
				nil, nil, "gpt-6-sol:winner_gpt-6-astra", nil, now, now))
	mock.ExpectCommit()

	repo := NewGroupStatusRepository(db)
	_, _, event, err := repo.SaveAstraCheckRun(context.Background(), result)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, int64(9), event.ID)
	require.Equal(t, "gpt-6-sol:winner_gpt-6-astra", event.SubStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupStatusRepositoryListSummariesAttachesAstraStates(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("FROM group_status_configs c")).
		WillReturnRows(sqlmock.NewRows([]string{
			"group_id", "id", "enabled", "probe_model", "latest_status", "stable_status", "response_excerpt",
			"latency_ms", "http_code", "total_latency_ms", "sub_status", "error_detail", "observed_at",
			"consecutive_down", "consecutive_non_down",
			"astra_check_enabled", "astra_check_models", "astra_check_tier", "astra_check_interval_seconds",
		}).AddRow(int64(7), int64(3), true, "gpt-6-sol", "up", "up", "", nil, nil, nil, "", "", now, int64(0), int64(2),
			true, []byte(`[{"expected_model":"gpt-6-sol"},{"expected_model":"gpt-6-astra"}]`), "low", int64(3600)))

	states := sqlmock.NewRows(columnNames(groupStatusAstraStateColumns))
	for _, model := range []string{"gpt-6-astra", "gpt-6-sol"} {
		values := astraStateValues(now, model, 0)
		row := make([]driver.Value, 0, len(values))
		for _, name := range columnNames(groupStatusAstraStateColumns) {
			row = append(row, values[name])
		}
		states.AddRow(row...)
	}
	mock.ExpectQuery(regexp.QuoteMeta("FROM group_status_astra_check_states")).WillReturnRows(states)

	repo := NewGroupStatusRepository(db)
	summaries, err := repo.ListSummaries(context.Background(), []int64{7})
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	summary := summaries[0]
	require.True(t, summary.AstraCheckEnabled)
	require.Equal(t, []service.AstraCheckModelConfig{{ExpectedModel: "gpt-6-sol"}, {ExpectedModel: "gpt-6-astra"}}, summary.AstraCheckModels)
	require.Len(t, summary.AstraCheckStates, 2)
	require.Equal(t, "gpt-6-astra", summary.AstraCheckStates[0].ExpectedModel)

	raw, err := json.Marshal(summary.AstraCheckStates[1])
	require.NoError(t, err)
	require.Contains(t, string(raw), `"expected_model":"gpt-6-sol"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupStatusRepositoryDeleteAstraCheckStatesExceptKeepsListedModels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM group_status_astra_check_states")).
		WithArgs(int64(7), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewGroupStatusRepository(db)
	require.NoError(t, repo.DeleteAstraCheckStatesExcept(context.Background(), 7, nil))
	require.NoError(t, mock.ExpectationsWereMet())
}
