package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fork：未勾选分组仅审计

func TestContentModerationGroupAuditOnlyScope(t *testing.T) {
	selected, other := int64(7), int64(8)
	cfg := defaultContentModerationConfig()
	cfg.AllGroups, cfg.GroupIDs = false, []int64{selected}

	require.False(t, cfg.groupAuditOnly(&other), "switch off keeps unselected groups out of the audit")

	cfg.UnselectedGroupsAuditOnly = true
	require.False(t, cfg.groupAuditOnly(&selected), "selected groups keep full enforcement")
	require.True(t, cfg.groupAuditOnly(&other))
	require.True(t, cfg.groupAuditOnly(nil), "requests without a group are outside the selection too")

	cfg.AllGroups = true
	require.False(t, cfg.groupAuditOnly(&other), "all groups leaves nothing unselected")
}

func newGroupAuditOnlyTestService(t *testing.T, cfg *ContentModerationConfig, hashes *contentModerationTestHashCache, client *http.Client) (*ContentModerationService, *banCountArgsTestRepo) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	repo := &banCountArgsTestRepo{}
	return &ContentModerationService{
		settingRepo: &contentModerationTestSettingRepo{values: map[string]string{
			SettingKeyRiskControlEnabled: "true", SettingKeyContentModerationConfig: string(raw),
		}},
		repo: repo, hashCache: hashes, emailService: &EmailService{},
		httpClient: client, asyncQueue: make(chan contentModerationTask, 4), keyHealth: make(map[string]*contentModerationKeyHealth),
	}, repo
}

func TestContentModerationGroupAuditOnlyAuditsWithoutWarnings(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeChatModerationScores(w, "", map[string]float64{"sexual": 0.99})
	}))
	defer server.Close()
	unselected := int64(8)
	for _, path := range []string{"keyword", "hash", "pre_block", "observe"} {
		t.Run(path, func(t *testing.T) {
			calls.Store(0)
			cfg := defaultContentModerationConfig()
			cfg.Enabled, cfg.AutoBanEnabled, cfg.EmailOnHit, cfg.SiteMessageOnHit = true, true, true, true
			cfg.BanThreshold = 1
			cfg.Mode = ContentModerationModePreBlock
			cfg.AllGroups, cfg.GroupIDs, cfg.UnselectedGroupsAuditOnly = false, []int64{7}, true
			cfg.BaseURL, cfg.APIKeys = server.URL, []string{"sk-test"}
			if path == "observe" {
				cfg.Mode = ContentModerationModeObserve
			}
			if path == "keyword" {
				cfg.BlockedKeywords = []string{"blocked prompt"}
			}
			hashes := &contentModerationTestHashCache{hashes: map[string]struct{}{}}
			if path == "hash" {
				cfg.PreHashCheckEnabled = true
				hashes.hashes[(ContentModerationInput{Text: "blocked prompt"}).Hash()] = struct{}{}
			}
			svc, repo := newGroupAuditOnlyTestService(t, cfg, hashes, server.Client())

			decision, err := svc.Check(context.Background(), ContentModerationCheckInput{
				UserID: 12, UserEmail: "user@example.com", GroupID: &unselected, Protocol: ContentModerationProtocolOpenAIChat,
				Body: []byte(`{"messages":[{"role":"user","content":"blocked prompt"}]}`),
			})
			require.NoError(t, err)
			require.True(t, decision.Allowed)
			require.False(t, decision.Blocked)
			require.Zero(t, calls.Load(), "audit-only groups never wait on the audit API")

			var task contentModerationTask
			select {
			case task = <-svc.asyncQueue:
			case <-time.After(time.Second):
				t.Fatal("expected an audit task")
			}
			if task.log != nil {
				svc.persistContentModerationLog(context.Background(), task.config, task.log, task.inputHash, task.recordHash, task.applySideEffects)
			} else {
				require.True(t, task.input.groupAuditOnly)
				delay := 1
				svc.checkSync(context.Background(), task.input, svc.runtimeSnapshot.Load().config, task.content, task.inputHash, &delay, false)
			}
			logs := repo.snapshotLogs()
			require.Len(t, logs, 1)
			require.True(t, logs[0].Flagged)
			require.Equal(t, ContentModerationModeGroupAuditOnly, logs[0].Mode)
			require.Equal(t, ContentModerationActionAllow, logs[0].Action)
			require.False(t, logs[0].AutoBanned)
			require.False(t, logs[0].EmailSent)
			require.Zero(t, logs[0].ViolationCount)
			require.Empty(t, repo.snapshotCountCalls())
			require.Zero(t, svc.preBlockBlocked.Load())
			if path != "hash" {
				require.Empty(t, hashes.hashes)
			}
		})
	}
}

