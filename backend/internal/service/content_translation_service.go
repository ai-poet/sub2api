package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	contentTranslationFirstRunDelay = 30 * time.Second
	contentTranslationSyncInterval  = 10 * time.Minute
	contentTranslationRunTimeout    = 30 * time.Minute
	// 查询遇到未知文本时触发重扫，两次至少间隔这么久（重扫只读库，有缺口才调模型）
	contentTranslationRescanInterval = 60 * time.Second
	// 每轮最多翻译这么多「原文 × 语言」组合，其余留到下一轮
	contentTranslationMaxPairsPerRun = 500
	contentTranslationWorkers        = 2
	// 失败的组合在内存里指数退避：10 分钟起，最长 24 小时
	contentTranslationBackoffBase = 10 * time.Minute
	contentTranslationBackoffMax  = 24 * time.Hour
	// 配置在内存里缓存一会儿，查询接口不必每次读设置表
	contentTranslationConfigTTL = 30 * time.Second
	// 装载译文失败后多久再试，避免查询接口把数据库打满
	contentTranslationReloadBackoff = 30 * time.Second
	contentTranslationTouchChunk    = 1000
)

const contentTranslationTestSample = "欢迎使用本站！分组「Claude 专用」支持 **Markdown** 与链接 https://example.com 。"

var errContentTranslationNotLoaded = errors.New("content translations are not loaded yet")

// ContentTranslationLookup 是一次查询的结果：translations 的键是调用方原样发来的文本。
type ContentTranslationLookup struct {
	Lang         string
	Translations map[string]string
	Pending      bool
}

// ContentTranslationStatus 是后台展示的运行状态。
type ContentTranslationStatus struct {
	Enabled     bool
	Running     bool
	Sources     int
	Translated  int
	Pending     int
	LastRunAt   *time.Time
	LastError   string
	LastErrorAt *time.Time
}

// ContentTranslationItem 是后台列表里的一行。
type ContentTranslationItem struct {
	*ContentTranslation
	InUse bool
}

// ContentTranslationTestResult 是后台「测试」按钮的结果。
type ContentTranslationTestResult struct {
	Translated string
	LatencyMS  int64
}

type contentTranslationFailure struct {
	count int
	until time.Time
}

type contentTranslationJob struct {
	hash string
	text string
}

type contentTranslationChunk struct {
	lang string
	jobs []contentTranslationJob
}

// ContentTranslationService 维护译文缓存并在后台补翻缺失的译文。
//
// 数据库 content_translations 是唯一的持久缓存，内存里只是它的镜像（启动时装载、每次写库后同步更新），
// 所以查询接口不查库、也永远不调模型。只有「原文 × 语言」在库里没有译文时才会请求模型。
type ContentTranslationService struct {
	repo             ContentTranslationRepository
	settingRepo      SettingRepository
	apiKeyRepo       APIKeyRepository
	userRepo         UserRepository
	groupRepo        GroupRepository
	channelRepo      ChannelRepository
	announcementRepo AnnouncementRepository
	settingService   *SettingService
	cfg              *config.Config

	translator contentTranslator
	collect    func(ctx context.Context) (contentTranslationSources, error)
	now        func() time.Time

	mu         sync.RWMutex
	loadMu     sync.Mutex
	loaded     bool
	loadErrAt  time.Time
	cache      map[string]map[string]string // lang → hash → 译文
	known      contentTranslationSources    // 最近一轮扫描到的文案源
	scanned    bool
	failures   map[string]contentTranslationFailure
	config     *ContentTranslationConfig
	configAt   time.Time
	lastRescan time.Time
	running    bool
	lastRunAt  *time.Time
	lastError  string
	lastErrAt  *time.Time

	syncMu       sync.Mutex
	trigger      chan struct{}
	pendingForce atomic.Bool
	rootCtx      context.Context
	rootCancel   context.CancelFunc
	stopCh       chan struct{}
	wg           sync.WaitGroup
	startOnce    sync.Once
	stopOnce     sync.Once
}

