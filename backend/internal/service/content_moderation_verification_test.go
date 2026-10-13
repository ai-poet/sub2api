package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// fork：两级审计 —— TypeSafe 初审命中后由 OpenAI 复核，复核确认才计次、警告、拦截。

type twoStageModerationFixture struct {
	svc         *ContentModerationService
	repo        *contentModerationTestRepo
	hashCache   *contentModerationTestHashCache
	openAICalls *atomic.Int32
}

func newTwoStageModerationFixture(t *testing.T, typeSafeSexual float64, openAIKeys []string, openAI http.HandlerFunc) *twoStageModerationFixture {
	t.Helper()
	typeSafe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/systemone", r.URL.Path)
		answers := map[string]any{}
		for _, category := range ContentModerationCategories() {
			answers[category] = map[string]any{"type": "noul", "noul": 0.01}
		}
		answers["sexual"] = map[string]any{"type": "noul", "noul": typeSafeSexual}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"model": "jev-test", "answers": answers}))
	}))
	t.Cleanup(typeSafe.Close)
	calls := &atomic.Int32{}
	verifier := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		openAI(w, r)
	}))
	t.Cleanup(verifier.Close)

	cfg := defaultContentModerationConfig()
	cfg.Enabled = true
	cfg.Mode = ContentModerationModePreBlock
	cfg.PreHashCheckEnabled = true
	cfg.BaseURL = verifier.URL
	cfg.APIKeys = openAIKeys
	cfg.RetryCount = 0
	cfg.TypeSafe = moderationEngineDefaults(ContentModerationEngineTypeSafe)
	cfg.TypeSafe.BaseURL = typeSafe.URL
	cfg.TypeSafe.APIKeys = []string{"typesafe-key"}
	cfg.TypeSafe.RetryCount = 0
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)

	repo := &contentModerationTestRepo{}
	hashCache := &contentModerationTestHashCache{}
	svc := NewContentModerationService(
		&contentModerationTestSettingRepo{values: map[string]string{
			SettingKeyRiskControlEnabled:      "true",
			SettingKeyContentModerationConfig: string(raw),
		}},
		repo, hashCache, nil, nil, nil, nil, nil,
	)
	return &twoStageModerationFixture{svc: svc, repo: repo, hashCache: hashCache, openAICalls: calls}
}

func openAIModerationScores(sexual float64) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeChatModerationScores(w, "omni-moderation-test", map[string]float64{"sexual": sexual})
	}
}

func (f *twoStageModerationFixture) check(t *testing.T) *ContentModerationDecision {
	t.Helper()
	decision, err := f.svc.Check(context.Background(), ContentModerationCheckInput{
		UserID:   7,
		Protocol: ContentModerationProtocolOpenAIChat,
		Body:     []byte(`{"messages":[{"role":"user","content":"needs a second opinion"}]}`),
	})
	require.NoError(t, err)
	return decision
}

func TestTwoStageModeration_ConfirmedHitCountsAndBlocks(t *testing.T) {
	f := newTwoStageModerationFixture(t, 0.99, []string{"openai-key"}, openAIModerationScores(0.9))

	decision := f.check(t)

	require.True(t, decision.Blocked)
	require.Equal(t, ContentModerationActionBlock, decision.Action)
	log := requireContentModerationLogCount(t, f.repo, 1)[0]
	require.True(t, log.Flagged)
	require.Equal(t, ContentModerationActionBlock, log.Action)
	require.Equal(t, 1, log.ViolationCount)
	require.Equal(t, ContentModerationEngineTypeSafe, log.EngineMeta.Engine)
	require.Equal(t, "sexual", log.HighestCategory)
	v := log.EngineMeta.Verification
	require.NotNil(t, v)
	require.Equal(t, ContentModerationVerificationConfirmed, v.Status)
	require.Equal(t, ContentModerationEngineOpenAI, v.Engine)
	require.Equal(t, "omni-moderation-test", v.Model)
	require.Equal(t, "sexual", v.HighestCategory)
	requireRecordedHashCount(t, f.hashCache, 1)
	require.Equal(t, int32(1), f.openAICalls.Load())
}

