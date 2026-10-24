package service

// fork：未勾选分组仅审计
//
// 审计范围选「指定分组」时，未勾选的分组默认完全跳过审计。开启
// UnselectedGroupsAuditOnly 后，这些分组（以及没有分组的请求）也会被审计并
// 写入日志，但只留证据：不拦截、不发邮件或站内信、不计违规次数、不自动封禁，
// 也不登记命中哈希。API 审计一律走异步队列，不给这些分组的请求加延迟。

// ContentModerationModeGroupAuditOnly 标记未勾选分组只审计的日志行；
// CountFlaggedByUserSince 的 SQL 字面量 'group_audit_only' 须与它一致，
// 这样分组后来被勾选时，此前只审计的命中也不会变成处罚依据。
const ContentModerationModeGroupAuditOnly = "group_audit_only"

// groupAuditOnly 报告该分组是否在审计范围外、但仍需只审计不警告。
func (cfg *ContentModerationConfig) groupAuditOnly(groupID *int64) bool {
	return cfg != nil && !cfg.AllGroups && cfg.UnselectedGroupsAuditOnly && !cfg.includesGroup(groupID)
}

// logOnly 报告这次审计是否只留证据：风控白名单用户，或未勾选分组只审计。
func (in *ContentModerationCheckInput) logOnly() bool {
	return in.riskControlLogOnly || in.groupAuditOnly
}

// contentModerationLogOnlyMode 报告日志行是否属于只留证据、不产生处罚的模式。
func contentModerationLogOnlyMode(mode string) bool {
	return mode == ContentModerationModeRiskControlLogOnly || mode == ContentModerationModeGroupAuditOnly
}
