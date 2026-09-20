import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import TicketsView from '../TicketsView.vue'

const state = vi.hoisted(() => ({
  routeQuery: {} as Record<string, string>,
  list: vi.fn(),
  get: vi.fn(),
  reply: vi.fn(),
  close: vi.fn(),
  reopen: vi.fn(),
  fetchCounts: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    tickets: {
      list: state.list,
      get: state.get,
      reply: state.reply,
      close: state.close,
      reopen: state.reopen
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: state.showSuccess, showError: state.showError, showInfo: vi.fn() })
}))

vi.mock('@/stores/tickets', () => ({
  useTicketsStore: () => ({ openCount: 2, unreadCount: 0, fetchCounts: state.fetchCounts })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: state.routeQuery })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key) })
  }
})

vi.mock('@/utils/format', () => ({ formatDateTime: (v: string) => v }))

const AppLayoutStub = { template: '<div><slot /></div>' }
const TablePageLayoutStub = {
  template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
}
const DataTableStub = defineComponent({
  props: ['columns', 'data', 'loading', 'rowKey', 'stickyRightColumns', 'clickableRows'],
  emits: ['rowClick'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" :data-test="'row-' + row.id">
        <slot name="cell-user" :row="row" :value="row.user" />
        <slot name="cell-title" :row="row" :value="row.title" />
        <slot name="cell-status" :row="row" :value="row.status" />
        <slot name="cell-actions" :row="row" :value="null" />
      </div>
      <slot v-if="!data.length" name="empty" />
    </div>
  `
})
const PaginationStub = { props: ['total', 'page', 'pageSize'], template: '<div data-test="pagination" />' }
const SelectStub = { props: ['modelValue', 'options', 'placeholder'], template: '<select data-test="category-select" />' }
const EmptyStateStub = { props: ['title', 'description'], template: '<div data-test="empty">{{ title }}</div>' }
const BaseDialogStub = defineComponent({
  props: ['show', 'title', 'width'],
  emits: ['close'],
  template: '<div v-if="show" data-test="detail-dialog"><slot /><slot name="footer" /></div>'
})
const ConfirmDialogStub = defineComponent({
  props: ['show', 'title', 'message', 'danger'],
  emits: ['confirm', 'cancel'],
  template: '<div v-if="show" data-test="confirm-dialog"><button data-test="confirm-ok" @click="$emit(\'confirm\')">ok</button></div>'
})
const TicketThreadStub = defineComponent({
  props: ['ticket', 'messages', 'viewer', 'submitting', 'loading'],
  emits: ['reply'],
  methods: {
    clearDraft() {
      /* noop */
    }
  },
  template: `
    <div data-test="thread" :data-viewer="viewer" :data-count="messages.length">
      <button data-test="thread-send" @click="$emit('reply', 'hello there')">send</button>
    </div>
  `
})

function ticketRow(id: number, extra: Record<string, unknown> = {}) {
  return {
    id,
    title: `Ticket ${id}`,
    category: 'api',
    status: 'open',
    user_unread: false,
    message_count: 1,
    last_message_at: '2026-09-20T10:00:00Z',
    created_at: '2026-09-20T10:00:00Z',
    updated_at: '2026-09-20T10:00:00Z',
    user: { id: 100 + id, email: `user${id}@example.com` },
    ...extra
  }
}

function message(id: number, role: string, body = 'body') {
  return { id, ticket_id: 1, author_role: role, author_email: role === 'user' ? undefined : 'ops@example.com', body, created_at: '2026-09-20T10:00:00Z' }
}

function mountView() {
  return mount(TicketsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: PaginationStub,
        Select: SelectStub,
        EmptyState: EmptyStateStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        TicketThread: TicketThreadStub
      }
    }
  })
}

describe('admin TicketsView (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    state.routeQuery = {}
    state.list.mockResolvedValue({ items: [ticketRow(1), ticketRow(2, { status: 'replied' })], total: 2, page: 1, page_size: 20, pages: 1 })
    state.get.mockImplementation(async (id: number) => ({ ticket: ticketRow(id), messages: [message(1, 'user')] }))
    state.reply.mockImplementation(async (id: number) => ({ ticket: ticketRow(id, { status: 'replied', user_unread: true }), message: message(2, 'operator', 'hello there') }))
    state.close.mockImplementation(async (id: number) => ticketRow(id, { status: 'closed' }))
    state.reopen.mockImplementation(async (id: number) => ticketRow(id, { status: 'open' }))
    state.fetchCounts.mockResolvedValue(undefined)
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('lists tickets with the requester email and the open-count pill', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(state.list).toHaveBeenCalledWith(1, 20, { status: 'open', category: '', search: '' }, expect.anything())
    expect(wrapper.find('[data-test="row-1"]').text()).toContain('user1@example.com')
    expect(wrapper.find('[data-test="ticket-status-2"]').text()).toBe('tickets.status.replied')
    expect(wrapper.find('[data-test="tickets-tab-open"]').text()).toContain('2')
    expect(state.fetchCounts).toHaveBeenCalled()
  })

  it('opens the detail and lets staff reply directly (no approval round-trip)', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="ticket-view-1"]').trigger('click')
    await flushPromises()
    expect(state.get).toHaveBeenCalledWith(1)
    const thread = wrapper.find('[data-test="thread"]')
    expect(thread.exists()).toBe(true)
    expect(thread.attributes('data-viewer')).toBe('staff')
    expect(wrapper.find('[data-test="ticket-detail-user"]').text()).toBe('user1@example.com')

    state.list.mockClear()
    await wrapper.find('[data-test="thread-send"]').trigger('click')
    await flushPromises()
    expect(state.reply).toHaveBeenCalledWith(1, 'hello there')
    expect(state.showSuccess).toHaveBeenCalledWith('tickets.toast.replied')
    expect(wrapper.find('[data-test="thread"]').attributes('data-count')).toBe('2')
    expect(wrapper.find('[data-test="ticket-detail-status"]').text()).toBe('tickets.status.replied')
    expect(state.list).toHaveBeenCalledTimes(1)
    expect(state.fetchCounts).toHaveBeenCalledTimes(2)
  })

  it('closes with confirmation and reopens', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="ticket-view-1"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="ticket-close"]').trigger('click')
    expect(wrapper.find('[data-test="confirm-dialog"]').exists()).toBe(true)
    await wrapper.find('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()
    expect(state.close).toHaveBeenCalledWith(1)
    expect(state.showSuccess).toHaveBeenCalledWith('tickets.toast.closed')
    expect(wrapper.find('[data-test="ticket-close"]').exists()).toBe(false)

    await wrapper.find('[data-test="ticket-reopen"]').trigger('click')
    await flushPromises()
    expect(state.reopen).toHaveBeenCalledWith(1)
    expect(state.showSuccess).toHaveBeenCalledWith('tickets.toast.reopened')
    expect(wrapper.find('[data-test="ticket-close"]').exists()).toBe(true)
  })

  it('surfaces API errors when a reply is rejected', async () => {
    state.reply.mockRejectedValueOnce({ status: 409, code: 409, reason: 'TICKET_CLOSED', message: 'ticket is closed' })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="ticket-view-1"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="thread-send"]').trigger('click')
    await flushPromises()
    expect(state.showError).toHaveBeenCalledWith('tickets.errors.closed')
    expect(state.get).toHaveBeenCalledTimes(2)
  })

  it('opens the ticket from a ?id= deep link and follows its status tab', async () => {
    state.routeQuery = { id: '2' }
    state.get.mockResolvedValueOnce({ ticket: ticketRow(2, { status: 'closed' }), messages: [message(1, 'user')] })
    const wrapper = mountView()
    await flushPromises()

    expect(state.get).toHaveBeenCalledWith(2)
    expect(wrapper.find('[data-test="detail-dialog"]').exists()).toBe(true)
    expect(state.list).toHaveBeenLastCalledWith(1, 20, { status: 'closed', category: '', search: '' }, expect.anything())
  })
})
