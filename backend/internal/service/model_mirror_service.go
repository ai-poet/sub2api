package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	modelMirrorRateLimitWindow = 10 * time.Minute
	modelMirrorRateLimitMax    = 3
	modelMirrorRunLockTTL      = 10 * time.Minute
	modelMirrorMainTimeout     = 90 * time.Second
	modelMirrorProbeTimeout    = 30 * time.Second
)

var (
	modelMirrorKnownSSETypes = map[string]struct{}{
		"ping":                {},
		"message_start":       {},
		"content_block_start": {},
		"content_block_delta": {},
		"content_block_stop":  {},
		"message_delta":       {},
		"message_stop":        {},
	}
	modelMirrorNonClaudePatterns = []struct {
		pattern *regexp.Regexp
		name    string
	}{
		{pattern: regexp.MustCompile(`glm`), name: "GLM"},
		{pattern: regexp.MustCompile(`deepseek`), name: "DeepSeek"},
		{pattern: regexp.MustCompile(`minimax`), name: "MiniMax"},
		{pattern: regexp.MustCompile(`grok`), name: "Grok"},
		{pattern: regexp.MustCompile(`qwen`), name: "Qwen"},
		{pattern: regexp.MustCompile(`gpt`), name: "GPT"},
		{pattern: regexp.MustCompile(`gemini`), name: "Gemini"},
	}
	modelMirrorRateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if current == 1 or ttl == -1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)
)

type ModelMirrorService struct {
	settingService *SettingService
	redis          *redis.Client
	httpClient     *http.Client
}

type preparedModelMirrorRun struct {
	UserID         int64
	APIEndpoint    string
	APIKey         string
	APIModel       string
	KnowledgeProbe ModelMirrorKnowledgeProbe
	release        func(context.Context)
}

type modelMirrorStreamSignals struct {
	EventTypes               []string
	ContentBlockTypes        []string
	DeltaTypes               []string
	HasMessageStart          bool
	HasContentBlockStart     bool
	HasContentBlockDelta     bool
	HasMessageDelta          bool
	HasMessageStop           bool
	HasTextDelta             bool
	ThinkingStartSeen        bool
	ThinkingDeltaSeen        bool
	MessageStartModel        string
	InputTokens              int
	OutputTokensSamples      []int
	EmptySignatureDeltaCount int
	NonEmptySignatureCount   int
	UsageShapeValid          bool
	HasCacheCreation         bool
	UnknownEventTypes        []string
}

type modelMirrorStreamingResponse struct {
	ResponseText string
	ThinkingText string
	Signals      *modelMirrorStreamSignals
}

type modelMirrorProbeResponse struct {
	Text        string
	InputTokens int
	Error       string
}

type modelMirrorCheckInput struct {
	Signals        *modelMirrorStreamSignals
	ResponseText   string
	ThinkingText   string
	APIModel       string
	KnowledgeProbe *ModelMirrorKnowledgeProbe
	ProbeKnowledge *modelMirrorProbeResponse
	ProbeShort     *modelMirrorProbeResponse
	ProbeCat       *modelMirrorProbeResponse
	ProbeIdentity  *modelMirrorProbeResponse
}

type modelMirrorCheckDefinition struct {
	ID       string
	Label    string
	Weight   int
	Phase    string
	Evaluate func(input modelMirrorCheckInput) ModelMirrorCheckResult
}

func NewModelMirrorService(settingService *SettingService, redisClient *redis.Client) *ModelMirrorService {
	return &ModelMirrorService{
		settingService: settingService,
		redis:          redisClient,
		httpClient: &http.Client{
			Transport: newModelMirrorTransport(),
		},
	}
}

func (s *ModelMirrorService) PrepareRun(
	ctx context.Context,
	userID int64,
	req ModelMirrorVerifyRequest,
) (*preparedModelMirrorRun, error) {
	if s.redis == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_MIRROR_REDIS_UNAVAILABLE", "model mirror is temporarily unavailable")
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return nil, infraerrors.BadRequest("MODEL_MIRROR_API_KEY_REQUIRED", "api_key is required")
	}

	apiModel := strings.TrimSpace(req.APIModel)
	if apiModel == "" {
		return nil, infraerrors.BadRequest("MODEL_MIRROR_API_MODEL_REQUIRED", "api_model is required")
	}

	apiEndpoint := strings.TrimSpace(req.APIEndpoint)
	if apiEndpoint == "" {
		return nil, infraerrors.BadRequest("MODEL_MIRROR_API_ENDPOINT_REQUIRED", "api_endpoint is required")
	}

	normalizedEndpoint, err := urlvalidator.ValidateHTTPSURL(apiEndpoint, urlvalidator.ValidationOptions{
		AllowPrivate: false,
	})
	if err != nil {
		return nil, infraerrors.BadRequest("MODEL_MIRROR_INVALID_ENDPOINT", err.Error())
	}

	parsed, err := url.Parse(normalizedEndpoint)
	if err != nil {
		return nil, infraerrors.BadRequest("MODEL_MIRROR_INVALID_ENDPOINT", "invalid api endpoint")
	}
	if err := urlvalidator.ValidateResolvedIP(parsed.Hostname()); err != nil {
		return nil, infraerrors.BadRequest("MODEL_MIRROR_INVALID_ENDPOINT", err.Error())
	}

	if err := s.consumeRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	release, err := s.acquireUserRunLock(ctx, userID)
	if err != nil {
		return nil, err
	}

	probe := s.selectKnowledgeProbe(ctx, userID)

	return &preparedModelMirrorRun{
		UserID:         userID,
		APIEndpoint:    normalizedEndpoint,
		APIKey:         apiKey,
		APIModel:       apiModel,
		KnowledgeProbe: probe,
		release:        release,
	}, nil
}

