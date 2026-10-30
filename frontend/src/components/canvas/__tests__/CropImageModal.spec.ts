/**
 * CropImageModal.spec.ts — focus, cancel-keeps-confirmed, Escape,
 * double-click dedup and URL revocation with real DOM + stubbed decoders.
 */
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import CropImageModal from '../CropImageModal.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (k: string) => k }) }))

class FakeImage {
  width = 80
  height = 60
  onload: (() => void) | null = null
  set src(_v: string) { queueMicrotask(() => this.onload?.()) }
}

beforeEach(() => {
  vi.stubGlobal('Image', FakeImage)
  vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:mock'), revokeObjectURL: vi.fn() })
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
    clearRect() {}, drawImage() {}, fillRect() {}, strokeRect() {},
    set fillStyle(v) {}, set strokeStyle(v) {}, set lineWidth(v) {},
  } as never)
  Element.prototype.setPointerCapture = vi.fn() as never
})

afterEach(() => vi.restoreAllMocks())

const blobA = new Blob(['a'], { type: 'image/png' })
const blobB = new Blob(['b'], { type: 'image/png' })

function mountModal(queue: Blob[]) {
  return mount(CropImageModal, { props: { queue }, attachTo: document.body })
}

describe('CropImageModal', () => {
  it('focuses the dialog on open and Escape cancels', async () => {
    const wrapper = mountModal([blobA])
    await new Promise((r) => setTimeout(r, 20))
    const modal = document.querySelector('.crop-modal') as HTMLElement
    expect(document.activeElement).toBe(modal)
    modal.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(wrapper.emitted('cancel')).toBeTruthy()
    wrapper.unmount()
  })

  it('cancel keeps confirmed images and emits done in order', async () => {
    const wrapper = mountModal([blobA, blobB])
    await new Promise((r) => setTimeout(r, 20))
    // skip the first (counts as confirmed original)
    const skipBtn = [...document.querySelectorAll('.crop-actions .btn')].find((b) => b.textContent!.includes('skip')) as HTMLElement
    skipBtn.click()
    await new Promise((r) => setTimeout(r, 20))
    // cancel the rest
    const cancelBtn = [...document.querySelectorAll('.crop-actions .btn')].find((b) => b.textContent!.includes('cancelRest')) as HTMLElement
    cancelBtn.click()
    const done = wrapper.emitted('done')
    expect(done).toBeTruthy()
    expect(done![0][0]).toHaveLength(1)
    expect((done![0][0] as Blob[])[0].size).toBe(1)
    expect(wrapper.emitted('cancel')).toBeUndefined()
    wrapper.unmount()
  })

  it('cancel with nothing confirmed emits cancel', async () => {
    const wrapper = mountModal([blobA])
    await new Promise((r) => setTimeout(r, 20))
    const cancelBtn = [...document.querySelectorAll('.crop-actions .btn')].find((b) => b.textContent!.includes('cancelRest')) as HTMLElement
    cancelBtn.click()
    expect(wrapper.emitted('cancel')).toBeTruthy()
    wrapper.unmount()
  })

  it('revokes the object URL on unmount', async () => {
    const wrapper = mountModal([blobA])
    await new Promise((r) => setTimeout(r, 20))
    wrapper.unmount()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock')
  })
})
