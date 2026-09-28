package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

// 分组运行状态的仓储是位置对应的原生 SQL：列清单、INSERT 占位符与 scanner 任何一处漏改都会在运行时才炸。
// 这里用 sqlmock 走一遍 ModelTrace 相关路径，钉住三者一致；真实库的往返由集成测试覆盖。

func splitColumnList(list string) []string {
	parts := strings.Split(list, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// splitTopLevel 按不在括号内的逗号切分 SELECT 列表。
func splitTopLevel(list string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range list {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(list[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(list[start:]))
}

var (
	fakeJSONColumns = map[string]bool{
		"expected_keywords": true, "ranking": true, "family_probabilities": true, "reasons": true, "outputs": true,
		"matches": true, "cells": true, "samples": true,
		"modeltrace_ranking": true, "modeltrace_reasons": true, "astra_check_matches": true, "astra_check_reasons": true,
	}
	fakeIntSuffixes = []string{"_id", "_tokens", "_ms", "_seconds", "_mismatch", "_down", "_outputs", "_samples",
		"_planned", "_made", "_completed", "http_code", "round", "calibration_queries"}
)

// baseColumnName 取表达式里的列名：去掉 COALESCE(...) 包装与表别名前缀。
func baseColumnName(expr string) string {
	name := strings.ToLower(strings.TrimSpace(expr))
	if i := strings.Index(name, "("); i >= 0 {
		name = name[i+1:]
		if j := strings.IndexAny(name, ",)"); j >= 0 {
			name = name[:j]
		}
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return strings.TrimSpace(name)
}

// fakeColumnValue 按列名给出能被对应 scanner 接受的值。
func fakeColumnValue(column string) driver.Value {
	name := baseColumnName(column)
	switch {
	case fakeJSONColumns[name]:
		return []byte("[]")
	case strings.HasSuffix(name, "_at"):
		return time.Now()
	case strings.Contains(name, "probability"), strings.HasSuffix(name, "cost_usd"), name == "beta":
		return 0.5
	case strings.HasSuffix(name, "enabled"):
		return true
	case name == "id":
		return int64(1)
	}
	for _, suffix := range fakeIntSuffixes {
		if strings.HasSuffix(name, suffix) {
			return int64(1)
		}
	}
	return "x"
}

func fakeRow(columns []string) *sqlmock.Rows {
	values := make([]driver.Value, len(columns))
	for i, column := range columns {
		values[i] = fakeColumnValue(column)
	}
	return sqlmock.NewRows(columns).AddRow(values...)
}

var placeholderPattern = regexp.MustCompile(`\$(\d+)`)

// assertInsertShape 校验 INSERT 的列数 = 占位符数 + NOW() 个数，且占位符连续编号到 wantArgs。
func assertInsertShape(t *testing.T, query string, wantArgs int) {
	t.Helper()
	open := strings.Index(query, "(")
	closeIdx := strings.Index(query, ")")
	require.Greater(t, closeIdx, open)
	columns := splitColumnList(query[open+1 : closeIdx])

	valuesAt := strings.Index(query, "VALUES")
	require.Greater(t, valuesAt, 0)
	valuesOpen := strings.Index(query[valuesAt:], "(") + valuesAt
	depth, valuesClose := 0, -1
	for i := valuesOpen; i < len(query); i++ {
		if query[i] == '(' {
			depth++
		} else if query[i] == ')' {
			depth--
			if depth == 0 {
				valuesClose = i
				break
			}
		}
	}
	require.Greater(t, valuesClose, valuesOpen)
	values := splitTopLevel(query[valuesOpen+1 : valuesClose])
	require.Len(t, values, len(columns), "INSERT column count must match VALUES count")

	maxPlaceholder := 0
	for _, m := range placeholderPattern.FindAllStringSubmatch(query[valuesOpen:valuesClose], -1) {
		n, _ := strconv.Atoi(m[1])
		if n > maxPlaceholder {
			maxPlaceholder = n
		}
	}
	require.Equal(t, wantArgs, maxPlaceholder)
}

func anyArgs(n int) []driver.Value {
	out := make([]driver.Value, n)
	for i := range out {
		out[i] = sqlmock.AnyArg()
	}
	return out
}

func TestSaveModelTraceRun_ColumnListsScannersAndPlaceholdersAgree(t *testing.T) {
	var captured []string
	matcher := sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
		captured = append(captured, actualSQL)
		return sqlmock.QueryMatcherRegexp.Match(expectedSQL, actualSQL)
	})
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	runColumns := splitColumnList(groupStatusModelTraceRunColumns)
	stateColumns := splitColumnList(groupStatusStateColumns)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO group_status_modeltrace_runs`).
		WithArgs(anyArgs(33)...).
		WillReturnRows(fakeRow(runColumns))
	mock.ExpectQuery(`FROM group_status_states\s+WHERE group_id = \$1\s+FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows(stateColumns))
	mock.ExpectQuery(`INSERT INTO group_status_states`).
		WithArgs(anyArgs(19)...).
		WillReturnRows(fakeRow(stateColumns))
	mock.ExpectCommit()

	p := 0.01
	result := &service.GroupStatusModelTraceResult{
		GroupID:             7,
		ConfigID:            3,
		Platform:            service.PlatformOpenAI,
		ExpectedModel:       "gpt-5.6-sol",
		RequestModel:        "gpt-5.6-sol",
		Verdict:             service.ModelTraceVerdictMatch,
		TopModel:            "gpt-5.6-sol",
		TopProbability:      0.99,
		ExpectedProbability: &p,
		Ranking:             []service.ModelTraceRankEntry{{Model: "gpt-5.6-sol", Probability: 0.99}},
		Outputs:             []service.ModelTraceOutputRecord{{Seq: 1, Accepted: true, Numbers: []int{1, 2, 3}}},
		StartedAt:           time.Now(),
		FinishedAt:          time.Now(),
	}
	repo := &groupStatusRepository{db: db}
	run, state, event, err := repo.SaveModelTraceRun(context.Background(), result)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.NotNil(t, state)
	require.Nil(t, event)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Len(t, captured, 3)
	assertInsertShape(t, captured[0], 33)
	assertInsertShape(t, captured[2], 19)
	// RETURNING 列表与 scanner 共用常量，fakeRow 已按常量构造；再确认两条 INSERT 的列都存在于迁移里
	for _, column := range append(append([]string{}, runColumns...), stateColumns...) {
		require.True(t, columnDefinedInMigrations(t, column), "column %s is not created by any migration", column)
	}
}

func TestListSummaries_SelectMatchesScanner(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	selectList := strings.TrimPrefix(groupStatusSummarySelect, "SELECT")
	selectList = selectList[:strings.Index(selectList, "FROM group_status_configs")]
	columns := splitTopLevel(selectList)
	mock.ExpectQuery(`FROM group_status_configs c`).WillReturnRows(fakeRow(columns))

	repo := &groupStatusRepository{db: db}
	summaries, err := repo.ListSummaries(context.Background(), []int64{7})
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.NotNil(t, summaries[0].ModelTraceRanking)
	require.NotNil(t, summaries[0].ModelTraceCheckedAt)
	require.NotNil(t, summaries[0].ModelTraceExpectedProbability)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertConfig_ModelTraceColumnsAgree(t *testing.T) {
	var captured string
	matcher := sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
		captured = actualSQL
		return sqlmock.QueryMatcherRegexp.Match(expectedSQL, actualSQL)
	})
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	configColumns := splitColumnList(groupStatusConfigColumns)
	mock.ExpectQuery(`INSERT INTO group_status_configs`).
		WithArgs(anyArgs(18)...).
		WillReturnRows(fakeRow(configColumns))

	repo := &groupStatusRepository{db: db}
	cfg, err := repo.UpsertConfig(context.Background(), &service.GroupStatusConfig{
		GroupID:                   7,
		ModelTraceEnabled:         true,
		ModelTraceExpectedModel:   "claude-opus-5-5",
		ModelTraceIntervalSeconds: 3600,
		ExpectedKeywords:          []string{},
	})
	require.NoError(t, err)
	require.True(t, cfg.ModelTraceEnabled)
	require.NoError(t, mock.ExpectationsWereMet())
	assertInsertShape(t, captured, 18)
	for _, column := range configColumns {
		require.True(t, columnDefinedInMigrations(t, column), "column %s is not created by any migration", column)
	}
}

var migrationsSQL string

func columnDefinedInMigrations(t *testing.T, column string) bool {
	t.Helper()
	if migrationsSQL == "" {
		entries, err := migrations.FS.ReadDir(".")
		require.NoError(t, err)
		var b strings.Builder
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			raw, err := migrations.FS.ReadFile(entry.Name())
			require.NoError(t, err)
			b.Write(raw)
			b.WriteString("\n")
		}
		migrationsSQL = b.String()
	}
	return regexp.MustCompile(`(?m)(^|[\s(,])` + regexp.QuoteMeta(column) + `\s+[A-Z]`).MatchString(migrationsSQL)
}
