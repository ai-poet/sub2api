package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// PayBridgeHandler 服务于支付服务（sub2apipay）的内部桥接接口：对象存储读写与
// 开票通知邮件。这些路由挂在 /api/internal/pay 下，由 internalPayAuthMiddleware
// 用 JWT_SECRET 派生的共享令牌保护。
type PayBridgeHandler struct {
	attachments   *service.PayAttachmentService
	invoiceNotify *service.PayInvoiceNotifyService
}

func NewPayBridgeHandler(
	attachments *service.PayAttachmentService,
	invoiceNotify *service.PayInvoiceNotifyService,
) *PayBridgeHandler {
	return &PayBridgeHandler{attachments: attachments, invoiceNotify: invoiceNotify}
}

// UploadAttachment 接收原始二进制请求体并写入对象存储。
//
// 走裸二进制而非 multipart：这一跳是服务间调用，元数据放 query/header 即可，
// 引入 multipart 会让 Gin 在超过 MaxMultipartMemory 时静默落临时文件，凭空多出
// 一个磁盘故障面。浏览器那一跳仍然是 multipart，由 Next.js 侧解析。
//
// POST /api/internal/pay/attachments?scope=invoice&ref=<id>&filename=<urlencoded>
func (h *PayBridgeHandler) UploadAttachment(c *gin.Context) {
	if h == nil || h.attachments == nil {
		response.ErrorFrom(c, service.ErrPayAttachmentStorageNotConfigured)
		return
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		// http.MaxBytesReader 超限时返回错误，映射成 413 而不是 400。
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			response.Error(c, http.StatusRequestEntityTooLarge, "attachment exceeds the maximum allowed size")
			return
		}
		response.BadRequest(c, "failed to read attachment body")
		return
	}

	fileName := c.Query("filename")
	if decoded, decodeErr := url.QueryUnescape(fileName); decodeErr == nil {
		fileName = decoded
	}

	result, err := h.attachments.Put(c.Request.Context(), service.PayAttachmentPutInput{
		Scope:               c.Query("scope"),
		Ref:                 c.Query("ref"),
		FileName:            fileName,
		DeclaredContentType: c.GetHeader("Content-Type"),
		Data:                data,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// DownloadAttachment 把附件内容流式回传。
//
// 不签发预签名链接：下载页跑在 iframe 里，让浏览器直接跳到对象存储会被父页面 CSP
// 的 frame-src 拦下（Chrome 提示 "This content is blocked"），HTTPS 页面里跳
// http:// 还会再撞一次混合内容。同源回传绕开这两道，对象存储也不必对公网暴露。
//
// GET /api/internal/pay/attachments/content?key=...&filename=...
func (h *PayBridgeHandler) DownloadAttachment(c *gin.Context) {
	if h == nil || h.attachments == nil {
		response.ErrorFrom(c, service.ErrPayAttachmentStorageNotConfigured)
		return
	}

	content, err := h.attachments.Open(c.Request.Context(), c.Query("key"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = content.Body.Close() }()

	fileName := c.Query("filename")
	if decoded, decodeErr := url.QueryUnescape(fileName); decodeErr == nil {
		fileName = decoded
	}

	contentType := content.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", service.AttachmentContentDisposition(fileName))
	// 发票是客户的商业信息，任何一层都不该缓存。
	c.Header("Cache-Control", "no-store, private")
	c.Header("X-Content-Type-Options", "nosniff")
	if content.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(content.Size, 10))
	}

	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, content.Body); err != nil {
		// 头已经发出去了，只能记日志——此时再写 JSON 错误体只会污染文件内容。
		log.Printf("[pay_bridge] streaming attachment failed: %v", err)
	}
}

// DeleteAttachment 删除附件对象（回滚孤儿对象、替换文件后清理旧对象）。
func (h *PayBridgeHandler) DeleteAttachment(c *gin.Context) {
	if h == nil || h.attachments == nil {
		response.ErrorFrom(c, service.ErrPayAttachmentStorageNotConfigured)
		return
	}

	var req dto.PayAttachmentDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if err := h.attachments.Delete(c.Request.Context(), req.Key); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// SendInvoiceReadyEmail 发送「发票已开具」通知邮件。
func (h *PayBridgeHandler) SendInvoiceReadyEmail(c *gin.Context) {
	if h == nil || h.invoiceNotify == nil {
		response.InternalError(c, "invoice notification service is unavailable")
		return
	}

	var req dto.PayInvoiceReadyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	err := h.invoiceNotify.SendInvoiceReady(c.Request.Context(), service.PayInvoiceReadyInput{
		UserID:        req.UserID,
		InvoiceID:     req.InvoiceID,
		OrderID:       req.OrderID,
		AmountDisplay: req.AmountDisplay,
		TitleName:     req.TitleName,
		TaxNo:         req.TaxNo,
		IssuedAt:      req.IssuedAt,
		ReminderKey:   req.ReminderKey,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"sent": true})
}
