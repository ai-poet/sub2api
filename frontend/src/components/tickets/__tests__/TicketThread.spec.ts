import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TicketThread from '@/components/tickets/TicketThread.vue'

const state = vi.hoisted(() => ({
  uploadUser: vi.fn(),
  uploadAdmin: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/tickets', () => ({
  uploadAttachment: state.uploadUser
}))

vi.mock('@/api/admin/tickets', () => ({
  uploadAttachment: state.uploadAdmin
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: state.showSuccess, showError: state.showError, showInfo: vi.fn() })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key) })
  }
})

vi.mock('@/utils/format', () => ({ formatDateTime: (v: string) => v }))

function ticket(status = 'open') {
  return {
    id: 1,
    title: 'Ticket 1',
    category: 'other',
    status,
    user_unread: false,
    message_count: 1,
    last_message_at: '2026-09-20T10:00:00Z',
    created_at: '2026-09-20T10:00:00Z',
    updated_at: '2026-09-20T10:00:00Z'
  }
}

function message(id: number, role: string, body: string) {
  return { id, ticket_id: 1, author_role: role, body, created_at: '2026-09-20T10:00:00Z' }
}

function mountThread(side: 'user' | 'admin', messages: ReturnType<typeof message>[] = []) {
  return mount(TicketThread, {
    props: {
      ticket: ticket(),
      messages,
      viewer: side === 'user' ? 'user' : 'staff',
      side
    }
  })
}

function pickFile(wrapper: ReturnType<typeof mountThread>, file: File) {
  const input = wrapper.find('[data-test="ticket-attachment-input"]')
  Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
  return input.trigger('change')
}

function draftValue(wrapper: ReturnType<typeof mountThread>): string {
  return (wrapper.find('textarea').element as HTMLTextAreaElement).value
}

describe('TicketThread attachments (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    state.uploadUser.mockResolvedValue({ key: 'tickets/1/a.png', content_type: 'image/png', size: 1 })
    state.uploadAdmin.mockResolvedValue({ key: 'tickets/1/b.png', content_type: 'image/png', size: 1 })
  })

  it('uploads a picked image on the user side and replaces the placeholder with attachment markdown', async () => {
    const wrapper = mountThread('user')
    await pickFile(wrapper, new File(['x'], 'a.png', { type: 'image/png' }))
    await flushPromises()

    expect(state.uploadUser).toHaveBeenCalledTimes(1)
    expect(state.uploadAdmin).not.toHaveBeenCalled()
    expect(draftValue(wrapper)).toContain('![image](ticket-attachment://tickets/1/a.png)')
    expect(draftValue(wrapper)).not.toContain('uploading-')
  })

  it('uploads a picked image on the admin side via the admin endpoint', async () => {
    const wrapper = mountThread('admin')
    await pickFile(wrapper, new File(['x'], 'b.png', { type: 'image/png' }))
    await flushPromises()

    expect(state.uploadAdmin).toHaveBeenCalledTimes(1)
    expect(state.uploadUser).not.toHaveBeenCalled()
    expect(draftValue(wrapper)).toContain('![image](ticket-attachment://tickets/1/b.png)')
  })

  it('uploads an image pasted into the textarea', async () => {
    const wrapper = mountThread('user')
    const file = new File(['x'], 'pasted.png', { type: 'image/png' })
    const event = new Event('paste', { bubbles: true })
    Object.defineProperty(event, 'clipboardData', { value: { files: [file] } })
    wrapper.find('textarea').element.dispatchEvent(event)
    await flushPromises()

    expect(state.uploadUser).toHaveBeenCalledTimes(1)
    expect(draftValue(wrapper)).toContain('![image](ticket-attachment://tickets/1/a.png)')
  })

  it('rejects files over the 5MiB pre-check without calling the API', async () => {
    const wrapper = mountThread('user')
    const file = new File(['x'], 'huge.png', { type: 'image/png' })
    Object.defineProperty(file, 'size', { value: 6 * 1024 * 1024 })
    await pickFile(wrapper, file)
    await flushPromises()

    expect(state.uploadUser).not.toHaveBeenCalled()
    expect(state.showError).toHaveBeenCalledWith('tickets.attachments.errors.tooLarge')
    expect(draftValue(wrapper)).toBe('')
  })

  it('maps TICKET_ATTACHMENT_STORAGE_NOT_CONFIGURED to a localized toast and removes the placeholder', async () => {
    state.uploadUser.mockRejectedValueOnce({ status: 503, reason: 'TICKET_ATTACHMENT_STORAGE_NOT_CONFIGURED', message: 'storage not configured' })
    const wrapper = mountThread('user')
    await pickFile(wrapper, new File(['x'], 'a.png', { type: 'image/png' }))
    await flushPromises()

    expect(state.showError).toHaveBeenCalledWith('tickets.attachments.errors.storageNotConfigured')
    expect(draftValue(wrapper)).not.toContain('uploading-')
    expect(draftValue(wrapper)).not.toContain('ticket-attachment://')
  })

  it('maps TICKET_ATTACHMENT_TOO_LARGE and TICKET_ATTACHMENT_BAD_TYPE to localized toasts', async () => {
    state.uploadUser
      .mockRejectedValueOnce({ status: 413, reason: 'TICKET_ATTACHMENT_TOO_LARGE', message: 'too large' })
      .mockRejectedValueOnce({ status: 400, reason: 'TICKET_ATTACHMENT_BAD_TYPE', message: 'bad type' })
    const wrapper = mountThread('user')
    await pickFile(wrapper, new File(['x'], 'a.png', { type: 'image/png' }))
    await flushPromises()
    await pickFile(wrapper, new File(['x'], 'a.png', { type: 'image/png' }))
    await flushPromises()

    expect(state.showError).toHaveBeenNthCalledWith(1, 'tickets.attachments.errors.tooLarge')
    expect(state.showError).toHaveBeenNthCalledWith(2, 'tickets.attachments.errors.badType')
  })

  it('rewrites ticket-attachment urls to the user-side content endpoint when rendering', async () => {
    const wrapper = mountThread('user', [message(1, 'staff', '看图 ![image](ticket-attachment://tickets/1/a.png)')])
    await flushPromises()

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('/api/v1/tickets/attachments/content?key=tickets%2F1%2Fa.png')
  })

  it('rewrites ticket-attachment urls to the admin-side content endpoint when rendering', async () => {
    const wrapper = mountThread('admin', [message(1, 'user', '![image](ticket-attachment://k.png)')])
    await flushPromises()

    expect(wrapper.find('img').attributes('src')).toBe('/api/v1/admin/tickets/attachments/content?key=k.png')
  })
})
