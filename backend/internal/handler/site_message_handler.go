package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SiteMessageHandler 用户侧站内信接口（fork 本地功能）：只读写自己的收件箱。
//
// 只从 gin context 读 AuthSubject，不检查账号状态，所以申诉会话（/api/v1/appeal/site-messages）
// 可以原样复用这些方法。
type SiteMessageHandler struct {
	svc *service.SiteMessageService
}

// NewSiteMessageHandler 构造用户侧站内信 handler。
func NewSiteMessageHandler(svc *service.SiteMessageService) *SiteMessageHandler {
	return &SiteMessageHandler{svc: svc}
}

func siteMessageUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

// List GET /api/v1/site-messages?page&page_size&unread_only&category
func (h *SiteMessageHandler) List(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	list, err := h.svc.ListMine(c.Request.Context(), userID, &service.SiteMessageFilter{
		Page:       page,
		PageSize:   pageSize,
		UnreadOnly: parseBoolQuery(c.Query("unread_only")),
		Category:   c.Query("category"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.SiteMessage, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, *dto.SiteMessageFromService(item))
	}
	response.Paginated(c, items, list.Total, list.Page, list.PageSize)
}

// UnreadCount GET /api/v1/site-messages/unread-count（顶栏角标，前端每分钟轮询）
func (h *SiteMessageHandler) UnreadCount(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	count, err := h.svc.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageCountResponse{Count: count})
}

// MarkRead POST /api/v1/site-messages/:id/read
func (h *SiteMessageHandler) MarkRead(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid site message id")
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), userID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

// MarkAllRead POST /api/v1/site-messages/read-all
func (h *SiteMessageHandler) MarkAllRead(c *gin.Context) {
	userID, ok := siteMessageUserID(c)
	if !ok {
		return
	}
	updated, err := h.svc.MarkAllRead(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.SiteMessageMarkAllResponse{Updated: updated})
}
