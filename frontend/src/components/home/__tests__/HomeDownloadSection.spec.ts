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

    expect(wrapper.findAll('a[data-platform]').length).toBe(0)
  })

  it('shows both platform downloads and prioritizes Windows for Windows browsers', () => {
    setPlatform('Windows')

    const wrapper = mount(HomeDownloadSection, {
      props: {
        windowsUrl: 'https://downloads.example.com/windows.exe',
        macosUrl: 'https://downloads.example.com/macos.dmg',
      },
      global: {
        stubs: { Icon: true }
      }
    })

    const links = wrapper.findAll('a[data-platform]')
    expect(links).toHaveLength(2)
    expect(links[0].attributes('data-platform')).toBe('windows')
    expect(links[0].attributes('href')).toBe('https://downloads.example.com/windows.exe')
    expect(links[1].attributes('data-platform')).toBe('macos')
  })

  it('shows both platform downloads and prioritizes macOS for macOS browsers', () => {
    setPlatform('macOS')

    const wrapper = mount(HomeDownloadSection, {
      props: {
        windowsUrl: 'https://downloads.example.com/windows.exe',
        macosUrl: 'https://downloads.example.com/macos.dmg',
      },
      global: {
        stubs: { Icon: true }
      }
    })

    const links = wrapper.findAll('a[data-platform]')
    expect(links).toHaveLength(2)
    expect(links[0].attributes('data-platform')).toBe('macos')
    expect(links[0].attributes('href')).toBe('https://downloads.example.com/macos.dmg')
    expect(links[1].attributes('data-platform')).toBe('windows')
  })

  it('shows only the configured platform when one URL is present', () => {
    const wrapper = mount(HomeDownloadSection, {
      props: {
        windowsUrl: '',
        macosUrl: 'https://downloads.example.com/macos.dmg',
      },
      global: {
        stubs: { Icon: true }
      }
    })

    const links = wrapper.findAll('a[data-platform]')
    expect(links).toHaveLength(1)
    expect(links[0].attributes('data-platform')).toBe('macos')
  })
})
