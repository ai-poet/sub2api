package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 运维管理员写操作审批的响应结构（fork 本地功能）。

// AdminApprovalActorRef 申请人 / 决策人引用。
type AdminApprovalActorRef struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

// AdminApprovalRequest 审批申请视图。RequestBody 永远是脱敏副本；原始 body 只在服务端解密重放。
type AdminApprovalRequest struct {
	ID            int64  `json:"id"`
	Status        string `json:"status"`
	Action        string `json:"action"`
	Method        string `json:"method"`
	RouteTemplate string `json:"route_template"`
	RequestPath   string `json:"request_path"`
	RequestQuery  string `json:"request_query,omitempty"`

	TargetType    string `json:"target_type"`
	TargetID      *int64 `json:"target_id,omitempty"`
	TargetSummary string `json:"target_summary"`

	Requester   AdminApprovalActorRef `json:"requester"`
	RequesterIP string                `json:"requester_ip,omitempty"`
	RequestBody string                `json:"request_body"`

	DecidedBy      *AdminApprovalActorRef `json:"decided_by,omitempty"`
	DecidedAt      *time.Time             `json:"decided_at,omitempty"`
	DecisionReason string                 `json:"decision_reason,omitempty"`

	ExecutedAt       *time.Time `json:"executed_at,omitempty"`
	ResultStatusCode *int       `json:"result_status_code,omitempty"`
	ResultBody       string     `json:"result_body,omitempty"`
	ResultError      string     `json:"result_error,omitempty"`

	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminApprovalRequestFromService 管理员视图（完整）。
func AdminApprovalRequestFromService(r *service.AdminApprovalRequest) *AdminApprovalRequest {
	if r == nil {
		return nil
	}
	out := &AdminApprovalRequest{
		ID:               r.ID,
		Status:           r.Status,
		Action:           r.Action,
		Method:           r.Method,
		RouteTemplate:    r.RouteTemplate,
		RequestPath:      r.RequestPath,
		RequestQuery:     r.RequestQuery,
		TargetType:       r.TargetType,
		TargetID:         r.TargetID,
		TargetSummary:    r.TargetSummary,
		Requester:        AdminApprovalActorRef{ID: r.RequesterUserID, Email: r.RequesterEmail},
		RequesterIP:      r.RequesterIP,
		RequestBody:      r.RequestBodyRedacted,
		DecidedAt:        r.DecidedAt,
		DecisionReason:   r.DecisionReason,
		ExecutedAt:       r.ExecutedAt,
		ResultStatusCode: r.ResultStatusCode,
		ResultBody:       r.ResultBody,
		ResultError:      r.ResultError,
		ExpiresAt:        r.ExpiresAt,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
	if r.DecidedByUserID != nil {
		out.DecidedBy = &AdminApprovalActorRef{ID: *r.DecidedByUserID, Email: r.DecidedByEmail}
	}
	return out
}

// AdminApprovalRequestFromServiceOperator 运维管理员视图：去掉申请人 IP 与重放响应体（可能含下游对象的完整字段）。
func AdminApprovalRequestFromServiceOperator(r *service.AdminApprovalRequest) *AdminApprovalRequest {
	out := AdminApprovalRequestFromService(r)
	if out == nil {
		return nil
	}
	out.RequesterIP = ""
	out.ResultBody = ""
	return out
}

// ApprovalReplayResult 重放结果摘要。
type ApprovalReplayResult struct {
	StatusCode int   `json:"status_code"`
	DurationMs int64 `json:"duration_ms"`
}

// ApprovalDecisionResponse 通过 / 拒绝 / 撤回的响应。
type ApprovalDecisionResponse struct {
	Approval *AdminApprovalRequest `json:"approval"`
	Replay   *ApprovalReplayResult `json:"replay,omitempty"`
}

// ApprovalPendingCountResponse 待审数量。
type ApprovalPendingCountResponse struct {
	Pending int64 `json:"pending"`
}

// RejectApprovalRequest 拒绝申请的请求体。
type RejectApprovalRequest struct {
	Reason string `json:"reason" binding:"max=500"`
}

// BatchApproveRequest 批量通过的请求体（最多 50 条）。
type BatchApproveRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,max=50,dive,gt=0"`
}

// ApprovalBatchItem 批量通过里单条的结果。
type ApprovalBatchItem struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

// ApprovalBatchResponse 批量通过的汇总。
type ApprovalBatchResponse struct {
	Results  []ApprovalBatchItem `json:"results"`
	Approved int                 `json:"approved"`
	Failed   int                 `json:"failed"`
	Skipped  int                 `json:"skipped"`
}
