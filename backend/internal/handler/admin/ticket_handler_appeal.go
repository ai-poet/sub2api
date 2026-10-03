package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ticketUserLookup 读发起人当前账号状态（实现是 *service.UserService）。
type ticketUserLookup interface {
	GetByID(ctx context.Context, id int64) (*service.User, error)
}

// ProvideTicketHandler 构造客服侧工单 handler 并挂上用户查询（fork：申诉工单详情显示账号状态）。
func ProvideTicketHandler(svc *service.TicketService, users *service.UserService) *TicketHandler {
	h := NewTicketHandler(svc)
	if users != nil {
		h.users = users
	}
	return h
}

// withUserStatus 给工单详情补上发起人当前账号状态；查询失败时不影响详情返回。
func (h *TicketHandler) withUserStatus(ctx context.Context, ticket *dto.AdminSupportTicket) *dto.AdminSupportTicket {
	if h == nil || h.users == nil || ticket == nil || ticket.User.ID <= 0 {
		return ticket
	}
	if user, err := h.users.GetByID(ctx, ticket.User.ID); err == nil && user != nil {
		ticket.User.Status = user.Status
	}
	return ticket
}
