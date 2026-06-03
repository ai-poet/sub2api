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
})
