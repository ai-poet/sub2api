import { describe, expect, it } from 'vitest'

import type { ModelCatalogItem } from '@/api/modelCatalog'
import {
  filterCatalogGroups,
  formatGroupMultiplier,
  formatMaxSavings,
  groupCatalogItems,
  prettifyModelName,
} from '../catalogGroups'

const item = (model: string, groupId: number, overrides: Partial<ModelCatalogItem> = {}): ModelCatalogItem => ({
  model,
  display_name: model,
  platform: 'anthropic',
  billing_mode: 'token',
  best_group: { id: groupId, name: `group-${groupId}`, rate_multiplier: 0.5, rate_source: 'group_default', sort_order: groupId === 2 ? 0 : 1 },
  available_group_count: 1,
  official_pricing: {
    input_per_mtok_usd: 3, output_per_mtok_usd: 15, cache_write_per_mtok_usd: null, cache_read_per_mtok_usd: null,
    per_request_usd: null, per_image_usd: null, per_second_usd: null, source: 'litellm', has_reference: true,
  },
  effective_pricing_usd: {
    input_per_mtok_usd: 1.5, output_per_mtok_usd: 7.5, cache_write_per_mtok_usd: null, cache_read_per_mtok_usd: null,
    per_request_usd: null, per_image_usd: null, per_second_usd: null, source: 'effective', has_reference: false,
  },
  comparison: {
    savings_percent: 0.5, is_cheaper_than_official: true, delta_input_per_mtok_usd: null, delta_output_per_mtok_usd: null,
    delta_per_request_usd: null, delta_per_image_usd: null, delta_per_second_usd: null,
  },
  pricing_details: {
    supports_prompt_caching: false, has_long_context_multiplier: false, long_context_input_threshold: 0,
    intervals: [], media_tiers: [], media_unit: '',
  },
  other_groups: [],
  ...overrides,
})

describe('prettifyModelName', () => {
  it('turns model ids into readable titles', () => {
    expect(prettifyModelName('claude-opus-4-6')).toBe('Claude Opus 4.6')
    expect(prettifyModelName('claude-haiku-4-5-20251001')).toBe('Claude Haiku 4.5')
    expect(prettifyModelName('gpt-5.5')).toBe('GPT 5.5')
    expect(prettifyModelName('gemini-3-pro-image')).toBe('Gemini 3 Pro Image')
    expect(prettifyModelName('deepseek-v3.2')).toBe('DeepSeek V3.2')
    expect(prettifyModelName('glm-4.6')).toBe('GLM 4.6')
  })
})

describe('groupCatalogItems', () => {
  it('groups by group, orders sections by sort order and keeps the text multiplier', () => {
    const sections = groupCatalogItems([
      item('claude-sonnet-4-6', 1),
      item('gpt-image-2', 2, {
        billing_mode: 'image',
        platform: 'openai',
        best_group: { id: 2, name: 'group-2', rate_multiplier: 2, rate_source: 'group_default', sort_order: 0 },
      }),
      item('gpt-5.5', 2, {
        platform: 'openai',
        best_group: { id: 2, name: 'group-2', rate_multiplier: 0.3, rate_source: 'group_default', sort_order: 0 },
        comparison: { ...item('x', 2).comparison, savings_percent: 0.7 },
      }),
      item('claude-opus-4-6', 1),
    ])

    expect(sections.map(section => section.id)).toEqual([2, 1])
    // 图片模型的独立倍率不代表分组倍率
    expect(sections[0].rateMultiplier).toBe(0.3)
    expect(sections[0].maxSavings).toBe(0.7)
    expect(sections[1].models.map(model => model.model)).toEqual(['claude-opus-4-6', 'claude-sonnet-4-6'])
  })

  it('keeps a whole group when the search matches the group, otherwise only matching models', () => {
    const sections = groupCatalogItems([item('claude-sonnet-4-6', 1), item('claude-opus-4-6', 1)])
    const base = { platform: 'all', pricing: 'all' as const, groupId: 'all' as const }
    expect(filterCatalogGroups(sections, { ...base, search: 'group-1' })[0].models).toHaveLength(2)
    expect(filterCatalogGroups(sections, { ...base, search: 'opus' })[0].models).toHaveLength(1)
    expect(filterCatalogGroups(sections, { ...base, search: 'nothing' })).toHaveLength(0)
    expect(filterCatalogGroups(sections, { ...base, search: '', pricing: 'image' })).toHaveLength(0)
  })
})

describe('format helpers', () => {
  it('formats savings and multipliers', () => {
    expect(formatMaxSavings(0.955)).toBe(96)
    expect(formatMaxSavings(null)).toBeNull()
    expect(formatMaxSavings(0)).toBeNull()
    expect(formatGroupMultiplier(5)).toBe('x5')
    expect(formatGroupMultiplier(0.5)).toBe('x0.5')
    expect(formatGroupMultiplier(0.05)).toBe('x0.05')
  })
})
