/**
 * 用户仪表盘快捷操作测试
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick, ref } from 'vue'
import { useAppStore } from '@/stores/app'
import type { PublicSettings } from '@/types'
import UserDashboardQuickActions from '../UserDashboardQuickActions.vue'

const routerPush = vi.fn()
const refreshBatchImageAccess = vi.fn()
const canUseBatchImage = ref(false)

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key),
    }),
  }
})

vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({
    canUseBatchImage,
    refreshBatchImageAccess,
  }),
}))

describe('UserDashboardQuickActions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    canUseBatchImage.value = false
  })

  it('仅在站点开启购买时展示购买入口，并按计费模式命名', async () => {
    const appStore = useAppStore()
    const wrapper = mount(UserDashboardQuickActions)

    expect(wrapper.find('[data-testid="purchase-quick-action"]').exists()).toBe(false)

    appStore.cachedPublicSettings = {
      purchase_subscription_enabled: true,
      subscription_enabled: false,
    } as PublicSettings
    await nextTick()

    const purchaseAction = wrapper.find('[data-testid="purchase-quick-action"]')
    expect(purchaseAction.exists()).toBe(true)
    expect(purchaseAction.text()).toContain('nav.recharge')

    await purchaseAction.trigger('click')
    expect(routerPush).toHaveBeenCalledWith('/purchase')
    wrapper.unmount()
  })

  it('创建密钥一行显示密钥数量，没有统计时显示默认说明', async () => {
    const wrapper = mount(UserDashboardQuickActions)
    expect(wrapper.get('[data-testid="api-key-quick-action-desc"]').text()).toBe('dashboard.generateNewKey')

    await wrapper.setProps({ apiKeys: { total: 5, active: 3 } })
    expect(wrapper.get('[data-testid="api-key-quick-action-desc"]').text())
      .toBe('dashboard.fork.quickActions.keyCount:{"active":3,"total":5}')

    await wrapper.get('[data-testid="api-key-quick-action"]').trigger('click')
    expect(routerPush).toHaveBeenCalledWith('/keys')
  })

  it('有批量生图权限时才显示批量生图入口', async () => {
    const wrapper = mount(UserDashboardQuickActions)
    expect(refreshBatchImageAccess).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="batch-image-quick-action"]').exists()).toBe(false)

    canUseBatchImage.value = true
    await nextTick()
    expect(wrapper.find('[data-testid="batch-image-quick-action"]').exists()).toBe(true)
    expect(wrapper.findAll('button').map((button) => button.attributes('data-testid'))).toEqual([
      'api-key-quick-action',
      'usage-quick-action',
      'batch-image-quick-action',
      'redeem-quick-action',
    ])
  })
})
