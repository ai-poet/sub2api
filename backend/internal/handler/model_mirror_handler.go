package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelMirrorHandler struct {
	modelMirrorService *service.ModelMirrorService
}

func NewModelMirrorHandler(modelMirrorService *service.ModelMirrorService) *ModelMirrorHandler {
	return &ModelMirrorHandler{
		modelMirrorService: modelMirrorService,
	}
}

func (h *ModelMirrorHandler) Verify(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req service.ModelMirrorVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	run, err := h.modelMirrorService.PrepareRun(c.Request.Context(), subject.UserID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if err := h.modelMirrorService.StreamRun(c, run); err != nil && !c.Writer.Written() {
		response.ErrorFrom(c, err)
	}
}
