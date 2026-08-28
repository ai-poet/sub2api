import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeDownloadSection from '../HomeDownloadSection.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}))

function setPlatform(platform: string, userAgent = '') {
  Object.defineProperty(window.navigator, 'userAgentData', {
    configurable: true,
    value: { platform },
  })
  Object.defineProperty(window.navigator, 'platform', {
    configurable: true,
    value: platform,
  })
  Object.defineProperty(window.navigator, 'userAgent', {
    configurable: true,
    value: userAgent || platform,
  })
}

describe('HomeDownloadSection', () => {
  beforeEach(() => {
    setPlatform('Linux')
  })

  it('does not render download links when no platform URL is configured', () => {
    const wrapper = mount(HomeDownloadSection, {
      props: {
        windowsUrl: '',
        macosUrl: '',
      },
      global: {
        stubs: { Icon: true }
      }
    })

    expect(wrapper.findAll('[data-platform]').length).toBe(0)
  })

  it('shows both platform downloads and prioritizes Windows for Windows browsers', () => {
    setPlatform('Windows')

    const wrapper = mount(HomeDownloadSection, {
      props: {
        windowsUrl: 'https://downloads.example.com/windows.exe',
        macosUrl: 'curl -fsSL https://example.com/install.sh | bash',
      },
      global: {
        stubs: { Icon: true }
      }
    })

    const items = wrapper.findAll('[data-platform]')
    expect(items).toHaveLength(2)
    // Windows is a download link
    expect(items[0].attributes('data-platform')).toBe('windows')
    expect(items[0].element.tagName).toBe('A')
    expect(items[0].attributes('href')).toBe('https://downloads.example.com/windows.exe')
    // macOS is a command button
    expect(items[1].attributes('data-platform')).toBe('macos')
    expect(items[1].element.tagName).toBe('BUTTON')
  })

  it('shows both platform downloads and prioritizes macOS for macOS browsers', () => {
    setPlatform('macOS')

    const wrapper = mount(HomeDownloadSection, {
      props: {
        windowsUrl: 'https://downloads.example.com/windows.exe',
        macosUrl: 'curl -fsSL https://example.com/install.sh | bash',
      },
      global: {
        stubs: { Icon: true }
      }
    })

    const items = wrapper.findAll('[data-platform]')
    expect(items).toHaveLength(2)
    // macOS is a command button (preferred)
    expect(items[0].attributes('data-platform')).toBe('macos')
    expect(items[0].element.tagName).toBe('BUTTON')
    // Windows is a download link
    expect(items[1].attributes('data-platform')).toBe('windows')
    expect(items[1].element.tagName).toBe('A')
  })

  it('shows only the configured platform when one URL is present', () => {
    const wrapper = mount(HomeDownloadSection, {
      props: {
        windowsUrl: '',
        macosUrl: 'curl -fsSL https://example.com/install.sh | bash',
      },
      global: {
        stubs: { Icon: true }
      }
    })

    const items = wrapper.findAll('[data-platform]')
    expect(items).toHaveLength(1)
    expect(items[0].attributes('data-platform')).toBe('macos')
    expect(items[0].element.tagName).toBe('BUTTON')
  })
})
