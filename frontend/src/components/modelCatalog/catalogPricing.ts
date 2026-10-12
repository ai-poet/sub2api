// fork：模型广场的价格格式。展示价沿用原有规则：拿到支付换算配置时显示实付价
// （中文按人民币，英文把人民币按汇率折回美元），否则显示美元余额价。

import { convertCnyAmountToUsd, convertUsdAmountToCny } from '@/api/modelCatalog'

export interface CatalogPriceContext {
  /** 1 美元余额折合多少人民币；null 时显示美元余额价 */
  cnyPerUsd: number | null
  usdExchangeRate: number | null
  locale: string
}

export function formatCatalogNumber(value: number): string {
  if (value === 0) return '0.0000'
  if (Math.abs(value) >= 100) return value.toFixed(2)
  if (Math.abs(value) >= 10) return value.toFixed(3)
  return value.toFixed(4)
}

export function formatUsd(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—'
  return `$${formatCatalogNumber(value)}`
}

/** 展示价：余额美元价换算成用户实际要付的金额。 */
export function formatCharge(usd: number | null | undefined, ctx: CatalogPriceContext): string {
  if (usd == null || !Number.isFinite(usd)) return '—'
  if (ctx.cnyPerUsd != null) {
    const cny = convertUsdAmountToCny(usd, ctx.cnyPerUsd)
    if (ctx.locale.toLowerCase().startsWith('en')) {
      return formatUsd(convertCnyAmountToUsd(cny, ctx.usdExchangeRate))
    }
    return cny == null ? '—' : `¥${formatCatalogNumber(cny)}`
  }
  return formatUsd(usd)
}

/** 上下文长度：200000 → 200K，1048576 → 1.0M。 */
export function formatTokenCount(value: number): string {
  if (!value) return '—'
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}K`
  return `${value}`
}

/** 上下文区间标签：0-200K、≥200K。 */
export function formatTokenRange(min: number, max: number | null): string {
  if (max == null) return `≥${formatTokenCount(min)}`
  return `${min === 0 ? '0' : formatTokenCount(min)}-${formatTokenCount(max)}`
}
