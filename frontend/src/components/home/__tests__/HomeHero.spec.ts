import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeHero from '../HomeHero.vue'

// 客户端演示（HomeAgentWorkflowPreview）的细节在它自己的 spec 里测，这里只放首屏文案
const translations: Record<string, string> = {
  'home.landing.hero.releaseBadge': 'v{version} is out',
  'home.landing.hero.releaseCta': 'See what changed',
  'home.landing.hero.client.label': 'Agent desktop client',
  'home.landing.hero.api.label': 'A quality-checked AI relay',
  'home.landing.hero.tagline': 'Smart models, checked nonstop.',
  'home.landing.hero.client.title': 'Every agent, one workspace, every API.',
  'home.landing.hero.client.subtitle': 'The built-in agent works out of the box.',
  'home.landing.hero.api.title': 'One API key for all.',
  'home.landing.hero.api.subtitle': 'GPT routes get Sol and Astra fingerprint checks.',
  'home.landing.hero.downloadFor': 'Download for {platform}',
  'home.landing.hero.copyInstall': 'Copy the {platform} install command',
  'home.landing.hero.switchPlatform': 'Get the {platform} version',
  'home.landing.hero.useApi': 'Use the API directly',
  'home.landing.hero.startApi': 'Start with the API',
  'home.landing.hero.viewDocs': 'Read the docs',
  'home.landing.hero.installHint': 'Paste it into a terminal to install',
  'home.landing.hero.terminal.title': 'Terminal',
  'home.landing.hero.terminal.claudeComment': '# Claude Code',
  'home.landing.hero.terminal.openaiComment': '# Codex · OpenAI SDK',
  'home.landing.hero.terminal.caption': 'Swap the base URL and key.',
  'home.hero.installPrimary': 'Copy install command',
  'home.download.commandCopied': 'Install command copied',
  'home.login': 'Login',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      te: (key: string) => key in translations,
      t: (key: string, params?: Record<string, string | number>) => {
        const message = translations[key] || key
        return Object.entries(params || {}).reduce(
          (result, [name, value]) => result.replace(`{${name}}`, String(value)),
          message,
        )
      },
    }),
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

