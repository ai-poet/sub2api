package service

import (
	"context"
	"fmt"
)

// CreateAppeal 封禁申诉工单（fork 本地，见 appeal.go）：只由申诉会话调用。
//
// 分类固定为 appeal，发起方角色固定为 user；每个用户同时最多一个未关闭的申诉工单，
// 不占用普通工单「最多 5 个未关闭」的配额（被封账号可能还留着旧工单）。
func (s *TicketService) CreateAppeal(ctx context.Context, actor TicketActor, title, body, requestOrigin string) (*SupportTicket, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if actor.UserID <= 0 {
		return nil, ErrTicketForbidden
	}
	actor.Role = TicketAuthorRoleUser
	normalizedTitle, err := normalizeTicketTitle(title)
	if err != nil {
		return nil, err
	}
	normalizedBody, err := normalizeTicketBody(body)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.CountActiveByUserCategory(ctx, actor.UserID, TicketCategoryAppeal)
	if err != nil {
		return nil, fmt.Errorf("count active appeal tickets: %w", err)
	}
	if active > 0 {
		return nil, ErrTicketAppealActive
	}
	return s.createTicket(ctx, actor, normalizedTitle, TicketCategoryAppeal, normalizedBody, requestOrigin)
}
