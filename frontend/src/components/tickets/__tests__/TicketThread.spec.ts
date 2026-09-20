import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import TicketThread from '@/components/tickets/TicketThread.vue'

const state = vi.hoisted(() => ({
  uploadUser: vi.fn(),
  uploadAdmin: vi.fn(),
  fetchUser: vi.fn(),
  fetchAdmin: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/tickets', () => ({
  uploadAttachment: state.uploadUser,
  fetchAttachment: state.fetchUser
}))

vi.mock('@/api/admin/tickets', () => ({
  uploadAttachment: state.uploadAdmin,
  fetchAttachment: state.fetchAdmin
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

let objectUrlSeq = 0

describe('TicketThread attachments (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    state.uploadUser.mockResolvedValue({ key: 'tickets/1/a.png', content_type: 'image/png', size: 1 })
    state.uploadAdmin.mockResolvedValue({ key: 'tickets/1/b.png', content_type: 'image/png', size: 1 })
    state.fetchUser.mockResolvedValue(new Blob(['x'], { type: 'image/png' }))
    state.fetchAdmin.mockResolvedValue(new Blob(['x'], { type: 'image/png' }))
    objectUrlSeq = 0
    vi.stubGlobal('URL', {
      ...URL,
      createObjectURL: vi.fn(() => `blob:http://localhost/obj-${++objectUrlSeq}`),
      revokeObjectURL: vi.fn()
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
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

  // <img> 自己请求时不会带 Authorization 头，所以正文里的图必须先经鉴权取回、
  // 再以 blob: URL 渲染，不能直接挂同源的 content 地址（那样一律 401 / 裂图）。
  it('fetches user-side attachments through the api and renders them as object urls', async () => {
    const wrapper = mountThread('user', [message(1, 'staff', '看图 ![image](ticket-attachment://tickets/1/a.png)')])
    await flushPromises()

    expect(state.fetchUser).toHaveBeenCalledWith('tickets/1/a.png')
    expect(state.fetchAdmin).not.toHaveBeenCalled()
    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('blob:http://localhost/obj-1')
  })

  it('fetches admin-side attachments through the admin endpoint', async () => {
    const wrapper = mountThread('admin', [message(1, 'user', '![image](ticket-attachment://k.png)')])
    await flushPromises()

    expect(state.fetchAdmin).toHaveBeenCalledWith('k.png')
    expect(state.fetchUser).not.toHaveBeenCalled()
    expect(wrapper.find('img').attributes('src')).toBe('blob:http://localhost/obj-1')
  })

  it('fetches each distinct key once even when it appears in several messages', async () => {
    mountThread('user', [
      message(1, 'user', '![image](ticket-attachment://same.png)'),
      message(2, 'staff', '还是它 ![image](ticket-attachment://same.png) 加一张 ![image](ticket-attachment://other.png)')
    ])
    await flushPromises()

    expect(state.fetchUser).toHaveBeenCalledTimes(2)
    expect(state.fetchUser.mock.calls.map(([key]) => key).sort()).toEqual(['other.png', 'same.png'])
  })

  it('leaves the image without a src when the attachment cannot be fetched', async () => {
    state.fetchUser.mockRejectedValueOnce(new Error('403'))
    const wrapper = mountThread('user', [message(1, 'staff', '![image](ticket-attachment://gone.png)')])
    await flushPromises()

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBeUndefined()
    // 单张取不回来不弹 toast——一条工单里可能有多张图，逐张报错只会刷屏。
    expect(state.showError).not.toHaveBeenCalled()
  })

  it('revokes the object urls it created when the thread unmounts', async () => {
    const wrapper = mountThread('user', [message(1, 'staff', '![image](ticket-attachment://a.png)')])
    await flushPromises()
    wrapper.unmount()

    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:http://localhost/obj-1')
  })
})
