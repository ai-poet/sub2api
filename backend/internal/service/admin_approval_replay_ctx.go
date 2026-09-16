package service

import "context"

// ApprovalReplay 审批通过后内部重放请求时挂在 request context 上的身份信息。
//
// 它只能由 Go 代码通过 WithApprovalReplay 设置（未导出的 context key），HTTP 请求头无法伪造；
// 认证中间件识别到它后直接以审批人（管理员）身份放行，不再校验 token。
// 放在 service 包是为了避免 middleware ↔ service 的 import 环。
type ApprovalReplay struct {
	ApprovalID          int64
	RequesterUserID     int64
	ApproverUserID      int64
	ApproverConcurrency int
	ApproverRole        string
	ApproverEmail       string
	ApproverSessionID   string
}

type approvalReplayCtxKey struct{}

// WithApprovalReplay 把重放身份挂到 ctx 上。
func WithApprovalReplay(ctx context.Context, replay *ApprovalReplay) context.Context {
	if ctx == nil || replay == nil {
		return ctx
	}
	return context.WithValue(ctx, approvalReplayCtxKey{}, replay)
}

// ApprovalReplayFromContext 读取重放身份；普通 HTTP 请求永远返回 false。
func ApprovalReplayFromContext(ctx context.Context) (*ApprovalReplay, bool) {
	if ctx == nil {
		return nil, false
	}
	replay, ok := ctx.Value(approvalReplayCtxKey{}).(*ApprovalReplay)
	if !ok || replay == nil {
		return nil, false
	}
	return replay, true
}
