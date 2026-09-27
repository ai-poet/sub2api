package service

import "context"

// IsPersonalTokenEnabled 运维管理员个人令牌总开关（fork 本地功能，默认关闭）。
// 关闭时所有令牌立即不可用（不删除），重新打开即恢复。读失败按关闭处理（fail-closed）。
func (s *SettingService) IsPersonalTokenEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPersonalTokenEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
