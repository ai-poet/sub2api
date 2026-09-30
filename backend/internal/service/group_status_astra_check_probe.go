package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// meow 指纹验证的执行（本 fork 自有功能）。
//
// 一次运行依次检测分组配置的每个预期模型：选一个与平台一致的账号 → 按该模型所在基准包的档位跑一批固定短答题 →
// 归一化计数 → 本地判定 → 落库 → 该模型的稳定结论切换时推送。与存活探测互不影响。

const groupStatusAstraCheckLogComponent = "service.group_status_astra_check"

// astraSample 是一个任务的最终结果。
type astraSample struct {
	CellID          string
	Category        string
	Answer          string
	HTTPCode        *int
	LatencyMS       int64
	Usage           openAIProbeUsage
	Completed       bool // 拿到 2xx 响应（答案可能无效）
	TransportFailed bool // 最后一次尝试仍是传输 / 非 2xx 错误
	ErrDetail       string
	Attempts        int
}

// astraResponse 是一次请求的结果（两种通道统一成同一形状）；Invalid 非空表示答案不参与判定的原因。
type astraResponse struct {
	Text    string
	Usage   openAIProbeUsage
	Invalid string
}

func (s *GroupStatusProbeService) astraBenchmarks() (*AstraBenchmarkRegistry, error) {
	if s != nil && s.astraBenchmarkSource != nil {
		return s.astraBenchmarkSource.Registry()
	}
	return LoadEmbeddedAstraBenchmarks()
}

// SetAstraBenchmarkProvider 替换基准来源（测试用）。
func (s *GroupStatusProbeService) SetAstraBenchmarkProvider(p astraBenchmarkProvider) {
	if s == nil {
		return
	}
	s.astraBenchmarkSource = p
}

func (s *GroupStatusProbeService) astraCheckConcurrency() int {
	if s != nil && s.astraConcurrency > 0 {
		return s.astraConcurrency
	}
	return groupStatusAstraCheckDefaultConcurrency
}

// IsAstraCheckRunning 报告该分组是否有正在进行的指纹验证。
func (s *GroupStatusProbeService) IsAstraCheckRunning(groupID int64) bool {
	if s == nil {
		return false
	}
	_, ok := s.astraRunning.Load(groupID)
	return ok
}

func (s *GroupStatusProbeService) markAstraCheckRunning(groupID int64) bool {
	_, loaded := s.astraRunning.LoadOrStore(groupID, time.Now())
	return !loaded
}

func (s *GroupStatusProbeService) clearAstraCheckRunning(groupID int64) {
	s.astraRunning.Delete(groupID)
}

// StartAstraCheckAsync 在后台启动一次验证（管理端「立即检测」用）：expectedModel 为空时检测分组配置的全部模型，
// 否则只检测该模型。分组、平台与模型在这里同步校验，错误直接返回；同一分组运行中则报错。
func (s *GroupStatusProbeService) StartAstraCheckAsync(groupID int64, expectedModel string) error {
	if s == nil {
		return errors.New("group status probe service is not configured")
	}
	if s.IsAstraCheckRunning(groupID) {
		return ErrGroupStatusAstraCheckRunning
	}
	loadCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	group, cfg, err := s.ensureProbeTarget(loadCtx, groupID)
	cancel()
	if err != nil {
		return err
	}
	if !astraCheckSupportsPlatform(group.Platform) {
		return ErrGroupStatusAstraCheckUnsupported
	}
	models, err := astraCheckModelsToRun(group, cfg, expectedModel)
	if err != nil {
		return err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), astraCheckRunBudget(len(models)))
		defer cancel()
		if _, err := s.probeAstraCheck(ctx, group, cfg, models); err != nil {
			logger.LegacyPrintf(groupStatusAstraCheckLogComponent, "[AstraCheck] group=%d manual run failed: %v", groupID, err)
		}
	}()
	return nil
}

