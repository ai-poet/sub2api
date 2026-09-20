import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import TicketsView from '../TicketsView.vue'

const state = vi.hoisted(() => ({
  routeQuery: {} as Record<string, string>,
  list: vi.fn(),
  get: vi.fn(),
  create: vi.fn(),
  reply: vi.fn(),
  close: vi.fn(),
  reopen: vi.fn(),
  uploadAttachment: vi.fn(),
  fetchCounts: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/tickets', () => {
  const api = {
    list: state.list,
    get: state.get,
    create: state.create,
    reply: state.reply,
    close: state.close,
    reopen: state.reopen,
    unreadCount: vi.fn(),
    uploadAttachment: state.uploadAttachment
  }
  return { default: api, ticketsAPI: api, uploadAttachment: state.uploadAttachment }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: state.showSuccess, showError: state.showError, showInfo: vi.fn() })
}))

vi.mock('@/stores/tickets', () => ({
  useTicketsStore: () => ({ openCount: 0, unreadCount: 1, fetchCounts: state.fetchCounts })
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
const DataTableStub = defineComponent({
  props: ['columns', 'data', 'loading', 'rowKey', 'clickableRows'],
  emits: ['rowClick'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" :data-test="'row-' + row.id" @click="$emit('rowClick', row)">
        <slot name="cell-title" :row="row" :value="row.title" />
        <slot name="cell-status" :row="row" :value="row.status" />
        <slot name="cell-actions" :row="row" :value="null" />
      </div>
      <slot v-if="!data.length" name="empty" />
    </div>
  `
})
const PaginationStub = { props: ['total', 'page', 'pageSize'], template: '<div data-test="pagination" />' }
const EmptyStateStub = { props: ['title', 'description'], template: '<div data-test="empty">{{ title }}</div>' }
const InputStub = defineComponent({
  props: ['modelValue', 'label', 'placeholder', 'hint', 'required'],
  emits: ['update:modelValue'],
  template: '<input data-test="stub-title" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
})
const TextAreaStub = defineComponent({
  props: ['modelValue', 'label', 'placeholder', 'hint', 'rows', 'required'],
  emits: ['update:modelValue'],
  template: '<textarea data-test="stub-body" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
})
const SelectStub = { props: ['modelValue', 'options'], template: '<select data-test="stub-category" />' }
const BaseDialogStub = defineComponent({
  props: ['show', 'title', 'width'],
  emits: ['close'],
  template: '<div v-if="show" :data-test="\'dialog-\' + title"><slot /><slot name="footer" /></div>'
})
const ConfirmDialogStub = defineComponent({
  props: ['show', 'title', 'message', 'danger'],
  emits: ['confirm', 'cancel'],
  template: '<div v-if="show" data-test="confirm-dialog"><button data-test="confirm-ok" @click="$emit(\'confirm\')">ok</button></div>'
})
const TicketThreadStub = defineComponent({
  props: ['ticket', 'messages', 'viewer', 'side', 'submitting', 'loading'],
  emits: ['reply'],
  methods: {
    clearDraft() {
      /* noop */
    }
  },
  template: `
    <div data-test="thread" :data-viewer="viewer" :data-side="side" :data-count="messages.length">
      <button data-test="thread-send" @click="$emit('reply', 'more details')">send</button>
    </div>
  `
})

function ticketRow(id: number, extra: Record<string, unknown> = {}) {
  return {
    id,
    title: `Ticket ${id}`,
    category: 'billing',
    status: 'open',
    user_unread: false,
    message_count: 1,
    last_message_at: '2026-09-20T10:00:00Z',
    created_at: '2026-09-20T10:00:00Z',
    updated_at: '2026-09-20T10:00:00Z',
    ...extra
  }
}

function message(id: number, role: string, body = 'body') {
  return { id, ticket_id: 1, author_role: role, body, created_at: '2026-09-20T10:00:00Z' }
}

function mountView() {
  return mount(TicketsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        DataTable: DataTableStub,
        Pagination: PaginationStub,
        EmptyState: EmptyStateStub,
        Input: InputStub,
        TextArea: TextAreaStub,
        Select: SelectStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        TicketThread: TicketThreadStub
      }
    }
  })
}

describe('user TicketsView (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    state.routeQuery = {}
    state.list.mockResolvedValue({ items: [ticketRow(1, { user_unread: true, status: 'replied' }), ticketRow(2)], total: 2, page: 1, page_size: 20, pages: 1 })
    state.get.mockImplementation(async (id: number) => ({ ticket: ticketRow(id, { status: 'replied' }), messages: [message(1, 'user'), message(2, 'staff')] }))
    state.create.mockResolvedValue(ticketRow(3, { title: 'New problem', category: 'other' }))
    state.reply.mockImplementation(async (id: number) => ({ ticket: ticketRow(id, { status: 'open' }), message: message(3, 'user', 'more details') }))
    state.close.mockImplementation(async (id: number) => ticketRow(id, { status: 'closed' }))
    state.reopen.mockImplementation(async (id: number) => ticketRow(id, { status: 'open' }))
    state.uploadAttachment.mockResolvedValue({ key: 'tickets/3/shot.png', content_type: 'image/png', size: 1 })
    state.fetchCounts.mockResolvedValue(undefined)
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('lists my tickets and marks the ones with unread replies', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(state.list).toHaveBeenCalledWith(1, 20, { status: '' }, expect.anything())
    expect(wrapper.find('[data-test="ticket-unread-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="ticket-unread-2"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="ticket-status-1"]').text()).toBe('tickets.status.replied')
  })

  it('filters by status tab', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="tickets-tab-closed"]').trigger('click')
    await flushPromises()
    expect(state.list).toHaveBeenLastCalledWith(1, 20, { status: 'closed' }, expect.anything())
  })

  it('opens the detail, refreshes the unread badge and replies from the thread', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="ticket-view-1"]').trigger('click')
    await flushPromises()
    expect(state.get).toHaveBeenCalledWith(1)
    expect(state.fetchCounts).toHaveBeenCalledTimes(1)
    const thread = wrapper.find('[data-test="thread"]')
    expect(thread.attributes('data-viewer')).toBe('user')
    expect(thread.attributes('data-side')).toBe('user')
    expect(thread.attributes('data-count')).toBe('2')

    await wrapper.find('[data-test="thread-send"]').trigger('click')
    await flushPromises()
    expect(state.reply).toHaveBeenCalledWith(1, 'more details')
    expect(state.showSuccess).toHaveBeenCalledWith('tickets.toast.replied')
    expect(wrapper.find('[data-test="thread"]').attributes('data-count')).toBe('3')
    expect(wrapper.find('[data-test="ticket-detail-status"]').text()).toBe('tickets.status.open')
  })

  it('creates a ticket from the dialog and opens it afterwards', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="ticket-create"]').trigger('click')
    const submit = wrapper.find('[data-test="ticket-submit"]')
    expect(submit.attributes('disabled')).toBeDefined()

    await wrapper.find('[data-test="stub-title"]').setValue('New problem')
    await wrapper.find('[data-test="stub-body"]').setValue('Something **broke**')
    expect(wrapper.find('[data-test="ticket-submit"]').attributes('disabled')).toBeUndefined()

    await wrapper.find('[data-test="ticket-submit"]').trigger('click')
    await flushPromises()
    expect(state.create).toHaveBeenCalledWith({ title: 'New problem', category: 'other', body: 'Something **broke**' })
    expect(state.showSuccess).toHaveBeenCalledWith('tickets.toast.created')
    expect(state.list).toHaveBeenCalledTimes(2)
    expect(state.get).toHaveBeenCalledWith(3)
  })

  it('inserts attachment markdown into the create form after a successful upload', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="ticket-create"]').trigger('click')
    const input = wrapper.find('[data-test="ticket-create-attachment-input"]')
    const file = new File(['x'], 'shot.png', { type: 'image/png' })
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')
    await flushPromises()

    expect(state.uploadAttachment).toHaveBeenCalledTimes(1)
    const body = wrapper.find('[data-test="stub-body"]').element as HTMLTextAreaElement
    expect(body.value).toContain('![image](ticket-attachment://tickets/3/shot.png)')

    await wrapper.find('[data-test="stub-title"]').setValue('New problem')
    await wrapper.find('[data-test="ticket-submit"]').trigger('click')
    await flushPromises()
    expect(state.create).toHaveBeenCalledWith({
      title: 'New problem',
      category: 'other',
      body: '![image](ticket-attachment://tickets/3/shot.png)'
    })
  })

  it('shows the mapped error when the create-form upload fails', async () => {
    state.uploadAttachment.mockRejectedValueOnce({ status: 503, reason: 'TICKET_ATTACHMENT_STORAGE_NOT_CONFIGURED', message: 'not configured' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="ticket-create"]').trigger('click')
    const input = wrapper.find('[data-test="ticket-create-attachment-input"]')
    const file = new File(['x'], 'shot.png', { type: 'image/png' })
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')
    await flushPromises()

    expect(state.showError).toHaveBeenCalledWith('tickets.attachments.errors.storageNotConfigured')
    const body = wrapper.find('[data-test="stub-body"]').element as HTMLTextAreaElement
    expect(body.value).not.toContain('uploading-')
    expect(body.value).not.toContain('ticket-attachment://')
  })

  it('maps the open-limit error to a friendly message', async () => {
    state.create.mockRejectedValueOnce({ status: 409, code: 409, reason: 'TICKET_OPEN_LIMIT', message: 'too many open tickets' })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="ticket-create"]').trigger('click')
    await wrapper.find('[data-test="stub-title"]').setValue('t')
    await wrapper.find('[data-test="stub-body"]').setValue('b')
    await wrapper.find('[data-test="ticket-submit"]').trigger('click')
    await flushPromises()
    expect(state.showError).toHaveBeenCalledWith('tickets.errors.openLimit')
  })

  it('closes with confirmation and can reopen', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="ticket-view-2"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="ticket-close"]').trigger('click')
    await wrapper.find('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()
    expect(state.close).toHaveBeenCalledWith(2)
    expect(wrapper.find('[data-test="ticket-reopen"]').exists()).toBe(true)

    await wrapper.find('[data-test="ticket-reopen"]').trigger('click')
    await flushPromises()
    expect(state.reopen).toHaveBeenCalledWith(2)
    expect(wrapper.find('[data-test="ticket-close"]').exists()).toBe(true)
  })

  it('opens a ticket from the ?id= deep link', async () => {
    state.routeQuery = { id: '2' }
    mountView()
    await flushPromises()
    expect(state.get).toHaveBeenCalledWith(2)
  })
})
