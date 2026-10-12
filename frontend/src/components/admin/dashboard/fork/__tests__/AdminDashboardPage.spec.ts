import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import type { DashboardStats } from '@/types'
import AdminDashboardPage from '../AdminDashboardPage.vue'
import { useDashboardDataSource } from '@/components/user/dashboard/fork/dashboardDataSource'
import { injectUsageChartState } from '@/components/user/dashboard/fork/usageChartState'
import { formatDayKey } from '@/components/user/dashboard/fork/usageChartData'

const {
  getUsageTrend, getModelStats, getStats, getUserSpendingRanking, getUserUsageTrend, getAllGroups,
  push, heatmapReload, liveReload,
} = vi.hoisted(() => ({
  getUsageTrend: vi.fn(),
  getModelStats: vi.fn(),
  getStats: vi.fn(),
  getUserSpendingRanking: vi.fn(),
  getUserUsageTrend: vi.fn(),
  getAllGroups: vi.fn(),
  push: vi.fn(),
  heatmapReload: vi.fn(),
  liveReload: vi.fn(),
}))

vi.mock('@/api/admin/dashboard', () => {
  const api = { getUsageTrend, getModelStats, getStats, getUserSpendingRanking, getUserUsageTrend }
  return { ...api, default: api }
})
vi.mock('@/api/admin/groups', () => ({ getAll: getAllGroups, default: { getAll: getAllGroups } }))
vi.mock('@/api/usage', () => ({ usageAPI: { getDashboardTrend: vi.fn(), getDashboardModels: vi.fn(), getDashboardStats: vi.fn() } }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh' } }) }
})

const daysAgo = (days: number): string => {
  const date = new Date()
  date.setDate(date.getDate() - days)
  return formatDayKey(date)
}

// 子组件替身：记录页面提供的数据源口径，带 reload 的暴露给刷新按钮
const seenScopes: string[] = []
const LiveStatsStub = defineComponent({
  name: 'UserDashboardLiveStats',
  emits: ['update'],
  setup(_, { expose }) {
    seenScopes.push(useDashboardDataSource().scope)
    expose({ reload: liveReload })
    return () => h('div', { 'data-stub': 'UserDashboardLiveStats' })
  },
})
const HeatmapStub = defineComponent({
  name: 'UserDashboardHeatmap',
  emits: ['select-day'],
  setup(_, { expose }) {
    expose({ reload: heatmapReload })
    return () => h('div', { 'data-stub': 'UserDashboardHeatmap' })
  },
})
// 工具栏替身：点击切到近 30 天，事件 refresh 触发整页刷新
const ToolbarStub = defineComponent({
  name: 'UserDashboardUsageToolbar',
  props: ['refreshing'],
  emits: ['refresh'],
  setup(_, { emit }) {
    const state = injectUsageChartState()
    return () => h('div', { 'data-stub': 'UserDashboardUsageToolbar' }, [
      h('button', { 'data-testid': 'pick-30d', onClick: () => state.selectPreset('30d') }),
      h('button', { 'data-testid': 'refresh', onClick: () => emit('refresh') }),
    ])
  },
})
const passthrough = (name: string, props: string[] = [], emits: string[] = []) => defineComponent({
  name,
  props,
  emits,
  setup: () => () => h('div', { 'data-stub': name }),
})
const OverviewStub = passthrough('AdminOverviewCards', ['stats'])
const DistributionStub = passthrough('AdminModelDistribution', ['modelStats', 'loading', 'rankingItems', 'startDate', 'endDate'], ['ranking-click'])
const UserTrendStub = defineComponent({
  name: 'AdminUserTrend',
  props: ['trend', 'metric', 'loading'],
  emits: ['update:metric'],
  setup: (_, { attrs }) => () => h('div', { ...attrs, 'data-stub': 'AdminUserTrend' }),
})

