// fork：管理员仪表盘的数字格式，取值规则与 TokenRouter 相同
// （views/admin/DashboardView.vue 的 formatNumber / costDigits，utils/format.ts 的 formatTokens）。金额以美元显示。
import { getLocale } from '@/i18n'
import { formatUsd } from '@/components/user/dashboard/fork/format'

export const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

/** 整数加千分位。 */
export const formatNumber = (value: number | null | undefined): string =>
  toFiniteNumber(value).toLocaleString(getLocale())

/** Token 数量：K / M / B 两位小数，小于 1000 用千分位，如 "950"、"1.50K"、"3.25M"、"1.20B"。 */
export const formatTokens = (value: number | null | undefined): string => {
  if (value === null || value === undefined) return '0'
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString(getLocale())
}

/** 统计卡金额的小数位：1 美元以上 2 位，1 分以上 3 位，更小的 4 位。 */
export const costDigits = (value: number | null | undefined): number => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) return 2
  if (safeValue >= 1) return 2
  if (safeValue >= 0.01) return 3
  return 4
}

/** 统计卡金额：带 $ 和千分位，小数位随金额大小变化。 */
export const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  return formatUsd(safeValue, costDigits(safeValue))
}
