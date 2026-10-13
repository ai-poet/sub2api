package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// fork：对话模型审计 —— OpenAI 引擎走 /chat/completions，模型以 JSON 返回各类别分数。

// writeChatModerationScores 模拟对话模型的审计回复。
func writeChatModerationScores(w http.ResponseWriter, model string, scores map[string]float64) {
	if scores == nil {
		scores = map[string]float64{"sexual": 0}
	}
	content, _ := json.Marshal(scores)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"model":   model,
		"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": string(content)}}},
	})
}

func decodeChatModerationRequest(t *testing.T, r *http.Request) chatModerationRequest {
	t.Helper()
	var req chatModerationRequest
	require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
	return req
}

// chatModerationAuditedText 取出请求里「待审内容」的文字。
func chatModerationAuditedText(t *testing.T, r *http.Request) string {
	t.Helper()
	req := decodeChatModerationRequest(t, r)
	require.Len(t, req.Messages, 2)
	text, ok := req.Messages[1].Content.(string)
	if !ok {
		parts, _ := req.Messages[1].Content.([]any)
		require.NotEmpty(t, parts)
		text, _ = parts[0].(map[string]any)["text"].(string)
	}
	start, end := strings.Index(text, "<<<\n"), strings.LastIndex(text, "\n>>>")
	require.True(t, start >= 0 && end > start, "audited text is fenced: %q", text)
	return text[start+4 : end]
}

func TestChatCompletionsURL(t *testing.T) {
	for base, want := range map[string]string{
		"https://api.openai.com":               "https://api.openai.com/v1/chat/completions",
		"https://api.deepseek.com/":            "https://api.deepseek.com/v1/chat/completions",
		"https://gateway.example/v1":           "https://gateway.example/v1/chat/completions",
		"https://open.bigmodel.cn/api/paas/v4": "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		"http://127.0.0.1:8080":                "http://127.0.0.1:8080/v1/chat/completions",
	} {
		got, err := chatCompletionsURL(base)
		require.NoError(t, err)
		require.Equal(t, want, got, base)
	}
}

func TestParseChatModerationScores(t *testing.T) {
	scores, err := parseChatModerationScores("```json\n{\"sexual\": 0.92, \"violence\": \"0.3\", \"hate\": 7, \"harassment\": -1, \"unknown\": 1}\n```")
	require.NoError(t, err)
	require.InDelta(t, 0.92, scores["sexual"], 1e-9)
	require.InDelta(t, 0.3, scores["violence"], 1e-9)
	require.Equal(t, 1.0, scores["hate"], "clamped to 1")
	require.Zero(t, scores["harassment"], "clamped to 0")
	require.Zero(t, scores["self-harm"], "missing categories count as 0")
	require.NotContains(t, scores, "unknown")
	require.Len(t, scores, len(ContentModerationCategories()))

	nested, err := parseChatModerationScores(`{"scores": {"sexual/minors": 0.8}}`)
	require.NoError(t, err)
	require.InDelta(t, 0.8, nested["sexual/minors"], 1e-9)

	for _, bad := range []string{"", "我无法判断", `{"verdict": "safe"}`, `{"sexual": `} {
		_, err := parseChatModerationScores(bad)
		require.Error(t, err, bad)
	}
}

func TestChatModeration_RequestCarriesRulesTextAndImages(t *testing.T) {
	var got chatModerationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer sk-chat", r.Header.Get("Authorization"))
		got = decodeChatModerationRequest(t, r)
		writeChatModerationScores(w, "deepseek-v4.1-flash", map[string]float64{"violence": 0.7})
	}))
	defer server.Close()
	cfg := defaultContentModerationConfig()
	cfg.BaseURL = server.URL
	s := &ContentModerationService{httpClient: server.Client()}

	input := ContentModerationInput{Text: "look at this", Images: []string{"data:image/png;base64,AAAA"}}.ModerationInput()
	result, err := s.callChatModeration(context.Background(), cfg, "sk-chat", input, nil)
	require.NoError(t, err)

	require.Equal(t, defaultContentModerationModel, got.Model)
	require.Equal(t, map[string]string{"type": "json_object"}, got.ResponseFormat)
	require.Equal(t, "system", got.Messages[0].Role)
	system := got.Messages[0].Content.(string)
	for _, category := range ContentModerationCategories() {
		require.Contains(t, system, category)
	}
	parts := got.Messages[1].Content.([]any)
	require.Len(t, parts, 2)
	require.Contains(t, parts[0].(map[string]any)["text"], "look at this")
	require.Equal(t, "data:image/png;base64,AAAA", parts[1].(map[string]any)["image_url"].(map[string]any)["url"])

	require.InDelta(t, 0.7, result.CategoryScores["violence"], 1e-9)
	require.Equal(t, &ContentModerationEngineMeta{Engine: ContentModerationEngineOpenAI, Model: "deepseek-v4.1-flash", RulesVersion: ChatModerationRulesVersion}, result.EngineMeta)
}

