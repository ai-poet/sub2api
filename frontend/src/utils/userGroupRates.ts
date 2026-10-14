/**
 * 专属倍率用户管理的纯函数（fork 本地功能）。
 */

import type { UserGroupRateBatchItem } from '@/api/admin/userGroupRates'

export type RateAdjustMode = 'set' | 'factor' | 'percent' | 'reset'

/** 倍率精度与下限，对应后端 DECIMAL(10,4) */
export const RATE_PRECISION = 4
export const RATE_MIN = 0.0001
export const RATE_MAX = 999999.9999

export function rowKey(row: { user_id: number; group_id: number }): string {
  return `${row.user_id}:${row.group_id}`
}

export function roundRate(value: number): number {
  const factor = 10 ** RATE_PRECISION
  return Math.round(value * factor) / factor
}

/** 合法的专属倍率：有限数，舍入后落在 (0, 1e6) 内 */
export function isValidRate(value: unknown): value is number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return false
  const rounded = roundRate(value)
  return rounded >= RATE_MIN && rounded <= RATE_MAX
}

/**
 * 按模式计算新倍率。reset 返回 null（恢复分组默认）；输入无效或结果越界时返回 undefined。
 *  - set：设为 value
 *  - factor：current × value
 *  - percent：current × (1 + value / 100)，如 10 表示上调 10%，-20 表示下调 20%
 * 结果舍入到 4 位小数，最低 0.0001。
 */
export function computeAdjustedRate(current: number, mode: RateAdjustMode, value: number | null | undefined): number | null | undefined {
  if (mode === 'reset') return null
  if (typeof value !== 'number' || !Number.isFinite(value)) return undefined
  let next: number
  switch (mode) {
    case 'set':
      next = value
      break
    case 'factor':
      next = current * value
      break
    case 'percent':
      next = current * (1 + value / 100)
      break
    default:
      return undefined
  }
  if (!Number.isFinite(next) || next <= 0) return undefined
  const rounded = Math.max(RATE_MIN, roundRate(next))
  return rounded > RATE_MAX ? undefined : rounded
}

/** 专属倍率相对分组倍率的变化百分比：-20 表示便宜 20%；分组倍率为 0 时无法比较 */
export function discountPercent(custom: number, groupDefault: number): number | null {
  if (!Number.isFinite(custom) || !Number.isFinite(groupDefault) || groupDefault <= 0) return null
  return Math.round((custom / groupDefault - 1) * 1000) / 10
}

export function formatRate(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-'
  return String(roundRate(value))
}

export function formatPercent(value: number | null): string {
  if (value === null) return '-'
  if (value === 0) return '0%'
  return `${value > 0 ? '+' : ''}${value}%`
}

export interface RateChangePreview<T> {
  row: T
  from: number
  /** null：恢复默认；undefined：无法计算（输入无效） */
  to: number | null | undefined
}

/** 把预览结果转成批量请求，跳过无法计算和没有变化的项 */
export function toBatchItems<T extends { user_id: number; group_id: number }>(changes: RateChangePreview<T>[]): UserGroupRateBatchItem[] {
  const items: UserGroupRateBatchItem[] = []
  for (const change of changes) {
    if (change.to === undefined) continue
    if (change.to !== null && roundRate(change.to) === roundRate(change.from)) continue
    items.push({ user_id: change.row.user_id, group_id: change.row.group_id, rate_multiplier: change.to })
  }
  return items
}
