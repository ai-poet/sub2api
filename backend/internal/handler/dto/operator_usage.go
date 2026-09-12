package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// OperatorUserRef 运维管理员（operator）在调用日志里看到的用户引用：只有身份，没有余额 / 并发 / 限额。
type OperatorUserRef struct {
	ID        int64      `json:"id"`
	Email     string     `json:"email"`
	Username  string     `json:"username"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// OperatorAPIKeyRef 运维管理员看到的 API Key 引用：没有明文 key、IP 名单、额度与速率窗口。
type OperatorAPIKeyRef struct {
	ID      int64  `json:"id"`
	UserID  int64  `json:"user_id"`
	Name    string `json:"name"`
	GroupID *int64 `json:"group_id"`
	Status  string `json:"status"`
}

// OperatorUsageLog 运维管理员看到的调用日志：在管理员 DTO 基础上去掉余额、明文 key、
// 会话标识、账号侧成本结构，并把 IP 掩码为 a.b.c.x。
//
// 外层的 user / api_key / ip_address 按 encoding/json 的浅层优先规则遮蔽嵌入结构中的同名字段；
// 嵌入层的对应字段同时置空作双保险（AdminUsageLog 与其嵌入的 UsageLog 各有一个 ip_address）。
type OperatorUsageLog struct {
	AdminUsageLog

	User      *OperatorUserRef   `json:"user,omitempty"`
	APIKey    *OperatorAPIKeyRef `json:"api_key,omitempty"`
	IPAddress *string            `json:"ip_address,omitempty"`
}

// UsageLogFromServiceOperator 把 usage log 投影成运维管理员可见的形态。
func UsageLogFromServiceOperator(l *service.UsageLog) *OperatorUsageLog {
	base := UsageLogFromServiceAdmin(l)
	if base == nil {
		return nil
	}
	out := &OperatorUsageLog{AdminUsageLog: *base}

	// 嵌入层清空：余额（User）、明文 key（APIKey）、完整 IP、会话标识、账号侧成本结构。
	out.AdminUsageLog.UsageLog.User = nil
	out.AdminUsageLog.UsageLog.APIKey = nil
	out.AdminUsageLog.UsageLog.IPAddress = nil
	out.AdminUsageLog.UsageLog.SessionID = nil
	out.AdminUsageLog.IPAddress = nil
	out.AdminUsageLog.AccountStatsCost = nil
	out.AdminUsageLog.AccountRateMultiplier = nil

	out.IPAddress = maskedIPPtr(l.IPAddress)
	if l.User != nil {
		out.User = &OperatorUserRef{
			ID:        l.User.ID,
			Email:     l.User.Email,
			Username:  l.User.Username,
			Role:      l.User.Role,
			Status:    l.User.Status,
			DeletedAt: l.User.DeletedAt,
		}
	}
	if l.APIKey != nil {
		out.APIKey = &OperatorAPIKeyRef{
			ID:      l.APIKey.ID,
			UserID:  l.APIKey.UserID,
			Name:    l.APIKey.Name,
			GroupID: l.APIKey.GroupID,
			Status:  l.APIKey.Status,
		}
	}
	return out
}

// maskedIPPtr 返回掩码后的 IP 指针；无法解析或为空时返回 nil（宁可不显示）。
func maskedIPPtr(value *string) *string {
	if value == nil {
		return nil
	}
	masked := ip.MaskIP(*value)
	if masked == "" {
		return nil
	}
	return &masked
}

// apiKeyWithoutSecret 列表类接口用的 API Key 视图：去掉明文 key、IP 名单、最近使用 IP 与嵌套用户（含余额）。
// 管理员的调用日志列表也使用它——列表页只需要 key 的名称与状态，明文 key 应当通过用户 API Key
// 接口（受敏感读取审计）单独获取。
func apiKeyWithoutSecret(k *service.APIKey) *APIKey {
	out := APIKeyFromService(k)
	if out == nil {
		return nil
	}
	out.Key = ""
	out.IPWhitelist = nil
	out.IPBlacklist = nil
	out.LastUsedIP = nil
	out.User = nil
	return out
}
