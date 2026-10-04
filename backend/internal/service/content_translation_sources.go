package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

// contentTranslationSources 是一轮扫描收集到的文案源：hash → 规范化原文。
type contentTranslationSources map[string]string

func (s contentTranslationSources) add(text string) {
	normalized := normalizeContentSource(text)
	if !contentSourceUsable(normalized) {
		return
	}
	s[contentSourceHash(normalized)] = normalized
}

// collectContentTranslationSources 收集当前所有需要翻译的管理员文案。全部只读，不在上游代码里加钩子。
//
// 收集：启用中分组的名称与描述、启用中渠道的描述、有效公告的标题与正文、公开设置里的站点副标题、
// 联系方式、用户可见的自定义菜单名、自定义端点的名称与说明、登录协议文档的标题与正文，以及外部登记的文案。
// 不收集：站点名称、首页自定义 HTML（未经过滤直接渲染）、站内信、工单。
//
// 单个来源读失败只记日志并跳过，不影响其它来源；对应文本暂时会被当成「未知」，查询时触发重扫。
func (s *ContentTranslationService) collectContentTranslationSources(ctx context.Context) (contentTranslationSources, error) {
	sources := contentTranslationSources{}
	warn := func(source string, err error) {
		logger.L().Warn("content_translation.collect_failed", zap.String("source", source), zap.Error(err))
	}

	if s.groupRepo != nil {
		groups, err := s.groupRepo.ListActive(ctx)
		if err != nil {
			warn("groups", err)
		}
		for _, group := range groups {
			sources.add(group.Name)
			sources.add(group.Description)
		}
	}

	if s.channelRepo != nil {
		channels, err := s.channelRepo.ListAll(ctx)
		if err != nil {
			warn("channels", err)
		}
		for _, channel := range channels {
			if channel.Status != StatusActive {
				continue
			}
			sources.add(channel.Description)
		}
	}

	if s.announcementRepo != nil {
		announcements, err := s.announcementRepo.ListActive(ctx, s.now())
		if err != nil {
			warn("announcements", err)
		}
		for _, announcement := range announcements {
			sources.add(announcement.Title)
			sources.add(announcement.Content)
		}
	}

	if s.settingService != nil {
		settings, err := s.settingService.GetPublicSettings(ctx)
		if err != nil {
			warn("public_settings", err)
		} else if settings != nil {
			sources.add(settings.SiteSubtitle)
			sources.add(settings.ContactInfo)
			for _, label := range userVisibleCustomMenuLabels(settings.CustomMenuItems) {
				sources.add(label)
			}
			for _, endpoint := range parseContentTranslationEndpoints(settings.CustomEndpoints) {
				sources.add(endpoint.Name)
				sources.add(endpoint.Description)
			}
			for _, doc := range settings.LoginAgreementDocuments {
				sources.add(doc.Title)
				sources.add(doc.ContentMD)
			}
		}
	}

	if s.repo != nil {
		external, err := s.repo.ListNamespaceSources(ctx)
		if err != nil {
			warn("external", err)
		}
		for _, item := range external {
			sources.add(item.SourceText)
		}
	}

	return sources, nil
}

// userVisibleCustomMenuLabels 解析自定义菜单 JSON，只取用户可见项的名称
// （与 dto.ParseUserVisibleMenuItems 的规则一致；service 不能引用 dto）。
func userVisibleCustomMenuLabels(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var items []struct {
		Label      string `json:"label"`
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	labels := make([]string, 0, len(items))
	for _, item := range items {
		if item.Visibility == "admin" {
			continue
		}
		labels = append(labels, item.Label)
	}
	return labels
}

type contentTranslationEndpoint struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func parseContentTranslationEndpoints(raw string) []contentTranslationEndpoint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var endpoints []contentTranslationEndpoint
	if err := json.Unmarshal([]byte(raw), &endpoints); err != nil {
		return nil
	}
	return endpoints
}