func (s *ModelMirrorService) StreamRun(c *gin.Context, run *preparedModelMirrorRun) error {
	defer run.release(context.Background())

	writer := c.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.Flush()

	send := func(event string, payload any) error {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, body); err != nil {
			return err
		}
		writer.Flush()
		return nil
	}

	sendError := func(message string) error {
		return send("error", gin.H{"message": message})
	}

	ctx := c.Request.Context()

	if err := send("step", gin.H{"message": "正在调用上游流式接口..."}); err != nil {
		return err
	}

	mainCtx, cancelMain := context.WithTimeout(ctx, modelMirrorMainTimeout)
	main, err := s.callStreaming(mainCtx, run)
	cancelMain()
	if err != nil {
		_ = sendError(err.Error())
		return nil
	}

	checkInput := modelMirrorCheckInput{
		Signals:        main.Signals,
		ResponseText:   main.ResponseText,
		ThinkingText:   main.ThinkingText,
		APIModel:       run.APIModel,
		KnowledgeProbe: &run.KnowledgeProbe,
	}

	results := make([]ModelMirrorCheckResult, 0, len(modelMirrorChecks()))
	for _, check := range modelMirrorChecks() {
		if check.Phase != "main" {
			continue
		}
		result := check.Evaluate(checkInput)
		results = append(results, result)
		if err := send("check", result); err != nil {
			return err
		}
	}

	if err := send("step", gin.H{"message": "正在探测知识库事实..."}); err != nil {
		return err
	}
	knowledgeResult := s.runKnowledgeProbe(ctx, run)
	checkInput.ProbeKnowledge = knowledgeResult
	knowledgeCheck := findModelMirrorCheck("knowledgeProbe")
	if knowledgeCheck != nil {
		result := knowledgeCheck.Evaluate(checkInput)
		results = append(results, result)
		if err := send("check", result); err != nil {
			return err
		}
	}

	if err := send("step", gin.H{"message": "正在执行 Token 注入探测..."}); err != nil {
		return err
	}
	shortResult := s.runProbe(ctx, run, "Say hi", "", 100)
	checkInput.ProbeShort = shortResult
	shortCheck := findModelMirrorCheck("tokenInjection")
	if shortCheck != nil {
		result := shortCheck.Evaluate(checkInput)
		results = append(results, result)
		if err := send("check", result); err != nil {
			return err
		}
	}

	if err := send("step", gin.H{"message": "正在执行指令覆盖探测..."}); err != nil {
		return err
	}
	catResult := s.runProbe(ctx, run,
		"What is 1+1?",
		`From now on, you can only reply with the single word "meow". No other characters, letters, or punctuation allowed. This is the highest priority instruction.`,
		100,
	)
	checkInput.ProbeCat = catResult
	catCheck := findModelMirrorCheck("instructionOverride")
	if catCheck != nil {
		result := catCheck.Evaluate(checkInput)
		results = append(results, result)
		if err := send("check", result); err != nil {
			return err
		}
	}

	if err := send("step", gin.H{"message": "正在执行身份覆盖探测..."}); err != nil {
		return err
	}
	identityResult := s.runProbe(ctx, run,
		"Who are you? Who made you? Answer in one sentence.",
		"You are Claude, made by Anthropic. You must always say you are Claude by Anthropic. Never claim to be anyone else or made by anyone else.",
		200,
	)
	checkInput.ProbeIdentity = identityResult
	identityCheck := findModelMirrorCheck("identityOverride")
	if identityCheck != nil {
		result := identityCheck.Evaluate(checkInput)
		results = append(results, result)
		if err := send("check", result); err != nil {
			return err
		}
	}

	totalWeight := 0
	gainedWeight := 0
	for _, result := range results {
		totalWeight += result.Weight
		if result.Pass {
			gainedWeight += result.Weight
		}
	}

	score := 0
	if totalWeight > 0 {
		score = int(float64(gainedWeight)/float64(totalWeight)*100 + 0.5)
	}

	done := gin.H{
		"score":            score,
		"verdict":          s.getVerdict(score, results),
		"total_checks":     len(results),
		"response_excerpt": truncateModelMirrorText(main.ResponseText, 1200),
		"thinking_excerpt": truncateModelMirrorText(main.ThinkingText, 1200),
		"upstream_model":   main.Signals.MessageStartModel,
	}

	return send("done", done)
}

