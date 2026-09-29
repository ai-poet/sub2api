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

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

// 分组运行状态的 Anthropic Messages 探测通道（本 fork 自有功能）。
//
// 地址与鉴权与存活探测 probeAnthropic 一致；头部按 headerMode 选择：存活探测保持最小头部，
// meow 指纹验证与账号测试一样带上 Claude Code 客户端头部（OAuth 凭证必须有 oauth beta）。

type anthropicProbeHeaderMode int

const (
	// anthropicProbeHeadersMinimal：只带 anthropic-version / accept，存活探测沿用至今的头部
	anthropicProbeHeadersMinimal anthropicProbeHeaderMode = iota
	// anthropicProbeHeadersClaudeCode：与 AccountTestService 相同的 Claude Code 客户端头部
	anthropicProbeHeadersClaudeCode
)

// anthropicProbeUsage 是 Messages 流里 message_start / message_delta 报告的用量。
type anthropicProbeUsage struct {
	InputTokens              int64
	OutputTokens             int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
}

// anthropicProbeResult 是一次 Messages 流的解析结果。
type anthropicProbeResult struct {
	Text        string
	StopReason  string
	Model       string
	Usage       anthropicProbeUsage
	SawThinking bool
	// Completed 表示流以 message_stop 正常收尾
	Completed bool
}

// buildAnthropicMessagesProbeRequest 生成一条发往账号上游的 Messages 请求。
// Bedrock 与其他账号类型返回错误；API-Key 账号按模型映射改写请求模型。
func (s *GroupStatusProbeService) buildAnthropicMessagesProbeRequest(
	ctx context.Context,
	account *Account,
	requestModel string,
	mode anthropicProbeHeaderMode,
	buildPayload func(modelID string, isOAuth bool) (map[string]any, error),
) (*http.Request, error) {
	if s.accountTestSvc == nil {
		return nil, errors.New("account test service is not configured")
	}
	if account == nil {
		return nil, errors.New("nil account")
	}
	if account.IsBedrock() {
		return nil, errors.New("bedrock accounts are not supported by this probe")
	}

	modelID := requestModel
	if account.Type == AccountTypeAPIKey {
		modelID = account.GetMappedModel(modelID)
	}

	var authToken string
	var useBearer bool
	var apiURL string

	if account.IsOAuth() {
		useBearer = true
		apiURL = testClaudeAPIURL
		authToken = account.GetCredential("access_token")
		if authToken == "" {
			return nil, errors.New("no access token available")
		}
	} else if account.Type == AccountTypeAPIKey {
		authToken = account.GetCredential("api_key")
		if authToken == "" {
			return nil, errors.New("no API key available")
		}
		baseURL := account.GetBaseURL()
		normalizedBaseURL, err := s.accountTestSvc.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/v1/messages?beta=true"
	} else {
		return nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}

	payload, err := buildPayload(modelID, useBearer)
	if err != nil {
		return nil, err
	}
	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	if mode == anthropicProbeHeadersClaudeCode {
		for key, value := range claude.DefaultHeaders() {
			req.Header.Set(key, value)
		}
		req.Header.Set("accept", "text/event-stream")
		if useBearer {
			req.Header.Set("anthropic-beta", claude.DefaultBetaHeader)
			req.Header.Set("Authorization", "Bearer "+authToken)
		} else {
			req.Header.Set("anthropic-beta", claude.APIKeyBetaHeader)
			setAnthropicAPIKeyAuthHeader(req.Header, account, authToken, account.GetBaseURL())
		}
		account.ApplyHeaderOverrides(req.Header)
		return req, nil
	}

	req.Header.Set("accept", "text/event-stream")
	if useBearer {
		req.Header.Set("Authorization", "Bearer "+authToken)
	} else {
		req.Header.Set("x-api-key", authToken)
	}
	return req, nil
}

// executeAnthropicStreamingProbe 发出请求并解析 Messages 流；非 2xx 返回上游错误与状态码。
func (s *GroupStatusProbeService) executeAnthropicStreamingProbe(req *http.Request, account *Account, parser func(io.Reader) (anthropicProbeResult, error)) (anthropicProbeResult, *int, error) {
	resp, err := s.doHTTPRequest(req, account)
	if err != nil {
		return anthropicProbeResult{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	code := resp.StatusCode
	if code < 200 || code >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return anthropicProbeResult{Text: string(body)}, &code, newProbeUpstreamError(account, code, body)
	}
	result, err := parser(resp.Body)
	return result, &code, err
}

// parseAnthropicMessagesStream 解析 Messages SSE：收集 text_delta，thinking_delta 只做标记；
// 从 message_start 读输入用量与模型、从 message_delta 读 stop_reason 与输出用量。
func parseAnthropicMessagesStream(body io.Reader) (anthropicProbeResult, error) {
	reader := bufio.NewReader(body)
	var result anthropicProbeResult
	var parts []string
	finish := func() anthropicProbeResult {
		result.Text = strings.Join(parts, "")
		return result
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return finish(), nil
			}
			return finish(), err
		}
		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}
		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			return finish(), nil
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}
		switch data["type"] {
		case "message_start":
			message, _ := data["message"].(map[string]any)
			if message != nil {
				if model, ok := message["model"].(string); ok {
					result.Model = model
				}
				if usage, ok := message["usage"].(map[string]any); ok {
					result.Usage.InputTokens = jsonNumberToInt64(usage["input_tokens"])
					result.Usage.CacheCreationInputTokens = jsonNumberToInt64(usage["cache_creation_input_tokens"])
					result.Usage.CacheReadInputTokens = jsonNumberToInt64(usage["cache_read_input_tokens"])
					if out := jsonNumberToInt64(usage["output_tokens"]); out > 0 {
						result.Usage.OutputTokens = out
					}
				}
			}
		case "content_block_start":
			if block, ok := data["content_block"].(map[string]any); ok {
				switch block["type"] {
				case "thinking", "redacted_thinking":
					result.SawThinking = true
				}
			}
		case "content_block_delta":
			delta, _ := data["delta"].(map[string]any)
			if delta == nil {
				continue
			}
			switch delta["type"] {
			case "thinking_delta", "signature_delta":
				result.SawThinking = true
			default:
				if text, ok := delta["text"].(string); ok && text != "" {
					parts = append(parts, text)
				}
			}
		case "message_delta":
			if delta, ok := data["delta"].(map[string]any); ok {
				if reason, ok := delta["stop_reason"].(string); ok && reason != "" {
					result.StopReason = reason
				}
			}
			if usage, ok := data["usage"].(map[string]any); ok {
				if out := jsonNumberToInt64(usage["output_tokens"]); out > 0 {
					result.Usage.OutputTokens = out
				}
				if in := jsonNumberToInt64(usage["input_tokens"]); in > 0 {
					result.Usage.InputTokens = in
				}
			}
		case "message_stop":
			result.Completed = true
			return finish(), nil
		case "error":
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok && msg != "" {
					return finish(), errors.New(msg)
				}
			}
			return finish(), errors.New("anthropic probe failed")
		}
	}
}
