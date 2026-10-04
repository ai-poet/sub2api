package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ContentTranslationHandler 内容自动翻译的公开查询与支付服务登记（fork 本地功能）。
//
// 查询只读缓存，永远不调模型：未知文本查不到就是查不到，所以无需登录也不会被用来刷管理员额度。
type ContentTranslationHandler struct {
	svc *service.ContentTranslationService
}

// NewContentTranslationHandler 构造 handler。
func NewContentTranslationHandler(svc *service.ContentTranslationService) *ContentTranslationHandler {
	return &ContentTranslationHandler{svc: svc}
}

// Lookup POST /api/v1/content-translations/lookup
func (h *ContentTranslationHandler) Lookup(c *gin.Context) {
	var req dto.ContentTranslationLookupRequest
	if !bindContentTranslationJSON(c, &req) {
		return
	}
	result, err := h.svc.Lookup(c.Request.Context(), req.Lang, req.Texts)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ContentTranslationLookupResponse{
		Lang:         result.Lang,
		Translations: result.Translations,
		Pending:      result.Pending,
	})
}

// RegisterPaySources PUT /api/internal/pay/content-translations/sources（只认支付服务的内部令牌）
func (h *ContentTranslationHandler) RegisterPaySources(c *gin.Context) {
	var req dto.RegisterContentTranslationSourcesRequest
	if !bindContentTranslationJSON(c, &req) {
		return
	}
	count, err := h.svc.RegisterExternalSources(c.Request.Context(), service.ContentTranslationNamespacePay, req.Texts)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"count": count})
}

// bindContentTranslationJSON 解析请求体；超过路由上的 RequestBodyLimit 时返回 413 而不是笼统的 400。
func bindContentTranslationJSON(c *gin.Context, out any) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		if _, ok := extractMaxBytesError(err); ok {
			response.Error(c, http.StatusRequestEntityTooLarge, "Request body too large")
			return false
		}
		response.BadRequest(c, "Invalid request body")
		return false
	}
	return true
}