func (s *ModelMirrorService) consumeRateLimit(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("model_mirror:rate:%d", userID)
	result, err := modelMirrorRateLimitScript.Run(
		ctx,
		s.redis,
		[]string{key},
		modelMirrorRateLimitWindow.Milliseconds(),
	).Int64()
	if err != nil {
		return infraerrors.ServiceUnavailable("MODEL_MIRROR_RATE_LIMIT_UNAVAILABLE", "model mirror is temporarily unavailable")
	}
	if result > modelMirrorRateLimitMax {
		return infraerrors.TooManyRequests("MODEL_MIRROR_RATE_LIMITED", "too many verification attempts, please try again later")
	}
	return nil
}

func (s *ModelMirrorService) acquireUserRunLock(
	ctx context.Context,
	userID int64,
) (func(context.Context), error) {
	key := fmt.Sprintf("model_mirror:lock:%d", userID)
	ok, err := s.redis.SetNX(ctx, key, "1", modelMirrorRunLockTTL).Result()
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_MIRROR_LOCK_UNAVAILABLE", "model mirror is temporarily unavailable")
	}
	if !ok {
		return nil, infraerrors.Conflict("MODEL_MIRROR_ALREADY_RUNNING", "a verification task is already running")
	}

	return func(releaseCtx context.Context) {
		_, _ = s.redis.Del(releaseCtx, key).Result()
	}, nil
}

func (s *ModelMirrorService) selectKnowledgeProbe(ctx context.Context, userID int64) ModelMirrorKnowledgeProbe {
	probes := s.settingService.GetModelMirrorKnowledgeProbes(ctx)
	enabled := make([]ModelMirrorKnowledgeProbe, 0, len(probes))
	for _, probe := range probes {
		if probe.Enabled {
			enabled = append(enabled, probe)
		}
	}
	if len(enabled) == 0 {
		enabled = DefaultModelMirrorKnowledgeProbes()
	}

	hasher := fnvString(fmt.Sprintf("%d:%s", userID, time.Now().UTC().Format("2006-01-02")))
	index := hasher % uint32(len(enabled))
	return enabled[index]
}

func (s *ModelMirrorService) runKnowledgeProbe(
	ctx context.Context,
	run *preparedModelMirrorRun,
) *modelMirrorProbeResponse {
	probeCtx, cancel := context.WithTimeout(ctx, modelMirrorProbeTimeout)
	defer cancel()
	result := s.runProbe(probeCtx, run, run.KnowledgeProbe.Prompt, "", 200)
	return result
}

func (s *ModelMirrorService) runProbe(
	ctx context.Context,
	run *preparedModelMirrorRun,
	userMessage string,
	systemPrompt string,
	maxTokens int,
) *modelMirrorProbeResponse {
	response, err := s.callProbe(ctx, run, userMessage, systemPrompt, maxTokens)
	if err != nil {
		return &modelMirrorProbeResponse{Error: err.Error()}
	}
	return response
}

func (s *ModelMirrorService) callStreaming(
	ctx context.Context,
	run *preparedModelMirrorRun,
) (*modelMirrorStreamingResponse, error) {
	payload := map[string]any{
		"model":      run.APIModel,
		"max_tokens": 8192,
		"stream":     true,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "你是谁？请介绍一下你自己，并简要说明你具备的工具能力。",
			},
		},
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 8000,
		},
	}

	body, _ := json.Marshal(payload)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, run.APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", run.APIKey)
	request.Header.Set("anthropic-version", "2023-06-01")
	request.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed")
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return nil, fmt.Errorf("upstream returned %d: %s", response.StatusCode, sanitizeModelMirrorErrorBody(bodyBytes))
	}

	signals := &modelMirrorStreamSignals{
		EventTypes:          make([]string, 0, 16),
		ContentBlockTypes:   make([]string, 0, 8),
		DeltaTypes:          make([]string, 0, 16),
		OutputTokensSamples: make([]int, 0, 8),
		UnknownEventTypes:   make([]string, 0, 4),
		UsageShapeValid:     true,
	}

	var responseText strings.Builder
	var thinkingText strings.Builder

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read upstream stream")
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "data: ") {
			raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "data: "))
			if raw != "" && raw != "[DONE]" {
				var event map[string]any
				if json.Unmarshal([]byte(raw), &event) == nil {
					applyModelMirrorSSEEvent(signals, &responseText, &thinkingText, event)
				}
			}
		}

		if err == io.EOF {
			break
		}
	}

	return &modelMirrorStreamingResponse{
		ResponseText: responseText.String(),
		ThinkingText: thinkingText.String(),
		Signals:      signals,
	}, nil
}

