import { apiClient } from './client'

export type ModelCatalogBillingMode = 'token' | 'per_request' | 'image' | 'video'
export type ModelCatalogSortKey = 'effective_price_asc' | 'model_asc'

export interface ModelCatalogSummary {
  total_models: number
  token_models: number
  non_token_models: number
  best_savings_model: string
  max_savings_percent: number
}

export interface ModelCatalogGroupRef {
  id: number
  name: string
  rate_multiplier: number
  rate_source: 'group_default' | 'user_override'
}

export interface ModelCatalogPricing {
  input_per_mtok_usd: number | null
  output_per_mtok_usd: number | null
  cache_write_per_mtok_usd: number | null
  cache_read_per_mtok_usd: number | null
  per_request_usd: number | null
  per_image_usd: number | null
  /** 视频按秒计费：总价 = 每秒价 x 时长 x 条数 */
  per_second_usd: number | null
  source: string
  has_reference: boolean
}

export interface ModelCatalogComparison {
  savings_percent: number | null
  is_cheaper_than_official: boolean
  delta_input_per_mtok_usd: number | null
  delta_output_per_mtok_usd: number | null
  delta_per_request_usd: number | null
  delta_per_image_usd: number | null
  delta_per_second_usd: number | null
}

export interface ModelCatalogPriceInterval {
  min_tokens: number
  max_tokens: number | null
  tier_label: string
  input_per_mtok_usd: number | null
  output_per_mtok_usd: number | null
  cache_write_per_mtok_usd: number | null
  cache_read_per_mtok_usd: number | null
  per_request_usd: number | null
  per_image_usd: number | null
}

/**
 * 图片/视频模型按分辨率分档的单价。
 *
 * 这条分档轴和 `intervals` 不同：`intervals` 按上下文 token 区间分档，
 * `min_tokens` / `max_tokens` 对分辨率档位没有意义，用它渲染会得到 ">= -"。
 */
export interface ModelCatalogMediaTier {
  /** 图片为 1K/2K/4K，视频为 480p/720p/1080p */
  tier: string
  official_usd: number | null
  effective_usd: number | null
  /** 请求未指定尺寸时，计费实际落到的档位 */
  is_default_tier: boolean
}

export type ModelCatalogMediaUnit = 'image' | 'second' | ''

export interface ModelCatalogPricingDetails {
  supports_prompt_caching: boolean
  has_long_context_multiplier: boolean
  long_context_input_threshold: number
  intervals: ModelCatalogPriceInterval[]
  media_tiers: ModelCatalogMediaTier[]
  media_unit: ModelCatalogMediaUnit
}

export interface ModelCatalogGroupCompanion {
  group: ModelCatalogGroupRef
  effective_pricing_usd: ModelCatalogPricing
  comparison: ModelCatalogComparison
}

export interface ModelCatalogItem {
  model: string
  display_name: string
  platform: string
  billing_mode: ModelCatalogBillingMode
  best_group: ModelCatalogGroupRef
  available_group_count: number
  official_pricing: ModelCatalogPricing
  effective_pricing_usd: ModelCatalogPricing
  comparison: ModelCatalogComparison
  pricing_details: ModelCatalogPricingDetails
  other_groups: ModelCatalogGroupCompanion[]
}

export interface ModelCatalogResponse {
  items: ModelCatalogItem[]
  summary: ModelCatalogSummary
}

export interface ModelCatalogFilters {
  search: string
  groupId: number | null
  platform: string
  billingMode: string
}

export interface PaymentConfigResult {
  balanceCreditCnyPerUsd: number | null
  usdExchangeRate: number | null
  error: string | null
}

export async function getCatalog(): Promise<ModelCatalogResponse> {
  const { data } = await apiClient.get<ModelCatalogResponse>('/models/catalog')
  return data
}

export function getCatalogItemKey(item: ModelCatalogItem): string {
  return `${item.best_group.id}:${item.model}`
}

export function getPrimaryEffectivePrice(item: ModelCatalogItem): number | null {
  return getPrimaryPrice(item.effective_pricing_usd, item.billing_mode)
}

export function getPrimaryOfficialPrice(item: ModelCatalogItem): number | null {
  return getPrimaryPrice(item.official_pricing, item.billing_mode)
}

