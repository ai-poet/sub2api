package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// ─── 测试桩 ───

type ctMemoryRepo struct {
	mu      sync.Mutex
	nextID  int64
	rows    map[string]*ContentTranslation // hash|lang → row
	sources map[string]map[string]string   // namespace → hash → text
}

func newCTMemoryRepo() *ctMemoryRepo {
	return &ctMemoryRepo{rows: map[string]*ContentTranslation{}, sources: map[string]map[string]string{}}
}

func (r *ctMemoryRepo) copyRow(row *ContentTranslation) *ContentTranslation {
	out := *row
	return &out
}

func (r *ctMemoryRepo) ListAll(context.Context) ([]*ContentTranslation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*ContentTranslation, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, r.copyRow(row))
	}
	return out, nil
}

func (r *ctMemoryRepo) Upsert(_ context.Context, entry *ContentTranslation) (*ContentTranslation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := contentTranslationKey(entry.SourceHash, entry.TargetLang)
	if existing, ok := r.rows[key]; ok {
		if existing.Manual {
			return r.copyRow(existing), nil
		}
		existing.TranslatedText = entry.TranslatedText
		existing.Model = entry.Model
		existing.SourceText = entry.SourceText
		return r.copyRow(existing), nil
	}
	r.nextID++
	row := *entry
	row.ID = r.nextID
	r.rows[key] = &row
	return r.copyRow(&row), nil
}

func (r *ctMemoryRepo) find(id int64) (string, *ContentTranslation) {
	for key, row := range r.rows {
		if row.ID == id {
			return key, row
		}
	}
	return "", nil
}

func (r *ctMemoryRepo) GetByID(_ context.Context, id int64) (*ContentTranslation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, row := r.find(id); row != nil {
		return r.copyRow(row), nil
	}
	return nil, ErrContentTranslationNotFound
}

func (r *ctMemoryRepo) UpdateManual(_ context.Context, id int64, translated string) (*ContentTranslation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, row := r.find(id)
	if row == nil {
		return nil, ErrContentTranslationNotFound
	}
	row.TranslatedText = translated
	row.Manual = true
	return r.copyRow(row), nil
}

func (r *ctMemoryRepo) Delete(_ context.Context, id int64) (*ContentTranslation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key, row := r.find(id)
	if row == nil {
		return nil, ErrContentTranslationNotFound
	}
	delete(r.rows, key)
	return r.copyRow(row), nil
}

func (r *ctMemoryRepo) DeleteMachine(context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for key, row := range r.rows {
		if !row.Manual {
			delete(r.rows, key)
			n++
		}
	}
	return n, nil
}

func (r *ctMemoryRepo) TouchSeen(context.Context, []string, time.Time) error { return nil }

func (r *ctMemoryRepo) List(_ context.Context, filter ContentTranslationFilter) ([]*ContentTranslation, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*ContentTranslation
	for _, row := range r.rows {
		if filter.Lang != "" && row.TargetLang != filter.Lang {
			continue
		}
		out = append(out, r.copyRow(row))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, int64(len(out)), nil
}

func (r *ctMemoryRepo) ReplaceNamespaceSources(_ context.Context, namespace string, sources map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := map[string]string{}
	for k, v := range sources {
		copied[k] = v
	}
	r.sources[namespace] = copied
	return nil
}

func (r *ctMemoryRepo) ListNamespaceSources(context.Context) ([]ContentTranslationSourceText, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ContentTranslationSourceText
	for ns, items := range r.sources {
		for hash, text := range items {
			out = append(out, ContentTranslationSourceText{Namespace: ns, SourceHash: hash, SourceText: text})
		}
	}
	return out, nil
}

func (r *ctMemoryRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.rows)
}

// ctFakeTranslator 记录每次请求的文本，译文是「[lang] 原文」。
type ctFakeTranslator struct {
	mu       sync.Mutex
	calls    int
	texts    []string
	failNext int
	models   []string
}

