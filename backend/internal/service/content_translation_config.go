package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// SettingKeyContentTranslationConfig 内容自动翻译的后台配置（JSON）。
const SettingKeyContentTranslationConfig = "content_translation_config"

const contentTranslationMaxModelLen = 128

var (
	ErrContentTranslationModelRequired = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_MODEL_REQUIRED", "a model name (at most 128 characters) is required to enable content translation")
	ErrContentTranslationAPIKeyRequired = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_API_KEY_REQUIRED", "an admin API key is required to enable content translation")
	ErrContentTranslationAPIKeyInvalid = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_API_KEY_INVALID", "the selected API key does not exist, is not active, or does not belong to an admin")
	ErrContentTranslationBaseURLInvalid = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_BASE_URL_INVALID", "base_url must be an http(s) URL")
	ErrContentTranslationLanguagesInvalid = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_LANGUAGES_INVALID", "languages may only contain zh / en / ja")
)

// ContentTranslationConfig 是后台可编辑的翻译配置。
//
// 只存管理员 API Key 的 id，调用时才从 api_keys 表读出 Key：密钥不会被复制进设置表，
// 管理员停用或删掉这把 Key，翻译随之停止。
type ContentTranslationConfig struct {
	Enabled  bool   `json:"enabled"`
	APIKeyID *int64 `json:"api_key_id"`
	Model    string `json:"model"`
	// BaseURL 留空走本机回环 http://127.0.0.1:<server.port>，只请求 {BaseURL}/v1/chat/completions。
	BaseURL   string   `json:"base_url"`
	Languages []string `json:"languages"`
}

// contentTranslationCredentials 是解析好的一次调用所需信息。
type contentTranslationCredentials struct {
	Endpoint string // 完整的 chat/completions 地址
	APIKey   string
	Model    string
}

func defaultContentTranslationConfig() ContentTranslationConfig {
	return ContentTranslationConfig{Languages: append([]string(nil), ContentTranslationLanguages...)}
}

// normalizeContentTranslationConfig 归一化并校验配置（不查库）。
func normalizeContentTranslationConfig(in *ContentTranslationConfig) error {
	in.Model = strings.TrimSpace(in.Model)
	if len(in.Model) > contentTranslationMaxModelLen {
		return ErrContentTranslationModelRequired
	}
	if in.APIKeyID != nil && *in.APIKeyID <= 0 {
		in.APIKeyID = nil
	}

	base := strings.TrimSpace(in.BaseURL)
	base = strings.TrimRight(base, "/")
	base = strings.TrimSuffix(base, "/v1")
	base = strings.TrimRight(base, "/")
	if base != "" {
		parsed, err := url.Parse(base)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return ErrContentTranslationBaseURLInvalid
		}
	}
	in.BaseURL = base

	if in.Languages == nil {
		in.Languages = []string{}
	}
	seen := map[string]bool{}
	langs := make([]string, 0, len(ContentTranslationLanguages))
	for _, raw := range in.Languages {
		lang := strings.ToLower(strings.TrimSpace(raw))
		if lang != ContentLangZH && lang != ContentLangEN && lang != ContentLangJA {
			return ErrContentTranslationLanguagesInvalid
		}
		seen[lang] = true
	}
	// 按固定顺序输出，配置比较与展示都稳定
	for _, lang := range ContentTranslationLanguages {
		if seen[lang] {
			langs = append(langs, lang)
		}
	}
	in.Languages = langs

	if in.Enabled {
		if in.Model == "" {
			return ErrContentTranslationModelRequired
		}
		if in.APIKeyID == nil {
			return ErrContentTranslationAPIKeyRequired
		}
	}
	return nil
}

func (c ContentTranslationConfig) languageEnabled(lang string) bool {
	for _, l := range c.Languages {
		if l == lang {
			return true
		}
	}
	return false
}

func (s *ContentTranslationService) loadConfigFromStore(ctx context.Context) (ContentTranslationConfig, error) {
	cfg := defaultContentTranslationConfig()
	if s.settingRepo == nil {
		return cfg, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyContentTranslationConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return cfg, nil
		}
		return cfg, err
	}
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	var stored ContentTranslationConfig
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return cfg, fmt.Errorf("parse content translation config: %w", err)
	}
	if stored.Languages == nil {
		stored.Languages = cfg.Languages
	}
	if err := normalizeContentTranslationConfig(&stored); err != nil {
		// 已保存的配置不合法（比如手改了库），按关闭处理，避免拿坏配置去调模型
		stored.Enabled = false
	}
	return stored, nil
}

// resolveCredentials 把配置解析成一次调用所需的地址、Key 与模型。
// Key 必须存在、处于启用状态、且属于 admin 角色用户。
func (s *ContentTranslationService) resolveCredentials(ctx context.Context, cfg ContentTranslationConfig) (*contentTranslationCredentials, error) {
	if cfg.Model == "" {
		return nil, ErrContentTranslationModelRequired
	}
	if cfg.APIKeyID == nil {
		return nil, ErrContentTranslationAPIKeyRequired
	}
	if s.apiKeyRepo == nil {
		return nil, ErrContentTranslationAPIKeyInvalid
	}
	key, err := s.apiKeyRepo.GetByID(ctx, *cfg.APIKeyID)
	if err != nil || key == nil || strings.TrimSpace(key.Key) == "" || key.Status != StatusActive {
		return nil, ErrContentTranslationAPIKeyInvalid
	}
	owner := key.User
	if owner == nil || owner.ID != key.UserID {
		if s.userRepo == nil {
			return nil, ErrContentTranslationAPIKeyInvalid
		}
		owner, err = s.userRepo.GetByID(ctx, key.UserID)
		if err != nil || owner == nil {
			return nil, ErrContentTranslationAPIKeyInvalid
		}
	}
	if owner.Role != RoleAdmin || owner.Status != StatusActive {
		return nil, ErrContentTranslationAPIKeyInvalid
	}

	base := cfg.BaseURL
	if base == "" {
		base = contentTranslationLoopbackBase(s.cfg)
	}
	return &contentTranslationCredentials{
		Endpoint: base + "/v1/chat/completions",
		APIKey:   key.Key,
		Model:    cfg.Model,
	}, nil
}

// apiKeyName 返回所选 Key 的名称（给后台展示用），找不到时返回空串。
func (s *ContentTranslationService) apiKeyName(ctx context.Context, id *int64) string {
	if id == nil || s.apiKeyRepo == nil {
		return ""
	}
	key, err := s.apiKeyRepo.GetByID(ctx, *id)
	if err != nil || key == nil {
		return ""
	}
	return key.Name
}

// contentTranslationLoopbackBase 是本机网关地址：监听 0.0.0.0 / :: / 空时改连 127.0.0.1。
// 本服务不终止 TLS，回环用 http 即可。
func contentTranslationLoopbackBase(cfg *config.Config) string {
	host := "127.0.0.1"
	port := 8080
	if cfg != nil {
		if h := strings.TrimSpace(cfg.Server.Host); h != "" && h != "0.0.0.0" && h != "::" && h != "[::]" {
			host = strings.Trim(h, "[]")
		}
		if cfg.Server.Port > 0 {
			port = cfg.Server.Port
		}
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}
