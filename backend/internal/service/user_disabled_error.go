package service

import "errors"

// UserDisabledError 「身份已验证、但账号被禁用」（fork 本地，封禁申诉会话用）。
//
// 它 Unwrap 成 ErrUserNotActive，所以 errors.Is、infraerrors.FromError / Reason / Message、
// 现有的 403 响应与第三方登录重定向都和原来一样；只有知道它的地方（handler/auth_appeal.go）
// 能从里面取出用户，给申诉会话签发令牌。
//
// 只能在凭证已经验证过的地方构造（密码校验之后、2FA 之后、Passkey 断言之后、第三方回调之后），
// 绝不能在先查状态后验密码的路径上用（例如 ValidatePasswordCredentials），否则会泄露账号状态。
type UserDisabledError struct {
	UserID int64
	Email  string
}

func (e *UserDisabledError) Error() string { return ErrUserNotActive.Error() }

func (e *UserDisabledError) Unwrap() error { return ErrUserNotActive }

// userNotActiveError 账号被禁用时返回带身份的 UserDisabledError，其它非 active 状态仍是 ErrUserNotActive。
func userNotActiveError(user *User) error {
	if user == nil || user.Status != StatusDisabled || user.ID <= 0 {
		return ErrUserNotActive
	}
	return &UserDisabledError{UserID: user.ID, Email: user.Email}
}

// NewUserNotActiveError 供 handler 包在已验证身份的位置使用（同 userNotActiveError）。
func NewUserNotActiveError(user *User) error {
	return userNotActiveError(user)
}

// AsUserDisabledError 取出被禁用账号的身份。
func AsUserDisabledError(err error) (*UserDisabledError, bool) {
	var target *UserDisabledError
	if errors.As(err, &target) && target != nil && target.UserID > 0 {
		return target, true
	}
	return nil, false
}
