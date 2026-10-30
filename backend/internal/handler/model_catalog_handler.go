package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelCatalogHandler struct {
	modelCatalogService interface {
		GetCatalog(context.Context, int64) (*service.ModelCatalogResponse, error)
		GetPayGroupCatalog(context.Context, service.Group) (*service.ModelCatalogResponse, error)
	}
	payGroupReader interface {
		GetGroup(context.Context, int64) (*service.Group, error)
	}
}

func NewModelCatalogHandler(modelCatalogService *service.ModelCatalogService, adminService service.AdminService) *ModelCatalogHandler {
	return &ModelCatalogHandler{modelCatalogService: modelCatalogService, payGroupReader: adminService}
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
