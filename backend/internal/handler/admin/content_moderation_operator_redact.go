package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 运维管理员（operator）只读查看内容审计时的响应投影：抹掉审核引擎 Key 的掩码与哈希，
// 只留序号、状态和计数。只在 operator 请求时生效，admin 看到的数据不变；原始对象不被修改。

func contentModerationConfigForViewer(c *gin.Context, cfg *service.ContentModerationConfigView) *service.ContentModerationConfigView {
	if cfg == nil || !middleware.IsOperatorRequest(c) {
		return cfg
	}
	return redactContentModerationConfig(cfg)
}

func contentModerationStatusForViewer(c *gin.Context, status *service.ContentModerationRuntimeStatus) *service.ContentModerationRuntimeStatus {
	if status == nil || !middleware.IsOperatorRequest(c) {
		return status
	}
	clone := *status
	clone.APIKeyStatuses = redactContentModerationAPIKeyStatuses(status.APIKeyStatuses)
	if status.PreBlockAPIKeyLoads != nil {
		loads := make([]service.ContentModerationAPIKeyLoad, len(status.PreBlockAPIKeyLoads))
		for i, load := range status.PreBlockAPIKeyLoads {
			load.KeyHash = ""
			load.Masked = ""
			loads[i] = load
		}
		clone.PreBlockAPIKeyLoads = loads
	}
	return &clone
}

func redactContentModerationConfig(cfg *service.ContentModerationConfigView) *service.ContentModerationConfigView {
	clone := *cfg
	clone.APIKeyMasked = ""
	clone.APIKeyMasks = nil
	clone.APIKeyStatuses = redactContentModerationAPIKeyStatuses(cfg.APIKeyStatuses)
	if cfg.EngineConfigs != nil {
		engines := make(map[string]*service.ContentModerationConfigView, len(cfg.EngineConfigs))
		for engine, engineCfg := range cfg.EngineConfigs {
			if engineCfg == nil {
				engines[engine] = nil
				continue
			}
			engines[engine] = redactContentModerationConfig(engineCfg)
		}
		clone.EngineConfigs = engines
	}
	return &clone
}

func redactContentModerationAPIKeyStatuses(statuses []service.ContentModerationAPIKeyStatus) []service.ContentModerationAPIKeyStatus {
	if statuses == nil {
		return nil
	}
	out := make([]service.ContentModerationAPIKeyStatus, len(statuses))
	for i, status := range statuses {
		status.KeyHash = ""
		status.Masked = ""
		out[i] = status
	}
	return out
}
