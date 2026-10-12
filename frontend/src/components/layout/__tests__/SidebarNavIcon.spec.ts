import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import SidebarNavIcon from '../SidebarNavIcon.vue'
import { SIDEBAR_ANIMATED_ICONS } from '../sidebarIconMap'

const StaticIcon = defineComponent({ render: () => h('svg', { 'data-static': 'true' }) })

describe('SidebarNavIcon', () => {
  it('keeps the static icon for paths without an animated icon', async () => {
    const wrapper = mount(SidebarNavIcon, { props: { path: '/custom/page', fallback: StaticIcon } })
    await flushPromises()
    expect(wrapper.attributes('data-sidebar-icon')).toBe('static')
    expect(wrapper.find('[data-static]').exists() || wrapper.attributes('data-static') === 'true').toBe(true)
  })

  it('shows the static icon first, then swaps in the animated icon once its chunk loads', async () => {
    const wrapper = mount(SidebarNavIcon, {
      props: { path: '/keys', fallback: StaticIcon },
      attrs: { class: 'h-5 w-5' },
    })
    // 第一帧还没加载完，显示原有图标，布局不跳
    expect(wrapper.attributes('data-sidebar-icon')).toBe('static')

    // 动态 import 在测试里首次要转译 motion-v，给足时间
    await vi.waitFor(() => {
      expect(wrapper.attributes('data-sidebar-icon')).toBe('animated')
    }, { timeout: 10000, interval: 20 })
    expect(wrapper.attributes('data-animated-icon')).toBe('key')
    expect(wrapper.classes()).toEqual(expect.arrayContaining(['h-5', 'w-5']))
  })
})

describe('sidebar animated icon map', () => {
  it('covers every route path declared in AppSidebar', () => {
    const source = readFileSync(resolve(__dirname, '../AppSidebar.vue'), 'utf8')
    const paths = [...source.matchAll(/path: '(\/[^']*)'/g)].map(match => match[1])
    expect(paths.length).toBeGreaterThan(20)
    const missing = [...new Set(paths)].filter(path => !(path in SIDEBAR_ANIMATED_ICONS))
    expect(missing).toEqual([])
  })
})
