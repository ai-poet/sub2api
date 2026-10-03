package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserSiteMessageHandler 用户管理里的站内信（fork 本地功能）。
//
// 管理员直接发送；运维管理员的 POST 被 middleware/console_scope.go 的 operatorApprovalScope 截获，
// 管理员批准后以管理员身份重放到这里——此时从 ApprovalReplayFromContext 取回发起的运维，记为发件人。
// 运维可以直接读历史（operatorReadScope）。
type UserSiteMessageHandler struct {
	svc *service.SiteMessageService
}

// NewUserSiteMessageHandler 构造用户管理站内信 handler。
func NewUserSiteMessageHandler(svc *service.SiteMessageService) *UserSiteMessageHandler {
	return &UserSiteMessageHandler{svc: svc}
}

func parseSiteMessageRecipientID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return 0, false
	}
	return id, true
}

// Send POST /api/v1/admin/users/:id/site-messages
func (h *UserSiteMessageHandler) Send(c *gin.Context) {
	userID, ok := parseSiteMessageRecipientID(c)
	if !ok {
		return
	}
	var req dto.SendSiteMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	in := service.SiteMessageSendInput{
		RecipientUserID: userID,
		Title:           req.Title,
		Content:         req.Content,
	}
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		in.SenderUserID = subject.UserID
	}
	if role, ok := middleware.GetUserRoleFromContext(c); ok {
		in.SenderRole = role
	}
	// 审批重放：handler 以批准的管理员身份运行，发件人记回发起的运维管理员
	if replay, ok := service.ApprovalReplayFromContext(c.Request.Context()); ok && replay.RequesterUserID > 0 {
		in.SenderUserID = replay.RequesterUserID
		in.SenderRole = service.RoleOperator
		approvalID := replay.ApprovalID
		in.ApprovalID = &approvalID
	}

	idempotencyPayload := struct {
		UserID int64                      `json:"user_id"`
		Body   dto.SendSiteMessageRequest `json:"body"`
	}{
		UserID: userID,
		Body:   req,
	}
	executeAdminIdempotentJSON(c, "admin.users.site_messages.create", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		msg, err := h.svc.SendFromStaff(ctx, in)
		if err != nil {
			return nil, err
		}
		middleware.SetAuditExtra(c, map[string]any{"site_message_id": msg.ID})
		return dto.AdminSiteMessageFromService(msg), nil
	})
}

// List GET /api/v1/admin/users/:id/site-messages?page&page_size
func (h *UserSiteMessageHandler) List(c *gin.Context) {
	userID, ok := parseSiteMessageRecipientID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	list, err := h.svc.ListForUserAsStaff(c.Request.Context(), userID, &service.SiteMessageFilter{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.AdminSiteMessage, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, *dto.AdminSiteMessageFromService(item))
	}
	response.Paginated(c, items, list.Total, list.Page, list.PageSize)
}
