// fork：用量排行的指标取值、格式化和前三名主题，领奖台和列表共用（样式对齐 TokenRouter UsageRankingView）。
import type { LeaderboardMetric, LeaderboardUserItem } from '@/api/leaderboard'
import { compactFormatterFor, formatCompact } from '@/utils/compactNumber'

export function metricValue(item: LeaderboardUserItem, metric: LeaderboardMetric): number {
  if (metric === 'requests') return item.requests
  if (metric === 'cost') return item.actual_cost ?? 0
  return item.tokens
}

export function formatCost(value: number): string {
  return `$${value.toFixed(value >= 100 ? 0 : 2)}`
}

export function formatMetric(metric: LeaderboardMetric, value: number): string {
  return metric === 'cost' ? formatCost(value) : formatCompact(value)
}

/** 滚动数字的格式：花费按金额，token / 请求数锁定目标值的单位。 */
export function metricFormatterFor(metric: LeaderboardMetric, target: number): (value: number) => string {
  return metric === 'cost' ? formatCost : compactFormatterFor(target)
}

/** 后台允许显示的全部指标；花费只在 show_cost 打开时出现。 */
export function visibleMetrics(showCost: boolean): LeaderboardMetric[] {
  return showCost ? ['tokens', 'requests', 'cost'] : ['tokens', 'requests']
}

/** 主指标：后台设定的排名指标，不在可见指标里时取第一个。 */
export function primaryMetricOf(sortBy: LeaderboardMetric | undefined, metrics: LeaderboardMetric[]): LeaderboardMetric {
  if (sortBy && metrics.includes(sortBy)) return sortBy
  return metrics[0] ?? 'tokens'
}

/** 主指标放第一列，其余按原顺序跟在后面。 */
export function orderedMetrics(metrics: LeaderboardMetric[], primary: LeaderboardMetric): LeaderboardMetric[] {
  return [primary, ...metrics.filter(metric => metric !== primary)]
}

export function metricGridStyle(count: number): Record<string, string> {
  return { gridTemplateColumns: `repeat(${Math.max(count, 1)}, minmax(0, 1fr))` }
}

/** 桌面端各指标列定宽，行间数值对齐。 */
export function metricWidthClass(metric: LeaderboardMetric | 'share'): string {
  switch (metric) {
    case 'requests':
      return 'sm:w-[110px]'
    case 'cost':
      return 'sm:w-[140px]'
    case 'share':
      return 'sm:w-[90px]'
    default:
      return 'sm:w-[130px]'
  }
}

export interface RankTheme {
  badge: string
  card: string
  glow: string
  icon: 'fire' | 'trendingUp' | 'badge'
}

/** 只有前三名用独立主题（金 / 玫红 / 天蓝），第四名之后是普通列表样式。 */
export function rankTheme(rank: number): RankTheme {
  if (rank === 1) {
    return {
      badge: 'bg-amber-100 text-amber-700 ring-amber-200 dark:bg-amber-500/15 dark:text-amber-200 dark:ring-amber-500/30',
      card: 'border-amber-200 bg-amber-50/70 dark:border-amber-500/30 dark:bg-amber-500/10',
      glow: 'bg-amber-400/20',
      icon: 'fire',
    }
  }
  if (rank === 2) {
    return {
      badge: 'bg-rose-100 text-rose-700 ring-rose-200 dark:bg-rose-500/15 dark:text-rose-200 dark:ring-rose-500/30',
      card: 'border-rose-200 bg-rose-50/70 dark:border-rose-500/30 dark:bg-rose-500/10',
      glow: 'bg-rose-400/20',
      icon: 'trendingUp',
    }
  }
  if (rank === 3) {
    return {
      badge: 'bg-sky-100 text-sky-700 ring-sky-200 dark:bg-sky-500/15 dark:text-sky-200 dark:ring-sky-500/30',
      card: 'border-sky-200 bg-sky-50/70 dark:border-sky-500/30 dark:bg-sky-500/10',
      glow: 'bg-sky-400/20',
      icon: 'badge',
    }
  }
  return {
    badge: 'bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600',
    card: 'border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800',
    glow: 'bg-transparent',
    icon: 'badge',
  }
}

/** 列表行的底色：前三名用主题底色，其余透明。 */
export function rankRowClass(rank: number): string {
  return rank <= 3 ? rankTheme(rank).card : 'border-transparent bg-transparent'
}

/** 「2026-10-01 - 2026-10-12」，单日只显示一个日期。 */
export function formatDateRangeLabel(start: string, end: string): string {
  return start === end ? start : `${start} - ${end}`
}
