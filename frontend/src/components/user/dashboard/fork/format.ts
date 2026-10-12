// fork：仪表盘金额格式。站点金额统一以美元显示，带 $ 前缀。

const formatters = new Map<number, Intl.NumberFormat>()

const formatterFor = (fractionDigits: number): Intl.NumberFormat => {
  let formatter = formatters.get(fractionDigits)
  if (!formatter) {
    formatter = new Intl.NumberFormat('en-US', {
      minimumFractionDigits: fractionDigits,
      maximumFractionDigits: fractionDigits,
    })
    formatters.set(fractionDigits, formatter)
  }
  return formatter
}

// formatUsd 按给定小数位格式化金额；不传小数位时，小于 1 的金额保留 4 位，否则 2 位。
export const formatUsd = (value: number, fractionDigits?: number): string => {
  const amount = Number.isFinite(value) ? value : 0
  const digits = fractionDigits ?? (Math.abs(amount) >= 1 ? 2 : 4)
  return `$${formatterFor(digits).format(amount)}`
}