// NewContentTranslationService 构造服务；后台循环由 Start 启动。
func NewContentTranslationService(
	repo ContentTranslationRepository,
	settingRepo SettingRepository,
	apiKeyRepo APIKeyRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	channelRepo ChannelRepository,
	announcementRepo AnnouncementRepository,
	settingService *SettingService,
	cfg *config.Config,
) *ContentTranslationService {
	rootCtx, rootCancel := context.WithCancel(context.Background())
	s := &ContentTranslationService{
		repo:             repo,
		settingRepo:      settingRepo,
		apiKeyRepo:       apiKeyRepo,
		userRepo:         userRepo,
		groupRepo:        groupRepo,
		channelRepo:      channelRepo,
		announcementRepo: announcementRepo,
		settingService:   settingService,
		cfg:              cfg,
		translator:       newChatCompletionsTranslator(),
		now:              time.Now,
		cache:            map[string]map[string]string{},
		failures:         map[string]contentTranslationFailure{},
		trigger:          make(chan struct{}, 1),
		rootCtx:          rootCtx,
		rootCancel:       rootCancel,
		stopCh:           make(chan struct{}),
	}
	s.collect = s.collectContentTranslationSources
	return s
}

// Start 启动后台循环：启动 30 秒后扫描一次，之后每 10 分钟一次，另外响应查询未命中、后台「立即同步」
// 与外部登记触发的扫描。扫描只读数据库，有缺口才调模型。
func (s *ContentTranslationService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.loop()
	})
}

// Stop 停止后台循环并取消进行中的翻译请求。
func (s *ContentTranslationService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.rootCancel()
	})
	s.wg.Wait()
}

func (s *ContentTranslationService) loop() {
	defer s.wg.Done()
	first := time.NewTimer(contentTranslationFirstRunDelay)
	defer first.Stop()
	ticker := time.NewTicker(contentTranslationSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-first.C:
			s.runSync(false)
		case <-ticker.C:
			s.runSync(false)
		case <-s.trigger:
			s.runSync(s.pendingForce.Swap(false))
		}
	}
}

func (s *ContentTranslationService) runSync(force bool) {
	ctx, cancel := context.WithTimeout(s.rootCtx, contentTranslationRunTimeout)
	defer cancel()
	if err := s.Sync(ctx, force); err != nil && !errors.Is(err, context.Canceled) {
		logger.L().Warn("content_translation.sync_failed", zap.Error(err))
	}
}

// signal 请求后台循环尽快扫描一次；force 为真时忽略失败退避（后台「立即同步」）。
func (s *ContentTranslationService) signal(force bool) {
	if force {
		s.pendingForce.Store(true)
	}
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}

// requestRescan 是查询遇到未知文本时的防抖重扫：管理员刚改的分组描述很快就会被收集并翻译，
// 而不需要在上游的管理接口里加钩子。返回是否真的触发了一次。
func (s *ContentTranslationService) requestRescan() bool {
	now := s.now()
	s.mu.Lock()
	if !s.lastRescan.IsZero() && now.Sub(s.lastRescan) < contentTranslationRescanInterval {
		s.mu.Unlock()
		return false
	}
	s.lastRescan = now
	s.mu.Unlock()
	s.signal(false)
	return true
}

// TriggerSync 由后台「立即同步」调用：异步扫描一次，并忽略失败退避。
func (s *ContentTranslationService) TriggerSync() {
	s.signal(true)
}

// ensureLoaded 把数据库里的译文装进内存镜像（只做一次；失败后 30 秒内不重试）。
func (s *ContentTranslationService) ensureLoaded(ctx context.Context) error {
	s.mu.RLock()
	loaded, loadErrAt := s.loaded, s.loadErrAt
	s.mu.RUnlock()
	if loaded {
		return nil
	}
	if !loadErrAt.IsZero() && s.now().Sub(loadErrAt) < contentTranslationReloadBackoff {
		return errContentTranslationNotLoaded
	}
	return s.reload(ctx, false)
}

