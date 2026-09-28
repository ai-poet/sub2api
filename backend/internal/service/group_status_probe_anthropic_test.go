package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 存活探测走 buildAnthropicMessagesProbeRequest 的最小头部模式后，请求必须与重构前完全一致。
func TestGroupStatusProbe_AnthropicLivenessRequestUnchanged(t *testing.T) {
	group := &Group{ID: 20, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}
	cfg := groupStatusProbeConfig(group.ID, "claude-sonnet-4-5")
	account := groupStatusProbeAccount(1, PlatformAnthropic, group.ID, 1, map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5-mapped"})
	upstream := &groupStatusProbeHTTPUpstream{responses: []*http.Response{
		groupStatusProbeResponse(http.StatusOK, "data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"ONLINE\"}}\n\ndata: {\"type\":\"message_stop\"}\n\n"),
	}}
	svc := newGroupStatusProbeServiceForTest(group, []Account{account}, nil, upstream, nil)

	execution, err := svc.executeProbe(context.Background(), group, cfg)
	require.NoError(t, err)
	require.Equal(t, GroupRuntimeStatusUp, execution.Result.Status)
	require.Len(t, upstream.requests, 1)

	req := upstream.requests[0]
	require.Equal(t, "https://example.com/v1/messages?beta=true", req.URL.String())
	require.Equal(t, "test-key", req.Header.Get("x-api-key"))
	require.Equal(t, "2023-06-01", req.Header.Get("anthropic-version"))
	require.Equal(t, "text/event-stream", req.Header.Get("accept"))
	require.Empty(t, req.Header.Get("anthropic-beta"))
	require.Empty(t, req.Header.Get("Authorization"))

	payload := decodeProbeRequestBody(t, req)
	require.Equal(t, "claude-sonnet-4-5-mapped", payload["model"])
	require.Equal(t, float64(64), payload["max_tokens"])
	require.Equal(t, float64(0), payload["temperature"])
	require.Equal(t, true, payload["stream"])
	system := payload["system"].([]any)[0].(map[string]any)
	require.Equal(t, claudeCodeSystemPrompt, system["text"])
	require.Equal(t, map[string]any{"type": "ephemeral"}, system["cache_control"])
}

func TestParseAnthropicMessagesStream(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"message_start","message":{"model":"claude-opus-5-5","usage":{"input_tokens":120,"cache_read_input_tokens":8,"output_tokens":1}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"hmm"}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"12, "}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"34"}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":42}}`,
		`data: {"type":"message_stop"}`,
	}, "\n\n") + "\n\n"
	result, err := parseAnthropicMessagesStream(strings.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, "12, 34", result.Text)
	require.Equal(t, "end_turn", result.StopReason)
	require.Equal(t, "claude-opus-5-5", result.Model)
	require.Equal(t, int64(120), result.Usage.InputTokens)
	require.Equal(t, int64(8), result.Usage.CacheReadInputTokens)
	require.Equal(t, int64(42), result.Usage.OutputTokens)
	require.True(t, result.SawThinking)
	require.True(t, result.Completed)

	truncated := `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"1, 2"}}` + "\n\n"
	result, err = parseAnthropicMessagesStream(strings.NewReader(truncated))
	require.NoError(t, err)
	require.False(t, result.Completed)
	require.Equal(t, "1, 2", result.Text)

	failed := `data: {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}` + "\n\n"
	_, err = parseAnthropicMessagesStream(strings.NewReader(failed))
	require.ErrorContains(t, err, "Overloaded")
}
