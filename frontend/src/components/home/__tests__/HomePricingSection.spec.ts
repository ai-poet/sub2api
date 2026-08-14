import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import HomePricingSection from '../HomePricingSection.vue'
import type { PublicPricingItem, PublicPricingResponse } from '@/api/pricing'

const translations: Record<string, string> = {
  'home.pricingTable.overline': 'Pricing advantage',
  'home.pricingTable.title': 'Friendlier than official pricing',
  'home.pricingTable.description': 'Pay for what you use.',
  'home.pricingTable.badge': 'Goal',
  'home.pricingTable.badgeValue': 'Lower total cost',
  'home.pricingTable.badgeSavings': 'Max savings',
  'home.pricingTable.badgeSavingsValue': '{percent}% off',
  'home.pricingTable.note': 'Final billing follows the dashboard price.',
  'home.pricingTable.table.model': 'Model',
  'home.pricingTable.table.group': 'Group',
  'home.pricingTable.table.input': 'Input / 1M',
  'home.pricingTable.table.output': 'Output / 1M',
  'home.pricingTable.table.cacheWrite': 'Cache write / 1M',
  'home.pricingTable.table.cacheRead': 'Cache read / 1M',
  'home.pricingTable.table.vsOfficial': 'vs official',
  'home.pricingTable.table.savings': '{percent}% off',
  'home.pricingTable.table.perRequest': '{price} / request',
  'home.pricingTable.table.perImage': '{price} / image',
  'home.pricingTable.table.modelCount': '{count} models',
  'home.pricingTable.table.expand': 'Show all ({count} more)',
  'home.pricingTable.table.collapse': 'Show less',
  'home.pricingTable.cards.claude.tag': 'Claude family',
  'home.pricingTable.cards.claude.title': 'For heavy Claude Code work',
  'home.pricingTable.cards.claude.description': 'Full coverage of Claude models.',
  'home.pricingTable.cards.codex.tag': 'Codex / GPT family',
  'home.pricingTable.cards.codex.title': 'Codex and GPT, both here',
  'home.pricingTable.cards.codex.description': 'GPT available alongside Codex.',
  'home.pricingTable.cards.compatible.tag': 'OpenAI-compatible',
  'home.pricingTable.cards.compatible.title': 'Other compatible models',
  'home.pricingTable.cards.compatible.description': 'Same gateway entry.',
  'home.providers.openaiCompatible': 'OpenAI compatible',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const message = translations[key] || key
        return Object.entries(params || {}).reduce(
          (result, [name, value]) => result.replace(`{${name}}`, String(value)),
          message,
        )
      },
    }),
  }
})

const getPublicPricing = vi.fn()
vi.mock('@/api/pricing', () => ({
  getPublicPricing: (...args: unknown[]) => getPublicPricing(...args),
}))

function makeItem(overrides: Partial<PublicPricingItem> = {}): PublicPricingItem {
  return {
    model: 'claude-sonnet-4-6',
    display_name: 'claude-sonnet-4-6',
    platform: 'anthropic',
    billing_mode: 'token',
    group: { name: 'Public', rate_multiplier: 0.8, rate_source: 'group_default' },
    available_group_count: 1,
    official_pricing: {
      input_per_mtok_usd: 3,
      output_per_mtok_usd: 15,
      cache_write_per_mtok_usd: null,
      cache_read_per_mtok_usd: null,
      per_request_usd: null,
      per_image_usd: null,
      has_reference: true,
    },
    effective_pricing_usd: {
      input_per_mtok_usd: 2.4,
      output_per_mtok_usd: 12,
      cache_write_per_mtok_usd: null,
      cache_read_per_mtok_usd: null,
      per_request_usd: null,
      per_image_usd: null,
      has_reference: true,
    },
    comparison: { savings_percent: 20, is_cheaper_than_official: true },
    ...overrides,
  }
}

function makeResponse(items: PublicPricingItem[]): PublicPricingResponse {
  return { enabled: true, items, summary: { total_models: items.length, max_savings_percent: 20 } }
}

describe('HomePricingSection', () => {
  beforeEach(() => {
    getPublicPricing.mockReset()
  })

  it('renders real prices returned by the public endpoint', async () => {
    getPublicPricing.mockResolvedValue(makeResponse([makeItem()]))

    const wrapper = mount(HomePricingSection)
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('table').exists()).toBe(true)
    expect(text).toContain('claude-sonnet-4-6')
    // 有效单价 = 官方 $3 × 分组倍率 0.8
    expect(text).toContain('$2.40')
    expect(text).toContain('$12.00')
    expect(text).toContain('Public')
    expect(text).toContain('×0.8')
    expect(text).toContain('20% off')
  })

  // 落地页对访客必须永远可用：接口失败时静默回落到静态文案卡片，不能抛错、不能空白。
  it('falls back to the static copy cards when the request fails', async () => {
    getPublicPricing.mockRejectedValue(new Error('network down'))

    const wrapper = mount(HomePricingSection)
    await flushPromises()

    expect(wrapper.find('table').exists()).toBe(false)
    const text = wrapper.text()
    expect(text).toContain('For heavy Claude Code work')
    expect(text).toContain('Codex and GPT, both here')
    expect(text).toContain('Lower total cost')
  })

  // 开关关闭时后端返回 enabled:false + 空数组，同样回落到文案卡片。
  it('falls back to the static copy cards when the feature is disabled', async () => {
    getPublicPricing.mockResolvedValue({
      enabled: false,
      items: [],
      summary: { total_models: 0, max_savings_percent: 0 },
    })

    const wrapper = mount(HomePricingSection)
    await flushPromises()

    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.text()).toContain('For heavy Claude Code work')
  })

  // 同一模型出现在多个公开分组时只保留最优的那条（后端已排好序，取首个）。
  it('deduplicates a model down to its best group', async () => {
    getPublicPricing.mockResolvedValue(
      makeResponse([
        makeItem({ group: { name: 'Best', rate_multiplier: 0.5, rate_source: 'group_default' } }),
        makeItem({ group: { name: 'Worse', rate_multiplier: 1.5, rate_source: 'group_default' } }),
      ]),
    )

    const wrapper = mount(HomePricingSection)
    await flushPromises()

    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.text()).toContain('Best')
    expect(wrapper.text()).not.toContain('Worse')
  })

  // 没有官方参考价时后端不下发 savings_percent，对比列显示占位而不是编造的比例。
  it('shows no savings badge when the backend omits savings_percent', async () => {
    getPublicPricing.mockResolvedValue(
      makeResponse([
        makeItem({ comparison: { savings_percent: null, is_cheaper_than_official: false } }),
      ]),
    )

    const wrapper = mount(HomePricingSection)
    await flushPromises()

    expect(wrapper.text()).not.toContain('% off')
    expect(wrapper.text()).toContain('—')
  })

  it('renders per-request pricing for non-token billing modes', async () => {
    getPublicPricing.mockResolvedValue(
      makeResponse([
        makeItem({
          model: 'some-image-model',
          display_name: 'some-image-model',
          billing_mode: 'image',
          effective_pricing_usd: {
            input_per_mtok_usd: null,
            output_per_mtok_usd: null,
            cache_write_per_mtok_usd: null,
            cache_read_per_mtok_usd: null,
            per_request_usd: null,
            per_image_usd: 0.04,
            has_reference: false,
          },
        }),
      ]),
    )

    const wrapper = mount(HomePricingSection)
    await flushPromises()

    expect(wrapper.text()).toContain('$0.04 / image')
  })
})
