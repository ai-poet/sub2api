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

// Astra 指纹验证的执行（本 fork 自有功能）。
//
// 一次运行 = 选一个 OpenAI 账号 → 按档位跑一批固定短答题 → 归一化计数 → 本地判定 → 落库 →
// 稳定结论切换时推送。与存活探测、Sol Juice 互不影响。

const groupStatusAstraCheckLogComponent = "service.group_status_astra_check"

// astraSample 是一个任务的最终结果。
type astraSample struct {
	CellID          string
	Category        string
	Answer          string
	HTTPCode        *int
	LatencyMS       int64
	Usage           openAIProbeUsage
	Completed       bool // 拿到 2xx 且流正常结束
	TransportFailed bool // 最后一次尝试仍是传输/HTTP 错误
	ErrDetail       string
	Attempts        int
}

func (s *GroupStatusProbeService) astraBenchmark() (*AstraBenchmark, *AstraBenchmarkMeta, error) {
	if s != nil && s.astraBenchmarks != nil {
		return s.astraBenchmarks.Active()
	}
	return LoadEmbeddedAstraBenchmark()
}

// SetAstraBenchmarkProvider 替换基准来源（测试用）。
func (s *GroupStatusProbeService) SetAstraBenchmarkProvider(p astraBenchmarkProvider) {
	if s == nil {
		return
	}
	s.astraBenchmarks = p
}

func (s *GroupStatusProbeService) astraCheckConcurrency() int {
	if s != nil && s.astraConcurrency > 0 {
		return s.astraConcurrency
	}
	return groupStatusAstraCheckDefaultConcurrency
}

// IsAstraCheckRunning 报告该分组是否有正在进行的 Astra 指纹验证。
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

// StartAstraCheckAsync 在后台启动一次验证（管理端「立即验证」用）；同一分组运行中则报错。
func (s *GroupStatusProbeService) StartAstraCheckAsync(groupID int64) error {
	if s == nil {
		return errors.New("group status probe service is not configured")
	}
	if s.IsAstraCheckRunning(groupID) {
		return ErrGroupStatusAstraCheckRunning
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), groupStatusAstraCheckRunBudget)
		defer cancel()
		if _, err := s.ProbeAstraCheckGroupNow(ctx, groupID); err != nil {
			logger.LegacyPrintf(groupStatusAstraCheckLogComponent, "[AstraCheck] group=%d manual run failed: %v", groupID, err)
		}
	}()
	return nil
}