func (t *ctFakeTranslator) Translate(_ context.Context, creds *contentTranslationCredentials, lang string, texts []string) ([]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls++
	t.models = append(t.models, creds.Model)
	if t.failNext > 0 {
		t.failNext--
		return nil, errors.New("upstream exploded")
	}
	out := make([]string, len(texts))
	for i, text := range texts {
		t.texts = append(t.texts, lang+":"+text)
		out[i] = "[" + lang + "] " + text
	}
	return out, nil
}

func (t *ctFakeTranslator) translatedCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.texts)
}

type ctSettingRepo struct {
	mu   sync.Mutex
	data map[string]string
}

func (r *ctSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.data[key]; ok {
		return &Setting{Key: key, Value: v}, nil
	}
	return nil, ErrSettingNotFound
}

func (r *ctSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	s, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

func (r *ctSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = value
	return nil
}

func (r *ctSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *ctSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *ctSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (r *ctSettingRepo) Delete(context.Context, string) error { return nil }

// ctAPIKeyRepo / ctUserRepo 只实现 GetByID，其余方法不应被调用（嵌入的 nil 接口会直接 panic）。
type ctAPIKeyRepo struct {
	APIKeyRepository
	keys map[int64]*APIKey
}

func (r *ctAPIKeyRepo) GetByID(_ context.Context, id int64) (*APIKey, error) {
	if key, ok := r.keys[id]; ok {
		out := *key
		return &out, nil
	}
	return nil, errors.New("not found")
}

type ctUserRepo struct {
	UserRepository
	users map[int64]*User
}

func (r *ctUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	if user, ok := r.users[id]; ok {
		out := *user
		return &out, nil
	}
	return nil, ErrUserNotFound
}

type ctHarness struct {
	svc        *ContentTranslationService
	repo       *ctMemoryRepo
	settings   *ctSettingRepo
	translator *ctFakeTranslator
	sources    []string
	now        time.Time
}

func newCTHarness(t *testing.T, repo *ctMemoryRepo, sources ...string) *ctHarness {
	t.Helper()
	keyID := int64(7)
	h := &ctHarness{
		repo:       repo,
		settings:   &ctSettingRepo{data: map[string]string{}},
		translator: &ctFakeTranslator{},
		sources:    sources,
		now:        time.Date(2026, 10, 4, 8, 0, 0, 0, time.UTC),
	}
	apiKeys := &ctAPIKeyRepo{keys: map[int64]*APIKey{
		7: {ID: 7, UserID: 1, Key: "sk-admin", Name: "translate", Status: StatusActive},
		8: {ID: 8, UserID: 2, Key: "sk-user", Name: "user", Status: StatusActive},
	}}
	users := &ctUserRepo{users: map[int64]*User{
		1: {ID: 1, Role: RoleAdmin, Status: StatusActive},
		2: {ID: 2, Role: RoleUser, Status: StatusActive},
	}}
	cfg := &config.Config{}
	cfg.Server.Port = 9090
	h.svc = NewContentTranslationService(repo, h.settings, apiKeys, users, nil, nil, nil, nil, cfg)
	h.svc.translator = h.translator
	h.svc.now = func() time.Time { return h.now }
	h.svc.collect = func(context.Context) (contentTranslationSources, error) {
		out := contentTranslationSources{}
		for _, text := range h.sources {
			out.add(text)
		}
		return out, nil
	}
	h.saveConfig(t, ContentTranslationConfig{Enabled: true, APIKeyID: &keyID, Model: "cheap-mini", Languages: []string{"zh", "en", "ja"}})
	return h
}

func (h *ctHarness) saveConfig(t *testing.T, cfg ContentTranslationConfig) {
	t.Helper()
	_, _, err := h.svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)
	h.drainSignal()
}

func (h *ctHarness) drainSignal() bool {
	select {
	case <-h.svc.trigger:
		return true
	default:
		return false
	}
}

func (h *ctHarness) sync(t *testing.T, force bool) {
	t.Helper()
	require.NoError(t, h.svc.Sync(context.Background(), force))
}

// ─── 工具函数 ───

func TestNormalizeContentLang(t *testing.T) {
	cases := map[string]string{
		"zh": "zh", "zh-CN": "zh", "ZH-tw": "zh", "cn": "zh",
		"en": "en", "en-US,en;q=0.9": "en",
		"ja": "ja", "ja-JP": "ja",
		"": "", "fr": "", "de-DE": "",
	}
	for in, want := range cases {
		require.Equal(t, want, NormalizeContentLang(in), in)
	}
}

func TestDetectContentScript(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"Claude Max 专用分组", "zh"},
		{"仅限 GPT-5 使用", "zh"},
		{"Fast group for Codex", "en"},
		{"高速なグループです", "ja"},
		{"日本語のテキスト", "ja"},
		{"GPT-5", "en"},
		{"12345 / 67", ""},
		{"Скоро", "other"},
		// URL / 代码 / 占位符不影响判断
		{"详见 https://example.com/docs", "zh"},
		{"```go\nfunc main() {}\n```\n这是示例", "zh"},
		{"{name} 你好", "zh"},
		{"https://example.com 中文", "zh"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, detectContentScript(c.text), c.text)
	}
}

