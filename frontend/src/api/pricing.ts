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

export const publicPricingAPI = { getPublicPricing }
