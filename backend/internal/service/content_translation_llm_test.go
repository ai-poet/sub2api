package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type ctChatServer struct {
	mu       sync.Mutex
	requests []contentChatRequest
	auth     []string
	reply    func(req contentChatRequest) (int, string)
}

func (s *ctChatServer) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req contentChatRequest
		require.NoError(t, json.Unmarshal(body, &req))
		require.NotContains(t, string(body), "temperature")
		s.mu.Lock()
		s.requests = append(s.requests, req)
		s.auth = append(s.auth, r.Header.Get("Authorization"))
		s.mu.Unlock()
		status, payload := s.reply(req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(payload))
	})
}

func ctChatAnswer(content string) string {
	data, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
	})
	return string(data)
}

func newCTChatServer(t *testing.T, reply func(req contentChatRequest) (int, string)) (*ctChatServer, *contentTranslationCredentials) {
	t.Helper()
	srv := &ctChatServer{reply: reply}
	ts := httptest.NewServer(srv.handler(t))
	t.Cleanup(ts.Close)
	return srv, &contentTranslationCredentials{Endpoint: ts.URL + "/v1/chat/completions", APIKey: "sk-test", Model: "cheap"}
}

func TestChatCompletionsTranslatorBatch(t *testing.T) {
	srv, creds := newCTChatServer(t, func(req contentChatRequest) (int, string) {
		var items []string
		require.NoError(t, json.Unmarshal([]byte(req.Messages[1].Content), &items))
		out := make([]string, len(items))
		for i, item := range items {
			out[i] = "EN " + item
		}
		payload, _ := json.Marshal(map[string]any{"t": out})
		return http.StatusOK, ctChatAnswer("```json\n" + string(payload) + "\n```")
	})
	out, err := newChatCompletionsTranslator().Translate(context.Background(), creds, "en", []string{"一", "二", "三"})
	require.NoError(t, err)
	require.Equal(t, []string{"EN 一", "EN 二", "EN 三"}, out)
	require.Len(t, srv.requests, 1, "short texts go out as one batch")
	require.Equal(t, "Bearer sk-test", srv.auth[0])
	require.Equal(t, "cheap", srv.requests[0].Model)
	require.False(t, srv.requests[0].Stream)
	require.Contains(t, srv.requests[0].Messages[0].Content, "English")
}

func TestChatCompletionsTranslatorBatchMismatchFallsBackToSingles(t *testing.T) {
	srv, creds := newCTChatServer(t, func(req contentChatRequest) (int, string) {
		if strings.HasPrefix(req.Messages[1].Content, "[") {
			return http.StatusOK, ctChatAnswer(`{"t":["only one"]}`)
		}
		return http.StatusOK, ctChatAnswer("EN " + req.Messages[1].Content)
	})
	out, err := newChatCompletionsTranslator().Translate(context.Background(), creds, "en", []string{"一", "二"})
	require.NoError(t, err)
	require.Equal(t, []string{"EN 一", "EN 二"}, out)
	require.Len(t, srv.requests, 3)
}

func TestChatCompletionsTranslatorLongTextGoesAlone(t *testing.T) {
	long := strings.Repeat("长", contentTranslationLongTextRunes+1)
	srv, creds := newCTChatServer(t, func(req contentChatRequest) (int, string) {
		if req.Messages[1].Content == long {
			// 模型自作主张包了代码围栏：应被剥掉
			return http.StatusOK, ctChatAnswer("```\nEN long\n```")
		}
		return http.StatusOK, ctChatAnswer("EN short")
	})
	out, err := newChatCompletionsTranslator().Translate(context.Background(), creds, "en", []string{long, "短"})
	require.NoError(t, err)
	require.Equal(t, []string{"EN long", "EN short"}, out)
	require.Len(t, srv.requests, 2)
}

func TestChatCompletionsTranslatorSurfacesUpstreamError(t *testing.T) {
	_, creds := newCTChatServer(t, func(contentChatRequest) (int, string) {
		return http.StatusForbidden, `{"error":{"message":"model not allowed for this group"}}`
	})
	_, err := newChatCompletionsTranslator().Translate(context.Background(), creds, "ja", []string{"一"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP 403")
	require.Contains(t, err.Error(), "model not allowed for this group")
}

func TestParseContentChatCompletion(t *testing.T) {
	text, err := parseContentChatCompletion([]byte(`{"choices":[{"message":{"content":[{"type":"text","text":"Hel"},{"type":"text","text":"lo"}]}}]}`))
	require.NoError(t, err)
	require.Equal(t, "Hello", text)

	_, err = parseContentChatCompletion([]byte(`{"choices":[{"message":{"content":"x"},"finish_reason":"length"}]}`))
	require.Error(t, err)
	_, err = parseContentChatCompletion([]byte(`{"choices":[{"message":{"refusal":"no"}}]}`))
	require.Error(t, err)
	_, err = parseContentChatCompletion([]byte(`{"choices":[]}`))
	require.Error(t, err)
}

func TestParseContentTranslationBatch(t *testing.T) {
	items, err := parseContentTranslationBatch(`["a","b"]`, 2)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, items)
	_, err = parseContentTranslationBatch(`{"t":["a",""]}`, 2)
	require.ErrorIs(t, err, errContentTranslationBatchMismatch)
	_, err = parseContentTranslationBatch(`Sure! here you go`, 1)
	require.ErrorIs(t, err, errContentTranslationBatchMismatch)
}

func TestStripContentTranslationFence(t *testing.T) {
	require.Equal(t, "hello", stripContentTranslationFence("```\nhello\n```", "你好"))
	require.Equal(t, "```go\nx\n```", stripContentTranslationFence("```go\nx\n```", "```go\nx\n```"))
	require.Equal(t, "plain", stripContentTranslationFence("plain", "原文"))
}
