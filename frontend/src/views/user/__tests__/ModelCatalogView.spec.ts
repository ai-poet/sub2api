import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { ModelCatalogItem, ModelCatalogMediaTier } from '@/api/modelCatalog'
import ModelCatalogView from '../ModelCatalogView.vue'

const { getCatalog, fetchBalanceCreditCnyPerUsd, showError } = vi.hoisted(() => ({
  getCatalog: vi.fn(),
  fetchBalanceCreditCnyPerUsd: vi.fn(),
  showError: vi.fn(),
}))

// 只翻译用例断言要读的几个键，其余原样返回键名，方便定位。
const messages: Record<string, string> = {
  'modelCatalog.units.perImage': 'Per image',
  'modelCatalog.units.perSecond': 'Per second of video',
  'modelCatalog.units.perMillionTokens': 'Per 1M tokens',
  'modelCatalog.priceColumns.official': 'Official',
  'modelCatalog.priceColumns.balance': 'Balance',
  'modelCatalog.defaultTierHint': 'Billed at this tier when no size is given',
  'modelCatalog.intervalDefaultLabel': 'Default tier',
}

vi.mock('@/api/modelCatalog', async () => {
  const actual = await vi.importActual<typeof import('@/api/modelCatalog')>('@/api/modelCatalog')
  return {
    ...actual,
    modelCatalogAPI: { getCatalog, fetchBalanceCreditCnyPerUsd },
    fetchBalanceCreditCnyPerUsd,
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    publicSettingsLoaded: true,
    cachedPublicSettings: { purchase_subscription_url: '' },
    fetchPublicSettings: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: { id: 1 }, token: 'test-token' }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const base = messages[key] ?? key
        if (!params) return base
        return `${base}:${Object.values(params).join(',')}`
      },
      locale: { value: 'en' },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const PassthroughStub = { template: '<div></div>' }

const createItem = (overrides: Partial<ModelCatalogItem> = {}): ModelCatalogItem => ({
  model: 'gemini-3-pro-image',
  display_name: 'gemini-3-pro-image',
  platform: 'gemini',
  billing_mode: 'image',
  best_group: { id: 1, name: 'Image', rate_multiplier: 1, rate_source: 'group_default' },
  available_group_count: 1,
  official_pricing: {
    input_per_mtok_usd: null,
    output_per_mtok_usd: null,
    cache_write_per_mtok_usd: null,
    cache_read_per_mtok_usd: null,
    per_request_usd: null,
    per_image_usd: 0.201,
    per_second_usd: null,
    source: 'litellm',
    has_reference: true,
  },
  effective_pricing_usd: {
    input_per_mtok_usd: null,
    output_per_mtok_usd: null,
    cache_write_per_mtok_usd: null,
    cache_read_per_mtok_usd: null,
    per_request_usd: null,
    per_image_usd: 0.201,
    per_second_usd: null,
    source: 'effective',
    has_reference: false,
  },
  comparison: {
    savings_percent: null,
    is_cheaper_than_official: false,
    delta_input_per_mtok_usd: null,
    delta_output_per_mtok_usd: null,
    delta_per_request_usd: null,
    delta_per_image_usd: null,
    delta_per_second_usd: null,
  },
  pricing_details: {
    supports_prompt_caching: false,
    has_long_context_multiplier: false,
    long_context_input_threshold: 0,
    intervals: [],
    media_tiers: [],
    media_unit: '',
  },
  other_groups: [],
  ...overrides,
})

const imageTiers: ModelCatalogMediaTier[] = [
  { tier: '1K', official_usd: 0.134, effective_usd: 0.134, is_default_tier: false },
  { tier: '2K', official_usd: 0.201, effective_usd: 0.201, is_default_tier: true },
  { tier: '4K', official_usd: 0.268, effective_usd: 0.268, is_default_tier: false },
]

const mountView = async () => {
  const wrapper = mount(ModelCatalogView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        SearchInput: PassthroughStub,
        PlatformIcon: PassthroughStub,
        Icon: PassthroughStub,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('ModelCatalogView image pricing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // 未接入支付换算时展示余额价（USD），断言不依赖汇率。
    fetchBalanceCreditCnyPerUsd.mockResolvedValue({
      balanceCreditCnyPerUsd: null,
      usdExchangeRate: null,
      error: '',
    })
  })

  it('renders one price row per resolution tier and marks the default tier', async () => {
    getCatalog.mockResolvedValue({
      items: [createItem({
        pricing_details: {
          supports_prompt_caching: false,
          has_long_context_multiplier: false,
          long_context_input_threshold: 0,
          intervals: [],
          media_tiers: imageTiers,
          media_unit: 'image',
        },
      })],
      summary: {
        total_models: 1,
        token_models: 0,
        non_token_models: 1,
        best_savings_model: '',
        max_savings_percent: 0,
      },
    })

    const wrapper = await mountView()
    const text = wrapper.text()

    // 三个分辨率档位各自成行，各自有价格 —— 这是本次修复的核心。
    expect(text).toContain('1K')
    expect(text).toContain('2K')
    expect(text).toContain('4K')
    expect(text).toContain('$0.1340')
    expect(text).toContain('$0.2010')
    expect(text).toContain('$0.2680')

    // 默认档位要点名，否则用户以为主价格是唯一价格。
    expect(text).toContain('Billed at this tier when no size is given')

    // 不能再出现 token 区间格式化器产出的空区间标题。
    expect(text).not.toContain('≥ —')
  })

  it('falls back to a single row when the backend sends no media tiers', async () => {
    getCatalog.mockResolvedValue({
      items: [createItem()],
      summary: {
        total_models: 1,
        token_models: 0,
        non_token_models: 1,
        best_savings_model: '',
        max_savings_percent: 0,
      },
    })

    const wrapper = await mountView()
    const text = wrapper.text()

    expect(text).toContain('Per image')
    expect(text).toContain('$0.2010')
    expect(text).not.toContain('Billed at this tier when no size is given')
  })

  it('renders video models per resolution instead of empty token rows', async () => {
    getCatalog.mockResolvedValue({
      items: [createItem({
        model: 'grok-imagine-video',
        display_name: 'grok-imagine-video',
        billing_mode: 'video',
        official_pricing: {
          input_per_mtok_usd: null,
          output_per_mtok_usd: null,
          cache_write_per_mtok_usd: null,
          cache_read_per_mtok_usd: null,
          per_request_usd: null,
          per_image_usd: null,
          per_second_usd: 0.05,
          source: 'litellm',
          has_reference: true,
        },
        effective_pricing_usd: {
          input_per_mtok_usd: null,
          output_per_mtok_usd: null,
          cache_write_per_mtok_usd: null,
          cache_read_per_mtok_usd: null,
          per_request_usd: null,
          per_image_usd: null,
          per_second_usd: 0.05,
          source: 'effective',
          has_reference: false,
        },
        pricing_details: {
          supports_prompt_caching: false,
          has_long_context_multiplier: false,
          long_context_input_threshold: 0,
          intervals: [],
          media_tiers: [
            { tier: '480p', official_usd: 0.05, effective_usd: 0.05, is_default_tier: true },
            { tier: '720p', official_usd: 0.07, effective_usd: 0.07, is_default_tier: false },
            { tier: '1080p', official_usd: 0.25, effective_usd: 0.25, is_default_tier: false },
          ],
          media_unit: 'second',
        },
      })],
      summary: {
        total_models: 1,
        token_models: 0,
        non_token_models: 1,
        best_savings_model: '',
        max_savings_percent: 0,
      },
    })

    const wrapper = await mountView()
    const text = wrapper.text()

    expect(text).toContain('480p')
    expect(text).toContain('720p')
    expect(text).toContain('1080p')
    expect(text).toContain('Per second of video')
    expect(text).toContain('$0.2500')
    // 视频模型不应再落进 token 分支显示每百万 Tokens。
    expect(text).not.toContain('Per 1M tokens')
  })
})
