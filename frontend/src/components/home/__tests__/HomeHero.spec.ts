import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeHero from '../HomeHero.vue'

const translations: Record<string, string> = {
  'home.hero.tags.coding': 'Coding',
  'home.hero.tags.agent': 'Agent',
  'home.hero.tags.tools': 'Tools',
  'home.hero.titleLeadPrimary': 'Claude Code',
  'home.hero.titleLeadSecondary': 'and Codex',
  'home.hero.titleAccent': 'in one gateway',
  'home.hero.titleTail': 'with metered billing',
  'home.hero.primaryNote': 'Use one key everywhere.',
  'home.hero.downloadPrimary': 'Download now',
  'home.cta.button': 'Start',
  'home.goToDashboard': 'Dashboard',
  'home.viewDocs': 'Docs',
  'home.login': 'Login',
  'home.clientShowcase.title': 'One desktop app for Claude Code and Codex',
  'home.clientShowcase.description':
    'Run multiple agent tasks in parallel across workspaces without switching windows.',
  'home.clientShowcase.pills.darkMode': 'Dark theme',
  'home.clientShowcase.pills.workspace': 'Workspace management',
  'home.clientShowcase.pills.terminal': 'Built-in terminal',
  'home.clientShowcase.pills.parallel': 'Parallel agents',
  'home.clientShowcase.caption': 'Client preview',
  'home.clientShowcase.downloadCta': 'Download {platform}',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => {
        const message = translations[key] || key
        return Object.entries(params || {}).reduce(
          (result, [name, value]) => result.replace(`{${name}}`, value),
          message,
        )
      },
    }),
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

function mountHero(props: Partial<InstanceType<typeof HomeHero>['$props']> = {}) {
  return mount(HomeHero, {
    props: {
      siteSubtitle: '',
      docUrl: '',
      isAuthenticated: false,
      dashboardPath: '/dashboard',
      windowsUrl: '',
      macosUrl: '',
      ...props,
    },
    global: {
      stubs: {
        Icon: true,
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

describe('HomeHero', () => {
  beforeEach(() => {
    setPlatform('Linux')
  })

  it('moves desktop download links into the primary hero CTA row', () => {
    setPlatform('Windows')

    const wrapper = mountHero({
      windowsUrl: 'https://downloads.example.com/windows.exe',
      macosUrl: 'https://downloads.example.com/macos.dmg',
    })

    const downloadLink = wrapper.find('[data-test="hero-primary-download"]')
    expect(downloadLink.exists()).toBe(true)
    expect(downloadLink.attributes('href')).toBe('https://downloads.example.com/windows.exe')
    expect(downloadLink.attributes('data-platform')).toBe('windows')
    expect(downloadLink.text()).toContain('Download now')
    const platformDownloads = wrapper.findAll('[data-test="hero-platform-download"]')
    expect(platformDownloads).toHaveLength(1)
    expect(platformDownloads[0].attributes('href')).toBe('https://downloads.example.com/macos.dmg')
    expect(platformDownloads[0].text()).toContain('Download macOS')
    expect(wrapper.find('[data-test="hero-primary-fallback"]').exists()).toBe(false)
  })

  it('falls back to the registration CTA when no client download is configured', () => {
    const wrapper = mountHero()

    expect(wrapper.find('[data-test="hero-primary-fallback"]').attributes('href')).toBe('/login')
    expect(wrapper.find('[data-test="hero-primary-download"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="hero-platform-download"]').exists()).toBe(false)
  })

  it('highlights parallel agent runs in the client preview copy and pills', () => {
    const wrapper = mountHero()

    expect(wrapper.text()).toContain('Run multiple agent tasks in parallel')
    expect(wrapper.text()).toContain('Parallel agents')
    expect(wrapper.text()).not.toContain('Cross-device sync')
  })
})