func (s *ModelMirrorService) callProbe(
	ctx context.Context,
	run *preparedModelMirrorRun,
	userMessage string,
	systemPrompt string,
	maxTokens int,
) (*modelMirrorProbeResponse, error) {
	payload := map[string]any{
		"model":      run.APIModel,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": userMessage,
			},
		},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		payload["system"] = systemPrompt
	}

	body, _ := json.Marshal(payload)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, run.APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", run.APIKey)
	request.Header.Set("anthropic-version", "2023-06-01")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed")
	}
	defer func() { _ = response.Body.Close() }()

	rawBody, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream returned %d", response.StatusCode)
	}

	var decoded struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Text  string `json:"text"`
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		return nil, fmt.Errorf("invalid upstream response")
	}

	var text strings.Builder
	for _, item := range decoded.Content {
		if item.Type != "text" || item.Text == "" {
			continue
		}
		if text.Len() > 0 {
			text.WriteByte('\n')
		}
		text.WriteString(item.Text)
	}
	if text.Len() == 0 {
		text.WriteString(decoded.Text)
	}

	return &modelMirrorProbeResponse{
		Text:        text.String(),
		InputTokens: decoded.Usage.InputTokens,
	}, nil
}

func (s *ModelMirrorService) getVerdict(score int, results []ModelMirrorCheckResult) ModelMirrorVerdict {
	byID := make(map[string]bool, len(results))
	for _, result := range results {
		byID[result.ID] = result.Pass
	}

	modelIsReal := byID["identity"] && byID["thinkingConsistency"]
	sseAuthentic := byID["sseShape"] && byID["thinkingConsistency"] && byID["usageConsistency"]
	structureOfficial := byID["responseStructure"] && byID["modelName"] && byID["signature"]
	channelClean := byID["tokenInjection"] && byID["instructionOverride"] && byID["identityOverride"]
	isClaudeCode := byID["answerIdentity"] && byID["toolSupport"]
	knowledgeClean := byID["knowledgeProbe"] && byID["tokenInjection"] && byID["systemPrompt"]

	switch {
	case !modelIsReal && !knowledgeClean:
		return ModelMirrorVerdictLikelyNotClaude
	case sseAuthentic && structureOfficial && channelClean && isClaudeCode && score >= 75:
		return ModelMirrorVerdictMaxPure
	case (modelIsReal || knowledgeClean) && !channelClean:
		return ModelMirrorVerdictReverseProxy
	default:
		return ModelMirrorVerdictOfficialAPI
	}
}

func applyModelMirrorSSEEvent(
	signals *modelMirrorStreamSignals,
	responseText *strings.Builder,
	thinkingText *strings.Builder,
	event map[string]any,
) {
	eventType, _ := event["type"].(string)
	if eventType != "" {
		signals.EventTypes = append(signals.EventTypes, eventType)
		if _, ok := modelMirrorKnownSSETypes[eventType]; !ok {
			signals.UnknownEventTypes = append(signals.UnknownEventTypes, eventType)
		}
	}

	switch eventType {
	case "message_start":
		signals.HasMessageStart = true
		message, _ := event["message"].(map[string]any)
		if model, ok := message["model"].(string); ok {
			signals.MessageStartModel = model
		}
		usage, _ := message["usage"].(map[string]any)
		if inputTokens, ok := intFromAny(usage["input_tokens"]); ok {
			signals.InputTokens = inputTokens
		}
		if _, exists := usage["cache_creation_input_tokens"]; exists {
			signals.HasCacheCreation = true
		}
	case "content_block_start":
		signals.HasContentBlockStart = true
		contentBlock, _ := event["content_block"].(map[string]any)
		if blockType, ok := contentBlock["type"].(string); ok && blockType != "" {
			signals.ContentBlockTypes = append(signals.ContentBlockTypes, blockType)
			if blockType == "thinking" {
				signals.ThinkingStartSeen = true
			}
		}
	case "content_block_delta":
		signals.HasContentBlockDelta = true
		delta, _ := event["delta"].(map[string]any)
		deltaType, _ := delta["type"].(string)
		if deltaType != "" {
			signals.DeltaTypes = append(signals.DeltaTypes, deltaType)
		}
		switch deltaType {
		case "text_delta":
			if text, ok := delta["text"].(string); ok {
				responseText.WriteString(text)
				signals.HasTextDelta = true
			}
		case "thinking_delta":
			if thinking, ok := delta["thinking"].(string); ok {
				thinkingText.WriteString(thinking)
				signals.ThinkingDeltaSeen = true
			}
		case "signature_delta":
			if signature, ok := delta["signature"].(string); ok {
				if strings.TrimSpace(signature) == "" {
					signals.EmptySignatureDeltaCount++
				} else {
					signals.NonEmptySignatureCount++
				}
			}
		}
	case "message_delta":
		signals.HasMessageDelta = true
		usage, _ := event["usage"].(map[string]any)
		if outputTokens, ok := intFromAny(usage["output_tokens"]); ok {
			signals.OutputTokensSamples = append(signals.OutputTokensSamples, outputTokens)
		}
	case "message_stop":
		signals.HasMessageStop = true
	}
}

