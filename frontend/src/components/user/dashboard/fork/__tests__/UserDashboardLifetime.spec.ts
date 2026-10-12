import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import type { UserDashboardStats } from '@/api/usage'
import UserDashboardLifetime from '../UserDashboardLifetime.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${Object.values(params).join(',')}` : key),
    }),
  }
})

const stats = (overrides: Partial<UserDashboardStats> = {}) =>
  ({
    total_api_keys: 2,
    active_api_keys: 1,
    total_requests: 12345,
    total_input_tokens: 1_000_000,
    total_output_tokens: 200_000,
    total_cache_creation_tokens: 50_000,
    total_cache_read_tokens: 750_000,
    total_tokens: 2_000_000,
    total_cost: 40,
    total_actual_cost: 10,
    today_requests: 321,
    today_input_tokens: 0,
    today_output_tokens: 0,
    today_cache_creation_tokens: 0,
    today_cache_read_tokens: 0,
    today_tokens: 45_600,
    today_cost: 4,
    today_actual_cost: 1,
    average_duration_ms: 0,
    rpm: 0,
    tpm: 0,
    ...overrides,
  }) as UserDashboardStats

describe('UserDashboardLifetime', () => {
  it('shows all-time requests, tokens with their split, and actual vs standard spend', () => {
    const wrapper = mount(UserDashboardLifetime, { props: { stats: stats() }, global: { stubs: { Icon: true } } })

    // 测试环境默认「减少动态效果」，滚动数字直接落在终值
    expect(wrapper.get('[data-testid="lifetime-requests"]').text()).toContain('12,345')
    expect(wrapper.get('[data-testid="lifetime-requests"]').text()).toContain('dashboard.fork.lifetime.today:321')

    const tokens = wrapper.get('[data-testid="lifetime-tokens"]').text()
    expect(tokens).toContain('2.0M')
    expect(tokens).toContain('dashboard.input 1.0M')
    expect(tokens).toContain('dashboard.output 200.0K')
    expect(tokens).toContain('dashboard.cache 800.0K')

    const cost = wrapper.get('[data-testid="lifetime-cost"]')
    expect(cost.text()).toContain('$10.00')
    expect(cost.text()).toContain('dashboard.standard $40.00')
    expect(cost.text()).toContain('dashboard.fork.lifetime.todayCost:$1.00,$4.00')
    expect(cost.get('[data-testid="lifetime-savings"]').text()).toBe('dashboard.fork.lifetime.saved:$30.00,75')
    expect(cost.get('.lifetime-bar').attributes('style')).toContain('--bar-w: 25%')
  })

  it('hides the savings pill when nothing was saved', () => {
    const wrapper = mount(UserDashboardLifetime, {
      props: { stats: stats({ total_cost: 10, total_actual_cost: 10 }) },
      global: { stubs: { Icon: true } },
    })
    expect(wrapper.find('[data-testid="lifetime-savings"]').exists()).toBe(false)
  })

  it('shows placeholders until the stats arrive', () => {
    const wrapper = mount(UserDashboardLifetime, { props: { stats: null }, global: { stubs: { Icon: true } } })
    expect(wrapper.get('[data-testid="lifetime-usage"]').attributes('aria-busy')).toBe('true')
    expect(wrapper.text()).not.toContain('$')
  })
})