func TestContentModerationGroupAuditOnlyLeavesOtherGroupsAlone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatModerationScores(w, "", map[string]float64{"sexual": 0.99})
	}))
	defer server.Close()
	selected, unselected := int64(7), int64(8)
	body := []byte(`{"messages":[{"role":"user","content":"blocked prompt"}]}`)

	t.Run("switch off skips unselected groups", func(t *testing.T) {
		cfg := defaultContentModerationConfig()
		cfg.Enabled, cfg.Mode = true, ContentModerationModePreBlock
		cfg.AllGroups, cfg.GroupIDs = false, []int64{selected}
		cfg.BaseURL, cfg.APIKeys = server.URL, []string{"sk-test"}
		svc, repo := newGroupAuditOnlyTestService(t, cfg, &contentModerationTestHashCache{hashes: map[string]struct{}{}}, server.Client())

		decision, err := svc.Check(context.Background(), ContentModerationCheckInput{
			UserID: 12, GroupID: &unselected, Protocol: ContentModerationProtocolOpenAIChat, Body: body,
		})
		require.NoError(t, err)
		require.True(t, decision.Allowed)
		require.Empty(t, svc.asyncQueue)
		require.Empty(t, repo.snapshotLogs())
	})

	t.Run("selected groups still block", func(t *testing.T) {
		cfg := defaultContentModerationConfig()
		cfg.Enabled, cfg.Mode = true, ContentModerationModePreBlock
		cfg.AllGroups, cfg.GroupIDs, cfg.UnselectedGroupsAuditOnly = false, []int64{selected}, true
		cfg.BaseURL, cfg.APIKeys = server.URL, []string{"sk-test"}
		svc, _ := newGroupAuditOnlyTestService(t, cfg, &contentModerationTestHashCache{hashes: map[string]struct{}{}}, server.Client())

		decision, err := svc.Check(context.Background(), ContentModerationCheckInput{
			UserID: 12, GroupID: &selected, Protocol: ContentModerationProtocolOpenAIChat, Body: body,
		})
		require.NoError(t, err)
		require.True(t, decision.Blocked)
		require.Equal(t, ContentModerationActionBlock, decision.Action)
	})
}

func TestRecordCyberPolicyEvent_GroupAuditOnlyRecordsWithoutPenalty(t *testing.T) {
	unselected := int64(8)
	repo := &banCountArgsTestRepo{}
	userRepo := &contentModerationTestUserRepo{user: &User{ID: 1, Role: RoleUser, Status: StatusActive}}
	svc := NewContentModerationService(
		&contentModerationTestSettingRepo{values: map[string]string{
			SettingKeyRiskControlEnabled:      "true",
			SettingKeyContentModerationConfig: `{"all_groups":false,"group_ids":[7],"unselected_groups_audit_only":true,"ban_threshold":1}`,
		}},
		repo, nil, nil, userRepo, nil, nil, nil,
	)

	svc.RecordCyberPolicyEvent(context.Background(), CyberPolicyRecordInput{UserID: 1, GroupID: &unselected, Model: "gpt-5"})

	logs := repo.snapshotLogs()
	require.Len(t, logs, 1)
	require.Equal(t, ContentModerationModeGroupAuditOnly, logs[0].Mode)
	require.False(t, logs[0].AutoBanned)
	require.Empty(t, repo.snapshotCountCalls())
	require.Empty(t, userRepo.updated)
}

func TestContentModerationGroupAuditOnlyConfigRoundTrips(t *testing.T) {
	svc := NewContentModerationService(
		&contentModerationTestSettingRepo{values: map[string]string{}},
		&contentModerationTestRepo{}, nil, nil, nil, nil, nil, nil,
	)
	on, allGroups := true, false
	view, err := svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{
		AllGroups: &allGroups, UnselectedGroupsAuditOnly: &on,
	})
	require.NoError(t, err)
	require.True(t, view.UnselectedGroupsAuditOnly)

	view, err = svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.True(t, view.UnselectedGroupsAuditOnly)
}
