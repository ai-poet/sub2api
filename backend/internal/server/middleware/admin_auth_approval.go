package middleware

import (
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 运维管理员（operator）写操作审批在认证层的两个出口（fork 本地，见 docs/OPERATOR_ROLE.md「审批」一节）：
//   - 捕获：operator 命中审批范围的写请求被交给 AdminApprovalGate 入队，直接返回 202 并 abort；
//   - 重放：管理员一键通过后，审批服务在进程内以带 ApprovalReplay 标记的 request context 重放该请求，
//     adminAuth 识别标记后直接以审批人身份放行。
// 两条路径都不经过后面的审计中间件（认证层 abort / 标记分支正常 Next），捕获这一侧在此直接写审计。

// applyApprovalReplayIdentity 以审批人身份放行重放请求。标记只能由 Go 代码通过
// service.WithApprovalReplay 设置，HTTP 头无法伪造；角色不是 admin 一律拒绝（防御性）。
func applyApprovalReplayIdentity(c *gin.Context, replay *service.ApprovalReplay) bool {
	if replay == nil || replay.ApproverUserID <= 0 || replay.ApproverRole != service.RoleAdmin {
		AbortWithError(c, 403, "FORBIDDEN", "Admin access required")
		return false
	}
	c.Set(string(ContextKeyUser), AuthSubject{
		UserID:      replay.ApproverUserID,
		Concurrency: replay.ApproverConcurrency,
	})
	c.Set(string(ContextKeyUserRole), replay.ApproverRole)
	c.Set(ContextKeyAuthEmail, replay.ApproverEmail)
	c.Set(ContextKeySessionID, replay.ApproverSessionID)
	c.Set("auth_method", service.AuditAuthMethodApprovalReplay)
	SetAuditExtra(c, map[string]any{
		"approval_id":           replay.ApprovalID,
		"approval_requester_id": replay.RequesterUserID,
	})
	return true
}

// auditActionFor 复用审计中间件的动作名规则：精确覆写优先，否则按路由模板推导。
func auditActionFor(method, fullPath string) string {
	if action, ok := auditActionOverrides[method+" "+fullPath]; ok {
		return action
	}
	return deriveAuditAction(method, fullPath)
}

// captureOperatorApproval 把 operator 的写请求交给审批服务入队：成功返回 202 并 abort。
// gate 未注入（wire 漏配 / 测试构造）时返回 503 —— fail-closed，绝不放行到 handler。
func captureOperatorApproval(c *gin.Context, gate service.AdminApprovalGate, auditService *service.AuditLogService, user *service.User) {
	if gate == nil {
		recordOperatorApprovalAudit(c, auditService, user, nil, nil, http.StatusServiceUnavailable, "APPROVAL_GATE_UNAVAILABLE")
		AbortWithError(c, http.StatusServiceUnavailable, "APPROVAL_GATE_UNAVAILABLE", "Approval gate is not available")
		return
	}

	body, err := readApprovalBody(c)
	if err != nil {
		recordOperatorApprovalAudit(c, auditService, user, nil, nil, http.StatusRequestEntityTooLarge, "APPROVAL_BODY_TOO_LARGE")
		AbortWithError(c, http.StatusRequestEntityTooLarge, "APPROVAL_BODY_TOO_LARGE", "Request body too large for approval")
		return
	}

	params := make(map[string]string, len(c.Params))
	for _, p := range c.Params {
		params[p.Key] = p.Value
	}
	requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
	in := &service.AdminApprovalCaptureInput{
		Method:          c.Request.Method,
		RouteTemplate:   c.FullPath(),
		Path:            c.Request.URL.Path,
		RawQuery:        c.Request.URL.RawQuery,
		Params:          params,
		ContentType:     c.GetHeader("Content-Type"),
		Action:          auditActionFor(c.Request.Method, c.FullPath()),
		Body:            body,
		RequesterUserID: user.ID,
		RequesterEmail:  user.Email,
		RequesterIP:     SecurityClientIP(c),
		RequestID:       requestID,
		RequestOrigin:   requestOrigin(c),
	}

	req, err := gate.Capture(c.Request.Context(), in)
	if err != nil {
		status, reason, message := approvalErrorToHTTP(err)
		recordOperatorApprovalAudit(c, auditService, user, nil, body, status, reason)
		AbortWithError(c, status, reason, message)
		return
	}

	recordOperatorApprovalAudit(c, auditService, user, req, body, http.StatusAccepted, "")
	c.JSON(http.StatusAccepted, gin.H{
		"code":    0,
		"message": "accepted",
		"data": gin.H{
			"approval_request_id": req.ID,
			"status":              req.Status,
			"action":              req.Action,
			"target_summary":      req.TargetSummary,
			"expires_at":          req.ExpiresAt,
		},
	})
	c.Abort()
}

// readApprovalBody 读取完整请求体；超过审计捕获上限视为错误（请求已在此终止，不需要回填 body）。
func readApprovalBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, service.AuditRequestBodyCaptureLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > service.AuditRequestBodyCaptureLimit {
		return nil, service.ErrApprovalBodyTooLarge
	}
	return raw, nil
}