const mountPage = async () => {
  const wrapper = mount(AdminDashboardPage, {
    global: {
      stubs: {
        UserDashboardLiveStats: LiveStatsStub,
        UserDashboardHeatmap: HeatmapStub,
        UserDashboardUsageToolbar: ToolbarStub,
        UserDashboardUsageChart: { template: '<div data-stub="UserDashboardUsageChart" />' },
        AdminOverviewCards: OverviewStub,
        AdminModelDistribution: DistributionStub,
        AdminQuickActions: { template: '<div data-stub="AdminQuickActions" />' },
        AdminUserTrend: UserTrendStub,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('AdminDashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    seenScopes.length = 0
    getUsageTrend.mockResolvedValue({ trend: [] })
    getModelStats.mockResolvedValue({ models: [{ model: 'gpt-5', requests: 1, total_tokens: 10 }] })
    getUserSpendingRanking.mockResolvedValue({ ranking: [], total_actual_cost: 0, total_requests: 0, total_tokens: 0 })
    getUserUsageTrend.mockResolvedValue({ trend: [] })
    getAllGroups.mockResolvedValue([{ id: 3, name: 'vip' }])
  })

  it('用户仪表盘的同一套区块按全站口径取数，下面接管理员特有的区块', async () => {
    const wrapper = await mountPage()
    expect(wrapper.findAll('[data-stub]').map((node) => node.attributes('data-stub'))).toEqual([
      'UserDashboardLiveStats',
      'UserDashboardUsageToolbar',
      'UserDashboardUsageChart',
      'AdminOverviewCards',
      'UserDashboardHeatmap',
      'AdminModelDistribution',
      'AdminQuickActions',
      'AdminUserTrend',
    ])
    expect(wrapper.get('h1').text()).toBe('admin.dashboard.title')
    expect(seenScopes).toEqual(['admin'])

    // 用量区：本期、上一周期两次趋势 + 模型统计，都走管理员接口；默认近 7 天
    expect(getUsageTrend).toHaveBeenCalledTimes(2)
    expect(getUsageTrend).toHaveBeenCalledWith(expect.objectContaining({ start_date: daysAgo(6), end_date: daysAgo(0), granularity: 'day' }))
    expect(getModelStats).toHaveBeenCalledTimes(1)
    expect(getAllGroups).toHaveBeenCalledTimes(1)
    expect(wrapper.findComponent(DistributionStub).props('modelStats')).toEqual([{ model: 'gpt-5', requests: 1, total_tokens: 10 }])
    expect(wrapper.findComponent(DistributionStub).props('startDate')).toBe(daysAgo(6))

    // 消费榜和用户趋势跟着工具栏的时间范围
    expect(getUserSpendingRanking).toHaveBeenCalledWith({ start_date: daysAgo(6), end_date: daysAgo(0), limit: 12 })
    expect(getUserUsageTrend).toHaveBeenCalledWith(expect.objectContaining({
      start_date: daysAgo(6), end_date: daysAgo(0), granularity: 'day', limit: 12, metric: 'tokens',
    }))
  })

  it('实时状态条的统计交给概览卡，失败的轮询不清掉上一次结果', async () => {
    const wrapper = await mountPage()
    const stats = { total_users: 9, rpm: 1, tpm: 2, average_duration_ms: 3, today_actual_cost: 4 } as unknown as DashboardStats
    wrapper.findComponent(LiveStatsStub).vm.$emit('update', stats)
    await flushPromises()
    expect(wrapper.findComponent(OverviewStub).props('stats')).toEqual(stats)

    wrapper.findComponent(LiveStatsStub).vm.$emit('update', null)
    await flushPromises()
    expect(wrapper.findComponent(OverviewStub).props('stats')).toEqual(stats)
  })

  it('工具栏换时间范围后，消费榜和用户趋势按新范围重新取数', async () => {
    const wrapper = await mountPage()
    vi.clearAllMocks()
    getUsageTrend.mockResolvedValue({ trend: [] })
    getModelStats.mockResolvedValue({ models: [] })
    getUserSpendingRanking.mockResolvedValue({ ranking: [] })
    getUserUsageTrend.mockResolvedValue({ trend: [] })

    await wrapper.get('[data-testid="pick-30d"]').trigger('click')
    await flushPromises()
    expect(getUserSpendingRanking).toHaveBeenCalledTimes(1)
    expect(getUserSpendingRanking).toHaveBeenCalledWith({ start_date: daysAgo(29), end_date: daysAgo(0), limit: 12 })
    expect(getUserUsageTrend).toHaveBeenCalledTimes(1)
  })

  it('刷新按钮刷新整页：用量区、热力图、实时状态、消费榜和用户趋势', async () => {
    const wrapper = await mountPage()
    vi.clearAllMocks()
    getUsageTrend.mockResolvedValue({ trend: [] })
    getModelStats.mockResolvedValue({ models: [] })
    getUserSpendingRanking.mockResolvedValue({ ranking: [] })
    getUserUsageTrend.mockResolvedValue({ trend: [] })

    await wrapper.get('[data-testid="refresh"]').trigger('click')
    expect(wrapper.findComponent(ToolbarStub).props('refreshing')).toBe(true)
    await flushPromises()

    expect(getUsageTrend).toHaveBeenCalledTimes(2)
    expect(heatmapReload).toHaveBeenCalledTimes(1)
    expect(liveReload).toHaveBeenCalledTimes(1)
    expect(getUserSpendingRanking).toHaveBeenCalledTimes(1)
    expect(getUserUsageTrend).toHaveBeenCalledTimes(1)
    expect(wrapper.findComponent(ToolbarStub).props('refreshing')).toBe(false)
  })

  it('用户趋势可切到按实际消费，消费榜点击进入该用户的使用记录', async () => {
    const wrapper = await mountPage()
    wrapper.findComponent(UserTrendStub).vm.$emit('update:metric', 'actual_cost')
    await flushPromises()
    expect(getUserUsageTrend).toHaveBeenLastCalledWith(expect.objectContaining({ metric: 'actual_cost', limit: 12 }))
    expect(wrapper.findComponent(UserTrendStub).props('metric')).toBe('actual_cost')

    wrapper.findComponent(DistributionStub).vm.$emit('ranking-click', { user_id: 9 })
    expect(push).toHaveBeenCalledWith({
      path: '/admin/usage',
      query: { user_id: '9', start_date: daysAgo(6), end_date: daysAgo(0) },
    })
  })

  it('入场动画只播放一次：最后一块播放完后移除入场类', async () => {
    const wrapper = await mountPage()
    const root = wrapper.get('[data-testid="admin-dashboard-root"]')
    expect(root.classes()).toContain('dash-rise')
    await wrapper.get('[data-stub="UserDashboardHeatmap"]').trigger('animationend')
    expect(root.classes()).toContain('dash-rise')
    await wrapper.get('[data-rise-last="true"]').trigger('animationend')
    expect(root.classes()).not.toContain('dash-rise')
  })
})