func TestContentNeedsTranslation(t *testing.T) {
	require.False(t, contentNeedsTranslation("中文描述", "zh"))
	require.True(t, contentNeedsTranslation("中文描述", "en"))
	require.True(t, contentNeedsTranslation("中文描述", "ja"))
	require.True(t, contentNeedsTranslation("English text", "zh"))
	require.False(t, contentNeedsTranslation("English text", "en"))
	require.False(t, contentNeedsTranslation("100%", "en"))
}

func TestNormalizeContentSourceAndHash(t *testing.T) {
	a := normalizeContentSource("  第一行\r\n第二行  ")
	b := normalizeContentSource("第一行\n第二行")
	require.Equal(t, a, b)
	require.Equal(t, contentSourceHash(a), contentSourceHash(b))
	require.Len(t, contentSourceHash(a), 64)
}

func TestNormalizeContentTranslationConfig(t *testing.T) {
	cfg := ContentTranslationConfig{Model: " m ", BaseURL: "https://gw.example.com/v1/", Languages: []string{"ja", "zh", "ja"}}
	require.NoError(t, normalizeContentTranslationConfig(&cfg))
	require.Equal(t, "m", cfg.Model)
	require.Equal(t, "https://gw.example.com", cfg.BaseURL)
	require.Equal(t, []string{"zh", "ja"}, cfg.Languages)

	require.ErrorIs(t, normalizeContentTranslationConfig(&ContentTranslationConfig{BaseURL: "ftp://x"}), ErrContentTranslationBaseURLInvalid)
	require.ErrorIs(t, normalizeContentTranslationConfig(&ContentTranslationConfig{Languages: []string{"fr"}}), ErrContentTranslationLanguagesInvalid)
	require.ErrorIs(t, normalizeContentTranslationConfig(&ContentTranslationConfig{Enabled: true}), ErrContentTranslationModelRequired)
	require.ErrorIs(t, normalizeContentTranslationConfig(&ContentTranslationConfig{Enabled: true, Model: "m"}), ErrContentTranslationAPIKeyRequired)
}

func TestContentTranslationLoopbackBase(t *testing.T) {
	cfg := &config.Config{}
	require.Equal(t, "http://127.0.0.1:8080", contentTranslationLoopbackBase(cfg))
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 3000
	require.Equal(t, "http://127.0.0.1:3000", contentTranslationLoopbackBase(cfg))
	cfg.Server.Host = "::1"
	require.Equal(t, "http://[::1]:3000", contentTranslationLoopbackBase(cfg))
}

// ─── 配置与凭证 ───

