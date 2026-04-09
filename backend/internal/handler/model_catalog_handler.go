package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelCatalogHandler struct {
	modelCatalogService *service.ModelCatalogService
}

func NewModelCatalogHandler(modelCatalogService *service.ModelCatalogService) *ModelCatalogHandler {
	return &ModelCatalogHandler{modelCatalogService: modelCatalogService}
}

func (h *ModelCatalogHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	result, err := h.modelCatalogService.GetCatalog(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
