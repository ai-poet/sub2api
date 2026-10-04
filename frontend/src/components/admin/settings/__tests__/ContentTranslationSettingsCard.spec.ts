import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  testConfig: vi.fn(),
  getStatus: vi.fn(),
  sync: vi.fn(),
  listKeys: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin/contentTranslations', () => ({
  default: {
    getConfig: mocks.getConfig,
    updateConfig: mocks.updateConfig,
    testConfig: mocks.testConfig,
    getStatus: mocks.getStatus,
    sync: mocks.sync
  }
}))
vi.mock('@/api/keys', () => ({ keysAPI: { list: mocks.listKeys } }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess })
}))
vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_err: unknown, fallback: string) => fallback
}))
vi.mock('@/utils/format', () => ({ formatDateTime: (value: string) => `at(${value})` }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key)
  })
}))

import ContentTranslationSettingsCard from '../ContentTranslationSettingsCard.vue'

enableAutoUnmount(afterEach)

const ToggleStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: `<button type="button" data-test="toggle" :data-on="String(modelValue)" @click="$emit('update:modelValue', !modelValue)" />`
}

const config = {
  enabled: true,
  api_key_id: 12,
  model: 'gpt-5.4-mini',
  base_url: '',
  languages: ['en', 'ja'],
  api_key_name: 'translator'
}

const status = {
  enabled: true,
  running: false,
  sources: 42,
  translated: 80,
  pending: 4,
  last_run_at: '2026-10-04T08:00:00Z',
  last_error: 'upstream 502',
  last_error_at: '2026-10-04T08:01:00Z'
}

const stubs = { Toggle: ToggleStub, ContentTranslationsDialog: true }

function mountCard() {
  return mount(ContentTranslationSettingsCard, { global: { stubs } })
}

/** 像 SettingsView 那样把卡片放进一张会被「保存设置」提交的表单里 */
function mountCardInsideForm() {
  return mount(
    {
      components: { ContentTranslationSettingsCard },
      template: `<form data-test="host-form" @submit.prevent><ContentTranslationSettingsCard /></form>`
    },
    { global: { stubs } }
  )
}

