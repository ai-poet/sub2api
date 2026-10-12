package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Histories 一次返回用户可见分组的可用率分桶（fork：模型广场可用率条）。
// GET /api/v1/group-status/history?period=24h|7d
func (h *GroupStatusHandler) Histories(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	period := c.DefaultQuery("period", service.GroupStatusPeriod24h)
	items, err := h.groupStatusService.ListUserHistories(c.Request.Context(), subject.UserID, period)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}