// ProbeAstraCheckGroupNow 立即验证（同步）；expectedModel 为空时检测全部配置的模型。不要求分组已开启 astra_check_enabled。
func (s *GroupStatusProbeService) ProbeAstraCheckGroupNow(ctx context.Context, groupID int64, expectedModel string) ([]*GroupStatusAstraCheckExecution, error) {
	group, cfg, err := s.ensureProbeTarget(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !astraCheckSupportsPlatform(group.Platform) {
		return nil, ErrGroupStatusAstraCheckUnsupported
	}
	models, err := astraCheckModelsToRun(group, cfg, expectedModel)
	if err != nil {
		return nil, err
	}
	return s.probeAstraCheck(ctx, group, cfg, models)
}

// ProbeAstraCheckWithConfig 供定时 runner 调用：只检测距上次检测已满间隔（或从未检测过）的模型。
func (s *GroupStatusProbeService) ProbeAstraCheckWithConfig(ctx context.Context, cfg *GroupStatusConfig) ([]*GroupStatusAstraCheckExecution, error) {
	if cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	group, err := s.groupRepo.GetByID(ctx, cfg.GroupID)
	if err != nil {
		return nil, err
	}
	if !astraCheckSupportsPlatform(group.Platform) {
		return nil, ErrGroupStatusAstraCheckUnsupported
	}
	models, err := astraCheckModelsToRun(group, cfg, "")
	if err != nil {
		return nil, err
	}
	due, err := s.dueAstraCheckModels(ctx, group.ID, cfg, models, time.Now())
	if err != nil {
		return nil, err
	}
	if len(due) == 0 {
		return nil, nil
	}
	return s.probeAstraCheck(ctx, group, cfg, due)
}

// astraCheckModelsToRun 返回本次要检测的模型：配置为空时回落到本平台默认目标；指定了模型时它必须在配置里。
func astraCheckModelsToRun(group *Group, cfg *GroupStatusConfig, expectedModel string) ([]AstraCheckModelConfig, error) {
	models := normalizeAstraCheckModels(cfg.AstraCheckModels)
	if len(models) == 0 && group != nil {
		models = astraCheckDefaultModels(group.Platform)
	}
	expectedModel = strings.TrimSpace(expectedModel)
	if expectedModel == "" {
		return models, nil
	}
	for _, m := range models {
		if m.ExpectedModel == expectedModel {
			return []AstraCheckModelConfig{m}, nil
		}
	}
	return nil, ErrGroupStatusAstraCheckTargetInvalid
}

// dueAstraCheckModels 过滤出到期的模型：没有状态、没检测过，或距上次检测已满间隔。
func (s *GroupStatusProbeService) dueAstraCheckModels(ctx context.Context, groupID int64, cfg *GroupStatusConfig, models []AstraCheckModelConfig, now time.Time) ([]AstraCheckModelConfig, error) {
	states, err := s.repo.ListAstraCheckStates(ctx, []int64{groupID})
	if err != nil {
		return nil, err
	}
	checkedAt := make(map[string]time.Time, len(states))
	for _, state := range states {
		if state.GroupID == groupID && state.CheckedAt != nil {
			checkedAt[state.ExpectedModel] = *state.CheckedAt
		}
	}
	interval := time.Duration(cfg.AstraCheckIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = groupStatusAstraCheckDefaultIntervalSecond * time.Second
	}
	due := make([]AstraCheckModelConfig, 0, len(models))
	for _, m := range models {
		last, ok := checkedAt[m.ExpectedModel]
		if !ok || !now.Before(last.Add(interval)) {
			due = append(due, m)
		}
	}
	return due, nil
}

// astraCheckRunBudget 是一次手动运行的总预算：每个模型一份，最多一小时。
func astraCheckRunBudget(models int) time.Duration {
	if models < 1 {
		models = 1
	}
	budget := time.Duration(models) * groupStatusAstraCheckRunBudget
	if budget > time.Hour {
		budget = time.Hour
	}
	return budget
}

// probeAstraCheck 并行检测给定的各个模型（同时最多 astraCheckModelParallelism 个，优先摊到不同账号）；
// 每个模型各自落库、各自判定、各自推送，结果按传入顺序返回。
func (s *GroupStatusProbeService) probeAstraCheck(ctx context.Context, group *Group, cfg *GroupStatusConfig, models []AstraCheckModelConfig) ([]*GroupStatusAstraCheckExecution, error) {
	if group == nil || cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	if !astraCheckSupportsPlatform(group.Platform) {
		return nil, ErrGroupStatusAstraCheckUnsupported
	}
	if err := ValidateGroupStatusConfig(cfg); err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, nil
	}
	if !s.markAstraCheckRunning(group.ID) {
		return nil, ErrGroupStatusAstraCheckRunning
	}
	defer s.clearAstraCheckRunning(group.ID)
	defer s.clearAstraProgress(group.ID)

	registry, regErr := s.astraBenchmarks()
	parallel := s.astraCheckModelParallelism()
	if parallel > len(models) {
		parallel = len(models)
	}
	results := make([]*GroupStatusAstraCheckExecution, len(models))
	errs := make([]error, len(models))
	// 名额在发起循环里按配置顺序占用，模型按顺序开始（并行度为 1 时即严格串行）
	gate := make(chan struct{}, parallel)
	var wg sync.WaitGroup
launch:
	for i, model := range models {
		select {
		case gate <- struct{}{}:
		case <-ctx.Done():
			break launch
		}
		wg.Add(1)
		go func(i int, model AstraCheckModelConfig) {
			defer wg.Done()
			defer func() { <-gate }()
			if ctx.Err() != nil {
				return
			}
			results[i], errs[i] = s.probeAstraCheckModel(ctx, group, cfg, model, i+1, len(models), registry, regErr)
		}(i, model)
	}
	wg.Wait()

	executions := make([]*GroupStatusAstraCheckExecution, 0, len(models))
	var firstErr error
	for i := range models {
		if errs[i] != nil && firstErr == nil {
			firstErr = errs[i]
		}
		if results[i] != nil {
			executions = append(executions, results[i])
		}
	}
	return executions, firstErr
}

