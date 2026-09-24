import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeHero from '../HomeHero.vue'

const translations: Record<string, string> = {
  'home.landing.hero.releaseBadge': 'v{version} is out',
  'home.landing.hero.releaseCta': 'See what changed',
  'home.landing.hero.client.titleLead': 'Every agent.',
  'home.landing.hero.client.titleAccent': 'One desktop app.',
  'home.landing.hero.client.subtitle': 'One desktop client for Claude Code, Codex and Grok.',
  'home.landing.hero.api.titleLead': 'Every top model.',
  'home.landing.hero.api.titleAccent': 'One API key.',
  'home.landing.hero.api.subtitle': 'OpenAI- and Anthropic-compatible endpoints.',
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
  'home.clientWorkflow.ariaLabel':
    'CheapRouter desktop client demo: switching routing groups with one click',
  'home.clientWorkflow.working': '工作中 · {seconds} 秒',
  'home.clientWorkflow.balanceBefore': '$999990.44',
  'home.clientWorkflow.balanceAfter': '$999990.41',
  'home.clientWorkflow.sidebar.newTask': '新建任务',
  'home.clientWorkflow.sidebar.search': '搜索',
  'home.clientWorkflow.sidebar.today': '今天',
  'home.clientWorkflow.sidebar.taskTitle': '在吗',
  'home.clientWorkflow.sidebar.project': 'amadeus-system',
  'home.clientWorkflow.sidebar.email': 'admin@cheaprouter.cc',
  'home.clientWorkflow.labels.read': '读取',
  'home.clientWorkflow.labels.list': '列出',
  'home.clientWorkflow.labels.thinking': '思考',
  'home.clientWorkflow.labels.edit': '编辑',
  'home.clientWorkflow.groupSummary': '正在执行：6 次文件读取 · 2 次文件列表 · 1 次思考 · 1 次文件修改',
  'home.clientWorkflow.rows.r1': 'main.ts',
  'home.clientWorkflow.rows.r2': 'components',
  'home.clientWorkflow.rows.r3': 'constants.ts',
  'home.clientWorkflow.rows.r4': '思考用时 1 秒',
  'home.clientWorkflow.rows.r5': 'chat.ts',
  'home.clientWorkflow.rows.r6': 'auth.ts',
  'home.clientWorkflow.rows.r7': 'ChatView.vue',
  'home.clientWorkflow.rows.r8': 'user.ts',
  'home.clientWorkflow.rows.r9': 'hooks',
  'home.clientWorkflow.rows.r10': 'index.vue',
  'home.clientWorkflow.composer.placeholder': '做什么都可以…',
  'home.clientWorkflow.composer.model': 'gpt-5.6-sol',
  'home.clientWorkflow.composer.effort': '高',
  'home.clientWorkflow.composer.access': '完全访问',
  'home.clientWorkflow.composer.build': '构建',
  'home.clientWorkflow.composer.stop': '停止',
  'home.clientWorkflow.statusBar.project': 'amadeus-system',
  'home.clientWorkflow.statusBar.local': '本地',
  'home.clientWorkflow.statusBar.branch': 'my_feature',
  'home.clientWorkflow.menu.balance': '余额 {amount}',
  'home.clientWorkflow.menu.topUp': '充值',
  'home.clientWorkflow.menu.claudeGroup': 'Claude Code 分组',
  'home.clientWorkflow.menu.claudeValue': 'Claude Sale',
  'home.clientWorkflow.menu.codexGroup': 'Codex 分组',
  'home.clientWorkflow.menu.codexValueBefore': 'Codex',
  'home.clientWorkflow.menu.codexValueAfter': 'Codex Sale',
  'home.clientWorkflow.menu.grokGroup': 'Grok 分组',
  'home.clientWorkflow.menu.grokValue': 'Grok',
  'home.clientWorkflow.menu.modelPlaza': '模型广场',
  'home.clientWorkflow.menu.usage': '使用记录',
  'home.clientWorkflow.menu.logout': '退出登录',
  'home.clientWorkflow.menu.submenu.default': '账号默认',
  'home.clientWorkflow.menu.submenu.codexName': 'Codex',
  'home.clientWorkflow.menu.submenu.codexMeta': '×0.29 · 96.2%',
  'home.clientWorkflow.menu.submenu.saleName': 'Codex Sale',
  'home.clientWorkflow.menu.submenu.saleMeta': '×0.19 · 86.4%',
  'home.clientWorkflow.menu.submenu.welfareName': 'Codex 福利分组',
  'home.clientWorkflow.menu.submenu.welfareMeta': '×0.09 · 69.3%',
  'home.clientWorkflow.toast': '已切换到 Codex Sale，对新启动的任务生效',
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
    expect(wrapper.text()).toContain('Every agent.')
    expect(wrapper.text()).toContain('One desktop app.')
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
    expect(wrapper.text()).toContain('Every top model.')
    expect(wrapper.text()).not.toContain('One desktop app.')

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
    expect(providers.findAll('li')).toHaveLength(3)
    expect(providers.text()).toContain('Claude')
    expect(providers.text()).toContain('GPT')
    expect(providers.text()).toContain('Grok')
  })

  it('renders the CheapRouter desktop client mockup with transcript, composer and status bar', () => {
    const wrapper = mountHero({ windowsUrl: 'https://downloads.example.com/windows.exe' })

    expect(wrapper.find('[data-test="agent-workflow-preview"]').exists()).toBe(true)
    expect(wrapper.find('img[src="/product.png"]').exists()).toBe(false)
    // sidebar
    expect(wrapper.text()).toContain('新建任务')
    expect(wrapper.text()).toContain('在吗')
    expect(wrapper.text()).toContain('amadeus-system')
    expect(wrapper.text()).toContain('admin@cheaprouter.cc')
    // transcript tool rows
    expect(wrapper.text()).toContain('读取')
    expect(wrapper.text()).toContain('main.ts')
    expect(wrapper.text()).toContain('正在执行：6 次文件读取 · 2 次文件列表 · 1 次思考 · 1 次文件修改')
    expect(wrapper.text()).toContain('列出')
    expect(wrapper.text()).toContain('components')
    expect(wrapper.text()).toContain('思考用时 1 秒')
    expect(wrapper.text()).toContain('编辑')
    expect(wrapper.text()).toContain('ChatView.vue')
    expect(wrapper.text()).toContain('+12')
    expect(wrapper.text()).toContain('-3')
    // composer
    expect(wrapper.text()).toContain('做什么都可以…')
    expect(wrapper.text()).toContain('gpt-5.6-sol')
    expect(wrapper.text()).toContain('完全访问')
    expect(wrapper.text()).toContain('构建')
    // status bar
    expect(wrapper.text()).toContain('本地')
    expect(wrapper.text()).toContain('my_feature')
    expect(wrapper.text()).toContain('$999990.4')
    // old mockup is gone
    expect(wrapper.text()).not.toContain('Daemon connected')
    expect(wrapper.text()).not.toContain('Opus 4.8 1M')
    expect(wrapper.text()).not.toContain('homepage-billing')
  })

  it('shows the group-switch account menu with rate multipliers and uptime', () => {
    const wrapper = mountHero({ windowsUrl: 'https://downloads.example.com/windows.exe' })

    expect(wrapper.find('[data-test="preview-account-menu"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="preview-group-submenu"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('充值')
    expect(wrapper.text()).toContain('Claude Code 分组')
    expect(wrapper.text()).toContain('Codex 分组')
    expect(wrapper.text()).toContain('Grok 分组')
    expect(wrapper.text()).toContain('账号默认')
    expect(wrapper.text()).toContain('×0.29 · 96.2%')
    expect(wrapper.text()).toContain('×0.19 · 86.4%')
    expect(wrapper.text()).toContain('×0.09 · 69.3%')
    expect(wrapper.text()).toContain('模型广场')
    expect(wrapper.text()).toContain('使用记录')
    expect(wrapper.text()).toContain('退出登录')
    expect(wrapper.text()).toContain('已切换到 Codex Sale，对新启动的任务生效')
  })

})
