/**
 * Public pricing API — 落地页定价区的数据源。
 *
 * 与 `api/modelCatalog.ts` 的区别：这个接口**不需要登录**，返回的是公开分组
 * （活跃 + 非专属 + 非免费订阅）下的模型单价与分组默认倍率。
 *
 * 后端契约保证该接口永不返回 401——否则 `api/client.ts` 的全局 401 拦截会把
 * 落地页访客强制跳转到 /login。开关关闭或后端异常时返回 `enabled: false` /
 * 空 items，调用方据此回落到静态文案。
 */

import { apiClient } from './client'

/** 价格字段，单位均为 USD。null 表示该维度不适用或未配置。 */
export interface PublicPricing {
  input_per_mtok_usd: number | null
  output_per_mtok_usd: number | null
  cache_write_per_mtok_usd: number | null
  cache_read_per_mtok_usd: number | null
  per_request_usd: number | null
  per_image_usd: number | null
  /** 是否存在官方参考价。为 false 时不应展示「比官方省 X%」。 */
  has_reference: boolean
}

export interface PublicPricingComparison {
  /** 仅在 official_pricing.has_reference 为 true 时下发。 */
  savings_percent: number | null
  is_cheaper_than_official: boolean
}

/** 公开可见的分组引用（不含内部 ID）。 */
export interface PublicPricingGroupRef {
  name: string
  rate_multiplier: number
  rate_source: string
}

export interface PublicPricingItem {
  model: string
  display_name: string
  platform: string
  billing_mode: string
  group: PublicPricingGroupRef
  available_group_count: number
  official_pricing: PublicPricing
  effective_pricing_usd: PublicPricing
  comparison: PublicPricingComparison
}

export interface PublicPricingSummary {
  total_models: number
  max_savings_percent: number
}

export interface PublicPricingResponse {
  /** 后端开关状态。false 表示管理员显式关闭了公开定价。 */
  enabled: boolean
  items: PublicPricingItem[]
  summary: PublicPricingSummary
}

/** 拉取公开定价目录（无需认证）。 */
export async function getPublicPricing(options?: {
  signal?: AbortSignal
}): Promise<PublicPricingResponse> {
  const { data } = await apiClient.get<PublicPricingResponse>('/pricing/public', {
    signal: options?.signal
  })
  return data
}

/**
 * 构造支付服务的匿名汇率端点地址。
 *
 * 与 `modelCatalog.buildPaymentCenterUserApiUrl` 的区别：这里**不带 user_id / token**，
 * 目标是 /api/public-config —— 一个只返回汇率、不含任何用户数据的匿名端点。
 * 默认部署下支付服务由网关反代在 /pay，因此通常是同源请求。
 */
export function buildPaymentCenterPublicConfigUrl(input: {
  purchaseSubscriptionUrl?: string | null
  baseOrigin?: string
}): string | null {
  const raw = (input.purchaseSubscriptionUrl || '').trim()
  const fallbackOrigin =
    input.baseOrigin || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost')

  try {
    const baseUrl = raw ? new URL(raw, fallbackOrigin) : new URL('/pay', fallbackOrigin)
    if (baseUrl.protocol !== 'http:' && baseUrl.protocol !== 'https:') {
      return null
    }

    const normalizedBasePath = baseUrl.pathname.replace(/\/+$/, '')
    const apiPath = normalizedBasePath
      ? `${normalizedBasePath}/api/public-config`
      : '/api/public-config'
    const apiUrl = new URL(baseUrl.origin)
    apiUrl.pathname = apiPath
    return apiUrl.toString()
  } catch {
    return null
  }
}

export interface PublicCurrencyRates {
  /** 1 USD 折合多少人民币。null 表示拿不到汇率，调用方应回退到美元展示。 */
  balanceCreditCnyPerUsd: number | null
  usdExchangeRate: number | null
}

function toPositiveNumber(value: number | string | null | undefined): number | null {
  if (value == null) return null
  const n = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(n) && n > 0 ? n : null
}

/**
 * 匿名拉取汇率。任何失败（未部署支付服务、未配置地址、超时、跨域）都返回 null 汇率，
 * 由调用方回退到美元展示——落地页不能因为拿不到汇率就报错或空白。
 */
export async function fetchPublicCurrencyRates(input: {
  purchaseSubscriptionUrl?: string | null
  baseOrigin?: string
  signal?: AbortSignal
}): Promise<PublicCurrencyRates> {
  const empty: PublicCurrencyRates = { balanceCreditCnyPerUsd: null, usdExchangeRate: null }

  const url = buildPaymentCenterPublicConfigUrl(input)
  if (!url) return empty

  try {
    const response = await fetch(url, { method: 'GET', mode: 'cors', signal: input.signal })
    if (!response.ok) return empty

    const payload = (await response.json()) as {
      balanceCreditCnyPerUsd?: number | string | null
      usdExchangeRate?: number | string | null
    } | null

    return {
      balanceCreditCnyPerUsd: toPositiveNumber(payload?.balanceCreditCnyPerUsd),
      usdExchangeRate: toPositiveNumber(payload?.usdExchangeRate)
    }
  } catch {
    return empty
  }
}

export const publicPricingAPI = { getPublicPricing, fetchPublicCurrencyRates }
