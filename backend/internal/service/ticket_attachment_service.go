package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 工单图片附件（fork 本地功能）。消息体是 Markdown，图片以
// ![...](ticket-attachment://<key>) 形式写进 body；key 由本服务在上传时生成，
// 渲染时换成同源回传端点 /api/v1/tickets/attachments/content?key=...。
//
// 对象前缀由 TicketAttachmentStorageSettingService 提供（后台可配置，默认
// DefaultTicketAttachmentStoragePrefix）。它同时是读取的授权边界：用户能读
// prefix+<自己的uid>/ 下的 key，以及 prefix+staff/ 下、被自己工单里某条客服消息
// 引用的 key（客服回复里贴的图）；客服侧只能读 prefix 下的 key。

// MaxTicketAttachmentBytes 单张工单图片上限（5 MiB）。
const MaxTicketAttachmentBytes = 5 << 20

// TicketAttachmentURLScheme 消息正文里引用附件的 URL scheme：![...](ticket-attachment://<key>)。
const TicketAttachmentURLScheme = "ticket-attachment://"

// ticketAttachmentStaffSegment 客服上传的 key 在前缀后的固定段：<prefix>staff/<uid>/...。
const ticketAttachmentStaffSegment = "staff/"

var (
	ErrTicketAttachmentStorageNotConfigured = infraerrors.ServiceUnavailable(
		"TICKET_ATTACHMENT_STORAGE_NOT_CONFIGURED",
		"ticket attachment storage is not configured",
	)
	ErrTicketAttachmentTooLarge = infraerrors.New(
		http.StatusRequestEntityTooLarge,
		"TICKET_ATTACHMENT_TOO_LARGE", "attachment exceeds the maximum allowed size")
	ErrTicketAttachmentBadType = infraerrors.BadRequest(
		"TICKET_ATTACHMENT_BAD_TYPE", "unsupported attachment content type")
	ErrTicketAttachmentEmpty = infraerrors.BadRequest(
		"TICKET_ATTACHMENT_EMPTY", "attachment is empty")
	// ErrTicketAttachmentNotFound 用户侧读到他人 / 不存在的 key 时统一按不存在处理，不泄露存在性。
	ErrTicketAttachmentNotFound = infraerrors.NotFound(
		"TICKET_ATTACHMENT_NOT_FOUND", "attachment not found")
	ErrTicketAttachmentInvalidKey = infraerrors.BadRequest(
		"TICKET_ATTACHMENT_INVALID_KEY", "attachment key is outside the allowed prefix")
)

// ticketAttachmentAllowedTypes 声明的 Content-Type → 规范扩展名。
// 扩展名只从这里取，绝不从客户端文件名推断。
var ticketAttachmentAllowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// TicketAttachmentAllowedContentType 报告 contentType 是否在工单图片白名单内，
// 供同源回传时二次过滤对象存储里返回的 Content-Type。
func TicketAttachmentAllowedContentType(contentType string) bool {
	_, ok := ticketAttachmentAllowedTypes[normalizeContentType(contentType)]
	return ok
}

// ticketAttachmentSniffAllowed 用 http.DetectContentType 的魔数嗅探兜底：
// 声明的类型可以是假的，图片随后会被同源回传给浏览器，若放过 HTML 就等于在
// 自己的源上落一个存储型 XSS。
func ticketAttachmentSniffAllowed(data []byte) bool {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	sniffed := normalizeContentType(http.DetectContentType(head))
	_, ok := ticketAttachmentAllowedTypes[sniffed]
	return ok
}

// TicketAttachmentStore 是工单图片附件的对象存储抽象，由 repository 层实现。
type TicketAttachmentStore interface {
	Upload(ctx context.Context, key, contentType string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadCloser, string, int64, error)
}

// TicketAttachmentStoreFactory 由 repository 层提供，把 S3 配置变成一个可用的存储实现。
type TicketAttachmentStoreFactory func(ctx context.Context, cfg *BackupS3Config) (TicketAttachmentStore, error)

// TicketAttachmentService 为工单消息里的图片附件提供对象存储读写能力。
// tickets 用来回答"这个客服 key 是否被该用户的工单引用"；为 nil 时用户只能读自己前缀下的 key。
type TicketAttachmentService struct {
	settings *TicketAttachmentStorageSettingService
	factory  TicketAttachmentStoreFactory
	tickets  SupportTicketRepository
}

func NewTicketAttachmentService(settings *TicketAttachmentStorageSettingService, factory TicketAttachmentStoreFactory, tickets SupportTicketRepository) *TicketAttachmentService {
	return &TicketAttachmentService{settings: settings, factory: factory, tickets: tickets}
}

// TicketAttachmentPutInput 描述一次附件上传。UserID 是当前登录用户；
// Staff 为真时 key 落在 prefix/staff/<uid>/ 下（客服侧上传）。
type TicketAttachmentPutInput struct {
	UserID              int64
	Staff               bool
	DeclaredContentType string
	Data                []byte
}

// TicketAttachmentPutResult 是上传结果，Key 由服务端生成。
type TicketAttachmentPutResult struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

