package middleware

import (
	"net/http"
	"sort"
	"strings"
)

// 运维个人令牌在用户侧（jwtAuth）接口上的范围（fork 本地，见 docs/PERSONAL_TOKENS.md）。
//
// 用户侧接口只作用于持有人自己的账号，令牌用在这里不存在越权，所以默认放行；
// 只挡下面这些「账号安全」写操作。挡它们不是因为越权，而是令牌是长期凭证（可能永不过期），
// 一旦泄露不能让拿到它的人改密码 / 关 2FA / 换登录方式把账号夺走，也不能用令牌再生成令牌、
// 或者把令牌换成一个完整的浏览器会话。读操作（GET / HEAD）一律放行。
//
// 按前缀匹配 gin 路由模板：前缀下今后新增的写接口自动被挡住。
// routes/personal_token_scope_equivalence_test.go 钉住这张表（每个前缀都必须命中已注册的写路由）。
var personalTokenUserDeniedPrefixes = []string{
	"/api/v1/user/password",         // 改密码
	"/api/v1/user/totp",             // 2FA 设置 / 启用 / 关闭 / step-up
	"/api/v1/user/passkeys",         // Passkey 注册 / 改名 / 删除
	"/api/v1/user/account-bindings", // 绑定 / 解绑邮箱等登录方式
	"/api/v1/user/auth-identities",  // 绑定第三方登录
	"/api/v1/user/personal-token",   // 令牌不能生成 / 覆盖 / 吊销令牌
	"/api/v1/auth/",                 // 签发桌面会话、撤销全部会话、OAuth 绑定凭证（GET /auth/me 仍可用）
}

// PersonalTokenUserRouteAllows 报告个人令牌能否调用这个用户侧路由（method + gin 路由模板）。
// 纯函数；空路径（未匹配到路由）一律拒绝。
func PersonalTokenUserRouteAllows(method, fullPath string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	fullPath = strings.TrimSpace(fullPath)
	if method == "" || fullPath == "" {
		return false
	}
	if method == http.MethodGet || method == http.MethodHead {
		return true
	}
	for _, prefix := range personalTokenUserDeniedPrefixes {
		if strings.HasPrefix(fullPath, prefix) {
			return false
		}
	}
	return true
}

// PersonalTokenUserDeniedPrefixes 返回拒绝表（已排序），供路由一致性测试使用。
func PersonalTokenUserDeniedPrefixes() []string {
	out := append([]string(nil), personalTokenUserDeniedPrefixes...)
	sort.Strings(out)
	return out
}