// reload 从数据库重建内存镜像；force 为假时若已装载则直接返回。
func (s *ContentTranslationService) reload(ctx context.Context, force bool) error {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()
	if !force {
		s.mu.RLock()
		loaded := s.loaded
		s.mu.RUnlock()
		if loaded {
			return nil
		}
	}
	if s.repo == nil {
		return errContentTranslationNotLoaded
	}
	rows, err := s.repo.ListAll(ctx)
	if err != nil {
		s.mu.Lock()
		s.loadErrAt = s.now()
		s.mu.Unlock()
		return fmt.Errorf("load content translations: %w", err)
	}
	cache := map[string]map[string]string{}
	for _, row := range rows {
		if row == nil {
			continue
		}
		byLang := cache[row.TargetLang]
		if byLang == nil {
			byLang = map[string]string{}
			cache[row.TargetLang] = byLang
		}
		byLang[row.SourceHash] = row.TranslatedText
	}
	s.mu.Lock()
	s.cache = cache
	s.loaded = true
	s.loadErrAt = time.Time{}
	s.mu.Unlock()
	return nil
}

func (s *ContentTranslationService) storeCache(row *ContentTranslation) {
	if row == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	byLang := s.cache[row.TargetLang]
	if byLang == nil {
		byLang = map[string]string{}
		s.cache[row.TargetLang] = byLang
	}
	byLang[row.SourceHash] = row.TranslatedText
	delete(s.failures, contentTranslationKey(row.SourceHash, row.TargetLang))
}

func (s *ContentTranslationService) dropCache(row *ContentTranslation) {
	if row == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if byLang := s.cache[row.TargetLang]; byLang != nil {
		delete(byLang, row.SourceHash)
	}
	delete(s.failures, contentTranslationKey(row.SourceHash, row.TargetLang))
}

// currentConfig 返回配置（内存缓存 30 秒）；读失败时按关闭处理。
func (s *ContentTranslationService) currentConfig(ctx context.Context) ContentTranslationConfig {
	now := s.now()
	s.mu.RLock()
	if s.config != nil && now.Sub(s.configAt) < contentTranslationConfigTTL {
		cfg := *s.config
		s.mu.RUnlock()
		return cfg
	}
	s.mu.RUnlock()

	cfg, err := s.loadConfigFromStore(ctx)
	if err != nil {
		logger.L().Warn("content_translation.config_load_failed", zap.Error(err))
		cfg = defaultContentTranslationConfig()
	}
	s.mu.Lock()
	s.config = &cfg
	s.configAt = now
	s.mu.Unlock()
	return cfg
}

func (s *ContentTranslationService) invalidateConfig() {
	s.mu.Lock()
	s.config = nil
	s.mu.Unlock()
}

func (s *ContentTranslationService) inBackoffLocked(hash, lang string, now time.Time) bool {
	failure, ok := s.failures[contentTranslationKey(hash, lang)]
	return ok && now.Before(failure.until)
}

func (s *ContentTranslationService) markFailed(chunk contentTranslationChunk) {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range chunk.jobs {
		key := contentTranslationKey(job.hash, chunk.lang)
		failure := s.failures[key]
		failure.count++
		delay := contentTranslationBackoffBase
		for i := 1; i < failure.count && delay < contentTranslationBackoffMax; i++ {
			delay *= 2
		}
		if delay > contentTranslationBackoffMax {
			delay = contentTranslationBackoffMax
		}
		failure.until = now.Add(delay)
		s.failures[key] = failure
	}
}

