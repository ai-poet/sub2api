import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import type { DashboardStats } from '@/types'
import AdminDashboardPage from '../AdminDashboardPage.vue'

const { getSnapshotV2, getUserUsageTrend, getUserSpendingRanking, showError, push } = vi.hoisted(() => ({
  getSnapshotV2: vi.fn(),
  getUserUsageTrend: vi.fn(),
  getUserSpendingRanking: vi.fn(),
  showError: vi.fn(),
  push: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { dashboard: { getSnapshotV2, getUserUsageTrend, getUserSpendingRanking } },
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const stats = (overrides: Partial<DashboardStats> = {}): DashboardStats => ({
  total_users: 1200,
  today_new_users: 7,
  active_users: 35,
  hourly_active_users: 4,
  stats_updated_at: '',
  stats_stale: false,
  total_api_keys: 88,
  active_api_keys: 60,
  total_accounts: 12,
  normal_accounts: 10,
  error_accounts: 2,
  ratelimit_accounts: 0,
  overload_accounts: 0,
  total_requests: 123456,
  total_input_tokens: 0,
  total_output_tokens: 0,
  total_cache_creation_tokens: 0,
  total_cache_read_tokens: 0,
  total_tokens: 2_500_000_000,
  total_cost: 5000,
  total_actual_cost: 1234.5,
  total_account_cost: 800,
  today_requests: 4321,
  today_input_tokens: 0,
  today_output_tokens: 0,
  today_cache_creation_tokens: 0,
  today_cache_read_tokens: 0,
  today_tokens: 3_450_000,
  today_cost: 12,
  today_actual_cost: 3.5,
  today_account_cost: 0.05,
  average_duration_ms: 1530,
  uptime: 0,
  rpm: 42,
  tpm: 12_500,
  ...overrides,
}) as DashboardStats

const passthrough = (name: string, props: string[] = []) => defineComponent({
  name,
  props,
  emits: ['update:metric', 'ranking-click'],
  setup: () => () => h('div', { 'data-stub': name }),
})

const ModelDistributionStub = passthrough('AdminModelDistribution', ['modelStats', 'loading', 'startDate', 'endDate', 'rankingItems'])
const TokenTrendStub = passthrough('AdminTokenTrend', ['trendData', 'loading', 'granularity'])
const UserTrendStub = defineComponent({
  name: 'AdminUserTrend',
  props: ['trend', 'metric', 'loading', 'granularity'],
  emits: ['update:metric'],
  setup: (_, { attrs }) => () => h('div', { ...attrs, 'data-stub': 'AdminUserTrend' }),
})
const DateRangePickerStub = defineComponent({
  name: 'DateRangePicker',
  emits: ['change', 'update:startDate', 'update:endDate'],
  setup: () => () => h('div', { 'data-stub': 'DateRangePicker' }),
})

const mountPage = () =>
  mount(AdminDashboardPage, {
    global: {
      stubs: {
        Icon: true,
        Select: true,
        AdminQuickActions: true,
        DateRangePicker: DateRangePickerStub,
        AdminModelDistribution: ModelDistributionStub,
        AdminTokenTrend: TokenTrendStub,
        AdminUserTrend: UserTrendStub,
      },
    },
  })

describe('AdminDashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getSnapshotV2.mockResolvedValue({ stats: stats(), trend: [], models: [] })
    getUserUsageTrend.mockResolvedValue({ trend: [] })
    getUserSpendingRanking.mockResolvedValue({ ranking: [], total_actual_cost: 0, total_requests: 0, total_tokens: 0 })
  })

  it('shows the skeleton until the first snapshot arrives', async () => {
    let resolve!: (value: unknown) => void
    getSnapshotV2.mockImplementationOnce(() => new Promise((r) => { resolve = r }))
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.find('[data-testid="dashboard-skeleton"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="stat-api-keys"]').exists()).toBe(false)

    resolve({ stats: stats(), trend: [], models: [] })
    await flushPromises()
    expect(wrapper.find('[data-testid="dashboard-skeleton"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="stat-api-keys"]').exists()).toBe(true)
  })

  it('renders the TokenRouter stat rows with actual / account cost / standard spend', async () => {
    const wrapper = mountPage()
    await flushPromises()

    // 测试环境默认「减少动态效果」，滚动数字直接显示终值
    expect(wrapper.get('[data-testid="stat-api-keys"]').text()).toContain('88')
    expect(wrapper.get('[data-testid="stat-accounts"]').text()).toContain('2 common.error')
    expect(wrapper.get('[data-testid="stat-today-requests"]').text()).toContain('common.total: 123,456')
    expect(wrapper.get('[data-testid="stat-users"]').text()).toContain('+7')
    expect(wrapper.get('[data-testid="stat-today-tokens"]').text()).toContain('3.45M')
    expect(wrapper.get('[data-testid="stat-total-tokens"]').text()).toContain('2.50B')
    expect(wrapper.get('[data-testid="stat-performance"]').text()).toContain('12.50K')
    expect(wrapper.get('[data-testid="stat-avg-response"]').text()).toContain('1.53s')

    const today = wrapper.get('[data-testid="today-costs"]').findAll('span').map((span) => span.text())
    expect(today).toEqual(['$3.50', '/', '$0.050', '/', '$12.00'])
    const total = wrapper.get('[data-testid="total-costs"]').findAll('span').map((span) => span.text())
    expect(total).toEqual(['$1,234.50', '/', '$800.00', '/', '$5,000.00'])
  })

  it('loads the last 24 hours by hour, and switches to days for longer ranges without reloading stats', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({ granularity: 'hour', include_stats: true }))
    expect(wrapper.findComponent(TokenTrendStub).props('granularity')).toBe('hour')

    wrapper.findComponent(DateRangePickerStub).vm.$emit('change', { startDate: '2026-10-01', endDate: '2026-10-12', preset: null })
    await flushPromises()
    expect(getSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({ granularity: 'day', include_stats: false }))
    expect(getUserUsageTrend).toHaveBeenLastCalledWith(expect.objectContaining({ granularity: 'day' }))
    expect(getUserSpendingRanking).toHaveBeenCalledTimes(2)
    expect(wrapper.findComponent(TokenTrendStub).props('granularity')).toBe('day')
  })

  it('the heading refresh button reloads stats and every chart, and is disabled meanwhile', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.clearAllMocks()
    getSnapshotV2.mockResolvedValue({ stats: stats({ total_api_keys: 90 }), trend: [], models: [] })

    const button = wrapper.get('[data-testid="admin-dashboard-refresh"]')
    await button.trigger('click')
    expect(button.attributes('disabled')).toBeDefined()
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({ include_stats: true }))
    expect(getUserUsageTrend).toHaveBeenCalledTimes(1)
    expect(getUserSpendingRanking).toHaveBeenCalledTimes(1)
    expect(button.attributes('disabled')).toBeUndefined()
    expect(wrapper.get('[data-testid="stat-api-keys"]').text()).toContain('90')
  })

  it('switches the user trend metric and opens a ranking user in the usage page', async () => {
    const wrapper = mountPage()
    await flushPromises()

    wrapper.findComponent(UserTrendStub).vm.$emit('update:metric', 'actual_cost')
    await flushPromises()
    expect(getUserUsageTrend).toHaveBeenLastCalledWith(expect.objectContaining({ metric: 'actual_cost', limit: 12 }))
    expect(wrapper.findComponent(UserTrendStub).props('metric')).toBe('actual_cost')

    wrapper.findComponent(ModelDistributionStub).vm.$emit('ranking-click', { user_id: 9, email: '', username: 'u', actual_cost: 1, requests: 1, tokens: 1 })
    expect(push).toHaveBeenCalledWith(expect.objectContaining({ path: '/admin/usage', query: expect.objectContaining({ user_id: '9' }) }))
  })

  it('reports a failed snapshot and keeps the entrance class until the last block finishes', async () => {
    getSnapshotV2.mockRejectedValueOnce(new Error('boom'))
    const failed = mountPage()
    await flushPromises()
    expect(showError).toHaveBeenCalledWith('admin.dashboard.failedToLoad')
    failed.unmount()

    const wrapper = mountPage()
    await flushPromises()
    const root = wrapper.get('[data-testid="admin-dashboard-root"]')
    expect(root.classes()).toContain('dash-rise')
    await wrapper.get('[data-testid="stat-api-keys"]').trigger('animationend')
    expect(root.classes()).toContain('dash-rise')
    await wrapper.get('[data-rise-last="true"]').trigger('animationend')
    expect(root.classes()).not.toContain('dash-rise')
  })
})
