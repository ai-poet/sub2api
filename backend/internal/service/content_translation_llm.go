package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	contentTranslationRequestTimeout = 120 * time.Second
	// 短文本按批翻译：每批最多这么多条 / 这么多字
	contentTranslationBatchItems = 40
	contentTranslationBatchRunes = 6000
	// 超过这个长度的文本单独发送，直接输出纯文本译文
	contentTranslationLongTextRunes = 2000
	contentTranslationMaxResponse   = 8 << 20
)

// contentTranslator 把一批同一目标语言的原文翻译成译文，返回值与输入一一对应。
type contentTranslator interface {
	Translate(ctx context.Context, creds *contentTranslationCredentials, lang string, texts []string) ([]string, error)
}

// chatCompletionsTranslator 通过网关自己的 /v1/chat/completions 翻译。
type chatCompletionsTranslator struct {
	client *http.Client
}

func newChatCompletionsTranslator() *chatCompletionsTranslator {
	return &chatCompletionsTranslator{client: &http.Client{Timeout: contentTranslationRequestTimeout}}
}

func contentTranslationSystemPrompt(lang string, batch bool) string {
	name := contentLanguageName(lang)
	rules := "You are a professional UI localization translator. Translate text written by a website administrator into " + name + ".\n" +
		"Rules:\n" +
		"- Keep Markdown syntax, HTML tags, line breaks, URLs, e-mail addresses, code, placeholders such as {name} / %{name} / %s, and numbers exactly as they are.\n" +
		"- Do not translate product, brand, company or AI model names (for example Claude, GPT-5, Gemini, Codex, DeepSeek); keep them verbatim.\n" +
		"- Keep the tone and length close to the original. Do not add explanations, notes or quotes.\n" +
		"- If a text is already written in " + name + ", return it unchanged.\n"
	if batch {
		return rules + "- The user message is a JSON array of strings. Reply with ONLY a JSON object of the form {\"t\": [...]} whose array has exactly the same number of items in the same order, each item being the translation of the corresponding input string."
	}
	return rules + "- The user message is the text to translate. Reply with ONLY the translated text."
}

// Translate 把 texts 翻成 lang。短文本合批（JSON 进 JSON 出，条数对不上时这一批改逐条），
// 长文本逐条发送。任一条失败即返回错误，调用方据此做退避。
func (t *chatCompletionsTranslator) Translate(ctx context.Context, creds *contentTranslationCredentials, lang string, texts []string) ([]string, error) {
	out := make([]string, len(texts))
	type pendingItem struct {
		index int
		text  string
	}
	var batch []pendingItem
	batchRunes := 0

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		items := make([]string, len(batch))
		for i, item := range batch {
			items[i] = item.text
		}
		translated, err := t.translateBatch(ctx, creds, lang, items)
		if err != nil {
			if !errors.Is(err, errContentTranslationBatchMismatch) {
				return err
			}
			// 模型没按条数返回：这一批改逐条，保证译文不会错位
			translated = make([]string, len(items))
			for i, item := range items {
				single, singleErr := t.translateSingle(ctx, creds, lang, item)
				if singleErr != nil {
					return singleErr
				}
				translated[i] = single
			}
		}
		for i, item := range batch {
			out[item.index] = translated[i]
		}
		batch = batch[:0]
		batchRunes = 0
		return nil
	}

	for i, text := range texts {
		runes := utf8.RuneCountInString(text)
		if runes > contentTranslationLongTextRunes {
			single, err := t.translateSingle(ctx, creds, lang, text)
			if err != nil {
				return nil, err
			}
			out[i] = single
			continue
		}
		if len(batch) >= contentTranslationBatchItems || (len(batch) > 0 && batchRunes+runes > contentTranslationBatchRunes) {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		batch = append(batch, pendingItem{index: i, text: text})
		batchRunes += runes
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return out, nil
}

var errContentTranslationBatchMismatch = errors.New("translation batch returned a different number of items")

func (t *chatCompletionsTranslator) translateBatch(ctx context.Context, creds *contentTranslationCredentials, lang string, items []string) ([]string, error) {
	if len(items) == 1 {
		single, err := t.translateSingle(ctx, creds, lang, items[0])
		if err != nil {
			return nil, err
		}
		return []string{single}, nil
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	content, err := t.complete(ctx, creds, contentTranslationSystemPrompt(lang, true), string(payload))
	if err != nil {
		return nil, err
	}
	return parseContentTranslationBatch(content, len(items))
}

func (t *chatCompletionsTranslator) translateSingle(ctx context.Context, creds *contentTranslationCredentials, lang, text string) (string, error) {
	content, err := t.complete(ctx, creds, contentTranslationSystemPrompt(lang, false), text)
	if err != nil {
		return "", err
	}
	translated := stripContentTranslationFence(strings.TrimSpace(content), text)
	if translated == "" {
		return "", errors.New("translation model returned an empty answer")
	}
	return translated, nil
}

// parseContentTranslationBatch 解析 {"t": [...]}；也接受模型直接返回的 JSON 数组，以及包在代码围栏里的 JSON。
func parseContentTranslationBatch(content string, want int) ([]string, error) {
	body := strings.TrimSpace(content)
	if strings.HasPrefix(body, "```") {
		body = strings.TrimPrefix(body, "```")
		if nl := strings.IndexByte(body, '\n'); nl >= 0 {
			body = body[nl+1:]
		}
		body = strings.TrimSuffix(strings.TrimSpace(body), "```")
		body = strings.TrimSpace(body)
	}
	var items []string
	var wrapped struct {
		T []string `json:"t"`
	}
	switch {
	case strings.HasPrefix(body, "{"):
		if err := json.Unmarshal([]byte(body), &wrapped); err != nil {
			return nil, errContentTranslationBatchMismatch
		}
		items = wrapped.T
	case strings.HasPrefix(body, "["):
		if err := json.Unmarshal([]byte(body), &items); err != nil {
			return nil, errContentTranslationBatchMismatch
		}
	default:
		return nil, errContentTranslationBatchMismatch
	}
	if len(items) != want {
		return nil, errContentTranslationBatchMismatch
	}
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
		if items[i] == "" {
			return nil, errContentTranslationBatchMismatch
		}
	}
	return items, nil
}

// stripContentTranslationFence 去掉模型自作主张加的整段代码围栏（原文本身就是围栏时保留）。
func stripContentTranslationFence(translated, source string) string {
	if !strings.HasPrefix(translated, "```") || strings.HasPrefix(strings.TrimSpace(source), "```") {
		return translated
	}
	body := strings.TrimPrefix(translated, "```")
	if nl := strings.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	} else {
		return translated
	}
	body = strings.TrimSpace(body)
	if !strings.HasSuffix(body, "```") {
		return translated
	}
	return strings.TrimSpace(strings.TrimSuffix(body, "```"))
}

type contentChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type contentChatRequest struct {
	Model    string               `json:"model"`
	Stream   bool                 `json:"stream"`
	Messages []contentChatMessage `json:"messages"`
}

// complete 发一次非流式 chat/completions。不传 temperature：推理类模型会拒绝这个参数。
func (t *chatCompletionsTranslator) complete(ctx context.Context, creds *contentTranslationCredentials, system, user string) (string, error) {
	body, err := json.Marshal(contentChatRequest{
		Model:  creds.Model,
		Stream: false,
		Messages: []contentChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, creds.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+creds.APIKey)
	req.Header.Set("User-Agent", "sub2api-content-translation/1")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("translation request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, contentTranslationMaxResponse))
	if err != nil {
		return "", fmt.Errorf("read translation response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("translation request returned HTTP %d: %s", resp.StatusCode, contentTranslationErrorMessage(raw))
	}
	return parseContentChatCompletion(raw)
}

// parseContentChatCompletion 取 choices[0].message.content；content 既可能是字符串，也可能是分段数组。
func parseContentChatCompletion(raw []byte) (string, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content json.RawMessage `json:"content"`
				Refusal string          `json:"refusal"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("translation response is not JSON: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("translation response has no choices")
	}
	choice := parsed.Choices[0]
	if strings.TrimSpace(choice.Message.Refusal) != "" {
		return "", errors.New("translation model refused: " + truncateContentTranslationText(choice.Message.Refusal, 200))
	}
	if choice.FinishReason == "length" {
		return "", errors.New("translation answer was cut off (finish_reason=length)")
	}
	content := choice.Message.Content
	var text string
	if len(content) > 0 && content[0] == '"' {
		if err := json.Unmarshal(content, &text); err != nil {
			return "", err
		}
		return text, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if len(content) > 0 && content[0] == '[' {
		if err := json.Unmarshal(content, &parts); err != nil {
			return "", err
		}
		var b strings.Builder
		for _, part := range parts {
			_, _ = b.WriteString(part.Text)
		}
		return b.String(), nil
	}
	return "", errors.New("translation response has no text content")
}

func contentTranslationErrorMessage(raw []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &parsed); err == nil {
		if msg := strings.TrimSpace(parsed.Error.Message); msg != "" {
			return truncateContentTranslationText(msg, 300)
		}
		if msg := strings.TrimSpace(parsed.Message); msg != "" {
			return truncateContentTranslationText(msg, 300)
		}
	}
	return truncateContentTranslationText(strings.TrimSpace(string(raw)), 300)
}

func truncateContentTranslationText(text string, maxRunes int) string {
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxRunes]) + "…"
}
