import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import UserDashboardPage from '../UserDashboardPage.vue'
import { usageAPI } from '@/api/usage'

const { refreshUser, fetchAnnouncements, getMyPlatformQuotas, heatmapReload, liveReload } = vi.hoisted(() => ({
  refreshUser: vi.fn(),
  fetchAnnouncements: vi.fn(),
  getMyPlatformQuotas: vi.fn(),
  heatmapReload: vi.fn(),
  liveReload: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: { value: 'en' } }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ refreshUser, isSimpleMode: false }),
}))

vi.mock('@/stores/announcements', () => ({
  useAnnouncementStore: () => ({ fetchAnnouncements }),
}))

vi.mock('@/api/user', () => ({ getMyPlatformQuotas }))

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardTrend: vi.fn(),
    getDashboardModels: vi.fn(),
  },
}))

vi.mock('@/api/keys', () => ({
  keysAPI: { list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 1 }) },
}))

vi.mock('@/api/groups', () => ({
  userGroupsAPI: { getAvailable: vi.fn().mockResolvedValue([]) },
}))

// 带 reload 的子组件替身，刷新按钮会调用它们
const reloadable = (name: string, reload: () => unknown) => defineComponent({
  name,
  emits: ['update', 'select-day'],
  setup(_, { expose }) {
    expose({ reload })
    return () => h('div', { 'data-stub': name })
  },
})

const LiveStatsStub = reloadable('UserDashboardLiveStats', liveReload)
const HeatmapStub = reloadable('UserDashboardHeatmap', heatmapReload)

const PlatformsStub = defineComponent({
  name: 'UserDashboardPlatforms',
  props: ['stats', 'platformQuotas', 'isSimple'],
  setup: () => () => h('div', { 'data-stub': 'UserDashboardPlatforms' }),
})

const LifetimeStub = defineComponent({
  name: 'UserDashboardLifetime',
  props: ['stats'],
  setup: () => () => h('div', { 'data-stub': 'UserDashboardLifetime' }),
})

const QuickActionsStub = defineComponent({
  name: 'UserDashboardQuickActions',
  setup: () => () => h('div', { 'data-stub': 'UserDashboardQuickActions' }),
})

const ToolbarStub = defineComponent({
  name: 'UserDashboardUsageToolbar',
  props: ['refreshing'],
  emits: ['refresh'],
  setup: (_, { emit }) => () => h('button', { 'data-stub': 'UserDashboardUsageToolbar', onClick: () => emit('refresh') }),
})

const mountPage = async () => {
  const wrapper = mount(UserDashboardPage, {
    global: {
      stubs: {
        UserDashboardLiveStats: LiveStatsStub,
        UserDashboardHeatmap: HeatmapStub,
        UserDashboardPlatforms: PlatformsStub,
        UserDashboardLifetime: LifetimeStub,
        UserDashboardQuickActions: QuickActionsStub,
        UserDashboardUsageToolbar: ToolbarStub,
        UserDashboardUsageChart: { template: '<div data-stub="UserDashboardUsageChart" />' },
        UserDashboardAnnouncements: { template: '<div data-stub="UserDashboardAnnouncements" />' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('UserDashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    getMyPlatformQuotas.mockResolvedValue({ platform_quotas: [{ platform: 'openai', daily_limit_usd: 5 }] })
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({ trend: [], start_date: '', end_date: '', granularity: 'day' })
    vi.mocked(usageAPI.getDashboardModels).mockResolvedValue({ models: [], start_date: '', end_date: '' })
  })

  it('按 TokenRouter 的顺序排布：标题行、用量区、热力图、公告与快捷操作，之后是本项目的累计用量和平台拆分', async () => {
    const wrapper = await mountPage()
    const order = wrapper.findAll('[data-stub]').map((node) => node.attributes('data-stub'))
    expect(order).toEqual([
      'UserDashboardLiveStats',
      'UserDashboardUsageToolbar',
      'UserDashboardUsageChart',
      'UserDashboardHeatmap',
      'UserDashboardAnnouncements',
      'UserDashboardQuickActions',
      'UserDashboardLifetime',
      'UserDashboardPlatforms',
    ])
    // 标题行照 TokenRouter：大标题和说明在正文里
    expect(wrapper.get('h1').text()).toBe('dashboard.title')
    expect(refreshUser).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledTimes(2)
    expect(wrapper.findComponent(PlatformsStub).props('platformQuotas')).toEqual([{ platform: 'openai', daily_limit_usd: 5 }])
  })

  it('实时统计交给累计用量和平台拆分，失败的轮询不清掉上一次结果', async () => {
    const wrapper = await mountPage()
    const stats = { total_api_keys: 4, active_api_keys: 2, by_platform: [] }
    wrapper.findComponent(LiveStatsStub).vm.$emit('update', stats)
    await flushPromises()
    expect(wrapper.findComponent(PlatformsStub).props('stats')).toEqual(stats)
    expect(wrapper.findComponent(LifetimeStub).props('stats')).toEqual(stats)

    wrapper.findComponent(LiveStatsStub).vm.$emit('update', null)
    await flushPromises()
    expect(wrapper.findComponent(PlatformsStub).props('stats')).toEqual(stats)
  })

  it('刷新按钮刷新整页：账户、用量区、热力图、实时统计、平台配额和公告', async () => {
    const wrapper = await mountPage()
    vi.clearAllMocks()
    getMyPlatformQuotas.mockResolvedValue({ platform_quotas: [] })

    await wrapper.get('[data-stub="UserDashboardUsageToolbar"]').trigger('click')
    expect(wrapper.findComponent(ToolbarStub).props('refreshing')).toBe(true)
    await flushPromises()

    expect(refreshUser).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledTimes(2)
    expect(heatmapReload).toHaveBeenCalledTimes(1)
    expect(liveReload).toHaveBeenCalledTimes(1)
    expect(getMyPlatformQuotas).toHaveBeenCalledTimes(1)
    expect(fetchAnnouncements).toHaveBeenCalledWith(true)
    expect(wrapper.findComponent(ToolbarStub).props('refreshing')).toBe(false)
  })

  it('入场动画只播放一次：最后一块播放完后移除入场类', async () => {
    const wrapper = await mountPage()
    const root = wrapper.get('[data-testid="dashboard-root"]')
    expect(root.classes()).toContain('dash-rise')

    // 其他区块的动画结束不影响
    await wrapper.get('[data-stub="UserDashboardHeatmap"]').trigger('animationend')
    expect(root.classes()).toContain('dash-rise')

    await wrapper.get('[data-rise-last="true"]').trigger('animationend')
    expect(root.classes()).not.toContain('dash-rise')
  })
})
