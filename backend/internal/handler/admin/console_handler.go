package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ConsoleHandler 提供控制台会话信息：当前角色、可用 scope 与 ops 开关。
// 运维管理员（operator）无权读取 GET /admin/settings，前端改用本接口决定要渲染什么。
type ConsoleHandler struct {
	settingService *service.SettingService
	opsService     *service.OpsService
}

func NewConsoleHandler(settingService *service.SettingService, opsService *service.OpsService) *ConsoleHandler {
	return &ConsoleHandler{settingService: settingService, opsService: opsService}
}

// ConsoleSessionResponse 控制台会话信息。字段与 GET /admin/settings 中的同名字段同源。
type ConsoleSessionResponse struct {
	Role                         string   `json:"role"`
	Scopes                       []string `json:"scopes"`
	OpsMonitoringEnabled         bool     `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled bool     `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault          string   `json:"ops_query_mode_default"`
}

const (
	ConsoleScopeAll       = "*"
	ConsoleScopeOpsRead   = "ops:read"
	ConsoleScopeUsageRead = "usage:read"
)

// ConsoleScopesForRole 返回角色对应的控制台 scope 列表（仅用于前端展示，授权以后端白名单为准）。
func ConsoleScopesForRole(role string) []string {
	switch role {
	case service.RoleAdmin:
		return []string{ConsoleScopeAll}
	case service.RoleOperator:
		return []string{ConsoleScopeOpsRead, ConsoleScopeUsageRead}
	default:
		return []string{}
	}
}

// GetSession 返回当前控制台会话信息。
// GET /api/v1/admin/console/session
func (h *ConsoleHandler) GetSession(c *gin.Context) {
	role, _ := middleware.GetUserRoleFromContext(c)
	resp := ConsoleSessionResponse{
		Role:                         role,
		Scopes:                       ConsoleScopesForRole(role),
		OpsMonitoringEnabled:         true,
		OpsRealtimeMonitoringEnabled: true,
		OpsQueryModeDefault:          string(service.ParseOpsQueryMode("")),
	}

	ctx := c.Request.Context()
	if h.settingService != nil {
		settings, err := h.settingService.GetAllSettings(ctx)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		resp.OpsMonitoringEnabled = settings.OpsMonitoringEnabled
		resp.OpsRealtimeMonitoringEnabled = settings.OpsRealtimeMonitoringEnabled
		resp.OpsQueryModeDefault = settings.OpsQueryModeDefault
	}
	// 与 GET /admin/settings 一致：config.ops.enabled 硬开关关闭时 ops 一律视为不可用。
	opsEnabled := h.opsService != nil && h.opsService.IsMonitoringEnabled(ctx)
	resp.OpsMonitoringEnabled = opsEnabled && resp.OpsMonitoringEnabled

	response.Success(c, resp)
}
