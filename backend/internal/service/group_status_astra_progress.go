package service

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// meow 指纹验证的实时进度（本 fork 自有功能）。
//
// 一轮验证要发几十个请求、持续几十秒到几分钟，管理端轮询时需要看到「跑到哪了、每个请求答了什么」。
// 进度只存在内存里（按分组），运行结束即清除；逐请求样本随运行记录落库供事后回看。

const (
	AstraCheckPhaseSelectingAccount = "selecting_account"
	AstraCheckPhaseRunning          = "running"
	AstraCheckPhaseScoring          = "scoring"

	AstraCheckSampleValid   = "valid"
	AstraCheckSampleInvalid = "invalid"
	AstraCheckSampleFailed  = "failed"

	astraProgressSnapshotSamples = 30
	astraProgressMaxSamples      = 600
	astraSampleAnswerMaxRunes    = 120
)

// AstraCheckSampleRecord 是一次请求尝试的可展示摘要。
type AstraCheckSampleRecord struct {
	Seq       int       `json:"seq"`
	CellID    string    `json:"cell_id"`
	Attempt   int       `json:"attempt"`
	Answer    string    `json:"answer"`
	Category  string    `json:"category"`
	Outcome   string    `json:"outcome"`
	Final     bool      `json:"final"`
	HTTPCode  *int      `json:"http_code"`
	LatencyMS int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
	At        time.Time `json:"at"`
}

