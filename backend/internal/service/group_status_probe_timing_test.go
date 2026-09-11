package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProbeTiming_FirstTokenAndHeaders(t *testing.T) {
	var nilTiming *probeTiming
	nilTiming.markHeaders()
	nilTiming.markFirstToken()
	_, ok := nilTiming.firstTokenMS()
	require.False(t, ok)
	require.Nil(t, probeTimingFromContext(context.Background()))

	started := time.Now().Add(-50 * time.Millisecond)
	timing := newProbeTiming(started)
	_, ok = timing.firstTokenMS()
	require.False(t, ok)
	_, ok = timing.headersMS()
	require.False(t, ok)

	ctx := withProbeTiming(context.Background(), timing)
	require.Same(t, timing, probeTimingFromContext(ctx))

	timing.markHeaders()
	timing.markFirstToken()
	first := timing.firstToken
	time.Sleep(2 * time.Millisecond)
	timing.markFirstToken() // 只记第一次
	require.Equal(t, first, timing.firstToken)

	ms, ok := timing.firstTokenMS()
	require.True(t, ok)
	require.GreaterOrEqual(t, ms, int64(50))
	headers, ok := timing.headersMS()
	require.True(t, ok)
	require.LessOrEqual(t, headers, ms)
}

// 三个平台的流解析器都在第一段内容到达时回调一次，且不影响拼出的文本。
func TestParseProbeStreams_FirstTokenCallbackOnce(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		parser func(body io.Reader, onFirstToken func()) (string, error)
		want   string
	}{
		{
			name: "openai",
			body: "data: {\"type\":\"response.created\"}\n\n" +
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\"ON\"}\n\n" +
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\"LINE\"}\n\n" +
				"data: {\"type\":\"response.completed\",\"response\":{}}\n\n",
			parser: parseOpenAIProbeStream,
			want:   "ONLINE",
		},
		{
			name: "claude",
			body: "data: {\"type\":\"message_start\"}\n\n" +
				"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"ON\"}}\n\n" +
				"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"LINE\"}}\n\n" +
				"data: {\"type\":\"message_stop\"}\n\n",
			parser: parseClaudeProbeStream,
			want:   "ONLINE",
		},
		{
			name: "gemini",
			body: "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ON\"}]}}]}\n\n" +
				"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"LINE\"}]},\"finishReason\":\"STOP\"}]}\n\n",
			parser: parseGeminiProbeStream,
			want:   "ONLINE",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			text, err := tc.parser(strings.NewReader(tc.body), func() { calls++ })
			require.NoError(t, err)
			require.Equal(t, tc.want, text)
			require.Equal(t, 1, calls)

			// 回调为 nil 也不能 panic
			text, err = tc.parser(strings.NewReader(tc.body), nil)
			require.NoError(t, err)
			require.Equal(t, tc.want, text)
		})
	}
}

// 存活探测的延迟取首字时间，总耗时另存；首字一定不晚于总耗时。
func TestExecuteAccountProbe_LatencyIsFirstToken(t *testing.T) {
	group := &Group{ID: 40, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}
	account := groupStatusProbeAccount(1, PlatformOpenAI, group.ID, 1, map[string]any{"gpt-5.6-sol": "gpt-5.6-sol"})
	account.Credentials["api_key"] = "sk-test"
	upstream := &groupStatusProbeHTTPUpstream{responses: []*http.Response{
		groupStatusProbeResponse(200, solJuiceSSEBody("ONLINE", 5)),
	}}
	svc := newGroupStatusProbeServiceForTest(group, []Account{account}, nil, upstream, nil)
	cfg := groupStatusProbeConfig(group.ID, "gpt-5.6-sol")

	result, err := svc.executeAccountProbe(context.Background(), &account, cfg)
	require.NoError(t, err)
	require.Equal(t, GroupRuntimeStatusUp, result.Status)
	require.Equal(t, "ONLINE", result.ResponseExcerpt)
	require.NotNil(t, result.LatencyMS)
	require.NotNil(t, result.TotalLatencyMS)
	require.LessOrEqual(t, *result.LatencyMS, *result.TotalLatencyMS)
	require.Len(t, upstream.requests, 1)
}

// 上游报错时没有首字，延迟退回总耗时且仍然有值。
func TestExecuteAccountProbe_ErrorFallsBackToTotalLatency(t *testing.T) {
	group := &Group{ID: 41, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}
	account := groupStatusProbeAccount(1, PlatformOpenAI, group.ID, 1, map[string]any{"gpt-5.6-sol": "gpt-5.6-sol"})
	account.Credentials["api_key"] = "sk-test"
	upstream := &groupStatusProbeHTTPUpstream{responses: []*http.Response{
		groupStatusProbeResponse(503, `{"error":{"message":"down"}}`),
	}}
	svc := newGroupStatusProbeServiceForTest(group, []Account{account}, nil, upstream, nil)
	cfg := groupStatusProbeConfig(group.ID, "gpt-5.6-sol")

	result, err := svc.executeAccountProbe(context.Background(), &account, cfg)
	require.Error(t, err)
	require.Equal(t, GroupRuntimeStatusDown, result.Status)
	require.NotNil(t, result.LatencyMS)
	require.NotNil(t, result.TotalLatencyMS)
	require.Equal(t, *result.TotalLatencyMS, *result.LatencyMS)
}
