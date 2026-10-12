import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import { mockMotionEnvironment } from '@/__tests__/helpers/motion'
import AnimatedNumber from '../AnimatedNumber.vue'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('AnimatedNumber', () => {
  it('动画文字对读屏隐藏，读屏只读格式化后的最终值', () => {
    mockMotionEnvironment(true)
    const wrapper = mount(AnimatedNumber, {
      props: { value: 1234.5, format: (value: number) => `${value.toFixed(1)}M` },
    })
    const [animated, final] = wrapper.findAll('span span')
    expect(animated.attributes('aria-hidden')).toBe('true')
    expect(final.classes()).toContain('sr-only')
    expect(final.text()).toBe('1234.5M')
    expect(animated.text()).toBe('1234.5M')
  })

  it('from 存在时首屏从 from 开始滚动，读屏一开始就读到目标值', async () => {
    mockMotionEnvironment(false)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation(() => 1)
    const wrapper = mount(AnimatedNumber, { props: { value: 500, from: 0 } })
    await nextTick()
    const [animated, final] = wrapper.findAll('span span')
    expect(animated.text()).toBe('0')
    expect(final.text()).toBe('500')
  })
})
