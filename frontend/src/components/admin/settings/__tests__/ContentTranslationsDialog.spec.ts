import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  updateItem: vi.fn(),
  deleteItem: vi.fn(),
  clearMachine: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin/contentTranslations', () => ({
  default: {
    list: mocks.list,
    updateItem: mocks.updateItem,
    deleteItem: mocks.deleteItem,
    clearMachine: mocks.clearMachine
  }
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess })
}))
vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_err: unknown, fallback: string) => fallback
}))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key)
  })
}))

import ContentTranslationsDialog from '../ContentTranslationsDialog.vue'

enableAutoUnmount(afterEach)

const DataTableStub = {
  props: ['columns', 'data', 'loading', 'rowKey'],
  template: `
    <table>
      <tr v-for="row in data" :key="row.id" data-test="row">
        <td v-for="col in columns" :key="col.key" :data-col="col.key">
          <slot :name="'cell-' + col.key" :row="row" :value="row[col.key]">{{ row[col.key] }}</slot>
        </td>
      </tr>
    </table>`
}

const ConfirmDialogStub = {
  props: ['show', 'title', 'message', 'danger'],
  emits: ['confirm', 'cancel'],
  template: `<div v-if="show" data-test="confirm"><button data-test="confirm-ok" @click="$emit('confirm')">ok</button></div>`
}

function item(id: number, extra: Record<string, unknown> = {}) {
  return {
    id,
    source_hash: `h${id}`,
    target_lang: 'en',
    source_text: `原文${id}`,
    translated_text: `Translation ${id}`,
    model: 'gpt-5.4-mini',
    manual: false,
    in_use: true,
    created_at: '2026-10-01T00:00:00Z',
    updated_at: '2026-10-01T00:00:00Z',
    last_seen_at: '2026-10-01T00:00:00Z',
    ...extra
  }
}

function mountDialog(show = true) {
  return mount(ContentTranslationsDialog, {
    props: { show },
    global: {
      stubs: {
        BaseDialog: { props: ['show', 'title', 'width'], template: '<div><slot /><slot name="footer" /></div>' },
        DataTable: DataTableStub,
        ConfirmDialog: ConfirmDialogStub,
        Pagination: true,
        Icon: true
      }
    }
  })
}

describe('ContentTranslationsDialog (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.list.mockResolvedValue({ items: [item(1), item(2, { manual: true, in_use: false })], total: 2, page: 1, page_size: 20, pages: 1 })
  })

  it('loads the first page when opened and renders the rows', async () => {
    const wrapper = mountDialog()
    await flushPromises()

    expect(mocks.list).toHaveBeenCalledWith({ lang: '', q: '', page: 1, page_size: 20 })
    const rows = wrapper.findAll('[data-test="row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('原文1')
    expect(rows[0].text()).toContain('Translation 1')
    expect(rows[0].find('[data-test="content-translations-manual"]').exists()).toBe(false)
    expect(rows[1].find('[data-test="content-translations-manual"]').exists()).toBe(true)
    expect(rows[1].text()).toContain('admin.settings.site.contentTranslation.dialog.notInUse')
  })

  it('does not load while closed', async () => {
    mountDialog(false)
    await flushPromises()
    expect(mocks.list).not.toHaveBeenCalled()
  })

  it('reloads with the language filter and the debounced search', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mountDialog()
      await flushPromises()

      await wrapper.get('[data-test="content-translations-lang"]').setValue('ja')
      await flushPromises()
      expect(mocks.list).toHaveBeenLastCalledWith({ lang: 'ja', q: '', page: 1, page_size: 20 })

      await wrapper.get('[data-test="content-translations-search"]').setValue('公告')
      expect(mocks.list).toHaveBeenCalledTimes(2)
      await vi.advanceTimersByTimeAsync(300)
      expect(mocks.list).toHaveBeenLastCalledWith({ lang: 'ja', q: '公告', page: 1, page_size: 20 })
    } finally {
      vi.useRealTimers()
    }
  })

  it('edits a translation inline and marks the row manual', async () => {
    mocks.updateItem.mockResolvedValue(item(1, { translated_text: 'Better', manual: true }))
    const wrapper = mountDialog()
    await flushPromises()

    await wrapper.findAll('[data-test="content-translations-edit"]')[0].trigger('click')
    await wrapper.get('[data-test="content-translations-edit-input"]').setValue('  Better  ')
    await wrapper.get('[data-test="content-translations-edit-save"]').trigger('click')
    await flushPromises()

    expect(mocks.updateItem).toHaveBeenCalledWith(1, 'Better')
    expect(mocks.showSuccess).toHaveBeenCalledWith('admin.settings.site.contentTranslation.dialog.updated')
    const row = wrapper.findAll('[data-test="row"]')[0]
    expect(row.text()).toContain('Better')
    expect(row.find('[data-test="content-translations-manual"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="content-translations-edit-input"]').exists()).toBe(false)
  })

  it('refuses to save an empty translation', async () => {
    const wrapper = mountDialog()
    await flushPromises()

    await wrapper.findAll('[data-test="content-translations-edit"]')[0].trigger('click')
    await wrapper.get('[data-test="content-translations-edit-input"]').setValue('   ')
    await wrapper.get('[data-test="content-translations-edit-save"]').trigger('click')
    await flushPromises()

    expect(mocks.updateItem).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith('admin.settings.site.contentTranslation.dialog.emptyTranslation')
  })

  it('deletes a translation after confirmation and reloads the page', async () => {
    mocks.deleteItem.mockResolvedValue(undefined)
    const wrapper = mountDialog()
    await flushPromises()
    expect(wrapper.find('[data-test="confirm"]').exists()).toBe(false)

    await wrapper.findAll('[data-test="content-translations-delete"]')[1].trigger('click')
    expect(mocks.deleteItem).not.toHaveBeenCalled()
    await wrapper.get('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()

    expect(mocks.deleteItem).toHaveBeenCalledWith(2)
    expect(mocks.list).toHaveBeenCalledTimes(2)
    expect(mocks.showSuccess).toHaveBeenCalledWith('admin.settings.site.contentTranslation.dialog.deleted')
  })

  it('clears machine translations after confirmation', async () => {
    mocks.clearMachine.mockResolvedValue({ deleted: 7 })
    const wrapper = mountDialog()
    await flushPromises()

    await wrapper.get('[data-test="content-translations-clear"]').trigger('click')
    await wrapper.get('[data-test="confirm-ok"]').trigger('click')
    await flushPromises()

    expect(mocks.clearMachine).toHaveBeenCalledTimes(1)
    expect(mocks.showSuccess).toHaveBeenCalledWith('admin.settings.site.contentTranslation.dialog.cleared:{"count":7}')
    expect(mocks.list).toHaveBeenCalledTimes(2)
  })
})
