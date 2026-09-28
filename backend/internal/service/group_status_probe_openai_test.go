package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// responsesSSEBody 生成一条 Responses 流：一个 output_text 增量 + response.completed（带 usage）。
func responsesSSEBody(answer string, reasoning int64) string {
	var b strings.Builder
	if answer != "" {
		b.WriteString(fmt.Sprintf("data: {\"type\":\"response.output_text.delta\",\"delta\":%q}\n\n", answer))
	}
	b.WriteString(fmt.Sprintf(
		"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":52,\"output_tokens\":%d,\"output_tokens_details\":{\"reasoning_tokens\":%d}}}}\n\n",
		reasoning+3, reasoning,
	))
	return b.String()
}

func decodeProbeRequestBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

func TestParseOpenAIResponsesStream_CollectsTextAndUsage(t *testing.T) {
	text, usage, completed, err := parseOpenAIResponsesStreamDetailed(strings.NewReader(responsesSSEBody("40", 192)), true)
	require.NoError(t, err)
	require.True(t, completed)
	require.Equal(t, "40", text)
	require.Equal(t, int64(52), usage.InputTokens)
	require.Equal(t, int64(195), usage.OutputTokens)
	require.Equal(t, int64(192), usage.ReasoningTokens)
}

func TestParseOpenAIResponsesStream_FallsBackToFinalOutput(t *testing.T) {
	body := "data: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"type\":\"reasoning\"},{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"32\"}]}],\"usage\":{\"input_tokens\":50,\"output_tokens\":20}}}\n\n"
	text, usage, err := parseOpenAIResponsesStream(strings.NewReader(body), true)
	require.NoError(t, err)
	require.Equal(t, "32", text)
	require.Equal(t, int64(50), usage.InputTokens)
	require.Equal(t, int64(20), usage.OutputTokens)
	require.Equal(t, int64(0), usage.ReasoningTokens)
}

func TestParseOpenAIResponsesStream_FailedResponse(t *testing.T) {
	body := "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"upstream exploded\"}}}\n\n"
	_, _, err := parseOpenAIResponsesStream(strings.NewReader(body), true)
	require.ErrorContains(t, err, "upstream exploded")

	body = "data: {\"type\":\"error\",\"error\":{\"message\":\"bad request\"}}\n\n"
	_, _, err = parseOpenAIResponsesStream(strings.NewReader(body), true)
	require.ErrorContains(t, err, "bad request")
}

func TestParseOpenAIResponsesStream_IncompleteHandling(t *testing.T) {
	body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"1, 2\"}\n\n" +
		"data: {\"type\":\"response.incomplete\",\"response\":{\"incomplete_details\":{\"reason\":\"max_output_tokens\"}}}\n\n"

	// Astra 把撞上 max_output_tokens 的回答照常计入
	text, _, completed, err := parseOpenAIResponsesStreamDetailed(strings.NewReader(body), false)
	require.NoError(t, err)
	require.True(t, completed)
	require.Equal(t, "1, 2", text)

	// ModelTrace 把截断视为无效回答
	_, _, completed, err = parseOpenAIResponsesStreamDetailed(strings.NewReader(body), true)
	require.ErrorContains(t, err, "max_output_tokens")
	require.False(t, completed)
}

func TestParseOpenAIResponsesStream_EOFWithoutCompletedIsNotCompleted(t *testing.T) {
	body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"12, 34\"}\n\n"
	text, _, completed, err := parseOpenAIResponsesStreamDetailed(strings.NewReader(body), true)
	require.NoError(t, err)
	require.False(t, completed)
	require.Equal(t, "12, 34", text)
}
