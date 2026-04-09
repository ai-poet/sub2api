import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  buildPaymentCenterUserApiUrl,
  compareModelCatalogItems,
  convertCnyAmountToUsd,
  convertUsdAmountToCny,
  fetchBalanceCreditCnyPerUsd,
  filterModelCatalogItems,
  normalizePaymentCenterOrigin,
  sortModelCatalogItems,
  type ModelCatalogItem,
} from './modelCatalog'

const createItem = (overrides: Partial<ModelCatalogItem> = {}): ModelCatalogItem => ({
  model: 'gpt-5.4',
  display_name: 'GPT-5.4',
  platform: 'openai',
  billing_mode: 'token',
  best_group: {
    id: 1,
    name: 'Alpha',
    rate_multiplier: 0.8,
    rate_source: 'group_default',
  },
  available_group_count: 1,
  official_pricing: {
    input_per_mtok_usd: 2.5,
    output_per_mtok_usd: 15,
    cache_write_per_mtok_usd: 2.5,
    cache_read_per_mtok_usd: 0.25,
    per_request_usd: null,
    per_image_usd: null,
    source: 'litellm',
    has_reference: true,
  },
  effective_pricing_usd: {
    input_per_mtok_usd: 1.25,
    output_per_mtok_usd: 7.5,
    cache_write_per_mtok_usd: 1.25,
    cache_read_per_mtok_usd: 0.125,
    per_request_usd: null,
    per_image_usd: null,
    source: 'effective',
    has_reference: false,
  },
  comparison: {
    savings_percent: 0.5,
    is_cheaper_than_official: true,
    delta_input_per_mtok_usd: 1.25,
    delta_output_per_mtok_usd: 7.5,
    delta_per_request_usd: null,
    delta_per_image_usd: null,
  },
  pricing_details: {
    supports_prompt_caching: true,
    has_long_context_multiplier: false,
    long_context_input_threshold: 0,
    intervals: [],
  },
  other_groups: [],
  ...overrides,
})

