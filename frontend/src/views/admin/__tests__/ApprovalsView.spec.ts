import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import ApprovalsView from '../ApprovalsView.vue'

const state = vi.hoisted(() => ({
  isAdmin: true,
  routeQuery: {} as Record<string, string>,
  list: vi.fn(),
  get: vi.fn(),
  pendingCount: vi.fn(),
  approve: vi.fn(),
  reject: vi.fn(),
  cancel: vi.fn(),
  batchApprove: vi.fn(),
  listFilterGroups: vi.fn(),
  listAttributeDefinitions: vi.fn(),
  fetchPendingCount: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    approvals: {
      list: state.list,
      get: state.get,
      pendingCount: state.pendingCount,
      approve: state.approve,
      reject: state.reject,
      cancel: state.cancel,
      batchApprove: state.batchApprove
    },
    usage: { listFilterGroups: state.listFilterGroups },
    userAttributes: { listDefinitions: state.listAttributeDefinitions }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: state.showSuccess, showError: state.showError, showInfo: vi.fn() })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isAdmin() {
      return state.isAdmin
    },
    get isOperator() {
      return !state.isAdmin
    }
  })
}))

vi.mock('@/stores/approvals', () => ({
  useApprovalsStore: () => ({ pendingCount: 0, fetchPendingCount: state.fetchPendingCount })
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
const TablePageLayoutStub = { template: '<div><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>' }
const DataTableStub = defineComponent({
  props: ['columns', 'data', 'loading', 'rowKey'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" :data-test="'row-' + row.id">
        <slot name="cell-action" :row="row" :value="row.action" />
        <slot name="cell-status" :row="row" :value="row.status" />
        <slot name="cell-actions" :row="row" :value="null" />
      </div>
      <slot v-if="!data.length" name="empty" />
    </div>
  `
})
const BaseDialogStub = defineComponent({
  props: ['show', 'title', 'width', 'closeOnClickOutside'],
  emits: ['close'],
  template: '<div v-if="show" :data-test="\'dialog-\' + title"><slot /><slot name="footer" /></div>'
})
const ConfirmDialogStub = defineComponent({
  props: ['show', 'title', 'message', 'danger'],
  emits: ['confirm', 'cancel'],
  template: '<div v-if="show" data-test="confirm-dialog"><button data-test="confirm-ok" @click="$emit(\'confirm\')">ok</button></div>'
})

function pendingRow(id: number, extra: Record<string, unknown> = {}) {
  return {
    id,
    status: 'pending',
    action: 'admin.users.balance.create',
    method: 'POST',
    route_template: '/api/v1/admin/users/:id/balance',
    request_path: `/api/v1/admin/users/${id}/balance`,
    target_type: 'user',
    target_summary: `user${id}@example.com`,
    requester: { id: 2, email: 'ops@example.com' },
    request_body: '{"balance":10}',
    expires_at: '2026-09-19T00:00:00Z',
    created_at: '2026-09-16T00:00:00Z',
    updated_at: '2026-09-16T00:00:00Z',
    ...extra
  }
}

function mountView() {
  return mount(ApprovalsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        Pagination: true
      }
    }
  })
}

describe('ApprovalsView', () => {
  beforeEach(() => {
    Object.keys(state.routeQuery).forEach((k) => delete state.routeQuery[k])
    state.list.mockReset().mockResolvedValue({ items: [pendingRow(1), pendingRow(2)], total: 2, page: 1, page_size: 20, pages: 1 })
    state.get.mockReset()
    state.pendingCount.mockReset().mockResolvedValue({ pending: 2 })
    state.approve.mockReset()
    state.reject.mockReset()
    state.cancel.mockReset()
    state.batchApprove.mockReset()
    state.listFilterGroups.mockReset().mockResolvedValue([{ id: 3, name: 'Pro 分组', platform: 'openai', status: 'active', subscription_type: 'subscription', is_exclusive: false }])
    state.listAttributeDefinitions.mockReset().mockResolvedValue([{ id: 5, name: '公司', key: 'company' }])
    state.fetchPendingCount.mockReset()
    state.showSuccess.mockReset()
    state.showError.mockReset()
  })

  afterEach(() => {
    state.isAdmin = true
  })

  it('admin approves with a single click and the list refreshes', async () => {
    state.isAdmin = true
    state.approve.mockResolvedValue({ approval: { ...pendingRow(1), status: 'approved', result_status_code: 200 }, replay: { status_code: 200, duration_ms: 3 } })
    const wrapper = mountView()
    await flushPromises()

    expect(state.list).toHaveBeenCalledWith(1, 20, { status: 'pending' }, expect.anything())
    expect(wrapper.find('[data-test="approve-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="reject-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cancel-1"]').exists()).toBe(true, 'admin can also withdraw a single pending request')

    state.list.mockClear()
    await wrapper.find('[data-test="approve-1"]').trigger('click')
    await flushPromises()

    expect(state.approve).toHaveBeenCalledTimes(1)
    expect(state.approve).toHaveBeenCalledWith(1)
    expect(wrapper.find('[data-test="confirm-dialog"]').exists()).toBe(false)
    expect(state.showSuccess).toHaveBeenCalled()
    expect(state.list).toHaveBeenCalledTimes(1)
    expect(state.fetchPendingCount).toHaveBeenCalled()
  })

  it('admin sees a failed replay as an error and the row status comes from the response', async () => {
    state.approve.mockResolvedValue({ approval: { ...pendingRow(1), status: 'failed', result_error: 'STEP_UP_REQUIRED' }, replay: { status_code: 403, duration_ms: 1 } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="approve-1"]').trigger('click')
    await flushPromises()
    expect(state.showError).toHaveBeenCalledWith(expect.stringContaining('STEP_UP_REQUIRED'))
  })

  it('admin rejects through the reason dialog', async () => {
    state.reject.mockResolvedValue({ approval: { ...pendingRow(2), status: 'rejected', decision_reason: 'nope' } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="reject-2"]').trigger('click')
    await flushPromises()
    const reason = wrapper.find('[data-test="reject-reason"]')
    expect(reason.exists()).toBe(true)
    await reason.setValue('  nope  ')
    await wrapper.find('[data-test="confirm-reject"]').trigger('click')
    await flushPromises()

    expect(state.reject).toHaveBeenCalledWith(2, 'nope')
    expect(state.showSuccess).toHaveBeenCalled()
  })

  it('admin approves everything pending in one click after a single confirmation', async () => {
    state.batchApprove.mockResolvedValue({ results: [{ id: 1, status: 'approved' }, { id: 2, status: 'approved' }], approved: 2, failed: 0, skipped: 0 })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="approve-selected"]').attributes('disabled')).toBeDefined()
    await wrapper.find('[data-test="approve-all"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="confirm-dialog"]').exists()).toBe(true)
    expect(state.batchApprove).not.toHaveBeenCalled()

    state.list.mockClear()
    await wrapper.find('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()

    expect(state.batchApprove).toHaveBeenCalledTimes(1)
    expect(state.batchApprove).toHaveBeenCalledWith([1, 2])
    expect(state.showSuccess).toHaveBeenCalledWith(expect.stringContaining('operator.approval.batchResult'))
    expect(state.list).toHaveBeenCalled()
    expect(state.fetchPendingCount).toHaveBeenCalled()
  })

  it('admin withdraws a single pending request without a reason', async () => {
    state.cancel.mockResolvedValue({ approval: { ...pendingRow(2), status: 'cancelled' } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="cancel-2"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()

    expect(state.cancel).toHaveBeenCalledWith(2)
    expect(state.reject).not.toHaveBeenCalled()
  })

  it('operator only sees withdraw and confirms it', async () => {
    state.isAdmin = false
    state.cancel.mockResolvedValue({ approval: { ...pendingRow(1), status: 'cancelled' } })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="approve-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="reject-1"]').exists()).toBe(false)
    await wrapper.find('[data-test="cancel-1"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="confirm-dialog"]').exists()).toBe(true)
    await wrapper.find('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()

    expect(state.cancel).toHaveBeenCalledWith(1)
    expect(state.fetchPendingCount).toHaveBeenCalled()
  })

  it('renders a plain-language description of each request inline, resolving group names', async () => {
    state.list.mockResolvedValue({
      items: [
        pendingRow(1, { request_body: '{"balance":10,"operation":"add","notes":"充值"}' }),
        pendingRow(2, {
          action: 'admin.subscriptions.assign.create',
          method: 'POST',
          route_template: '/api/v1/admin/subscriptions/assign',
          request_path: '/api/v1/admin/subscriptions/assign',
          target_type: 'subscription',
          target_summary: 'user2@example.com · Pro 分组',
          request_body: '{"user_id":2,"group_id":3,"validity_days":30}'
        })
      ],
      total: 2, page: 1, page_size: 20, pages: 1
    })
    const wrapper = mountView()
    await flushPromises()

    const first = wrapper.find('[data-test="summary-1"]').text()
    expect(first).toContain('operator.approval.describe.summary.balanceAdd:{"target":"user1@example.com","amount":"10"}')
    expect(first).toContain('operator.approval.describe.fields.notes：充值')

    const second = wrapper.find('[data-test="summary-2"]').text()
    expect(second).toContain('operator.approval.describe.summary.subscriptionAssign')
    expect(second).toContain('operator.approval.describe.fields.group：Pro 分组')
    expect(second).toContain('operator.approval.describe.values.days:{"n":30}')
    expect(state.listFilterGroups).toHaveBeenCalledTimes(1)
  })

  it('opens the detail dialog from a deep link and switches to the processed tab for a decided request', async () => {
    state.routeQuery.id = '9'
    state.get.mockResolvedValue({ ...pendingRow(9), status: 'approved', result_status_code: 200 })
    const wrapper = mountView()
    await flushPromises()

    expect(state.get).toHaveBeenCalledWith(9)
    expect(wrapper.find('[data-test="dialog-operator.approval.detail.titleWithId:{\\"id\\":9}"]').exists()).toBe(true)
    expect(state.list).toHaveBeenLastCalledWith(1, 20, { status: 'processed' }, expect.anything())
  })
})
