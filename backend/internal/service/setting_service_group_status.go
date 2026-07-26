package service

import (
	"context"
	"errors"
)

// IsGroupStatusEnabled 读取分组运行状态功能开关（本 fork 自有功能）。
func (s *SettingService) IsGroupStatusEnabled(ctx context.Context) (bool, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyGroupStatusEnabled)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return false, nil
		}
		return false, err
	}
	return value == "true", nil
}
