package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// 分组运行状态的 OpenAI Responses 探测通道（本 fork 自有功能）。
//
// meow 指纹验证使用：这里只处理账号模型映射、API-Key 与 Codex OAuth 的地址与头部，
// 请求体与流解析由调用方提供；与存活探测是两条独立的路径。

// openAIProbeUsage 是 Responses 流 response.completed 里的用量摘要。
type openAIProbeUsage struct {
	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
}

// openAIResponsesProbeRequest 发一条流式 Responses 探测请求：处理账号模型映射、API-Key 与
// Codex OAuth 的地址与头部，请求体和流解析由调用方提供。
func (s *GroupStatusProbeService) openAIResponsesProbeRequest(
	ctx context.Context,
	account *Account,
	requestModel string,
	buildPayload func(modelID string, isOAuth bool) map[string]any,
	parser func(io.Reader) (string, openAIProbeUsage, error),
) (string, openAIProbeUsage, *int, error) {
	var usage openAIProbeUsage
	if s.accountTestSvc == nil {
		return "", usage, nil, errors.New("account test service is not configured")
	}
	if account == nil {
		return "", usage, nil, errors.New("nil account")
	}

	modelID := strings.TrimSpace(requestModel)
	if modelID == "" {
		return "", usage, nil, errors.New("request model is empty")
	}
	if account.Type == AccountTypeAPIKey {
		if mapping := account.GetModelMapping(); len(mapping) > 0 {
			if mapped, ok := mapping[modelID]; ok {
				modelID = mapped
			}
		}
	}

	var authToken string
	var apiURL string
	var isOAuth bool
	var chatgptAccountID string
	if account.IsOAuth() {
		isOAuth = true
		authToken = account.GetOpenAIAccessToken()
		if authToken == "" {
			return "", usage, nil, errors.New("no access token available")
		}
		apiURL = chatgptCodexAPIURL
		chatgptAccountID = account.GetChatGPTAccountID()
	} else if account.Type == AccountTypeAPIKey {
		authToken = account.GetOpenAIApiKey()
		if authToken == "" {
			return "", usage, nil, errors.New("no API key available")
		}
		normalizedBaseURL, err := s.accountTestSvc.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
		if err != nil {
			return "", usage, nil, fmt.Errorf("invalid base URL: %w", err)
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/responses"
	} else {
		return "", usage, nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}

	payloadBytes, _ := json.Marshal(buildPayload(modelID, isOAuth))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", usage, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("accept", "text/event-stream")
	if isOAuth {
		req.Host = "chatgpt.com"
		if chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	return s.executeStreamingProbeWithUsage(req, account, parser)
}

func (s *GroupStatusProbeService) executeStreamingProbeWithUsage(req *http.Request, account *Account, parser func(io.Reader) (string, openAIProbeUsage, error)) (string, openAIProbeUsage, *int, error) {
	resp, err := s.doHTTPRequest(req, account)
	if err != nil {
		return "", openAIProbeUsage{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	code := resp.StatusCode
	if code < 200 || code >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return string(body), openAIProbeUsage{}, &code, newProbeUpstreamError(account, code, body)
	}
	text, usage, err := parser(resp.Body)
	return text, usage, &code, err
}

// parseOpenAIResponsesStream 是 Responses SSE 的通用解析：incompleteIsError 为 false 时，
// response.incomplete（如撞到 max_output_tokens）照常返回已收到的文本与 usage。
func parseOpenAIResponsesStream(body io.Reader, incompleteIsError bool) (string, openAIProbeUsage, error) {
	text, usage, _, err := parseOpenAIResponsesStreamDetailed(body, incompleteIsError)
	return text, usage, err
}

// parseOpenAIResponsesStreamDetailed 与 parseOpenAIResponsesStream 相同，另外报告流是否以
// response.completed（或 incompleteIsError=false 时的 response.incomplete）正常收尾；
// 流在 EOF / [DONE] 处中断时 completed 为 false。
func parseOpenAIResponsesStreamDetailed(body io.Reader, incompleteIsError bool) (string, openAIProbeUsage, bool, error) {
	reader := bufio.NewReader(body)
	var parts []string
	var usage openAIProbeUsage
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return strings.Join(parts, ""), usage, false, nil
			}
			return "", usage, false, err
		}
		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}
		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			return strings.Join(parts, ""), usage, false, nil
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}
		switch data["type"] {
		case "response.output_text.delta":
			if delta, ok := data["delta"].(string); ok && delta != "" {
				parts = append(parts, delta)
			}
		case "response.completed":
			resp, _ := data["response"].(map[string]any)
			usage = parseOpenAIResponseUsage(resp)
			text := strings.Join(parts, "")
			if strings.TrimSpace(text) == "" {
				text = extractOpenAIResponseOutputText(resp)
			}
			return text, usage, true, nil
		case "response.failed", "response.incomplete":
			resp, _ := data["response"].(map[string]any)
			usage = parseOpenAIResponseUsage(resp)
			if data["type"] == "response.incomplete" && !incompleteIsError {
				text := strings.Join(parts, "")
				if strings.TrimSpace(text) == "" {
					text = extractOpenAIResponseOutputText(resp)
				}
				return text, usage, true, nil
			}
			return strings.Join(parts, ""), usage, false, errors.New(openAIResponseFailureMessage(resp, fmt.Sprintf("openai probe %v", data["type"])))
		case "error":
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok && msg != "" {
					return strings.Join(parts, ""), usage, false, errors.New(msg)
				}
			}
			return strings.Join(parts, ""), usage, false, errors.New("openai probe failed")
		}
	}
}

func parseOpenAIResponseUsage(resp map[string]any) openAIProbeUsage {
	var usage openAIProbeUsage
	if resp == nil {
		return usage
	}
	raw, ok := resp["usage"].(map[string]any)
	if !ok {
		return usage
	}
	usage.InputTokens = jsonNumberToInt64(raw["input_tokens"])
	usage.OutputTokens = jsonNumberToInt64(raw["output_tokens"])
	if details, ok := raw["output_tokens_details"].(map[string]any); ok {
		usage.ReasoningTokens = jsonNumberToInt64(details["reasoning_tokens"])
	}
	return usage
}

func jsonNumberToInt64(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n
		}
	}
	return 0
}

func extractOpenAIResponseOutputText(resp map[string]any) string {
	if resp == nil {
		return ""
	}
	output, ok := resp["output"].([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, item := range output {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		contents, ok := itemMap["content"].([]any)
		if !ok {
			continue
		}
		for _, content := range contents {
			contentMap, ok := content.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] != "output_text" {
				continue
			}
			if text, ok := contentMap["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "")
}

func openAIResponseFailureMessage(resp map[string]any, fallback string) string {
	if resp != nil {
		if errData, ok := resp["error"].(map[string]any); ok {
			if msg, ok := errData["message"].(string); ok && strings.TrimSpace(msg) != "" {
				return msg
			}
		}
		if details, ok := resp["incomplete_details"].(map[string]any); ok {
			if reason, ok := details["reason"].(string); ok && strings.TrimSpace(reason) != "" {
				return fallback + ": " + reason
			}
		}
	}
	return fallback
}
