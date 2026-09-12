package middleware

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// operatorReadScope 运维管理员（operator）可访问的管理接口白名单。
//
// key 为 "METHOD <gin 路由模板>"，与 c.FullPath() 逐字比对；不在表内的一律拒绝。
// 改动前先读 docs/OPERATOR_ROLE.md，设计约束：
//   - 默认拒绝：上游合并新增的任何 /admin 路由对 operator 天然不可见；
//   - 只读：本表只允许 GET，OperatorScopeAllows 在代码层再兜一次，误加的写条目不会生效；
//   - 每加一条都要评估返回字段是否含凭证 / 余额 / 明文 key / 完整 IP，必要时在 handler 做投影；
//   - 与 routes/console_scope_coverage_test.go 的 golden 列表保持一致，放大权限必须显式改测试。
var operatorReadScope = map[string]struct{}{
	// 合规确认状态（AdminComplianceGuard 对 operator 同样生效，未确认前只能访问这里）
	"GET /api/v1/admin/compliance": {},

	// 控制台会话信息：角色、scope 与 ops 开关，替代 operator 无权访问的 GET /admin/settings
	"GET /api/v1/admin/console/session": {},

	// 运维监控：首屏
	"GET /api/v1/admin/ops/dashboard/snapshot-v2":        {},
	"GET /api/v1/admin/ops/dashboard/overview":           {},
	"GET /api/v1/admin/ops/dashboard/throughput-trend":   {},
	"GET /api/v1/admin/ops/dashboard/latency-histogram":  {},
	"GET /api/v1/admin/ops/dashboard/error-trend":        {},
	"GET /api/v1/admin/ops/dashboard/error-distribution": {},
	"GET /api/v1/admin/ops/dashboard/openai-token-stats": {},
	"GET /api/v1/admin/ops/realtime-traffic":             {},
	"GET /api/v1/admin/ops/concurrency":                  {},
	"GET /api/v1/admin/ops/user-concurrency":             {},
	"GET /api/v1/admin/ops/account-availability":         {},
	"GET /api/v1/admin/ops/alert-events":                 {},
	"GET /api/v1/admin/ops/alert-events/:id":             {},
	"GET /api/v1/admin/ops/system-logs":                  {},
	"GET /api/v1/admin/ops/system-logs/health":           {},
	"GET /api/v1/admin/ops/settings/metric-thresholds":   {},
	"GET /api/v1/admin/ops/advanced-settings":            {},

	// 运维监控：钻取
	"GET /api/v1/admin/ops/requests":                           {},
	"GET /api/v1/admin/ops/errors":                             {},
	"GET /api/v1/admin/ops/errors/:id":                         {},
	"GET /api/v1/admin/ops/request-errors":                     {},
	"GET /api/v1/admin/ops/request-errors/:id":                 {},
	"GET /api/v1/admin/ops/request-errors/:id/upstream-errors": {},
	"GET /api/v1/admin/ops/upstream-errors":                    {},
	"GET /api/v1/admin/ops/upstream-errors/:id":                {},
	"GET /api/v1/admin/ops/ws/qps":                             {},

	// 调用日志（响应在 handler 层按 operator 投影：去明文 key / 余额，IP 掩码）
	"GET /api/v1/admin/usage":                 {},
	"GET /api/v1/admin/usage/stats":           {},
	"GET /api/v1/admin/usage/search-users":    {},
	"GET /api/v1/admin/usage/search-api-keys": {},
}

// operatorWriteScope 白名单中唯一允许的非 GET 条目：合规确认是进入控制台的前置条件，
// 不确认就无法使用任何管理接口。除此之外 operator 的所有写请求一律拒绝。
var operatorWriteScope = map[string]struct{}{
	"POST /api/v1/admin/compliance/accept": {},
}

// OperatorScopeAllows 报告 operator 是否可以访问 method + fullPath（gin 路由模板）。
// 纯函数，不接触 gin.Context，便于单测；空路径（未匹配到路由）一律拒绝。
func OperatorScopeAllows(method, fullPath string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	fullPath = strings.TrimSpace(fullPath)
	if method == "" || fullPath == "" {
		return false
	}
	key := method + " " + fullPath
	if method != http.MethodGet {
		_, ok := operatorWriteScope[key]
		return ok
	}
	_, ok := operatorReadScope[key]
	return ok
}

// OperatorScopeRoutes 返回白名单全部条目（"METHOD path"，已排序），供路由一致性测试使用。
func OperatorScopeRoutes() []string {
	out := make([]string, 0, len(operatorReadScope)+len(operatorWriteScope))
	for key := range operatorReadScope {
		out = append(out, key)
	}
	for key := range operatorWriteScope {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// IsOperatorRequest 报告当前已认证请求的角色是否为 operator。
func IsOperatorRequest(c *gin.Context) bool {
	role, ok := GetUserRoleFromContext(c)
	return ok && role == service.RoleOperator
}

// operatorDenyAuditLimit 每个 operator 每分钟最多落库的拒绝审计条数。
// 拒绝发生在面板限流之前，这里兜底防止恶意刷审计表；超出后仍然拒绝请求，只是不再落库。
const (
	operatorDenyAuditLimit  = 30
	operatorDenyAuditWindow = time.Minute
)

type operatorDenyAuditLimiter struct {
	mu      sync.Mutex
	buckets map[int64]*operatorDenyBucket
}

type operatorDenyBucket struct {
	windowStart time.Time
	count       int
}

var denyAuditLimiter = &operatorDenyAuditLimiter{buckets: make(map[int64]*operatorDenyBucket)}

// allow 报告该用户在当前窗口内是否还可以写入拒绝审计。
func (l *operatorDenyAuditLimiter) allow(userID int64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket, ok := l.buckets[userID]
	if !ok || now.Sub(bucket.windowStart) >= operatorDenyAuditWindow {
		// 顺带回收其它过期桶，避免 map 无限增长。
		for id, b := range l.buckets {
			if now.Sub(b.windowStart) >= operatorDenyAuditWindow {
				delete(l.buckets, id)
			}
		}
		l.buckets[userID] = &operatorDenyBucket{windowStart: now, count: 1}
		return true
	}
	if bucket.count >= operatorDenyAuditLimit {
		return false
	}
	bucket.count++
	return true
}

// recordOperatorScopeDenied 把 operator 命中白名单之外接口的请求写入审计。
// 审计中间件挂在认证之后，不会记录认证层的 403，所以这里直接落库；不记录请求体。
func recordOperatorScopeDenied(c *gin.Context, auditService *service.AuditLogService, user *service.User) {
	if c == nil || auditService == nil || user == nil {
		return
	}
	if !denyAuditLimiter.allow(user.ID, time.Now()) {
		return
	}
	uid := user.ID
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	entry := &service.AuditLog{
		ActorUserID: &uid,
		ActorEmail:  user.Email,
		ActorRole:   user.Role,
		AuthMethod:  service.AuditAuthMethodJWT,
		Action:      service.AuditActionAdminScopeDenied,
		Method:      c.Request.Method,
		Path:        path,
		ClientIP:    SecurityClientIP(c),
		UserAgent:   normalizePersistentText(c.Request.UserAgent(), maxPersistentUserAgentBytes),
		StatusCode:  http.StatusForbidden,
	}
	if requestID, ok := c.Request.Context().Value(ctxkey.RequestID).(string); ok {
		entry.RequestID = requestID
	}
	if q := service.RedactAuditQuery(c.Request.URL.RawQuery); q != "" {
		entry.Extra = map[string]any{"query": q}
	}
	auditService.Record(entry)
}
