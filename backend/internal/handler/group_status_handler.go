package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GroupStatusHandler struct {
	groupStatusService *service.GroupStatusService
}

func NewGroupStatusHandler(groupStatusService *service.GroupStatusService) *GroupStatusHandler {
	return &GroupStatusHandler{groupStatusService: groupStatusService}
}

func (h *GroupStatusHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	items, err := h.groupStatusService.ListUserStatuses(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *GroupStatusHandler) History(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	period := c.DefaultQuery("period", service.GroupStatusPeriod24h)
	history, err := h.groupStatusService.GetUserHistory(c.Request.Context(), subject.UserID, groupID, period)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, history)
}

func (h *GroupStatusHandler) Events(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	events, err := h.groupStatusService.GetUserEvents(c.Request.Context(), subject.UserID, groupID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, events)
}

func (h *GroupStatusHandler) Records(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	limit := 24
	if raw := c.Query("limit"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	records, err := h.groupStatusService.GetUserRecentRecords(c.Request.Context(), subject.UserID, groupID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, records)
}
