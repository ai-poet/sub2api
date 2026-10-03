package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 内容审计 → 站内信（fork 本地）。全部用 emailService=nil：站内信不依赖 SMTP。

func newSiteMessageModerationService(t *testing.T, user *User) (*ContentModerationService, *siteMessageTestRepo, *contentModerationTestUserRepo) {
	t.Helper()
	repo := &contentModerationTestRepo{}
	userRepo := &contentModerationTestUserRepo{user: user}
	svc := NewContentModerationService(nil, repo, nil, nil, userRepo, nil, &contentModerationTestAuthCacheInvalidator{}, nil)
	messages := &siteMessageTestRepo{}
	svc.SetSiteMessageNotifier(NewSiteMessageService(messages))
	return svc, messages, userRepo
}

func TestContentModerationSiteMessage_ViolationNotice(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.EmailOnHit = false // 与邮件开关互相独立
	cfg.BanThreshold = 10
	svc, messages, _ := newSiteMessageModerationService(t, &User{ID: 7, Role: RoleUser, Status: StatusActive})

	svc.persistContentModerationLog(context.Background(), cfg, newContentModerationFlaggedLog(7), "", false, true)

	msgs := messages.snapshot()
	require.Len(t, msgs, 1)
	require.Equal(t, int64(7), msgs[0].UserID)
	require.Equal(t, SiteMessageCategorySecurity, msgs[0].Category)
	require.Equal(t, SiteMessageSourceContentModeration, msgs[0].SourceType)
	require.Equal(t, "账户风控提醒 / Risk control notice", msgs[0].Title)
	require.Contains(t, msgs[0].Content, "| 命中类别 | sexual / 0.900 |")
	require.Contains(t, msgs[0].Content, "| Category | sexual / 0.900 |")
	require.Contains(t, msgs[0].Content, "1 次（阈值 10）")
	require.Nil(t, msgs[0].SenderUserID)
}

func TestContentModerationSiteMessage_DisabledToggleSendsNothing(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.SiteMessageOnHit = false
	cfg.BanThreshold = 1
	svc, messages, userRepo := newSiteMessageModerationService(t, &User{ID: 7, Role: RoleUser, Status: StatusActive})

	svc.persistContentModerationLog(context.Background(), cfg, newContentModerationFlaggedLog(7), "", false, true)

	require.Equal(t, StatusDisabled, userRepo.user.Status, "the ban itself still happens")
	require.Empty(t, messages.snapshot(), "toggle off means no violation and no ban notice")
}

func TestContentModerationSiteMessage_AutoBanAddsDisabledNotice(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.BanThreshold = 1
	svc, messages, userRepo := newSiteMessageModerationService(t, &User{ID: 7, Role: RoleUser, Status: StatusActive})

	svc.persistContentModerationLog(context.Background(), cfg, newContentModerationFlaggedLog(7), "", false, true)

	require.Equal(t, StatusDisabled, userRepo.user.Status)
	msgs := messages.snapshot()
	require.Len(t, msgs, 2)
	require.Equal(t, SiteMessageSourceContentModeration, msgs[0].SourceType)
	require.Contains(t, msgs[0].Content, siteMessageBannedBannerZH)
	require.Equal(t, SiteMessageSourceContentModerationBan, msgs[1].SourceType)
	require.Equal(t, "账户已被禁用 / Account disabled", msgs[1].Title)
	require.Contains(t, msgs[1].Content, siteMessageAppealHintZH)
	require.Contains(t, msgs[1].Content, siteMessageAppealHintEN)
}

func TestContentModerationSiteMessage_SkipsLogOnlyUnflaggedAndAnonymous(t *testing.T) {
	cfg := defaultContentModerationConfig()
	svc, messages, _ := newSiteMessageModerationService(t, &User{ID: 7, Role: RoleUser, Status: StatusActive})
	ctx := context.Background()

	logOnly := newContentModerationFlaggedLog(7)
	logOnly.Mode = ContentModerationModeRiskControlLogOnly
	svc.persistContentModerationLog(ctx, cfg, logOnly, "", false, true)

	unflagged := newContentModerationFlaggedLog(7)
	unflagged.Flagged = false
	svc.persistContentModerationLog(ctx, cfg, unflagged, "", false, true)

	anonymous := newContentModerationFlaggedLog(7)
	anonymous.UserID = nil
	anonymous.UserEmail = "someone@example.com"
	require.NotPanics(t, func() { svc.persistContentModerationLog(ctx, cfg, anonymous, "", false, true) })

	svc.persistContentModerationLog(ctx, cfg, newContentModerationFlaggedLog(7), "", false, false)

	require.Empty(t, messages.snapshot())
}

type failingSiteMessageNotifier struct{ calls int }

func (n *failingSiteMessageNotifier) DeliverSystemMessage(context.Context, SystemSiteMessageInput) error {
	n.calls++
	return errors.New("db down")
}

