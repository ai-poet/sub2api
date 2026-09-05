package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	GroupRuntimeStatusUp       = "up"
	GroupRuntimeStatusDegraded = "degraded"
	GroupRuntimeStatusDown     = "down"

	GroupStatusValidationNonEmpty    = "non_empty"
	GroupStatusValidationKeywordsAny = "keywords_any"
	GroupStatusValidationKeywordsAll = "keywords_all"

	GroupStatusEventUp   = "up"
	GroupStatusEventDown = "down"

	GroupStatusPeriod24h = "24h"
	GroupStatusPeriod7d  = "7d"

	groupStatusDefaultIntervalSeconds = 60
	groupStatusDefaultTimeoutSeconds  = 30
	groupStatusDefaultSlowLatencyMS   = int64(15000)
	groupStatusRetentionDays          = 30
)

var (
	ErrGroupStatusConfigNotFound = infraerrors.NotFound("GROUP_STATUS_CONFIG_NOT_FOUND", "group runtime status config not found")
	ErrGroupStatusFeatureClosed  = infraerrors.NotFound("GROUP_STATUS_FEATURE_DISABLED", "group runtime status feature is disabled")
	ErrGroupStatusForbidden      = infraerrors.Forbidden("GROUP_STATUS_FORBIDDEN", "group runtime status is not available for this group")
	ErrGroupStatusInvalidConfig  = infraerrors.BadRequest("GROUP_STATUS_INVALID_CONFIG", "invalid group runtime status config")
)