func TestTwoStageModeration_ClearedHitIsOnlyRecorded(t *testing.T) {
	f := newTwoStageModerationFixture(t, 0.99, []string{"openai-key"}, openAIModerationScores(0.01))

	decision := f.check(t)

	require.True(t, decision.Allowed)
	require.False(t, decision.Blocked)
	require.False(t, decision.Flagged)
	log := requireContentModerationLogCount(t, f.repo, 1)[0]
	require.False(t, log.Flagged)
	require.Equal(t, ContentModerationActionUnconfirmed, log.Action)
	require.Zero(t, log.ViolationCount)
	require.Equal(t, "sexual", log.HighestCategory, "the TypeSafe finding stays on the record")
	require.Equal(t, ContentModerationVerificationCleared, log.EngineMeta.Verification.Status)
	require.InDelta(t, 0.01, log.EngineMeta.Verification.CategoryScores["sexual"], 1e-9)
	require.Empty(t, f.hashCache.snapshotRecorded(), "an unconfirmed hit must not poison the hash cache")

	count, err := f.repo.CountFlaggedByUserSince(context.Background(), 7, log.CreatedAt.Add(-1), false)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestTwoStageModeration_VerifierFailureDoesNotCount(t *testing.T) {
	f := newTwoStageModerationFixture(t, 0.99, []string{"openai-key"}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	decision := f.check(t)

	require.False(t, decision.Blocked)
	log := requireContentModerationLogCount(t, f.repo, 1)[0]
	require.False(t, log.Flagged)
	require.Equal(t, ContentModerationActionUnconfirmed, log.Action)
	require.Equal(t, ContentModerationVerificationError, log.EngineMeta.Verification.Status)
	require.Contains(t, log.EngineMeta.Verification.Error, "500")
	require.Empty(t, f.hashCache.snapshotRecorded())
}

func TestTwoStageModeration_NoVerifierKeyDoesNotCount(t *testing.T) {
	f := newTwoStageModerationFixture(t, 0.99, nil, openAIModerationScores(0.9))

	decision := f.check(t)

	require.False(t, decision.Blocked)
	log := requireContentModerationLogCount(t, f.repo, 1)[0]
	require.Equal(t, ContentModerationActionUnconfirmed, log.Action)
	require.Equal(t, ContentModerationVerificationUnavailable, log.EngineMeta.Verification.Status)
	require.Zero(t, f.openAICalls.Load())
}

func TestTwoStageModeration_PrimaryPassSkipsVerifier(t *testing.T) {
	f := newTwoStageModerationFixture(t, 0.01, []string{"openai-key"}, openAIModerationScores(0.9))

	decision := f.check(t)

	require.True(t, decision.Allowed)
	require.False(t, decision.Flagged)
	require.Zero(t, f.openAICalls.Load())
	require.Empty(t, f.repo.snapshotLogs(), "non-hits stay unrecorded unless record_non_hits is on")
}

func TestModerationPipeline_FallsBackToOpenAIWithoutTypeSafeKeys(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.APIKeys = []string{"openai-key"}

	pipeline := cfg.moderationPipeline()
	require.Equal(t, ContentModerationEngineOpenAI, pipeline.Engine)
	require.Equal(t, []string{"openai-key"}, pipeline.apiKeys())
	require.Nil(t, pipeline.verifier)
	require.Empty(t, pipeline.verifierEngine())

	cfg.TypeSafe = moderationEngineDefaults(ContentModerationEngineTypeSafe)
	cfg.TypeSafe.APIKeys = []string{"typesafe-key"}
	pipeline = cfg.moderationPipeline()
	require.Equal(t, ContentModerationEngineTypeSafe, pipeline.Engine)
	require.Equal(t, []string{"typesafe-key"}, pipeline.apiKeys())
	require.Equal(t, ContentModerationEngineOpenAI, pipeline.verifierEngine())
	require.Equal(t, []string{"openai-key"}, pipeline.verifier.apiKeys())

	clone := cloneContentModerationConfig(pipeline)
	clone.verifier.APIKeys[0] = "changed"
	require.Equal(t, []string{"openai-key"}, pipeline.verifier.apiKeys(), "clones must not share the verifier")
}
