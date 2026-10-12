import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { LeaderboardModelsResponse, LeaderboardUserItem, LeaderboardUsersResponse } from '@/api/leaderboard'
import LeaderboardView from '../LeaderboardView.vue'

const { getModelLeaderboard, getUserLeaderboard, settings, routeQuery, replace } = vi.hoisted(() => ({
  getModelLeaderboard: vi.fn(),
  getUserLeaderboard: vi.fn(),
  settings: { model_ranking_enabled: true, usage_ranking_enabled: true } as Record<string, unknown>,
  routeQuery: {} as Record<string, string>,
  replace: vi.fn(),
}))

vi.mock('@/api/leaderboard', async () => {
  const actual = await vi.importActual<typeof import('@/api/leaderboard')>('@/api/leaderboard')
  return { ...actual, getModelLeaderboard, getUserLeaderboard }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({ cachedPublicSettings: settings }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: routeQuery }),
  useRouter: () => ({ replace }),
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

// 日期选择器换成一个按钮：点一下就选中 2026-10-01 ~ 2026-10-07（或测试指定的范围）
const pickedRange = { startDate: '2026-10-01', endDate: '2026-10-07' }
const DateRangePickerStub = {
  props: ['startDate', 'endDate'],
  emits: ['update:startDate', 'update:endDate', 'change'],
  template: `<button data-testid="range-picker" :data-start="startDate" :data-end="endDate" @click="pick"></button>`,
  methods: {
    pick(this: { $emit: (event: string, ...args: unknown[]) => void }) {
      this.$emit('update:startDate', pickedRange.startDate)
      this.$emit('update:endDate', pickedRange.endDate)
      this.$emit('change', { ...pickedRange, preset: null })
    },
  },
}

const stubs = {
  AppLayout: { template: '<div><slot /></div>' },
  Icon: { template: '<i></i>' },
  ModelIcon: { template: '<i></i>' },
  DateRangePicker: DateRangePickerStub,
  RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' },
}

function localToday(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}

const models: LeaderboardModelsResponse = {
  enabled: true,
  start_date: '2026-10-12',
  end_date: '2026-10-12',
  total_tokens: 1000,
  other_share_percent: 20,
  items: [
    { rank: 1, model: 'claude-sonnet-4-6', platform: 'anthropic', tokens: 500, requests: 10, share_percent: 50 },
    { rank: 2, model: 'gpt-5.5', platform: 'openai', tokens: 300, requests: 6, share_percent: 30 },
  ],
}

const userItem = (rank: number, overrides: Partial<LeaderboardUserItem> = {}): LeaderboardUserItem => ({
  rank,
  display_name: `u***${rank}`,
  avatar_seed: `seed${rank}`,
  tokens: 1000 - rank,
  requests: 10 + rank,
  is_self: false,
  ...overrides,
})

const users = (overrides: Partial<LeaderboardUsersResponse> = {}): LeaderboardUsersResponse => ({
  enabled: true,
  start_date: '2026-10-12',
  end_date: '2026-10-12',
  sort_by: 'tokens',
  show_cost: false,
  limit: 20,
  insufficient_data: false,
  items: [userItem(1), userItem(2, { is_self: true }), userItem(3), userItem(4), userItem(5, { is_self: false })],
  ...overrides,
})

const mountView = async () => {
  const wrapper = mount(LeaderboardView, { global: { stubs } })
  await flushPromises()
  return wrapper
}

describe('LeaderboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    settings.model_ranking_enabled = true
    settings.usage_ranking_enabled = true
    for (const key of Object.keys(routeQuery)) delete routeQuery[key]
    pickedRange.startDate = '2026-10-01'
    pickedRange.endDate = '2026-10-07'
    getModelLeaderboard.mockResolvedValue(models)
    getUserLeaderboard.mockResolvedValue(users())
  })

  it('opens on the user board for today and puts #1 in the middle of the podium', async () => {
    const wrapper = await mountView()

    const today = localToday()
    expect(getUserLeaderboard).toHaveBeenCalledWith({ start_date: today, end_date: today })
    expect(getModelLeaderboard).not.toHaveBeenCalled()
    expect(wrapper.get('h1').text()).toBe('leaderboard.title')

    // 两个榜都开着：用户在前
    const switches = wrapper.findAll('[data-board]')
    expect(switches.map(button => button.attributes('data-board'))).toEqual(['users', 'models'])
    expect(switches[0].attributes('aria-selected')).toBe('true')

    const podium = wrapper.findAll('[data-testid="user-podium"] article')
    expect(podium).toHaveLength(3)
    expect(podium.find(card => card.attributes('data-rank') === '1')!.classes()).toEqual(
      expect.arrayContaining(['md:order-2', 'md:min-h-[260px]', 'md:p-6']),
    )
    expect(podium.find(card => card.attributes('data-rank') === '2')!.classes()).toEqual(
      expect.arrayContaining(['md:order-1', 'md:min-h-[220px]']),
    )
    expect(podium.find(card => card.attributes('data-rank') === '3')!.classes()).toContain('md:order-3')

    const rows = wrapper.findAll('[data-testid="user-ranking-list"] li')
    expect(rows).toHaveLength(5)
    expect(rows[0].text()).toContain('#1')
    expect(wrapper.get('[data-testid="user-ranking-list"]').text()).toContain('leaderboard.limitHint:20')
    expect(wrapper.get('[data-testid="ranking-range-label"]').text()).toBe('2026-10-12')
  })

  it('highlights the viewer on the podium and in the list', async () => {
    const wrapper = await mountView()

    expect(wrapper.find('[data-testid="user-podium"] article[data-self]').attributes('data-rank')).toBe('2')
    const selfRows = wrapper.findAll('[data-testid="user-ranking-list"] li[data-self]')
    expect(selfRows).toHaveLength(1)
    expect(selfRows[0].text()).toContain('leaderboard.users.you')
  })

  it('shows every visible metric with the admin-chosen sort metric first and no metric switch', async () => {
    getUserLeaderboard.mockResolvedValue(users({ sort_by: 'requests' }))
    const wrapper = await mountView()

    expect(wrapper.find('[data-metric-option]').exists()).toBe(false)
    const firstCard = wrapper.find('[data-testid="user-podium"] article[data-rank="1"]')
    expect(firstCard.text()).toContain('leaderboard.metrics.requests')
    expect(firstCard.find('[data-metric="tokens"]').exists()).toBe(true)
    const columns = wrapper.findAll('[data-testid="user-ranking-list"] li')[0].findAll('[data-column]')
    expect(columns.map(column => column.attributes('data-column'))).toEqual(['requests', 'tokens'])
  })

  it('leaves out spend unless the admin shows it', async () => {
    const wrapper = await mountView()
    expect(wrapper.find('[data-column="cost"]').exists()).toBe(false)
    expect(wrapper.find('[data-metric="cost"]').exists()).toBe(false)

    getUserLeaderboard.mockResolvedValue(users({
      show_cost: true,
      sort_by: 'cost',
      items: [userItem(1, { actual_cost: 12.5 }), userItem(2, { actual_cost: 3 }), userItem(3, { actual_cost: 1 }), userItem(4, { actual_cost: 0.5 })],
    }))
    await wrapper.get('[data-testid="leaderboard-refresh"]').trigger('click')
    await flushPromises()
    const firstRowColumns = wrapper.findAll('[data-testid="user-ranking-list"] li')[0].findAll('[data-column]')
    expect(firstRowColumns.map(column => column.attributes('data-column'))).toEqual(['cost', 'tokens', 'requests'])
    expect(firstRowColumns[0].text()).toContain('$12.50')
  })

  it('refetches the active board when the date range changes', async () => {
    const wrapper = await mountView()
    await wrapper.get('[data-testid="range-picker"]').trigger('click')
    await flushPromises()
    expect(getUserLeaderboard).toHaveBeenLastCalledWith({ start_date: '2026-10-01', end_date: '2026-10-07' })
    expect(getUserLeaderboard).toHaveBeenCalledTimes(2)
  })

  it('refuses ranges longer than 90 days without asking the server', async () => {
    pickedRange.startDate = '2026-01-01'
    pickedRange.endDate = '2026-10-12'
    const wrapper = await mountView()
    await wrapper.get('[data-testid="range-picker"]').trigger('click')
    await flushPromises()
    expect(getUserLeaderboard).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="leaderboard-error"]').text()).toBe('leaderboard.rangeTooLong:90')
  })

  it('switches to the model board, which links each model to the catalog', async () => {
    const wrapper = await mountView()
    await wrapper.get('[data-board="models"]').trigger('click')
    await flushPromises()

    const today = localToday()
    expect(getModelLeaderboard).toHaveBeenCalledWith({ start_date: today, end_date: today })
    expect(replace).toHaveBeenCalledWith({ query: { tab: 'models' } })
    const rows = wrapper.findAll('[data-testid="model-ranking"] li')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('claude-sonnet-4-6')
    expect(rows[0].find('a').attributes('data-to')).toContain('"q":"claude-sonnet-4-6"')
    expect(rows[0].attributes('style')).toContain('--bar-w: 50%')
    expect(wrapper.text()).toContain('leaderboard.models.others')
  })

  it('shows only the model board, without a switch, while the user board is off', async () => {
    settings.usage_ranking_enabled = false
    const wrapper = await mountView()
    expect(wrapper.find('[data-board]').exists()).toBe(false)
    expect(getUserLeaderboard).not.toHaveBeenCalled()
    expect(getModelLeaderboard).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="model-ranking"]').exists()).toBe(true)
  })

  it('opens the board named in the link', async () => {
    routeQuery.tab = 'models'
    const wrapper = await mountView()
    expect(getModelLeaderboard).toHaveBeenCalledTimes(1)
    expect(getUserLeaderboard).not.toHaveBeenCalled()
    expect(wrapper.get('[data-board="models"]').attributes('aria-selected')).toBe('true')
  })

  it('says so when too few users took part', async () => {
    getUserLeaderboard.mockResolvedValue(users({ insufficient_data: true, items: [] }))
    const wrapper = await mountView()
    expect(wrapper.find('[data-testid="leaderboard-insufficient"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="user-podium"]').exists()).toBe(false)
  })

  it('shows the dashed empty state when nobody used anything in the range', async () => {
    getUserLeaderboard.mockResolvedValue(users({ items: [] }))
    const wrapper = await mountView()
    expect(wrapper.get('[data-testid="leaderboard-empty"]').text()).toContain('leaderboard.emptyTitle')
  })

  it('shows the load error', async () => {
    getUserLeaderboard.mockRejectedValue({ message: 'boom' })
    const wrapper = await mountView()
    expect(wrapper.get('[data-testid="leaderboard-error"]').text()).toBe('boom')
  })

  it('shows a closed notice when both boards are off', async () => {
    settings.model_ranking_enabled = false
    settings.usage_ranking_enabled = false
    const wrapper = await mountView()
    expect(wrapper.find('[data-testid="leaderboard-disabled"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="range-picker"]').exists()).toBe(false)
    expect(getModelLeaderboard).not.toHaveBeenCalled()
    expect(getUserLeaderboard).not.toHaveBeenCalled()
  })
})
