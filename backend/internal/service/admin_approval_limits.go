package service

import (
	"strconv"
	"strings"
)

// 运维写操作审批的数量上限（fork 本地功能）。
//
// 两个上限都能在站点设置里调整（approval_pending_limit_per_user / approval_batch_limit），
// admin_approval.go 里的常量只是默认值。解析时缺失 / 非法 / 越界一律回退默认，
// 保证服务层永远拿到一个正数，不会因为设置表里的脏值把审批锁死或放开成无限。

const (
	// AdminApprovalPendingLimitMax 每人待审上限允许配置的最大值。
	AdminApprovalPendingLimitMax = 1000
	// AdminApprovalBatchLimitMax 批量通过单次上限允许配置的最大值；dto.BatchApproveRequest 的绑定校验与之对齐。
	AdminApprovalBatchLimitMax = 500
)

// ApprovalLimits 生效中的审批数量上限。
type ApprovalLimits struct {
	// PendingPerUser 每个运维管理员同时最多有多少条待审申请。
	PendingPerUser int
	// Batch 批量通过单次最多处理多少条。
	Batch int
}

// DefaultApprovalLimits 未配置时的上限。
func DefaultApprovalLimits() ApprovalLimits {
	return ApprovalLimits{PendingPerUser: AdminApprovalPendingLimitPerUser, Batch: AdminApprovalBatchLimit}
}

// ApprovalLimitSettingKeys 读取上限所需的设置键。
func ApprovalLimitSettingKeys() []string {
	return []string{SettingKeyApprovalPendingLimitPerUser, SettingKeyApprovalBatchLimit}
}

// ParseApprovalLimits 从设置项解析上限；缺失、非法或越界的值回退到默认，绝不返回 0 或负数。
func ParseApprovalLimits(values map[string]string) ApprovalLimits {
	limits := DefaultApprovalLimits()
	limits.PendingPerUser = parseApprovalLimit(values[SettingKeyApprovalPendingLimitPerUser], limits.PendingPerUser, AdminApprovalPendingLimitMax)
	limits.Batch = parseApprovalLimit(values[SettingKeyApprovalBatchLimit], limits.Batch, AdminApprovalBatchLimitMax)
	return limits
}

// ValidateApprovalLimit 设置项的取值范围校验（1..max）。
func ValidateApprovalLimit(value, max int) bool {
	return value >= 1 && value <= max
}

func parseApprovalLimit(raw string, def, max int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || !ValidateApprovalLimit(v, max) {
		return def
	}
	return v
}
