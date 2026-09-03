package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *GroupHandler) SetGroupStatusServices(groupStatusService *service.GroupStatusService, groupStatusProbeSvc *service.GroupStatusProbeService) {
	h.groupStatusService = groupStatusService
	h.groupStatusProbeSvc = groupStatusProbeSvc
}

type UpdateRuntimeStatusRequest struct {
	Enabled          bool     `json:"enabled"`
	ProbeModel       string   `json:"probe_model"`
	ProbePrompt      string   `json:"probe_prompt"`
	ValidationMode   string   `json:"validation_mode"`
	ExpectedKeywords []string `json:"expected_keywords"`
	IntervalSeconds  int      `json:"interval_seconds"`
	TimeoutSeconds   int      `json:"timeout_seconds"`
	SlowLatencyMS    int64    `json:"slow_latency_ms"`
	// 为 nil 时保留已保存的值（省略 = 保持现值）
	NotifyEnabled *bool `json:"notify_enabled"`
}

// GetRuntimeStatus handles loading runtime status config for a group.
// GET /api/v1/admin/groups/:id/runtime-status
func (h *GroupHandler) GetRuntimeStatus(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	view, err := h.groupStatusService.GetAdminView(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, view)
}

// UpdateRuntimeStatus handles saving runtime status config for a group.
// PUT /api/v1/admin/groups/:id/runtime-status
func (h *GroupHandler) UpdateRuntimeStatus(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	var req UpdateRuntimeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	view, err := h.groupStatusService.UpdateConfig(c.Request.Context(), groupID, &service.GroupStatusConfigUpsertInput{
		Enabled:          req.Enabled,
		ProbeModel:       req.ProbeModel,
		ProbePrompt:      req.ProbePrompt,
		ValidationMode:   req.ValidationMode,
		ExpectedKeywords: req.ExpectedKeywords,
		IntervalSeconds:  req.IntervalSeconds,
		TimeoutSeconds:   req.TimeoutSeconds,
		SlowLatencyMS:    req.SlowLatencyMS,
		NotifyEnabled:    req.NotifyEnabled,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, view)
}

// ProbeRuntimeStatus handles immediate runtime status probing for a group.
// POST /api/v1/admin/groups/:id/runtime-status/probe
func (h *GroupHandler) ProbeRuntimeStatus(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	if _, err := h.groupStatusProbeSvc.ProbeGroupNow(c.Request.Context(), groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	view, err := h.groupStatusService.GetAdminView(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, view)
}

// GetRuntimeStatusSummary handles loading runtime status summary for all configured groups.
// GET /api/v1/admin/groups/runtime-status/summary
func (h *GroupHandler) GetRuntimeStatusSummary(c *gin.Context) {
	summaries, err := h.groupStatusService.ListAdminSummaries(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summaries)
}
