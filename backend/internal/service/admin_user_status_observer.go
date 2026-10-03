package service

import "context"

// 账号状态变更通知（fork 本地，见 appeal.go）。
//
// 上游 admin_user.go 里只有一行 notifyUserStatusChanged 调用、admin_service.go 里一个字段；
// 运维经审批修改用户状态时，重放同样走到这里。

// SetUserStatusObserver 注入状态观察者（由 ProvideAppealService 注入）。
func (s *adminServiceImpl) SetUserStatusObserver(o UserStatusObserver) {
	s.userStatusObserver = o
}

func (s *adminServiceImpl) notifyUserStatusChanged(ctx context.Context, userID int64, oldStatus, newStatus string) {
	if s == nil || s.userStatusObserver == nil || oldStatus == newStatus {
		return
	}
	s.userStatusObserver.OnUserStatusChanged(ctx, userID, oldStatus, newStatus)
}

// SetUserStatusObserver 注入状态观察者：风控中心「解封」时通知（由 ProvideAppealService 注入）。
func (s *ContentModerationService) SetUserStatusObserver(o UserStatusObserver) {
	if s == nil {
		return
	}
	s.userStatusObserver = o
}

func (s *ContentModerationService) notifyUserStatusChanged(ctx context.Context, userID int64, oldStatus, newStatus string) {
	if s == nil || s.userStatusObserver == nil || oldStatus == newStatus {
		return
	}
	s.userStatusObserver.OnUserStatusChanged(ctx, userID, oldStatus, newStatus)
}