// probeAstraCheckModel 检测一个模型；首次出现「强指向其他模型」时在同一账号上立即复测一轮确认：
// 问的是「这个上游是不是冒充的」，换号复测会让 N 个账号里只有 1 个假的分组永远凑不满连续两次。
func (s *GroupStatusProbeService) probeAstraCheckModel(
	ctx context.Context,
	group *Group,
	cfg *GroupStatusConfig,
	model AstraCheckModelConfig,
	index, count int,
	registry *AstraBenchmarkRegistry,
	regErr error,
) (*GroupStatusAstraCheckExecution, error) {
	// 结果落库后移除这个模型的进度，前端转而显示它的结果
	defer s.endAstraProgress(group.ID, model.ExpectedModel)
	progress := s.beginAstraProgress(group.ID, 1, model.ExpectedModel, index, count)
	account, result := s.executeAstraCheckRun(ctx, group, cfg, model, registry, regErr, progress, 1, nil)
	logAstraCheckRun(group, account, result)
	execution, err := s.saveAstraCheckExecution(ctx, group, cfg, account, result)
	if err != nil {
		return nil, err
	}

	if execution.State != nil &&
		result.Verdict == AstraCheckVerdictMismatch &&
		execution.State.ConsecutiveMismatch == 1 &&
		execution.State.StableStatus != AstraCheckStatusMismatch &&
		ctx.Err() == nil {
		confirmProgress := s.beginAstraProgress(group.ID, 2, model.ExpectedModel, index, count)
		confirmAccount, confirmResult := s.executeAstraCheckRun(ctx, group, cfg, model, registry, regErr, confirmProgress, 2, account)
		logAstraCheckRun(group, confirmAccount, confirmResult)
		confirmed, err := s.saveAstraCheckExecution(ctx, group, cfg, confirmAccount, confirmResult)
		if err != nil {
			return nil, err
		}
		confirmed.Confirmed = true
		return confirmed, nil
	}
	return execution, nil
}

// logAstraCheckRun 每轮结束记一行摘要，方便在服务端日志里定位慢、失败和判定。
func logAstraCheckRun(group *Group, account *Account, result *GroupStatusAstraCheckResult) {
	if group == nil || result == nil {
		return
	}
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	latency := int64(0)
	if result.LatencyMS != nil {
		latency = *result.LatencyMS
	}
	logger.LegacyPrintf(groupStatusAstraCheckLogComponent,
		"[AstraCheck] group=%d model=%s round=%d verdict=%s winner=%q strongest=%q valid=%d/%d completed=%d/%d account=%d latency=%dms reasons=%s detail=%q",
		group.ID, result.ExpectedModel, result.Round, result.Verdict, result.Winner, result.Strongest, result.ValidSamples, result.PlannedSamples,
		result.RequestsCompleted, result.RequestsPlanned, accountID, latency, strings.Join(result.Reasons, ","), result.ErrorDetail)
}

