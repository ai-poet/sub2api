import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import ChangelogView from '../ChangelogView.vue'
import { resetClientChangelogCache } from '@/composables/useClientChangelog'
import type { ClientChangelogEntry } from '@/api/changelog'

const getClientChangelog = vi.fn<() => Promise<ClientChangelogEntry[]>>()

vi.mock('@/api/changelog', () => ({
  getClientChangelog: () => getClientChangelog(),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ isAuthenticated: false, homePath: '/dashboard', user: null }),
  useAppStore: () => ({
    cachedPublicSettings: {
      site_name: 'CheapRouter',
      client_download_windows_url: 'https://downloads.example.com/setup.exe',
    },
    siteName: 'CheapRouter',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ currentRoute: { value: { path: '/changelog' } }, push: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('en'),
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

function mountView() {
  return mount(ChangelogView, {
    global: {
      stubs: {
        HomeHeader: true,
        HomeFooter: true,
        Icon: true,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        MarkdownRenderer: { props: ['content'], template: '<div class="md">{{ content }}</div>' },
      },
    },
  })
}

describe('ChangelogView', () => {
  beforeEach(() => {
    resetClientChangelogCache()
    getClientChangelog.mockReset()
  })

  it('renders releases synced from GitHub, newest first', async () => {
    getClientChangelog.mockResolvedValue([
      {
        version: '0.2.1',
        published_at: '2026-09-24T11:16:42Z',
        title: 'Images',
        items: ['Draw pictures', 'Prompt caching', 'Faster startup'],
      },
      { version: '0.2.0', published_at: '2026-09-23T11:46:53Z', title: '', items: ['x'.repeat(200), 'Profiles'] },
    ])

    const wrapper = mountView()
    await flushPromises()

    const entries = wrapper.findAll('[data-test="changelog-entry"]')
    expect(entries).toHaveLength(2)
    expect(entries[0].text()).toContain('v0.2.1')
    expect(entries[0].text()).toContain('— Images')
    expect(entries[0].text()).toContain('changelog.latest')
    expect(entries[0].find('time').attributes('datetime')).toBe('2026-09-24T11:16:42Z')
    expect(entries[0].findAll('.md').map((item) => item.text())).toEqual([
      'Draw pictures',
      'Prompt caching',
      'Faster startup',
    ])
    expect(entries[1].text()).not.toContain('changelog.latest')
  })

  it('uses two columns only when every item is short', async () => {
    getClientChangelog.mockResolvedValue([
      { version: '0.2.1', published_at: '', title: '', items: ['a', 'b', 'c'] },
      { version: '0.2.0', published_at: '', title: '', items: ['x'.repeat(200), 'y'] },
    ])

    const wrapper = mountView()
    await flushPromises()

    const [short, long] = wrapper.findAll('[data-test="changelog-entry"] ul')
    expect(short.classes()).toContain('sm:grid-cols-2')
    expect(short.findAll('li')).toHaveLength(4)
    expect(long.classes()).not.toContain('sm:grid-cols-2')
    expect(long.findAll('li')).toHaveLength(2)
  })

  it('shows the empty state when there are no releases', async () => {
    getClientChangelog.mockResolvedValue([])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="changelog-empty"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="changelog-entry"]').exists()).toBe(false)
  })

  it('shows the empty state when the request fails', async () => {
    getClientChangelog.mockRejectedValue(new Error('offline'))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="changelog-empty"]').exists()).toBe(true)
  })

  it('shows a loading line until the releases arrive', async () => {
    let resolve: (entries: ClientChangelogEntry[]) => void = () => {}
    getClientChangelog.mockReturnValue(new Promise((r) => (resolve = r)))

    const wrapper = mountView()
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-test="changelog-loading"]').exists()).toBe(true)

    resolve([{ version: '1.0.0', published_at: '', title: '', items: ['done'] }])
    await flushPromises()
    expect(wrapper.find('[data-test="changelog-loading"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-test="changelog-entry"]')).toHaveLength(1)
  })
})
