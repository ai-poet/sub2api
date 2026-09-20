package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTicketNotifyServiceForTest(baseURL string, values map[string]string) *TicketNotifyService {
	return &TicketNotifyService{
		settingRepo: &groupStatusNotifySettingRepo{values: values},
		client:      newServerChanClientForTest(baseURL),
	}
}

func sampleTicketForNotify() (*SupportTicket, *SupportTicketMessage) {
	ticket := &SupportTicket{
		ID:            7,
		UserID:        3,
		UserEmail:     "alice@example.com",
		Title:         "API 报 429",
		Category:      TicketCategoryAPI,
		Status:        TicketStatusOpen,
		MessageCount:  1,
		LastMessageAt: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
	}
	msg := &SupportTicketMessage{
		ID:         11,
		TicketID:   7,
		AuthorRole: TicketAuthorRoleUser,
		Body:       "调用 gpt-5 时\n一直返回 429，\r\n从今天早上开始。",
		CreatedAt:  time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
	}
	return ticket, msg
}

func TestTicketNotifyService_DeliversCreatedWithLink(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newTicketNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyTicketNotifyServerChanEnabled:      "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
		SettingKeySiteName:                           "MySite",
		SettingKeyFrontendURL:                        "https://console.example.com/",
	})
	ticket, msg := sampleTicketForNotify()

	require.NoError(t, svc.deliver(ticketNotifyKindCreated, ticket, msg, "http://ignored.local"))
	require.Equal(t, 1, ts.count())
	got := ts.request(0)
	require.Equal(t, "/send/sk_test.send", got.Path)
	require.Equal(t, "[MySite] 新工单 #7：API 报 429", got.Title)
	require.Contains(t, got.Desp, "**用户**：alice@example.com")
	require.Contains(t, got.Desp, "**分类**：API 调用")
	require.Contains(t, got.Desp, "**内容**：调用 gpt-5 时 一直返回 429， 从今天早上开始。", "正文折成单行")
	require.Contains(t, got.Desp, "[查看工单](https://console.example.com/admin/tickets?id=7)")
}

func TestTicketNotifyService_UserRepliedFallsBackToOrigin(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newTicketNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyTicketNotifyServerChanEnabled:      "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
	})
	ticket, msg := sampleTicketForNotify()
	msg.Body = strings.Repeat("长", ticketNotifyExcerptRunes+50)

	require.NoError(t, svc.deliver(ticketNotifyKindUserReplied, ticket, msg, "https://origin.example.com"))
	require.Equal(t, 1, ts.count())
	got := ts.request(0)
	require.Equal(t, "[Sub2API] 工单 #7 有新回复：API 报 429", got.Title, "站点名回退默认值")
	require.Contains(t, got.Desp, "https://origin.example.com/admin/tickets?id=7")
	require.Contains(t, got.Desp, strings.Repeat("长", ticketNotifyExcerptRunes)+"…")
	require.NotContains(t, got.Desp, strings.Repeat("长", ticketNotifyExcerptRunes+1))
}

func TestTicketNotifyService_SkipsWhenDisabledOrMisconfigured(t *testing.T) {
	ts := newServerChanTestServer(t)
	ticket, msg := sampleTicketForNotify()

	disabled := newTicketNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyTicketNotifyServerChanEnabled:      "false",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
	})
	require.NoError(t, disabled.deliver(ticketNotifyKindCreated, ticket, msg, ""))

	missingKey := newTicketNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyTicketNotifyServerChanEnabled:  "true",
		SettingKeyGroupStatusNotifyServerChanUID: "uid-1",
	})
	require.NoError(t, missingKey.deliver(ticketNotifyKindCreated, ticket, msg, ""))

	badUID := newTicketNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyTicketNotifyServerChanEnabled:      "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "evil.host/",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
	})
	require.NoError(t, badUID.deliver(ticketNotifyKindCreated, ticket, msg, ""))

	require.Equal(t, 0, ts.count())

	var nilSvc *TicketNotifyService
	nilSvc.NotifyTicketCreated(ticket, msg, "")
	nilSvc.NotifyTicketUserReplied(ticket, msg, "")
	require.NoError(t, nilSvc.deliver(ticketNotifyKindCreated, ticket, msg, ""))
}

func TestTicketNotifyService_RedactsSendKeyInErrors(t *testing.T) {
	ts := newServerChanTestServer(t, serverChanTestResponse{Status: 500, Body: `{"code":500}`})
	svc := newTicketNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyTicketNotifyServerChanEnabled:      "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_secret_value",
	})
	ticket, msg := sampleTicketForNotify()
	err := svc.deliver(ticketNotifyKindCreated, ticket, msg, "")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "sk_secret_value")
	require.Equal(t, groupStatusNotifyMaxAttempts, ts.count(), "带重试")
}

func TestTicketNotifyHelpers(t *testing.T) {
	require.Equal(t, "", ticketPageLink("", "", 1))
	require.Equal(t, "https://a.b/admin/tickets?id=7", ticketPageLink("https://a.b/", "", 7))
	require.Equal(t, "http://c.d/admin/tickets?id=7", ticketPageLink("", "http://c.d/", 7))

	require.Equal(t, "账号", ticketCategoryLabel(TicketCategoryAccount))
	require.Equal(t, "-", ticketCategoryLabel(""))
	require.Equal(t, "weird", ticketCategoryLabel("weird"))

	require.Equal(t, "-", ticketNotifyExcerpt("  \n ", 10))
	require.Equal(t, "a b c", ticketNotifyExcerpt(" a\nb\r\n  c ", 10))
	require.Equal(t, "abc…", ticketNotifyExcerpt("abcdef", 3))

	// 没有消息时正文用 "-"，时间回退到 last_message_at
	ticket, _ := sampleTicketForNotify()
	title, desp := buildTicketNotifyMessage(ticketNotifyKindCreated, "", ticket, nil, "")
	require.Equal(t, "[Sub2API] 新工单 #7：API 报 429", title)
	require.Contains(t, desp, "**内容**：-")
	require.NotContains(t, desp, "[查看工单]")
}
