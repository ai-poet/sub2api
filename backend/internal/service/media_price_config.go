package service

func imagePriceConfigFromAPIKey(apiKey *APIKey) *ImagePriceConfig {
	if apiKey == nil {
		return nil
	}
	return imagePriceConfigFromGroup(apiKey.Group)
}

// imagePriceConfigFromGroup 按分组构建图片计费配置。模型广场走的是分组而非
// API Key，但单价必须与计费口径同源，所以两条路径共用同一份字段映射。
func imagePriceConfigFromGroup(group *Group) *ImagePriceConfig {
	if group == nil {
		return nil
	}
	return &ImagePriceConfig{
		Price1K: group.ImagePrice1K,
		Price2K: group.ImagePrice2K,
		Price4K: group.ImagePrice4K,
	}
}

func apiKeyHasConfiguredImagePrice(apiKey *APIKey, imageSize string) bool {
	return apiKey != nil && apiKey.Group != nil && apiKey.Group.GetImagePrice(imageSize) != nil
}

func videoPriceConfigFromAPIKey(apiKey *APIKey) *VideoPriceConfig {
	if apiKey == nil {
		return nil
	}
	return videoPriceConfigFromGroup(apiKey.Group)
}

// videoPriceConfigFromGroup 按分组构建视频计费配置，口径同 imagePriceConfigFromGroup。
func videoPriceConfigFromGroup(group *Group) *VideoPriceConfig {
	if group == nil {
		return nil
	}
	return &VideoPriceConfig{
		Price480P:   group.VideoPrice480P,
		Price720P:   group.VideoPrice720P,
		Price1080P:  group.VideoPrice1080P,
		ModelPrices: group.VideoModelPrices,
	}
}

func apiKeyHasConfiguredVideoPrice(apiKey *APIKey, model, resolution string) bool {
	return apiKey != nil && apiKey.Group != nil && apiKey.Group.GetVideoPriceForModel(model, resolution) != nil
}

func webSearchPricePerCallFromAPIKey(apiKey *APIKey) *float64 {
	if apiKey == nil || apiKey.Group == nil {
		return nil
	}
	return apiKey.Group.WebSearchPricePerCall
}

func groupSearchPricePer1kFromAPIKey(apiKey *APIKey) *float64 {
	if apiKey == nil || apiKey.Group == nil {
		return nil
	}
	return apiKey.Group.GetSearchPricePer1k()
}

func groupAudioPriceConfigFromAPIKey(apiKey *APIKey) *audioPriceConfig {
	if apiKey == nil || apiKey.Group == nil {
		return nil
	}
	g := apiKey.Group
	return &audioPriceConfig{
		RealtimePerMin: g.AudioRealtimePricePerMin,
		TTSPerMChars:   g.AudioTTSPricePerMillionChars,
		STTPerHour:     g.AudioSTTPricePerHour,
	}
}
