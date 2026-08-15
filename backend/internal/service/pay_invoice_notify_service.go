package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrInvoiceRecipientNotFound = infraerrors.NotFound(
	"INVOICE_RECIPIENT_NOT_FOUND", "invoice recipient user not found")

// PayInvoiceReadyInput 描述一次「发票已开具」通知。
//
// 注意这里没有收件邮箱字段：收件人一律由 UserID 在服务端解析。支付服务持有内部
// 桥接令牌，若允许它指定收件人，这个接口就成了一个可定向发信的中继；发票通知里
// 又带着站点品牌和下载链接，正好是钓鱼所需要的一切。
type PayInvoiceReadyInput struct {
	UserID        int64
	InvoiceID     string
	OrderID       string
	AmountDisplay string
	TitleName     string
	TaxNo         string
	IssuedAt      string
	// ReminderKey 参与投递去重键。传对象存储的 fileKey：同一份文件重复通知会被
	// 去重，换了发票文件后重发才会真正发出。若留空，则「重新发送通知」会因为投递
	// 标记已存在而永久静默失败。
	ReminderKey string
}

// PayInvoiceNotifyService 把支付服务的开票结果转成一封模板化通知邮件。
type PayInvoiceNotifyService struct {
	notification *NotificationEmailService
	users        *UserService
}

func NewPayInvoiceNotifyService(notification *NotificationEmailService, users *UserService) *PayInvoiceNotifyService {
	return &PayInvoiceNotifyService{notification: notification, users: users}
}

// SendInvoiceReady 发送「发票已开具」通知。
func (s *PayInvoiceNotifyService) SendInvoiceReady(ctx context.Context, in PayInvoiceReadyInput) error {
	if s == nil || s.notification == nil || s.users == nil {
		return infraerrors.InternalServer("INVOICE_NOTIFY_UNAVAILABLE", "invoice notification service is unavailable")
	}

	user, err := s.users.GetByID(ctx, in.UserID)
	if err != nil || user == nil {
		return ErrInvoiceRecipientNotFound
	}
	recipient := strings.TrimSpace(user.Email)
	if recipient == "" {
		return ErrInvoiceRecipientNotFound
	}

	// 链接由本服务拼装，不接受调用方传入。isSafeNotificationEmailURL 只校验
	// scheme 不校验 host，放行外部 URL 等于在自家品牌邮件里开一个钓鱼位。
	// 指向 Vue 的 /purchase 而非 /pay/orders：后者脱离控制台就没有 token，打不开。
	invoiceURL := ""
	if base := s.notification.baseURL(ctx); base != "" {
		invoiceURL = base + "/purchase"
	}

	return s.notification.Send(ctx, NotificationEmailSendInput{
		Event:          NotificationEmailEventBillingInvoiceReady,
		RecipientEmail: recipient,
		RecipientName:  user.Username,
		UserID:         user.ID,
		SourceType:     "invoice",
		SourceID:       in.InvoiceID,
		ReminderKey:    in.ReminderKey,
		// Locale 留空：交给 Send 走 ResolveRecipientLocale，用用户记住的语言。
		Variables: map[string]string{
			"invoice_amount":    in.AmountDisplay,
			"invoice_title":     in.TitleName,
			"invoice_tax_no":    in.TaxNo,
			"order_id":          in.OrderID,
			"invoice_issued_at": in.IssuedAt,
			"invoice_url":       invoiceURL,
		},
	})
}