describe('modelCatalog helpers', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('filters items by keyword, group, platform, and billing mode', () => {
    const items = [
      createItem(),
      createItem({
        model: 'claude-sonnet-4',
        display_name: 'Claude Sonnet 4',
        platform: 'anthropic',
        best_group: {
          id: 2,
          name: 'Beta',
          rate_multiplier: 1,
          rate_source: 'group_default',
        },
        comparison: {
          savings_percent: 0,
          is_cheaper_than_official: false,
          delta_input_per_mtok_usd: 0,
          delta_output_per_mtok_usd: 0,
          delta_per_request_usd: null,
          delta_per_image_usd: null,
        },
      }),
      createItem({
        model: 'image-gen',
        display_name: 'Image Gen',
        platform: 'openai',
        billing_mode: 'image',
        official_pricing: {
          input_per_mtok_usd: null,
          output_per_mtok_usd: null,
          cache_write_per_mtok_usd: null,
          cache_read_per_mtok_usd: null,
          per_request_usd: null,
          per_image_usd: 0.2,
          source: 'litellm',
          has_reference: true,
        },
        effective_pricing_usd: {
          input_per_mtok_usd: null,
          output_per_mtok_usd: null,
          cache_write_per_mtok_usd: null,
          cache_read_per_mtok_usd: null,
          per_request_usd: null,
          per_image_usd: 0.08,
          source: 'effective',
          has_reference: false,
        },
        comparison: {
          savings_percent: 0.6,
          is_cheaper_than_official: true,
          delta_input_per_mtok_usd: null,
          delta_output_per_mtok_usd: null,
          delta_per_request_usd: null,
          delta_per_image_usd: 0.12,
        },
      }),
    ]

    const filtered = filterModelCatalogItems(items, {
      search: 'beta claude',
      groupId: 2,
      platform: 'anthropic',
      billingMode: 'token',
    })

    expect(filtered).toHaveLength(1)
    expect(filtered[0].best_group.name).toBe('Beta')
  })

  it('sorts by effective price while keeping alphabetical fallback', () => {
    const items = [
      createItem({
        model: 'zeta',
        display_name: 'Zeta',
        best_group: { id: 10, name: 'Zulu', rate_multiplier: 1, rate_source: 'group_default' },
        comparison: {
          savings_percent: 0.3,
          is_cheaper_than_official: true,
          delta_input_per_mtok_usd: 1,
          delta_output_per_mtok_usd: 3,
          delta_per_request_usd: null,
          delta_per_image_usd: null,
        },
        effective_pricing_usd: {
          input_per_mtok_usd: 1.5,
          output_per_mtok_usd: 8,
          cache_write_per_mtok_usd: null,
          cache_read_per_mtok_usd: null,
          per_request_usd: null,
          per_image_usd: null,
          source: 'effective',
          has_reference: false,
        },
      }),
      createItem({
        model: 'alpha',
        display_name: 'Alpha',
        best_group: { id: 11, name: 'Alpha', rate_multiplier: 1, rate_source: 'group_default' },
        comparison: {
          savings_percent: 0.3,
          is_cheaper_than_official: true,
          delta_input_per_mtok_usd: 1.2,
          delta_output_per_mtok_usd: 4,
          delta_per_request_usd: null,
          delta_per_image_usd: null,
        },
        effective_pricing_usd: {
          input_per_mtok_usd: 1.1,
          output_per_mtok_usd: 7,
          cache_write_per_mtok_usd: null,
          cache_read_per_mtok_usd: null,
          per_request_usd: null,
          per_image_usd: null,
          source: 'effective',
          has_reference: false,
        },
      }),
      createItem({
        model: 'omega',
        display_name: 'Omega',
        best_group: { id: 12, name: 'Omega', rate_multiplier: 1, rate_source: 'group_default' },
        comparison: {
          savings_percent: null,
          is_cheaper_than_official: false,
          delta_input_per_mtok_usd: null,
          delta_output_per_mtok_usd: null,
          delta_per_request_usd: null,
          delta_per_image_usd: null,
        },
      }),
    ]

    expect(sortModelCatalogItems(items, 'effective_price_asc').map(item => item.model)).toEqual([
      'alpha',
      'omega',
      'zeta',
    ])

    expect(compareModelCatalogItems(items[0], items[1], 'model_asc')).toBeGreaterThan(0)
  })

  it('normalizes payment center origin and fetches balance conversion config', async () => {
    expect(normalizePaymentCenterOrigin('https://pay.example.com/pay?mode=embed')).toBe('https://pay.example.com')
    expect(normalizePaymentCenterOrigin('/pay', 'https://sub2api.example.com')).toBe('https://sub2api.example.com')
    expect(normalizePaymentCenterOrigin('')).toBeNull()
    expect(normalizePaymentCenterOrigin('javascript:alert(1)')).toBeNull()
    expect(buildPaymentCenterUserApiUrl({
      purchaseSubscriptionUrl: 'https://pay.example.com/pay',
      userId: 42,
      token: 'token-123',
      locale: 'zh-CN',
    })).toBe('https://pay.example.com/pay/api/user?user_id=42&token=token-123&lang=zh-CN')
    expect(buildPaymentCenterUserApiUrl({
      purchaseSubscriptionUrl: 'https://pay.example.com',
      userId: 42,
      token: 'token-123',
    })).toBe('https://pay.example.com/api/user?user_id=42&token=token-123')
    expect(buildPaymentCenterUserApiUrl({
      purchaseSubscriptionUrl: '',
      userId: 42,
      token: 'token-123',
      baseOrigin: 'https://sub2api.example.com',
    })).toBe('https://sub2api.example.com/pay/api/user?user_id=42&token=token-123')

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        config: {
          balanceCreditCnyPerUsd: 7.2,
          usdExchangeRate: 6.9,
        },
      }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const success = await fetchBalanceCreditCnyPerUsd({
      purchaseSubscriptionUrl: 'https://pay.example.com/pay',
      userId: 42,
      token: 'token-123',
      locale: 'zh-CN',
    })

    expect(success).toEqual({
      balanceCreditCnyPerUsd: 7.2,
      usdExchangeRate: 6.9,
      error: null,
    })
    expect(fetchMock).toHaveBeenCalledWith(
      'https://pay.example.com/pay/api/user?user_id=42&token=token-123&lang=zh-CN',
      expect.objectContaining({ method: 'GET' }),
    )

    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('cors failed')))

    const failure = await fetchBalanceCreditCnyPerUsd({
      purchaseSubscriptionUrl: 'https://pay.example.com/pay',
      userId: 42,
      token: 'token-123',
    })

    expect(failure).toEqual({
      balanceCreditCnyPerUsd: null,
      usdExchangeRate: null,
      error: 'request_failed',
    })
  })

  it('converts usd/cny amounts only when both values are valid', () => {
    expect(convertUsdAmountToCny(1.25, 7.2)).toBeCloseTo(9)
    expect(convertCnyAmountToUsd(6.9, 6.9)).toBeCloseTo(1)
    expect(convertUsdAmountToCny(null, 7.2)).toBeNull()
    expect(convertUsdAmountToCny(1.25, null)).toBeNull()
    expect(convertUsdAmountToCny(1.25, 0)).toBeNull()
    expect(convertCnyAmountToUsd(null, 6.9)).toBeNull()
    expect(convertCnyAmountToUsd(6.9, null)).toBeNull()
    expect(convertCnyAmountToUsd(6.9, 0)).toBeNull()
  })
})