func TestChatModeration_ModelWithoutVisionFallsBackToText(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		req := decodeChatModerationRequest(t, r)
		if _, multimodal := req.Messages[1].Content.([]any); multimodal {
			http.Error(w, `{"error":{"message":"image input is not supported"}}`, http.StatusBadRequest)
			return
		}
		writeChatModerationScores(w, "text-only", map[string]float64{"sexual": 0.1})
	}))
	defer server.Close()
	cfg := defaultContentModerationConfig()
	cfg.BaseURL = server.URL
	s := &ContentModerationService{httpClient: server.Client()}

	input := ContentModerationInput{Text: "hello", Images: []string{"https://img.example/a.png"}}.ModerationInput()
	result, err := s.callChatModeration(context.Background(), cfg, "sk", input, nil)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
	require.Equal(t, 1, result.EngineMeta.SkippedImages)

	// 只有图片时没有文字可退回，照常报错。
	_, err = s.callChatModeration(context.Background(), cfg, "sk", ContentModerationInput{Images: []string{"https://img.example/a.png"}}.ModerationInput(), nil)
	var statusErr *chatModerationStatusError
	require.ErrorAs(t, err, &statusErr)
	require.Equal(t, http.StatusBadRequest, statusErr.status)
}

func TestContentModerationConfig_MigratesModerationAPIDefaults(t *testing.T) {
	cfg, err := parseContentModerationConfig(`{"engine":"openai","mode":"observe","base_url":"https://api.openai.com","model":"omni-moderation-latest","timeout_ms":3000,"block_status":403}`)
	require.NoError(t, err)
	require.Equal(t, "deepseek-v4.1-flash", cfg.Model)
	require.Equal(t, 15000, cfg.TimeoutMS)

	cfg, err = parseContentModerationConfig(`{"engine":"openai","mode":"observe","model":"deepseek-v4-pro","timeout_ms":8000,"block_status":403}`)
	require.NoError(t, err)
	require.Equal(t, "deepseek-v4-pro", cfg.Model, "a chosen chat model is kept")
	require.Equal(t, 8000, cfg.TimeoutMS)

	require.Equal(t, typeSafeDefaultTimeoutMS, moderationEngineDefaults(ContentModerationEngineTypeSafe).TimeoutMS)
}

type stubModerationChatFallback struct {
	base, key string
	err       error
}

func (f stubModerationChatFallback) ModerationChatEndpoint(context.Context) (string, string, error) {
	return f.base, f.key, f.err
}

func TestTwoStageModeration_VerifierBorrowsTranslationEndpoint(t *testing.T) {
	var gotKey, gotModel string
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		gotKey = r.Header.Get("Authorization")
		gotModel = decodeChatModerationRequest(t, r).Model
		writeChatModerationScores(w, gotModel, map[string]float64{"sexual": 0.9})
	}))
	defer gateway.Close()

	cfg := defaultContentModerationConfig()
	cfg.TypeSafe = moderationEngineDefaults(ContentModerationEngineTypeSafe)
	cfg.TypeSafe.APIKeys = []string{"typesafe-key"}
	proxyID := int64(9)
	cfg.ProxyID = &proxyID
	pipeline := cfg.moderationPipeline()
	require.Empty(t, pipeline.verifier.apiKeys())

	s := &ContentModerationService{httpClient: gateway.Client()}
	content := ContentModerationInput{Text: "flagged by typesafe"}
	v := s.verifyModeration(context.Background(), pipeline.verifier, content, false)
	require.Equal(t, ContentModerationVerificationUnavailable, v.Status, "no keys and no fallback")
	require.Empty(t, s.verifierSource(context.Background(), pipeline))

	s.SetVerifierChatFallback(stubModerationChatFallback{base: gateway.URL, key: "sk-admin-gateway"})
	v = s.verifyModeration(context.Background(), pipeline.verifier, content, false)
	require.Equal(t, ContentModerationVerificationConfirmed, v.Status)
	require.Equal(t, ContentModerationVerifierSourceTranslation, v.Source)
	require.Equal(t, "Bearer sk-admin-gateway", gotKey)
	require.Equal(t, "deepseek-v4.1-flash", gotModel, "the model comes from the audit config, not translation")
	require.Equal(t, ContentModerationVerifierSourceTranslation, s.verifierSource(context.Background(), pipeline))
	require.Equal(t, &proxyID, pipeline.verifier.ProxyID, "borrowing must not touch the shared verifier config")

	s.SetVerifierChatFallback(stubModerationChatFallback{err: errors.New("translation key missing")})
	v = s.verifyModeration(context.Background(), pipeline.verifier, content, false)
	require.Equal(t, ContentModerationVerificationUnavailable, v.Status)

	cfg.APIKeys = []string{"own-key"}
	require.Equal(t, ContentModerationVerifierSourceKeys, s.verifierSource(context.Background(), cfg.moderationPipeline()))
}