func TestContentTranslationResolveCredentials(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo())
	ctx := context.Background()

	adminKey := int64(7)
	creds, err := h.svc.resolveCredentials(ctx, ContentTranslationConfig{APIKeyID: &adminKey, Model: "m"})
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:9090/v1/chat/completions", creds.Endpoint)
	require.Equal(t, "sk-admin", creds.APIKey)

	creds, err = h.svc.resolveCredentials(ctx, ContentTranslationConfig{APIKeyID: &adminKey, Model: "m", BaseURL: "https://gw.example.com"})
	require.NoError(t, err)
	require.Equal(t, "https://gw.example.com/v1/chat/completions", creds.Endpoint)

	userKey := int64(8)
	_, err = h.svc.resolveCredentials(ctx, ContentTranslationConfig{APIKeyID: &userKey, Model: "m"})
	require.ErrorIs(t, err, ErrContentTranslationAPIKeyInvalid, "a key owned by a normal user must be refused")

	missing := int64(99)
	_, err = h.svc.resolveCredentials(ctx, ContentTranslationConfig{APIKeyID: &missing, Model: "m"})
	require.ErrorIs(t, err, ErrContentTranslationAPIKeyInvalid)
}

func TestContentTranslationUpdateConfigOnlyStoresKeyID(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo())
	raw := h.settings.data[SettingKeyContentTranslationConfig]
	require.Contains(t, raw, `"api_key_id":7`)
	require.NotContains(t, raw, "sk-admin", "the API key itself must never be copied into settings")

	// 启用时拒绝普通用户的 Key；关闭时 Key 无效也能保存
	userKey := int64(8)
	_, _, err := h.svc.UpdateConfig(context.Background(), ContentTranslationConfig{Enabled: true, APIKeyID: &userKey, Model: "m"})
	require.ErrorIs(t, err, ErrContentTranslationAPIKeyInvalid)
	missing := int64(99)
	_, _, err = h.svc.UpdateConfig(context.Background(), ContentTranslationConfig{Enabled: false, APIKeyID: &missing, Model: "m"})
	require.NoError(t, err)
}

// ─── 缓存规则：文案不变就不再请求模型 ───

func TestContentTranslationSyncTranslatesOnlyMissingPairs(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组", "Fast lane", "12345")
	h.sync(t, false)

	// 中文 → en / ja，英文 → zh / ja，纯数字不翻
	require.ElementsMatch(t, []string{
		"en:专用分组", "ja:专用分组", "zh:Fast lane", "ja:Fast lane",
	}, h.translator.texts)
	require.Equal(t, 4, h.repo.count())

	result, err := h.svc.Lookup(context.Background(), "en-US", []string{"专用分组", " 专用分组 ", "Fast lane", "12345"})
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"专用分组":   "[en] 专用分组",
		" 专用分组 ": "[en] 专用分组",
	}, result.Translations)
	require.False(t, result.Pending)
}

func TestContentTranslationNeverRequestsModelForUnchangedText(t *testing.T) {
	repo := newCTMemoryRepo()
	h := newCTHarness(t, repo, "专用分组", "公告：本周维护")
	h.sync(t, false)
	first := h.translator.translatedCount()
	require.Equal(t, 4, first)

	// 第二轮扫描
	h.sync(t, false)
	require.Equal(t, first, h.translator.translatedCount(), "second scan must not call the model")

	// 换模型：已有译文全部保留
	keyID := int64(7)
	h.saveConfig(t, ContentTranslationConfig{Enabled: true, APIKeyID: &keyID, Model: "another-model", Languages: []string{"zh", "en", "ja"}})
	h.sync(t, false)
	require.Equal(t, first, h.translator.translatedCount(), "changing the model must not re-translate")

	// 关掉日语再打开
	h.saveConfig(t, ContentTranslationConfig{Enabled: true, APIKeyID: &keyID, Model: "another-model", Languages: []string{"zh", "en"}})
	h.sync(t, false)
	h.saveConfig(t, ContentTranslationConfig{Enabled: true, APIKeyID: &keyID, Model: "another-model", Languages: []string{"zh", "en", "ja"}})
	h.sync(t, false)
	require.Equal(t, first, h.translator.translatedCount(), "toggling a language must not re-translate")

	// 文案下线后再上线
	h.sources = []string{"专用分组"}
	h.sync(t, false)
	h.sources = []string{"专用分组", "公告：本周维护"}
	h.sync(t, false)
	require.Equal(t, first, h.translator.translatedCount(), "text that comes back must hit the cache")

	// 改成新写法 → 只翻新写法；再改回旧写法 → 命中
	h.sources = []string{"专用分组（新）", "公告：本周维护"}
	h.sync(t, false)
	require.Equal(t, first+2, h.translator.translatedCount())
	h.sources = []string{"专用分组", "公告：本周维护"}
	h.sync(t, false)
	require.Equal(t, first+2, h.translator.translatedCount(), "reverting to an old wording must hit the cache")

	// 服务重启：新实例从同一个库装载，不调模型
	restarted := newCTHarness(t, repo, "专用分组", "公告：本周维护")
	restarted.sync(t, false)
	require.Equal(t, 0, restarted.translator.translatedCount(), "a restart must not call the model")
	result, err := restarted.svc.Lookup(context.Background(), "ja", []string{"公告：本周维护"})
	require.NoError(t, err)
	require.Equal(t, "[ja] 公告：本周维护", result.Translations["公告：本周维护"])
}

