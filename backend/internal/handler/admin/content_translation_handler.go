package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContentTranslationHandler 内容自动翻译的后台管理（fork 本地功能，仅管理员）。
type ContentTranslationHandler struct {
	svc *service.ContentTranslationService
}

// NewContentTranslationHandler 构造后台 handler。
func NewContentTranslationHandler(svc *service.ContentTranslationService) *ContentTranslationHandler {
	return &ContentTranslationHandler{svc: svc}
}

// GetConfig GET /api/v1/admin/content-translations/config
func (h *ContentTranslationHandler) GetConfig(c *gin.Context) {
	cfg, keyName, err := h.svc.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ContentTranslationConfigFromService(cfg, keyName))
}

// UpdateConfig PUT /api/v1/admin/content-translations/config
func (h *ContentTranslationHandler) UpdateConfig(c *gin.Context) {
	var req dto.ContentTranslationConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	cfg, keyName, err := h.svc.UpdateConfig(c.Request.Context(), req.ToService())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ContentTranslationConfigFromService(cfg, keyName))
}

// TestConfig POST /api/v1/admin/content-translations/config/test
func (h *ContentTranslationHandler) TestConfig(c *gin.Context) {
	var req dto.ContentTranslationConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	result, err := h.svc.TestConfig(c.Request.Context(), req.ToService())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ContentTranslationTestResult{Translated: result.Translated, LatencyMS: result.LatencyMS})
}

// Status GET /api/v1/admin/content-translations/status
func (h *ContentTranslationHandler) Status(c *gin.Context) {
	response.Success(c, dto.ContentTranslationStatusFromService(h.svc.Status(c.Request.Context())))
}

// Sync POST /api/v1/admin/content-translations/sync：异步扫描一次（忽略失败退避），立即返回当前状态。
func (h *ContentTranslationHandler) Sync(c *gin.Context) {
	h.svc.TriggerSync()
	response.Success(c, dto.ContentTranslationStatusFromService(h.svc.Status(c.Request.Context())))
}

// List GET /api/v1/admin/content-translations
func (h *ContentTranslationHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	items, total, err := h.svc.List(c.Request.Context(), service.ContentTranslationFilter{
		Lang:     c.Query("lang"),
		Query:    c.Query("q"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.ContentTranslationItem, 0, len(items))
	for _, item := range items {
		out = append(out, dto.ContentTranslationItemFromService(item))
	}
	response.Paginated(c, out, total, page, pageSize)
}

// Update PUT /api/v1/admin/content-translations/:id
func (h *ContentTranslationHandler) Update(c *gin.Context) {
	id, ok := parseContentTranslationID(c)
	if !ok {
		return
	}
	var req dto.UpdateContentTranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	item, err := h.svc.UpdateTranslation(c.Request.Context(), id, req.TranslatedText)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ContentTranslationItemFromService(item))
}

// Delete DELETE /api/v1/admin/content-translations/:id
func (h *ContentTranslationHandler) Delete(c *gin.Context) {
	id, ok := parseContentTranslationID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteTranslation(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": 1})
}

// Clear DELETE /api/v1/admin/content-translations?scope=machine：清空全部机器译文，人工译文保留。
func (h *ContentTranslationHandler) Clear(c *gin.Context) {
	if strings.TrimSpace(c.Query("scope")) != "machine" {
		response.BadRequest(c, "scope must be machine")
		return
	}
	deleted, err := h.svc.ClearMachineTranslations(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": deleted})
}

func parseContentTranslationID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid translation ID")
		return 0, false
	}
	return id, true
}
