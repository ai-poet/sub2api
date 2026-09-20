package service

func resolveImageRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64) float64 {
	if apiKey == nil {
		return effectiveGroupMultiplier
	}
	return resolveImageRateMultiplierForGroup(apiKey.Group, effectiveGroupMultiplier)
}

// resolveImageRateMultiplierForGroup 解析图片计费倍率。模型广场只持有分组，
// 但展示价必须与扣费用的倍率一致，所以与 API Key 路径共用这一份判定。
func resolveImageRateMultiplierForGroup(group *Group, effectiveGroupMultiplier float64) float64 {
	if group != nil && group.ImageRateIndependent {
		if group.ImageRateMultiplier < 0 {
			return 0
		}
		return group.ImageRateMultiplier
	}
	return effectiveGroupMultiplier
}

func resolveVideoRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64) float64 {
	if apiKey == nil {
		return effectiveGroupMultiplier
	}
	return resolveVideoRateMultiplierForGroup(apiKey.Group, effectiveGroupMultiplier)
}

// resolveVideoRateMultiplierForGroup 口径同 resolveImageRateMultiplierForGroup。
func resolveVideoRateMultiplierForGroup(group *Group, effectiveGroupMultiplier float64) float64 {
	if group != nil && group.VideoRateIndependent {
		if group.VideoRateMultiplier < 0 {
			return 0
		}
		return group.VideoRateMultiplier
	}
	return effectiveGroupMultiplier
}