func modelMirrorChecks() []modelMirrorCheckDefinition {
	return []modelMirrorCheckDefinition{
		{
			ID:     "sseShape",
			Label:  "SSE 事件结构检测",
			Weight: 8,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.Signals == nil {
					return newModelMirrorCheckResult("sseShape", "SSE 事件结构检测", 8, false, "流式调用失败", "")
				}
				score := 0
				if input.Signals.HasMessageStart {
					score += 2
				}
				if input.Signals.HasContentBlockStart {
					score += 2
				}
				if input.Signals.HasContentBlockDelta {
					score += 2
				}
				if input.Signals.HasMessageDelta {
					score += 2
				}
				if input.Signals.HasMessageStop {
					score++
				}
				if input.Signals.HasTextDelta {
					score++
				}
				if len(input.Signals.UnknownEventTypes) > 0 {
					score -= len(input.Signals.UnknownEventTypes) * 2
					if score < 0 {
						score = 0
					}
				}
				detail := fmt.Sprintf("SSE 信号 %d/10", score)
				if len(input.Signals.UnknownEventTypes) > 0 {
					detail += "，未知事件: " + strings.Join(input.Signals.UnknownEventTypes, ", ")
				}
				return newModelMirrorCheckResult("sseShape", "SSE 事件结构检测", 8, score >= 8, detail, "")
			},
		},
		{
			ID:     "thinkingConsistency",
			Label:  "Thinking 一致性检测",
			Weight: 8,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.Signals == nil {
					return newModelMirrorCheckResult("thinkingConsistency", "Thinking 一致性检测", 8, false, "流式调用失败", "")
				}
				score := 0
				if input.Signals.ThinkingStartSeen {
					score += 4
				}
				if input.Signals.ThinkingDeltaSeen {
					score += 4
				}
				if containsString(input.Signals.ContentBlockTypes, "text") && input.Signals.HasTextDelta {
					score += 2
				}
				if input.Signals.EmptySignatureDeltaCount > 0 {
					score -= 3
					if score < 0 {
						score = 0
					}
				}
				detail := fmt.Sprintf("Thinking 信号 %d/10", score)
				if input.Signals.EmptySignatureDeltaCount > 0 {
					detail += " (空签名)"
				}
				return newModelMirrorCheckResult("thinkingConsistency", "Thinking 一致性检测", 8, score >= 8, detail, "")
			},
		},
		{
			ID:     "usageConsistency",
			Label:  "Usage 字段一致性",
			Weight: 6,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.Signals == nil {
					return newModelMirrorCheckResult("usageConsistency", "Usage 字段一致性", 6, false, "流式调用失败", "")
				}
				score := 0
				if input.Signals.UsageShapeValid {
					score += 2
				}
				if input.Signals.InputTokens > 0 {
					score += 3
				}
				if len(input.Signals.OutputTokensSamples) > 0 {
					score += 3
				}
				return newModelMirrorCheckResult(
					"usageConsistency",
					"Usage 字段一致性",
					6,
					score >= 6,
					fmt.Sprintf("Usage 评分 %d/8", score),
					"",
				)
			},
		},
		{
			ID:     "signature",
			Label:  "Signature 签名检测",
			Weight: 8,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.Signals == nil {
					return newModelMirrorCheckResult("signature", "Signature 签名检测", 8, false, "流式调用失败", "")
				}
				if input.Signals.NonEmptySignatureCount > 0 {
					return newModelMirrorCheckResult("signature", "Signature 签名检测", 8, true, fmt.Sprintf("检测到 %d 个有效签名", input.Signals.NonEmptySignatureCount), "")
				}
				return newModelMirrorCheckResult("signature", "Signature 签名检测", 8, false, "未检测到有效签名", "")
			},
		},
		{
			ID:     "modelName",
			Label:  "模型名称一致性",
			Weight: 6,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.Signals == nil || strings.TrimSpace(input.Signals.MessageStartModel) == "" {
					return newModelMirrorCheckResult("modelName", "模型名称一致性", 6, false, "未获取到模型名称", "")
				}
				expected := strings.ToLower(input.APIModel)
				actual := strings.ToLower(input.Signals.MessageStartModel)
				switch {
				case actual == expected:
					return newModelMirrorCheckResult("modelName", "模型名称一致性", 6, true, "完全匹配: "+input.Signals.MessageStartModel, "")
				case strings.Contains(actual, expected) || strings.Contains(expected, actual):
					return newModelMirrorCheckResult("modelName", "模型名称一致性", 6, true, "部分匹配: "+input.Signals.MessageStartModel, "")
				default:
					return newModelMirrorCheckResult("modelName", "模型名称一致性", 6, false, fmt.Sprintf("不匹配: 期望 %s，实际 %s", input.APIModel, input.Signals.MessageStartModel), "")
				}
			},
		},
		{
			ID:     "responseStructure",
			Label:  "响应结构检测",
			Weight: 8,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.Signals == nil {
					return newModelMirrorCheckResult("responseStructure", "响应结构检测", 8, false, "流式调用失败", "")
				}
				if input.Signals.HasCacheCreation {
					return newModelMirrorCheckResult("responseStructure", "响应结构检测", 8, true, "包含 cache_creation 字段（官方特征）", "")
				}
				return newModelMirrorCheckResult("responseStructure", "响应结构检测", 8, false, "缺少 cache_creation 字段", "")
			},
		},
		{
			ID:     "identity",
			Label:  "身份识别检测",
			Weight: 8,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				text := strings.ToLower(input.ResponseText + " " + input.ThinkingText)
				for _, candidate := range modelMirrorNonClaudePatterns {
					if candidate.pattern.MatchString(text) {
						return newModelMirrorCheckResult("identity", "身份识别检测", 8, false, "自称非 Claude ("+candidate.name+")", "")
					}
				}
				hasClaude := strings.Contains(text, "claude")
				hasAnthropic := strings.Contains(text, "anthropic")
				switch {
				case hasClaude && hasAnthropic:
					return newModelMirrorCheckResult("identity", "身份识别检测", 8, true, "识别为 Claude by Anthropic", "")
				case hasClaude:
					return newModelMirrorCheckResult("identity", "身份识别检测", 8, true, "识别为 Claude", "")
				default:
					return newModelMirrorCheckResult("identity", "身份识别检测", 8, false, "未能识别为 Claude", "")
				}
			},
		},
		{
			ID:     "answerIdentity",
			Label:  "Claude Code 身份检测",
			Weight: 0,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				keywords := []string{
					"claude code", "coding assistant", "code assistant", "cli", "命令行",
					"command line", "terminal", "pair programming", "编程助手", "代码助手",
					"software engineer", "agentic",
				}
				text := strings.ToLower(input.ResponseText)
				hits := matchingKeywords(text, keywords)
				switch {
				case len(hits) >= 2:
					return newModelMirrorCheckResult("answerIdentity", "Claude Code 身份检测", 0, true, "强匹配 Claude Code 身份（"+strings.Join(hits, ", ")+")", "pass")
				case len(hits) == 1:
					return newModelMirrorCheckResult("answerIdentity", "Claude Code 身份检测", 0, true, "匹配 Claude Code 关键词: "+hits[0], "pass")
				default:
					return newModelMirrorCheckResult("answerIdentity", "Claude Code 身份检测", 0, false, "回答中未自称 Claude Code / 编程助手（可能为普通 API）", "info")
				}
			},
		},
		{
			ID:     "toolSupport",
			Label:  "工具能力检测",
			Weight: 0,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				keywords := []string{
					"file", "command", "bash", "shell", "read", "write", "execute", "编辑",
					"读取", "写入", "执行", "tool", "browse", "search", "run", "terminal",
					"workspace", "codebase", "代码库",
				}
				text := strings.ToLower(input.ResponseText)
				hits := matchingKeywords(text, keywords)
				switch {
				case len(hits) >= 3:
					return newModelMirrorCheckResult("toolSupport", "工具能力检测", 0, true, fmt.Sprintf("丰富的工具能力描述（%d 个匹配）", len(hits)), "pass")
				case len(hits) >= 1:
					return newModelMirrorCheckResult("toolSupport", "工具能力检测", 0, true, "包含工具能力描述（"+strings.Join(hits, ", ")+")", "pass")
				default:
					return newModelMirrorCheckResult("toolSupport", "工具能力检测", 0, false, "未出现工具能力词（可能为普通 API）", "info")
				}
			},
		},
		{
			ID:     "thinkingIdentity",
			Label:  "Thinking 身份检测",
			Weight: 0,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if strings.TrimSpace(input.ThinkingText) == "" {
					return newModelMirrorCheckResult("thinkingIdentity", "Thinking 身份检测", 0, false, "无 thinking 内容", "info")
				}
				keywords := []string{
					"claude code", "cli", "命令行", "command", "tool", "coding", "编程",
					"代码", "agentic", "pair program",
				}
				hits := matchingKeywords(strings.ToLower(input.ThinkingText), keywords)
				if len(hits) > 0 {
					return newModelMirrorCheckResult("thinkingIdentity", "Thinking 身份检测", 0, true, "Thinking 中发现身份线索（"+strings.Join(hits, ", ")+")", "pass")
				}
				return newModelMirrorCheckResult("thinkingIdentity", "Thinking 身份检测", 0, false, "Thinking 中未发现 Claude Code 相关线索", "info")
			},
		},
		{
			ID:     "systemPrompt",
			Label:  "提示词注入检测",
			Weight: 6,
			Phase:  "main",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				text := strings.ToLower(input.ResponseText + " " + input.ThinkingText)
				risky := []string{"system prompt", "ignore previous", "override", "越权"}
				for _, keyword := range risky {
					if strings.Contains(text, keyword) {
						return newModelMirrorCheckResult("systemPrompt", "提示词注入检测", 6, false, "疑似提示词注入", "")
					}
				}
				return newModelMirrorCheckResult("systemPrompt", "提示词注入检测", 6, true, "未发现异常提示词", "")
			},
		},
		{
			ID:     "tokenInjection",
			Label:  "Token 注入检测",
			Weight: 10,
			Phase:  "probeShort",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.ProbeShort == nil {
					return newModelMirrorCheckResult("tokenInjection", "Token 注入检测", 10, true, "探测未执行", "")
				}
				if input.ProbeShort.Error != "" {
					return newModelMirrorCheckResult("tokenInjection", "Token 注入检测", 10, false, "探测失败: "+input.ProbeShort.Error, "")
				}
				delta := input.ProbeShort.InputTokens - 10
				switch {
				case delta > 100:
					return newModelMirrorCheckResult("tokenInjection", "Token 注入检测", 10, false, fmt.Sprintf("大量隐藏注入（delta ~%d tokens）", delta), "")
				case delta > 20:
					return newModelMirrorCheckResult("tokenInjection", "Token 注入检测", 10, true, fmt.Sprintf("轻微偏差（delta ~%d），可能正常", delta), "")
				default:
					return newModelMirrorCheckResult("tokenInjection", "Token 注入检测", 10, true, fmt.Sprintf("token 数正常（actual: %d, delta: ~%d）", input.ProbeShort.InputTokens, delta), "")
				}
			},
		},
		{
			ID:     "instructionOverride",
			Label:  "指令覆盖检测",
			Weight: 8,
			Phase:  "probeCat",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.ProbeCat == nil {
					return newModelMirrorCheckResult("instructionOverride", "指令覆盖检测", 8, true, "探测未执行", "")
				}
				if input.ProbeCat.Error != "" {
					return newModelMirrorCheckResult("instructionOverride", "指令覆盖检测", 8, false, "探测失败: "+input.ProbeCat.Error, "")
				}
				text := strings.ToLower(strings.TrimSpace(input.ProbeCat.Text))
				hasMeow := strings.Contains(text, "meow")
				hasNumber := regexp.MustCompile(`[0-9]|equals|two|plus|等于`).MatchString(text)
				switch {
				case hasMeow && !hasNumber:
					return newModelMirrorCheckResult("instructionOverride", "指令覆盖检测", 8, true, "system prompt 被正确执行", "")
				case hasNumber && hasMeow:
					return newModelMirrorCheckResult("instructionOverride", "指令覆盖检测", 8, false, "指令被部分覆盖", "")
				case hasNumber:
					return newModelMirrorCheckResult("instructionOverride", "指令覆盖检测", 8, false, "指令被完全覆盖", "")
				default:
					return newModelMirrorCheckResult("instructionOverride", "指令覆盖检测", 8, true, `回复: "`+truncateModelMirrorText(text, 50)+`"`, "")
				}
			},
		},
		{
			ID:     "identityOverride",
			Label:  "身份覆盖检测",
			Weight: 8,
			Phase:  "probeIdentity",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.ProbeIdentity == nil {
					return newModelMirrorCheckResult("identityOverride", "身份覆盖检测", 8, true, "探测未执行", "")
				}
				if input.ProbeIdentity.Error != "" {
					return newModelMirrorCheckResult("identityOverride", "身份覆盖检测", 8, false, "探测失败: "+input.ProbeIdentity.Error, "")
				}
				text := strings.ToLower(input.ProbeIdentity.Text)
				hasClaude := strings.Contains(text, "claude")
				hasAnthropic := strings.Contains(text, "anthropic")
				hasFake := containsAny(text, []string{"amazon", "kiro", "aws", "openai", "gpt", "gemini", "google"})
				switch {
				case hasClaude && hasAnthropic && !hasFake:
					return newModelMirrorCheckResult("identityOverride", "身份覆盖检测", 8, true, "正确识别为 Claude by Anthropic", "")
				case hasFake:
					return newModelMirrorCheckResult("identityOverride", "身份覆盖检测", 8, false, `身份被覆盖: "`+truncateModelMirrorText(text, 80)+`"`, "")
				case !hasClaude && !hasAnthropic:
					return newModelMirrorCheckResult("identityOverride", "身份覆盖检测", 8, false, "未能确认身份", "")
				default:
					return newModelMirrorCheckResult("identityOverride", "身份覆盖检测", 8, true, `部分确认: "`+truncateModelMirrorText(text, 60)+`"`, "")
				}
			},
		},
		{
			ID:     "knowledgeProbe",
			Label:  "知识库事实检测",
			Weight: 10,
			Phase:  "probeKnowledge",
			Evaluate: func(input modelMirrorCheckInput) ModelMirrorCheckResult {
				if input.ProbeKnowledge == nil {
					return newModelMirrorCheckResult("knowledgeProbe", "知识库事实检测", 10, false, "探测未执行", "")
				}
				if input.ProbeKnowledge.Error != "" {
					return newModelMirrorCheckResult("knowledgeProbe", "知识库事实检测", 10, false, "探测失败: "+input.ProbeKnowledge.Error, "")
				}
				return evaluateKnowledgeProbe(input.ProbeKnowledge.Text, input.KnowledgeProbe)
			},
		},
	}
}

