package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 运维管理员（operator）写操作审批（fork 本地功能，见 docs/OPERATOR_ROLE.md「审批」一节）。
//
// operator 对用户管理 / 订阅管理发起的写请求不会直接执行：认证层把原始请求（方法、路径、body）
// 存进 admin_approval_requests 并返回 202，由唯一的超级管理员在审批页一键通过后，以管理员身份
// 在进程内重放这条请求。这里只放领域类型、仓储端口与错误常量；流程见 admin_approval_service.go。

// 审批申请状态。pending → executing → approved | failed；pending → rejected | cancelled | expired。
const (
	ApprovalStatusPending   = "pending"
	ApprovalStatusExecuting = "executing"
	ApprovalStatusApproved  = "approved"
	ApprovalStatusFailed    = "failed"
	ApprovalStatusRejected  = "rejected"
	ApprovalStatusCancelled = "cancelled"
	ApprovalStatusExpired   = "expired"

	// ApprovalStatusFilterProcessed 列表筛选用的伪状态：除 pending / executing 之外的全部终态。
	ApprovalStatusFilterProcessed = "processed"
)

// 审批申请的目标类型，供列表展示与前端标签使用。
const (
	ApprovalTargetUser         = "user"
	ApprovalTargetUsersBatch   = "users_batch"
	ApprovalTargetSubscription = "subscription"
	ApprovalTargetAPIKey       = "api_key"
)

const (
	// AdminApprovalTTL 待审申请的有效期，过期后不可再通过。
	AdminApprovalTTL = 72 * time.Hour
	// AdminApprovalExecutingTimeout executing 状态卡住多久视为失败（重放中途进程崩溃等）。
	AdminApprovalExecutingTimeout = 10 * time.Minute
	// AdminApprovalPendingLimitPerUser 每个 operator 同时最多有多少条待审申请。
	AdminApprovalPendingLimitPerUser = 20
	// AdminApprovalCreateRateLimit / AdminApprovalCreateRateWindow 每个 operator 创建申请的频率上限。
	AdminApprovalCreateRateLimit  = 30
	AdminApprovalCreateRateWindow = time.Minute
	// AdminApprovalResultBodyMaxBytes 重放响应体入库上限（字节）。
	AdminApprovalResultBodyMaxBytes = 16 * 1024
	// AdminApprovalReplayTimeout 单次重放的总超时。
	AdminApprovalReplayTimeout = 60 * time.Second
	// AdminApprovalBatchLimit 批量通过单次最多处理的申请数。
	AdminApprovalBatchLimit = 50
)

var (
	ErrApprovalNotFound          = infraerrors.NotFound("APPROVAL_NOT_FOUND", "approval request not found")
	ErrApprovalNotPending        = infraerrors.Conflict("APPROVAL_NOT_PENDING", "approval request is not pending")
	ErrApprovalActionForbidden   = infraerrors.Forbidden("OPERATOR_ACTION_FORBIDDEN", "this action cannot be requested by an operator")
	ErrApprovalPendingLimit      = infraerrors.Conflict("APPROVAL_PENDING_LIMIT", "too many pending approval requests")
	ErrApprovalRateLimited       = infraerrors.TooManyRequests("APPROVAL_RATE_LIMITED", "too many approval requests, slow down")
	ErrApprovalBodyTooLarge      = infraerrors.New(413, "APPROVAL_BODY_TOO_LARGE", "request body too large for approval")
	ErrApprovalBodyNotJSON       = infraerrors.New(415, "APPROVAL_BODY_NOT_JSON", "only JSON request bodies can be queued for approval")
	ErrApprovalGateUnavailable   = infraerrors.ServiceUnavailable("APPROVAL_GATE_UNAVAILABLE", "approval gate is not available")
	ErrApprovalDispatcherMissing = infraerrors.ServiceUnavailable("APPROVAL_DISPATCHER_UNAVAILABLE", "approval executor is not available")
	ErrApprovalForbidden         = infraerrors.Forbidden("APPROVAL_FORBIDDEN", "not allowed to act on this approval request")
	ErrApprovalApproverInvalid   = infraerrors.Forbidden("APPROVAL_APPROVER_INVALID", "only an active admin session can decide approval requests")
	ErrApprovalBatchEmpty        = infraerrors.BadRequest("APPROVAL_BATCH_EMPTY", "no approval ids given")
	ErrApprovalBatchTooLarge     = infraerrors.BadRequest("APPROVAL_BATCH_TOO_LARGE", "too many approval ids in one batch")
)