// AstraCheckProgress 是某分组当前这一轮验证的进度快照；一次运行依次检测多个模型时，
// ModelIndex / ModelCount 表示正在检测第几个（从 1 开始）。
type AstraCheckProgress struct {
	ExpectedModel string                   `json:"expected_model"`
	ModelIndex    int                      `json:"model_index"`
	ModelCount    int                      `json:"model_count"`
	Round         int                      `json:"round"`
	Phase         string                   `json:"phase"`
	AccountID     *int64                   `json:"account_id"`
	Planned       int                      `json:"planned"`
	Completed     int                      `json:"completed"`
	Valid         int                      `json:"valid"`
	Invalid       int                      `json:"invalid"`
	Failed        int                      `json:"failed"`
	Requests      int                      `json:"requests"`
	InFlight      int                      `json:"in_flight"`
	StartedAt     time.Time                `json:"started_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	ElapsedMS     int64                    `json:"elapsed_ms"`
	Samples       []AstraCheckSampleRecord `json:"samples"`
}

type astraProgressTracker struct {
	mu       sync.Mutex
	progress AstraCheckProgress
	samples  []AstraCheckSampleRecord
}

func newAstraProgressTracker(round int, expectedModel string, index, count int) *astraProgressTracker {
	now := time.Now()
	return &astraProgressTracker{
		progress: AstraCheckProgress{
			ExpectedModel: expectedModel,
			ModelIndex:    index,
			ModelCount:    count,
			Round:         round,
			Phase:         AstraCheckPhaseSelectingAccount,
			StartedAt:     now,
			UpdatedAt:     now,
		},
	}
}

func (t *astraProgressTracker) update(fn func(p *AstraCheckProgress)) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	fn(&t.progress)
	t.progress.UpdatedAt = time.Now()
}

func (t *astraProgressTracker) setPhase(phase string) {
	t.update(func(p *AstraCheckProgress) { p.Phase = phase })
}

func (t *astraProgressTracker) setPlanned(n int) {
	t.update(func(p *AstraCheckProgress) { p.Planned = n })
}

func (t *astraProgressTracker) setAccount(id int64) {
	t.update(func(p *AstraCheckProgress) {
		v := id
		p.AccountID = &v
	})
}

// requestStarted 记一次发出的 HTTP 请求（含重试）。
func (t *astraProgressTracker) requestStarted() {
	t.update(func(p *AstraCheckProgress) {
		p.Requests++
		p.InFlight++
	})
}

// record 记录一次尝试的结果；endsRequest 表示这条记录对应一个已发出的请求（在途数减一）；
// rec.Final 表示该任务到此结束（成功、或重试耗尽），任务计数与结果计数才会增加。
func (t *astraProgressTracker) record(rec AstraCheckSampleRecord, endsRequest bool) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if endsRequest && t.progress.InFlight > 0 {
		t.progress.InFlight--
	}
	rec.Seq = len(t.samples) + 1
	rec.At = time.Now()
	rec.Answer = truncateAstraAnswer(rec.Answer)
	if len(t.samples) < astraProgressMaxSamples {
		t.samples = append(t.samples, rec)
	}
	if rec.Final {
		t.progress.Completed++
		switch rec.Outcome {
		case AstraCheckSampleValid:
			t.progress.Valid++
		case AstraCheckSampleInvalid:
			t.progress.Invalid++
		default:
			t.progress.Failed++
		}
	}
	t.progress.UpdatedAt = rec.At
}

// rollbackJob 换号重跑首个任务时，把上一账号那次「已结束」的计数撤回。
func (t *astraProgressTracker) rollbackJob(outcome string) {
	t.update(func(p *AstraCheckProgress) {
		if p.Completed > 0 {
			p.Completed--
		}
		switch outcome {
		case AstraCheckSampleValid:
			if p.Valid > 0 {
				p.Valid--
			}
		case AstraCheckSampleInvalid:
			if p.Invalid > 0 {
				p.Invalid--
			}
		default:
			if p.Failed > 0 {
				p.Failed--
			}
		}
	})
}

// snapshot 返回带最近样本的进度副本（最新的在最后）。
func (t *astraProgressTracker) snapshot() *AstraCheckProgress {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	out := t.progress
	out.ElapsedMS = time.Since(t.progress.StartedAt).Milliseconds()
	start := 0
	if len(t.samples) > astraProgressSnapshotSamples {
		start = len(t.samples) - astraProgressSnapshotSamples
	}
	// 一定是非 nil 切片：JSON 里要输出 []，前端会直接读 .length
	out.Samples = make([]AstraCheckSampleRecord, 0, len(t.samples)-start)
	out.Samples = append(out.Samples, t.samples[start:]...)
	return &out
}

func (t *astraProgressTracker) allSamples() []AstraCheckSampleRecord {
	if t == nil {
		return []AstraCheckSampleRecord{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]AstraCheckSampleRecord{}, t.samples...)
}

// astraSampleRecordFrom 把一次尝试的内部结果转成可展示记录。
func astraSampleRecordFrom(sample *astraSample, attempt int, final bool) AstraCheckSampleRecord {
	rec := AstraCheckSampleRecord{
		CellID:    sample.CellID,
		Attempt:   attempt,
		Answer:    sample.Answer,
		Category:  sample.Category,
		Final:     final,
		HTTPCode:  sample.HTTPCode,
		LatencyMS: sample.LatencyMS,
		Error:     sample.ErrDetail,
	}
	switch {
	case sample.TransportFailed || !sample.Completed:
		rec.Outcome = AstraCheckSampleFailed
	case sample.Category == AstraCheckInvalidOutput:
		rec.Outcome = AstraCheckSampleInvalid
	default:
		rec.Outcome = AstraCheckSampleValid
	}
	return rec
}

func truncateAstraAnswer(text string) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= astraSampleAnswerMaxRunes {
		return text
	}
	return string(runes[:astraSampleAnswerMaxRunes]) + "…"
}

// ---------- 探测服务上的进度存取 ----------

// astraGroupProgress 是一个分组里正在并行检测的各模型的进度（按预期模型）。
type astraGroupProgress struct {
	mu       sync.Mutex
	trackers map[string]*astraProgressTracker
}

func (s *GroupStatusProbeService) groupAstraProgress(groupID int64) *astraGroupProgress {
	v, _ := s.astraProgress.LoadOrStore(groupID, &astraGroupProgress{trackers: make(map[string]*astraProgressTracker)})
	return v.(*astraGroupProgress)
}

// beginAstraProgress 为某个模型开始一轮新的进度（同一模型的复测会替换上一轮）。
func (s *GroupStatusProbeService) beginAstraProgress(groupID int64, round int, expectedModel string, index, count int) *astraProgressTracker {
	tracker := newAstraProgressTracker(round, expectedModel, index, count)
	if s != nil {
		group := s.groupAstraProgress(groupID)
		group.mu.Lock()
		group.trackers[expectedModel] = tracker
		group.mu.Unlock()
	}
	return tracker
}

// endAstraProgress 在某个模型检测结束（结果已落库）后移除它的进度。
func (s *GroupStatusProbeService) endAstraProgress(groupID int64, expectedModel string) {
	if s == nil {
		return
	}
	v, ok := s.astraProgress.Load(groupID)
	if !ok {
		return
	}
	group := v.(*astraGroupProgress)
	group.mu.Lock()
	delete(group.trackers, expectedModel)
	group.mu.Unlock()
}

func (s *GroupStatusProbeService) clearAstraProgress(groupID int64) {
	if s != nil {
		s.astraProgress.Delete(groupID)
	}
}

// AstraCheckProgresses 返回该分组正在检测的各模型进度（按模型序号）；没有在跑时返回空切片。
func (s *GroupStatusProbeService) AstraCheckProgresses(groupID int64) []AstraCheckProgress {
	out := []AstraCheckProgress{}
	if s == nil {
		return out
	}
	v, ok := s.astraProgress.Load(groupID)
	if !ok {
		return out
	}
	group := v.(*astraGroupProgress)
	group.mu.Lock()
	trackers := make([]*astraProgressTracker, 0, len(group.trackers))
	for _, tracker := range group.trackers {
		trackers = append(trackers, tracker)
	}
	group.mu.Unlock()
	for _, tracker := range trackers {
		if snap := tracker.snapshot(); snap != nil {
			out = append(out, *snap)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ModelIndex != out[j].ModelIndex {
			return out[i].ModelIndex < out[j].ModelIndex
		}
		return out[i].ExpectedModel < out[j].ExpectedModel
	})
	return out
}
