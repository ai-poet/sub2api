import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import AuthBackground from '../AuthBackground.vue'
import AuthShell from '../AuthShell.vue'

const { loadAnimation, destroy, goToAndStop, appStore } = vi.hoisted(() => ({
  loadAnimation: vi.fn(),
  destroy: vi.fn(),
  goToAndStop: vi.fn(),
  appStore: {
    siteName: 'CheapRouter',
    siteLogo: '',
    publicSettingsLoaded: true,
    cachedPublicSettings: { site_subtitle: '一个 Key 用遍所有模型' } as Record<string, unknown> | null,
    fetchPublicSettings: vi.fn(),
  },
}))

vi.mock('lottie-web', () => ({ default: { loadAnimation } }))
vi.mock('@/assets/lottie/auth-bg-light.json', () => ({ default: { nm: 'light' } }))
vi.mock('@/assets/lottie/auth-bg-dark.json', () => ({ default: { nm: 'dark' } }))
vi.mock('@/stores', () => ({ useAppStore: () => appStore }))
vi.mock('@/composables/useContentTranslation', () => ({ useContentTranslation: () => ({ tx: (text: string) => `tx:${text}` }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

describe('AuthBackground', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    loadAnimation.mockReturnValue({ destroy, goToAndStop })
    document.documentElement.classList.remove('dark')
  })

  afterEach(() => {
    document.documentElement.classList.remove('dark')
  })

  it('loads the animation for the current theme, swaps it when the theme changes and cleans up', async () => {
    const wrapper = mount(AuthBackground, { attachTo: document.body })
    await flushPromises()

    expect(loadAnimation).toHaveBeenCalledTimes(1)
    expect(loadAnimation.mock.calls[0][0]).toMatchObject({ renderer: 'svg', loop: true, animationData: { nm: 'light' } })
    expect(wrapper.get('[data-testid="auth-background"]').classes()).toContain('opacity-100')
    // 测试环境默认「减少动态效果」：不自动播放，停在第一帧
    expect(loadAnimation.mock.calls[0][0].autoplay).toBe(false)
    expect(goToAndStop).toHaveBeenCalledWith(0, true)

    document.documentElement.classList.add('dark')
    await nextTick()
    await flushPromises()
    expect(destroy).toHaveBeenCalledTimes(1)
    expect(loadAnimation.mock.calls[1][0].animationData).toEqual({ nm: 'dark' })

    wrapper.unmount()
    expect(destroy).toHaveBeenCalledTimes(2)
  })

  it('stays hidden instead of throwing when the animation cannot be loaded', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    loadAnimation.mockImplementation(() => { throw new Error('chunk failed') })
    const wrapper = mount(AuthBackground, { attachTo: document.body })
    await flushPromises()
    expect(wrapper.get('[data-testid="auth-background"]').classes()).toContain('opacity-0')
    expect(warn).toHaveBeenCalled()
    warn.mockRestore()
  })
})

describe('AuthShell', () => {
  it('renders the TokenRouter auth layout: brand, translated subtitle, flat card, footer and copyright', () => {
    const wrapper = mount(AuthShell, {
      slots: { default: '<form data-testid="form" />', footer: '<a data-testid="footer-link" />' },
      global: { stubs: { AuthBackground: true } },
    })
    expect(wrapper.classes()).toContain('fork-auth-shell')
    expect(wrapper.get('h1').text()).toBe('CheapRouter')
    expect(wrapper.get('h1').classes()).not.toContain('text-gradient')
    expect(wrapper.text()).toContain('tx:一个 Key 用遍所有模型')
    expect(wrapper.get('.fork-auth-card [data-testid="form"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="footer-link"]').exists()).toBe(true)
    expect(wrapper.text()).toContain(`© ${new Date().getFullYear()} CheapRouter. common.rightsReserved`)
    expect(appStore.fetchPublicSettings).toHaveBeenCalled()
  })
})
