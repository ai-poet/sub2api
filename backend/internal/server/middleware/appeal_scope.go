package middleware

import (
	"net/http"
	"sort"
	"strings"
)

// 封禁申诉会话的路由范围（fork 本地，见 service/appeal.go）。
//
// 默认拒绝：申诉令牌只能访问这张表里的 METHOD + gin 路由模板，其余一律 403 APPEAL_ROUTE_FORBIDDEN。
// 表里只有「看自己的站内信」「提交 / 查看 / 回复自己的申诉工单」；没有任何能换取 token、改账号或碰网关的入口。
// routes/appeal_scope_coverage_test.go 钉住这张表与实际注册的 /api/v1/appeal/* 路由一一对应。
var appealScope = map[string]struct{}{
	"GET /api/v1/appeal/session": {},
	"POST /api/v1/appeal/logout": {},

	"GET /api/v1/appeal/site-messages":              {},
	"GET /api/v1/appeal/site-messages/unread-count": {},
	"POST /api/v1/appeal/site-messages/read-all":    {},
	"POST /api/v1/appeal/site-messages/:id/read":    {},

	"GET /api/v1/appeal/tickets":                     {},
	"POST /api/v1/appeal/tickets":                    {},
	"GET /api/v1/appeal/tickets/:id":                 {},
	"POST /api/v1/appeal/tickets/:id/messages":       {},
	"POST /api/v1/appeal/tickets/attachments":        {},
	"GET /api/v1/appeal/tickets/attachments/content": {},
}

// AppealScopeAllows 报告申诉令牌能否调用这个路由（method + gin 路由模板）。空路径（未匹配到路由）一律拒绝。
func AppealScopeAllows(method, fullPath string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	fullPath = strings.TrimSpace(fullPath)
	if method == "" || fullPath == "" {
		return false
	}
	if method == http.MethodHead {
		method = http.MethodGet
	}
	_, ok := appealScope[method+" "+fullPath]
	return ok
}

// AppealScopeRoutes 返回申诉白名单（已排序），供路由一致性测试使用。
func AppealScopeRoutes() []string {
	out := make([]string, 0, len(appealScope))
	for route := range appealScope {
		out = append(out, route)
	}
	sort.Strings(out)
	return out
}
