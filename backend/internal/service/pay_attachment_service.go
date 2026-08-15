package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 发票附件的对象前缀由 InvoiceStorageSettingService 提供（后台可配置，默认
// DefaultInvoiceStoragePrefix）。
//
// 它同时是 presign/delete 的授权边界——只有该前缀下的 key 才允许签名或删除。
// 没有这条边界，持有内部桥接令牌就能为同一个桶里的数据库备份签发下载链接；
// 因此设置项那侧还会拒绝与备份前缀重叠的取值。

// MaxPayAttachmentBytes 单个附件上限（10 MiB）。发票 PDF/OFD 通常在 1 MiB 内。
const MaxPayAttachmentBytes = 10 << 20

const (
	minPayAttachmentPresignTTL     = time.Minute
	maxPayAttachmentPresignTTL     = 15 * time.Minute
	defaultPayAttachmentPresignTTL = 5 * time.Minute
)

var (
	ErrPayAttachmentStorageNotConfigured = infraerrors.BadRequest(
		"PAY_ATTACHMENT_STORAGE_NOT_CONFIGURED",
		"object storage is not configured; configure the backup S3 settings first",
	)
	ErrPayAttachmentInvalidScope = infraerrors.BadRequest(
		"PAY_ATTACHMENT_INVALID_SCOPE", "unsupported attachment scope")
	ErrPayAttachmentInvalidRef = infraerrors.BadRequest(
		"PAY_ATTACHMENT_INVALID_REF", "invalid attachment reference id")
	ErrPayAttachmentInvalidType = infraerrors.BadRequest(
		"PAY_ATTACHMENT_INVALID_TYPE", "unsupported attachment content type")
	ErrPayAttachmentContentMismatch = infraerrors.BadRequest(
		"PAY_ATTACHMENT_CONTENT_MISMATCH", "attachment content does not match its declared type")
	ErrPayAttachmentEmpty = infraerrors.BadRequest(
		"PAY_ATTACHMENT_EMPTY", "attachment is empty")
	ErrPayAttachmentTooLarge = infraerrors.BadRequest(
		"PAY_ATTACHMENT_TOO_LARGE", "attachment exceeds the maximum allowed size")
	ErrPayAttachmentInvalidKey = infraerrors.BadRequest(
		"PAY_ATTACHMENT_INVALID_KEY", "attachment key is outside the allowed prefix")
)

// payAttachmentScopes 限定允许的业务域，key 由它拼出，不接受任意字符串。
var payAttachmentScopes = map[string]struct{}{
	"invoice": {},
}

