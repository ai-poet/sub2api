import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserCreateModal from '../UserCreateModal.vue'

const { create, showSuccess, showError } = vi.hoisted(() => ({
  create: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { users: { create } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

const mountModal = () => mount(UserCreateModal, {
  props: { show: true },
  global: {
    stubs: {
      BaseDialog: {
        props: ['show', 'title'],
        template: '<div v-if="show"><slot /><slot name="footer" /></div>'
      },
      Icon: true,
      TotpStepUpDialog: true
    }
  }
})

const fillRequiredFields = async (wrapper: ReturnType<typeof mountModal>) => {
  await wrapper.get('input[type="email"]').setValue('new-user@example.test')
  await wrapper.get('input[type="text"]').setValue('strong-pass')
}

describe('UserCreateModal concurrency', () => {
  beforeEach(() => {
    create.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    create.mockResolvedValue({})
  })

  it('omits concurrency while following the global default', async () => {
    const wrapper = mountModal()
    await fillRequiredFields(wrapper)

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(create).toHaveBeenCalledOnce()
    expect(create.mock.calls[0][0]).not.toHaveProperty('concurrency')
    expect(create.mock.calls[0][0]).not.toHaveProperty('use_default_concurrency')
  })

  it('submits an explicit zero after switching to a custom override', async () => {
    const wrapper = mountModal()
    await fillRequiredFields(wrapper)
    await wrapper.get('[role="switch"]').trigger('click')
    await wrapper.get('[data-test="concurrency-input"]').setValue('0')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(create).toHaveBeenCalledWith(expect.objectContaining({ concurrency: 0 }))
  })
})
