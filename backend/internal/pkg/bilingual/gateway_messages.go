package bilingual

// gatewayMessages 网关报错双语对照表（账户 / 风控 / 计费）。
//
// 每条里有一半是代码里原有的文本，必须逐字保留（大小写可以不同，匹配不区分大小写）；
// 另一半是补上的翻译。英文翻译里不要出现 "upstream"：桌面客户端 paywall.rs 见到它就不再归类。
var gatewayMessages = []Text{
	// —— API Key 认证（server/middleware/api_key_auth.go、api_key_auth_google.go）——
	{ZH: "无效认证尝试过多，请稍后重试", EN: "Too many invalid authentication attempts; retry later"},
	{ZH: "API Key 无效", EN: "Invalid API key"},
	{ZH: "不再支持在查询参数中传递 API Key，请改用 Authorization 请求头", EN: "API key in query parameter is deprecated. Please use Authorization header instead."},
	{ZH: "不再支持查询参数 api_key，请改用 Authorization 请求头或 key 参数", EN: "Query parameter api_key is deprecated. Use Authorization header or key instead."},
	{ZH: "缺少 API Key，请通过 Authorization（Bearer）、x-api-key 或 x-goog-api-key 请求头提供", EN: "API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header"},
	{ZH: "缺少 API Key", EN: "API key is required"},
	{ZH: "API Key 认证暂时不可用，请稍后重试", EN: "API key authentication is temporarily unavailable"},
	{ZH: "API Key 校验失败", EN: "Failed to validate API key"},
	{ZH: "API Key 已停用", EN: "API key is disabled"},
	{ZH: "未找到该 API Key 所属的用户", EN: "User associated with API key not found"},
	{ZH: "账户已被停用，请登录网站查看通知或提交申诉", EN: "User account is not active. Sign in to the website to view notices or submit an appeal.",
		Aliases: []string{"User account is not active"}},
	{ZH: "API Key 所属分组已删除", EN: "The API key's group has been deleted"},
	{ZH: "API Key 所属分组已停用", EN: "The API key's group has been disabled"},
	{ZH: "API Key 所属专属分组不再允许当前用户使用", EN: "The API key's exclusive group is no longer available to this user"},
	{ZH: "API Key 未分配分组，无法使用，请联系管理员分配分组", EN: "API Key is not assigned to any group and cannot be used. Please contact the administrator to assign it to a group."},
	{ZH: "当前分组没有有效订阅", EN: "No active subscription found for this group"},
	{ZH: "订阅用量窗口维护失败，请稍后重试", EN: "Failed to maintain subscription usage windows"},
	{ZH: "API key 已过期", EN: "API key has expired"},
	{ZH: "API key 额度已用完", EN: "API key quota exhausted"},
	{ZH: "账户余额不足", EN: "Insufficient account balance"},

	// —— 订阅 / 计费（service 层 infraerrors 消息，经 subscription 校验与 billingErrorDetails 输出）——
	{ZH: "订阅已过期", EN: "subscription has expired"},
	{ZH: "订阅已暂停", EN: "subscription is suspended"},
	{ZH: "订阅无效或已过期", EN: "subscription is invalid or expired"},
	{ZH: "已超出每日用量上限", EN: "daily usage limit exceeded"},
	{ZH: "已超出每周用量上限", EN: "weekly usage limit exceeded"},
	{ZH: "已超出每月用量上限", EN: "monthly usage limit exceeded"},
	{ZH: "计费服务暂时不可用，请稍后重试", EN: "Billing service temporarily unavailable. Please retry later."},
	{ZH: "api key 5小时限额已用完", EN: "API key 5-hour limit exhausted"},
	{ZH: "api key 日限额已用完", EN: "API key daily limit exhausted"},
	{ZH: "api key 7天限额已用完", EN: "API key 7-day limit exhausted"},
	{ZH: "分组每分钟请求数已达上限", EN: "group requests-per-minute limit exceeded"},
	{ZH: "用户每分钟请求数已达上限", EN: "user requests-per-minute limit exceeded"},
	{ZH: "该平台今日用量额度已用完", EN: "Daily usage quota exhausted for this platform."},
	{ZH: "该平台本周用量额度已用完", EN: "Weekly usage quota exhausted for this platform."},
	{ZH: "该平台本月用量额度已用完", EN: "Monthly usage quota exhausted for this platform."},
	{ZH: "余额不足", EN: "insufficient balance"},
	{ZH: "计费错误", EN: "Billing error"},

	// —— 内容审计 / 提示词审计 ——
	// 默认拦截文案：中文是 Go 默认值，英文是前端英文界面保存的默认值（i18n admin.riskControl.defaultBlockMessage）。
	{ZH: "内容审计命中风险规则，请调整输入后重试", EN: "Content audit matched a risk rule. Please adjust your input and try again."},
	{ZH: "请求被内容安全策略拦截", EN: "Request blocked by content policy"},
	{ZH: "内容审计拦截了该请求", EN: "content moderation blocked this request"},
	{ZH: "提示词安全审计拒绝了该请求，请调整输入后重试", EN: "Prompt security audit rejected this request. Please adjust your input and try again."},
	{ZH: "提示词安全审计暂时不可用，请稍后重试", EN: "Prompt security audit is temporarily unavailable. Please retry later."},
}
