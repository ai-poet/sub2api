package dto

import "time"

// 封禁申诉会话的请求 / 响应结构（fork 本地功能）。

// AppealSession GET /api/v1/appeal/session
type AppealSession struct {
	Email           string    `json:"email"`
	Status          string    `json:"status"`
	ExpiresAt       time.Time `json:"expires_at"`
	HasActiveAppeal bool      `json:"has_active_appeal"`
}

// CreateAppealTicketRequest POST /api/v1/appeal/tickets（分类固定为 appeal）。
type CreateAppealTicketRequest struct {
	Title string `json:"title" binding:"required,max=200"`
	Body  string `json:"body" binding:"required,max=20000"`
}