export function getPrimaryPrice(pricing: ModelCatalogPricing, billingMode: string): number | null {
  if (billingMode === 'per_request') {
    return pricing.per_request_usd ?? null
  }
  if (billingMode === 'image') {
    return pricing.per_image_usd ?? null
  }
  if (billingMode === 'video') {
    return pricing.per_second_usd ?? null
  }
  return pricing.input_per_mtok_usd ?? null
}

/**
 * 各计费模式的排序分桶。
 *
 * 不同模式的主价单位不同（$/1M tokens、$/次、$/张、$/秒），直接比数值大小没有
 * 意义 —— $0.134/张 会排在 $2.50/1M tokens 前面，读起来像"更便宜"。所以先按
 * 模式分桶，只在同一单位内部比价格。
 */
const BILLING_MODE_SORT_RANK: Record<string, number> = {
  token: 0,
  per_request: 1,
  image: 2,
  video: 3,
}

export function getBillingModeSortRank(billingMode: string): number {
  return BILLING_MODE_SORT_RANK[billingMode] ?? BILLING_MODE_SORT_RANK.token
}

export function filterModelCatalogItems(
  items: ModelCatalogItem[],
  filters: ModelCatalogFilters,
): ModelCatalogItem[] {
  const keywords = normalizeSearch(filters.search)
    .split(/\s+/)
    .filter(Boolean)
  const targetPlatform = normalizeSearch(filters.platform)
  const targetMode = normalizeSearch(filters.billingMode)

  return items.filter((item) => {
    if (keywords.length > 0) {
      const haystack = [
        item.model,
        item.display_name,
        item.platform,
        item.best_group.name,
        item.best_group.id,
      ]
        .map(value => String(value))
        .join(' ')
        .toLowerCase()
      if (!keywords.every(keyword => haystack.includes(keyword))) {
        return false
      }
    }

    if (targetPlatform && targetPlatform !== 'all' && normalizeSearch(item.platform) !== targetPlatform) {
      return false
    }

    if (targetMode && targetMode !== 'all' && normalizeSearch(item.billing_mode) !== targetMode) {
      return false
    }

    if (filters.groupId != null && item.best_group.id !== filters.groupId) {
      return false
    }

    return true
  })
}

export function sortModelCatalogItems(
  items: ModelCatalogItem[],
  sortKey: ModelCatalogSortKey,
): ModelCatalogItem[] {
  return [...items].sort((a, b) => compareModelCatalogItems(a, b, sortKey))
}

export function compareModelCatalogItems(
  a: ModelCatalogItem,
  b: ModelCatalogItem,
  sortKey: ModelCatalogSortKey,
): number {
  if (sortKey === 'model_asc') {
    return compareModelCatalogFallback(a, b)
  }

  const byMode = getBillingModeSortRank(a.billing_mode) - getBillingModeSortRank(b.billing_mode)
  if (byMode !== 0) {
    return byMode
  }

  const byEffective = compareNullableNumberAsc(getPrimaryEffectivePrice(a), getPrimaryEffectivePrice(b))
  if (byEffective !== 0) {
    return byEffective
  }

  return compareModelCatalogFallback(a, b)
}

export function convertUsdAmountToCny(
  amountUsd: number | null | undefined,
  balanceCreditCnyPerUsd: number | null | undefined,
): number | null {
  if (amountUsd == null || balanceCreditCnyPerUsd == null) {
    return null
  }

  if (!Number.isFinite(amountUsd) || !Number.isFinite(balanceCreditCnyPerUsd) || balanceCreditCnyPerUsd <= 0) {
    return null
  }

  return amountUsd * balanceCreditCnyPerUsd
}

export function convertCnyAmountToUsd(
  amountCny: number | null | undefined,
  usdExchangeRate: number | null | undefined,
): number | null {
  if (amountCny == null || usdExchangeRate == null) {
    return null
  }

  if (!Number.isFinite(amountCny) || !Number.isFinite(usdExchangeRate) || usdExchangeRate <= 0) {
    return null
  }

  return amountCny / usdExchangeRate
}

