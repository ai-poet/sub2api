package admin

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// TicketAttachmentHandler 客服侧工单图片附件接口（fork 本地功能）：
// 上传落在 prefix/staff/<uid>/ 下，读取只要 key 在附件前缀内即可（operator 同权）。
type TicketAttachmentHandler struct {
	svc *service.TicketAttachmentService
}

// NewTicketAttachmentHandler 构造客服侧工单附件 handler。
func NewTicketAttachmentHandler(svc *service.TicketAttachmentService) *TicketAttachmentHandler {
	return &TicketAttachmentHandler{svc: svc}
}

// Upload POST /api/v1/admin/tickets/attachments
func (h *TicketAttachmentHandler) Upload(c *gin.Context) {
	// 二进制 body 不进审计日志。
	middleware.SkipAudit(c)

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, service.ErrTicketAttachmentStorageNotConfigured)
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.ErrorFrom(c, service.ErrTicketAttachmentTooLarge)
			return
		}
		response.ErrorFrom(c, service.ErrTicketAttachmentEmpty)
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.ErrorFrom(c, service.ErrTicketAttachmentTooLarge)
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	declaredType := header.Header.Get("Content-Type")
	if declaredType == "" {
		declaredType = http.DetectContentType(data)
	}

	result, err := h.svc.Put(c.Request.Context(), service.TicketAttachmentPutInput{
		UserID:              subject.UserID,
		Staff:               true,
		DeclaredContentType: declaredType,
		Data:                data,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, result)
}

// Content GET /api/v1/admin/tickets/attachments/content?key=...
func (h *TicketAttachmentHandler) Content(c *gin.Context) {
	if h == nil || h.svc == nil {
		response.ErrorFrom(c, service.ErrTicketAttachmentStorageNotConfigured)
		return
	}

	content, err := h.svc.OpenForStaff(c.Request.Context(), c.Query("key"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
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
