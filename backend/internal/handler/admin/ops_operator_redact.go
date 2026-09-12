package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 运维管理员（operator）查看 ops 错误日志时的响应投影：client_ip 掩码为 a.b.c.x。
// 只在 operator 请求时生效，admin 看到的数据不变；原始对象不被修改（先拷贝再脱敏）。

func opsErrorLogsForViewer(c *gin.Context, logs []*service.OpsErrorLog) []*service.OpsErrorLog {
	if !middleware.IsOperatorRequest(c) {
		return logs
	}
	out := make([]*service.OpsErrorLog, 0, len(logs))
	for _, item := range logs {
		if item == nil {
			continue
		}
		clone := *item
		maskOpsErrorLogForOperator(&clone)
		out = append(out, &clone)
	}
	return out
}

func opsErrorLogDetailsForViewer(c *gin.Context, details []*service.OpsErrorLogDetail) []*service.OpsErrorLogDetail {
	if !middleware.IsOperatorRequest(c) {
		return details
	}
	out := make([]*service.OpsErrorLogDetail, 0, len(details))
	for _, item := range details {
		if item == nil {
			continue
		}
		out = append(out, opsErrorLogDetailForViewer(c, item))
	}
	return out
}

func opsErrorLogDetailForViewer(c *gin.Context, detail *service.OpsErrorLogDetail) *service.OpsErrorLogDetail {
	if detail == nil || !middleware.IsOperatorRequest(c) {
		return detail
	}
	clone := *detail
	maskOpsErrorLogForOperator(&clone.OpsErrorLog)
	return &clone
}

func maskOpsErrorLogForOperator(log *service.OpsErrorLog) {
	if log == nil || log.ClientIP == nil {
		return
	}
	masked := ip.MaskIP(*log.ClientIP)
	if masked == "" {
		log.ClientIP = nil
		return
	}
	log.ClientIP = &masked
}
