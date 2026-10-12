import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import type { DashboardStats } from '@/types'
import AdminOverviewCards from '../AdminOverviewCards.vue'
import AdminQuickActions from '../AdminQuickActions.vue'

const { push, canUseBatchImage } = vi.hoisted(() => ({ push: vi.fn(), canUseBatchImage: { value: false } }))

vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({ canUseBatchImage, refreshBatchImageAccess: vi.fn() }),
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const stats = (overrides: Partial<DashboardStats> = {}) => ({
  total_accounts: 12,
  normal_accounts: 9,
  error_accounts: 2,
  ratelimit_accounts: 1,
  total_users: 1200,
  today_new_users: 7,
  active_users: 35,
  total_api_keys: 88,
  active_api_keys: 60,
  total_tokens: 2_500_000_000,
  total_actual_cost: 1234.5,
  total_account_cost: 800,
  total_cost: 5000,
  today_actual_cost: 3.5,
  today_account_cost: 0.05,
  today_cost: 12,
  ...overrides,
}) as DashboardStats

describe('AdminOverviewCards', () => {
  it('shows accounts, users, keys and lifetime usage with actual / account cost / standard spend', () => {
    const wrapper = mount(AdminOverviewCards, { props: { stats: stats() }, global: { stubs: { Icon: true } } })

    // 测试环境默认「减少动态效果」，滚动数字直接显示终值
    const accounts = wrapper.get('[data-testid="overview-accounts"]').text()
    expect(accounts).toContain('12')
    expect(accounts).toContain('9 common.active')
    expect(accounts).toContain('2 common.error')
    expect(accounts).toContain('1 admin.dashboard.fork.rateLimited')
    expect(wrapper.get('[data-testid="overview-users"]').text()).toContain('+7 admin.dashboard.fork.newToday')
    expect(wrapper.get('[data-testid="overview-keys"]').text()).toContain('60 common.active')
    expect(wrapper.get('[data-testid="overview-usage"]').text()).toContain('2.50B')

    const total = wrapper.get('[data-testid="overview-costs-total"]').findAll('span').map((span) => span.text())
    expect(total).toEqual(['admin.dashboard.fork.total', '$1,234.50', '/', '$800.00', '/', '$5,000.00'])
    const today = wrapper.get('[data-testid="overview-costs-today"]').findAll('span').map((span) => span.text())
    expect(today).toEqual(['admin.dashboard.fork.today', '$3.50', '/', '$0.050', '/', '$12.00'])
  })

  it('hides account error / rate-limit counts when there are none, and shows placeholders before stats arrive', () => {
    const wrapper = mount(AdminOverviewCards, {
      props: { stats: stats({ error_accounts: 0, ratelimit_accounts: 0 }) },
      global: { stubs: { Icon: true } },
    })
    expect(wrapper.get('[data-testid="overview-accounts"]').text()).not.toContain('common.error')
    expect(wrapper.get('[data-testid="overview-accounts"]').text()).not.toContain('rateLimited')

    const empty = mount(AdminOverviewCards, { props: { stats: null }, global: { stubs: { Icon: true } } })
    expect(empty.text()).not.toContain('$')
  })
})

describe('AdminQuickActions', () => {
  it('lists user, account and group management, plus batch image only with access', async () => {
    canUseBatchImage.value = false
    const wrapper = mount(AdminQuickActions, { global: { stubs: { Icon: true } } })
    expect(wrapper.findAll('button').map((button) => button.attributes('data-testid'))).toEqual([
      'users-quick-action',
      'accounts-quick-action',
      'group-pricing-quick-action',
    ])
    await wrapper.get('[data-testid="accounts-quick-action"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/admin/accounts')

    canUseBatchImage.value = true
    const withImage = mount(AdminQuickActions, { global: { stubs: { Icon: true } } })
    expect(withImage.find('[data-testid="batch-image-quick-action"]').exists()).toBe(true)
  })
})
