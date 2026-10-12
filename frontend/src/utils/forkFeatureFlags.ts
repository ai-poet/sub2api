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
  // 首页公开统计：默认关闭
  publicStats: { key: 'public_stats_enabled', mode: 'opt-in', label: 'Public Stats' },
} as const satisfies Record<string, FeatureFlagDefinition>

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