type GroupStatusConfig struct {
	ID               int64    `json:"id"`
	GroupID          int64    `json:"group_id"`
	Enabled          bool     `json:"enabled"`
	ProbeModel       string   `json:"probe_model"`
	ProbePrompt      string   `json:"probe_prompt"`
	ValidationMode   string   `json:"validation_mode"`
	ExpectedKeywords []string `json:"expected_keywords"`
	IntervalSeconds  int      `json:"interval_seconds"`
	TimeoutSeconds   int      `json:"timeout_seconds"`
	SlowLatencyMS    int64    `json:"slow_latency_ms"`
	NotifyEnabled    bool     `json:"notify_enabled"`
	// 纯 Sol 验证（Juice 指纹探测），仅 OpenAI 分组可开启
	SolJuiceEnabled         bool      `json:"sol_juice_enabled"`
	SolJuiceIntervalSeconds int       `json:"sol_juice_interval_seconds"`
	SolJuiceModel           string    `json:"sol_juice_model"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type GroupStatusRecord struct {
	ID              int64     `json:"id"`
	GroupID         int64     `json:"group_id"`
	ConfigID        int64     `json:"config_id"`
	Status          string    `json:"status"`
	ResponseExcerpt string    `json:"response_excerpt"`
	LatencyMS       *int64    `json:"latency_ms"`
	HTTPCode        *int      `json:"http_code"`
	SubStatus       string    `json:"sub_status"`
	ErrorDetail     string    `json:"error_detail"`
	ObservedAt      time.Time `json:"observed_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type GroupStatusState struct {
	ID                 int64      `json:"id"`
	GroupID            int64      `json:"group_id"`
	ConfigID           int64      `json:"config_id"`
	LatestStatus       string     `json:"latest_status"`
	StableStatus       string     `json:"stable_status"`
	ResponseExcerpt    string     `json:"response_excerpt"`
	LatencyMS          *int64     `json:"latency_ms"`
	HTTPCode           *int       `json:"http_code"`
	SubStatus          string     `json:"sub_status"`
	ErrorDetail        string     `json:"error_detail"`
	ObservedAt         *time.Time `json:"observed_at"`
	ConsecutiveDown    int        `json:"consecutive_down"`
	ConsecutiveNonDown int        `json:"consecutive_non_down"`
	// 纯 Sol 验证的最近结果与稳定结论
	SolJuiceStatus              string     `json:"sol_juice_status"`
	SolJuiceStableStatus        string     `json:"sol_juice_stable_status"`
	SolJuiceValue               string     `json:"sol_juice_value"`
	SolJuiceDetail              string     `json:"sol_juice_detail"`
	SolJuiceCheckedAt           *time.Time `json:"sol_juice_checked_at"`
	SolJuiceConsecutiveMismatch int        `json:"sol_juice_consecutive_mismatch"`
	SolJuiceInputTokens         int64      `json:"sol_juice_input_tokens"`
	SolJuiceOutputTokens        int64      `json:"sol_juice_output_tokens"`
	SolJuiceReasoningTokens     int64      `json:"sol_juice_reasoning_tokens"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

type GroupStatusEvent struct {
	ID          int64     `json:"id"`
	GroupID     int64     `json:"group_id"`
	ConfigID    int64     `json:"config_id"`
	EventType   string    `json:"event_type"`
	FromStatus  string    `json:"from_status"`
	ToStatus    string    `json:"to_status"`
	LatencyMS   *int64    `json:"latency_ms"`
	HTTPCode    *int      `json:"http_code"`
	SubStatus   string    `json:"sub_status"`
	ErrorDetail string    `json:"error_detail"`
	ObservedAt  time.Time `json:"observed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type GroupStatusSummary struct {
	GroupID            int64      `json:"group_id"`
	ConfigID           int64      `json:"config_id"`
	Enabled            bool       `json:"enabled"`
	ProbeModel         string     `json:"probe_model"`
	LatestStatus       string     `json:"latest_status"`
	StableStatus       string     `json:"stable_status"`
	ResponseExcerpt    string     `json:"response_excerpt"`
	LatencyMS          *int64     `json:"latency_ms"`
	HTTPCode           *int       `json:"http_code"`
	SubStatus          string     `json:"sub_status"`
	ErrorDetail        string     `json:"error_detail"`
	ObservedAt         *time.Time `json:"observed_at"`
	ConsecutiveDown    int        `json:"consecutive_down"`
	ConsecutiveNonDown int        `json:"consecutive_non_down"`
	// 纯 Sol 验证：配置 + 最近结果；LastCostUSD 为派生字段，不落库
	SolJuiceEnabled             bool       `json:"sol_juice_enabled"`
	SolJuiceModel               string     `json:"sol_juice_model"`
	SolJuiceIntervalSeconds     int        `json:"sol_juice_interval_seconds"`
	SolJuiceStatus              string     `json:"sol_juice_status"`
	SolJuiceStableStatus        string     `json:"sol_juice_stable_status"`
	SolJuiceValue               string     `json:"sol_juice_value"`
	SolJuiceDetail              string     `json:"sol_juice_detail"`
	SolJuiceCheckedAt           *time.Time `json:"sol_juice_checked_at"`
	SolJuiceConsecutiveMismatch int        `json:"sol_juice_consecutive_mismatch"`
	SolJuiceInputTokens         int64      `json:"sol_juice_input_tokens"`
	SolJuiceOutputTokens        int64      `json:"sol_juice_output_tokens"`
	SolJuiceReasoningTokens     int64      `json:"sol_juice_reasoning_tokens"`
	SolJuiceLastCostUSD         float64    `json:"sol_juice_last_cost_usd"`
}

type GroupStatusHistoryBucket struct {
	BucketStart  time.Time `json:"bucket_start"`
	BucketEnd    time.Time `json:"bucket_end"`
	Availability float64   `json:"availability"`
	AvgLatencyMS *float64  `json:"avg_latency_ms"`
	TotalCount   int       `json:"total_count"`
	DownCount    int       `json:"down_count"`
	LatestStatus string    `json:"latest_status"`
}

type GroupStatusListItem struct {
	Group          Group              `json:"group"`
	Summary        GroupStatusSummary `json:"summary"`
	Availability24 float64            `json:"availability_24h"`
	Availability7d float64            `json:"availability_7d"`
}

