package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newApprovalNotifyServiceForTest(baseURL string, values map[string]string, repo AdminApprovalRepository) *ApprovalNotifyService {
	return &ApprovalNotifyService{
		settingRepo: &groupStatusNotifySettingRepo{values: values},
		repo:        repo,
		client:      newServerChanClientForTest(baseURL),
	}
}

func sampleApprovalForNotify() *AdminApprovalRequest {
	return &AdminApprovalRequest{
		ID:              42,
		Status:          ApprovalStatusPending,
		Action:          "admin.users.balance.create",
		Method:          "POST",
		RequestPath:     "/api/v1/admin/users/7/balance",
		TargetSummary:   "target@example.com",
		RequesterUserID: 2,
		RequesterEmail:  "ops@example.com",
		CreatedAt:       time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC),
		ExpiresAt:       time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC),
	}
}

func TestApprovalNotifyService_DeliversWithLinkAndMarksNotified(t *testing.T) {
	ts := newServerChanTestServer(t)
	repo := newApprovalRepoStub()
	created, err := repo.Create(context.Background(), sampleApprovalForNotify())
	require.NoError(t, err)

	svc := newApprovalNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyApprovalNotifyServerChanEnabled:    "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
		SettingKeySiteName:                           "MySite",
		SettingKeyFrontendURL:                        "https://console.example.com/",
	}, repo)

	require.NoError(t, svc.deliver(created, "http://ignored.local"))
	require.Equal(t, 1, ts.count())
	got := ts.request(0)
	require.Equal(t, "/send/sk_test.send", got.Path)
	require.Equal(t, "[MySite] 新的审批申请 #1：调整余额", got.Title)
	require.Contains(t, got.Desp, "ops@example.com")
	require.Contains(t, got.Desp, "target@example.com")
	require.Contains(t, got.Desp, "POST /api/v1/admin/users/7/balance")
	require.Contains(t, got.Desp, "[前往审批](https://console.example.com/admin/approvals?id=1)")

	stored, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.NotifiedAt, "推送成功后回写 notified_at")
}

func TestApprovalNotifyService_LinkFallsBackToRequestOrigin(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newApprovalNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyApprovalNotifyServerChanEnabled:    "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
	}, nil)

	require.NoError(t, svc.deliver(sampleApprovalForNotify(), "https://origin.example.com"))
	require.Equal(t, 1, ts.count())
	require.Contains(t, ts.request(0).Desp, "https://origin.example.com/admin/approvals?id=42")
	require.True(t, strings.HasPrefix(ts.request(0).Title, "[Sub2API] "), "site name falls back to default")
}

func TestApprovalNotifyService_SkipsWhenDisabledOrMisconfigured(t *testing.T) {
	ts := newServerChanTestServer(t)

	disabled := newApprovalNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyApprovalNotifyServerChanEnabled:    "false",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
	}, nil)
	require.NoError(t, disabled.deliver(sampleApprovalForNotify(), ""))

	missingKey := newApprovalNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyApprovalNotifyServerChanEnabled: "true",
		SettingKeyGroupStatusNotifyServerChanUID:  "uid-1",
	}, nil)
	require.NoError(t, missingKey.deliver(sampleApprovalForNotify(), ""))

	badUID := newApprovalNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyApprovalNotifyServerChanEnabled:    "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "evil.host/",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_test",
	}, nil)
	require.NoError(t, badUID.deliver(sampleApprovalForNotify(), ""))

	require.Equal(t, 0, ts.count())

	var nilSvc *ApprovalNotifyService
	nilSvc.NotifyRequested(sampleApprovalForNotify(), "")
}

func TestApprovalNotifyService_RedactsSendKeyInErrors(t *testing.T) {
	ts := newServerChanTestServer(t, serverChanTestResponse{Status: 500, Body: `{"code":500}`})
	svc := newApprovalNotifyServiceForTest(ts.server.URL, map[string]string{
		SettingKeyApprovalNotifyServerChanEnabled:    "true",
		SettingKeyGroupStatusNotifyServerChanUID:     "uid-1",
		SettingKeyGroupStatusNotifyServerChanSendKey: "sk_secret_value",
	}, nil)
	err := svc.deliver(sampleApprovalForNotify(), "")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "sk_secret_value")
	require.Equal(t, groupStatusNotifyMaxAttempts, ts.count(), "带重试")
}

func TestApprovalActionLabelAndLink(t *testing.T) {
	require.Equal(t, "分配订阅", approvalActionLabel("admin.subscriptions.assign.create"))
	require.Equal(t, "admin.unknown.create", approvalActionLabel("admin.unknown.create"))
	require.Equal(t, "写操作", approvalActionLabel(""))
	require.Equal(t, "", approvalPageLink("", "", 1))
	require.Equal(t, "https://a.b/admin/approvals?id=7", approvalPageLink("https://a.b/", "", 7))
	require.Equal(t, "http://c.d/admin/approvals?id=7", approvalPageLink("", "http://c.d", 7))
}
