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

const stubs = {
  AppLayout: { template: '<div><slot /></div>' },
  Icon: { template: '<i></i>' },
  ModelIcon: { template: '<i></i>' },
  RouterLink: { props: ['to'], template: '<a :data-to="JSON.stringify(to)"><slot /></a>' },
}

const models: LeaderboardModelsResponse = {
  enabled: true,
  window: '7d',
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
  requests: 10,
  is_self: false,
  ...overrides,
})

const users = (overrides: Partial<LeaderboardUsersResponse> = {}): LeaderboardUsersResponse => ({
  enabled: true,
  window: '7d',
  metric: 'tokens',
  show_cost: false,
  insufficient_data: false,
  items: [userItem(1), userItem(2, { is_self: true }), userItem(3), userItem(4)],
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
    getModelLeaderboard.mockResolvedValue(models)
    getUserLeaderboard.mockResolvedValue(users())
  })

  it('opens on the model board and links each model to the catalog', async () => {
    const wrapper = await mountView()

    expect(getModelLeaderboard).toHaveBeenCalledWith('7d')
    expect(getUserLeaderboard).not.toHaveBeenCalled()
    const rows = wrapper.findAll('[data-testid="model-ranking"] li')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('claude-sonnet-4-6')
    expect(rows[0].find('a').attributes('data-to')).toContain('"q":"claude-sonnet-4-6"')
    expect(rows[0].attributes('style')).toContain('--bar-w: 50%')
    expect(wrapper.text()).toContain('leaderboard.models.others')
  })

  it('refetches when the time range changes', async () => {
    const wrapper = await mountView()
    await wrapper.get('[data-window="30d"]').trigger('click')
    await flushPromises()
    expect(getModelLeaderboard).toHaveBeenLastCalledWith('30d')
  })

  it('hides the user tab while the user board is off', async () => {
    settings.usage_ranking_enabled = false
    const wrapper = await mountView()
    expect(wrapper.find('[data-tab="users"]').exists()).toBe(false)
  })

  it('puts #1 in the middle of the podium and highlights the viewer', async () => {
    const wrapper = await mountView()
    await wrapper.get('[data-tab="users"]').trigger('click')
    await flushPromises()

    expect(getUserLeaderboard).toHaveBeenCalledWith('7d', 'tokens')
    expect(replace).toHaveBeenCalledWith({ query: { tab: 'users' } })
    const podium = wrapper.findAll('[data-testid="user-podium"] article')
    expect(podium).toHaveLength(3)
    expect(podium.find(card => card.attributes('data-rank') === '1')!.classes()).toContain('md:order-2')
    expect(podium.find(card => card.attributes('data-rank') === '2')!.classes()).toContain('md:order-1')
    expect(podium.find(card => card.attributes('data-rank') === '3')!.classes()).toContain('md:order-3')

    const selfRows = wrapper.findAll('[data-testid="user-ranking-list"] li[data-self]')
    expect(selfRows).toHaveLength(1)
    expect(selfRows[0].text()).toContain('leaderboard.users.you')
  })

  it('leaves out spend unless the admin shows it', async () => {
    routeQuery.tab = 'users'
    const wrapper = await mountView()
    expect(wrapper.find('[data-metric="cost"]').exists()).toBe(false)
    expect(wrapper.find('[data-column="cost"]').exists()).toBe(false)

    getUserLeaderboard.mockResolvedValue(users({ show_cost: true, items: [userItem(1, { actual_cost: 12.5 }), userItem(2, { actual_cost: 3 })] }))
    await wrapper.get('[data-window="24h"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-metric="cost"]').exists()).toBe(true)
    expect(wrapper.find('[data-column="cost"]').text()).toBe('$12.50')
  })

  it('says so when too few users took part', async () => {
    routeQuery.tab = 'users'
    getUserLeaderboard.mockResolvedValue(users({ insufficient_data: true, items: [] }))
    const wrapper = await mountView()
    expect(wrapper.find('[data-testid="leaderboard-insufficient"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="user-podium"]').exists()).toBe(false)
  })

  it('shows a closed notice when both boards are off', async () => {
    settings.model_ranking_enabled = false
    settings.usage_ranking_enabled = false
    const wrapper = await mountView()
    expect(wrapper.text()).toContain('leaderboard.disabled')
    expect(getModelLeaderboard).not.toHaveBeenCalled()
  })
})
