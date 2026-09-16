package admin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ApprovalHandler 运维管理员写操作审批（fork 本地功能）：
// 管理员看全部并一键通过 / 拒绝；运维管理员只能看到并撤回自己的申请。
type ApprovalHandler struct {
	svc *service.AdminApprovalService
}

// NewApprovalHandler 构造审批 handler。
func NewApprovalHandler(svc *service.AdminApprovalService) *ApprovalHandler {
	return &ApprovalHandler{svc: svc}
}

func approvalActorFromContext(c *gin.Context) service.ApprovalActor {
	actor := service.ApprovalActor{
		AuthMethod:     c.GetString("auth_method"),
		ClientIP:       middleware.SecurityClientIP(c),
		AcceptLanguage: c.GetHeader("Accept-Language"),
		Email:          c.GetString(middleware.ContextKeyAuthEmail),
		SessionID:      c.GetString(middleware.ContextKeySessionID),
	}
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		actor.UserID = subject.UserID
		actor.Concurrency = subject.Concurrency
	}
	if role, ok := middleware.GetUserRoleFromContext(c); ok {
		actor.Role = role
	}
	if requestID, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
		actor.RequestID = requestID
	}
	return actor
}

func approvalView(req *service.AdminApprovalRequest, operator bool) *dto.AdminApprovalRequest {
	if operator {
		return dto.AdminApprovalRequestFromServiceOperator(req)
	}
	return dto.AdminApprovalRequestFromService(req)
}

func parseApprovalID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid approval id")
		return 0, false
	}
	return id, true
}

// List GET /api/v1/admin/approvals?status=pending|processed|<status>&page&page_size
// 运维管理员只能看到自己发起的申请。
func (h *ApprovalHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	actor := approvalActorFromContext(c)
	operator := middleware.IsOperatorRequest(c)

	filter := &service.AdminApprovalFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   strings.TrimSpace(c.Query("status")),
	}
	if operator {
		uid := actor.UserID
		filter.RequesterUserID = &uid
	}

	list, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.AdminApprovalRequest, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, *approvalView(item, operator))
	}
	response.Paginated(c, items, list.Total, list.Page, list.PageSize)
}

// PendingCount GET /api/v1/admin/approvals/pending-count
// 管理员：全站待审数（侧边栏角标）；运维管理员：自己的待审数。
func (h *ApprovalHandler) PendingCount(c *gin.Context) {
	var requester *int64
	if middleware.IsOperatorRequest(c) {
		uid := approvalActorFromContext(c).UserID
		requester = &uid
	}
	pending, err := h.svc.PendingCount(c.Request.Context(), requester)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ApprovalPendingCountResponse{Pending: pending})
}

// Get GET /api/v1/admin/approvals/:id
func (h *ApprovalHandler) Get(c *gin.Context) {
	id, ok := parseApprovalID(c)
	if !ok {
		return
	}
	actor := approvalActorFromContext(c)
	req, err := h.svc.Get(c.Request.Context(), id, actor)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, approvalView(req, middleware.IsOperatorRequest(c)))
}

// Approve POST /api/v1/admin/approvals/:id/approve（管理员）
// 一键通过：以审批人身份内部重放原始请求，响应里附带重放的 HTTP 状态。
func (h *ApprovalHandler) Approve(c *gin.Context) {
	id, ok := parseApprovalID(c)
	if !ok {
		return
	}
	actor := approvalActorFromContext(c)
	req, replay, err := h.svc.Approve(c.Request.Context(), id, actor)
	extra := map[string]any{"approval_id": id}
	if req != nil {
		extra["approval_status"] = req.Status
	}
	if replay != nil {
		extra["http_status"] = replay.StatusCode
	}
	middleware.SetAuditExtra(c, extra)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := dto.ApprovalDecisionResponse{Approval: dto.AdminApprovalRequestFromService(req)}
	if replay != nil {
		out.Replay = &dto.ApprovalReplayResult{StatusCode: replay.StatusCode, DurationMs: replay.DurationMs}
	}
	response.Success(c, out)
}

// BatchApprove POST /api/v1/admin/approvals/batch-approve {ids}（管理员）
// 一键通过多条：逐条重放，单条失败不影响其它；返回每条的结果与汇总。
func (h *ApprovalHandler) BatchApprove(c *gin.Context) {
	var body dto.BatchApproveRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	actor := approvalActorFromContext(c)
	results, err := h.svc.ApproveBatch(c.Request.Context(), body.IDs, actor)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := dto.ApprovalBatchResponse{Results: make([]dto.ApprovalBatchItem, 0, len(results))}
	for _, item := range results {
		switch item.Status {
		case service.ApprovalStatusApproved:
			out.Approved++
		case service.ApprovalStatusFailed:
			out.Failed++
		default:
			out.Skipped++
		}
		out.Results = append(out.Results, dto.ApprovalBatchItem{ID: item.ID, Status: item.Status, Error: item.Error, HTTPStatus: item.StatusCode})
	}
	middleware.SetAuditExtra(c, map[string]any{
		"requested_count": len(body.IDs),
		"result":          fmt.Sprintf("approved=%d failed=%d skipped=%d", out.Approved, out.Failed, out.Skipped),
	})
	response.Success(c, out)
}

// Reject POST /api/v1/admin/approvals/:id/reject {reason}（管理员）
func (h *ApprovalHandler) Reject(c *gin.Context) {
	id, ok := parseApprovalID(c)
	if !ok {
		return
	}
	var body dto.RejectApprovalRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			response.BadRequest(c, "Invalid request body")
			return
		}
	}
	actor := approvalActorFromContext(c)
	req, err := h.svc.Reject(c.Request.Context(), id, actor, body.Reason)
	middleware.SetAuditExtra(c, map[string]any{"approval_id": id, "approval_status": service.ApprovalStatusRejected})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ApprovalDecisionResponse{Approval: dto.AdminApprovalRequestFromService(req)})
}

// Cancel POST /api/v1/admin/approvals/:id/cancel（申请人本人或管理员）
func (h *ApprovalHandler) Cancel(c *gin.Context) {
	id, ok := parseApprovalID(c)
	if !ok {
		return
	}
	actor := approvalActorFromContext(c)
	req, err := h.svc.Cancel(c.Request.Context(), id, actor)
	middleware.SetAuditExtra(c, map[string]any{"approval_id": id, "approval_status": service.ApprovalStatusCancelled})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ApprovalDecisionResponse{Approval: approvalView(req, middleware.IsOperatorRequest(c))})
}