export function normalizePaymentCenterOrigin(
  purchaseSubscriptionUrl: string | null | undefined,
  baseOrigin?: string,
): string | null {
  const raw = (purchaseSubscriptionUrl || '').trim()
  if (!raw) {
    return null
  }

  const fallbackOrigin =
    baseOrigin || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost')

  try {
    const url = new URL(raw, fallbackOrigin)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return null
    }
    return url.origin
  } catch {
    return null
  }
}

export function buildPaymentCenterUserApiUrl(input: {
  purchaseSubscriptionUrl?: string | null
  userId?: number | null
  token?: string | null
  baseOrigin?: string
}): string | null {
  if (!input.userId || !input.token) {
    return null
  }

  const raw = (input.purchaseSubscriptionUrl || '').trim()
  const fallbackOrigin =
    input.baseOrigin || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost')

  try {
    const baseUrl = raw ? new URL(raw, fallbackOrigin) : new URL('/pay', fallbackOrigin)
    if (baseUrl.protocol !== 'http:' && baseUrl.protocol !== 'https:') {
      return null
    }

    const normalizedBasePath = baseUrl.pathname.replace(/\/+$/, '')
    const apiPath = normalizedBasePath ? `${normalizedBasePath}/api/user` : '/api/user'
    const apiUrl = new URL(baseUrl.origin)
    apiUrl.pathname = apiPath
    apiUrl.searchParams.set('user_id', String(input.userId))
    apiUrl.searchParams.set('token', input.token)

    return apiUrl.toString()
  } catch {
    return null
  }
}

export async function fetchBalanceCreditCnyPerUsd(input: {
  purchaseSubscriptionUrl?: string | null
  userId?: number | null
  token?: string | null
  locale?: string
  baseOrigin?: string
}): Promise<PaymentConfigResult> {
  if (!input.userId || !input.token) {
    return { balanceCreditCnyPerUsd: null, usdExchangeRate: null, error: 'missing_auth' }
  }

  const url = buildPaymentCenterUserApiUrl(input)
  if (!url) {
    return { balanceCreditCnyPerUsd: null, usdExchangeRate: null, error: 'missing_origin' }
  }

  try {
    const response = await fetch(url, {
      method: 'GET',
      mode: 'cors',
    })

    if (!response.ok) {
      return { balanceCreditCnyPerUsd: null, usdExchangeRate: null, error: `http_${response.status}` }
    }

    const payload = await response.json() as {
      config?: {
        balanceCreditCnyPerUsd?: number | string | null
        usdExchangeRate?: number | string | null
      }
    }

    const parseNullablePositiveNumber = (value: number | string | null | undefined): number | null => {
      const numericValue =
        typeof value === 'number'
          ? value
          : typeof value === 'string'
            ? Number(value)
            : NaN
      return Number.isFinite(numericValue) && numericValue > 0 ? numericValue : null
    }

    const balanceCreditCnyPerUsd = parseNullablePositiveNumber(payload?.config?.balanceCreditCnyPerUsd)
    const usdExchangeRate = parseNullablePositiveNumber(payload?.config?.usdExchangeRate)

    if (balanceCreditCnyPerUsd == null) {
      return { balanceCreditCnyPerUsd: null, usdExchangeRate, error: 'missing_rate' }
    }

    return { balanceCreditCnyPerUsd, usdExchangeRate, error: null }
  } catch {
    return { balanceCreditCnyPerUsd: null, usdExchangeRate: null, error: 'request_failed' }
  }
}

function compareModelCatalogFallback(a: ModelCatalogItem, b: ModelCatalogItem): number {
  const byModel = a.display_name.localeCompare(b.display_name, undefined, { sensitivity: 'base' })
  if (byModel !== 0) {
    return byModel
  }

  const byGroup = a.best_group.name.localeCompare(b.best_group.name, undefined, { sensitivity: 'base' })
  if (byGroup !== 0) {
    return byGroup
  }

  return a.model.localeCompare(b.model, undefined, { sensitivity: 'base' })
}

function compareNullableNumberAsc(a: number | null, b: number | null): number {
  if (a == null && b == null) return 0
  if (a == null) return 1
  if (b == null) return -1
  if (a === b) return 0
  return a - b
}

function normalizeSearch(value: string | null | undefined): string {
  return (value || '').trim().toLowerCase()
}

export const modelCatalogAPI = {
  getCatalog,
  fetchBalanceCreditCnyPerUsd,
}

export default modelCatalogAPI
