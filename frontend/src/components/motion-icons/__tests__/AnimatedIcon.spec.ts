import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AnimatedIcon from '../AnimatedIcon.vue'
import { icons, type AnimatedIconName } from '../registry'
import { SIDEBAR_ANIMATED_ICONS } from '@/components/layout/sidebarIconMap'

// fork：移植自 TokenRouter components/icons/__tests__/Icon.spec.ts，按本仓库精简后的注册表调整。
describe('AnimatedIcon', () => {
  it('每个登记的图形都渲染成单个可见 SVG，装饰节点不进 Tab 顺序', () => {
    for (const name of Object.keys(icons) as AnimatedIconName[]) {
      const wrapper = mount(AnimatedIcon, { props: { name } })
      expect(wrapper.element.tagName.toLowerCase(), name).toBe('svg')
      expect(wrapper.findAll('svg'), name).toHaveLength(1)
      expect(
        wrapper.find('path, circle, rect, line, polyline, polygon, ellipse').exists(),
        name,
      ).toBe(true)
      expect(wrapper.html(), name).not.toContain('NaN')
      for (const node of wrapper.findAll('g, path, circle, rect, line, polyline, polygon, ellipse')) {
        expect(node.attributes('tabindex'), name).toBe('-1')
        expect(node.attributes('focusable'), name).toBe('false')
      }
      wrapper.unmount()
    }
  })

  it('侧栏映射引用的图标都已登记', () => {
    for (const [path, name] of Object.entries(SIDEBAR_ANIMATED_ICONS)) {
      expect(name in icons, path).toBe(true)
    }
  })

  it('默认是装饰图标，调用方传入的尺寸类优先', () => {
    const wrapper = mount(AnimatedIcon, {
      props: { name: 'key' },
      attrs: { class: 'h-5 w-5' },
    })
    expect(wrapper.attributes('aria-hidden')).toBe('true')
    expect(wrapper.attributes('focusable')).toBe('false')
    expect(wrapper.attributes('tabindex')).toBe('-1')
    expect(wrapper.attributes('stroke-width')).toBe('1.75')
    expect(wrapper.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
    expect(wrapper.classes()).not.toContain('h-[18px]')
    wrapper.unmount()
  })

  it('切换图形只替换内部节点，根节点和外层样式保持不变', async () => {
    const wrapper = mount(AnimatedIcon, {
      props: { name: 'sun' },
      attrs: { class: 'text-amber-500' },
    })
    const svg = wrapper.element
    await wrapper.setProps({ name: 'moon' })
    expect(wrapper.element).toBe(svg)
    expect(wrapper.attributes('data-animated-icon')).toBe('moon')
    expect(wrapper.classes()).toContain('text-amber-500')
    wrapper.unmount()
  })
})