// ProbeAstraCheckGroupNow 立即验证；不要求分组已开启 astra_check_enabled。
func (s *GroupStatusProbeService) ProbeAstraCheckGroupNow(ctx context.Context, groupID int64) (*GroupStatusAstraCheckExecution, error) {
	group, cfg, err := s.ensureProbeTarget(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return s.probeAstraCheck(ctx, group, cfg)
}

// ProbeAstraCheckWithConfig 供定时 runner 调用。
func (s *GroupStatusProbeService) ProbeAstraCheckWithConfig(ctx context.Context, cfg *GroupStatusConfig) (*GroupStatusAstraCheckExecution, error) {
	if cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	group, err := s.groupRepo.GetByID(ctx, cfg.GroupID)
	if err != nil {
		return nil, err
	}
	return s.probeAstraCheck(ctx, group, cfg)
}

func (s *GroupStatusProbeService) probeAstraCheck(ctx context.Context, group *Group, cfg *GroupStatusConfig) (*GroupStatusAstraCheckExecution, error) {
	if group == nil || cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	if group.Platform != PlatformOpenAI {
		return nil, ErrGroupStatusAstraCheckUnsupported
	}
	if err := ValidateGroupStatusConfig(cfg); err != nil {
		return nil, err
	}
	bench, meta, err := s.astraBenchmark()
	if err != nil {
		return nil, err
	}
	if !s.markAstraCheckRunning(group.ID) {
		return nil, ErrGroupStatusAstraCheckRunning
	}
	defer s.clearAstraCheckRunning(group.ID)

	account, result := s.executeAstraCheckRun(ctx, group, cfg, bench, meta)
	execution, err := s.saveAstraCheckExecution(ctx, group, cfg, account, result)
	if err != nil {
		return nil, err
	}

	// 首次出现「强指向其他模型」时立即复测一轮确认，不等下一个间隔。
	if execution.State != nil &&
		result.Verdict == AstraCheckVerdictMismatch &&
		execution.State.AstraCheckConsecutiveMismatch == 1 &&
		execution.State.AstraCheckStableStatus != AstraCheckStatusMismatch &&
		ctx.Err() == nil {
		confirmAccount, confirmResult := s.executeAstraCheckRun(ctx, group, cfg, bench, meta)
		confirmed, err := s.saveAstraCheckExecution(ctx, group, cfg, confirmAccount, confirmResult)
		if err != nil {
			return nil, err
		}
		confirmed.Confirmed = true
		return confirmed, nil
	}
	return execution, nil
}

// executeAstraCheckRun 选账号、跑整批、判定；永远返回一个结果，不返回 error。
func (s *GroupStatusProbeService) executeAstraCheckRun(ctx context.Context, group *Group, cfg *GroupStatusConfig, bench *AstraBenchmark, meta *AstraBenchmarkMeta) (*Account, *GroupStatusAstraCheckResult) {
	startedAt := time.Now()
	requestModel := strings.TrimSpace(cfg.AstraCheckRequestModel)
	if requestModel == "" {
		requestModel = groupStatusAstraCheckDefaultRequestModel
	}
	tier := strings.TrimSpace(cfg.AstraCheckTier)
	if tier == "" {
		tier = groupStatusAstraCheckDefaultTier
	}
	result := &GroupStatusAstraCheckResult{
		GroupID:      group.ID,
		ConfigID:     cfg.ID,
		RequestModel: requestModel,
		Tier:         tier,
		Verdict:      AstraCheckVerdictInsufficient,
		StartedAt:    startedAt,
	}
	if meta != nil {
		result.BenchmarkPackageID = meta.PackageID
		result.BenchmarkVersion = meta.Version
		result.BenchmarkSHA256 = meta.BodySHA256
	}
	finish := func(account *Account, samples []astraSample, extraDetail string) (*Account, *GroupStatusAstraCheckResult) {
		result.FinishedAt = time.Now()
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
		list := make([]AstraCellObservation, 0, len(observations))
		for _, cell := range bench.Cells {
			if obs, ok := observations[cell.ID]; ok {
				list = append(list, *obs)
			}
		}
		score, err := ScoreAstraCheck(bench, tier, list)
		if err != nil {
			result.Verdict = AstraCheckVerdictInsufficient
			result.Reasons = []string{"scoring_failed"}
			extraDetail = mergeProbeErrorDetails(extraDetail, err.Error())
		} else {
			result.Verdict = score.Verdict
			result.Winner = score.Winner
			result.Matches = score.Matches
			result.Cells = score.Cells
			result.Reasons = score.Reasons
			result.ValidSamples = score.ValidSamples
			result.PlannedSamples = score.PlannedSamples
		}
		detail := mergeProbeErrorDetails(extraDetail, lastTransportDetail)
		result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(detail))
		return account, result
	}

	jobs, err := PlanAstraJobs(bench, tier)
	if err != nil || len(jobs) == 0 {
		detail := "no jobs planned"
		if err != nil {
			detail = err.Error()
		}
		return finish(nil, nil, detail)
	}
	result.RequestsPlanned = len(jobs)

	// 让调度器按 Astra 请求模型过滤账号，其余配置照抄
	probeCfg := *cfg
	probeCfg.ProbeModel = requestModel

	// 账号锁定：同步跑第一个任务，拿到 2xx 后整批固定在该账号
	excludedIDs := make(map[int64]struct{})
	maxAttempts := s.maxProbeAttempts(group)
	var (
		account            *Account
		firstSample        *astraSample
		firstFailureDetail string
		failedSamples      []astraSample // 换号前失败的首个任务，保留以便汇总 HTTP 码与 token
	)
	for attemptNo := 0; attemptNo < maxAttempts; attemptNo++ {
		attempt, selectErr := s.selectProbeAttempt(ctx, group, &probeCfg, excludedIDs)
		if selectErr != nil {
			return finish(nil, failedSamples, mergeProbeErrorDetails(firstFailureDetail, "no schedulable account: "+selectErr.Error()))
		}
		if attempt == nil || attempt.Account == nil {
			return finish(nil, failedSamples, mergeProbeErrorDetails(firstFailureDetail, "no schedulable account available"))
		}
		candidate := attempt.Account
		if candidate.Platform != PlatformOpenAI || attempt.WaitPlan != nil {
			excludedIDs[candidate.ID] = struct{}{}
			continue
		}
		sample := s.runAstraJob(ctx, candidate, requestModel, bench, jobs[0])
		if sample.TransportFailed {
			failure := &GroupStatusProbeResult{HTTPCode: sample.HTTPCode}
			if s.shouldProbeFailover(candidate, failure, errors.New(sample.ErrDetail)) && attemptNo < maxAttempts-1 {
				if firstFailureDetail == "" {
					firstFailureDetail = fmt.Sprintf("account %d: %s", candidate.ID, truncateProbeText(sample.ErrDetail))
				}
				failedSamples = append(failedSamples, sample)
				excludedIDs[candidate.ID] = struct{}{}
				continue
			}
		}
		account = candidate
		firstSample = &sample
		break
	}
	if account == nil {
		return finish(nil, failedSamples, mergeProbeErrorDetails(firstFailureDetail, "failover_exhausted"))
	}

	samples := make([]astraSample, 0, len(jobs))
	samples = append(samples, *firstSample)
	if len(jobs) > 1 {
		rest := jobs[1:]
		var mu sync.Mutex
		var wg sync.WaitGroup
		queue := make(chan AstraJob)
		workers := s.astraCheckConcurrency()
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
					sample := s.runAstraJob(ctx, account, requestModel, bench, job)
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
func (s *GroupStatusProbeService) runAstraJob(ctx context.Context, account *Account, requestModel string, bench *AstraBenchmark, job AstraJob) astraSample {
	sample := astraSample{CellID: job.CellID}
	cell := bench.CellIndex[job.CellID]
	if cell == nil {
		sample.ErrDetail = "unknown cell " + job.CellID
		sample.TransportFailed = true
		return sample
	}
	for attempt := 1; attempt <= groupStatusAstraCheckMaxAttempts; attempt++ {
		if ctx.Err() != nil {
			sample.TransportFailed = true
			sample.ErrDetail = ctx.Err().Error()
			return sample
		}
		sample.Attempts = attempt
		timeoutCtx, cancel := context.WithTimeout(ctx, groupStatusAstraCheckRequestTimeout)
		started := time.Now()
		text, usage, httpCode, err := s.astraCheckRequest(timeoutCtx, account, requestModel, cell)
		cancel()
		sample.LatencyMS = time.Since(started).Milliseconds()
		sample.Usage.InputTokens += usage.InputTokens
		sample.Usage.OutputTokens += usage.OutputTokens
		sample.Usage.ReasoningTokens += usage.ReasoningTokens
		sample.HTTPCode = httpCode

		if err != nil || (httpCode != nil && (*httpCode < 200 || *httpCode >= 300)) {
			sample.TransportFailed = true
			sample.Completed = false
			if err != nil {
				sample.ErrDetail = sanitizeProbeErrorDetail(err)
			} else {
				sample.ErrDetail = fmt.Sprintf("unexpected http status: %d", *httpCode)
			}
		} else {
			sample.TransportFailed = false
			sample.Completed = true
			sample.ErrDetail = ""
			sample.Answer = truncateProbeText(text)
			sample.Category = NormalizeAstraAnswer(cell.Normalizer, text)
			if sample.Category != AstraCheckInvalidOutput {
				return sample
			}
		}
		if attempt == groupStatusAstraCheckMaxAttempts {
			break
		}
		// 无效答案只是模型没按格式答，短暂等一下即可；传输 / HTTP 错误多半是上游限流，退避长一些
		backoff := time.Duration(attempt) * 500 * time.Millisecond
		if sample.TransportFailed {
			backoff = time.Duration(attempt) * time.Second
		}
		if err := s.astraSleepFor(ctx, backoff); err != nil {
			sample.TransportFailed = true
			sample.Completed = false
			sample.ErrDetail = err.Error()
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

// astraCheckRequest 复用 Sol Juice 的鉴权 / 地址 / 模型映射逻辑，只换请求体。
func (s *GroupStatusProbeService) astraCheckRequest(ctx context.Context, account *Account, requestModel string, cell *AstraBenchmarkCell) (string, openAIProbeUsage, *int, error) {
	return s.openAIResponsesProbeRequest(ctx, account, requestModel, func(modelID string, isOAuth bool) map[string]any {
		return createOpenAIAstraCheckPayload(modelID, cell, isOAuth)
	}, func(body io.Reader) (string, openAIProbeUsage, error) {
		return parseOpenAIResponsesStream(body, false)
	})
}

// createOpenAIAstraCheckPayload 按基准包的请求契约生成请求体：system 句点 + 题面、low 档、128 token 上限。
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
