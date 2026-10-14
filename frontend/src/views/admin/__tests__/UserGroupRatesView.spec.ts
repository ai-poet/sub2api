import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import UserGroupRatesView from '../UserGroupRatesView.vue'
import { APPROVAL_PENDING_CODE } from '@/utils/approval'

const state = vi.hoisted(() => ({
  list: vi.fn(),
  groups: vi.fn(),
  usage: vi.fn(),
  applyBatch: vi.fn(),
  usersList: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    userGroupRates: { list: state.list, groups: state.groups, usage: state.usage, applyBatch: state.applyBatch },
    users: { list: state.usersList }
  }
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

vi.mock('@/composables/usePersistedPageSize', () => ({ getPersistedPageSize: () => 20 }))

const AppLayoutStub = { template: '<div><slot /></div>' }
const TablePageLayoutStub = { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' }
const DataTableStub = defineComponent({
  props: ['columns', 'data', 'loading', 'rowKey', 'selectedKeys'],
  emits: ['update:selected-keys', 'sort'],
  template: `
    <div>
      <div v-for="row in data" :key="rowKey(row)" :data-test="'row-' + rowKey(row)">
        <slot name="cell-user" :row="row" :value="null" />
        <slot name="cell-rate_multiplier" :row="row" :value="row.rate_multiplier" />
        <slot name="cell-discount" :row="row" :value="null" />
        <slot name="cell-usage" :row="row" :value="null" />
        <slot name="cell-actions" :row="row" :value="null" />
      </div>
      <button data-test="select-all-page" @click="$emit('update:selected-keys', data.map(rowKey))">select</button>
      <slot v-if="!data.length" name="empty" />
    </div>
  `
})
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

function rateRow(userId: number, groupId: number, rate: number, extra: Record<string, unknown> = {}) {
  return {
    user_id: userId,
    user_email: `u${userId}@example.com`,
    username: '',
    user_notes: '',
    user_status: 'active',
    group_id: groupId,
    group_name: `G${groupId}`,
    group_platform: 'anthropic',
    group_status: 'active',
    group_rate_multiplier: 1,
    rate_multiplier: rate,
    rpm_override: null,
    updated_at: '2026-10-14T00:00:00Z',
    ...extra
  }
}

function mountView() {
  return mount(UserGroupRatesView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        Select: true,
        GroupBadge: true,
        Pagination: true,
        EmptyState: true
      }
    }
  })
}

const queued = () => ({
  status: 202,
  code: APPROVAL_PENDING_CODE,
  message: 'accepted',
  approval: { approval_request_id: 9, status: 'pending', action: 'admin.users.batch_group_rates.create', target_summary: '1 users · 1 rates', expires_at: '' }
})

