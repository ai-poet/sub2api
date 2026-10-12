import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import HomeLiveStats from '../HomeLiveStats.vue'

const { getPublicStats, settings } = vi.hoisted(() => ({
  getPublicStats: vi.fn(),
  settings: { public_stats_enabled: true } as Record<string, unknown>,
}))

vi.mock('@/api/publicStats', () => ({ getPublicStats }))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ cachedPublicSettings: settings }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

// 让 IntersectionObserver 立刻报告「可见」，计数开始滚动
class VisibleObserver {
  constructor(private readonly callback: IntersectionObserverCallback) {}
  observe(target: Element) {
    this.callback([{ isIntersecting: true, target } as IntersectionObserverEntry], this as unknown as IntersectionObserver)
  }
  disconnect() {}
  unobserve() {}
}

describe('HomeLiveStats', () => {
  const originalObserver = globalThis.IntersectionObserver

  beforeEach(() => {
    vi.clearAllMocks()
    settings.public_stats_enabled = true
    globalThis.IntersectionObserver = VisibleObserver as unknown as typeof IntersectionObserver
  })

  afterEach(() => {
    globalThis.IntersectionObserver = originalObserver
  })

  it('renders nothing and requests nothing while the switch is off', async () => {
    settings.public_stats_enabled = false
    const wrapper = mount(HomeLiveStats)
    await flushPromises()
    expect(getPublicStats).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="home-live-stats"]').exists()).toBe(false)
  })

  it('counts up to the totals with the unit locked to the target', async () => {
    getPublicStats.mockResolvedValue({
      enabled: true,
      available: true,
      total_tokens: 1_234_000_000,
      today_tokens: 5_600_000,
      total_requests: 987_654,
      as_of: '2026-10-12T10:00:00Z',
    })
    const wrapper = mount(HomeLiveStats)
    await flushPromises()

    expect(getPublicStats).toHaveBeenCalledTimes(1)
    // 测试环境默认「减少动态效果」，直接落在终值；读屏文本同样是终值
    expect(wrapper.get('[data-stat="total"]').text()).toContain('1.2B')
    expect(wrapper.get('[data-stat="today"]').text()).toContain('5.6M')
    expect(wrapper.get('[data-stat="requests"]').text()).toContain('987.7K')
    expect(wrapper.text()).toContain('home.liveStats.asOf')
  })

  it('waits until the section scrolls into view before counting and rising', async () => {
    let reveal: (() => void) | undefined
    class DeferredObserver {
      constructor(private readonly callback: IntersectionObserverCallback) {}
      observe(target: Element) {
        reveal = () => this.callback([{ isIntersecting: true, target } as IntersectionObserverEntry], this as unknown as IntersectionObserver)
      }
      disconnect() {}
      unobserve() {}
    }
    globalThis.IntersectionObserver = DeferredObserver as unknown as typeof IntersectionObserver
    getPublicStats.mockResolvedValue({ enabled: true, available: true, total_tokens: 2_000_000, today_tokens: 0, total_requests: 10 })
    const wrapper = mount(HomeLiveStats)
    await flushPromises()

    const section = wrapper.get('[data-test="home-live-stats"]')
    expect(section.classes()).not.toContain('live-stats-visible')
    expect(wrapper.get('[data-stat="total"]').text()).toContain('0.0M')

    reveal?.()
    await flushPromises()
    expect(section.classes()).toContain('live-stats-visible')
    expect(wrapper.get('[data-stat="total"]').text()).toContain('2.0M')
  })

  it('hides itself when the data is unavailable', async () => {
    getPublicStats.mockResolvedValue({ enabled: true, available: false, total_tokens: 0, today_tokens: 0, total_requests: 0 })
    const wrapper = mount(HomeLiveStats)
    await flushPromises()
    expect(wrapper.find('[data-test="home-live-stats"]').exists()).toBe(false)
  })
})