// AdminApprovalRequest 一条待审 / 已决的 operator 写请求。
type AdminApprovalRequest struct {
	ID     int64
	Status string
	// Action 审计动作名（与审计中间件同一套推导），如 admin.users.balance.create。
	Action        string
	Method        string
	RouteTemplate string
	RequestPath   string
	RequestQuery  string
	ContentType   string
	// RequestBodyEnc 加密后的原始请求体（重放用）；RequestBodyRedacted 脱敏后的展示副本。
	RequestBodyEnc      string
	RequestBodyRedacted string
	RequestBodySHA256   string

	TargetType    string
	TargetID      *int64
	TargetSummary string

	RequesterUserID int64
	RequesterEmail  string
	RequesterIP     string
	RequestID       string

	DecidedByUserID *int64
	DecidedByEmail  string
	DecidedAt       *time.Time
	DecisionReason  string

	ExecutedAt       *time.Time
	ResultStatusCode *int
	ResultBody       string
	ResultError      string

	NotifiedAt *time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsPendingAt 报告申请在 now 时刻是否仍可被通过 / 拒绝 / 撤回。
func (r *AdminApprovalRequest) IsPendingAt(now time.Time) bool {
	return r != nil && r.Status == ApprovalStatusPending && r.ExpiresAt.After(now)
}

// EffectiveStatus 返回展示用状态：pending 但已过期的申请按 expired 展示（真正落库由 sweeper 完成）。
func (r *AdminApprovalRequest) EffectiveStatus(now time.Time) string {
	if r == nil {
		return ""
	}
	if r.Status == ApprovalStatusPending && !r.ExpiresAt.After(now) {
		return ApprovalStatusExpired
	}
	return r.Status
}

// AdminApprovalFilter 列表筛选。Status 为空表示全部，可为具体状态或 ApprovalStatusFilterProcessed。
type AdminApprovalFilter struct {
	Page            int
	PageSize        int
	Status          string
	RequesterUserID *int64
}

// AdminApprovalList 分页结果。
type AdminApprovalList struct {
	Items    []*AdminApprovalRequest
	Total    int64
	Page     int
	PageSize int
}

// AdminApprovalRepository 审批申请仓储端口（实现在 internal/repository/admin_approval_repo.go）。
type AdminApprovalRepository interface {
	Create(ctx context.Context, req *AdminApprovalRequest) (*AdminApprovalRequest, error)
	GetByID(ctx context.Context, id int64) (*AdminApprovalRequest, error)
	List(ctx context.Context, filter *AdminApprovalFilter) ([]*AdminApprovalRequest, int64, error)
	// CountPending 统计仍在有效期内的 pending 申请；requesterUserID 为 nil 表示全部。
	CountPending(ctx context.Context, now time.Time, requesterUserID *int64) (int64, error)
	// TransitionToExecuting 原子地把 pending 且未过期的申请置为 executing；
	// 状态不对或已过期返回 ErrApprovalNotPending，不存在返回 ErrApprovalNotFound。
	TransitionToExecuting(ctx context.Context, id, approverID int64, approverEmail string, now time.Time) (*AdminApprovalRequest, error)
	// FinishExecution 写入重放结果并把状态置为 approved / failed。
	FinishExecution(ctx context.Context, id int64, status string, statusCode int, body, errText string, now time.Time) error
	// Decide 把 pending 申请置为 rejected / cancelled；返回是否命中（未命中说明不存在或状态已变）。
	Decide(ctx context.Context, id int64, toStatus string, actorID int64, actorEmail, reason string, now time.Time) (bool, error)
	MarkNotified(ctx context.Context, id int64, now time.Time) error
	// ExpirePending 把已过期的 pending 申请置为 expired，返回行数。
	ExpirePending(ctx context.Context, now time.Time) (int64, error)
	// FailStuckExecuting 把 decided_at 早于 before 仍处于 executing 的申请置为 failed，返回行数。
	FailStuckExecuting(ctx context.Context, before time.Time) (int64, error)
}

// ApprovalNotifier 新申请落库后的推送钩子（Server酱³ 实现见 approval_notify_service.go）。
// fallbackOrigin 是捕获时请求的 scheme://host，站点未配置 frontend_url 时用于拼审批页链接。
type ApprovalNotifier interface {
	NotifyRequested(req *AdminApprovalRequest, fallbackOrigin string)
}
