package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 内容自动翻译的请求 / 响应结构（fork 本地功能，接口约定见 docs/CONTENT_TRANSLATION.md）。

// ContentTranslationLookupRequest 公开查询：texts 是调用方要显示的原文。
type ContentTranslationLookupRequest struct {
	Lang  string   `json:"lang"`
	Texts []string `json:"texts"`
}

// ContentTranslationLookupResponse 的 translations 键是请求里原样发来的文本；没有译文的不出现。
type ContentTranslationLookupResponse struct {
	Lang         string            `json:"lang"`
	Translations map[string]string `json:"translations"`
	Pending      bool              `json:"pending"`
}

// ContentTranslationConfig 后台配置。api_key_name 只读，保存时忽略。
type ContentTranslationConfig struct {
	Enabled    bool     `json:"enabled"`
	APIKeyID   *int64   `json:"api_key_id"`
	APIKeyName string   `json:"api_key_name,omitempty"`
	Model      string   `json:"model"`
	BaseURL    string   `json:"base_url"`
	Languages  []string `json:"languages"`
}

// ContentTranslationTestResult 后台「测试」按钮的结果。
type ContentTranslationTestResult struct {
	Translated string `json:"translated"`
	LatencyMS  int64  `json:"latency_ms"`
}

// ContentTranslationStatus 后台运行状态。
type ContentTranslationStatus struct {
	Enabled     bool       `json:"enabled"`
	Running     bool       `json:"running"`
	Sources     int        `json:"sources"`
	Translated  int        `json:"translated"`
	Pending     int        `json:"pending"`
	LastRunAt   *time.Time `json:"last_run_at"`
	LastError   string     `json:"last_error"`
	LastErrorAt *time.Time `json:"last_error_at"`
}

// ContentTranslationItem 后台译文列表的一行。
type ContentTranslationItem struct {
	ID             int64     `json:"id"`
	SourceHash     string    `json:"source_hash"`
	TargetLang     string    `json:"target_lang"`
	SourceText     string    `json:"source_text"`
	TranslatedText string    `json:"translated_text"`
	Model          string    `json:"model"`
	Manual         bool      `json:"manual"`
	InUse          bool      `json:"in_use"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastSeenAt     time.Time `json:"last_seen_at"`
}

// UpdateContentTranslationRequest 管理员改写译文。
type UpdateContentTranslationRequest struct {
	TranslatedText string `json:"translated_text"`
}

// RegisterContentTranslationSourcesRequest 支付服务登记自己的文案。
type RegisterContentTranslationSourcesRequest struct {
	Texts []string `json:"texts"`
}

func ContentTranslationConfigFromService(cfg service.ContentTranslationConfig, apiKeyName string) ContentTranslationConfig {
	languages := cfg.Languages
	if languages == nil {
		languages = []string{}
	}
	return ContentTranslationConfig{
		Enabled:    cfg.Enabled,
		APIKeyID:   cfg.APIKeyID,
		APIKeyName: apiKeyName,
		Model:      cfg.Model,
		BaseURL:    cfg.BaseURL,
		Languages:  languages,
	}
}

func (c ContentTranslationConfig) ToService() service.ContentTranslationConfig {
	return service.ContentTranslationConfig{
		Enabled:   c.Enabled,
		APIKeyID:  c.APIKeyID,
		Model:     c.Model,
		BaseURL:   c.BaseURL,
		Languages: c.Languages,
	}
}

func ContentTranslationStatusFromService(s service.ContentTranslationStatus) ContentTranslationStatus {
	return ContentTranslationStatus{
		Enabled:     s.Enabled,
		Running:     s.Running,
		Sources:     s.Sources,
		Translated:  s.Translated,
		Pending:     s.Pending,
		LastRunAt:   s.LastRunAt,
		LastError:   s.LastError,
		LastErrorAt: s.LastErrorAt,
	}
}

func ContentTranslationItemFromService(item *service.ContentTranslationItem) ContentTranslationItem {
	if item == nil || item.ContentTranslation == nil {
		return ContentTranslationItem{}
	}
	return ContentTranslationItem{
		ID:             item.ID,
		SourceHash:     item.SourceHash,
		TargetLang:     item.TargetLang,
		SourceText:     item.SourceText,
		TranslatedText: item.TranslatedText,
		Model:          item.Model,
		Manual:         item.Manual,
		InUse:          item.InUse,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
		LastSeenAt:     item.LastSeenAt,
	}
}
