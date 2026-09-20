package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// TicketHandler 客服侧工单接口（fork 本地功能）：管理员与运维管理员同权，看全部并回复。
// 运维管理员的写操作在 middleware/console_scope.go 的 operatorWriteScope 里显式放行（不走审批），全部留审计。
type TicketHandler struct {
	svc *service.TicketService
}

// NewTicketHandler 构造客服侧工单 handler。
func NewTicketHandler(svc *service.TicketService) *TicketHandler {
	return &TicketHandler{svc: svc}
}

func ticketActorFromContext(c *gin.Context) service.TicketActor {
	actor := service.TicketActor{
		Email: c.GetString(middleware.ContextKeyAuthEmail),
	}
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		actor.UserID = subject.UserID
	}
	if role, ok := middleware.GetUserRoleFromContext(c); ok {
		actor.Role = role
	}
	return actor
}

func parseTicketID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ticket id")
		return 0, false
	}
	return id, true
}

func setTicketAuditExtra(c *gin.Context, id int64, ticket *service.SupportTicket) {
	extra := map[string]any{"ticket_id": id}
	if ticket != nil {
		extra["ticket_status"] = ticket.Status
		extra["ticket_user_id"] = ticket.UserID
	}
	middleware.SetAuditExtra(c, extra)
}

// List GET /api/v1/admin/tickets?status&category&search&page&page_size
func (h *TicketHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	list, err := h.svc.ListAll(c.Request.Context(), ticketActorFromContext(c), &service.TicketFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Category: c.Query("category"),
		Search:   c.Query("search"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.AdminSupportTicket, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, *dto.AdminSupportTicketFromService(item))
	}
	response.Paginated(c, items, list.Total, list.Page, list.PageSize)
}

// OpenCount GET /api/v1/admin/tickets/open-count（侧边栏角标：待处理数）
func (h *TicketHandler) OpenCount(c *gin.Context) {
	count, err := h.svc.OpenCount(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketCountResponse{Count: count})
}

// Get GET /api/v1/admin/tickets/:id
func (h *TicketHandler) Get(c *gin.Context) {
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	ticket, msgs, err := h.svc.GetForStaff(c.Request.Context(), id, ticketActorFromContext(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminSupportTicketDetail{
		Ticket:   dto.AdminSupportTicketFromService(ticket),
		Messages: dto.SupportTicketMessagesForStaff(msgs),
	})
}

// Reply POST /api/v1/admin/tickets/:id/messages
func (h *TicketHandler) Reply(c *gin.Context) {
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	var body dto.ReplyTicketRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	ticket, msg, err := h.svc.ReplyAsStaff(c.Request.Context(), id, ticketActorFromContext(c), body.Body)
	setTicketAuditExtra(c, id, ticket)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := dto.SupportTicketMessageForStaff(msg)
	response.Success(c, dto.AdminTicketReplyResponse{
		Ticket:  dto.AdminSupportTicketFromService(ticket),
		Message: &out,
	})
}

// Close POST /api/v1/admin/tickets/:id/close
func (h *TicketHandler) Close(c *gin.Context) {
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	ticket, err := h.svc.CloseAsStaff(c.Request.Context(), id, ticketActorFromContext(c))
	setTicketAuditExtra(c, id, ticket)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminSupportTicketFromService(ticket))
}

// Reopen POST /api/v1/admin/tickets/:id/reopen
func (h *TicketHandler) Reopen(c *gin.Context) {
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	ticket, err := h.svc.ReopenAsStaff(c.Request.Context(), id, ticketActorFromContext(c))
	setTicketAuditExtra(c, id, ticket)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AdminSupportTicketFromService(ticket))
}
