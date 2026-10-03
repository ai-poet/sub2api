package service

// passkeyLoginAccountUsable Passkey 登录时账号是否可以继续（fork 本地，封禁申诉）。
//
// 上游只放行 active；这里额外放行 disabled：Passkey 断言本身就是强身份验证，
// 被禁用的账号拿到的不是 token，而是 PasskeyHandler.FinishLogin 返回的 403 + 申诉令牌
// （handler 在签发 token 之前还会用 ensureLoginUserActive 再拦一次）。
func passkeyLoginAccountUsable(account *User) bool {
	if account == nil {
		return false
	}
	return account.IsActive() || account.Status == StatusDisabled
}
