import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, onMounted } from 'vue'

import UserDashboardUsageToolbar from '../UserDashboardUsageToolbar.vue'
import UserDashboardFilterDropdown from '../UserDashboardFilterDropdown.vue'
import DashboardDateRangePicker from '../DashboardDateRangePicker.vue'
import { provideDashboardDataSource, type DashboardDataSource } from '../dashboardDataSource'
import { provideUsageChartState } from '../usageChartState'
import { formatDayKey } from '../usageChartData'
import Select from '@/components/common/Select.vue'
import { usageAPI } from '@/api/usage'

const mocks = vi.hoisted(() => ({ settings: null as Record<string, unknown> | null }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'en' },
    }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: mocks.settings }),
}))

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardTrend: vi.fn(),
    getDashboardModels: vi.fn(),
  },
}))

vi.mock('@/api/keys', () => ({
  keysAPI: {
    list: vi.fn().mockResolvedValue({ items: [{ id: 7, name: 'personal-key' }], total: 1, page: 1, page_size: 100, pages: 1 }),
  },
}))

vi.mock('@/api/groups', () => ({
  userGroupsAPI: {
    getAvailable: vi.fn().mockResolvedValue([{ id: 3, name: 'Claude' }]),
  },
}))

enableAutoUnmount(afterEach)

// 模拟仪表盘页面：提供共享状态并渲染标题行工具栏。
const ToolbarHarness = defineComponent({
  inheritAttrs: false,
  setup(_, { attrs }) {
    const state = provideUsageChartState()
    onMounted(() => {
      void state.load()
      void state.loadFilterOptions()
    })
    return () => h(UserDashboardUsageToolbar, attrs)
  },
})

