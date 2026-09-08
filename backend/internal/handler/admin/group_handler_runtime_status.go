package admin

import (
	"errors"
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
	// 纯 Sol 验证（Juice 指纹探测），仅 OpenAI 分组；SolJuiceEnabled 为 nil 时保留已保存的三项
	SolJuiceEnabled         *bool  `json:"sol_juice_enabled"`
	SolJuiceIntervalSeconds int    `json:"sol_juice_interval_seconds"`
	SolJuiceModel           string `json:"sol_juice_model"`
	// Astra 指纹验证（meow 基准），仅 OpenAI 分组；AstraCheckEnabled 为 nil 时保留已保存的四项
	AstraCheckEnabled         *bool  `json:"astra_check_enabled"`
	AstraCheckRequestModel    string `json:"astra_check_request_model"`
	AstraCheckTier            string `json:"astra_check_tier"`
	AstraCheckIntervalSeconds int    `json:"astra_check_interval_seconds"`
}

// withAstraCheckRunning 把「验证进行中」的内存标记补进管理视图。
func (h *GroupHandler) withAstraCheckRunning(view *service.GroupStatusAdminView, groupID int64) *service.GroupStatusAdminView {
	if view != nil && h.groupStatusProbeSvc != nil {
		view.Summary.AstraCheckRunning = h.groupStatusProbeSvc.IsAstraCheckRunning(groupID)
		view.AstraCheckProgress = h.groupStatusProbeSvc.AstraCheckProgress(groupID)
	}
	return view
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
	response.Success(c, h.withAstraCheckRunning(view, groupID))
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
		Enabled:                   req.Enabled,
		ProbeModel:                req.ProbeModel,
		ProbePrompt:               req.ProbePrompt,
		ValidationMode:            req.ValidationMode,
		ExpectedKeywords:          req.ExpectedKeywords,
		IntervalSeconds:           req.IntervalSeconds,
		TimeoutSeconds:            req.TimeoutSeconds,
		SlowLatencyMS:             req.SlowLatencyMS,
		NotifyEnabled:             req.NotifyEnabled,
		SolJuiceEnabled:           req.SolJuiceEnabled,
		SolJuiceIntervalSeconds:   req.SolJuiceIntervalSeconds,
		SolJuiceModel:             req.SolJuiceModel,
		AstraCheckEnabled:         req.AstraCheckEnabled,
		AstraCheckRequestModel:    req.AstraCheckRequestModel,
		AstraCheckTier:            req.AstraCheckTier,
		AstraCheckIntervalSeconds: req.AstraCheckIntervalSeconds,
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

// ProbeRuntimeStatusSolJuice runs the Sol Juice identity probe immediately for an OpenAI group.
// POST /api/v1/admin/groups/:id/runtime-status/sol-juice/probe
func (h *GroupHandler) ProbeRuntimeStatusSolJuice(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	if _, err := h.groupStatusProbeSvc.ProbeSolJuiceGroupNow(c.Request.Context(), groupID); err != nil {
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

// ProbeRuntimeStatusAstraCheck starts an Astra fingerprint check in the background for an OpenAI group.
// The run takes a minute or more, so the response only reports that it started; poll GET runtime-status.
// POST /api/v1/admin/groups/:id/runtime-status/astra-check/probe
func (h *GroupHandler) ProbeRuntimeStatusAstraCheck(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	if err := h.groupStatusProbeSvc.StartAstraCheckAsync(groupID); err != nil && !errors.Is(err, service.ErrGroupStatusAstraCheckRunning) {
		response.ErrorFrom(c, err)
		return
	}
	view, err := h.groupStatusService.GetAdminView(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	view = h.withAstraCheckRunning(view, groupID)
	view.Summary.AstraCheckRunning = true
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
