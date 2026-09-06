package service

// withDefaultMaxReasoningEffortMultiplier 在渠道模型定价未显式配置 max 推理档倍率时，
// 按模型默认值补齐。上游把它放在 model_plaza_service.go 里，本 fork 已移除 model plaza，
// 因此单独保留在这里供 channel_available.go 使用。
func withDefaultMaxReasoningEffortMultiplier(pricing *ChannelModelPricing, model string) *ChannelModelPricing {
	if pricing == nil || pricing.MaxReasoningEffortMultiplier != nil {
		return pricing
	}
	multiplier := defaultMaxReasoningEffortMultiplier(model)
	if multiplier == nil {
		return pricing
	}
	cloned := pricing.Clone()
	cloned.MaxReasoningEffortMultiplier = multiplier
	return &cloned
}