// Lookup 只读缓存：返回已有的译文；未知文本触发一次防抖重扫，永远不会直接调模型。
func (s *ContentTranslationService) Lookup(ctx context.Context, rawLang string, texts []string) (*ContentTranslationLookup, error) {
	lang := NormalizeContentLang(rawLang)
	if lang == "" {
		return nil, ErrContentTranslationLangInvalid
	}
	if len(texts) > ContentTranslationMaxLookupTexts {
		return nil, ErrContentTranslationTooManyTexts
	}
	result := &ContentTranslationLookup{Lang: lang, Translations: map[string]string{}}
	cfg := s.currentConfig(ctx)
	if !cfg.Enabled || !cfg.languageEnabled(lang) {
		return result, nil
	}
	if err := s.ensureLoaded(ctx); err != nil {
		if !errors.Is(err, errContentTranslationNotLoaded) {
			logger.L().Warn("content_translation.lookup_load_failed", zap.Error(err))
		}
		return result, nil
	}

	now := s.now()
	unknown := false
	s.mu.RLock()
	byLang := s.cache[lang]
	for _, raw := range texts {
		normalized := normalizeContentSource(raw)
		if !contentSourceUsable(normalized) || !contentNeedsTranslation(normalized, lang) {
			continue
		}
		hash := contentSourceHash(normalized)
		if translated, ok := byLang[hash]; ok {
			result.Translations[raw] = translated
			continue
		}
		if _, ok := s.known[hash]; ok {
			if !s.inBackoffLocked(hash, lang, now) {
				result.Pending = true
			}
			continue
		}
		unknown = true
	}
	s.mu.RUnlock()

	if unknown && s.requestRescan() {
		result.Pending = true
	}
	return result, nil
}

// Sync 扫描一轮：收集文案源 → 标记在用 → 只翻库里还没有译文的组合。没有缺口就零次模型调用。
// 同一时刻只跑一轮；正在跑时直接返回。
func (s *ContentTranslationService) Sync(ctx context.Context, force bool) (err error) {
	if !s.syncMu.TryLock() {
		return nil
	}
	defer s.syncMu.Unlock()

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	// keepError：这一轮什么都没请求、但还有退避中的组合时，保留上次的错误给后台看
	keepError := false
	defer func() {
		now := s.now()
		s.mu.Lock()
		s.running = false
		s.lastRunAt = &now
		if err != nil && !errors.Is(err, context.Canceled) {
			s.lastError = err.Error()
			s.lastErrAt = &now
		} else if err == nil && !keepError {
			s.lastError = ""
			s.lastErrAt = nil
		}
		s.mu.Unlock()
	}()

	if err := s.ensureLoaded(ctx); err != nil {
		return err
	}
	sources, err := s.collect(ctx)
	if err != nil {
		return fmt.Errorf("collect content translation sources: %w", err)
	}
	s.mu.Lock()
	s.known = sources
	s.scanned = true
	s.mu.Unlock()
	s.touchSeen(ctx, sources)

	s.invalidateConfig()
	cfg := s.currentConfig(ctx)
	if !cfg.Enabled {
		return nil
	}

	chunks := s.planChunks(sources, cfg, force)
	if len(chunks) == 0 {
		keepError = len(s.missingJobs(sources, cfg, true, 1)) > 0
		return nil
	}
	creds, err := s.resolveCredentials(ctx, cfg)
	if err != nil {
		return err
	}
	return s.translateChunks(ctx, creds, chunks)
}

