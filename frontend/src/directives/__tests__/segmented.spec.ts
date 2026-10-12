import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import { vSegmented } from '../segmented'

let wrapper: VueWrapper | undefined
const observers: ResizeObserverMock[] = []

class ResizeObserverMock {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()

  constructor(readonly notify: () => void) {
    observers.push(this)
  }
}

const Fixture = defineComponent({
  directives: { segmented: vSegmented },
  props: {
    active: { type: Number, default: 0 },
    count: { type: Number, default: 3 },
  },
  template: `
    <div v-segmented class="segmented">
      <button v-for="(_, index) in count" :key="index"
        class="segmented-item" :class="{ 'segmented-item-active': active === index }"
        :data-x="index * 80 + 2" data-y="2" :data-width="80 + index * 20" data-height="28"
        :aria-pressed="active === index">option {{ index }}</button>
    </div>`,
})

beforeEach(() => {
  observers.length = 0
  vi.stubGlobal('ResizeObserver', ResizeObserverMock)
  // jsdom 不做布局，通过几何输入覆盖不等宽选项和换行后的坐标。
  for (const [property, field] of Object.entries({
    offsetLeft: 'x', offsetTop: 'y', offsetWidth: 'width', offsetHeight: 'height',
  })) {
    vi.spyOn(HTMLElement.prototype, property as 'offsetLeft', 'get').mockImplementation(function (this: HTMLElement) {
      return Number(this.dataset[field] || 0)
    })
  }
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('vSegmented', () => {
  it('首次直接定位，切换到不同宽度的选项时不替换按钮', async () => {
    wrapper = mount(Fixture)
    const root = wrapper.element as HTMLElement
    const buttons = wrapper.findAll('button').map(button => button.element)
    expect(root.hasAttribute('data-segmented-ready')).toBe(true)
    expect(root.style.getPropertyValue('--segmented-x')).toBe('2px')
    expect(root.style.getPropertyValue('--segmented-width')).toBe('80px')

    await wrapper.setProps({ active: 1 })
    expect(root.style.getPropertyValue('--segmented-x')).toBe('82px')
    expect(root.style.getPropertyValue('--segmented-width')).toBe('100px')
    expect(wrapper.findAll('button').map(button => button.element)).toEqual(buttons)
  })

  it('尺寸变化后更新位置，隐藏时移除背景、重现后恢复', () => {
    wrapper = mount(Fixture)
    const root = wrapper.element as HTMLElement
    const selected = wrapper.get('button').element as HTMLElement
    selected.dataset.y = '34'
    selected.dataset.width = '130'
    observers[0].notify()
    expect(root.style.getPropertyValue('--segmented-y')).toBe('34px')
    expect(root.style.getPropertyValue('--segmented-width')).toBe('130px')

    selected.dataset.width = '0'
    observers[0].notify()
    expect(root.hasAttribute('data-segmented-ready')).toBe(false)
    selected.dataset.width = '96'
    observers[0].notify()
    expect(root.hasAttribute('data-segmented-ready')).toBe(true)
  })

  it('没有选中项时隐藏背景，卸载会清理观察器和变量', async () => {
    wrapper = mount(Fixture, { props: { active: -1 } })
    const root = wrapper.element as HTMLElement
    expect(root.hasAttribute('data-segmented-ready')).toBe(false)
    const removed = wrapper.findAll('button')[2].element
    await wrapper.setProps({ count: 2, active: 1 })
    expect(observers[0].unobserve).toHaveBeenCalledWith(removed)

    wrapper.unmount()
    wrapper = undefined
    expect(observers[0].disconnect).toHaveBeenCalledOnce()
    expect(root.hasAttribute('data-segmented-ready')).toBe(false)
    expect(root.style.getPropertyValue('--segmented-x')).toBe('')
  })
})
