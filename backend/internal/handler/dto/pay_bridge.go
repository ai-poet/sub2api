package dto

// PayAttachmentPresignRequest 请求为已上传附件签发临时下载链接。
type PayAttachmentPresignRequest struct {
	Key      string `json:"key" binding:"required"`
	FileName string `json:"file_name"`
	// ExpiresInSeconds 会被服务端钳制到 [60, 900]。
	ExpiresInSeconds int `json:"expires_in_seconds"`
}

// PayAttachmentPresignResponse 是签发结果。
type PayAttachmentPresignResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

// PayAttachmentDeleteRequest 请求删除附件对象。
type PayAttachmentDeleteRequest struct {
	Key string `json:"key" binding:"required"`
}

// PayInvoiceReadyRequest 触发「发票已开具」通知。
//
// 刻意不包含收件邮箱与跳转链接：两者都由后端根据 user_id 与站点设置解析，
// 避免这个内部接口被当作定向发信中继使用。
type PayInvoiceReadyRequest struct {
	UserID        int64  `json:"user_id" binding:"required"`
	InvoiceID     string `json:"invoice_id" binding:"required"`
	OrderID       string `json:"order_id"`
	AmountDisplay string `json:"amount_display"`
	TitleName     string `json:"title_name"`
	TaxNo         string `json:"tax_no"`
	IssuedAt      string `json:"issued_at"`
	ReminderKey   string `json:"reminder_key"`
}