const mountToolbar = async (props: Record<string, unknown> = {}) => {
  const wrapper = mount(ToolbarHarness, {
    props,
    attachTo: document.body,
    global: {
      stubs: {
        Icon: true,
        Select: true,
        DashboardDateRangePicker: true,
        MotionTransition: { template: '<div><slot /></div>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

const selectByTestId = (wrapper: Awaited<ReturnType<typeof mountToolbar>>, testId: string) =>
  wrapper.findAllComponents(Select).find((item) => item.attributes('data-testid') === testId)!

// 模拟在 Select 里选中一项：先更新取值，再通知变更。
const choose = async (select: ReturnType<typeof selectByTestId>, value: unknown) => {
  select.vm.$emit('update:modelValue', value)
  select.vm.$emit('change', value, null)
  await flushPromises()
}

const openFilters = async (wrapper: Awaited<ReturnType<typeof mountToolbar>>) => {
  await wrapper.get('button[aria-haspopup="dialog"]').trigger('click')
}

const trendCalls = () => vi.mocked(usageAPI.getDashboardTrend).mock.calls.map((call) => call[0]!)

const daysAgo = (days: number) => {
  const date = new Date()
  date.setDate(date.getDate() - days)
  return formatDayKey(date)
}

describe('UserDashboardUsageToolbar', () => {
  beforeEach(() => {
    localStorage.clear()
    mocks.settings = null
    vi.mocked(usageAPI.getDashboardTrend).mockReset()
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({ trend: [], start_date: '', end_date: '', granularity: 'day' })
    vi.mocked(usageAPI.getDashboardModels).mockReset()
    vi.mocked(usageAPI.getDashboardModels).mockResolvedValue({ models: [], start_date: '', end_date: '' })
  })

  it('按 TokenRouter 的布局：宽度按页面内边距封顶，筛选面板三列、七个字段', async () => {
    const wrapper = await mountToolbar()
    const root = wrapper.get('[data-testid="usage-toolbar"]')
    expect(root.classes()).toEqual(expect.arrayContaining(['max-w-[calc(100vw-2rem)]', 'md:max-w-[calc(100vw-3rem)]']))
    expect(wrapper.findComponent(UserDashboardFilterDropdown).props('columns')).toBe(3)

    await openFilters(wrapper)
    expect(wrapper.get('.filter-panel-body').classes()).toContain('lg:grid-cols-3')
    expect(wrapper.findAll('label.filter-field-label').map((label) => label.text())).toEqual([
      'usage.apiKeyFilter',
      'usage.model',
      'admin.usage.group',
      'usage.type',
      'admin.usage.billingType',
      'admin.usage.billingMode',
      'usage.compactionFilter',
    ])
  })

  it('管理员口径（全站）没有密钥筛选，也不显示管理员接口不支持的计费模式和请求类别', async () => {
    const adminSource: DashboardDataSource = {
      scope: 'admin',
      getTrend: vi.fn().mockResolvedValue([]),
      getModels: vi.fn().mockResolvedValue([]),
      getLiveStats: vi.fn(),
      loadGroups: vi.fn().mockResolvedValue([{ id: 3, name: 'vip' }]),
      usagePath: '/admin/usage',
      unsupportedFilters: ['billing_mode', 'native_compaction_v2'],
    }
    const AdminHarness = defineComponent({
      setup() {
        const state = provideUsageChartState(provideDashboardDataSource(adminSource))
        onMounted(() => {
          void state.load()
          void state.loadFilterOptions()
        })
        return () => h(UserDashboardUsageToolbar)
      },
    })
    const wrapper = mount(AdminHarness, {
      attachTo: document.body,
      global: { stubs: { Icon: true, Select: true, DashboardDateRangePicker: true, MotionTransition: { template: '<div><slot /></div>' } } },
    })
    await flushPromises()

    await openFilters(wrapper)
    expect(wrapper.findAll('label.filter-field-label').map((label) => label.text())).toEqual([
      'usage.model',
      'admin.usage.group',
      'usage.type',
      'admin.usage.billingType',
    ])
    expect(adminSource.getTrend).toHaveBeenCalled()
    expect(adminSource.loadGroups).toHaveBeenCalledTimes(1)
    expect(usageAPI.getDashboardTrend).not.toHaveBeenCalled()
  })

  it('订阅功能关闭时隐藏计费类型', async () => {
    mocks.settings = { subscription_enabled: false }
    const wrapper = await mountToolbar()
    await openFilters(wrapper)
    expect(wrapper.findAll('label.filter-field-label').map((label) => label.text())).not.toContain('admin.usage.billingType')
    expect(wrapper.findAllComponents(Select)).toHaveLength(6)
  })

  it('请求类别筛选把 native_compaction_v2 带到趋势和模型请求里，false 也照样带上', async () => {
    const wrapper = await mountToolbar()
    // 字段只在面板展开后渲染
    await openFilters(wrapper)
    const select = selectByTestId(wrapper, 'usage-filter-compaction')
    expect(select.props('options')).toEqual([
      { value: null, label: 'usage.allCompactionTypes' },
      { value: true, label: 'usage.compactionOnly' },
      { value: false, label: 'dashboard.fork.filters.legacyCompaction' },
    ])

    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    vi.mocked(usageAPI.getDashboardModels).mockClear()
    await choose(select, true)
    expect(trendCalls()).toHaveLength(2)
    expect(trendCalls().every((params) => params.native_compaction_v2 === true)).toBe(true)
    expect(vi.mocked(usageAPI.getDashboardModels).mock.calls[0][0]).toMatchObject({ native_compaction_v2: true })

    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    await choose(select, false)
    expect(trendCalls()[0]).toMatchObject({ native_compaction_v2: false })

    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    await choose(select, null)
    expect(trendCalls()[0]).not.toHaveProperty('native_compaction_v2')
  })

  it('已选条件显示为标签和数量，移除标签清空该条件并重新取数，重置清空全部', async () => {
    const wrapper = await mountToolbar()
    await openFilters(wrapper)
    expect(wrapper.find('[data-testid="filter-chips"]').exists()).toBe(false)
    await choose(selectByTestId(wrapper, 'usage-filter-group'), 3)
    await choose(selectByTestId(wrapper, 'usage-filter-compaction'), false)

    expect(wrapper.get('.filter-trigger-count').text()).toBe('2')
    expect(wrapper.get('.filter-panel-count').text()).toBe('2')
    expect(wrapper.findAll('.filter-chip').map((chip) => chip.text())).toEqual([
      'admin.usage.groupClaude',
      'usage.compactionFilterdashboard.fork.filters.legacyCompaction',
    ])

    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    await wrapper.get('.filter-chip-remove').trigger('click')
    await flushPromises()
    expect(trendCalls()[0]).not.toHaveProperty('group_id')
    expect(trendCalls()[0]).toMatchObject({ native_compaction_v2: false })
    expect(selectByTestId(wrapper, 'usage-filter-group').props('modelValue')).toBeNull()
    expect(wrapper.findAll('.filter-chip')).toHaveLength(1)

    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    await wrapper.get('[data-testid="filter-reset"]').trigger('click')
    await flushPromises()
    expect(trendCalls()[0]).not.toHaveProperty('native_compaction_v2')
    expect(wrapper.find('[data-testid="filter-chips"]').exists()).toBe(false)
    expect(wrapper.find('.filter-trigger-count').exists()).toBe(false)
    expect(wrapper.get('[data-testid="filter-reset"]').attributes('disabled')).toBeDefined()
  })

  it.each([
    ['today', () => daysAgo(0), () => daysAgo(0), 'hour'],
    ['yesterday', () => daysAgo(1), () => daysAgo(1), 'hour'],
    ['14days', () => daysAgo(13), () => daysAgo(0), 'day'],
    ['lastMonth', () => {
      const now = new Date()
      return formatDayKey(new Date(now.getFullYear(), now.getMonth() - 1, 1))
    }, () => {
      const now = new Date()
      return formatDayKey(new Date(now.getFullYear(), now.getMonth(), 0))
    }, 'day'],
  ])('日期选择器的「%s」按整天区间查询', async (preset, start, end, granularity) => {
    const wrapper = await mountToolbar()
    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    wrapper.findComponent(DashboardDateRangePicker).vm.$emit('change', { startDate: start(), endDate: end(), preset })
    await flushPromises()

    expect(trendCalls()[0]).toMatchObject({ start_date: start(), end_date: end(), granularity })
    const picker = wrapper.findComponent(DashboardDateRangePicker)
    expect(picker.props('startDate')).toBe(start())
    expect(picker.props('endDate')).toBe(end())
  })

  it('本月从 1 日查到今天', async () => {
    const wrapper = await mountToolbar()
    const now = new Date()
    const first = formatDayKey(new Date(now.getFullYear(), now.getMonth(), 1))
    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    wrapper.findComponent(DashboardDateRangePicker).vm.$emit('change', { startDate: first, endDate: daysAgo(0), preset: 'thisMonth' })
    await flushPromises()
    expect(trendCalls()[0]).toMatchObject({ start_date: first, end_date: daysAgo(0) })
  })

  it('近 7 天 / 30 天 / 24 小时映射成快捷范围', async () => {
    const wrapper = await mountToolbar()
    const picker = wrapper.findComponent(DashboardDateRangePicker)

    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    picker.vm.$emit('change', { startDate: daysAgo(29), endDate: daysAgo(0), preset: '30days' })
    await flushPromises()
    expect(trendCalls()[0]).toMatchObject({ start_date: daysAgo(29), end_date: daysAgo(0), granularity: 'day' })

    vi.mocked(usageAPI.getDashboardTrend).mockClear()
    picker.vm.$emit('change', { startDate: daysAgo(1), endDate: daysAgo(0), preset: 'last24Hours' })
    await flushPromises()
    expect(trendCalls()[0]).toMatchObject({ granularity: 'hour' })
  })

  it('刷新按钮通知页面刷新，刷新中禁用并转动图标（尊重减少动效）', async () => {
    const wrapper = await mountToolbar()
    const button = wrapper.get('[data-testid="usage-refresh"]')
    await button.trigger('click')
    expect(wrapper.findComponent(UserDashboardUsageToolbar).emitted('refresh')).toHaveLength(1)

    await wrapper.setProps({ refreshing: true })
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.get('icon-stub').classes()).toEqual(expect.arrayContaining(['animate-spin', 'motion-reduce:animate-none']))
  })
})
