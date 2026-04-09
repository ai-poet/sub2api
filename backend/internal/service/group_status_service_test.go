package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildGroupStatusHistory_AvgLatencyIgnoresMissingSamples(t *testing.T) {
	start := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	step := time.Hour

	latency100 := int64(100)
	latency200 := int64(200)

	buckets := buildGroupStatusHistory([]GroupStatusRecord{
		{
			Status:     GroupRuntimeStatusDown,
			ObservedAt: start.Add(5 * time.Minute),
		},
		{
			Status:     GroupRuntimeStatusUp,
			LatencyMS:  &latency100,
			ObservedAt: start.Add(15 * time.Minute),
		},
		{
			Status:     GroupRuntimeStatusDegraded,
			LatencyMS:  &latency200,
			ObservedAt: start.Add(25 * time.Minute),
		},
	}, start, end, step)

	require.Len(t, buckets, 1)
	require.NotNil(t, buckets[0].AvgLatencyMS)
	require.InDelta(t, 150.0, *buckets[0].AvgLatencyMS, 0.001)
	require.Equal(t, 3, buckets[0].TotalCount)
	require.Equal(t, 1, buckets[0].DownCount)
	require.InDelta(t, 66.6667, buckets[0].Availability, 0.001)
}
