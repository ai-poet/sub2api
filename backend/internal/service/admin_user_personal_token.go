package service

import (
	"context"
	"log/slog"
)

// SetPersonalTokenRevoker 注入个人令牌吊销器（fork 本地，见 ProvidePersonalTokenService）。
// 用 setter 而不是构造参数，保持上游 NewAdminService 签名不变。
func (s *adminServiceImpl) SetPersonalTokenRevoker(revoker PersonalTokenRevoker) {
	s.personalTokenRevoker = revoker
}

// revokePersonalTokenOnRoleChange 角色从可持有令牌的角色（operator）变成别的角色时吊销其个人令牌。
// 认证时本来就会实时校验角色，这里额外删除记录，保证再次提升为 operator 后旧令牌不会"复活"。
// 用户记录已经落库，吊销失败只记日志，不回滚角色变更。
func (s *adminServiceImpl) revokePersonalTokenOnRoleChange(ctx context.Context, userID int64, oldRole, newRole string) {
	if s.personalTokenRevoker == nil {
		return
	}
	if !IsPersonalTokenEligibleRole(oldRole) || IsPersonalTokenEligibleRole(newRole) {
		return
	}
	if err := s.personalTokenRevoker.RevokeForUser(ctx, userID); err != nil {
		slog.Error("personal token: revoke on role change failed", "user_id", userID, "old_role", oldRole, "new_role", newRole, "error", err)
	}
}
