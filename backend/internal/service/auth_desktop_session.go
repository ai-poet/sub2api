package service

import (
	"context"
	"errors"
)

// GenerateDesktopTokenPair 为通过浏览器桥接登录的原生桌面客户端签发一对 token。
//
// 这对 token 自成一个会话家族。若把浏览器自己的 token 原样交给桌面端，
// 同一个 refresh token 就会被两个客户端持有；而 RefreshTokenPair 是轮转制
// ——旧 token 一经使用立即失效——于是后刷新的一方会被拒绝，表现为「掉登录」。
//
// 不为桌面端的 token 记录会话绑定指纹：浏览器的 IP/UA 并不是桌面端的，
// 若绑定到浏览器指纹，开启会话绑定时桌面端的第一次刷新就会撤销整个家族。
// 首次刷新时会按「绑定功能开启前签发的旧会话」处理，用桌面端自己的指纹补齐。
func (s *AuthService) GenerateDesktopTokenPair(ctx context.Context, userID int64) (*TokenPair, error) {
	result, err := s.GenerateDesktopTokenPairWithUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &result.TokenPair, nil
}

// GenerateDesktopTokenPairWithUser 与 GenerateDesktopTokenPair 相同，另带回用户角色。
// 客户端登录码的兑换接口是公开路由，没有 JWT 中间件替它做后台模式判断，
// 需要凭角色自己判断（与 RefreshToken 一致）。
func (s *AuthService) GenerateDesktopTokenPairWithUser(ctx context.Context, userID int64) (*TokenPairWithUser, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if !user.IsActive() {
		return nil, ErrUserNotActive
	}
	pair, err := s.GenerateTokenPair(withoutSessionBinding(ctx), user, "")
	if err != nil {
		return nil, err
	}
	return &TokenPairWithUser{TokenPair: *pair, UserRole: user.Role}, nil
}

// withoutSessionBinding 返回携带空会话指纹的 ctx，使其下签发的 token 不带绑定哈希。
func withoutSessionBinding(ctx context.Context) context.Context {
	return context.WithValue(ctx, sessionBindingCtxKey{}, &SessionBinding{})
}
