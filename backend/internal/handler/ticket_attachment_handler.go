package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// TicketAttachmentHandler 用户侧工单图片附件接口（fork 本地功能）：
// 上传走 multipart，读取按 prefix+<自己的uid>/ 前缀授权，越界一律 404。
type TicketAttachmentHandler struct {
	svc *service.TicketAttachmentService
}

// NewTicketAttachmentHandler 构造用户侧工单附件 handler。
func NewTicketAttachmentHandler(svc *service.TicketAttachmentService) *TicketAttachmentHandler {
	return &TicketAttachmentHandler{svc: svc}
}

// readTicketAttachmentFile 读出 multipart 的 file 字段与声明的 Content-Type。
// 路由上的 RequestBodyLimit 超限时会冒成 http.MaxBytesError，映射成 413。
func readTicketAttachmentFile(c *gin.Context) (data []byte, declaredType string, err error) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, "", service.ErrTicketAttachmentTooLarge
		}
		return nil, "", service.ErrTicketAttachmentEmpty
	}
	defer func() { _ = file.Close() }()

	data, err = io.ReadAll(file)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return nil, "", service.ErrTicketAttachmentTooLarge
		}
		return nil, "", err
	}
	declaredType = header.Header.Get("Content-Type")
	if declaredType == "" {
		declaredType = http.DetectContentType(data)
	}
	return data, declaredType, nil
}

// streamTicketAttachment 把附件内容同源回传：Content-Type 再过一遍白名单
// （挡住存储端被带外写入的非图片对象），并且任何一层都不该缓存。
func streamTicketAttachment(c *gin.Context, content *service.TicketAttachmentContent) {
	defer func() { _ = content.Body.Close() }()

	contentType := content.ContentType
	if !service.TicketAttachmentAllowedContentType(contentType) {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "no-store, private")
	c.Header("X-Content-Type-Options", "nosniff")
	if content.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(content.Size, 10))
	}

	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, content.Body); err != nil {
		// 头已经发出去了，只能记日志——此时再写 JSON 错误体只会污染文件内容。
		log.Printf("[ticket_attachment] streaming attachment failed: %v", err)
	}
}

// Upload POST /api/v1/tickets/attachments
func (h *TicketAttachmentHandler) Upload(c *gin.Context) {
	// 二进制 body 不进审计日志。
	middleware2.SkipAudit(c)

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, service.ErrTicketAttachmentStorageNotConfigured)
		return
	}

	data, declaredType, err := readTicketAttachmentFile(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	result, err := h.svc.Put(c.Request.Context(), service.TicketAttachmentPutInput{
		UserID:              subject.UserID,
		DeclaredContentType: declaredType,
		Data:                data,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, result)
}

// Content GET /api/v1/tickets/attachments/content?key=...
func (h *TicketAttachmentHandler) Content(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, service.ErrTicketAttachmentStorageNotConfigured)
		return
	}

	content, err := h.svc.OpenForUser(c.Request.Context(), c.Query("key"), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	streamTicketAttachment(c, content)
}