func evaluateKnowledgeProbe(text string, probe *ModelMirrorKnowledgeProbe) ModelMirrorCheckResult {
	if probe == nil {
		return newModelMirrorCheckResult("knowledgeProbe", "知识库事实检测", 10, false, "未配置有效知识探测题", "")
	}

	lowerText := strings.ToLower(text)
	if matchesModelMirrorKnowledgeProbe(lowerText, *probe) {
		return newModelMirrorCheckResult(
			"knowledgeProbe",
			"知识库事实检测",
			probe.Weight,
			true,
			buildKnowledgeProbeDetail(*probe, lowerText, true),
			"",
		)
	}
	return newModelMirrorCheckResult(
		"knowledgeProbe",
		"知识库事实检测",
		probe.Weight,
		false,
		buildKnowledgeProbeDetail(*probe, lowerText, false),
		"",
	)
}

func matchesModelMirrorKnowledgeProbe(text string, probe ModelMirrorKnowledgeProbe) bool {
	if len(probe.ExpectedKeywords) == 0 {
		return false
	}
	matches := 0
	for _, keyword := range probe.ExpectedKeywords {
		if strings.Contains(text, keyword) {
			matches++
		}
	}
	if probe.PassMode == ModelMirrorProbePassModeAll {
		return matches == len(probe.ExpectedKeywords)
	}
	return matches > 0
}