describe('ContentTranslationSettingsCard (fork)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getConfig.mockResolvedValue({ ...config })
    mocks.getStatus.mockResolvedValue({ ...status })
    mocks.updateConfig.mockImplementation(async (payload) => ({ ...payload, api_key_name: 'translator' }))
    mocks.listKeys.mockResolvedValue({
      items: [
        { id: 12, name: 'translator', key: 'sk-abcdef1234567890wxyz', status: 'active' },
        { id: 13, name: 'old', key: 'sk-0000001111112222', status: 'inactive' }
      ],
      total: 2,
      page: 1,
      page_size: 100,
      pages: 1
    })
  })

  it('loads the config, the admin keys and the status', async () => {
    const wrapper = mountCard()
    await flushPromises()

    expect(mocks.listKeys).toHaveBeenCalledWith(1, 100, { sort_by: 'created_at', sort_order: 'desc' })
    const select = wrapper.get('[data-test="content-translation-key"]').element as HTMLSelectElement
    const options = Array.from(select.options).map((option) => option.textContent?.trim())
    expect(options).toContain('translator (sk-abc...wxyz)')
    expect(options).toContain('old (sk-000...2222) · admin.settings.site.contentTranslation.apiKeyInactive')
    expect(select.value).toBe('12')

    expect((wrapper.get('[data-test="content-translation-model"]').element as HTMLInputElement).value).toBe('gpt-5.4-mini')
    expect((wrapper.get('[data-test="content-translation-lang-zh"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-test="content-translation-lang-en"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-test="content-translation-lang-ja"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.get('[data-test="content-translation-enabled"]').attributes('data-on')).toBe('true')

    const statusText = wrapper.get('[data-test="content-translation-status"]').text()
    expect(statusText).toContain('admin.settings.site.contentTranslation.status.enabled')
    expect(statusText).toContain('"sources":42')
    expect(statusText).toContain('"pending":4')
    expect(statusText).toContain('at(2026-10-04T08:00:00Z)')
    expect(wrapper.get('[data-test="content-translation-last-error"]').text()).toContain('upstream 502')
    // 刚加载完没有未保存改动
    expect(wrapper.find('[data-test="content-translation-unsaved"]').exists()).toBe(false)
  })

  it('shows the saved key by name when it is no longer in the list', async () => {
    mocks.getConfig.mockResolvedValue({ ...config, api_key_id: 99, api_key_name: 'gone' })
    const wrapper = mountCard()
    await flushPromises()

    const select = wrapper.get('[data-test="content-translation-key"]').element as HTMLSelectElement
    expect(select.value).toBe('99')
    expect(select.selectedOptions[0].textContent?.trim()).toBe('gone · admin.settings.site.contentTranslation.apiKeyMissing')
  })

  it('saves the edited config on its own and shows the unsaved hint until then', async () => {
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-model"]').setValue('  gpt-5.5-mini ')
    await wrapper.get('[data-test="content-translation-base-url"]').setValue('https://api.example.com')
    await wrapper.get('[data-test="content-translation-lang-zh"]').setValue(true)
    expect(wrapper.find('[data-test="content-translation-unsaved"]').exists()).toBe(true)

    await wrapper.get('[data-test="content-translation-save"]').trigger('click')
    await flushPromises()

    expect(mocks.updateConfig).toHaveBeenCalledWith({
      enabled: true,
      api_key_id: 12,
      model: 'gpt-5.5-mini',
      base_url: 'https://api.example.com',
      languages: ['zh', 'en', 'ja']
    })
    expect(mocks.showSuccess).toHaveBeenCalledWith('admin.settings.site.contentTranslation.saved')
    expect(wrapper.find('[data-test="content-translation-unsaved"]').exists()).toBe(false)
    // 保存后刷新一次状态
    expect(mocks.getStatus).toHaveBeenCalledTimes(2)
  })

  it('saves at once when the switch is flipped', async () => {
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-enabled"]').trigger('click')
    await flushPromises()

    expect(mocks.updateConfig).toHaveBeenCalledTimes(1)
    expect(mocks.updateConfig).toHaveBeenCalledWith(expect.objectContaining({ enabled: false, api_key_id: 12 }))
    expect(wrapper.get('[data-test="content-translation-enabled"]').attributes('data-on')).toBe('false')
    expect(wrapper.find('[data-test="content-translation-unsaved"]').exists()).toBe(false)
  })

  it('flips the switch back when saving it fails', async () => {
    mocks.updateConfig.mockRejectedValue(new Error('boom'))
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-enabled"]').trigger('click')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('admin.settings.site.contentTranslation.saveFailed')
    expect(wrapper.get('[data-test="content-translation-enabled"]').attributes('data-on')).toBe('true')
  })

  it('refuses to enable without an API key and leaves the switch off', async () => {
    mocks.getConfig.mockResolvedValue({ ...config, enabled: false, api_key_id: null, api_key_name: '' })
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-enabled"]').trigger('click')
    await flushPromises()

    expect(mocks.updateConfig).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith('admin.settings.site.contentTranslation.apiKeyRequired')
    expect(wrapper.get('[data-test="content-translation-enabled"]').attributes('data-on')).toBe('false')
  })

  it('also saves pending edits when the page form around it is submitted', async () => {
    const wrapper = mountCardInsideForm()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-model"]').setValue('gpt-5.5-mini')
    await wrapper.get('[data-test="host-form"]').trigger('submit')
    await flushPromises()

    expect(mocks.updateConfig).toHaveBeenCalledTimes(1)
    expect(mocks.updateConfig).toHaveBeenCalledWith(expect.objectContaining({ model: 'gpt-5.5-mini' }))

    // 没有改动时提交表单不会再发请求
    await wrapper.get('[data-test="host-form"]').trigger('submit')
    await flushPromises()
    expect(mocks.updateConfig).toHaveBeenCalledTimes(1)
  })

  it('tests the current form and shows the sample translation with its latency', async () => {
    mocks.testConfig.mockResolvedValue({ translated: 'Group description', latency_ms: 1234 })
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-test"]').trigger('click')
    await flushPromises()

    expect(mocks.testConfig).toHaveBeenCalledWith({
      enabled: true,
      api_key_id: 12,
      model: 'gpt-5.4-mini',
      base_url: '',
      languages: ['en', 'ja']
    })
    const result = wrapper.get('[data-test="content-translation-test-result"]').text()
    expect(result).toContain('Group description')
    expect(result).toContain('"ms":1234')
  })

  it('starts a sync and shows the returned status', async () => {
    mocks.sync.mockResolvedValue({ ...status, running: true, pending: 9, last_error: '' })
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-sync"]').trigger('click')
    await flushPromises()

    expect(mocks.sync).toHaveBeenCalledTimes(1)
    expect(mocks.showSuccess).toHaveBeenCalledWith('admin.settings.site.contentTranslation.syncStarted')
    const statusText = wrapper.get('[data-test="content-translation-status"]').text()
    expect(statusText).toContain('"pending":9')
    expect(statusText).toContain('admin.settings.site.contentTranslation.status.running')
  })

  it('warns instead of celebrating when a sync runs with translation disabled on the server', async () => {
    mocks.sync.mockResolvedValue({ ...status, enabled: false, running: false })
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="content-translation-sync"]').trigger('click')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('admin.settings.site.contentTranslation.syncDisabled')
    expect(mocks.showSuccess).not.toHaveBeenCalledWith('admin.settings.site.contentTranslation.syncStarted')
    expect(wrapper.get('[data-test="content-translation-status-enabled"]').text()).toBe(
      'admin.settings.site.contentTranslation.status.disabled'
    )
  })
})