// astraAccountCompatible 报告账号能否承载本平台的指纹请求。
func astraAccountCompatible(group *Group, account *Account) bool {
	if group == nil || account == nil {
		return false
	}
	if !account.IsOAuth() && account.Type != AccountTypeAPIKey {
		return false
	}
	switch group.Platform {
	case PlatformOpenAI:
		return account.Platform == PlatformOpenAI
	case PlatformAnthropic:
		return account.Platform == PlatformAnthropic && !account.IsBedrock()
	default:
		return false
	}
}

// executeAstraCheckRun 为一个预期模型选账号、跑整批、判定；永远返回一个结果，不返回 error。
// pinned 非空时先用该账号（复测确认），不可用再回到正常调度。
func (s *GroupStatusProbeService) executeAstraCheckRun(
	ctx context.Context,
	group *Group,
	cfg *GroupStatusConfig,
	model AstraCheckModelConfig,
	registry *AstraBenchmarkRegistry,
	regErr error,
	progress *astraProgressTracker,
	round int,
	pinned *Account,
) (*Account, *GroupStatusAstraCheckResult) {
	target, targetKnown := astraCheckTarget(model.ExpectedModel)
	requestModel := model.requestModelFor(target)
	// GPT-5.6 Sol 走 Juice 读数、Claude Opus 5.5 / Opus 5 走 ModelTrace，都不用 meow 基准包
	if targetKnown && astraCheckTargetAllowed(group.Platform, model.ExpectedModel) {
		switch target.Method {
		case AstraCheckMethodSolJuice:
			return s.executeSolJuiceRun(ctx, group, cfg, model, target, requestModel, progress, round, pinned)
		case AstraCheckMethodModelTrace:
			return s.executeModelTraceRun(ctx, group, cfg, model, target, requestModel, progress, round, pinned)
		}
	}
	startedAt := time.Now()
	tier := strings.TrimSpace(cfg.AstraCheckTier)
	if tier == "" {
		tier = groupStatusAstraCheckDefaultTier
	}
	result := &GroupStatusAstraCheckResult{
		GroupID:       group.ID,
		ConfigID:      cfg.ID,
		Platform:      group.Platform,
		ExpectedModel: model.ExpectedModel,
		Round:         round,
		RequestModel:  requestModel,
		Tier:          tier,
		Verdict:       AstraCheckVerdictInsufficient,
		StartedAt:     startedAt,
	}
	var bench *AstraBenchmark
	if registry != nil && targetKnown {
		bench = registry.Package(target.PackageID)
	}
	if bench != nil {
		result.BenchmarkPackageID = bench.PackageID
		result.BenchmarkVersion = bench.Version
		result.BenchmarkSHA256 = bench.BodySHA256
		result.ScoringVersion = bench.ScoringVersion
	}

	finish := func(account *Account, samples []astraSample, extraDetail string, reasons ...string) (*Account, *GroupStatusAstraCheckResult) {
		progress.setPhase(AstraCheckPhaseScoring)
		result.FinishedAt = time.Now()
		result.Samples = progress.allSamples()
		latency := result.FinishedAt.Sub(startedAt).Milliseconds()
		result.LatencyMS = &latency
		if account != nil {
			id := account.ID
			result.AccountID = &id
		}
		observations := make(map[string]*AstraCellObservation)
		var lastTransportDetail string
		var lastHTTPCode *int
		anyCompleted := false
		for _, sample := range samples {
			result.InputTokens += sample.Usage.InputTokens
			result.OutputTokens += sample.Usage.OutputTokens
			result.ReasoningTokens += sample.Usage.ReasoningTokens
			if sample.Completed {
				anyCompleted = true
				result.RequestsCompleted++
				obs := observations[sample.CellID]
				if obs == nil {
					obs = &AstraCellObservation{CellID: sample.CellID, Counts: map[string]int{}}
					observations[sample.CellID] = obs
				}
				obs.Counts[sample.Category]++
				continue
			}
			if sample.TransportFailed {
				lastTransportDetail = sample.ErrDetail
				lastHTTPCode = sample.HTTPCode
			}
		}
		if !anyCompleted && lastHTTPCode != nil {
			result.HTTPCode = lastHTTPCode
		}
		result.CostUSD = s.estimateAstraCheckCostUSD(target, requestModel, result.InputTokens, result.OutputTokens)
		detail := mergeProbeErrorDetails(extraDetail, lastTransportDetail)
		if len(reasons) > 0 {
			result.Reasons = reasons
			result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(detail))
			return account, result
		}
		list := make([]AstraCellObservation, 0, len(observations))
		for _, cell := range bench.Cells {
			if obs, ok := observations[cell.ID]; ok {
				list = append(list, *obs)
			}
		}
		score, err := ScoreAstraCheck(bench, tier, model.ExpectedModel, list)
		if err != nil {
			result.Verdict = AstraCheckVerdictInsufficient
			result.Reasons = []string{AstraCheckReasonScoringFailed}
			detail = mergeProbeErrorDetails(detail, err.Error())
		} else {
			result.Verdict = score.Verdict
			result.Winner = score.Winner
			result.Strongest = score.Strongest
			result.Matches = score.Matches
			result.Cells = score.Cells
			result.Reasons = score.Reasons
			result.ValidSamples = score.ValidSamples
			result.PlannedSamples = score.PlannedSamples
		}
		result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(detail))
		return account, result
	}

	switch {
	case !targetKnown || !astraCheckTargetAllowed(group.Platform, model.ExpectedModel):
		return finish(nil, nil, "expected model "+model.ExpectedModel+" is not a target for "+group.Platform+" groups", AstraCheckReasonTargetNotAllowed)
	case regErr != nil || registry == nil:
		detail := "benchmark unavailable"
		if regErr != nil {
			detail = regErr.Error()
		}
		return finish(nil, nil, detail, AstraCheckReasonBenchmarkInvalid)
	case bench == nil || !bench.HasModel(model.ExpectedModel):
		return finish(nil, nil, "benchmark "+target.PackageID+" does not contain "+model.ExpectedModel, AstraCheckReasonTargetNotInBenchmark)
	}

	jobs, err := PlanAstraJobs(bench, tier)
	if err != nil || len(jobs) == 0 {
		detail := "no jobs planned"
		if err != nil {
			detail = err.Error()
		}
		return finish(nil, nil, detail, AstraCheckReasonScoringFailed)
	}
	result.RequestsPlanned = len(jobs)
	progress.setPlanned(len(jobs))

	// 让调度器按请求模型过滤账号（模型支持 / 模型级限流），其余配置照抄
	probeCfg := *cfg
	probeCfg.ProbeModel = requestModel

	// 账号锁定：同步跑第一个任务，拿到 2xx 后整批固定在该账号
	lock := s.lockAstraAccount(ctx, group, &probeCfg, pinned, true, progress, func(candidate *Account) astraSample {
		return s.runAstraJob(ctx, candidate, requestModel, bench, jobs[0], progress)
	})
	defer lock.release()
	account, firstSample, failedSamples, firstFailureDetail := lock.account, lock.first, lock.failed, lock.firstFailureDetail
	if account == nil {
		return finish(nil, failedSamples, mergeProbeErrorDetails(firstFailureDetail, lock.noAccountDetail), AstraCheckReasonNoAccount)
	}
	progress.setPhase(AstraCheckPhaseRunning)

	samples := make([]astraSample, 0, len(jobs)+len(failedSamples))
	samples = append(samples, failedSamples...)
	samples = append(samples, *firstSample)
	if len(jobs) > 1 {
		rest := jobs[1:]
		var mu sync.Mutex
		var wg sync.WaitGroup
		queue := make(chan AstraJob)
		// 连接池按账号隔离时上游连接数 = 账号并发数，超出的请求只会在传输层排队并把超时耗光，
		// 所以在途请求不能高于账号并发：runAstraJob 每次请求前占用账号的服务级名额，
		// 同一账号上并行的其他模型 / 分组合计也不超过这个上限。
		workers := s.astraAccountCapacity(account)
		if workers > len(rest) {
			workers = len(rest)
		}
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range queue {
					if ctx.Err() != nil {
						continue
					}
					sample := s.runAstraJob(ctx, account, requestModel, bench, job, progress)
					mu.Lock()
					samples = append(samples, sample)
					mu.Unlock()
				}
			}()
		}
		for _, job := range rest {
			queue <- job
		}
		close(queue)
		wg.Wait()
	}
	extra := firstFailureDetail
	if ctx.Err() != nil {
		extra = mergeProbeErrorDetails(extra, "run interrupted: "+ctx.Err().Error())
	}
	return finish(account, samples, extra)
}

