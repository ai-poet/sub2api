package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvoiceReadyEventIsListedAndPreviewableInBothLocales(t *testing.T) {
	ctx := context.Background()
	svc := NewNotificationEmailService(newNotificationEmailMemorySettingRepo(), nil)

	var info NotificationEmailEventInfo
	var found bool
	for _, candidate := range svc.ListEventInfos() {
		if candidate.Event == NotificationEmailEventBillingInvoiceReady {
			info, found = candidate, true
			break
		}
	}
	require.True(t, found, "billing.invoice_ready must be listed for the admin template editor")

	// Transactional: the user asked for this invoice, so unsubscribe must not
	// be able to suppress it (and no unsubscribe_url placeholder is needed).
	require.False(t, info.Optional)
	require.NotContains(t, info.Placeholders, "unsubscribe_url")
	for _, placeholder := range []string{
		"site_name", "recipient_name", "recipient_email",
		"invoice_amount", "invoice_title", "invoice_tax_no",
		"order_id", "invoice_issued_at", "invoice_url",
	} {
		require.Containsf(t, info.Placeholders, placeholder, "missing placeholder %s", placeholder)
	}

	// An official template must exist for every supported locale, otherwise
	// GetTemplate fails at send time with "official template not found".
	for _, locale := range svc.SupportedLocales() {
		preview, err := svc.PreviewTemplate(ctx, NotificationEmailPreviewInput{
			Event:  NotificationEmailEventBillingInvoiceReady,
			Locale: locale,
		})
		require.NoErrorf(t, err, "preview failed for locale %s", locale)
		require.NotEmpty(t, preview.Subject)
		// Locale-independent: every template must actually substitute the
		// invoice fields rather than leaving raw {{placeholders}} behind.
		sample := notificationEmailSampleVariables(locale)
		require.Containsf(t, preview.HTML, sample["invoice_amount"], "locale %s dropped invoice_amount", locale)
		require.Containsf(t, preview.HTML, sample["invoice_title"], "locale %s dropped invoice_title", locale)
		require.NotContainsf(t, preview.HTML, "{{", "locale %s left an unrendered placeholder", locale)
	}
}

func TestInvoiceReadyTemplateRejectsUnregisteredPlaceholder(t *testing.T) {
	_, err := renderNotificationEmail(
		NotificationEmailEventBillingInvoiceReady,
		"[{{site_name}}] invoice",
		"<p>{{invoice_amount}} {{invoice_pdf_password}}</p>",
		map[string]string{"site_name": "sub2api", "invoice_amount": "¥1.00"},
		nil,
	)
	require.Error(t, err, "an unregistered placeholder must be rejected at render time")
}

func TestInvoiceReadyTemplateEscapesValuesAndDropsUnsafeURL(t *testing.T) {
	preview, err := renderNotificationEmail(
		NotificationEmailEventBillingInvoiceReady,
		"[{{site_name}}] invoice",
		`<p>{{invoice_title}}</p><a href="{{invoice_url}}">go</a>`,
		map[string]string{
			"site_name":     "sub2api",
			"invoice_title": `<script>alert(1)</script>`,
			"invoice_url":   "javascript:alert(1)",
		},
		nil,
	)
	require.NoError(t, err)
	require.NotContains(t, preview.HTML, "<script>")
	require.Contains(t, preview.HTML, "&lt;script&gt;")
	require.NotContains(t, preview.HTML, "javascript:")
}