func TestContentTranslationLookupNeverCallsModel(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	for i := 0; i < 5; i++ {
		_, err := h.svc.Lookup(context.Background(), "en", []string{"专用分组", "随便一段没登记过的文字"})
		require.NoError(t, err)
	}
	require.Equal(t, 0, h.translator.calls)
}

func TestContentTranslationManualTranslationIsKept(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	h.sync(t, false)

	items, _, err := h.svc.List(context.Background(), ContentTranslationFilter{Lang: "en"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].InUse)

	_, err = h.svc.UpdateTranslation(context.Background(), items[0].ID, "Dedicated group")
	require.NoError(t, err)

	// 清空机器译文：人工译文保留，其余重新翻译
	deleted, err := h.svc.ClearMachineTranslations(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	before := h.translator.translatedCount()
	h.sync(t, false)
	require.Equal(t, before+1, h.translator.translatedCount(), "only the cleared ja translation is requested again")

	result, err := h.svc.Lookup(context.Background(), "en", []string{"专用分组"})
	require.NoError(t, err)
	require.Equal(t, "Dedicated group", result.Translations["专用分组"])

	_, err = h.svc.UpdateTranslation(context.Background(), items[0].ID, "   ")
	require.ErrorIs(t, err, ErrContentTranslationTextInvalid)
}

func TestContentTranslationDeleteRetranslatesNextRun(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	h.sync(t, false)
	items, _, err := h.svc.List(context.Background(), ContentTranslationFilter{Lang: "ja"})
	require.NoError(t, err)
	require.NoError(t, h.svc.DeleteTranslation(context.Background(), items[0].ID))

	result, err := h.svc.Lookup(context.Background(), "ja", []string{"专用分组"})
	require.NoError(t, err)
	require.Empty(t, result.Translations)
	require.True(t, result.Pending)

	before := h.translator.translatedCount()
	h.sync(t, false)
	require.Equal(t, before+1, h.translator.translatedCount())
}

func TestContentTranslationFailureBackoff(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	// 只开一种目标语言，保证只有一块请求（并发的另一块可能在出错时被取消、不计入退避）
	keyID := int64(7)
	h.saveConfig(t, ContentTranslationConfig{Enabled: true, APIKeyID: &keyID, Model: "m", Languages: []string{"en"}})
	h.translator.failNext = 100
	err := h.svc.Sync(context.Background(), false)
	require.Error(t, err)
	calls := h.translator.calls
	require.Positive(t, calls)
	require.Contains(t, h.svc.Status(context.Background()).LastError, "upstream exploded")

	// 退避中：普通扫描不再请求，但错误仍留给后台看
	h.sync(t, false)
	require.Equal(t, calls, h.translator.calls)
	require.Contains(t, h.svc.Status(context.Background()).LastError, "upstream exploded")
	result, err := h.svc.Lookup(context.Background(), "en", []string{"专用分组"})
	require.NoError(t, err)
	require.False(t, result.Pending, "pairs in backoff are not reported as pending")

	// 退避期过后再试
	h.translator.failNext = 0
	h.now = h.now.Add(contentTranslationBackoffBase + time.Second)
	h.sync(t, false)
	require.Greater(t, h.translator.calls, calls)
	require.Equal(t, 1, h.repo.count())
	require.Empty(t, h.svc.Status(context.Background()).LastError)
}

func TestContentTranslationCancelledChunksAreNotBackedOff(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	h.translator.failNext = 100
	require.Error(t, h.svc.Sync(context.Background(), false))
	// 每一轮最多让「还没失败过」的块各试一次，几轮之内全部进入退避，之后不再请求
	for i := 0; i < 3; i++ {
		_ = h.svc.Sync(context.Background(), false)
	}
	calls := h.translator.calls
	require.LessOrEqual(t, calls, 2, "each pair is tried at most once before backing off")
	h.sync(t, false)
	require.Equal(t, calls, h.translator.calls)
}

func TestContentTranslationForceSyncIgnoresBackoff(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	h.translator.failNext = 100
	require.Error(t, h.svc.Sync(context.Background(), false))
	calls := h.translator.calls

	h.translator.failNext = 0
	h.sync(t, true)
	require.Greater(t, h.translator.calls, calls)
	require.Equal(t, 2, h.repo.count())
}

func TestContentTranslationBackoffGrowsAndCaps(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo())
	chunk := contentTranslationChunk{lang: "en", jobs: []contentTranslationJob{{hash: "h", text: "中文"}}}
	var last time.Duration
	for i := 0; i < 12; i++ {
		h.svc.markFailed(chunk)
		until := h.svc.failures[contentTranslationKey("h", "en")].until
		d := until.Sub(h.now)
		require.GreaterOrEqual(t, d, last)
		require.LessOrEqual(t, d, contentTranslationBackoffMax)
		last = d
	}
	require.Equal(t, contentTranslationBackoffMax, last)
}

