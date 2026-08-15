package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

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

// PresignAttachment 为附件签发短期下载链接。
func (h *PayBridgeHandler) PresignAttachment(c *gin.Context) {
	if h == nil || h.attachments == nil {
		response.ErrorFrom(c, service.ErrPayAttachmentStorageNotConfigured)
		return
	}

	var req dto.PayAttachmentPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ttl := time.Duration(req.ExpiresInSeconds) * time.Second
	signedURL, expiresAt, err := h.attachments.Presign(c.Request.Context(), req.Key, req.FileName, ttl)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.PayAttachmentPresignResponse{
		URL:       signedURL,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
	})
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