describe('UserGroupRatesView (fork)', () => {
  beforeEach(() => {
    state.list.mockReset().mockResolvedValue({
      items: [rateRow(1, 3, 0.8), rateRow(2, 3, 1.2)],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
      user_count: 2
    })
    state.groups.mockReset().mockResolvedValue([])
    state.usage.mockReset().mockResolvedValue([{ user_id: 1, group_id: 3, requests: 12, actual_cost: 3.5 }])
    state.applyBatch.mockReset().mockResolvedValue({ updated: 1, cleared: 0 })
    state.usersList.mockReset()
    state.showSuccess.mockReset()
    state.showError.mockReset()
  })

  it('loads the list, the group options and the 30-day usage of the listed users', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(state.list).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, page_size: 20, sort_by: 'updated_at', sort_order: 'desc' }),
      expect.anything()
    )
    expect(state.groups).toHaveBeenCalled()
    expect(state.usage).toHaveBeenCalledWith([1, 2])
    expect(wrapper.find('[data-test="ugr-summary"]').text()).toContain('admin.userGroupRates.summary:{"users":2,"entries":2}')
    expect(wrapper.find('[data-test="row-1:3"]').text()).toContain('$3.50')
    expect(wrapper.find('[data-test="row-2:3"]').text()).toContain('admin.userGroupRates.noUsage')
  })

  it('saves an inline edit as a one-item batch and reloads', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="row-1:3"] [data-test="ugr-edit"]').trigger('click')
    await wrapper.find('[data-test="ugr-edit-input"]').setValue('0.75')
    await wrapper.find('[data-test="ugr-edit-save"]').trigger('click')
    await flushPromises()

    expect(state.applyBatch).toHaveBeenCalledWith([{ user_id: 1, group_id: 3, rate_multiplier: 0.75 }])
    expect(state.showSuccess).toHaveBeenCalled()
    expect(state.list).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-test="ugr-edit-input"]').exists()).toBe(false)
  })

  it('rejects an invalid inline rate without calling the API', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="row-1:3"] [data-test="ugr-edit"]').trigger('click')
    await wrapper.find('[data-test="ugr-edit-input"]').setValue('0')
    await wrapper.find('[data-test="ugr-edit-save"]').trigger('click')
    await flushPromises()

    expect(state.applyBatch).not.toHaveBeenCalled()
    expect(state.showError).toHaveBeenCalledWith('admin.userGroupRates.invalidRate')
  })

  it('treats a queued approval as done without reporting an error (operator)', async () => {
    state.applyBatch.mockRejectedValue(queued())
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="row-1:3"] [data-test="ugr-edit"]').trigger('click')
    await wrapper.find('[data-test="ugr-edit-input"]').setValue('0.7')
    await wrapper.find('[data-test="ugr-edit-save"]').trigger('click')
    await flushPromises()

    expect(state.applyBatch).toHaveBeenCalledTimes(1)
    expect(state.showError).not.toHaveBeenCalled()
    expect(state.showSuccess).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="ugr-edit-input"]').exists()).toBe(false)
  })

  it('submits a batch percent adjustment for the selected rows', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="select-all-page"]').trigger('click')
    await wrapper.find('[data-test="ugr-batch"]').trigger('click')
    await wrapper.find('[data-test="ugr-mode-percent"]').trigger('click')
    await wrapper.find('[data-test="ugr-batch-value"]').setValue('10')
    expect(wrapper.findAll('[data-test="ugr-preview-row"]')).toHaveLength(2)
    await wrapper.find('[data-test="ugr-batch-submit"]').trigger('click')
    await flushPromises()

    expect(state.applyBatch).toHaveBeenCalledWith([
      { user_id: 1, group_id: 3, rate_multiplier: 0.88 },
      { user_id: 2, group_id: 3, rate_multiplier: 1.32 }
    ])
  })

  it('restores the group default after confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="row-2:3"] [data-test="ugr-reset"]').trigger('click')
    await wrapper.find('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()

    expect(state.applyBatch).toHaveBeenCalledWith([{ user_id: 2, group_id: 3, rate_multiplier: null }])
  })

  it('adds an exclusive rate for a searched user in the chosen groups', async () => {
    vi.useFakeTimers()
    state.groups.mockResolvedValue([
      { id: 3, name: 'G3', platform: 'anthropic', status: 'active', subscription_type: 'standard', is_exclusive: false, rate_multiplier: 1 },
      { id: 4, name: 'Sub', platform: 'anthropic', status: 'active', subscription_type: 'subscription', is_exclusive: false, rate_multiplier: 1 }
    ])
    state.usersList.mockResolvedValue({ items: [{ id: 5, email: 'new@example.com', username: '', group_rates: { 3: 0.9 } }], total: 1 })
    try {
      const wrapper = mountView()
      await flushPromises()

      await wrapper.find('[data-test="ugr-add"]').trigger('click')
      await wrapper.find('[data-test="ugr-add-user-search"]').setValue('new')
      await vi.advanceTimersByTimeAsync(350)
      await flushPromises()
      expect(state.usersList).toHaveBeenCalledWith(1, 10, { search: 'new' })
      await wrapper.find('[data-test="ugr-add-user-option"]').trigger('click')

      const groupBoxes = wrapper.findAll('[data-test="ugr-add-group"]')
      expect(groupBoxes).toHaveLength(1)
      await groupBoxes[0].setValue(true)
      await wrapper.find('[data-test="ugr-add-rate"]').setValue('0.85')
      await wrapper.find('[data-test="ugr-add-submit"]').trigger('click')
      await flushPromises()

      expect(state.applyBatch).toHaveBeenCalledWith([{ user_id: 5, group_id: 3, rate_multiplier: 0.85 }])
    } finally {
      vi.useRealTimers()
    }
  })
})