func TestContentTranslationLookupUnknownTriggersDebouncedRescan(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	h.sync(t, false)
	h.drainSignal()

	result, err := h.svc.Lookup(context.Background(), "en", []string{"刚改过的描述"})
	require.NoError(t, err)
	require.True(t, result.Pending)
	require.True(t, h.drainSignal(), "unknown text should request a rescan")

	result, err = h.svc.Lookup(context.Background(), "en", []string{"刚改过的描述"})
	require.NoError(t, err)
	require.False(t, result.Pending)
	require.False(t, h.drainSignal(), "rescans are debounced")

	h.now = h.now.Add(contentTranslationRescanInterval + time.Second)
	_, err = h.svc.Lookup(context.Background(), "en", []string{"刚改过的描述"})
	require.NoError(t, err)
	require.True(t, h.drainSignal())

	// 重扫后被收集到，下一次查询就能拿到
	h.sources = append(h.sources, "刚改过的描述")
	h.sync(t, false)
	result, err = h.svc.Lookup(context.Background(), "en", []string{"刚改过的描述"})
	require.NoError(t, err)
	require.Equal(t, "[en] 刚改过的描述", result.Translations["刚改过的描述"])
}

func TestContentTranslationLookupRespectsSwitches(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	h.sync(t, false)
	keyID := int64(7)

	h.saveConfig(t, ContentTranslationConfig{Enabled: true, APIKeyID: &keyID, Model: "m", Languages: []string{"zh", "en"}})
	result, err := h.svc.Lookup(context.Background(), "ja", []string{"专用分组"})
	require.NoError(t, err)
	require.Empty(t, result.Translations, "a disabled language serves nothing")

	h.saveConfig(t, ContentTranslationConfig{Enabled: false, APIKeyID: &keyID, Model: "m", Languages: []string{"zh", "en", "ja"}})
	result, err = h.svc.Lookup(context.Background(), "en", []string{"专用分组"})
	require.NoError(t, err)
	require.Empty(t, result.Translations, "the master switch turns lookups off")

	_, err = h.svc.Lookup(context.Background(), "fr", []string{"x"})
	require.ErrorIs(t, err, ErrContentTranslationLangInvalid)
	_, err = h.svc.Lookup(context.Background(), "en", make([]string, ContentTranslationMaxLookupTexts+1))
	require.ErrorIs(t, err, ErrContentTranslationTooManyTexts)
}