// Put 校验并上传附件，返回服务端生成的对象 key。
func (s *TicketAttachmentService) Put(ctx context.Context, in TicketAttachmentPutInput) (*TicketAttachmentPutResult, error) {
	if in.UserID <= 0 {
		return nil, infraerrors.Unauthorized("TICKET_ATTACHMENT_NO_OWNER", "attachment uploader is not authenticated")
	}

	contentType := normalizeContentType(in.DeclaredContentType)
	ext, ok := ticketAttachmentAllowedTypes[contentType]
	if !ok {
		return nil, ErrTicketAttachmentBadType
	}

	switch {
	case len(in.Data) == 0:
		return nil, ErrTicketAttachmentEmpty
	case len(in.Data) > MaxTicketAttachmentBytes:
		return nil, ErrTicketAttachmentTooLarge
	}

	if !ticketAttachmentSniffAllowed(in.Data) {
		return nil, ErrTicketAttachmentBadType
	}

	store, prefix, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}

	key, err := buildTicketAttachmentKey(prefix, in.UserID, in.Staff, ext)
	if err != nil {
		return nil, err
	}
	if err := store.Upload(ctx, key, contentType, in.Data); err != nil {
		return nil, err
	}

	return &TicketAttachmentPutResult{
		Key:         key,
		ContentType: contentType,
		Size:        len(in.Data),
	}, nil
}

// TicketAttachmentContent 是一次附件读取的结果，调用方负责 Close。
type TicketAttachmentContent struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

// OpenForUser 取回用户侧附件内容。放行两类 key：
//   - prefix+<userID>/ 下的（自己上传的）；
//   - prefix+staff/ 下、且被该用户拥有的某个工单里的一条客服消息引用的（客服回复里贴的图）。
//
// 引用只认客服写的消息（author_role 不是 user），用户在自己正文里塞进别人的 key 读不到；
// 也只放行 staff/ 段，客服误贴另一个用户 prefix+<他人uid>/ 下的 key 不会让他人的上传物外泄。
// 其余（他人的、未被引用的、前缀之外的、穿越的）一律按不存在处理（404），不泄露对象是否存在。
func (s *TicketAttachmentService) OpenForUser(ctx context.Context, key string, userID int64) (*TicketAttachmentContent, error) {
	store, prefix, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if userID <= 0 {
		return nil, ErrTicketAttachmentNotFound
	}
	if validTicketAttachmentKeyWithin(key, prefix+strconv.FormatInt(userID, 10)+"/") {
		return openTicketAttachment(ctx, store, key)
	}
	if s.tickets == nil || !validTicketAttachmentKeyWithin(key, prefix+ticketAttachmentStaffSegment) {
		return nil, ErrTicketAttachmentNotFound
	}
	referenced, err := s.tickets.StaffAttachmentReferencedForUser(ctx, userID, key)
	if err != nil {
		return nil, err
	}
	if !referenced {
		return nil, ErrTicketAttachmentNotFound
	}
	return openTicketAttachment(ctx, store, key)
}

// OpenForStaff 取回客服侧附件内容：key 必须落在当前生效的附件前缀下，
// 且通过穿越校验（挡 '..'）。
func (s *TicketAttachmentService) OpenForStaff(ctx context.Context, key string) (*TicketAttachmentContent, error) {
	store, prefix, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}
	if !validTicketAttachmentKeyWithin(key, prefix) {
		return nil, ErrTicketAttachmentInvalidKey
	}
	return openTicketAttachment(ctx, store, key)
}

func openTicketAttachment(ctx context.Context, store TicketAttachmentStore, key string) (*TicketAttachmentContent, error) {
	body, contentType, size, err := store.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	return &TicketAttachmentContent{Body: body, ContentType: contentType, Size: size}, nil
}

// resolve 取出当前生效的存储客户端与对象前缀。
//
// 每次现建客户端，刻意不缓存：设置服务已经缓存了解析后的配置并在管理员改设置时
// 失效，这里再存一份就会在改配置后读到陈旧凭证；S3 客户端构造是纯本地操作，
// 现建的代价可以忽略。
func (s *TicketAttachmentService) resolve(ctx context.Context) (TicketAttachmentStore, string, error) {
	if s == nil || s.settings == nil || s.factory == nil {
		return nil, "", ErrTicketAttachmentStorageNotConfigured
	}
	cfg, prefix, ok := s.settings.Resolve(ctx)
	if !ok {
		return nil, "", ErrTicketAttachmentStorageNotConfigured
	}
	store, err := s.factory(ctx, cfg)
	if err != nil {
		return nil, "", err
	}
	return store, prefix, nil
}

// buildTicketAttachmentKey 完全由服务端拼 key，不含任何客户端可控的自由文本。
// 用户：tickets/<uid>/<yyyymm>/<rand8hex>.<ext>；客服：tickets/staff/<uid>/...。
func buildTicketAttachmentKey(prefix string, userID int64, staff bool, ext string) (string, error) {
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return "", infraerrors.InternalServer("TICKET_ATTACHMENT_KEY_FAILED", "failed to generate attachment key")
	}
	owner := strconv.FormatInt(userID, 10)
	if staff {
		owner = ticketAttachmentStaffSegment + owner
	}
	now := time.Now().UTC()
	return prefix + owner + "/" + now.Format("200601") + "/" + hex.EncodeToString(suffix) + ext, nil
}

// validTicketAttachmentKeyWithin 是读取的授权边界。boundary 必须来自当前生效的
// 设置与已认证身份，不能取调用方传入的值。
func validTicketAttachmentKeyWithin(key, boundary string) bool {
	key = strings.TrimSpace(key)
	if key == "" || boundary == "" || !strings.HasPrefix(key, boundary) {
		return false
	}
	// path.Clean 会把 "a/../../b" 规约掉，规约后仍必须原样落在边界内。
	if strings.Contains(key, "..") || path.Clean(key) != key {
		return false
	}
	return true
}
