package admin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// 分组运行状态 → Server酱³ 推送的测试接口（本 fork 自有功能）。

// groupStatusNotifyTester 抽象出测试发送，便于 handler 单测注入假实现。
type groupStatusNotifyTester interface {
	SendTest(ctx context.Context, uid, sendkey string) error
}

func (h *SettingHandler) SetGroupStatusNotifyService(svc groupStatusNotifyTester) {
	h.groupStatusNotify = svc
}

type TestGroupStatusNotifyRequest struct {
	UID     string `json:"uid"`
	SendKey string `json:"sendkey"`
}

// TestGroupStatusNotify 发送一条 Server酱³ 测试推送；uid / sendkey 留空时使用已保存的配置。
// POST /api/v1/admin/settings/group-status-notify/test
func (h *SettingHandler) TestGroupStatusNotify(c *gin.Context) {
	var req TestGroupStatusNotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if h.groupStatusNotify == nil {
		response.Error(c, http.StatusServiceUnavailable, "group status notify service is not configured")
		return
	}
	if err := h.groupStatusNotify.SendTest(c.Request.Context(), strings.TrimSpace(req.UID), strings.TrimSpace(req.SendKey)); err != nil {
		response.BadRequest(c, "Server酱 push failed: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Test push sent"})
}
