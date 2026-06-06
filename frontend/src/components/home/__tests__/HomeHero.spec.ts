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
  'home.clientShowcase.title': 'An agent client that brings Claude Code and Codex together. Download-ready in China',
  'home.clientShowcase.description':
    'Run multiple agent tasks in parallel across workspaces without switching windows.',
  'home.clientShowcase.pills.darkMode': 'Dark theme',
  'home.clientShowcase.pills.workspace': 'Workspace management',
  'home.clientShowcase.pills.terminal': 'Built-in terminal',
  'home.clientShowcase.pills.parallel': 'Parallel agents',
  'home.clientShowcase.downloadCta': 'Download {platform}',
  'home.clientWorkflow.ariaLabel': 'Animated agent workflow',
  'home.clientWorkflow.windowTitle': 'CheapRouter Client',
  'home.clientWorkflow.connected': 'Daemon connected',
  'home.clientWorkflow.sidebarSubtitle': 'Local agent console',
  'home.clientWorkflow.workspaces': 'Workspaces',
  'home.clientWorkflow.workspaceName': 'New workspace',
  'home.clientWorkflow.workspaceDocs': 'Client style source',
  'home.clientWorkflow.newWorktree': 'New WorkTree',
  'home.clientWorkflow.creatingWorktree': 'Creating',
  'home.clientWorkflow.agents': 'Agents',
  'home.clientWorkflow.draftAgent': 'Draft agent',
  'home.clientWorkflow.runningAgent': 'Agent running',
  'home.clientWorkflow.branch': 'main',
  'home.clientWorkflow.balance': 'Balance ¥128.40',
  'home.clientWorkflow.terminal': 'Terminal',
  'home.clientWorkflow.tabDraft': 'New agent',
  'home.clientWorkflow.tabAgent': 'Agent workflow preview',
  'home.clientWorkflow.emptyTitle': 'What do you want to build in {project}?',
  'home.clientWorkflow.emptyCopy':
    'Your code stays local while Claude Code and Codex runs are visible in one desktop app.',
  'home.clientWorkflow.composerTitle': 'What should the agent build?',
  'home.clientWorkflow.provider': 'Claude',
  'home.clientWorkflow.group': 'Claude',
  'home.clientWorkflow.model': 'Opus 4.8 1M',
  'home.clientWorkflow.mode': 'Bypass',
  'home.clientWorkflow.thinking': 'Medium',
  'home.clientWorkflow.prompt': 'Create a workspace, wire the billing dashboard, and verify the Claude agent flow.',
  'home.clientWorkflow.runningStatus': 'Agent is streaming with Claude / Opus 4.8 1M...',
  'home.clientWorkflow.workingRead': 'Reading workspace files',
  'home.clientWorkflow.workingEdit': 'Editing dashboard and store',
  'home.clientWorkflow.workingVerify': 'Running verification',
  'home.clientWorkflow.streamThinking':
    'Reading the workspace files and planning the billing dashboard changes...',
  'home.clientWorkflow.requestTitle': 'Agent request',
  'home.clientWorkflow.requestBody':
    'Allow the agent to inspect local files and run the verification command.',
  'home.clientWorkflow.permissionQuestion': 'Allow this action?',
  'home.clientWorkflow.requestDeny': 'Deny',
  'home.clientWorkflow.requestApproved': 'Allow',
  'home.clientWorkflow.toolInspect': 'Read workspace files',
  'home.clientWorkflow.toolEdit': 'Apply dashboard changes',
  'home.clientWorkflow.toolTerminal': 'Run verification',
  'home.clientWorkflow.streamStepOne':
    'Updated the dashboard cards and connected the usage summary to the workspace state.',
  'home.clientWorkflow.streamStepTwo':
    'Verified the agent request, file changes, and terminal status inside the desktop workflow.',
  'home.clientWorkflow.streamComplete': 'Done: Claude agent workflow completed',
  'home.clientWorkflow.replay': 'Replay',
  'home.clientWorkflow.filesChanged': 'Files changed',
  'home.clientWorkflow.terminalRun': 'Checks',
  'home.clientWorkflow.typecheckDone': 'typecheck passed',
  'home.clientWorkflow.buildDone': 'production build ready',
  'home.clientWorkflow.spendTitle': 'Transparent spend',
  'home.clientWorkflow.spendInput': 'Input tokens',
  'home.clientWorkflow.spendOutput': 'Output tokens',
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

  it('shows only the preferred desktop client download in the hero CTA row', () => {
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
    expect(platformDownloads).toHaveLength(0)
    expect(wrapper.text()).not.toContain('Download macOS')
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

    expect(wrapper.text()).toContain('An agent client that brings Claude Code and Codex together')
    expect(wrapper.text()).toContain('Run multiple agent tasks in parallel')
    expect(wrapper.text()).toContain('Parallel agents')
    expect(wrapper.text()).not.toContain('Cross-device sync')
  })

  it('renders the animated agent workflow preview instead of the static product image', () => {
    const wrapper = mountHero()

    expect(wrapper.find('[data-test="agent-workflow-preview"]').exists()).toBe(true)
    expect(wrapper.find('img[src="/product.png"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('What do you want to build in')
    expect(wrapper.text()).not.toContain('What do you want to build in sub2api?')
    expect(wrapper.text()).toContain('Claude')
    expect(wrapper.text()).toContain('Opus 4.8 1M')
    expect(wrapper.text()).toContain('Bypass')
    expect(wrapper.text()).toContain('Medium')
    expect(wrapper.text()).toContain('homepage-billing')
    expect(wrapper.text()).toContain('pricing-copy')
    expect(wrapper.text()).toContain('api-metering')
    expect(wrapper.text()).toContain('new-agent-flow')
    expect(wrapper.text()).toContain('Creating')
    expect(wrapper.text()).toContain('Reviewer')
    expect(wrapper.text()).toContain('Message the agent, tag @files')
    expect(wrapper.text()).toContain('Changes')
    // 流式完成回执不再渲染（去掉 "Done" 完成行）
    expect(wrapper.text()).not.toContain('Done: Claude agent workflow completed')
    // Bypass 模式下不再展示权限确认卡片
    expect(wrapper.text()).not.toContain('Agent request')
    expect(wrapper.text()).not.toContain('Allow this action?')
    expect(wrapper.text()).not.toContain('Client preview')
    expect(wrapper.text()).not.toContain('客户端界面预览')
  })

  it('lets a viewer click any selectable workspace and replay the stream', async () => {
    const wrapper = mountHero()
    const target = wrapper.find('[data-test="workspace-row-pricing-copy"]')
    expect(target.exists()).toBe(true)
    await target.trigger('click')
    expect(target.attributes('aria-pressed')).toBe('true')
  })
})
