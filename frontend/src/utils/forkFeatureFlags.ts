// fork：本地功能的开关，和上游 utils/featureFlags.ts 的注册表分开放，减少合并冲突。
// 取值来自公开设置（insights_config 派生的三个布尔值，随 HTML 注入下发）；
// 设置还没加载时按 mode 兜底：opt-out 先显示，opt-in 先隐藏。

import { makeSidebarFlag, resolveFeatureFlag, type FeatureFlagDefinition } from './featureFlags'
import type { PublicSettings } from '@/types'

export const ForkFeatureFlags = {
  // 模型热度榜：默认开启
  modelRanking: { key: 'model_ranking_enabled', mode: 'opt-out', label: 'Model Ranking' },
  // 用户用量榜：默认关闭
  usageRanking: { key: 'usage_ranking_enabled', mode: 'opt-in', label: 'Usage Ranking' },
  // 画布工作台：默认关闭，管理员在站点设置中显式开启
  canvasStudio: { key: 'canvas_studio_enabled', mode: 'opt-in', label: 'Canvas Studio' },
  // 首页公开统计：默认关闭
  publicStats: { key: 'public_stats_enabled', mode: 'opt-in', label: 'Public Stats' },
} as const satisfies Record<string, FeatureFlagDefinition>

/** 画布入口与直达路由共用同一个 opt-in 判断，缺省和未加载都保持隐藏。 */
export function isCanvasStudioEnabled(settings: Partial<PublicSettings> | null | undefined): boolean {
  return resolveFeatureFlag(settings, ForkFeatureFlags.canvasStudio)
}

/** 排行榜入口：两个榜任一开启就显示。 */
export function isLeaderboardEnabled(settings: Partial<PublicSettings> | null | undefined): boolean {
  return (
    resolveFeatureFlag(settings, ForkFeatureFlags.modelRanking) ||
    resolveFeatureFlag(settings, ForkFeatureFlags.usageRanking)
  )
}

export const flagModelRanking = makeSidebarFlag(ForkFeatureFlags.modelRanking)
export const flagUsageRanking = makeSidebarFlag(ForkFeatureFlags.usageRanking)
export const flagPublicStats = makeSidebarFlag(ForkFeatureFlags.publicStats)