// runAstraJob 执行一个任务：最多 3 次尝试，传输错误 / 非 2xx / 无效答案都重试。
func (s *GroupStatusProbeService) runAstraJob(ctx context.Context, account *Account, requestModel string, bench *AstraBenchmark, job AstraJob, progress *astraProgressTracker) astraSample {
	sample := astraSample{CellID: job.CellID}
	cell := bench.CellIndex[job.CellID]
	if cell == nil {
		sample.ErrDetail = "unknown cell " + job.CellID
		sample.TransportFailed = true
		progress.record(astraSampleRecordFrom(&sample, 0, true), false)
		return sample
	}
	for attempt := 1; attempt <= groupStatusAstraCheckMaxAttempts; attempt++ {
		if ctx.Err() != nil {
			sample.TransportFailed = true
			sample.ErrDetail = ctx.Err().Error()
			progress.record(astraSampleRecordFrom(&sample, attempt, true), false)
			return sample
		}
		sample.Attempts = attempt
		// 占用账号的一个在途名额（与同一账号上并行的其他检测共享），等待不计入请求超时
		release, acquireErr := s.astraAccounts.acquire(ctx, account.ID, s.astraAccountCapacity(account))
		if acquireErr != nil {
			sample.TransportFailed = true
			sample.Completed = false
			sample.ErrDetail = acquireErr.Error()
			progress.record(astraSampleRecordFrom(&sample, attempt, true), false)
			return sample
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, groupStatusAstraCheckRequestTimeout)
		progress.requestStarted()
		started := time.Now()
		resp, httpCode, err := s.astraCheckRequest(timeoutCtx, account, requestModel, cell)
		cancel()
		release()
		sample.LatencyMS = time.Since(started).Milliseconds()
		sample.Usage.InputTokens += resp.Usage.InputTokens
		sample.Usage.OutputTokens += resp.Usage.OutputTokens
		sample.Usage.ReasoningTokens += resp.Usage.ReasoningTokens
		sample.HTTPCode = httpCode

		non2xx := httpCode != nil && (*httpCode < 200 || *httpCode >= 300)
		switch {
		case httpCode == nil || non2xx:
			sample.TransportFailed = true
			sample.Completed = false
			switch {
			case err != nil:
				sample.ErrDetail = sanitizeProbeErrorDetail(err)
			case httpCode != nil:
				sample.ErrDetail = fmt.Sprintf("unexpected http status: %d", *httpCode)
			default:
				sample.ErrDetail = "no response"
			}
		case err != nil || resp.Invalid != "":
			// 拿到 2xx 但流报错、被截断或拒答：这次回答不投票，按无效答案重试
			sample.TransportFailed = false
			sample.Completed = true
			sample.Category = AstraCheckInvalidOutput
			sample.Answer = truncateProbeText(resp.Text)
			if err != nil {
				sample.ErrDetail = sanitizeProbeErrorDetail(err)
			} else {
				sample.ErrDetail = resp.Invalid
			}
		default:
			sample.TransportFailed = false
			sample.Completed = true
			sample.ErrDetail = ""
			sample.Answer = truncateProbeText(resp.Text)
			sample.Category = NormalizeAstraAnswer(cell.Normalizer, resp.Text)
		}
		valid := sample.Completed && sample.Category != AstraCheckInvalidOutput
		final := valid || attempt == groupStatusAstraCheckMaxAttempts
		progress.record(astraSampleRecordFrom(&sample, attempt, final), true)
		if valid {
			return sample
		}
		if attempt == groupStatusAstraCheckMaxAttempts {
			break
		}
		// 无效答案只是这次没答好，短暂等一下即可；传输 / HTTP 错误多半是上游限流，退避长一些
		backoff := time.Duration(attempt) * 500 * time.Millisecond
		if sample.TransportFailed {
			backoff = time.Duration(attempt) * time.Second
		}
		if err := s.astraSleepFor(ctx, backoff); err != nil {
			sample.TransportFailed = true
			sample.Completed = false
			sample.ErrDetail = err.Error()
			progress.record(astraSampleRecordFrom(&sample, attempt, true), false)
			return sample
		}
	}
	return sample
}

