// fork：滚动数字用的紧凑格式（K / M / B）。
// 单位锁定为目标值的单位：1.2B 从 0.0B 滚到 1.2B，中途不会在 K、M、B 之间跳来跳去
// （参考 TokenRouter HomeView 的首页计数）。

interface CompactUnit {
  divisor: number
  suffix: string
}

const UNITS: CompactUnit[] = [
  { divisor: 1_000_000_000, suffix: 'B' },
  { divisor: 1_000_000, suffix: 'M' },
  { divisor: 1_000, suffix: 'K' },
]

function unitFor(value: number): CompactUnit | null {
  const abs = Math.abs(value)
  return UNITS.find(unit => abs >= unit.divisor) ?? null
}

/** 按 value 自己的单位格式化，例如 1234 → 1.2K。 */
export function formatCompact(value: number, fractionDigits = 1): string {
  if (!Number.isFinite(value)) return '0'
  const unit = unitFor(value)
  if (!unit) return Math.round(value).toLocaleString()
  return `${(value / unit.divisor).toFixed(fractionDigits)}${unit.suffix}`
}

/** 返回一个把任意值都按 target 的单位格式化的函数，给滚动数字用。 */
export function compactFormatterFor(target: number, fractionDigits = 1): (value: number) => string {
  const unit = unitFor(target)
  if (!unit) return (value: number) => Math.round(value).toLocaleString()
  return (value: number) => `${(value / unit.divisor).toFixed(fractionDigits)}${unit.suffix}`
}
