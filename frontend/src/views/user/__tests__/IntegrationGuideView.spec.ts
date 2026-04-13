import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import IntegrationGuideView from '../IntegrationGuideView.vue'

const { list, getPublicSettings, showError } = vi.hoisted(() => ({
  list: vi.fn(),
  getPublicSettings: vi.fn(),
  showError: vi.fn()
}))

const messages: Record<string, string> = {
  'integrationGuide.caption': 'Multi-platform setup',
  'integrationGuide.intro': 'Intro',
  'integrationGuide.bindingNote': 'Binding note'
}

vi.mock('@/api', () => ({
  keysAPI: {
    list
  },
  authAPI: {
    getPublicSettings
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

function createApiKey(platform: 'openai' | 'anthropic', status: 'active' | 'inactive' = 'active') {
  return {
    id: platform === 'openai' ? 1 : 2,
    key: `sk-${platform}-test`,
    name: `${platform}-key`,
    status,
    group: {
      platform
    }
  }
}

describe('IntegrationGuideView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders all platforms and distributes keys by group platform', async () => {
    list.mockResolvedValue({
      items: [
        createApiKey('openai'),
        createApiKey('anthropic', 'inactive'),
        { id: 3, key: 'sk-no-group', name: 'ungrouped' }
      ],
      total: 3,
      pages: 1
    })

    getPublicSettings.mockResolvedValue({
      api_base_url: 'https://api.example.com/v1',
      custom_endpoints: [{ name: 'Backup', endpoint: 'https://backup.example.com/v1', description: '' }]
    })

    const wrapper = mount(IntegrationGuideView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          LoadingSpinner: { template: '<div class="loading-spinner" />' },
          Icon: { template: '<span />' },
          EndpointPopover: {
            props: ['apiBaseUrl', 'customEndpoints'],
            template: '<div class="endpoint-popover">{{ apiBaseUrl }}|{{ customEndpoints.length }}</div>'
          },
          IntegrationGuidePanel: {
            props: ['platform', 'apiKeys', 'baseUrl'],
            template: '<div class="guide-panel-stub">{{ platform }}|{{ apiKeys.length }}|{{ baseUrl }}</div>'
          }
        }
      }
    })

    await flushPromises()

    expect(list).toHaveBeenCalledWith(1, 100)
    expect(wrapper.find('.endpoint-popover').text()).toContain('https://api.example.com/v1|1')

    const panels = wrapper.findAll('.guide-panel-stub').map((node) => node.text())
    expect(panels).toEqual([
      'anthropic|0|https://api.example.com/v1',
      'openai|1|https://api.example.com/v1'
    ])
  })
})