func TestContentTranslationDisabledSyncDoesNotCallModel(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组")
	keyID := int64(7)
	h.saveConfig(t, ContentTranslationConfig{Enabled: false, APIKeyID: &keyID, Model: "m", Languages: []string{"zh", "en", "ja"}})
	h.sync(t, false)
	require.Equal(t, 0, h.translator.calls)
	require.Equal(t, 1, h.svc.Status(context.Background()).Sources)
}

func TestContentTranslationRegisterExternalSources(t *testing.T) {
	repo := newCTMemoryRepo()
	h := newCTHarness(t, repo)
	// 用真实的收集逻辑（没有分组 / 公告等仓储，只剩外部登记）
	h.svc.collect = h.svc.collectContentTranslationSources

	count, err := h.svc.RegisterExternalSources(context.Background(), ContentTranslationNamespacePay, []string{"月度套餐", " 月度套餐 ", "", "Unlimited"})
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.True(t, h.drainSignal(), "registration should trigger a scan")

	h.sync(t, false)
	require.ElementsMatch(t, []string{"en:月度套餐", "ja:月度套餐", "zh:Unlimited", "ja:Unlimited"}, h.translator.texts)

	_, err = h.svc.RegisterExternalSources(context.Background(), ContentTranslationNamespacePay, make([]string, ContentTranslationMaxExternalSources+1))
	require.ErrorIs(t, err, ErrContentTranslationTooManyTexts)
}

func TestContentTranslationStatusCountsPending(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo(), "专用分组", "Fast lane")
	require.NoError(t, h.svc.Sync(context.Background(), false))
	status := h.svc.Status(context.Background())
	require.True(t, status.Enabled)
	require.Equal(t, 2, status.Sources)
	require.Equal(t, 4, status.Translated)
	require.Equal(t, 0, status.Pending)
	require.NotNil(t, status.LastRunAt)

	h.sources = append(h.sources, "新增的描述")
	h.translator.failNext = 100
	_ = h.svc.Sync(context.Background(), false)
	status = h.svc.Status(context.Background())
	require.Equal(t, 2, status.Pending)
	require.NotEmpty(t, status.LastError)
}

func TestContentTranslationPlanChunksSplitsLongAndBatches(t *testing.T) {
	h := newCTHarness(t, newCTMemoryRepo())
	sources := contentTranslationSources{}
	long := strings.Repeat("长", contentTranslationLongTextRunes+1)
	sources.add(long)
	for i := 0; i < contentTranslationBatchItems+5; i++ {
		sources.add("短文本" + strings.Repeat("字", i+1))
	}
	cfg := ContentTranslationConfig{Enabled: true, Languages: []string{"en"}}
	chunks := h.svc.planChunks(sources, cfg, false)
	var singles, total int
	for _, chunk := range chunks {
		require.LessOrEqual(t, len(chunk.jobs), contentTranslationBatchItems)
		if len(chunk.jobs) == 1 && chunk.jobs[0].text == long {
			singles++
		}
		total += len(chunk.jobs)
	}
	require.Equal(t, 1, singles)
	require.Equal(t, contentTranslationBatchItems+6, total)
}

func TestUserVisibleCustomMenuLabels(t *testing.T) {
	labels := userVisibleCustomMenuLabels(`[{"label":"文档","visibility":"user"},{"label":"内部","visibility":"admin"},{"label":"帮助"}]`)
	require.Equal(t, []string{"文档", "帮助"}, labels)
	require.Nil(t, userVisibleCustomMenuLabels("not json"))
}
