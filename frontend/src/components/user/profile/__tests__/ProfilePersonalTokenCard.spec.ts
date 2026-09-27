import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfilePersonalTokenCard from '@/components/user/profile/ProfilePersonalTokenCard.vue'

const { getStatusMock, generateMock, revokeMock, showSuccessMock, showErrorMock, copyMock } = vi.hoisted(() => ({
  getStatusMock: vi.fn(),
  generateMock: vi.fn(),
  revokeMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  copyMock: vi.fn()
}))

vi.mock('@/api/personalToken', () => ({
  personalTokenAPI: {
    getStatus: getStatusMock,
    generate: generateMock,
    revoke: revokeMock
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: showSuccessMock, showError: showErrorMock })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: copyMock, copied: { value: false } })
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => `fmt(${value})`
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key)
    })
  }
})

const PLAINTEXT = 'pat-' + 'ab'.repeat(32)

const tokenInfo = {
  hint: 'pat-ababab...abab',
  created_at: '2026-09-27T00:00:00Z',
  expires_at: null,
  expired: false,
  last_used_at: null,
  last_used_ip: ''
}

function status(overrides: Record<string, unknown> = {}) {
  return {
    feature_enabled: true,
    eligible: true,
    token: null,
    expiry_options: [0, 30, 90, 180, 365],
    ...overrides
  }
}

function mountCard() {
  return mount(ProfilePersonalTokenCard, {
    global: {
      stubs: {
        Icon: true,
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show" data-test="generate-dialog"><slot /><slot name="footer" /></div>'
        },
        ConfirmDialog: {
          props: ['show', 'title', 'message'],
          emits: ['confirm', 'cancel'],
          template: '<div v-if="show" data-test="confirm-dialog"><button data-test="confirm-yes" @click="$emit(\'confirm\')" /></div>'
        }
      }
    }
  })
}

describe('ProfilePersonalTokenCard', () => {
  beforeEach(() => {
    getStatusMock.mockReset()
    generateMock.mockReset()
    revokeMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    copyMock.mockReset()
  })

  it('cannot generate while the administrator has the feature off', async () => {
    getStatusMock.mockResolvedValue(status({ feature_enabled: false }))
    const wrapper = mountCard()
    await flushPromises()

    expect(wrapper.find('[data-test="personal-token-disabled"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="personal-token-generate"]').exists()).toBe(false)
  })

  it('shows the plaintext exactly once after generating with the current password', async () => {
    getStatusMock.mockResolvedValue(status())
    generateMock.mockResolvedValue({ token: PLAINTEXT, info: tokenInfo })
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="personal-token-generate"]').trigger('click')
    expect(wrapper.find('[data-test="generate-dialog"]').exists()).toBe(true)
    await wrapper.get('[data-test="personal-token-password"]').setValue('secret-pw')
    await wrapper.get('#personal-token-generate-form').trigger('submit')
    await flushPromises()

    expect(generateMock).toHaveBeenCalledWith({ password: 'secret-pw', expires_in_days: 90 })
    const plaintext = wrapper.get('[data-test="personal-token-plaintext"]')
    expect(plaintext.text()).toContain(PLAINTEXT)
    expect(plaintext.text()).toContain(`Authorization: Bearer ${PLAINTEXT}`)
    expect(wrapper.find('[data-test="generate-dialog"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="personal-token-current"]').text()).toContain(tokenInfo.hint)

    // 确认已保存后明文消失，只剩提示串
    const done = plaintext.findAll('button').find((b) => b.text() === 'operator.personalToken.done')
    await done!.trigger('click')
    expect(wrapper.find('[data-test="personal-token-plaintext"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain(PLAINTEXT)
  })

  it('keeps the dialog open and shows no plaintext when generation fails', async () => {
    getStatusMock.mockResolvedValue(status())
    generateMock.mockRejectedValue({ message: 'current password is incorrect' })
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="personal-token-generate"]').trigger('click')
    await wrapper.get('[data-test="personal-token-password"]').setValue('wrong')
    await wrapper.get('#personal-token-generate-form').trigger('submit')
    await flushPromises()

    expect(showErrorMock).toHaveBeenCalled()
    expect(wrapper.find('[data-test="personal-token-plaintext"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="generate-dialog"]').exists()).toBe(true)
  })

  it('revokes only after confirmation', async () => {
    getStatusMock.mockResolvedValue(status({ token: tokenInfo }))
    revokeMock.mockResolvedValue({ revoked: true })
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-test="personal-token-revoke"]').trigger('click')
    expect(revokeMock).not.toHaveBeenCalled()
    await wrapper.get('[data-test="confirm-yes"]').trigger('click')
    await flushPromises()

    expect(revokeMock).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-test="personal-token-current"]').exists()).toBe(false)
    expect(showSuccessMock).toHaveBeenCalledWith('operator.personalToken.revoked')
  })
})
