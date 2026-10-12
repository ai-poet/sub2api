import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'

import UserDashboardFilterDropdown from '../UserDashboardFilterDropdown.vue'
import UserDashboardFilterField from '../UserDashboardFilterField.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

enableAutoUnmount(afterEach)

const stubs = { Icon: true, MotionTransition: { template: '<div><slot /></div>' } }
const originalWidth = window.innerWidth

// 把触发区放在指定横坐标，模拟按钮位于页面左侧或右侧。
const openAt = async (left: number, columns: 1 | 2 | 3) => {
  const wrapper = mount(UserDashboardFilterDropdown, {
    props: { activeCount: 0, columns },
    global: { stubs },
  })
  vi.spyOn(wrapper.element, 'getBoundingClientRect').mockReturnValue({ left, right: left + 36, top: 0, bottom: 36, width: 36 } as DOMRect)
  await wrapper.get('button').trigger('click')
  return wrapper.get('.filter-panel')
}

describe('UserDashboardFilterDropdown', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: originalWidth })
  })

  it('三列面板宽 48rem、栅格三列，右侧放得下时左缘对齐触发按钮', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1024 })
    const panel = await openAt(100, 3)
    expect(panel.attributes('style')).toContain('width: 768px')
    expect(panel.attributes('style')).toContain('left: 0px')
    expect(panel.get('.filter-panel-body').classes()).toEqual(
      expect.arrayContaining(['grid-cols-1', 'sm:grid-cols-2', 'lg:grid-cols-3']),
    )
  })

  it('右侧放不下时面板向左平移并留出视口边距', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1024 })
    const panel = await openAt(900, 3)
    // 48rem = 768px，右缘停在 1024 - 16 = 1008px，相对触发器左移 900 - 240 = 660px。
    expect(panel.attributes('style')).toContain('left: -660px')
  })

  it('没有生效条件时禁用重置；有条件时按钮和标题旁显示数量，重置触发 reset', async () => {
    const wrapper = mount(UserDashboardFilterDropdown, { props: { activeCount: 0 }, global: { stubs } })
    await wrapper.get('button').trigger('click')
    expect(wrapper.find('.filter-trigger-count').exists()).toBe(false)
    expect(wrapper.find('.filter-panel-count').exists()).toBe(false)
    expect(wrapper.get('[data-testid="filter-reset"]').attributes('disabled')).toBeDefined()

    await wrapper.setProps({ activeCount: 2 })
    expect(wrapper.get('button').classes()).toContain('filter-trigger-active')
    expect(wrapper.get('.filter-trigger-count').text()).toBe('2')
    expect(wrapper.get('.filter-panel-count').text()).toBe('2')
    await wrapper.get('[data-testid="filter-reset"]').trigger('click')
    expect(wrapper.emitted('reset')).toHaveLength(1)
  })

  it('面板顶部按字段顺序列出已选条件，移除标签触发该字段的 clear', async () => {
    const Host = defineComponent({
      components: { UserDashboardFilterDropdown, UserDashboardFilterField },
      setup() {
        const keyword = ref('gpt')
        const brand = ref('')
        const range = ref('6 小时')
        return { keyword, brand, range }
      },
      template: `
        <UserDashboardFilterDropdown :active-count="2" :columns="3">
          <UserDashboardFilterField label="关键词" :value-text="keyword" @clear="keyword = ''"><input /></UserDashboardFilterField>
          <UserDashboardFilterField label="品牌" :value-text="brand" @clear="brand = ''"><input /></UserDashboardFilterField>
          <UserDashboardFilterField label="时间" :value-text="range" @clear="range = ''"><input /></UserDashboardFilterField>
        </UserDashboardFilterDropdown>
      `,
    })
    const wrapper = mount(Host, { attachTo: document.body, global: { stubs } })
    await wrapper.get('button[aria-haspopup="dialog"]').trigger('click')

    // 字段标签用面板专用的轻量样式
    expect(wrapper.findAll('label.filter-field-label').map((label) => label.text())).toEqual(['关键词', '品牌', '时间'])

    const chipTexts = () => wrapper.findAll('.filter-chip').map((chip) => chip.text())
    expect(chipTexts()).toEqual(['关键词gpt', '时间6 小时'])
    expect(wrapper.get('.filter-chip-remove').attributes('aria-label')).toBe('common.remove 关键词')

    await wrapper.findAll('.filter-chip-remove')[1]!.trigger('click')
    expect((wrapper.vm as unknown as { range: string }).range).toBe('')
    // 移除标签不会关闭面板
    expect(wrapper.find('.filter-panel').exists()).toBe(true)
    await wrapper.get('.filter-chip-remove').trigger('click')
    expect(wrapper.find('[data-testid="filter-chips"]').exists()).toBe(false)
  })

  it('按 Esc 或点外部关闭面板', async () => {
    const wrapper = mount(UserDashboardFilterDropdown, { props: { activeCount: 0 }, attachTo: document.body, global: { stubs } })
    await wrapper.get('button').trigger('click')
    await wrapper.get('.filter-panel').trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('.filter-panel').exists()).toBe(false)

    await wrapper.get('button').trigger('click')
    document.body.click()
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.filter-panel').exists()).toBe(false)
  })
})
