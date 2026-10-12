// fork：排行榜的指标取值与格式化，领奖台和列表共用。
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

/** 除排序指标外的其他指标；花费只在后台允许时出现。 */
export function secondaryMetricEntries(
  item: LeaderboardUserItem,
  metric: LeaderboardMetric,
  showCost: boolean,
): Array<{ metric: LeaderboardMetric; value: number }> {
  const metrics: LeaderboardMetric[] = showCost ? ['tokens', 'requests', 'cost'] : ['tokens', 'requests']
  return metrics.filter(m => m !== metric).map(m => ({ metric: m, value: metricValue(item, m) }))
}
