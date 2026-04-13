import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const copyToClipboard = vi.fn().mockResolvedValue(true)

const messages: Record<string, string> = {
  'integrationGuide.exampleBadge': 'Example Mode',
  'integrationGuide.liveBadge': 'Live Config',
  'integrationGuide.keyLabel': 'Current API Key',
  'integrationGuide.selectKey': 'Select an API key for this platform',
  'integrationGuide.keyHelp': 'Real key injected',
  'integrationGuide.noKeysForPlatform': 'No API key is currently available for this platform.',
  'integrationGuide.exampleOption': 'Show example config',
  'integrationGuide.exampleDescription': 'Example snippet mode',
  'integrationGuide.platforms.openai': 'OpenAI',
  'integrationGuide.platforms.anthropic': 'Anthropic',
  'keys.useKeyModal.description': 'Claude Code environment variables',
  'keys.useKeyModal.note': 'Claude note',
  'keys.useKeyModal.openai.description': 'Codex CLI config directory',
  'keys.useKeyModal.openai.note': 'OpenAI note',
  'keys.useKeyModal.openai.noteWindows': 'OpenAI Windows note',
  'keys.useKeyModal.openai.configTomlHint': 'config.toml should stay at the top',
  'keys.useKeyModal.opencode.hint': 'OpenCode config path',
  'keys.useKeyModal.cliTabs.claudeCode': 'Claude Code',
  'keys.useKeyModal.cliTabs.codexCli': 'Codex CLI',
  'keys.useKeyModal.cliTabs.codexCliWs': 'Codex CLI (WebSocket)',
  'keys.useKeyModal.cliTabs.opencode': 'OpenCode',
  'keys.useKeyModal.copy': 'Copy',
  'keys.useKeyModal.copied': 'Copied',
  'keys.copied': 'Copied!',
  'keys.status.inactive': 'Inactive'
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => messages[key] ?? key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

import IntegrationGuidePanel from '../IntegrationGuidePanel.vue'

function createApiKey(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    user_id: 1,
    key: 'sk-live-example-1234567890',
    name: 'Demo Key',
    group_id: 11,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: null,
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '2026-04-01T00:00:00Z',
    updated_at: '2026-04-01T00:00:00Z',
    group: {
      id: 11,
      name: 'OpenAI Group',
      description: null,
      platform: 'openai',
      rate_multiplier: 1,
      is_exclusive: false,
      status: 'active',
      subscription_type: 'standard',
      daily_limit_usd: null,
      weekly_limit_usd: null,
      monthly_limit_usd: null,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      allow_messages_dispatch: false,
      require_oauth_only: false,
      require_privacy_set: false,
      created_at: '2026-04-01T00:00:00Z',
      updated_at: '2026-04-01T00:00:00Z'
    },
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    window_5h_start: null,
    window_1d_start: null,
    window_7d_start: null,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
    ...overrides
  }
}

describe('IntegrationGuidePanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows example mode placeholders when no key exists for the platform', () => {
    const wrapper = mount(IntegrationGuidePanel, {
      props: {
        platform: 'anthropic',
        apiKeys: [],
        baseUrl: 'https://example.com/v1'
      },
      global: {
        stubs: {
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' }
        }
      }
    })

    expect(wrapper.text()).toContain('Example Mode')
    expect(wrapper.text()).toContain('No API key is currently available for this platform.')
    expect(wrapper.find('pre').text()).toContain('YOUR_API_KEY')
  })

  it('toggles Claude Code availability based on the selected OpenAI key', async () => {
    const liveKey = createApiKey({
      id: 1,
      group: {
        ...(createApiKey().group as Record<string, unknown>),
        allow_messages_dispatch: true
      }
    })
    const standardKey = createApiKey({
      id: 2,
      key: 'sk-plain-example-abcdef123456',
      name: 'Plain Key',
      group: {
        ...(createApiKey().group as Record<string, unknown>),
        allow_messages_dispatch: false
      }
    })

    const wrapper = mount(IntegrationGuidePanel, {
      props: {
        platform: 'openai',
        apiKeys: [liveKey as any, standardKey as any],
        baseUrl: 'https://example.com/v1'
      },
      global: {
        stubs: {
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' }
        }
      }
    })

    expect(wrapper.text()).toContain('Claude Code')
    expect(wrapper.findAll('pre code')[1].text()).toContain('sk-live-example-1234567890')

    await wrapper.find('select').setValue('2')
    await nextTick()

    expect(wrapper.text()).not.toContain('Claude Code')
    expect(wrapper.findAll('pre code')[1].text()).toContain('sk-plain-example-abcdef123456')
  })

  it('renders updated GPT-5.4 mini and nano names in OpenCode config', async () => {
    const wrapper = mount(IntegrationGuidePanel, {
      props: {
        platform: 'openai',
        apiKeys: [createApiKey() as any],
        baseUrl: 'https://example.com/v1'
      },
      global: {
        stubs: {
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' }
        }
      }
    })

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('OpenCode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.text()).toContain('"name": "GPT-5.4 Mini"')
    expect(codeBlock.text()).toContain('"name": "GPT-5.4 Nano"')
  })

  it('updates snippets when switching client and system tabs', async () => {
    const wrapper = mount(IntegrationGuidePanel, {
      props: {
        platform: 'openai',
        apiKeys: [createApiKey() as any],
        baseUrl: 'https://example.com/v1'
      },
      global: {
        stubs: {
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' }
        }
      }
    })

    expect(wrapper.text()).toContain('~/.codex/config.toml')

    const windowsTab = wrapper.findAll('button').find((button) =>
      button.text().includes('Windows')
    )
    expect(windowsTab).toBeDefined()
    await windowsTab!.trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('%userprofile%\\.codex/config.toml')

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('OpenCode')
    )
    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.text()).toContain('"baseURL": "https://example.com/v1"')
    expect(wrapper.text()).toContain('opencode.json')
  })

  it('copies the currently visible snippet', async () => {
    const wrapper = mount(IntegrationGuidePanel, {
      props: {
        platform: 'anthropic',
        apiKeys: [createApiKey({
          group: {
            ...(createApiKey().group as Record<string, unknown>),
            platform: 'anthropic'
          }
        }) as any],
        baseUrl: 'https://example.com/v1'
      },
      global: {
        stubs: {
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' }
        }
      }
    })

    const copyButton = wrapper.findAll('button').find((button) => button.text().includes('Copy'))
    expect(copyButton).toBeDefined()

    await copyButton!.trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenCalledWith(
      `export ANTHROPIC_BASE_URL="https://example.com/v1"
export ANTHROPIC_AUTH_TOKEN="sk-live-example-1234567890"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`,
      'Copied!'
    )
  })
})
