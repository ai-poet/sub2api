package service

import (
	"context"
	"sync"
	"time"
)

// probeTiming 记录一次存活探测里流式响应的关键时刻（响应头到达、首个内容 token 到达）。
//
// 探测的「延迟」按首字时间计：非流式的完整返回时间受输出长度影响，反映不了用户感受到的等待。
// 通过 context 传给各平台的请求构造函数，避免改动它们的签名；拿不到 timing（非流式探测）时退回总耗时。
type probeTiming struct {
	mu         sync.Mutex
	startedAt  time.Time
	headersAt  time.Time
	firstToken time.Time
}

type probeTimingContextKey struct{}

func newProbeTiming(startedAt time.Time) *probeTiming {
	return &probeTiming{startedAt: startedAt}
}

func withProbeTiming(ctx context.Context, timing *probeTiming) context.Context {
	if timing == nil {
		return ctx
	}
	return context.WithValue(ctx, probeTimingContextKey{}, timing)
}

// probeTimingFromContext 取出探测计时器；没有时返回 nil，所有方法对 nil 接收者都安全。
func probeTimingFromContext(ctx context.Context) *probeTiming {
	if ctx == nil {
		return nil
	}
	timing, _ := ctx.Value(probeTimingContextKey{}).(*probeTiming)
	return timing
}

// markHeaders 记响应头到达（只记第一次）。
func (t *probeTiming) markHeaders() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.headersAt.IsZero() {
		t.headersAt = time.Now()
	}
}

// markFirstToken 记首个内容 token 到达（只记第一次）。
func (t *probeTiming) markFirstToken() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.firstToken.IsZero() {
		t.firstToken = time.Now()
	}
}

// firstTokenMS 返回请求开始到首字的毫秒数；没收到内容时 ok 为 false。
func (t *probeTiming) firstTokenMS() (int64, bool) {
	if t == nil {
		return 0, false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.firstToken.IsZero() || t.startedAt.IsZero() {
		return 0, false
	}
	ms := t.firstToken.Sub(t.startedAt).Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return ms, true
}

// headersMS 返回请求开始到响应头到达的毫秒数；没收到响应头时 ok 为 false。
func (t *probeTiming) headersMS() (int64, bool) {
	if t == nil {
		return 0, false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.headersAt.IsZero() || t.startedAt.IsZero() {
		return 0, false
	}
	ms := t.headersAt.Sub(t.startedAt).Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return ms, true
}

// probeNotifyFirstToken 解析器在拿到第一段内容时调用；回调可为 nil。
func probeNotifyFirstToken(fn func()) {
	if fn != nil {
		fn()
	}
}