type GroupStatusRepository interface {
	GetConfig(ctx context.Context, groupID int64) (*GroupStatusConfig, error)
	UpsertConfig(ctx context.Context, config *GroupStatusConfig) (*GroupStatusConfig, error)
	ListDueConfigs(ctx context.Context, now time.Time, limit int) ([]*GroupStatusConfig, error)
	GetState(ctx context.Context, groupID int64) (*GroupStatusState, error)
	ListSummaries(ctx context.Context, groupIDs []int64) ([]GroupStatusSummary, error)
	ListAllSummaries(ctx context.Context) ([]GroupStatusSummary, error)
	SaveProbeResult(ctx context.Context, result *GroupStatusProbeResult) (*GroupStatusState, *GroupStatusEvent, error)
	ListRecordsSince(ctx context.Context, groupID int64, since time.Time) ([]GroupStatusRecord, error)
	ListRecentRecords(ctx context.Context, groupID int64, limit int) ([]GroupStatusRecord, error)
	ListEvents(ctx context.Context, groupID int64, limit int) ([]GroupStatusEvent, error)
	CalculateAvailability(ctx context.Context, groupIDs []int64, since time.Time) (map[int64]float64, error)
	DeleteRecordsOlderThan(ctx context.Context, before time.Time) (int64, error)
	// 纯 Sol 验证（Juice 指纹探测）
	ListDueSolJuiceConfigs(ctx context.Context, now time.Time, limit int) ([]*GroupStatusConfig, error)
	SaveSolJuiceResult(ctx context.Context, result *GroupStatusSolJuiceResult) (*GroupStatusState, *GroupStatusEvent, error)
	ListRecentSolJuiceRecords(ctx context.Context, groupID int64, limit int) ([]GroupStatusJuiceRecord, error)
	DeleteSolJuiceRecordsOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type GroupStatusProbeResult struct {
	GroupID         int64     `json:"group_id"`
	ConfigID        int64     `json:"config_id"`
	Status          string    `json:"status"`
	ResponseExcerpt string    `json:"response_excerpt"`
	LatencyMS       *int64    `json:"latency_ms"`
	HTTPCode        *int      `json:"http_code"`
	SubStatus       string    `json:"sub_status"`
	ErrorDetail     string    `json:"error_detail"`
	ObservedAt      time.Time `json:"observed_at"`
}

type GroupStatusProbeExecution struct {
	Group   *Group                  `json:"group,omitempty"`
	Config  *GroupStatusConfig      `json:"config,omitempty"`
	Account *Account                `json:"account,omitempty"`
	Result  *GroupStatusProbeResult `json:"result,omitempty"`
	State   *GroupStatusState       `json:"state,omitempty"`
	Event   *GroupStatusEvent       `json:"event,omitempty"`
}

// GroupStatusSolJuiceResult 是一次 Juice 探测的样本。
type GroupStatusSolJuiceResult struct {
	GroupID         int64     `json:"group_id"`
	ConfigID        int64     `json:"config_id"`
	Model           string    `json:"model"`
	Effort          string    `json:"effort"`
	Classification  string    `json:"classification"`
	NormalizedValue string    `json:"normalized_value"`
	AnswerExcerpt   string    `json:"answer_excerpt"`
	HTTPCode        *int      `json:"http_code"`
	LatencyMS       *int64    `json:"latency_ms"`
	InputTokens     int64     `json:"input_tokens"`
	OutputTokens    int64     `json:"output_tokens"`
	ReasoningTokens int64     `json:"reasoning_tokens"`
	ErrorDetail     string    `json:"error_detail"`
	ObservedAt      time.Time `json:"observed_at"`
}

// GroupStatusJuiceRecord 是落库后的 Juice 样本。
type GroupStatusJuiceRecord struct {
	ID              int64     `json:"id"`
	GroupID         int64     `json:"group_id"`
	ConfigID        int64     `json:"config_id"`
	Model           string    `json:"model"`
	Effort          string    `json:"effort"`
	Classification  string    `json:"classification"`
	NormalizedValue string    `json:"normalized_value"`
	AnswerExcerpt   string    `json:"answer_excerpt"`
	HTTPCode        *int      `json:"http_code"`
	LatencyMS       *int64    `json:"latency_ms"`
	InputTokens     int64     `json:"input_tokens"`
	OutputTokens    int64     `json:"output_tokens"`
	ReasoningTokens int64     `json:"reasoning_tokens"`
	ErrorDetail     string    `json:"error_detail"`
	ObservedAt      time.Time `json:"observed_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type GroupStatusSolJuiceExecution struct {
	Group   *Group                     `json:"group,omitempty"`
	Config  *GroupStatusConfig         `json:"config,omitempty"`
	Account *Account                   `json:"account,omitempty"`
	Result  *GroupStatusSolJuiceResult `json:"result,omitempty"`
	State   *GroupStatusState          `json:"state,omitempty"`
	Event   *GroupStatusEvent          `json:"event,omitempty"`
	// Confirmed 表示本次是首次 mismatch 后的立即复测
	Confirmed bool `json:"confirmed"`
}

type GroupStatusConfigUpsertInput struct {
	Enabled          bool
	ProbeModel       string
	ProbePrompt      string
	ValidationMode   string
	ExpectedKeywords []string
	IntervalSeconds  int
	TimeoutSeconds   int
	SlowLatencyMS    int64
	// NotifyEnabled 为 nil 表示请求未携带，保留已保存的值
	NotifyEnabled *bool
	// SolJuiceEnabled 为 nil 表示请求未携带，保留已保存的三项 Juice 配置
	SolJuiceEnabled         *bool
	SolJuiceIntervalSeconds int
	SolJuiceModel           string
}

type AvailableGroupReader interface {
	GetAvailableGroups(ctx context.Context, userID int64) ([]Group, error)
}

func DefaultGroupStatusConfig(group *Group) *GroupStatusConfig {
	model := defaultProbeModelByPlatform("")
	if group != nil {
		model = defaultProbeModelByPlatform(group.Platform)
	}
	return &GroupStatusConfig{
		GroupID: func() int64 {
			if group == nil {
				return 0
			}
			return group.ID
		}(),
		Enabled:                 false,
		ProbeModel:              model,
		ProbePrompt:             "Please reply with the single word ONLINE.",
		ValidationMode:          GroupStatusValidationNonEmpty,
		ExpectedKeywords:        []string{},
		IntervalSeconds:         groupStatusDefaultIntervalSeconds,
		TimeoutSeconds:          groupStatusDefaultTimeoutSeconds,
		SlowLatencyMS:           groupStatusDefaultSlowLatencyMS,
		NotifyEnabled:           true,
		SolJuiceEnabled:         false,
		SolJuiceIntervalSeconds: groupStatusSolJuiceDefaultIntervalSeconds,
		SolJuiceModel:           groupStatusSolJuiceDefaultModel,
	}
}

func defaultProbeModelByPlatform(platform string) string {
	switch platform {
	case PlatformOpenAI:
		return openaiDefaultProbeModel()
	case PlatformGemini:
		return geminiDefaultProbeModel()
	case PlatformAntigravity:
		return "claude-sonnet-4-5"
	case PlatformGrok:
		return grokDefaultProbeModel()
	default:
		return claudeDefaultProbeModel()
	}
}

func claudeDefaultProbeModel() string {
	return "claude-sonnet-4-5"
}

func openaiDefaultProbeModel() string {
	return "gpt-4.1-mini"
}

func geminiDefaultProbeModel() string {
	return "gemini-2.5-flash"
}

// grokDefaultProbeModel 跟随网关转发 Grok Responses 时的默认模型，避免分组默认落到
// claude-sonnet-4-5 —— 没有任何 Grok 账号支持它，探测会恒定失败。
func grokDefaultProbeModel() string {
	return grokDefaultResponsesModel
}

func NormalizeGroupStatusConfig(group *Group, input *GroupStatusConfigUpsertInput) (*GroupStatusConfig, error) {
	cfg := DefaultGroupStatusConfig(group)
	if input == nil {
		return cfg, nil
	}
	cfg.Enabled = input.Enabled
	cfg.ProbeModel = strings.TrimSpace(input.ProbeModel)
	cfg.ProbePrompt = strings.TrimSpace(input.ProbePrompt)
	cfg.ValidationMode = strings.TrimSpace(input.ValidationMode)
	cfg.ExpectedKeywords = normalizeKeywords(input.ExpectedKeywords)
	cfg.IntervalSeconds = input.IntervalSeconds
	cfg.TimeoutSeconds = input.TimeoutSeconds
	cfg.SlowLatencyMS = input.SlowLatencyMS
	if input.NotifyEnabled != nil {
		cfg.NotifyEnabled = *input.NotifyEnabled
	}
	if input.SolJuiceEnabled != nil {
		cfg.SolJuiceEnabled = *input.SolJuiceEnabled
	}
	if input.SolJuiceIntervalSeconds > 0 {
		cfg.SolJuiceIntervalSeconds = input.SolJuiceIntervalSeconds
	}
	if model := strings.TrimSpace(input.SolJuiceModel); model != "" {
		cfg.SolJuiceModel = model
	}
	if cfg.SolJuiceEnabled && (group == nil || group.Platform != PlatformOpenAI) {
		return nil, fmt.Errorf("%w: sol_juice is only available for openai groups", ErrGroupStatusInvalidConfig)
	}
	if err := ValidateGroupStatusConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ValidateGroupStatusConfig(cfg *GroupStatusConfig) error {
	if cfg == nil {
		return ErrGroupStatusInvalidConfig
	}
	if cfg.GroupID <= 0 {
		return fmt.Errorf("%w: missing group_id", ErrGroupStatusInvalidConfig)
	}
	if strings.TrimSpace(cfg.ProbeModel) == "" {
		return fmt.Errorf("%w: probe_model is required", ErrGroupStatusInvalidConfig)
	}
	if strings.TrimSpace(cfg.ProbePrompt) == "" {
		return fmt.Errorf("%w: probe_prompt is required", ErrGroupStatusInvalidConfig)
	}
	switch cfg.ValidationMode {
	case GroupStatusValidationNonEmpty, GroupStatusValidationKeywordsAny, GroupStatusValidationKeywordsAll:
	default:
		return fmt.Errorf("%w: unsupported validation_mode", ErrGroupStatusInvalidConfig)
	}
	if (cfg.ValidationMode == GroupStatusValidationKeywordsAny || cfg.ValidationMode == GroupStatusValidationKeywordsAll) && len(cfg.ExpectedKeywords) == 0 {
		return fmt.Errorf("%w: expected_keywords is required for keyword validation", ErrGroupStatusInvalidConfig)
	}
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = groupStatusDefaultIntervalSeconds
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = groupStatusDefaultTimeoutSeconds
	}
	if cfg.SlowLatencyMS <= 0 {
		cfg.SlowLatencyMS = groupStatusDefaultSlowLatencyMS
	}
	cfg.SolJuiceModel = strings.TrimSpace(cfg.SolJuiceModel)
	if cfg.SolJuiceModel == "" {
		cfg.SolJuiceModel = groupStatusSolJuiceDefaultModel
	}
	if cfg.SolJuiceIntervalSeconds <= 0 {
		cfg.SolJuiceIntervalSeconds = groupStatusSolJuiceDefaultIntervalSeconds
	}
	if cfg.SolJuiceIntervalSeconds < groupStatusSolJuiceMinIntervalSeconds {
		return fmt.Errorf("%w: sol_juice_interval_seconds must be >= %d", ErrGroupStatusInvalidConfig, groupStatusSolJuiceMinIntervalSeconds)
	}
	return nil
}

func normalizeKeywords(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func EvaluateGroupStatusValidation(mode string, keywords []string, responseText string) bool {
	text := strings.TrimSpace(responseText)
	switch mode {
	case GroupStatusValidationKeywordsAny:
		lower := strings.ToLower(text)
		for _, keyword := range normalizeKeywords(keywords) {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return true
			}
		}
		return false
	case GroupStatusValidationKeywordsAll:
		lower := strings.ToLower(text)
		for _, keyword := range normalizeKeywords(keywords) {
			if !strings.Contains(lower, strings.ToLower(keyword)) {
				return false
			}
		}
		return len(normalizeKeywords(keywords)) > 0
	default:
		return text != ""
	}
}

func ComputeGroupStatusTransition(prev *GroupStatusState, result *GroupStatusProbeResult) (*GroupStatusState, *GroupStatusEvent) {
	next := &GroupStatusState{}
	if prev != nil {
		*next = *prev
	}

	next.GroupID = result.GroupID
	next.ConfigID = result.ConfigID
	next.LatestStatus = result.Status
	next.ResponseExcerpt = result.ResponseExcerpt
	next.LatencyMS = result.LatencyMS
	next.HTTPCode = result.HTTPCode
	next.SubStatus = result.SubStatus
	next.ErrorDetail = result.ErrorDetail
	next.ObservedAt = &result.ObservedAt

	if result.Status == GroupRuntimeStatusDown {
		next.ConsecutiveDown++
		next.ConsecutiveNonDown = 0
	} else {
		next.ConsecutiveDown = 0
		next.ConsecutiveNonDown++
	}

	prevStable := strings.TrimSpace(next.StableStatus)
	if prevStable == "" && result.Status != GroupRuntimeStatusDown {
		next.StableStatus = result.Status
		return next, nil
	}

	if prevStable == "" && result.Status == GroupRuntimeStatusDown {
		if next.ConsecutiveDown >= 2 {
			next.StableStatus = GroupRuntimeStatusDown
			return next, &GroupStatusEvent{
				GroupID:     result.GroupID,
				ConfigID:    result.ConfigID,
				EventType:   GroupStatusEventDown,
				FromStatus:  "",
				ToStatus:    GroupRuntimeStatusDown,
				LatencyMS:   result.LatencyMS,
				HTTPCode:    result.HTTPCode,
				SubStatus:   result.SubStatus,
				ErrorDetail: result.ErrorDetail,
				ObservedAt:  result.ObservedAt,
			}
		}
		return next, nil
	}

	if prevStable != GroupRuntimeStatusDown {
		if result.Status != GroupRuntimeStatusDown {
			next.StableStatus = result.Status
			return next, nil
		}
		if next.ConsecutiveDown >= 2 {
			next.StableStatus = GroupRuntimeStatusDown
			return next, &GroupStatusEvent{
				GroupID:     result.GroupID,
				ConfigID:    result.ConfigID,
				EventType:   GroupStatusEventDown,
				FromStatus:  prevStable,
				ToStatus:    GroupRuntimeStatusDown,
				LatencyMS:   result.LatencyMS,
				HTTPCode:    result.HTTPCode,
				SubStatus:   result.SubStatus,
				ErrorDetail: result.ErrorDetail,
				ObservedAt:  result.ObservedAt,
			}
		}
		return next, nil
	}

	if result.Status != GroupRuntimeStatusDown && next.ConsecutiveNonDown >= 1 {
		next.StableStatus = result.Status
		return next, &GroupStatusEvent{
			GroupID:     result.GroupID,
			ConfigID:    result.ConfigID,
			EventType:   GroupStatusEventUp,
			FromStatus:  GroupRuntimeStatusDown,
			ToStatus:    result.Status,
			LatencyMS:   result.LatencyMS,
			HTTPCode:    result.HTTPCode,
			SubStatus:   result.SubStatus,
			ErrorDetail: result.ErrorDetail,
			ObservedAt:  result.ObservedAt,
		}
	}

	return next, nil
}

func GroupStatusPeriodRange(period string, now time.Time) (time.Time, time.Time, time.Duration, error) {
	switch period {
	case GroupStatusPeriod24h:
		start := now.Add(-24 * time.Hour).Truncate(time.Hour)
		return start, now, time.Hour, nil
	case GroupStatusPeriod7d:
		start := now.AddDate(0, 0, -6)
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		return start, now, 24 * time.Hour, nil
	default:
		return time.Time{}, time.Time{}, 0, infraerrors.BadRequest("GROUP_STATUS_INVALID_PERIOD", "invalid group status history period")
	}
}
