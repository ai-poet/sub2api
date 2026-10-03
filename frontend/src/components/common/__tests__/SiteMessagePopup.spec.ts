import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import SiteMessagePopup from '../SiteMessagePopup.vue'

const state = vi.hoisted(() => ({
  path: '/dashboard',
  acknowledgePopup: vi.fn(),
  openInboxFromPopup: vi.fn(),
  markAllRead: vi.fn(),
  showError: vi.fn()
}))

const siteStore = reactive({
  currentPopup: null as Record<string, unknown> | null,
  popupQueue: [] as Record<string, unknown>[],
  acknowledgePopup: state.acknowledgePopup,
  openInboxFromPopup: state.openInboxFromPopup,
  markAllRead: state.markAllRead
})
const announcementStore = reactive({ currentPopup: null as Record<string, unknown> | null })

vi.mock('@/stores/siteMessages', () => ({ useSiteMessagesStore: () => siteStore }))
vi.mock('@/stores/announcements', () => ({ useAnnouncementStore: () => announcementStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: state.showError }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ path: state.path }) }))
vi.mock('@/utils/format', () => ({ formatRelativeWithDateTime: (v: string) => v }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key) })
}))

enableAutoUnmount(afterEach)

function mountPopup() {
  return mount(SiteMessagePopup, {
    global: {
      stubs: {
        Teleport: true,
        Transition: false,
        MarkdownRenderer: { props: ['content'], template: '<div class="md">{{ content }}</div>' },
        Icon: true
      }
    }
  })
}

const message = { id: 1, category: 'security', title: '账户风控提醒', content: '正文', from: 'system', read_at: null, created_at: '2026-10-01T00:00:00Z' }

beforeEach(() => {
  vi.clearAllMocks()
  state.path = '/dashboard'
  siteStore.currentPopup = { ...message }
  siteStore.popupQueue = []
  announcementStore.currentPopup = null
})

describe('SiteMessagePopup (fork)', () => {
  it('shows the current message and acknowledges it', async () => {
    const w = mountPopup()
    expect(w.text()).toContain('账户风控提醒')
    expect(w.text()).toContain('siteMessages.categories.security')
    await w.get('[data-testid="site-message-popup-ack"]').trigger('click')
    expect(state.acknowledgePopup).toHaveBeenCalledTimes(1)
  })

  it('shows how many more are queued and offers mark-all', async () => {
    siteStore.popupQueue = [{ ...message, id: 2 }, { ...message, id: 3 }]
    const w = mountPopup()
    expect(w.text()).toContain('siteMessages.popup.more:{"count":2}')
    await w.get('[data-testid="site-message-popup-read-all"]').trigger('click')
    expect(state.markAllRead).toHaveBeenCalledTimes(1)
  })

  it('waits for an announcement popup to close first', () => {
    announcementStore.currentPopup = { id: 99 }
    const w = mountPopup()
    expect(w.find('[data-testid="site-message-popup"]').exists()).toBe(false)
  })

  it('never shows on the appeal page', () => {
    state.path = '/appeal'
    const w = mountPopup()
    expect(w.find('[data-testid="site-message-popup"]').exists()).toBe(false)
  })
})
