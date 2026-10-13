package service

import (
	"context"
	"time"
)

// fork：两级审计。内容审计固定为 TypeSafe 初审、OpenAI 复核：TypeSafe 判定违规时先记录，
// 再交给 OpenAI 审计引擎复核，复核也判定违规才算命中（计入违规次数、自动封禁、邮件 /
// 站内信警告、记录违规哈希、同步拦截）。复核放行、复核失败或复核引擎没有可用 Key 时
// 只记一条 action=unconfirmed 的日志：flagged=false，不计次、不警告、不拦截。

// ContentModerationActionUnconfirmed TypeSafe 判定违规但未经复核确认的日志 action。
const ContentModerationActionUnconfirmed = "unconfirmed"

const (
	ContentModerationVerificationConfirmed   = "confirmed"
	ContentModerationVerificationCleared     = "cleared"
	ContentModerationVerificationError       = "error"
	ContentModerationVerificationUnavailable = "unavailable"
)

// 复核所用接口的来源：审计配置里自己的 Key，或借用内容翻译的地址与 Key。
const (
	ContentModerationVerifierSourceKeys        = "keys"
	ContentModerationVerifierSourceTranslation = "translation"
)

// contentModerationChatFallback LLM 复核引擎没配 Key 时借用的对话接口（内容翻译的地址与 Key）。
type contentModerationChatFallback interface {
	ModerationChatEndpoint(ctx context.Context) (baseURL string, apiKey string, err error)
}

// SetVerifierChatFallback 注入复核的备用对话接口；传 nil 关闭。
func (s *ContentModerationService) SetVerifierChatFallback(fallback contentModerationChatFallback) {
	s.verifierFallback = fallback
}

// ContentModerationVerification 复核结果，存在日志 engine_meta.verification 里。
type ContentModerationVerification struct {
	Status          string             `json:"status"`
	Engine          string             `json:"engine"`
	Source          string             `json:"source,omitempty"`
	Model           string             `json:"model,omitempty"`
	HighestCategory string             `json:"highest_category,omitempty"`
	HighestScore    float64            `json:"highest_score,omitempty"`
	CategoryScores  map[string]float64 `json:"category_scores,omitempty"`
	LatencyMS       int                `json:"latency_ms"`
	Error           string             `json:"error,omitempty"`
}

func (v *ContentModerationVerification) confirmed() bool {
	return v != nil && v.Status == ContentModerationVerificationConfirmed
}

// moderationPipeline 生效配置：顶层字段是 TypeSafe 初审，verifier 是 OpenAI 复核。
// 旧配置里的 engine 字段（二选一时代）不再决定谁生效。TypeSafe 没配 Key 时只用 OpenAI 审计，
// 它本来就是确认引擎，判定直接算数；否则只配了 OpenAI 的部署会悄悄失去 API 审计。
func (cfg *ContentModerationConfig) moderationPipeline() *ContentModerationConfig {
	primary := cfg.effectiveEngine(ContentModerationEngineTypeSafe)
	if len(primary.apiKeys()) == 0 {
		return cfg.effectiveEngine(ContentModerationEngineOpenAI)
	}
	primary.verifier = cfg.effectiveEngine(ContentModerationEngineOpenAI)
	return primary
}

// verifierEngine 复核引擎名，只用 OpenAI 单级审计时为空。
func (cfg *ContentModerationConfig) verifierEngine() string {
	if cfg == nil || cfg.verifier == nil {
		return ""
	}
	return ContentModerationEngineOpenAI
}

// verifyFlaggedModeration 只复核 TypeSafe 初审的命中；初审未命中、或本身就是 OpenAI 单级审计时
// 返回 nil，判定保持不变。
func (s *ContentModerationService) verifyFlaggedModeration(ctx context.Context, cfg *ContentModerationConfig, content ContentModerationInput, primaryFlagged bool, trackKeyLoad bool) *ContentModerationVerification {
	if !primaryFlagged || cfg == nil || cfg.Engine != ContentModerationEngineTypeSafe {
		return nil
	}
	return s.verifyModeration(ctx, cfg.verifier, content, trackKeyLoad)
}

// verifyModeration 用 OpenAI 审计引擎复核 TypeSafe 命中的内容（图片一并送审）。
func (s *ContentModerationService) verifyModeration(ctx context.Context, verifier *ContentModerationConfig, content ContentModerationInput, trackKeyLoad bool) *ContentModerationVerification {
	v := &ContentModerationVerification{Engine: ContentModerationEngineOpenAI}
	verifier, v.Source = s.resolveVerifier(ctx, verifier)
	if verifier == nil {
		v.Status = ContentModerationVerificationUnavailable
		return v
	}
	v.Model = verifier.Model
	start := time.Now()
	result, err := s.callModeration(ctx, verifier, content.ModerationInput(), trackKeyLoad)
	v.LatencyMS = int(time.Since(start).Milliseconds())
	if err != nil {
		v.Status = ContentModerationVerificationError
		v.Error = err.Error()
		return v
	}
	if result.EngineMeta != nil && result.EngineMeta.Model != "" {
		v.Model = result.EngineMeta.Model
	}
	flagged, highestCategory, highestScore := evaluateModerationScores(result.CategoryScores, verifier.Thresholds)
	v.HighestCategory, v.HighestScore, v.CategoryScores = highestCategory, highestScore, result.CategoryScores
	v.Status = ContentModerationVerificationCleared
	if flagged {
		v.Status = ContentModerationVerificationConfirmed
	}
	return v
}

// resolveVerifier 复核实际使用的配置：有自己的 Key 就用；没有时借内容翻译的地址与 Key
// （模型、超时、阈值仍取复核配置；借来的是本机或自建网关地址，不走审计代理）。都没有时返回 nil。
func (s *ContentModerationService) resolveVerifier(ctx context.Context, verifier *ContentModerationConfig) (*ContentModerationConfig, string) {
	if verifier == nil {
		return nil, ""
	}
	if len(verifier.apiKeys()) > 0 {
		return verifier, ContentModerationVerifierSourceKeys
	}
	if s.verifierFallback == nil {
		return nil, ""
	}
	baseURL, apiKey, err := s.verifierFallback.ModerationChatEndpoint(ctx)
	if err != nil || baseURL == "" || apiKey == "" {
		return nil, ""
	}
	borrowed := cloneContentModerationConfig(verifier)
	borrowed.BaseURL = baseURL
	borrowed.APIKeys = []string{apiKey}
	borrowed.ProxyID = nil
	return borrowed, ContentModerationVerifierSourceTranslation
}

// verifierSource 运行状态里的复核来源；没有复核引擎或复核不可用时为空。
func (s *ContentModerationService) verifierSource(ctx context.Context, cfg *ContentModerationConfig) string {
	if cfg == nil || cfg.verifier == nil {
		return ""
	}
	_, source := s.resolveVerifier(ctx, cfg.verifier)
	return source
}

// withVerification 把复核结果挂到初审的引擎元数据上（复制一份，不改共享对象）。
func withVerification(meta *ContentModerationEngineMeta, verification *ContentModerationVerification) *ContentModerationEngineMeta {
	if verification == nil {
		return meta
	}
	out := &ContentModerationEngineMeta{}
	if meta != nil {
		*out = *meta
	}
	out.Verification = verification
	return out
}
