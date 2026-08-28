import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeHero from '../HomeHero.vue'

const translations: Record<string, string> = {
  'home.hero.tags.coding': 'Claude Code',
  'home.hero.tags.agent': 'Codex',
  'home.hero.tags.tools': 'Grok · Pi · More',
  'home.hero.tags.more': 'More',
  'home.hero.titleLeadPrimary': 'Price-competitive AI relay',
  'home.hero.titleLeadSecondary': '× Agent desktop client',
  'home.hero.titleAccent': '',
  'home.hero.titleTail': '',
  'home.hero.primaryNote': 'Use one key everywhere.',
  'home.hero.downloadPrimary': 'Download now',
  'home.hero.installPrimary': 'Install now',
  'home.download.commandCopied': 'Install command copied',
  'home.hero.connectApi': 'Use the API',
  'home.hero.startApi': 'Start with the API',
  'home.goToDashboard': 'Dashboard',
  'home.viewDocs': 'Docs',
  'home.login': 'Login',
  'home.clientShowcase.title': 'Relay gateway × agent workbench, deeply unified in one desktop client',
  'home.clientShowcase.description':
    'Sign in and you are routed — balance, group rates, and uptime sit right next to your tasks.',
  'home.clientShowcase.pills.autoRoute': 'Sign in, routed',
  'home.clientShowcase.pills.groupSwitch': 'One-click group switch (rate · uptime)',
  'home.clientShowcase.pills.liveBalance': 'Live balance',
  'home.clientShowcase.pills.cliInstall': 'Missing CLI? One-click install',
  'home.clientShowcase.pills.aggregate': 'Mainstream AI tools in one place',
  'home.clientShowcase.advantages.tiny.title': 'An installer of just a dozen MB',
  'home.clientShowcase.advantages.tiny.body': 'A single native binary — no Electron shell',
  'home.clientShowcase.advantages.native.title': 'Native-grade smoothness',
  'home.clientShowcase.advantages.native.body': 'Rust + GPUI rendering — zero lag on scroll and input',
  'home.clientShowcase.advantages.ready.title': 'Works out of the box',
  'home.clientShowcase.advantages.ready.body':
    'Sign in and you are routed; missing Node or a CLI is fixed in one click',
  'home.clientShowcase.apiOnly.title': 'Just want the raw API?',
  'home.clientShowcase.apiOnly.body':
    'Register for a key and call the OpenAI/Anthropic-compatible endpoints directly.',
  'home.clientShowcase.apiOnly.dashboardCta': 'Get an API key',
  'home.clientShowcase.apiOnly.docsCta': 'Read the docs',
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

  it('shows only the preferred desktop client download in the hero CTA row', () => {
    setPlatform('Windows')

    const wrapper = mountHero({
      windowsUrl: 'https://downloads.example.com/windows.exe',
      macosUrl: 'curl -fsSL https://example.com/install.sh | bash',
    })

    const downloadLink = wrapper.find('[data-test="hero-primary-download"]')
    expect(downloadLink.exists()).toBe(true)
    expect(downloadLink.attributes('href')).toBe('https://downloads.example.com/windows.exe')
    expect(downloadLink.attributes('data-platform')).toBe('windows')
    expect(downloadLink.text()).toContain('Download now')
    const platformDownloads = wrapper.findAll('[data-test="hero-platform-download"]')
    expect(platformDownloads).toHaveLength(0)
    expect(wrapper.text()).not.toContain('Download macOS')
    expect(wrapper.find('[data-test="hero-primary-fallback"]').exists()).toBe(false)
    // 配置了客户端下载地址时：标题附带 Agent 桌面客户端副标题
    expect(wrapper.text()).toContain('Price-competitive AI relay')
    expect(wrapper.text()).toContain('× Agent desktop client')
  })

  it('shows install command button when macOS is the preferred platform', () => {
    setPlatform('macOS')

    const wrapper = mountHero({
      windowsUrl: 'https://downloads.example.com/windows.exe',
      macosUrl: 'curl -fsSL https://example.com/install.sh | bash',
    })

    const installButton = wrapper.find('[data-test="hero-primary-download"]')
    expect(installButton.exists()).toBe(true)
    expect(installButton.element.tagName).toBe('BUTTON')
    expect(installButton.attributes('data-platform')).toBe('macos')
    expect(installButton.text()).toContain('Install now')
    expect(wrapper.find('[data-test="hero-primary-fallback"]').exists()).toBe(false)
  })

  it('falls back to the API CTA when no client download is configured', () => {
    const wrapper = mountHero()

    const fallback = wrapper.find('[data-test="hero-primary-fallback"]')
    expect(fallback.attributes('href')).toBe('/login')
    expect(fallback.text()).toContain('Start with the API')
    expect(wrapper.find('[data-test="hero-primary-download"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="hero-platform-download"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="hero-connect-api"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Use the API')
  })

  it('hides client-related info and the API-only card when no client download is configured', () => {
    const wrapper = mountHero()

    expect(wrapper.find('[data-test="client-showcase"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="agent-workflow-preview"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Use one key everywhere.')
    // 无客户端下载地址时：标题只保留中转站主线；CLI 图标条保留展示
    expect(wrapper.text()).toContain('Price-competitive AI relay')
    expect(wrapper.text()).not.toContain('× Agent desktop client')
    expect(wrapper.find('.border-dashed').exists()).toBe(true)
    // API-only card is hidden too
    expect(wrapper.find('[data-test="api-only-card"]').exists()).toBe(false)
  })

  it('sells the gateway + workbench integration in the showcase pills', () => {
    const wrapper = mountHero({ windowsUrl: 'https://downloads.example.com/windows.exe' })

    expect(wrapper.text()).toContain('Sign in, routed')
    expect(wrapper.text()).toContain('One-click group switch (rate · uptime)')
    expect(wrapper.text()).toContain('Live balance')
    expect(wrapper.text()).toContain('Missing CLI? One-click install')
    expect(wrapper.text()).toContain('Mainstream AI tools in one place')
    expect(wrapper.text()).not.toContain('Dark theme')
    expect(wrapper.text()).not.toContain('Parallel agents')
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

  it('renders the CLI icons strip below the headline without a Pi card', () => {
    const wrapper = mountHero({ windowsUrl: 'https://downloads.example.com/windows.exe' })

    const text = wrapper.text()
    expect(text).toContain('Claude Code')
    expect(text).toContain('Codex')
    expect(text).toContain('Grok')
    // dashed "more" pill in the icon strip
    const morePill = wrapper.find('.border-dashed')
    expect(morePill.exists()).toBe(true)
    expect(morePill.text()).toBe('+ More')
    expect(text).not.toContain('+ Grok · Pi · More')
  })

  it('renders the advantages strip and the API-only card within the showcase', () => {
    const wrapper = mountHero({
      windowsUrl: 'https://downloads.example.com/windows.exe',
      docUrl: 'https://docs.example.com',
    })

    const advantages = wrapper.find('[data-test="client-advantages"]')
    expect(advantages.exists()).toBe(true)
    expect(advantages.text()).toContain('An installer of just a dozen MB')
    expect(advantages.text()).toContain('A single native binary — no Electron shell')
    expect(advantages.text()).toContain('Native-grade smoothness')
    expect(advantages.text()).toContain('Rust + GPUI rendering — zero lag on scroll and input')
    expect(advantages.text()).toContain('Works out of the box')

    const apiOnly = wrapper.find('[data-test="api-only-card"]')
    expect(apiOnly.exists()).toBe(true)
    expect(apiOnly.text()).toContain('Just want the raw API?')
    expect(apiOnly.find('[data-test="api-only-dashboard"]').attributes('href')).toBe('/dashboard')
    const docsLink = apiOnly.find('[data-test="api-only-docs"]')
    expect(docsLink.attributes('href')).toBe('https://docs.example.com')
  })

  it('hides the API-only card when no client download is configured', () => {
    const wrapper = mountHero({ docUrl: 'https://docs.example.com' })

    expect(wrapper.find('[data-test="api-only-card"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Just want the raw API?')
  })

  it('hides the API-only docs link when no doc url is configured', () => {
    const wrapper = mountHero({ windowsUrl: 'https://downloads.example.com/windows.exe' })

    expect(wrapper.find('[data-test="api-only-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-only-docs"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="api-only-dashboard"]').attributes('href')).toBe('/dashboard')
  })
})