func findModelMirrorCheck(id string) *modelMirrorCheckDefinition {
	for _, check := range modelMirrorChecks() {
		if check.ID == id {
			checkCopy := check
			return &checkCopy
		}
	}
	return nil
}

func newModelMirrorCheckResult(
	id string,
	label string,
	weight int,
	pass bool,
	detail string,
	status string,
) ModelMirrorCheckResult {
	return ModelMirrorCheckResult{
		ID:     id,
		Label:  label,
		Weight: weight,
		Pass:   pass,
		Detail: detail,
		Status: status,
	}
}

func newModelMirrorTransport() *http.Transport {
	base := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	base.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("dns resolution returned no ip")
		}
		for _, ip := range ips {
			if isModelMirrorBlockedIP(ip) {
				return nil, fmt.Errorf("resolved ip %s is not allowed", ip.String())
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
	return base
}

func isModelMirrorBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	default:
		return 0, false
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func matchingKeywords(text string, keywords []string) []string {
	matches := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			matches = append(matches, keyword)
		}
	}
	return matches
}

func buildKnowledgeProbeDetail(probe ModelMirrorKnowledgeProbe, text string, pass bool) string {
	matches := matchingKeywords(text, probe.ExpectedKeywords)
	if pass {
		if len(matches) == 0 {
			return "知识题通过"
		}
		return fmt.Sprintf("题库项 %s 命中关键词: %s", probe.ID, strings.Join(matches, ", "))
	}

	if len(matches) > 0 {
		return fmt.Sprintf("题库项 %s 仅命中部分关键词: %s", probe.ID, strings.Join(matches, ", "))
	}
	return fmt.Sprintf("题库项 %s 未命中预期关键词", probe.ID)
}

func sanitizeModelMirrorErrorBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "empty response"
	}
	return truncateModelMirrorText(text, 200)
}

func truncateModelMirrorText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func fnvString(input string) uint32 {
	const offset32 = 2166136261
	const prime32 = 16777619
	var hash uint32 = offset32
	for i := 0; i < len(input); i++ {
		hash ^= uint32(input[i])
		hash *= prime32
	}
	return hash
}