function mountHero(props: Partial<InstanceType<typeof HomeHero>['$props']> = {}) {
  return mount(HomeHero, {
    props: {
      siteName: 'CheapRouter',
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
        PlatformIcon: true,
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

  const WINDOWS_URL = 'https://downloads.example.com/windows.exe'
  const MAC_COMMAND = 'curl -fsSL https://example.com/install.sh | bash'

  it('leads with the Windows download and offers macOS beside it', () => {
    setPlatform('Windows')
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL, macosUrl: MAC_COMMAND })

    const download = wrapper.find('[data-test="hero-primary-download"]')
    expect(download.element.tagName).toBe('A')
    expect(download.attributes('href')).toBe(WINDOWS_URL)
    expect(download.attributes('data-platform')).toBe('windows')
    expect(download.text()).toContain('Download for Windows')

    const switches = wrapper.findAll('[data-test="hero-platform-switch"]')
    expect(switches).toHaveLength(1)
    expect(switches[0].attributes('data-platform')).toBe('macos')
    expect(wrapper.find('[data-test="hero-install-command"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="hero-primary-fallback"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Smart models, checked nonstop.')
    expect(wrapper.text()).toContain('Every agent, one workspace, every API.')
  })

  it('leads with the install command on macOS', () => {
    setPlatform('macOS')
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL, macosUrl: MAC_COMMAND })

    const install = wrapper.find('[data-test="hero-primary-download"]')
    expect(install.element.tagName).toBe('BUTTON')
    expect(install.attributes('data-platform')).toBe('macos')
    expect(install.text()).toContain('Copy the macOS install command')
    expect(wrapper.find('[data-test="hero-install-command"]').text()).toContain(MAC_COMMAND)
    expect(wrapper.find('[data-test="hero-platform-switch"]').attributes('data-platform')).toBe('windows')
  })

  it('switches platform from the side segment', async () => {
    setPlatform('Windows')
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL, macosUrl: MAC_COMMAND })

    await wrapper.find('[data-test="hero-platform-switch"]').trigger('click')

    expect(wrapper.find('[data-test="hero-primary-download"]').attributes('data-platform')).toBe('macos')
    expect(wrapper.find('[data-test="hero-install-command"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="hero-platform-switch"]').attributes('data-platform')).toBe('windows')
  })

  it('shows no switch when only one platform is configured', () => {
    setPlatform('macOS')
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL })

    expect(wrapper.find('[data-test="hero-primary-download"]').attributes('data-platform')).toBe('windows')
    expect(wrapper.find('[data-test="hero-platform-switch"]').exists()).toBe(false)
  })

  it('links the secondary API path to the dashboard when a client is available', () => {
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL })

    const connect = wrapper.find('[data-test="hero-connect-api"]')
    expect(connect.attributes('href')).toBe('/dashboard')
    expect(connect.text()).toContain('Use the API directly')
    expect(wrapper.find('[data-test="client-showcase"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-terminal"]').exists()).toBe(false)
  })

  it('keeps the gateway description beside the client one when a client is available', () => {
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL })

    const gateway = wrapper.find('[data-test="hero-description-api"]')
    expect(gateway.text()).toContain('A quality-checked AI relay')
    expect(gateway.text()).toContain('GPT routes get Sol and Astra fingerprint checks.')
    const client = wrapper.find('[data-test="hero-description-client"]')
    expect(client.text()).toContain('Agent desktop client')
    expect(client.text()).toContain('The built-in agent works out of the box.')

    const apiOnly = mountHero()
    expect(apiOnly.find('[data-test="hero-descriptions"]').exists()).toBe(false)
    expect(apiOnly.text()).toContain('GPT routes get Sol and Astra fingerprint checks.')
    expect(apiOnly.text()).not.toContain('The built-in agent works out of the box.')
  })

  it('turns into an API landing page when no client download is configured', () => {
    const wrapper = mountHero({ apiBaseUrl: 'https://api.example.com/' })

    const fallback = wrapper.find('[data-test="hero-primary-fallback"]')
    expect(fallback.attributes('href')).toBe('/login')
    expect(fallback.text()).toContain('Start with the API')
    expect(wrapper.find('[data-test="hero-secondary-login"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="hero-primary-download"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="hero-connect-api"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="client-showcase"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="agent-workflow-preview"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Smart models, checked nonstop.')
    expect(wrapper.text()).toContain('One API key for all.')
    expect(wrapper.text()).not.toContain('Every agent, one workspace, every API.')

    const terminal = wrapper.find('[data-test="api-terminal"]')
    expect(terminal.text()).toContain('ANTHROPIC_BASE_URL="https://api.example.com"')
    expect(terminal.text()).toContain('OPENAI_BASE_URL="https://api.example.com/v1"')
  })

  it('offers the docs instead of login when a doc url is configured', () => {
    const wrapper = mountHero({ docUrl: 'https://docs.example.com', isAuthenticated: true })

    expect(wrapper.find('[data-test="hero-primary-fallback"]').attributes('href')).toBe('/dashboard')
    expect(wrapper.find('[data-test="hero-secondary-docs"]').attributes('href')).toBe('https://docs.example.com')
    expect(wrapper.find('[data-test="hero-secondary-login"]').exists()).toBe(false)
  })

  it('shows the latest client release only when a client is available', () => {
    const withClient = mountHero({ windowsUrl: WINDOWS_URL, latestVersion: '0.2.1' })
    const badge = withClient.find('[data-test="hero-release-badge"]')
    expect(badge.attributes('href')).toBe('/changelog')
    expect(badge.text()).toContain('v0.2.1 is out')

    expect(mountHero({ windowsUrl: WINDOWS_URL }).find('[data-test="hero-release-badge"]').exists()).toBe(false)
    expect(mountHero({ latestVersion: '0.2.1' }).find('[data-test="hero-release-badge"]').exists()).toBe(false)
  })

  it('lists the providers behind the gateway', () => {
    const wrapper = mountHero()

    const providers = wrapper.find('[data-test="hero-providers"]')
    expect(providers.findAll('li')).toHaveLength(6)
    expect(providers.text()).toContain('Claude')
    expect(providers.text()).toContain('GPT')
    expect(providers.text()).toContain('Grok')
    expect(providers.text()).toContain('DeepSeek')
    expect(providers.text()).toContain('GLM')
    expect(providers.text()).toContain('Kimi')
  })

  it('keeps each title phrase whole and puts the quality line under the title', () => {
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL })

    const phrases = wrapper.find('[data-test="hero-title"]').findAll('span').map((span) => span.text())
    expect(phrases).toEqual(['Every agent,', 'one workspace,', 'every API.'])
    expect(wrapper.find('[data-test="hero-tagline"]').text()).toBe('Smart models, checked nonstop.')
  })

  it('shows the client window mockup when a client is available', () => {
    const wrapper = mountHero({ windowsUrl: WINDOWS_URL })

    expect(wrapper.find('[data-test="agent-workflow-preview"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="preview-agent-scene"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="preview-image-studio"]').exists()).toBe(true)
    expect(wrapper.find('img[src="/product.png"]').exists()).toBe(false)
  })
})
