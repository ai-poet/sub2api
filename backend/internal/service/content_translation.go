package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 内容自动翻译（fork 本地功能，见 CLAUDE.md「内容自动翻译」与 docs/CONTENT_TRANSLATION.md）。
//
// 管理员手写的文案用管理员自己的网关 API Key 翻译后永久缓存在 content_translations 表里，键是
// 「规范化原文的 sha256 + 目标语言」：原文不变就永远命中，不再请求模型。各端只在显示层拿原文换译文，
// 接口返回的文案一律不改（桌面客户端按原始分组名里的「国模」等标记分流）。

const (
	ContentLangZH = "zh"
	ContentLangEN = "en"
	ContentLangJA = "ja"

	// contentScriptOther 表示文本里只有中日英以外的文字（如韩文、俄文），三种目标语言都需要翻译。
	contentScriptOther = "other"

	// ContentTranslationMaxTextRunes 单条原文的长度上限；更长的文本既不收集也不查询。
	ContentTranslationMaxTextRunes = 20000
	// ContentTranslationMaxLookupTexts 一次查询最多带多少条文本。
	ContentTranslationMaxLookupTexts = 200
	// ContentTranslationMaxExternalSources 外部（支付服务）一次最多登记多少条文案。
	ContentTranslationMaxExternalSources = 1000
	// ContentTranslationMaxManualRunes 管理员改写译文的长度上限。
	ContentTranslationMaxManualRunes = 100000

	// ContentTranslationNamespacePay 支付服务登记的文案源。
	ContentTranslationNamespacePay = "pay"
)

// ContentTranslationLanguages 支持的目标语言，顺序即后台展示顺序。
var ContentTranslationLanguages = []string{ContentLangZH, ContentLangEN, ContentLangJA}

var (
	ErrContentTranslationLangInvalid = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_LANG_INVALID", "lang must be one of zh / en / ja")
	ErrContentTranslationTooManyTexts = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_TOO_MANY_TEXTS", "too many texts in one request")
	ErrContentTranslationNotFound = infraerrors.NotFound(
		"CONTENT_TRANSLATION_NOT_FOUND", "translation not found")
	ErrContentTranslationTextInvalid = infraerrors.BadRequest(
		"CONTENT_TRANSLATION_TEXT_INVALID", "translated text must be non-empty and not too long")
)

// ContentTranslation 一条缓存的译文。
type ContentTranslation struct {
	ID             int64
	SourceHash     string
	TargetLang     string
	SourceText     string
	TranslatedText string
	Model          string
	Manual         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastSeenAt     time.Time
}

// ContentTranslationFilter 后台译文列表的筛选条件。
type ContentTranslationFilter struct {
	Lang     string
	Query    string
	Page     int
	PageSize int
}

// ContentTranslationSourceText 一条外部登记的文案源。
type ContentTranslationSourceText struct {
	Namespace  string
	SourceHash string
	SourceText string
}

// ContentTranslationRepository 译文缓存与外部文案源的持久化（原生 SQL）。
type ContentTranslationRepository interface {
	// ListAll 读出全部译文，用于启动时装载内存镜像。
	ListAll(ctx context.Context) ([]*ContentTranslation, error)
	// Upsert 写入一条机器译文；已存在的人工译文不会被覆盖。返回库里最终的那一行。
	Upsert(ctx context.Context, entry *ContentTranslation) (*ContentTranslation, error)
	GetByID(ctx context.Context, id int64) (*ContentTranslation, error)
	// UpdateManual 改写译文并标为人工。
	UpdateManual(ctx context.Context, id int64, translated string) (*ContentTranslation, error)
	// Delete 删除一条译文，返回被删的行。
	Delete(ctx context.Context, id int64) (*ContentTranslation, error)
	// DeleteMachine 删除全部机器译文（人工译文保留），返回删除条数。
	DeleteMachine(ctx context.Context) (int64, error)
	// TouchSeen 把这些原文的译文标为「当前在用」。
	TouchSeen(ctx context.Context, hashes []string, at time.Time) error
	List(ctx context.Context, filter ContentTranslationFilter) ([]*ContentTranslation, int64, error)
	// ReplaceNamespaceSources 用 sources（hash → 原文）整体替换某个 namespace 的登记。
	ReplaceNamespaceSources(ctx context.Context, namespace string, sources map[string]string) error
	ListNamespaceSources(ctx context.Context) ([]ContentTranslationSourceText, error)
}

