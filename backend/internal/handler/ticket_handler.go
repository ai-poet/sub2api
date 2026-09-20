package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// TicketHandler 用户侧工单接口（fork 本地功能）：只能操作自己的工单。
type TicketHandler struct {
	svc *service.TicketService
}

// NewTicketHandler 构造用户侧工单 handler。
func NewTicketHandler(svc *service.TicketService) *TicketHandler {
	return &TicketHandler{svc: svc}
}

func ticketActorFromContext(c *gin.Context) (service.TicketActor, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		return service.TicketActor{}, false
	}
	actor := service.TicketActor{
		UserID: subject.UserID,
		Email:  c.GetString(middleware2.ContextKeyAuthEmail),
	}
	if role, ok := middleware2.GetUserRoleFromContext(c); ok {
		actor.Role = role
	}
	return actor, true
}

func parseTicketID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ticket id")
		return 0, false
	}
	return id, true
}

// ticketRequestOrigin 请求的 scheme://host，仅用于站点未配置 frontend_url 时拼推送链接。
func ticketRequestOrigin(c *gin.Context) string {
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

// Create POST /api/v1/tickets
func (h *TicketHandler) Create(c *gin.Context) {
	actor, ok := ticketActorFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var body dto.CreateTicketRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	ticket, err := h.svc.Create(c.Request.Context(), actor, service.TicketCreateInput{
		Title:    body.Title,
		Category: body.Category,
		Body:     body.Body,
	}, ticketRequestOrigin(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, dto.SupportTicketFromService(ticket))
}

// List GET /api/v1/tickets?status&category&page&page_size
func (h *TicketHandler) List(c *gin.Context) {
	actor, ok := ticketActorFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	list, err := h.svc.ListMine(c.Request.Context(), actor.UserID, &service.TicketFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Category: c.Query("category"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.SupportTicket, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, *dto.SupportTicketFromService(item))
	}
	response.Paginated(c, items, list.Total, list.Page, list.PageSize)
}

// UnreadCount GET /api/v1/tickets/unread-count（侧边栏角标）
func (h *TicketHandler) UnreadCount(c *gin.Context) {
	actor, ok := ticketActorFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	count, err := h.svc.UnreadCount(c.Request.Context(), actor.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.TicketCountResponse{Count: count})
}

// Get GET /api/v1/tickets/:id（打开即清未读）
func (h *TicketHandler) Get(c *gin.Context) {
	actor, ok := ticketActorFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	ticket, msgs, err := h.svc.GetForUser(c.Request.Context(), id, actor.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SupportTicketDetail{
		Ticket:   dto.SupportTicketFromService(ticket),
		Messages: dto.SupportTicketMessagesForUser(msgs),
	})
}

// Reply POST /api/v1/tickets/:id/messages
func (h *TicketHandler) Reply(c *gin.Context) {
	actor, ok := ticketActorFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	var body dto.ReplyTicketRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	ticket, msg, err := h.svc.ReplyAsUser(c.Request.Context(), id, actor, body.Body, ticketRequestOrigin(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := dto.SupportTicketMessageForUser(msg)
	response.Success(c, dto.TicketReplyResponse{
		Ticket:  dto.SupportTicketFromService(ticket),
		Message: &out,
	})
}

// Close POST /api/v1/tickets/:id/close
func (h *TicketHandler) Close(c *gin.Context) {
	actor, ok := ticketActorFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	ticket, err := h.svc.CloseAsUser(c.Request.Context(), id, actor)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SupportTicketFromService(ticket))
}

// Reopen POST /api/v1/tickets/:id/reopen
func (h *TicketHandler) Reopen(c *gin.Context) {
	actor, ok := ticketActorFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseTicketID(c)
	if !ok {
		return
	}
	ticket, err := h.svc.ReopenAsUser(c.Request.Context(), id, actor)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SupportTicketFromService(ticket))
}
