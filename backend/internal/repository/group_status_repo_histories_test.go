package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGroupStatusRepo_ListRecordStatusesSince_GroupsRowsByGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	since := time.Now().Add(-24 * time.Hour)
	observed := since.Add(time.Hour)
	latency := int64(820)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE group_id = ANY($1) AND observed_at >= $2")).
		WithArgs(sqlmock.AnyArg(), since).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "status", "latency_ms", "observed_at"}).
			AddRow(int64(1), "up", latency, observed).
			AddRow(int64(1), "down", nil, observed.Add(time.Minute)).
			AddRow(int64(3), "degraded", latency, observed))

	repo := &groupStatusRepository{db: db}
	got, err := repo.ListRecordStatusesSince(context.Background(), []int64{1, 3}, since)
	require.NoError(t, err)
	require.Len(t, got[1], 2)
	require.Len(t, got[3], 1)
	require.Equal(t, "down", got[1][1].Status)
	require.Nil(t, got[1][1].LatencyMS)
	require.Equal(t, latency, *got[3][0].LatencyMS)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupStatusRepo_ListRecordStatusesSince_EmptyIDsSkipsQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := &groupStatusRepository{db: db}
	got, err := repo.ListRecordStatusesSince(context.Background(), nil, time.Now())
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}
