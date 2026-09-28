package service

import (
	"sync"
	"time"
)

// ModelTrace 指纹验证的实时进度（本 fork 自有功能）。
//
// 一轮验证要发 3–6 条长输出请求，默认推理下可能持续数分钟；管理端轮询时需要看到进行到哪一步、
// 每条挑战是否计入。进度只存在内存里（按分组），运行结束即清除；逐条诊断随运行记录落库。

const (
	ModelTracePhaseSelectingAccount = "selecting_account"
	ModelTracePhaseRunning          = "running"
	ModelTracePhaseScoring          = "scoring"

	ModelTraceAttemptAccepted = "accepted"
	ModelTraceAttemptRejected = "rejected"
	ModelTraceAttemptFailed   = "failed"
)

// ModelTraceAttemptProgress 是一条挑战的可展示摘要（不含回答原文）。
type ModelTraceAttemptProgress struct {
	Seq            int       `json:"seq"`
	ExpectedCount  int       `json:"expected_count"`
	ParsedNumbers  int       `json:"parsed_numbers"`
	MinimumNumbers int       `json:"minimum_numbers"`
	Outcome        string    `json:"outcome"`
	Rejection      string    `json:"rejection,omitempty"`
	StopReason     string    `json:"stop_reason,omitempty"`
	HTTPCode       *int      `json:"http_code"`
	LatencyMS      int64     `json:"latency_ms"`
	At             time.Time `json:"at"`
}

// ModelTraceProgress 是某分组当前这一轮验证的进度快照。
type ModelTraceProgress struct {
	Round     int                         `json:"round"`
	Phase     string                      `json:"phase"`
	AccountID *int64                      `json:"account_id"`
	Planned   int                         `json:"planned"`
	Target    int                         `json:"target"`
	Used      int                         `json:"used"`
	Accepted  int                         `json:"accepted"`
	Rejected  int                         `json:"rejected"`
	Failed    int                         `json:"failed"`
	Requests  int                         `json:"requests"`
	InFlight  int                         `json:"in_flight"`
	StartedAt time.Time                   `json:"started_at"`
	UpdatedAt time.Time                   `json:"updated_at"`
	ElapsedMS int64                       `json:"elapsed_ms"`
	Attempts  []ModelTraceAttemptProgress `json:"attempts"`
}

type modelTraceProgressTracker struct {
	mu       sync.Mutex
	progress ModelTraceProgress
	attempts []ModelTraceAttemptProgress
}

func newModelTraceProgressTracker(round int) *modelTraceProgressTracker {
	now := time.Now()
	return &modelTraceProgressTracker{
		progress: ModelTraceProgress{
			Round:     round,
			Phase:     ModelTracePhaseSelectingAccount,
			Planned:   groupStatusModelTraceMaxChallenges,
			Target:    groupStatusModelTraceTargetOutputs,
			StartedAt: now,
			UpdatedAt: now,
		},
	}
}

func (t *modelTraceProgressTracker) update(fn func(p *ModelTraceProgress)) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	fn(&t.progress)
	t.progress.UpdatedAt = time.Now()
}

func (t *modelTraceProgressTracker) setPhase(phase string) {
	t.update(func(p *ModelTraceProgress) { p.Phase = phase })
}

func (t *modelTraceProgressTracker) setAccount(id int64) {
	t.update(func(p *ModelTraceProgress) {
		v := id
		p.AccountID = &v
	})
}

// requestStarted 记一次发出的 HTTP 请求（含传输层重试）。
func (t *modelTraceProgressTracker) requestStarted() {
	t.update(func(p *ModelTraceProgress) {
		p.Requests++
		p.InFlight++
	})
}

func (t *modelTraceProgressTracker) requestFinished() {
	t.update(func(p *ModelTraceProgress) {
		if p.InFlight > 0 {
			p.InFlight--
		}
	})
}

// record 记录一条挑战的最终结果。
func (t *modelTraceProgressTracker) record(rec ModelTraceOutputRecord) {
	if t == nil {
		return
	}
	outcome := ModelTraceAttemptFailed
	switch {
	case rec.Accepted:
		outcome = ModelTraceAttemptAccepted
	case rec.Error == "":
		outcome = ModelTraceAttemptRejected
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.attempts = append(t.attempts, ModelTraceAttemptProgress{
		Seq:            rec.Seq,
		ExpectedCount:  rec.ExpectedCount,
		ParsedNumbers:  rec.ParsedNumbers,
		MinimumNumbers: rec.MinimumNumbers,
		Outcome:        outcome,
		Rejection:      rec.Rejection,
		StopReason:     rec.StopReason,
		HTTPCode:       rec.HTTPCode,
		LatencyMS:      rec.LatencyMS,
		At:             now,
	})
	t.progress.Used++
	switch outcome {
	case ModelTraceAttemptAccepted:
		t.progress.Accepted++
	case ModelTraceAttemptRejected:
		t.progress.Rejected++
	default:
		t.progress.Failed++
	}
	t.progress.UpdatedAt = now
}

// rollbackFailover 换号重跑首条挑战时，撤回上一账号那次失败的计数与记录。
func (t *modelTraceProgressTracker) rollbackFailover() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.attempts) > 0 {
		t.attempts = t.attempts[:len(t.attempts)-1]
	}
	if t.progress.Used > 0 {
		t.progress.Used--
	}
	if t.progress.Failed > 0 {
		t.progress.Failed--
	}
	t.progress.UpdatedAt = time.Now()
}

func (t *modelTraceProgressTracker) snapshot() *ModelTraceProgress {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	out := t.progress
	out.ElapsedMS = time.Since(t.progress.StartedAt).Milliseconds()
	// 一定是非 nil 切片：JSON 里要输出 []，前端会直接读 .length
	out.Attempts = append(make([]ModelTraceAttemptProgress, 0, len(t.attempts)), t.attempts...)
	return &out
}

// ---------- 探测服务上的进度存取 ----------

func (s *GroupStatusProbeService) beginModelTraceProgress(groupID int64, round int) *modelTraceProgressTracker {
	tracker := newModelTraceProgressTracker(round)
	if s != nil {
		s.modelTraceProgress.Store(groupID, tracker)
	}
	return tracker
}

func (s *GroupStatusProbeService) clearModelTraceProgress(groupID int64) {
	if s != nil {
		s.modelTraceProgress.Delete(groupID)
	}
}

// ModelTraceProgress 返回该分组正在进行的验证进度；没有在跑时返回 nil。
func (s *GroupStatusProbeService) ModelTraceProgress(groupID int64) *ModelTraceProgress {
	if s == nil {
		return nil
	}
	v, ok := s.modelTraceProgress.Load(groupID)
	if !ok {
		return nil
	}
	tracker, ok := v.(*modelTraceProgressTracker)
	if !ok {
		return nil
	}
	return tracker.snapshot()
}
