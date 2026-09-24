package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ClientChangelogHandler 给官网 /changelog 页提供客户端更新日志（fork 本地）。
//
// 数据来自 GitHub Releases，由 ClientChangelogService 缓存。和公开定价一样，
// 这个接口**永不返回 401**，GitHub 不可用时返回 200 + 空列表，让页面显示空状态。
type ClientChangelogHandler struct {
	changelog *service.ClientChangelogService
}

// NewClientChangelogHandler 创建客户端更新日志 handler。
func NewClientChangelogHandler(changelog *service.ClientChangelogService) *ClientChangelogHandler {
	return &ClientChangelogHandler{changelog: changelog}
}

type clientChangelogResponse struct {
	Entries []service.ClientChangelogEntry `json:"entries"`
}

// List 返回最近的客户端发版记录，新的在前。
// GET /api/v1/changelog
func (h *ClientChangelogHandler) List(c *gin.Context) {
	entries := []service.ClientChangelogEntry{}
	if h.changelog != nil {
		entries = h.changelog.Entries(c.Request.Context())
	}
	c.Header("Cache-Control", "public, max-age=300")
	response.Success(c, clientChangelogResponse{Entries: entries})
}
