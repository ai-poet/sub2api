//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 上游 issue 7873：原生 /v1/messages 流在 message_delta 之前结束（上游 EOF、读错误、超时）时，
// message_start 已经上报的输出用量不能记成 0；message_delta 的 output_tokens 是累计值，
// 到达后整体覆盖起始值，不能相加。

const (
	startWithOutputEvent = `{"type":"message_start","message":{"id":"msg_repro","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"usage":{"input_tokens":2,"output_tokens":8,"cache_read_input_tokens":1000}}}`
	cumulativeDeltaEvent = `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":20}}`
)

func startOutputParsers() map[string]func(string, *ClaudeUsage) {
	svc := newMinimalGatewayService()
	return map[string]func(string, *ClaudeUsage){
		"standard":    svc.parseSSEUsage,
		"passthrough": parseSSEUsagePassthrough,
	}
}

func TestMessageStartOutputUsage_RetainedWithoutDelta(t *testing.T) {
	for name, parse := range startOutputParsers() {
		t.Run(name, func(t *testing.T) {
			usage := &ClaudeUsage{}
			parse(startWithOutputEvent, usage)

			require.Equal(t, 8, usage.OutputTokens)
			require.Equal(t, 2, usage.InputTokens)
			require.Equal(t, 1000, usage.CacheReadInputTokens)
		})
	}
}

func TestMessageStartOutputUsage_CumulativeDeltaOverwritesStart(t *testing.T) {
	for name, parse := range startOutputParsers() {
		t.Run(name, func(t *testing.T) {
			usage := &ClaudeUsage{}
			parse(startWithOutputEvent, usage)
			parse(cumulativeDeltaEvent, usage)

			require.Equal(t, 20, usage.OutputTokens, "message_delta 是累计值，应覆盖起始值而不是 8+20")
		})
	}
}

func TestMessageStartOutputUsage_ZeroDeltaKeepsStart(t *testing.T) {
	for name, parse := range startOutputParsers() {
		t.Run(name, func(t *testing.T) {
			usage := &ClaudeUsage{}
			parse(startWithOutputEvent, usage)
			parse(`{"type":"message_delta","usage":{"output_tokens":0}}`, usage)

			require.Equal(t, 8, usage.OutputTokens)
		})
	}
}

func TestMessageStartOutputUsage_IgnoresMissingZeroAndNegative(t *testing.T) {
	starts := map[string]string{
		"missing":  `{"type":"message_start","message":{"usage":{"input_tokens":5}}}`,
		"zero":     `{"type":"message_start","message":{"usage":{"input_tokens":5,"output_tokens":0}}}`,
		"negative": `{"type":"message_start","message":{"usage":{"input_tokens":5,"output_tokens":-3}}}`,
	}
	for parserName, parse := range startOutputParsers() {
		for startName, start := range starts {
			t.Run(parserName+"/"+startName, func(t *testing.T) {
				usage := &ClaudeUsage{}
				parse(start, usage)
				require.Equal(t, 0, usage.OutputTokens)
				require.Equal(t, 5, usage.InputTokens)

				// 没有可用的起始值时，也不能把已有的输出用量清零
				usage.OutputTokens = 4
				parse(start, usage)
				require.Equal(t, 4, usage.OutputTokens)
			})
		}
	}
}

func newStartOutputTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c, rec
}

func sseBody(events ...string) io.ReadCloser {
	var b strings.Builder
	for _, event := range events {
		b.WriteString("data: ")
		b.WriteString(event)
		b.WriteString("\n\n")
	}
	return io.NopCloser(strings.NewReader(b.String()))
}

func TestHandleStreamingResponse_StartOutputRetainedWhenStreamEnds(t *testing.T) {
	t.Run("read error after message_start", func(t *testing.T) {
		svc := newMinimalGatewayService()
		c, _ := newStartOutputTestContext()
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       &streamReadCloser{payload: []byte("data: " + startWithOutputEvent + "\n\n"), err: io.ErrUnexpectedEOF},
		}

		result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)

		require.Error(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.usage)
		require.Equal(t, 8, result.usage.OutputTokens)
		require.Equal(t, 2, result.usage.InputTokens)
		require.Equal(t, 1000, result.usage.CacheReadInputTokens)
	})

	t.Run("upstream closes before terminal event", func(t *testing.T) {
		svc := newMinimalGatewayService()
		c, _ := newStartOutputTestContext()
		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: sseBody(startWithOutputEvent)}

		result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)

		require.Error(t, err)
		require.Contains(t, err.Error(), "missing terminal event")
		require.NotNil(t, result)
		require.Equal(t, 8, result.usage.OutputTokens)
	})

	t.Run("complete stream settles the cumulative delta once", func(t *testing.T) {
		svc := newMinimalGatewayService()
		c, _ := newStartOutputTestContext()
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       sseBody(startWithOutputEvent, cumulativeDeltaEvent, `{"type":"message_stop"}`),
		}

		result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 20, result.usage.OutputTokens)
	})
}

func TestAnthropicAPIKeyPassthrough_StartOutputRetainedWhenStreamEnds(t *testing.T) {
	const model = "claude-sonnet-4-5"

	t.Run("upstream closes before terminal event", func(t *testing.T) {
		svc := newMinimalGatewayService()
		c, rec := newStartOutputTestContext()
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       sseBody(startWithOutputEvent),
		}

		result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), model)

		require.Error(t, err)
		require.Contains(t, err.Error(), "missing terminal event")
		require.NotNil(t, result)
		require.Equal(t, 8, result.usage.OutputTokens)
		require.Equal(t, 2, result.usage.InputTokens)
		// 透传路径原样转发上游字节
		require.Contains(t, rec.Body.String(), `"output_tokens":8`)
	})

	t.Run("complete stream settles the cumulative delta once", func(t *testing.T) {
		svc := newMinimalGatewayService()
		c, _ := newStartOutputTestContext()
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       sseBody(startWithOutputEvent, cumulativeDeltaEvent, `{"type":"message_stop"}`),
		}

		result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), model)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 20, result.usage.OutputTokens)
	})
}