// astraSleepFor 重试退避；测试可通过 astraSleep 字段替换为不等待。
func (s *GroupStatusProbeService) astraSleepFor(ctx context.Context, d time.Duration) error {
	if s != nil && s.astraSleep != nil {
		return s.astraSleep(ctx, d)
	}
	return serverChanSleep(ctx, d)
}

// astraCheckRequest 按账号平台发一道题：OpenAI 走 Responses，Anthropic 走 Messages。
func (s *GroupStatusProbeService) astraCheckRequest(ctx context.Context, account *Account, requestModel string, cell *AstraBenchmarkCell) (astraResponse, *int, error) {
	if account != nil && account.Platform == PlatformAnthropic {
		req, err := s.buildAnthropicMessagesProbeRequest(ctx, account, requestModel, anthropicProbeHeadersClaudeCode, func(modelID string, isOAuth bool) (map[string]any, error) {
			return createAnthropicAstraCheckPayload(modelID, cell, isOAuth)
		})
		if err != nil {
			return astraResponse{}, nil, err
		}
		res, code, err := s.executeAnthropicStreamingProbe(req, account, parseAnthropicMessagesStream)
		resp := astraResponse{
			Text: res.Text,
			Usage: openAIProbeUsage{
				InputTokens:  res.Usage.InputTokens + res.Usage.CacheCreationInputTokens + res.Usage.CacheReadInputTokens,
				OutputTokens: res.Usage.OutputTokens,
			},
		}
		switch {
		case res.StopReason == "max_tokens":
			resp.Invalid = "truncated at max_tokens"
		case res.StopReason == "refusal":
			resp.Invalid = "refused"
		case !res.Completed:
			resp.Invalid = "stream ended without message_stop"
		}
		return resp, code, err
	}

	var completed bool
	text, usage, code, err := s.openAIResponsesProbeRequest(ctx, account, requestModel, func(modelID string, isOAuth bool) map[string]any {
		return createOpenAIAstraCheckPayload(modelID, cell, isOAuth)
	}, func(body io.Reader) (string, openAIProbeUsage, error) {
		t, u, c, e := parseOpenAIResponsesStreamDetailed(body, true)
		completed = c
		return t, u, e
	})
	resp := astraResponse{Text: text, Usage: usage}
	if !completed {
		resp.Invalid = "stream ended without response.completed"
	}
	return resp, code, err
}