func (s *ContentTranslationService) touchSeen(ctx context.Context, sources contentTranslationSources) {
	if s.repo == nil || len(sources) == 0 {
		return
	}
	hashes := make([]string, 0, len(sources))
	for hash := range sources {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	now := s.now()
	for start := 0; start < len(hashes); start += contentTranslationTouchChunk {
		end := start + contentTranslationTouchChunk
		if end > len(hashes) {
			end = len(hashes)
		}
		if err := s.repo.TouchSeen(ctx, hashes[start:end], now); err != nil {
			logger.L().Warn("content_translation.touch_seen_failed", zap.Error(err))
			return
		}
	}
}

// missingJobs 按语言列出库里还没有译文的组合（跳过退避中的，force 时不跳），总数封顶。
func (s *ContentTranslationService) missingJobs(sources contentTranslationSources, cfg ContentTranslationConfig, force bool, limit int) map[string][]contentTranslationJob {
	hashes := make([]string, 0, len(sources))
	for hash := range sources {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)

	now := s.now()
	out := map[string][]contentTranslationJob{}
	total := 0
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, lang := range cfg.Languages {
		byLang := s.cache[lang]
		for _, hash := range hashes {
			if limit > 0 && total >= limit {
				return out
			}
			text := sources[hash]
			if !contentNeedsTranslation(text, lang) {
				continue
			}
			if _, ok := byLang[hash]; ok {
				continue
			}
			if !force && s.inBackoffLocked(hash, lang, now) {
				continue
			}
			out[lang] = append(out[lang], contentTranslationJob{hash: hash, text: text})
			total++
		}
	}
	return out
}

// planChunks 把缺失的组合切成一次请求的大小：长文本单独一块，短文本按条数 / 字数合批。
func (s *ContentTranslationService) planChunks(sources contentTranslationSources, cfg ContentTranslationConfig, force bool) []contentTranslationChunk {
	missing := s.missingJobs(sources, cfg, force, contentTranslationMaxPairsPerRun)
	var chunks []contentTranslationChunk
	for _, lang := range cfg.Languages {
		var current []contentTranslationJob
		currentRunes := 0
		for _, job := range missing[lang] {
			runes := utf8.RuneCountInString(job.text)
			if runes > contentTranslationLongTextRunes {
				chunks = append(chunks, contentTranslationChunk{lang: lang, jobs: []contentTranslationJob{job}})
				continue
			}
			if len(current) >= contentTranslationBatchItems || (len(current) > 0 && currentRunes+runes > contentTranslationBatchRunes) {
				chunks = append(chunks, contentTranslationChunk{lang: lang, jobs: current})
				current, currentRunes = nil, 0
			}
			current = append(current, job)
			currentRunes += runes
		}
		if len(current) > 0 {
			chunks = append(chunks, contentTranslationChunk{lang: lang, jobs: current})
		}
	}
	return chunks
}

// translateChunks 两路并发翻译；每块成功就立即入库。碰到第一个错误就停止派发后续块，
// 出错的块进入退避（被连带取消的块不算失败）。
func (s *ContentTranslationService) translateChunks(ctx context.Context, creds *contentTranslationCredentials, chunks []contentTranslationChunk) error {
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		firstErr error
		errMu    sync.Mutex
		wg       sync.WaitGroup
	)
	setErr := func(err error) {
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
		cancel()
	}
	sem := make(chan struct{}, contentTranslationWorkers)

dispatch:
	for _, chunk := range chunks {
		select {
		case <-workCtx.Done():
			break dispatch
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(chunk contentTranslationChunk) {
			defer wg.Done()
			defer func() { <-sem }()
			texts := make([]string, len(chunk.jobs))
			for i, job := range chunk.jobs {
				texts[i] = job.text
			}
			translated, err := s.translator.Translate(workCtx, creds, chunk.lang, texts)
			if err == nil && len(translated) != len(texts) {
				err = errContentTranslationBatchMismatch
			}
			if err != nil {
				if workCtx.Err() == nil || !errors.Is(err, context.Canceled) {
					s.markFailed(chunk)
					setErr(err)
				}
				return
			}
			for i, job := range chunk.jobs {
				text := strings.TrimSpace(translated[i])
				if text == "" {
					continue
				}
				// 入库用外层 ctx：别的块出错取消时，已经拿到的译文照样落库
				stored, err := s.repo.Upsert(ctx, &ContentTranslation{
					SourceHash:     job.hash,
					TargetLang:     chunk.lang,
					SourceText:     job.text,
					TranslatedText: text,
					Model:          creds.Model,
				})
				if err != nil {
					setErr(fmt.Errorf("save content translation: %w", err))
					return
				}
				s.storeCache(stored)
			}
		}(chunk)
	}
	wg.Wait()

	errMu.Lock()
	defer errMu.Unlock()
	if firstErr == nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return firstErr
}

// Status 返回运行状态；pending 是当前还没有译文的组合数（含退避中的）。
func (s *ContentTranslationService) Status(ctx context.Context) ContentTranslationStatus {
	cfg := s.currentConfig(ctx)
	_ = s.ensureLoaded(ctx)

	s.mu.RLock()
	status := ContentTranslationStatus{
		Enabled:   cfg.Enabled,
		Running:   s.running,
		Sources:   len(s.known),
		LastError: s.lastError,
	}
	for _, byLang := range s.cache {
		status.Translated += len(byLang)
	}
	if s.lastRunAt != nil {
		at := *s.lastRunAt
		status.LastRunAt = &at
	}
	if s.lastErrAt != nil {
		at := *s.lastErrAt
		status.LastErrorAt = &at
	}
	known := s.known
	s.mu.RUnlock()

	if cfg.Enabled {
		for _, jobs := range s.missingJobs(known, cfg, true, 0) {
			status.Pending += len(jobs)
		}
	}
	return status
}

// GetConfig 读取配置（不走缓存），并带上所选 Key 的名称。
func (s *ContentTranslationService) GetConfig(ctx context.Context) (ContentTranslationConfig, string, error) {
	cfg, err := s.loadConfigFromStore(ctx)
	if err != nil {
		return cfg, "", err
	}
	return cfg, s.apiKeyName(ctx, cfg.APIKeyID), nil
}

// UpdateConfig 保存配置并立即生效。启用时校验所选 Key（存在、启用、属于管理员）。
// 换 Key / 换模型都不会让已有译文失效；只清掉失败退避，并触发一轮扫描。
func (s *ContentTranslationService) UpdateConfig(ctx context.Context, in ContentTranslationConfig) (ContentTranslationConfig, string, error) {
	if err := normalizeContentTranslationConfig(&in); err != nil {
		return in, "", err
	}
	// 只在启用时校验 Key：Key 被删掉后管理员仍要能保存「关闭」
	if in.Enabled {
		if _, err := s.resolveCredentials(ctx, in); err != nil {
			return in, "", err
		}
	}
	data, err := json.Marshal(in)
	if err != nil {
		return in, "", fmt.Errorf("marshal content translation config: %w", err)
	}
	if s.settingRepo == nil {
		return in, "", errors.New("setting repository is unavailable")
	}
	if err := s.settingRepo.Set(ctx, SettingKeyContentTranslationConfig, string(data)); err != nil {
		return in, "", fmt.Errorf("save content translation config: %w", err)
	}
	s.mu.Lock()
	s.config = nil
	s.failures = map[string]contentTranslationFailure{}
	s.mu.Unlock()
	if in.Enabled {
		s.signal(false)
	}
	return in, s.apiKeyName(ctx, in.APIKeyID), nil
}

// TestConfig 用给定配置（不必已保存、不必已启用）翻译一段示例文本。
func (s *ContentTranslationService) TestConfig(ctx context.Context, in ContentTranslationConfig) (*ContentTranslationTestResult, error) {
	in.Enabled = false
	if err := normalizeContentTranslationConfig(&in); err != nil {
		return nil, err
	}
	creds, err := s.resolveCredentials(ctx, in)
	if err != nil {
		return nil, err
	}
	lang := ContentLangEN
	for _, l := range in.Languages {
		if l != ContentLangZH {
			lang = l
			break
		}
	}
	start := s.now()
	translated, err := s.translator.Translate(ctx, creds, lang, []string{contentTranslationTestSample})
	if err == nil && len(translated) != 1 {
		err = errContentTranslationBatchMismatch
	}
	if err != nil {
		// 把上游的具体原因带给管理员（只在后台可见），而不是笼统的 500
		return nil, infraerrors.New(http.StatusBadGateway, "CONTENT_TRANSLATION_TEST_FAILED", err.Error())
	}
	return &ContentTranslationTestResult{Translated: translated[0], LatencyMS: s.now().Sub(start).Milliseconds()}, nil
}

// List 分页列出译文，并标出原文当前是否还在文案源集合里。
func (s *ContentTranslationService) List(ctx context.Context, filter ContentTranslationFilter) ([]*ContentTranslationItem, int64, error) {
	if s.repo == nil {
		return nil, 0, errContentTranslationNotLoaded
	}
	if lang := strings.TrimSpace(filter.Lang); lang != "" {
		filter.Lang = NormalizeContentLang(lang)
		if filter.Lang == "" {
			return nil, 0, ErrContentTranslationLangInvalid
		}
	}
	filter.Query = strings.TrimSpace(filter.Query)
	rows, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	s.mu.RLock()
	known, scanned := s.known, s.scanned
	s.mu.RUnlock()
	items := make([]*ContentTranslationItem, 0, len(rows))
	for _, row := range rows {
		inUse := true
		if scanned {
			_, inUse = known[row.SourceHash]
		}
		items = append(items, &ContentTranslationItem{ContentTranslation: row, InUse: inUse})
	}
	return items, total, nil
}

// UpdateTranslation 改写一条译文并标为人工译文，自动翻译不会再覆盖它。
func (s *ContentTranslationService) UpdateTranslation(ctx context.Context, id int64, translated string) (*ContentTranslationItem, error) {
	translated = normalizeContentSource(translated)
	if translated == "" || utf8.RuneCountInString(translated) > ContentTranslationMaxManualRunes {
		return nil, ErrContentTranslationTextInvalid
	}
	row, err := s.repo.UpdateManual(ctx, id, translated)
	if err != nil {
		return nil, err
	}
	if s.isLoaded() {
		s.storeCache(row)
	}
	return s.itemFor(row), nil
}

// DeleteTranslation 删除一条译文；只要原文还在，下一轮扫描会重新翻译。
func (s *ContentTranslationService) DeleteTranslation(ctx context.Context, id int64) error {
	row, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	s.dropCache(row)
	return nil
}

// ClearMachineTranslations 删除全部机器译文（人工译文保留），并重建内存镜像。
func (s *ContentTranslationService) ClearMachineTranslations(ctx context.Context) (int64, error) {
	deleted, err := s.repo.DeleteMachine(ctx)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	s.failures = map[string]contentTranslationFailure{}
	s.mu.Unlock()
	if err := s.reload(ctx, true); err != nil {
		return deleted, err
	}
	return deleted, nil
}

// RegisterExternalSources 用 texts 整体替换某个 namespace（目前只有支付服务）登记的文案，并触发一轮扫描。
// 不再登记的文案只是不再被收集，已有译文不删。
func (s *ContentTranslationService) RegisterExternalSources(ctx context.Context, namespace string, texts []string) (int, error) {
	if len(texts) > ContentTranslationMaxExternalSources {
		return 0, ErrContentTranslationTooManyTexts
	}
	sources := contentTranslationSources{}
	for _, text := range texts {
		sources.add(text)
	}
	if err := s.repo.ReplaceNamespaceSources(ctx, namespace, sources); err != nil {
		return 0, err
	}
	s.signal(false)
	return len(sources), nil
}

func (s *ContentTranslationService) isLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

func (s *ContentTranslationService) itemFor(row *ContentTranslation) *ContentTranslationItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inUse := true
	if s.scanned {
		_, inUse = s.known[row.SourceHash]
	}
	return &ContentTranslationItem{ContentTranslation: row, InUse: inUse}
}