// approvalErrorToHTTP 把审批服务的错误映射成 (状态码, 错误码, 文案)；未知错误统一 500 且不泄露内部信息。
func approvalErrorToHTTP(err error) (int, string, string) {
	status := infraerrors.Code(err)
	if status < 400 || status > 599 {
		return http.StatusInternalServerError, "APPROVAL_CAPTURE_FAILED", "Failed to queue approval request"
	}
	reason := infraerrors.Reason(err)
	if reason == "" {
		reason = "APPROVAL_CAPTURE_FAILED"
	}
	message := infraerrors.Message(err)
	if message == "" {
		message = "Failed to queue approval request"
	}
	return status, reason, message
}

// requestOrigin 捕获时请求的 scheme://host，仅用于站点未配置 frontend_url 时拼推送链接。
func requestOrigin(c *gin.Context) string {
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}
	return scheme + "://" + host
}

// recordOperatorApprovalAudit 记录捕获结果：202 记为 admin.approval.requested，其余记为 admin.approval.refused。
// 审计中间件挂在认证之后，看不到这里的响应，所以直接落库（同 recordOperatorScopeDenied）。
func recordOperatorApprovalAudit(
	c *gin.Context,
	auditService *service.AuditLogService,
	user *service.User,
	req *service.AdminApprovalRequest,
	rawBody []byte,
	status int,
	errorCode string,
) {
	if c == nil || auditService == nil || user == nil {
		return
	}
	uid := user.ID
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	action := service.AuditActionAdminApprovalRequested
	if status != http.StatusAccepted {
		action = service.AuditActionAdminApprovalRefused
	}
	entry := &service.AuditLog{
		ActorUserID: &uid,
		ActorEmail:  user.Email,
		ActorRole:   user.Role,
		AuthMethod:  service.AuditAuthMethodJWT,
		Action:      action,
		Method:      c.Request.Method,
		Path:        path,
		ClientIP:    SecurityClientIP(c),
		UserAgent:   normalizePersistentText(c.Request.UserAgent(), maxPersistentUserAgentBytes),
		RequestBody: service.RedactAuditBody(rawBody, c.GetHeader("Content-Type")),
		StatusCode:  status,
	}
	if requestID, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
		entry.RequestID = requestID
	}
	extra := map[string]any{}
	if req != nil {
		extra["approval_id"] = req.ID
		extra["approval_status"] = req.Status
	}
	if errorCode != "" {
		extra["error_code"] = errorCode
	}
	if q := service.RedactAuditQuery(c.Request.URL.RawQuery); q != "" {
		extra["query"] = q
	}
	if len(extra) > 0 {
		entry.Extra = extra
	}
	auditService.Record(entry)
}