// createOpenAIAstraCheckPayload 按基准包的请求契约生成 Responses 请求体：system 句点 + 题面、档位 effort、输出上限。
// Codex OAuth 后端要求 instructions 非空，把 system 文本放进 instructions 并带 encrypted_content。
func createOpenAIAstraCheckPayload(modelID string, cell *AstraBenchmarkCell, isOAuth bool) map[string]any {
	system := cell.System
	if strings.TrimSpace(system) == "" {
		system = "."
	}
	effort := cell.Effort
	if effort == "" {
		effort = "low"
	}
	maxOutput := cell.MaxOutputTokens
	if maxOutput <= 0 {
		maxOutput = 128
	}
	input := []map[string]any{
		{
			"role": "user",
			"content": []map[string]any{
				{"type": "input_text", "text": cell.Prompt},
			},
		},
	}
	payload := map[string]any{
		"model":             modelID,
		"input":             input,
		"reasoning":         map[string]any{"effort": effort},
		"max_output_tokens": maxOutput,
		"stream":            true,
		"store":             false,
	}
	if isOAuth {
		payload["instructions"] = system
		payload["include"] = []string{"reasoning.encrypted_content"}
		return payload
	}
	payload["input"] = append([]map[string]any{
		{
			"role": "system",
			"content": []map[string]any{
				{"type": "input_text", "text": system},
			},
		},
	}, input...)
	return payload
}

