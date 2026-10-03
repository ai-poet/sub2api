package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AppealHandler 封禁申诉会话自己的接口（fork 本地，见 service/appeal.go）。
// 站内信 / 工单列表 / 详情 / 回复 / 附件在 /api/v1/appeal/* 下直接复用用户侧 handler。
type AppealHandler struct {
	svc *service.AppealService
}

// NewAppealHandler 构造申诉 handler。
func NewAppealHandler(svc *service.AppealService) *AppealHandler {
	return &AppealHandler{svc: svc}
}

// Authenticator 供 AppealAuth 中间件使用。
func (h *AppealHandler) Authenticator() middleware2.AppealAuthenticator {
	if h == nil || h.svc == nil {
		return nil
	}
	return h.svc
}

// Session GET /api/v1/appeal/session
func (h *AppealHandler) Session(c *gin.Context) {
	session, ok := middleware2.GetAppealSessionFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrAppealTokenInvalid)
		return
	}
	active, err := h.svc.HasActiveAppealTicket(c.Request.Context(), session.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AppealSession{
		Email:           session.Email,
		Status:          service.StatusDisabled,
		ExpiresAt:       session.ExpiresAt,
		HasActiveAppeal: active,
	})
}

// Logout POST /api/v1/appeal/logout：作废当前申诉令牌。
func (h *AppealHandler) Logout(c *gin.Context) {
	if err := h.svc.Revoke(c.Request.Context(), strings.TrimSpace(c.GetHeader(service.AppealTokenHeader))); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"logged_out": true})
}

// CreateTicket POST /api/v1/appeal/tickets：提交申诉工单（分类固定 appeal，每人最多一个未关闭的）。
func (h *AppealHandler) CreateTicket(c *gin.Context) {
	session, ok := middleware2.GetAppealSessionFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrAppealTokenInvalid)
		return
	}
	var req dto.CreateAppealTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	ticket, err := h.svc.CreateTicket(c.Request.Context(), session, req.Title, req.Body, ticketRequestOrigin(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware2.SetAuditExtra(c, map[string]any{"ticket_id": ticket.ID})
	response.Created(c, dto.SupportTicketFromService(ticket))
}
