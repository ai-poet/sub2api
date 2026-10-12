// fork：管理员仪表盘的数字格式（参考 TokenRouter views/admin/DashboardView.vue）。
import { formatUsd } from '@/components/user/dashboard/fork/format'

export const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

/** 整数加千分位。 */
export const formatNumber = (value: number | null | undefined): string =>
  toFiniteNumber(value).toLocaleString()

/** Token 数：1.23K / 4.56M / 7.89B，千以下原样显示。 */
export const formatTokens = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1_000_000_000) return `${(safeValue / 1_000_000_000).toFixed(2)}B`
  if (safeValue >= 1_000_000) return `${(safeValue / 1_000_000).toFixed(2)}M`
  if (safeValue >= 1_000) return `${(safeValue / 1_000).toFixed(2)}K`
  return safeValue.toLocaleString()
}

/** 金额的小数位：1 美元以上 2 位，1 分以上 3 位，更小的 4 位。 */
export const costDigits = (value: number | null | undefined): number => {
  const safeValue = Math.abs(toFiniteNumber(value))
  if (safeValue >= 1) return 2
  if (safeValue >= 0.01) return 3
  return 4
}

/** 美元金额，带千分位，小数位随金额大小变化。 */
export const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  return formatUsd(safeValue, costDigits(safeValue))
}

/** 小时粒度的横轴只显示时分（2026-10-12 08:00 → 08:00），完整时间留在提示标题里。 */
export const formatHourLabel = (value: string): string => value.match(/(?:T|\s)(\d{2}:\d{2})/)?.[1] || value

/** 日粒度的横轴显示月-日（2026-10-12 → 10-12）。 */
export const formatDayLabel = (value: string): string => {
  const match = value.match(/^(?:\d{4}[-/])?(\d{1,2})[-/](\d{1,2})/)
  if (!match) return value
  return `${match[1].padStart(2, '0')}-${match[2].padStart(2, '0')}`
}

/** 耗时：1 秒以上显示两位小数的秒，否则显示毫秒。 */
export const formatDuration = (ms: number | null | undefined): string => {
  const safeValue = toFiniteNumber(ms)
  if (safeValue >= 1000) return `${(safeValue / 1000).toFixed(2)}s`
  return `${Math.round(safeValue)}ms`
}