// createAnthropicAstraCheckPayload 按基准包的 claude-code 契约生成 Messages 请求体：system 句点、题面、
// 输出上限、adaptive thinking 与 output_config.effort。OAuth 凭证只允许 Claude Code 客户端使用，
// 所以在句点前加一行 Claude Code 身份并带 metadata.user_id；API-Key 账号只发句点。
func createAnthropicAstraCheckPayload(modelID string, cell *AstraBenchmarkCell, isOAuth bool) (map[string]any, error) {
	system := cell.System
	if strings.TrimSpace(system) == "" {
		system = "."
	}
	maxOutput := cell.MaxOutputTokens
	if maxOutput <= 0 {
		maxOutput = 128
	}
	payload := map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": cell.Prompt},
				},
			},
		},
		"max_tokens": maxOutput,
		"stream":     true,
		"system":     system,
	}
	if cell.Profile == "claude-code" {
		effort := cell.Effort
		if effort == "" {
			effort = "low"
		}
		payload["thinking"] = map[string]any{"type": "adaptive"}
		payload["output_config"] = map[string]any{"effort": effort}
	}
	if isOAuth {
		sessionID, err := generateSessionString()
		if err != nil {
			return nil, err
		}
		payload["system"] = []map[string]any{
			{"type": "text", "text": claudeCodeSystemPrompt},
			{"type": "text", "text": system},
		}
		payload["metadata"] = map[string]string{"user_id": sessionID}
	}
	return payload, nil
}

// estimateAstraCheckCostUSD 按网关计价估算一次运行的费用（目标的默认请求模型优先，其次实际请求模型）；无价格时为 0。
func (s *GroupStatusProbeService) estimateAstraCheckCostUSD(target AstraCheckTarget, requestModel string, inputTokens, outputTokens int64) float64 {
	if s == nil || s.gatewaySvc == nil || s.gatewaySvc.billingService == nil {
		return 0
	}
	if inputTokens <= 0 && outputTokens <= 0 {
		return 0
	}
	tokens := UsageTokens{InputTokens: int(inputTokens), OutputTokens: int(outputTokens)}
	for _, model := range []string{target.DefaultRequestModel, requestModel} {
		if strings.TrimSpace(model) == "" {
			continue
		}
		if cost, err := s.gatewaySvc.billingService.CalculateCost(model, tokens, 1); err == nil && cost != nil {
			return cost.TotalCost
		}
	}
	return 0
}

func (s *GroupStatusProbeService) saveAstraCheckExecution(ctx context.Context, group *Group, cfg *GroupStatusConfig, account *Account, result *GroupStatusAstraCheckResult) (*GroupStatusAstraCheckExecution, error) {
	if result == nil {
		result = &GroupStatusAstraCheckResult{
			Verdict:     AstraCheckVerdictInsufficient,
			Reasons:     []string{"empty_result"},
			ErrorDetail: "empty run result",
			StartedAt:   time.Now(),
			FinishedAt:  time.Now(),
		}
	}
	result.GroupID = group.ID
	result.ConfigID = cfg.ID
	if result.Platform == "" {
		result.Platform = group.Platform
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Now()
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = result.FinishedAt
	}
	result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(result.ErrorDetail))

	run, state, event, err := s.repo.SaveAstraCheckRun(ctx, result)
	if err != nil {
		return nil, err
	}
	if event != nil && s.notifier != nil {
		s.notifier.NotifyTransition(group, cfg, event)
	}
	return &GroupStatusAstraCheckExecution{
		Group:   group,
		Config:  cfg,
		Account: account,
		Result:  result,
		Run:     run,
		State:   state,
		Event:   event,
	}, nil
}
