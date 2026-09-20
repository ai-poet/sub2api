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
//   - 只读：本表只允许 GET，OperatorScopeAllows 在代码层再兜一次，误加的写条目不会生效（operatorWriteScope 为空）；
//   - 每加一条都要评估返回字段是否含凭证 / 余额 / 明文 key / 完整 IP，必要时在 handler 做投影；
//   - 与 routes/console_scope_coverage_test.go 的 golden 列表保持一致，放大权限必须显式改测试。
var operatorReadScope = map[string]struct{}{
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
	// 调用日志筛选项：分组 / 账号只返回 id、name、platform，模型只返回名称（usage_handler_filter_options.go）；
	// 替代 operator 无权访问的 /admin/groups、/admin/accounts、/admin/dashboard/models
	"GET /api/v1/admin/usage/search-accounts": {},
	"GET /api/v1/admin/usage/filter-groups":   {},
	"GET /api/v1/admin/usage/filter-models":   {},
	// 调用日志图表：趋势 / 分组 / 模型分布聚合（usage_handler_charts.go），operator 响应抹掉 account_cost；
	// 替代 operator 无权访问的 /admin/dashboard/{snapshot-v2,models}，不含用户级明细
	"GET /api/v1/admin/usage/charts":      {},
	"GET /api/v1/admin/usage/model-stats": {},

	// 用户管理（只读部分；写操作走审批，见 operatorApprovalScope）。
	// api-keys / balance-history 在 handler 层按 operator 投影：无明文 key、无卡密串。
	"GET /api/v1/admin/users":                     {},
	"GET /api/v1/admin/users/:id":                 {},
	"GET /api/v1/admin/users/:id/api-keys":        {},
	"GET /api/v1/admin/users/:id/usage":           {},
	"GET /api/v1/admin/users/:id/balance-history": {},
	"GET /api/v1/admin/users/:id/rpm-status":      {},
	"GET /api/v1/admin/users/:id/platform-quotas": {},
	"GET /api/v1/admin/users/:id/attributes":      {},
	"GET /api/v1/admin/users/:id/subscriptions":   {},
	"GET /api/v1/admin/user-attributes":           {},

	// 订阅管理（只读部分）
	"GET /api/v1/admin/subscriptions":              {},
	"GET /api/v1/admin/subscriptions/:id":          {},
	"GET /api/v1/admin/subscriptions/:id/progress": {},

	// 审批申请：operator 只能看到自己的（handler 强制按申请人过滤）
	"GET /api/v1/admin/approvals":               {},
	"GET /api/v1/admin/approvals/pending-count": {},
	"GET /api/v1/admin/approvals/:id":           {},

	// 工单（fork 本地）：admin 与 operator 同权看全部并回复；响应只含用户邮箱与工单文本，无凭证 / 余额 / IP
	"GET /api/v1/admin/tickets":            {},
	"GET /api/v1/admin/tickets/open-count": {},
	"GET /api/v1/admin/tickets/:id":        {},
	// 工单图片附件的同源回传：handler 按附件前缀做授权边界，响应只有图片字节
	"GET /api/v1/admin/tickets/attachments/content": {},
}

// operatorWriteScope 允许 operator 直接调用（不经审批）的非 GET 条目。只放三类：
// 读语义的 POST（批量取用户属性）、撤回自己的审批申请、以及客服工单的回复 / 关闭 / 重开
// （工单是客服工作流，排队审批没有意义；只触及工单表，全部留审计）。其它写操作要么走
// operatorApprovalScope，要么默认拒绝。每加一条都必须同步 golden 测试。
var operatorWriteScope = map[string]struct{}{
	"POST /api/v1/admin/user-attributes/batch": {},
	"POST /api/v1/admin/approvals/:id/cancel":  {},
	"POST /api/v1/admin/tickets/:id/messages":  {},
	"POST /api/v1/admin/tickets/:id/close":     {},
	"POST /api/v1/admin/tickets/:id/reopen":    {},
	// 工单图片附件上传：与工单回复同属客服工作流，key 由服务端生成、类型与大小双重受限
	"POST /api/v1/admin/tickets/attachments": {},
}

// operatorApprovalScope operator 可以发起、但必须由管理员在审批页一键通过后才会执行的写接口。
// 认证层把这些请求交给 AdminApprovalGate 入队并返回 202，绝不直接到达 handler；
// 审批通过后以管理员身份内部重放（见 admin_auth_approval.go / service.AdminApprovalService）。
// 角色变更（body.role 与目标当前角色不同）即使命中本表也会被审批服务拒绝。
var operatorApprovalScope = map[string]struct{}{
	// 用户管理
	"POST /api/v1/admin/users":                           {},
	"PUT /api/v1/admin/users/:id":                        {},
	"POST /api/v1/admin/users/:id/balance":               {},
	"POST /api/v1/admin/users/:id/replace-group":         {},
	"POST /api/v1/admin/users/batch-concurrency":         {},
	"POST /api/v1/admin/users/batch-limits":              {},
	"PUT /api/v1/admin/users/:id/platform-quotas":        {},
	"POST /api/v1/admin/users/:id/platform-quotas/reset": {},
	"PUT /api/v1/admin/users/:id/attributes":             {},
	"POST /api/v1/admin/users/:id/auth-identities":       {},
	// 用户 API Key 的分组调整（用户管理页的 API Key 弹窗）
	"PUT /api/v1/admin/api-keys/:id": {},
	// 订阅管理
	"POST /api/v1/admin/subscriptions/assign":          {},
	"POST /api/v1/admin/subscriptions/bulk-assign":     {},
	"POST /api/v1/admin/subscriptions/:id/extend":      {},
	"POST /api/v1/admin/subscriptions/:id/reset-quota": {},
	"POST /api/v1/admin/subscriptions/:id/revoke":      {},
	"POST /api/v1/admin/subscriptions/:id/restore":     {},
	"DELETE /api/v1/admin/subscriptions/:id":           {},
}

// operatorRefusedScope 永远不允许 operator 发起、也不入队的动作：直接 403 并审计。
var operatorRefusedScope = map[string]struct{}{
	"DELETE /api/v1/admin/users/:id": {},
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

// OperatorApprovalRequired 报告 method + fullPath 是否属于"operator 可发起、须管理员审批"的写接口。
func OperatorApprovalRequired(method, fullPath string) bool {
	key := scopeKey(method, fullPath)
	if key == "" {
		return false
	}
	_, ok := operatorApprovalScope[key]
	return ok
}

// OperatorActionRefused 报告 method + fullPath 是否属于 operator 永远不能发起的动作。
func OperatorActionRefused(method, fullPath string) bool {
	key := scopeKey(method, fullPath)
	if key == "" {
		return false
	}
	_, ok := operatorRefusedScope[key]
	return ok
}

// OperatorApprovalScopeRoutes / OperatorRefusedScopeRoutes 返回对应表的全部条目（已排序），供 golden 测试使用。
func OperatorApprovalScopeRoutes() []string {
	return sortedScopeKeys(operatorApprovalScope)
}

func OperatorRefusedScopeRoutes() []string {
	return sortedScopeKeys(operatorRefusedScope)
}

func scopeKey(method, fullPath string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	fullPath = strings.TrimSpace(fullPath)
	if method == "" || fullPath == "" {
		return ""
	}
	return method + " " + fullPath
}

func sortedScopeKeys(table map[string]struct{}) []string {
	out := make([]string, 0, len(table))
	for key := range table {
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