// payAttachmentRefPattern 约束引用 id（发票 id 为 cuid）：不含 '/'、'.'，
// 因此拼进 key 时不可能造成目录穿越。
var payAttachmentRefPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// payAttachmentAllowedTypes 声明的 Content-Type → 规范扩展名。
// 扩展名只从这里取，绝不从客户端文件名推断。
var payAttachmentAllowedTypes = map[string]string{
	"application/pdf": ".pdf",
	"application/ofd": ".ofd",
	"application/zip": ".zip",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

// payAttachmentAllowedSniffs 是 http.DetectContentType 的粗粒度白名单。
//
// 这是一个「够用即可」的闸门而非严格相等校验：OFD 实际是 zip 容器，会被嗅探成
// application/zip。它要挡住的是「扩展名 .pdf、内容其实是 HTML」——该文件随后会
// 从对象存储的域上以预签名链接提供，若允许 HTML 就等于在自己的源上落一个存储型 XSS。
var payAttachmentAllowedSniffs = map[string]struct{}{
	"application/pdf": {},
	"application/zip": {},
	"image/jpeg":      {},
	"image/png":       {},
}

// PayAttachmentStore 是支付服务附件的对象存储抽象，由 repository 层实现。
type PayAttachmentStore interface {
	Upload(ctx context.Context, key, contentType string, data []byte) error
	PresignDownloadURL(ctx context.Context, key, downloadName string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

// PayAttachmentStoreFactory 由 repository 层提供，把 S3 配置变成一个可用的存储实现。
type PayAttachmentStoreFactory func(ctx context.Context, cfg *BackupS3Config) (PayAttachmentStore, error)

// PayAttachmentService 为支付服务（sub2apipay）提供对象存储读写能力。
//
// 凭证与前缀来自独立的「发票文件存储」设置（InvoiceStorageSettingService），
// 默认复用备份 S3 的凭证，也可以整套单独配置到别的桶或别的服务商。
type PayAttachmentService struct {
	settings *InvoiceStorageSettingService
	factory  PayAttachmentStoreFactory
}

func NewPayAttachmentService(settings *InvoiceStorageSettingService, factory PayAttachmentStoreFactory) *PayAttachmentService {
	return &PayAttachmentService{settings: settings, factory: factory}
}

// PayAttachmentPutInput 描述一次附件上传。
type PayAttachmentPutInput struct {
	Scope               string
	Ref                 string
	FileName            string
	DeclaredContentType string
	Data                []byte
}

// PayAttachmentPutResult 是上传结果，Key 由服务端生成。
type PayAttachmentPutResult struct {
	Key         string `json:"key"`
	FileName    string `json:"file_name"`
	Size        int    `json:"size"`
	ContentType string `json:"content_type"`
}

// Put 校验并上传附件，返回服务端生成的对象 key。
func (s *PayAttachmentService) Put(ctx context.Context, in PayAttachmentPutInput) (*PayAttachmentPutResult, error) {
	scope := strings.TrimSpace(in.Scope)
	if _, ok := payAttachmentScopes[scope]; !ok {
		return nil, ErrPayAttachmentInvalidScope
	}
	ref := strings.TrimSpace(in.Ref)
	if !payAttachmentRefPattern.MatchString(ref) {
		return nil, ErrPayAttachmentInvalidRef
	}

	contentType := normalizeContentType(in.DeclaredContentType)
	ext, ok := payAttachmentAllowedTypes[contentType]
	if !ok {
		return nil, ErrPayAttachmentInvalidType
	}

	switch {
	case len(in.Data) == 0:
		return nil, ErrPayAttachmentEmpty
	case len(in.Data) > MaxPayAttachmentBytes:
		return nil, ErrPayAttachmentTooLarge
	}

	if !sniffAllowed(in.Data) {
		return nil, ErrPayAttachmentContentMismatch
	}

	store, prefix, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}

	key, err := buildPayAttachmentKey(prefix, ref, ext)
	if err != nil {
		return nil, err
	}
	if err := store.Upload(ctx, key, contentType, in.Data); err != nil {
		return nil, err
	}

	return &PayAttachmentPutResult{
		Key:         key,
		FileName:    fallbackFileName(in.FileName, ref, ext),
		Size:        len(in.Data),
		ContentType: contentType,
	}, nil
}

// Presign 为已上传的附件生成短期下载链接。
func (s *PayAttachmentService) Presign(ctx context.Context, key, downloadName string, ttl time.Duration) (string, time.Time, error) {
	ttl = clampPresignTTL(ttl)

	store, prefix, err := s.resolve(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	// 前缀校验必须在解析出当前前缀之后做：它是授权边界，不能用调用方给的值。
	if err := validatePayAttachmentKey(key, prefix); err != nil {
		return "", time.Time{}, err
	}
	url, err := store.PresignDownloadURL(ctx, key, downloadName, ttl)
	if err != nil {
		return "", time.Time{}, err
	}
	return url, time.Now().Add(ttl), nil
}

// Delete 删除附件对象（用于回滚孤儿对象、或替换文件后清理旧对象）。
func (s *PayAttachmentService) Delete(ctx context.Context, key string) error {
	store, prefix, err := s.resolve(ctx)
	if err != nil {
		return err
	}
	if err := validatePayAttachmentKey(key, prefix); err != nil {
		return err
	}
	return store.Delete(ctx, key)
}

// resolve 取出当前生效的存储客户端与对象前缀。
//
// 每次现建客户端，刻意不缓存客户端本身：设置服务已经缓存了解析后的配置并在管理员
// 改设置时失效，这里再存一份就会在改配置后读到陈旧凭证。发票流量只有每天个位数，
// 而 S3 客户端构造是纯本地操作（不发网络请求），现建的代价可以忽略。
func (s *PayAttachmentService) resolve(ctx context.Context) (PayAttachmentStore, string, error) {
	if s == nil || s.settings == nil || s.factory == nil {
		return nil, "", ErrPayAttachmentStorageNotConfigured
	}
	cfg, prefix, ok := s.settings.Resolve(ctx)
	if !ok {
		return nil, "", ErrPayAttachmentStorageNotConfigured
	}
	store, err := s.factory(ctx, cfg)
	if err != nil {
		return nil, "", err
	}
	return store, prefix, nil
}

// buildPayAttachmentKey 完全由服务端拼 key，不含任何客户端可控的自由文本。
// 形如 invoices/2026/08/<ref>-<rand8>.pdf
//
// scope 没有出现在路径里：前缀本身就是命名空间，由管理员配置。若将来新增第二种
// scope，需要把 scope 重新拼回路径，否则两类附件会混在同一前缀下。
func buildPayAttachmentKey(prefix, ref, ext string) (string, error) {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return "", infraerrors.InternalServer("PAY_ATTACHMENT_KEY_FAILED", "failed to generate attachment key")
	}
	now := time.Now().UTC()
	return prefix + now.Format("2006/01") + "/" + ref + "-" + hex.EncodeToString(suffix) + ext, nil
}

// validatePayAttachmentKey 是 presign/delete 的授权边界。prefix 必须来自当前生效的
// 设置，不能取调用方传入的值。
func validatePayAttachmentKey(key, prefix string) error {
	key = strings.TrimSpace(key)
	if key == "" || prefix == "" || !strings.HasPrefix(key, prefix) {
		return ErrPayAttachmentInvalidKey
	}
	// path.Clean 会把 "a/../../b" 规约掉，规约后仍必须落在前缀内。
	if strings.Contains(key, "..") || path.Clean(key) != key {
		return ErrPayAttachmentInvalidKey
	}
	return nil
}

func clampPresignTTL(ttl time.Duration) time.Duration {
	switch {
	case ttl <= 0:
		return defaultPayAttachmentPresignTTL
	case ttl < minPayAttachmentPresignTTL:
		return minPayAttachmentPresignTTL
	case ttl > maxPayAttachmentPresignTTL:
		return maxPayAttachmentPresignTTL
	default:
		return ttl
	}
}

func normalizeContentType(raw string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(raw, ";")[0]))
}

func sniffAllowed(data []byte) bool {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	sniffed := normalizeContentType(http.DetectContentType(head))
	_, ok := payAttachmentAllowedSniffs[sniffed]
	return ok
}

// fallbackFileName 清洗客户端文件名；它只用作下载展示名，不参与 key 构造。
func fallbackFileName(name, ref, ext string) string {
	name = strings.NewReplacer("\r", "", "\n", "", "\x00", "", "/", "_", "\\", "_").Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return ref + ext
	}
	if len([]rune(name)) > 120 {
		name = string([]rune(name)[:120])
	}
	return name
}
