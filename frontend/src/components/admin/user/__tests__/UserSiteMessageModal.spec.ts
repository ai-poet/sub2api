import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import UserSiteMessageModal from '../UserSiteMessageModal.vue'
import type { AdminUser } from '@/types'

const mocks = vi.hoisted(() => ({
  sendSiteMessage: vi.fn(),
  listSiteMessages: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))
vi.mock('@/api/admin', () => ({
  adminAPI: { users: { sendSiteMessage: mocks.sendSiteMessage, listSiteMessages: mocks.listSiteMessages } }
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }) }))
vi.mock('@/utils/format', () => ({ formatDateTime: (v: string) => v }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key) })
}))

enableAutoUnmount(afterEach)

const user = { id: 9, email: 'u@example.com', status: 'active' } as AdminUser

function mountModal(props: Record<string, unknown> = {}) {
  return mount(UserSiteMessageModal, {
    props: { show: true, user, ...props },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        MarkdownRenderer: { props: ['content'], template: '<div class="md">{{ content }}</div>' }
      }
    }
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.listSiteMessages.mockResolvedValue({ items: [], total: 0 })
})

describe('UserSiteMessageModal (fork)', () => {
  it('loads the history when opened', async () => {
    mocks.listSiteMessages.mockResolvedValue({
      items: [
        { id: 1, category: 'admin', title: '旧消息', content: 'x', from: 'staff', read_at: '2026-10-01T00:00:00Z', created_at: '2026-10-01T00:00:00Z', user_id: 9, source_type: 'admin', sender_user_id: 7, sender_role: 'operator', sender_email: 'ops@example.com', approval_id: 33 }
      ],
      total: 1
    })
    const w = mountModal()
    await flushPromises()
    expect(mocks.listSiteMessages).toHaveBeenCalledWith(9, 1, 10)
    const row = w.get('[data-testid="site-message-history-row"]')
    expect(row.text()).toContain('旧消息')
    expect(row.text()).toContain('admin.users.siteMessage.read')
    expect(row.text()).toContain('ops@example.com (operator)')
    expect(row.text()).toContain('admin.users.siteMessage.viaApproval:{"id":33}')
  })

  it('validates title and content before sending', async () => {
    const w = mountModal()
    await flushPromises()
    await w.get('form').trigger('submit')
    expect(mocks.showError).toHaveBeenCalledWith('admin.users.siteMessage.titleRequired')

    await w.get('input#site-message-title').setValue('标题')
    await w.get('form').trigger('submit')
    expect(mocks.showError).toHaveBeenLastCalledWith('admin.users.siteMessage.contentRequired')
    expect(mocks.sendSiteMessage).not.toHaveBeenCalled()
  })

  it('sends with an idempotency key, then clears the form and reloads history', async () => {
    mocks.sendSiteMessage.mockResolvedValue({ id: 2 })
    const w = mountModal()
    await flushPromises()
    await w.get('input#site-message-title').setValue('  标题 ')
    await w.get('textarea#site-message-content').setValue('**正文**')
    await w.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.sendSiteMessage).toHaveBeenCalledTimes(1)
    const [id, payload, key] = mocks.sendSiteMessage.mock.calls[0]
    expect(id).toBe(9)
    expect(payload).toEqual({ title: '标题', content: '**正文**' })
    expect(key).toMatch(/^site-message-9-/)
    expect(mocks.showSuccess).toHaveBeenCalledWith('admin.users.siteMessage.sent')
    expect((w.get('input#site-message-title').element as HTMLInputElement).value).toBe('')
    expect(mocks.listSiteMessages).toHaveBeenCalledTimes(2)
    expect(w.emitted('close')).toBeUndefined()
  })

  it('closes quietly when an operator request is queued for approval', async () => {
    mocks.sendSiteMessage.mockRejectedValue({ status: 202, code: 'APPROVAL_PENDING', approval: { approval_request_id: 5 } })
    const w = mountModal({ readonly: true })
    await flushPromises()
    expect(w.find('[data-testid="site-message-operator-hint"]').exists()).toBe(true)
    await w.get('input#site-message-title').setValue('t')
    await w.get('textarea#site-message-content').setValue('c')
    await w.get('form').trigger('submit')
    await flushPromises()

    expect(w.emitted('close')).toHaveLength(1)
    expect(mocks.showError).not.toHaveBeenCalled()
  })

  it('shows the API error and keeps the dialog open', async () => {
    mocks.sendSiteMessage.mockRejectedValue({ message: 'user not found' })
    const w = mountModal()
    await flushPromises()
    await w.get('input#site-message-title').setValue('t')
    await w.get('textarea#site-message-content').setValue('c')
    await w.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('user not found')
    expect(w.emitted('close')).toBeUndefined()
  })
})
