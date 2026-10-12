import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { ModelStat, UserSpendingRankingItem, UserUsageTrendPoint } from '@/types'
import AdminModelDistribution from '../AdminModelDistribution.vue'
import AdminTokenTrend from '../AdminTokenTrend.vue'
import AdminUserTrend from '../AdminUserTrend.vue'
import { costDigits, formatCost, formatDayLabel, formatHourLabel, formatTokens } from '../format'

const { getUserBreakdown } = vi.hoisted(() => ({ getUserBreakdown: vi.fn() }))

vi.mock('@/api/admin/dashboard', () => ({ getUserBreakdown }))

vi.mock('vue-chartjs', () => ({
  Line: { name: 'Line', props: ['data', 'options'], template: '<div class="line-chart" />' },
  Doughnut: { name: 'Doughnut', props: ['data', 'options'], template: '<div class="doughnut-chart" />' },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${Object.values(params).join(',')}` : key),
    }),
  }
})

const model = (name: string, tokens: number): ModelStat => ({
  model: name,
  requests: 10,
  input_tokens: 0,
  output_tokens: 0,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  total_tokens: tokens,
  cost: 2,
  actual_cost: 1,
  account_cost: 0.5,
}) as ModelStat

const rankingUser = (id: number, cost: number): UserSpendingRankingItem => ({
  user_id: id,
  email: `u${id}@example.com`,
  username: id === 1 ? 'alice' : '',
  actual_cost: cost,
  requests: 5,
  tokens: 100,
})

describe('admin dashboard format helpers', () => {
  it('formats axis labels, tokens and costs like TokenRouter', () => {
    expect(formatHourLabel('2026-10-12 08:00')).toBe('08:00')
    expect(formatHourLabel('2026-10-12T23:00:00Z')).toBe('23:00')
    expect(formatDayLabel('2026-10-02')).toBe('10-02')
    expect(formatDayLabel('weird')).toBe('weird')
    expect(formatTokens(999)).toBe('999')
    expect(formatTokens(1_234_567)).toBe('1.23M')
    expect(costDigits(0.004)).toBe(4)
    expect(costDigits(0.5)).toBe(3)
    expect(formatCost(12345.678)).toBe('$12,345.68')
    expect(formatCost(undefined)).toBe('$0.0000')
  })
})

describe('AdminModelDistribution', () => {
  beforeEach(() => {
    getUserBreakdown.mockReset()
    getUserBreakdown.mockResolvedValue({ users: [{ user_id: 3, email: 'x@example.com', requests: 1, total_tokens: 10, cost: 1, actual_cost: 1, account_cost: 1 }] })
  })

  const mountChart = (props: Record<string, unknown> = {}) =>
    mount(AdminModelDistribution, {
      props: { modelStats: [model('small', 10), model('big', 500)], startDate: '2026-10-11', endDate: '2026-10-12', ...props },
      global: { stubs: { Icon: true, UserBreakdownSubTable: { props: ['items', 'loading'], template: '<div data-stub="breakdown">{{ items.length }}</div>' } } },
    })

  it('sorts models by tokens, keeps the account cost column and shows the share in the body-level tooltip', () => {
    const wrapper = mountChart()
    const rows = wrapper.findAll('[data-testid="model-row"]')
    expect(rows.map((row) => row.findAll('td')[0].text())).toEqual(['big', 'small'])
    expect(rows[0].findAll('td').map((td) => td.text())).toEqual(['big', '10', '500', '$1.00', '$0.500', '$2.00'])

    const doughnut = wrapper.findComponent({ name: 'Doughnut' })
    const tooltip = doughnut.props('options').plugins.tooltip
    expect(tooltip.enabled).toBe(false)
    expect(typeof tooltip.external).toBe('function')
    expect(tooltip.callbacks.label({ label: 'big', dataIndex: 0 })).toBe('big: 500 (98.0%)')
  })

  it('shows the skeleton while loading and expands a model into its per-user breakdown', async () => {
    const loading = mountChart({ loading: true })
    expect(loading.find('[role="status"]').exists()).toBe(true)
    expect(loading.findComponent({ name: 'Doughnut' }).exists()).toBe(false)

    const wrapper = mountChart()
    const row = wrapper.findAll('[data-testid="model-row"]')[0]
    await row.trigger('click')
    await flushPromises()
    expect(getUserBreakdown).toHaveBeenCalledWith({ start_date: '2026-10-11', end_date: '2026-10-12', model: 'big', model_source: 'requested' })
    expect(row.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[data-stub="breakdown"]').text()).toBe('1')

    await row.trigger('click')
    expect(row.attributes('aria-expanded')).toBe('false')
  })

  it('switches to the spending ranking with an "other" row and emits clicks for real users only', async () => {
    const wrapper = mountChart({
      rankingItems: [rankingUser(1, 30), rankingUser(2, 10)],
      rankingTotalActualCost: 50,
      rankingTotalRequests: 20,
      rankingTotalTokens: 300,
    })
    const [, rankingButton] = wrapper.findAll('.segmented-item')
    await rankingButton.trigger('click')
    expect(rankingButton.classes()).toContain('segmented-item-active')
    expect(wrapper.text()).toContain('admin.dashboard.spendingRankingTitle')

    const rows = wrapper.findAll('[data-testid="ranking-row"]')
    expect(rows.map((row) => row.findAll('td')[0].text())).toEqual(['#1alice', '#2u2@example.com', 'Σadmin.dashboard.spendingRankingOther'])
    expect(rows[2].findAll('td')[3].text()).toBe('$10.00')

    await rows[2].trigger('click')
    expect(wrapper.emitted('ranking-click')).toBeUndefined()
    await rows[0].trigger('click')
    expect(wrapper.emitted('ranking-click')?.[0]?.[0]).toMatchObject({ user_id: 1 })

    const data = wrapper.findComponent({ name: 'Doughnut' }).props('data')
    expect(data.datasets[0].data).toEqual([30, 10, 10])
    expect(data.datasets[0].backgroundColor[2]).toBe('#94a3b8')
  })
})

describe('AdminTokenTrend', () => {
  const point = (date: string) => ({
    date,
    requests: 1,
    input_tokens: 60,
    output_tokens: 10,
    cache_creation_tokens: 20,
    cache_read_tokens: 20,
    total_tokens: 110,
    cost: 0.5,
    actual_cost: 0.25,
  })

  it('shortens hourly labels, keeps the full time in the tooltip title and computes the cached input share', () => {
    const wrapper = mount(AdminTokenTrend, { props: { trendData: [point('2026-10-12 08:00')], granularity: 'hour' } })
    const line = wrapper.findComponent({ name: 'Line' })
    expect(line.props('data').labels).toEqual(['08:00'])
    expect(line.props('data').datasets[4].data).toEqual([20])

    const callbacks = line.props('options').plugins.tooltip.callbacks
    expect(callbacks.title([{ dataIndex: 0 }])).toBe('2026-10-12 08:00')
    expect(callbacks.footer([{ dataIndex: 0 }])).toBe('Actual: $0.2500 | Standard: $0.5000')
    expect(line.props('options').scales.x.ticks.maxRotation).toBe(0)
  })

  it('shows the skeleton while loading and an empty state without data', () => {
    expect(mount(AdminTokenTrend, { props: { trendData: [], loading: true } }).find('[role="status"]').exists()).toBe(true)
    expect(mount(AdminTokenTrend, { props: { trendData: [] } }).text()).toContain('admin.dashboard.noDataAvailable')
  })
})

describe('AdminUserTrend', () => {
  const trend: UserUsageTrendPoint[] = [
    { date: '2026-10-12 01:00', user_id: 1, username: 'same', email: '', requests: 1, tokens: 1000, cost: 3, actual_cost: 2 },
    { date: '2026-10-12 00:00', user_id: 2, username: 'same', email: '', requests: 1, tokens: 500, cost: 1, actual_cost: 1 },
  ] as UserUsageTrendPoint[]

  it('keeps users with the same name apart and formats by the selected metric', async () => {
    const wrapper = mount(AdminUserTrend, { props: { trend, metric: 'tokens' } })
    const line = wrapper.findComponent({ name: 'Line' })
    expect(line.props('data').labels).toEqual(['00:00', '01:00'])
    expect(line.props('data').datasets.map((set: { data: number[] }) => set.data)).toEqual([[0, 1000], [500, 0]])
    expect(line.props('options').scales.y.ticks.callback(1500)).toBe('1.50K')
    expect(line.props('options').plugins.tooltip.callbacks.title([{ dataIndex: 1 }])).toBe('2026-10-12 01:00')

    await wrapper.setProps({ metric: 'actual_cost' })
    expect(wrapper.findComponent({ name: 'Line' }).props('options').scales.y.ticks.callback(5)).toBe('$5.00')
  })

  it('emits the metric switch from the segmented control', async () => {
    const wrapper = mount(AdminUserTrend, { props: { trend: [], metric: 'tokens' } })
    const buttons = wrapper.findAll('.segmented-item')
    expect(buttons[0].attributes('aria-pressed')).toBe('true')
    await buttons[1].trigger('click')
    expect(wrapper.emitted('update:metric')?.[0]).toEqual(['actual_cost'])
    expect(wrapper.text()).toContain('admin.dashboard.noDataAvailable')
  })
})
