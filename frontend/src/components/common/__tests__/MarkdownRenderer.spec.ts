import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MarkdownRenderer from '../MarkdownRenderer.vue'

describe('MarkdownRenderer', () => {
  it('renders Markdown as structured HTML', () => {
    const wrapper = mount(MarkdownRenderer, {
      props: {
        content: '## 0.1.78\n\n#### Added\n\n- Added Cloud route',
      },
    })

    expect(wrapper.find('h2').text()).toBe('0.1.78')
    expect(wrapper.find('h4').text()).toBe('Added')
    expect(wrapper.find('ul').exists()).toBe(true)
    expect(wrapper.find('li').text()).toBe('Added Cloud route')
  })

  it('sanitizes unsafe HTML', () => {
    const wrapper = mount(MarkdownRenderer, {
      props: {
        content: '<img src=x onerror=alert(1)>',
      },
    })

    expect(wrapper.html()).not.toContain('onerror')
  })

  // blob: 只为工单图片附件放开（见 utils/markdown.ts）。默认路径必须继续拦掉它，
  // 否则这个口子就从"工单线程"漏到了每一处 Markdown 渲染。
  it('strips blob: image sources by default', () => {
    const wrapper = mount(MarkdownRenderer, {
      props: { content: '![x](blob:http://localhost/abc)' },
    })

    expect(wrapper.find('img').exists()).toBe(true)
    expect(wrapper.find('img').attributes('src')).toBeUndefined()
  })

  it('keeps blob: image sources when allowBlobImages is set', () => {
    const wrapper = mount(MarkdownRenderer, {
      props: { content: '![x](blob:http://localhost/abc)', allowBlobImages: true },
    })

    expect(wrapper.find('img').attributes('src')).toBe('blob:http://localhost/abc')
  })

  it('still rejects javascript: sources even with allowBlobImages', () => {
    const wrapper = mount(MarkdownRenderer, {
      props: { content: '![x](javascript:alert(1))', allowBlobImages: true },
    })

    expect(wrapper.html()).not.toContain('javascript:')
  })
})
