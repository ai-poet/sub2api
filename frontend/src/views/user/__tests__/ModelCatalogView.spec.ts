import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { ModelCatalogItem, ModelCatalogMediaTier } from '@/api/modelCatalog'
import ModelCatalogView from '../ModelCatalogView.vue'

const {
  getCatalog,
  fetchBalanceCreditCnyPerUsd,
  showError,
  getHistories,
  getHistory,
  getPublicPricing,
  fetchPublicCurrencyRates,
  copyToClipboard,
  authState,
  publicSettings,
} = vi.hoisted(() => ({
  getCatalog: vi.fn(),
  fetchBalanceCreditCnyPerUsd: vi.fn(),
  showError: vi.fn(),
  getHistories: vi.fn(),
  getHistory: vi.fn(),
  getPublicPricing: vi.fn(),
  fetchPublicCurrencyRates: vi.fn(),
  copyToClipboard: vi.fn(),
  authState: { isAuthenticated: true },
  publicSettings: { purchase_subscription_url: '', group_status_enabled: false } as Record<string, unknown>,
}))

vi.mock('@/api/groupStatus', () => ({ getHistories, getHistory }))

vi.mock('@/api/pricing', () => ({ getPublicPricing, fetchPublicCurrencyRates }))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
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
  'modelCatalog.card.perImage': '/ image',
  'modelCatalog.card.perSecond': '/ sec',
  'modelCatalog.card.perMTok': '/ 1M tokens',
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
    cachedPublicSettings: publicSettings,
    fetchPublicSettings: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { id: 1 },
    token: 'test-token',
    get isAuthenticated() {
      return authState.isAuthenticated
    },
  }),
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
const PublicShellStub = { template: '<div data-test="public-shell"><slot /></div>' }
const PassthroughStub = { template: '<div></div>' }
const RouterLinkStub = { props: ['to'], template: '<a><slot /></a>' }

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
        PublicCatalogShell: PublicShellStub,
        SearchInput: PassthroughStub,
        PlatformIcon: PassthroughStub,
        ModelIcon: PassthroughStub,
        Icon: PassthroughStub,
        RouterLink: RouterLinkStub,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('ModelCatalogView image pricing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authState.isAuthenticated = true
    publicSettings.group_status_enabled = false
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

    expect(text).toContain('$0.2010 / image')
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
    expect(text).toContain('/ sec')
    expect(text).toContain('$0.2500')
    // 视频模型不应再落进 token 分支显示每百万 Tokens。
    expect(text).not.toContain('/ 1M tokens')
  })
})

const tokenItem = (overrides: Partial<ModelCatalogItem> = {}): ModelCatalogItem => createItem({
  model: 'claude-sonnet-4-6',
  display_name: 'claude-sonnet-4-6',
  platform: 'anthropic',
  billing_mode: 'token',
  best_group: { id: 7, name: 'Claude', rate_multiplier: 0.5, rate_source: 'group_default' },
  pricing_details: {
    supports_prompt_caching: true,
    has_long_context_multiplier: false,
    long_context_input_threshold: 0,
    intervals: [
      {
        min_tokens: 0,
        max_tokens: 200000,
        tier_label: '',
        input_per_mtok_usd: 1.5,
        output_per_mtok_usd: 7.5,
        cache_write_per_mtok_usd: null,
        cache_read_per_mtok_usd: null,
        per_request_usd: null,
        per_image_usd: null,
      },
    ],
    media_tiers: [],
    media_unit: '',
  },
  context_window: 200000,
  ...overrides,
})

const catalogResponse = (items: ModelCatalogItem[]) => ({
  items,
  summary: { total_models: items.length, token_models: items.length, non_token_models: 0, best_savings_model: '', max_savings_percent: 0 },
})