// NormalizeContentLang 把各端传来的语言写法归一成 zh / en / ja；无法识别时返回空串。
func NormalizeContentLang(raw string) string {
	lang := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.IndexAny(lang, ",;"); i >= 0 {
		lang = strings.TrimSpace(lang[:i])
	}
	switch {
	case lang == "":
		return ""
	case strings.HasPrefix(lang, "zh") || lang == "cn":
		return ContentLangZH
	case strings.HasPrefix(lang, "ja") || lang == "jp":
		return ContentLangJA
	case strings.HasPrefix(lang, "en"):
		return ContentLangEN
	default:
		return ""
	}
}

// normalizeContentSource 是计算哈希前的规范化：统一换行、去掉首尾空白。
// 各端发来的原文必须经过同一个函数，才能和收集到的原文对上号。
func normalizeContentSource(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

func contentSourceHash(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// contentSourceUsable 判断规范化后的原文是否可以参与翻译与查询。
func contentSourceUsable(normalized string) bool {
	return normalized != "" && utf8.RuneCountInString(normalized) <= ContentTranslationMaxTextRunes
}

var (
	contentCodeFencePattern   = regexp.MustCompile("(?s)```.*?```")
	contentInlineCodePattern  = regexp.MustCompile("`[^`\n]*`")
	contentURLPattern         = regexp.MustCompile(`(?i)\b(?:https?|ftp)://\S+|\bwww\.\S+|\S+@\S+\.\S+`)
	contentPlaceholderPattern = regexp.MustCompile(`%?\{\{?[^{}\s]*\}?\}|%[sd]`)
	contentHTMLTagPattern     = regexp.MustCompile(`</?[a-zA-Z][^<>]*>`)
)

// detectContentScript 粗略判断一段文本是什么语言：有假名算日文，有汉字算中文，有拉丁字母算英文，
// 只有其它文字算 other，什么字母都没有（纯数字、符号）返回空串。
// 判断前去掉代码、URL、邮箱、占位符和 HTML 标签，避免英文 URL 把一段中文判成英文。
func detectContentScript(text string) string {
	text = contentCodeFencePattern.ReplaceAllString(text, " ")
	text = contentInlineCodePattern.ReplaceAllString(text, " ")
	text = contentURLPattern.ReplaceAllString(text, " ")
	text = contentPlaceholderPattern.ReplaceAllString(text, " ")
	text = contentHTMLTagPattern.ReplaceAllString(text, " ")

	var kana, han, latin, other int
	for _, r := range text {
		switch {
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			kana++
		case unicode.Is(unicode.Han, r):
			han++
		case unicode.Is(unicode.Latin, r):
			latin++
		case unicode.IsLetter(r):
			other++
		}
	}
	switch {
	case kana > 0:
		return ContentLangJA
	case han > 0:
		return ContentLangZH
	case latin > 0:
		return ContentLangEN
	case other > 0:
		return contentScriptOther
	default:
		return ""
	}
}

// contentNeedsTranslation 判断规范化后的原文要不要翻成 lang：原文已经是目标语言、或者根本没有文字时不翻。
func contentNeedsTranslation(normalized, lang string) bool {
	script := detectContentScript(normalized)
	return script != "" && script != lang
}

// contentLanguageName 是写给模型看的语言名。
func contentLanguageName(lang string) string {
	switch lang {
	case ContentLangZH:
		return "Simplified Chinese"
	case ContentLangJA:
		return "Japanese"
	default:
		return "English"
	}
}

func contentTranslationKey(hash, lang string) string {
	return hash + "|" + lang
}