func TestContentModerationSiteMessage_NotifierFailureDoesNotBreakModeration(t *testing.T) {
	cfg := defaultContentModerationConfig()
	repo := &contentModerationTestRepo{}
	svc := NewContentModerationService(nil, repo, nil, nil, &contentModerationTestUserRepo{user: &User{ID: 7, Status: StatusActive}}, nil, nil, nil)
	notifier := &failingSiteMessageNotifier{}
	svc.SetSiteMessageNotifier(notifier)

	svc.persistContentModerationLog(context.Background(), cfg, newContentModerationFlaggedLog(7), "", false, true)

	require.Equal(t, 1, notifier.calls)
	requireContentModerationLogCount(t, repo, 1)
}

func TestContentModerationSiteMessage_CyberPolicy(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		logOnly   bool
		wantTypes []string
	}{
		{name: "notice", config: `{"ban_threshold":100}`, wantTypes: []string{SiteMessageSourceCyberPolicy}},
		{name: "notice and ban", config: `{"ban_threshold":1}`, wantTypes: []string{SiteMessageSourceCyberPolicy, SiteMessageSourceCyberPolicyBan}},
		{name: "log only", config: `{"ban_threshold":1}`, logOnly: true},
		{name: "toggle off", config: `{"ban_threshold":100,"site_message_on_hit":false}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &contentModerationTestUserRepo{user: &User{ID: 1, Role: RoleUser, Status: StatusActive}}
			svc := NewContentModerationService(
				&contentModerationTestSettingRepo{values: map[string]string{
					SettingKeyRiskControlEnabled:      "true",
					SettingKeyContentModerationConfig: tt.config,
				}},
				&cyberOrderingTestRepo{}, nil, nil, userRepo, nil, nil, nil,
			)
			messages := &siteMessageTestRepo{}
			svc.SetSiteMessageNotifier(NewSiteMessageService(messages))

			svc.RecordCyberPolicyEvent(context.Background(), CyberPolicyRecordInput{
				UserID:          1,
				UserEmail:       "u@example.com",
				Model:           "gpt-5",
				Endpoint:        "/v1/responses",
				UpstreamMessage: "blocked by policy",
				UpstreamBody:    `{"secret":"raw body must not leak"}`,
				LogOnly:         tt.logOnly,
			})

			msgs := messages.snapshot()
			var types []string
			for _, m := range msgs {
				types = append(types, m.SourceType)
				require.Equal(t, "log:1", m.SourceID, "cyber path persists the log first")
			}
			require.Equal(t, tt.wantTypes, types)
			if len(msgs) > 0 {
				require.Contains(t, msgs[0].Content, "blocked by policy")
				require.NotContains(t, msgs[0].Content, "raw body must not leak", "only the first line of the upstream error is shown")
			}
		})
	}
}

func TestContentModerationConfig_SiteMessageOnHitDefaultsAndUpdates(t *testing.T) {
	// 旧配置 JSON 没有这个字段：叠加到默认值上，保持开启
	cfg, err := parseContentModerationConfig(`{"email_on_hit":false}`)
	require.NoError(t, err)
	require.True(t, cfg.SiteMessageOnHit)
	require.False(t, cfg.EmailOnHit)

	repo := &contentModerationTestSettingRepo{values: map[string]string{}}
	svc := NewContentModerationService(repo, nil, nil, nil, nil, nil, nil, nil)
	off := false
	view, err := svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{SiteMessageOnHit: &off})
	require.NoError(t, err)
	require.False(t, view.SiteMessageOnHit)

	var saved map[string]any
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyContentModerationConfig]), &saved))
	require.Equal(t, false, saved["site_message_on_hit"])

	view, err = svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.False(t, view.SiteMessageOnHit)
}

func TestContentModerationSiteMessage_RenderEscapesUntrustedValues(t *testing.T) {
	cfg := defaultContentModerationConfig()
	log := newContentModerationFlaggedLog(7)
	log.GroupName = "evil | [click](https://x.example) ![img](https://x.example/a.png)"
	log.RequestID = "req`|*"
	log.Model = "gpt<script>"

	_, content := buildContentModerationViolationSiteMessage(log, cfg)
	require.NotContains(t, content, "[click](")
	require.NotContains(t, content, "![img](")
	require.Contains(t, content, `evil \| \[click\]\(https://x\.example\)`)
	require.Contains(t, content, "req\\`\\|\\*")
	require.Contains(t, content, "\n\n---\n\n", "zh and en halves are separated")

	log.Error = "contains ``` fence\nsecond line"
	_, cyber := buildCyberPolicySiteMessage(log)
	require.Contains(t, cyber, "````text\ncontains ``` fence\n````")
	require.NotContains(t, cyber, "second line")
	require.Contains(t, cyber, `gpt\<script\>`)
	require.True(t, strings.HasPrefix(cyber, "您的请求被网络安全策略"))
}
