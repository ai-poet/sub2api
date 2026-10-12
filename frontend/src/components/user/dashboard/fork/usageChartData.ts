import type { TrendDataPoint } from '@/types'

// fork：仪表盘用量区的时间窗口与汇总（移植自 TokenRouter usageChartData.ts）。
// 与 TokenRouter 的差别：本项目的用户趋势接口只接受 YYYY-MM-DD，结束日包含当天，
// 所以查询时按日期覆盖整个窗口（queryDates），再用 bucketKeys 把结果裁回窗口内的时段。

// UsageRangePreset 是用量大图的快捷时间范围，custom 表示来自日期选择器的自定义范围。
export type UsageRangePreset = '24h' | '7d' | '30d' | '90d' | 'custom'

// UsageGranularity 是趋势接口支持的聚合粒度。
export type UsageGranularity = 'hour' | 'day'

// UsageRange 描述一次趋势查询的时间窗口，结束时间不包含在内。
export interface UsageRange {
  startAt: Date
  endAt: Date
  granularity: UsageGranularity
}

// UsageTotals 是一段时间内各指标的合计。
export interface UsageTotals {
  requests: number
  tokens: number
  cost: number
  // 输入侧没有 token 时命中率无意义，记为 null。
  cacheHitRate: number | null
}

const HOUR_MS = 60 * 60 * 1000
// 跨度不超过两天时按小时聚合，展示每小时的数据点。
const HOURLY_MAX_SPAN_MS = 2 * 24 * HOUR_MS

const pad = (value: number): string => String(value).padStart(2, '0')

// formatDayKey 生成与后端按天分组一致的键（YYYY-MM-DD，本地时区）。
export const formatDayKey = (date: Date): string =>
  `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`

// formatHourKey 生成与后端按小时分组一致的键（YYYY-MM-DD HH:00，本地时区）。
export const formatHourKey = (date: Date): string => `${formatDayKey(date)} ${pad(date.getHours())}:00`

// queryDates 生成覆盖窗口的查询日期；结束时间是排他边界，回退一毫秒得到最后一天。
export const queryDates = (range: UsageRange): { start_date: string; end_date: string } => ({
  start_date: formatDayKey(range.startAt),
  end_date: formatDayKey(new Date(range.endAt.getTime() - 1)),
})

const startOfDay = (date: Date): Date => new Date(date.getFullYear(), date.getMonth(), date.getDate())

const addDays = (date: Date, days: number): Date => {
  const next = new Date(date)
  next.setDate(next.getDate() + days)
  return next
}

// resolveGranularity 按窗口跨度选择粒度。
export const resolveGranularity = (startAt: Date, endAt: Date): UsageGranularity =>
  endAt.getTime() - startAt.getTime() <= HOURLY_MAX_SPAN_MS ? 'hour' : 'day'

// resolvePresetRange 把快捷范围换算成整点或整天对齐的查询窗口。
export const resolvePresetRange = (preset: Exclude<UsageRangePreset, 'custom'>, now: Date = new Date()): UsageRange => {
  if (preset === '24h') {
    const endAt = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours() + 1)
    return { startAt: new Date(endAt.getTime() - 24 * HOUR_MS), endAt, granularity: 'hour' }
  }
  const days = preset === '7d' ? 7 : preset === '30d' ? 30 : 90
  const endAt = addDays(startOfDay(now), 1)
  return { startAt: addDays(endAt, -days), endAt, granularity: 'day' }
}

// parsePickerValue 解析日期选择器的值；只有日期时取当天零点。
const parsePickerValue = (value: string): Date | null => {
  if (!value) return null
  const date = new Date(value.length === 10 ? `${value}T00:00:00` : value)
  return Number.isFinite(date.getTime()) ? date : null
}

// resolveCustomRange 把日期选择器的起止值换算成查询窗口，纯日期的结束值包含当天。
export const resolveCustomRange = (startValue: string, endValue: string): UsageRange | null => {
  const startAt = parsePickerValue(startValue)
  const parsedEnd = parsePickerValue(endValue)
  if (!startAt || !parsedEnd) return null
  const endAt = endValue.length === 10 ? addDays(parsedEnd, 1) : parsedEnd
  if (endAt.getTime() <= startAt.getTime()) return null
  return { startAt, endAt, granularity: resolveGranularity(startAt, endAt) }
}

// previousRange 返回紧邻当前窗口之前、长度相同的窗口，用于计算环比。
export const previousRange = (range: UsageRange): UsageRange => {
  if (range.granularity === 'day') {
    // 按日历天回退，跨夏令时的一天也保持整天对齐。
    const days = Math.round((startOfDay(range.endAt).getTime() - startOfDay(range.startAt).getTime()) / (24 * HOUR_MS))
    return { startAt: addDays(range.startAt, -days), endAt: range.startAt, granularity: 'day' }
  }
  const span = range.endAt.getTime() - range.startAt.getTime()
  return { startAt: new Date(range.startAt.getTime() - span), endAt: range.startAt, granularity: 'hour' }
}

// bucketKeys 列出窗口内的全部分组键，截止到当前时刻，未来的时段不画。
export const bucketKeys = (range: UsageRange, now: Date = new Date()): string[] => {
  const keys: string[] = []
  const limit = Math.min(range.endAt.getTime(), now.getTime() + 1)
  if (range.granularity === 'hour') {
    const cursor = new Date(range.startAt)
    cursor.setMinutes(0, 0, 0)
    while (cursor.getTime() < limit) {
      keys.push(formatHourKey(cursor))
      cursor.setHours(cursor.getHours() + 1)
    }
    return keys
  }
  let cursor = startOfDay(range.startAt)
  while (cursor.getTime() < limit) {
    keys.push(formatDayKey(cursor))
    cursor = addDays(cursor, 1)
  }
  return keys
}

const emptyPoint = (date: string): TrendDataPoint => ({
  date,
  requests: 0,
  input_tokens: 0,
  output_tokens: 0,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  total_tokens: 0,
  cost: 0,
  actual_cost: 0,
})

// fillTrendBuckets 按分组键补齐后端省略的空时段，保证横轴间隔均匀；窗口外的时段被丢弃。
export const fillTrendBuckets = (points: TrendDataPoint[], keys: string[]): TrendDataPoint[] => {
  const byKey = new Map(points.map((point) => [point.date, point]))
  return keys.map((key) => byKey.get(key) ?? emptyPoint(key))
}

// cacheHitRateOf 计算缓存命中率，分母是输入侧全部 token（后端的 input_tokens 已扣除缓存读取）。
export const cacheHitRateOf = (input: number, cacheCreation: number, cacheRead: number): number | null => {
  const inputSide = input + cacheCreation + cacheRead
  return inputSide > 0 ? (cacheRead / inputSide) * 100 : null
}

// summarizeTrend 汇总一段时间的各项指标。
export const summarizeTrend = (points: TrendDataPoint[]): UsageTotals => {
  let requests = 0
  let tokens = 0
  let cost = 0
  let input = 0
  let cacheCreation = 0
  let cacheRead = 0
  for (const point of points) {
    requests += point.requests
    tokens += point.total_tokens
    cost += point.actual_cost
    input += point.input_tokens
    cacheCreation += point.cache_creation_tokens
    cacheRead += point.cache_read_tokens
  }
  return { requests, tokens, cost, cacheHitRate: cacheHitRateOf(input, cacheCreation, cacheRead) }
}
