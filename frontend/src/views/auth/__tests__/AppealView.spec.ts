import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'

const state = vi.hoisted(() => ({
  replace: vi.fn(),
  getSession: vi.fn(),
  logout: vi.fn(),
  listSiteMessages: vi.fn(),
  markSiteMessageRead: vi.fn(),
  listTickets: vi.fn(),
  getTicket: vi.fn(),
  createTicket: vi.fn(),
  reply: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/appeal', () => ({
  default: {
    getSession: state.getSession,
    logout: state.logout,
    listSiteMessages: state.listSiteMessages,
    markSiteMessageRead: state.markSiteMessageRead,
    listTickets: state.listTickets,
    getTicket: state.getTicket,
    createTicket: state.createTicket,
    reply: state.reply
  }
}))
vi.mock('vue-router', () => ({ useRouter: () => ({ replace: state.replace }) }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ siteName: 'Test Site', showSuccess: state.showSuccess, showError: state.showError })
}))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key) })
  }
})
vi.mock('@/utils/format', () => ({ formatDateTime: (v: string) => v }))

import AppealView from '../AppealView.vue'
import { APPEAL_LOGIN_NOTICE_KEY, readAppealSession, storeAppealSession } from '@/utils/appeal'

enableAutoUnmount(afterEach)

const TicketThreadStub = defineComponent({
  props: ['ticket', 'messages', 'viewer', 'side', 'submitting', 'loading'],
  emits: ['reply'],
  methods: { clearDraft() {} },
  template: '<div data-testid="thread" :data-side="side" :data-viewer="viewer"><button data-testid="thread-send" @click="$emit(\'reply\', \'more info\')" /></div>'
})

function mountView() {
  return mount(AppealView, {
    global: {
      stubs: {
        LocaleSwitcher: true,
        MarkdownRenderer: { props: ['content'], template: '<div class="md">{{ content }}</div>' },
        TicketThread: TicketThreadStub
      }
    }
  })
}

const sessionInfo = { email: 'banned@example.com', status: 'disabled', expires_at: '2026-10-03T12:00:00Z', has_active_appeal: false }
const notice = { id: 1, category: 'security', title: '账户已被禁用 / Account disabled', content: '原因', from: 'system', read_at: null, created_at: '2026-10-03T10:00:00Z' }

describe('AppealView (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    storeAppealSession('apl_token', 7200)
    state.getSession.mockResolvedValue(sessionInfo)
    state.listSiteMessages.mockResolvedValue({ items: [notice], total: 1 })
    state.listTickets.mockResolvedValue({ items: [], total: 0 })
    state.markSiteMessageRead.mockResolvedValue(undefined)
  })

  it('goes back to login when there is no appeal session', async () => {
    sessionStorage.clear()
    mountView()
    await flushPromises()
    expect(state.replace).toHaveBeenCalledWith('/login')
    expect(sessionStorage.getItem(APPEAL_LOGIN_NOTICE_KEY)).toBe('expired')
    expect(state.getSession).not.toHaveBeenCalled()
  })

  it('shows the ban notices and the appeal form, then submits', async () => {
    state.createTicket.mockResolvedValue({ id: 5, title: '申请恢复账户', category: 'appeal', status: 'open' })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('banned@example.com')
    const row = wrapper.get('[data-testid="appeal-notice"]')
    expect(row.text()).toContain('账户已被禁用')
    await row.get('button').trigger('click')
    expect(state.markSiteMessageRead).toHaveBeenCalledWith(1)
    expect(wrapper.find('.md').text()).toBe('原因')

    expect(wrapper.find('[data-testid="appeal-form"]').exists()).toBe(true)
    await wrapper.get('#appeal-body').setValue('我没有违规')
    state.listTickets.mockResolvedValue({ items: [{ id: 5, status: 'open', category: 'appeal', title: 't' }], total: 1 })
    state.getTicket.mockResolvedValue({ ticket: { id: 5, status: 'open', category: 'appeal', title: 't' }, messages: [] })
    await wrapper.get('[data-testid="appeal-form"]').trigger('submit')
    await flushPromises()

    expect(state.createTicket).toHaveBeenCalledWith('appeal.form.defaultTitle', '我没有违规')
    expect(state.showSuccess).toHaveBeenCalledWith('appeal.form.submitted')
    expect(wrapper.get('[data-testid="thread"]').attributes('data-side')).toBe('appeal')
  })

  it('continues an open appeal in the thread', async () => {
    state.listTickets.mockResolvedValue({ items: [{ id: 5, status: 'replied', category: 'appeal', title: 't' }], total: 1 })
    state.getTicket.mockResolvedValue({ ticket: { id: 5, status: 'replied', category: 'appeal', title: 't' }, messages: [] })
    state.reply.mockResolvedValue({ ticket: { id: 5, status: 'open' }, message: { id: 9, body: 'more info' } })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="appeal-form"]').exists()).toBe(false)
    await wrapper.get('[data-testid="thread-send"]').trigger('click')
    await flushPromises()
    expect(state.reply).toHaveBeenCalledWith(5, 'more info')
  })

  it('leaves with a restored notice once the account is active again', async () => {
    state.getSession.mockRejectedValue({ status: 409, reason: 'APPEAL_ACCOUNT_ACTIVE', message: 'restored' })
    mountView()
    await flushPromises()
    expect(state.replace).toHaveBeenCalledWith('/login')
    expect(sessionStorage.getItem(APPEAL_LOGIN_NOTICE_KEY)).toBe('restored')
    expect(readAppealSession()).toBeNull()
  })

  it('exit revokes the session and clears the token', async () => {
    state.logout.mockResolvedValue(undefined)
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="appeal-exit"]').trigger('click')
    await flushPromises()
    expect(state.logout).toHaveBeenCalled()
    expect(readAppealSession()).toBeNull()
    expect(state.replace).toHaveBeenCalledWith('/login')
  })
})
