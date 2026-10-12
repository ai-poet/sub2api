import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserDashboardLiveStats from '../UserDashboardLiveStats.vue'
import { usageAPI } from '@/api/usage'

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

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardStats: vi.fn(),
  },
}))

const stats = { rpm: 12, tpm: 3400, average_duration_ms: 850, today_actual_cost: 2.5 }

// setVisibility 模拟页面切到后台或回到前台。
const setVisibility = (state: DocumentVisibilityState) => {
  Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => state })
  document.dispatchEvent(new Event('visibilitychange'))
}

describe('UserDashboardLiveStats', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(usageAPI.getDashboardStats).mockReset()
    vi.mocked(usageAPI.getDashboardStats).mockResolvedValue(stats as any)
  })

  afterEach(() => {
    setVisibility('visible')
    vi.useRealTimers()
  })

  it('显示 RPM、TPM、平均耗时和今日消费，并把统计交给页面', async () => {
    const wrapper = mount(UserDashboardLiveStats)
    await flushPromises()
    expect(wrapper.get('[data-testid="live-stat-rpm"]').text()).toContain('12')
    expect(wrapper.get('[data-testid="live-stat-tpm"]').text()).toContain('3.4K')
    expect(wrapper.get('[data-testid="live-stat-latency"]').text()).toContain('850ms')
    expect(wrapper.get('[data-testid="live-stat-todayCost"]').text()).toContain('$2.50')
    expect(wrapper.emitted('update')?.[0]).toEqual([stats])
    wrapper.unmount()
  })

  it('每分钟轮询一次，页面隐藏时暂停，回到页面立即刷新', async () => {
    const wrapper = mount(UserDashboardLiveStats)
    await flushPromises()
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(60 * 1000)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(2)

    setVisibility('hidden')
    await vi.advanceTimersByTimeAsync(3 * 60 * 1000)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(2)

    setVisibility('visible')
    await flushPromises()
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(3)

    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(60 * 1000)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(3)
  })

  it('过期响应被丢弃，只保留最后一次结果', async () => {
    let resolveFirst!: (value: unknown) => void
    vi.mocked(usageAPI.getDashboardStats)
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }) as any)
      .mockResolvedValueOnce({ ...stats, rpm: 99 } as any)
    const wrapper = mount(UserDashboardLiveStats)
    await (wrapper.vm as unknown as { reload: () => Promise<void> }).reload()
    resolveFirst({ ...stats, rpm: 1 })
    await flushPromises()
    expect(wrapper.get('[data-testid="live-stat-rpm"]').text()).toContain('99')
    wrapper.unmount()
  })

  it('取数失败时数值显示破折号，不抛出错误', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(usageAPI.getDashboardStats).mockRejectedValue(new Error('boom'))
    const wrapper = mount(UserDashboardLiveStats)
    await flushPromises()
    expect(wrapper.get('[data-testid="live-stat-rpm"]').text()).toContain('—')
    expect(wrapper.emitted('update')?.[0]).toEqual([null])
    wrapper.unmount()
  })
})