describe('ModelCatalogView polish (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authState.isAuthenticated = true
    publicSettings.group_status_enabled = false
    fetchBalanceCreditCnyPerUsd.mockResolvedValue({ balanceCreditCnyPerUsd: null, usdExchangeRate: null, error: '' })
    copyToClipboard.mockResolvedValue(true)
  })

  it('requests availability once for all groups and draws the bar for the best group', async () => {
    publicSettings.group_status_enabled = true
    getCatalog.mockResolvedValue(catalogResponse([
      tokenItem(),
      tokenItem({ model: 'claude-opus-4-6', display_name: 'claude-opus-4-6' }),
    ]))
    getHistories.mockResolvedValue([
      {
        group_id: 7,
        availability: 99.5,
        total_count: 3,
        buckets: [
          { bucket_start: '2026-10-12T00:00:00Z', bucket_end: '2026-10-12T01:00:00Z', availability: 100, avg_latency_ms: null, total_count: 2, down_count: 0, latest_status: 'up' },
          { bucket_start: '2026-10-12T01:00:00Z', bucket_end: '2026-10-12T02:00:00Z', availability: 0, avg_latency_ms: null, total_count: 1, down_count: 1, latest_status: 'down' },
        ],
      },
    ])

    const wrapper = await mountView()

    expect(getHistories).toHaveBeenCalledTimes(1)
    expect(getHistory).not.toHaveBeenCalled()
    // 两个模型同在分组 7：一个分区、一条可用率条，挂在分组头部
    const sections = wrapper.findAll('[data-testid="catalog-group-section"]')
    expect(sections).toHaveLength(1)
    expect(sections[0].findAll('[data-testid="catalog-model-card"]')).toHaveLength(2)
    const bars = wrapper.findAll('[data-testid="group-availability-bar"]')
    expect(bars).toHaveLength(1)
    expect(bars[0].text()).toContain('99.50%')
  })

  it('skips the availability request when group status is off', async () => {
    getCatalog.mockResolvedValue(catalogResponse([tokenItem()]))
    const wrapper = await mountView()
    expect(getHistories).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="group-availability-bar"]').exists()).toBe(false)
  })

  it('shows modality icons, context window and copies the model id', async () => {
    getCatalog.mockResolvedValue(catalogResponse([tokenItem()]))
    const wrapper = await mountView()

    expect(wrapper.find('[data-modality="input-image"]').exists()).toBe(true)
    expect(wrapper.find('[data-modality="output-text"]').exists()).toBe(true)
    // 标题排版成 Claude Sonnet 4.6，原始 ID 在下一行；上下文长度在标题旁的信息浮层里（Teleport 到 body）
    expect(wrapper.text()).toContain('Claude Sonnet 4.6')
    expect(wrapper.find('[data-testid="model-info-trigger"]').exists()).toBe(true)
    expect(document.body.innerHTML).toContain('200K')

    await wrapper.get('[data-testid="model-id-copy"]').trigger('click')
    await flushPromises()
    expect(copyToClipboard).toHaveBeenCalledWith('claude-sonnet-4-6', 'modelCatalog.copiedModelId')
  })

  it('expands detailed pricing inside a Collapse', async () => {
    getCatalog.mockResolvedValue(catalogResponse([tokenItem()]))
    const wrapper = await mountView()

    const toggle = wrapper.get('[data-testid="model-pricing-toggle"]')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.text()).toContain('modelCatalog.card.viewPricing')

    await toggle.trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="model-pricing-toggle"]').attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[data-testid="model-pricing-toggle"]').text()).toContain('modelCatalog.card.collapsePricing')
    expect(wrapper.find('.motion-collapse').exists()).toBe(true)
    // 一档上下文分档：预览行显示区间，展开后是该档的输入 / 输出价
    expect(wrapper.text()).toContain('0-200K')
    expect(wrapper.get('[data-testid="model-pricing-panel"]').text()).toContain('$1.5000 / 1M tokens')
  })

  it('splits the catalog into group sections ordered by sort order, with savings and multiplier pills', async () => {
    const official = {
      input_per_mtok_usd: 3,
      output_per_mtok_usd: 15,
      cache_write_per_mtok_usd: null,
      cache_read_per_mtok_usd: null,
      per_request_usd: null,
      per_image_usd: null,
      per_second_usd: null,
      source: 'litellm',
      has_reference: true,
    }
    getCatalog.mockResolvedValue(catalogResponse([
      tokenItem({
        best_group: { id: 9, name: 'Codex', rate_multiplier: 0.3, rate_source: 'group_default', sort_order: 5 },
        model: 'gpt-5.5',
        display_name: 'gpt-5.5',
        platform: 'openai',
      }),
      tokenItem({
        best_group: { id: 7, name: 'Claude Max', rate_multiplier: 0.05, rate_source: 'group_default', sort_order: 1, description: 'Cheap Claude' },
        official_pricing: official,
        comparison: { ...createItem().comparison, savings_percent: 0.95, is_cheaper_than_official: true },
      }),
    ]))
    const wrapper = await mountView()

    const sections = wrapper.findAll('[data-testid="catalog-group-section"]')
    expect(sections.map(section => section.attributes('data-group'))).toEqual(['Claude Max', 'Codex'])
    expect(sections[0].text()).toContain('Cheap Claude')
    expect(sections[0].get('[data-testid="group-max-savings"]').text()).toContain('modelCatalog.group.maxSavings:95')
    expect(sections[0].get('[data-testid="group-rate"]').text()).toContain('modelCatalog.group.rate:x0.05')
    expect(sections[1].find('[data-testid="group-max-savings"]').exists()).toBe(false)
  })

  it('filters by search across groups and models', async () => {
    getCatalog.mockResolvedValue(catalogResponse([
      tokenItem(),
      tokenItem({
        best_group: { id: 9, name: 'Codex', rate_multiplier: 0.3, rate_source: 'group_default' },
        model: 'gpt-5.5',
        display_name: 'gpt-5.5',
        platform: 'openai',
      }),
    ]))
    const wrapper = await mount(ModelCatalogView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          SearchInput: { props: ['modelValue'], emits: ['update:modelValue'], template: '<input data-test="search" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />' },
          PlatformIcon: PassthroughStub,
          ModelIcon: PassthroughStub,
          Icon: PassthroughStub,
          RouterLink: RouterLinkStub,
        },
      },
    })
    await flushPromises()
    expect(wrapper.findAll('[data-testid="catalog-group-section"]')).toHaveLength(2)

    await wrapper.get('[data-test="search"]').setValue('gpt')
    expect(wrapper.findAll('[data-testid="catalog-group-section"]')).toHaveLength(1)
    expect(wrapper.find('[data-model="gpt-5.5"]').exists()).toBe(true)

    await wrapper.get('[data-test="search"]').setValue('no-such-model')
    expect(wrapper.find('[data-testid="model-catalog-empty"]').exists()).toBe(true)
    await wrapper.get('[data-testid="model-catalog-reset-filters"]').trigger('click')
    expect(wrapper.findAll('[data-testid="catalog-group-section"]')).toHaveLength(2)
  })

  it('renders the anonymous catalog from the public pricing endpoint', async () => {
    authState.isAuthenticated = false
    getPublicPricing.mockResolvedValue({
      enabled: true,
      items: [{
        model: 'gpt-5.5',
        display_name: 'gpt-5.5',
        platform: 'openai',
        billing_mode: 'token',
        group: { name: 'Codex', rate_multiplier: 0.3, rate_source: 'group_default' },
        available_group_count: 1,
        official_pricing: { input_per_mtok_usd: 5, output_per_mtok_usd: 20, cache_write_per_mtok_usd: null, cache_read_per_mtok_usd: 0.5, per_request_usd: null, per_image_usd: null, has_reference: true },
        effective_pricing_usd: { input_per_mtok_usd: 1.5, output_per_mtok_usd: 6, cache_write_per_mtok_usd: null, cache_read_per_mtok_usd: 0.15, per_request_usd: null, per_image_usd: null, has_reference: true },
        comparison: { savings_percent: 0.7, is_cheaper_than_official: true },
      }],
      summary: { total_models: 1, max_savings_percent: 0.7 },
    })
    fetchPublicCurrencyRates.mockResolvedValue({ balanceCreditCnyPerUsd: null, usdExchangeRate: null })

    const wrapper = await mountView()

    expect(getCatalog).not.toHaveBeenCalled()
    expect(getHistories).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="public-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="model-catalog-login-cta"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('gpt-5.5')
    expect(wrapper.text()).toContain('$1.5000')
  })

  it('tells anonymous visitors to sign in when public pricing is off', async () => {
    authState.isAuthenticated = false
    getPublicPricing.mockResolvedValue({ enabled: false, items: [], summary: { total_models: 0, max_savings_percent: 0 } })

    const wrapper = await mountView()

    expect(wrapper.text()).toContain('modelCatalog.public.disabled')
    expect(fetchPublicCurrencyRates).not.toHaveBeenCalled()
  })
})
