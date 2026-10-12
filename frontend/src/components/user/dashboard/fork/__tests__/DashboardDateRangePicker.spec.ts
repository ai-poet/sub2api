import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { ref } from 'vue'

import DashboardDateRangePicker from '../DashboardDateRangePicker.vue'

const messages: Record<string, string> = {
  'dates.today': 'Today',
  'dates.yesterday': 'Yesterday',
  'dates.last24Hours': 'Last 24 Hours',
  'dates.last7Days': 'Last 7 Days',
  'dates.last14Days': 'Last 14 Days',
  'dates.last30Days': 'Last 30 Days',
  'dates.thisMonth': 'This Month',
  'dates.lastMonth': 'Last Month',
  'dates.apply': 'Apply',
  'dates.selectDateRange': 'Select date range',
  'dashboard.fork.datePicker.selectEndDate': 'Select an end date',
  'dashboard.fork.datePicker.previousMonth': 'Previous month',
  'dashboard.fork.datePicker.nextMonth': 'Next month',
  'common.cancel': 'Cancel',
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => messages[key] ?? key,
    locale: ref('en'),
  }),
}))

enableAutoUnmount(afterEach)

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const daysAgo = (days: number) => {
  const date = new Date()
  date.setDate(date.getDate() - days)
  return formatLocalDate(date)
}

const mountPicker = (startDate: string, endDate: string) => mount(DashboardDateRangePicker, {
  props: { startDate, endDate },
  attachTo: document.body,
  global: {
    stubs: {
      Icon: true,
      MotionTransition: { template: '<div><slot /></div>' },
    },
  },
})

const presetButton = (wrapper: ReturnType<typeof mountPicker>, label: string) =>
  wrapper.findAll('.date-picker-preset').find((node) => node.text() === label)!

const applyButton = (wrapper: ReturnType<typeof mountPicker>) => wrapper.get('.date-picker-actions .btn-primary')

describe('DashboardDateRangePicker', () => {
  it('快捷项按 TokenRouter 分组排列，不含分钟级快捷项', async () => {
    const wrapper = mountPicker(daysAgo(6), daysAgo(0))
    await wrapper.get('.input-trigger').trigger('click')

    const groups = wrapper.findAll('.date-picker-preset-group')
      .map((group) => group.findAll('.date-picker-preset').map((node) => node.text()))
    expect(groups).toEqual([
      ['Last 24 Hours'],
      ['Today', 'Yesterday'],
      ['Last 7 Days', 'Last 14 Days', 'Last 30 Days'],
      ['This Month', 'Last Month'],
    ])
    expect(wrapper.text()).not.toContain('dates.last15Minutes')
    expect(wrapper.text()).not.toContain('dates.last30Minutes')
  })

  it('按初始范围识别快捷项，选中的快捷项带勾', async () => {
    const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000)
    const wrapper = mountPicker(formatLocalDate(yesterday), formatLocalDate(new Date()))
    expect(wrapper.get('.input-trigger').text()).toBe('Last 24 Hours')

    await wrapper.get('.input-trigger').trigger('click')
    const active = wrapper.findAll('.date-picker-preset-active')
    expect(active).toHaveLength(1)
    expect(active[0].text()).toBe('Last 24 Hours')
    expect(active[0].find('icon-stub[name="check"]').exists()).toBe(true)
    expect(presetButton(wrapper, 'Last 7 Days').find('icon-stub[name="check"]').exists()).toBe(false)

    // 换选快捷项后勾跟着移动
    await presetButton(wrapper, 'Last 7 Days').trigger('click')
    expect(presetButton(wrapper, 'Last 7 Days').find('icon-stub[name="check"]').exists()).toBe(true)
    expect(presetButton(wrapper, 'Last 24 Hours').find('icon-stub[name="check"]').exists()).toBe(false)
  })

  it('选快捷项后点应用，change 带上起止日期和快捷项并收起面板', async () => {
    const wrapper = mountPicker('2026-01-10', '2026-01-12')
    await wrapper.get('.input-trigger').trigger('click')
    await presetButton(wrapper, 'Last 7 Days').trigger('click')
    // 底栏摘要显示待应用的区间
    expect(wrapper.get('.date-picker-summary').findAll('.date-picker-summary-value')).toHaveLength(2)
    await applyButton(wrapper).trigger('click')

    expect(wrapper.emitted('change')?.[0]).toEqual([{ startDate: daysAgo(6), endDate: daysAgo(0), preset: '7days' }])
    expect(wrapper.find('.date-picker-dropdown').exists()).toBe(false)
    expect(wrapper.get('.input-trigger').text()).toBe('Last 7 Days')
  })

  it('日历先点开始再点结束，反向点选自动对调，区间连成一条带', async () => {
    const wrapper = mountPicker('2026-01-10', '2026-01-12')
    await wrapper.get('.input-trigger').trigger('click')
    const dayButton = (day: number) => wrapper.findAll('.date-picker-day').find((node) => node.text() === String(day))!

    await dayButton(20).trigger('click')
    // 等待结束日期时不能应用，底栏提示选择结束日期
    expect(applyButton(wrapper).attributes('disabled')).toBeDefined()
    expect(wrapper.get('.date-picker-summary').text()).toContain('Select an end date')

    await dayButton(5).trigger('click')
    expect(wrapper.findAll('.date-picker-day-in-range')).toHaveLength(16)
    expect(wrapper.findAll('.date-picker-day-selected').map((node) => node.text())).toEqual(['5', '20'])

    await applyButton(wrapper).trigger('click')
    expect(wrapper.emitted('change')?.[0]).toEqual([{ startDate: '2026-01-05', endDate: '2026-01-20', preset: null }])
  })

  it('取消丢弃未应用的改动，触发器始终显示已生效的范围', async () => {
    const wrapper = mountPicker('2026-01-10', '2026-01-12')
    const triggerText = wrapper.get('.input-trigger').text()

    await wrapper.get('.input-trigger').trigger('click')
    await presetButton(wrapper, 'Last 30 Days').trigger('click')
    // 未应用前触发器仍显示原范围
    expect(wrapper.get('.input-trigger').text()).toBe(triggerText)
    expect(wrapper.get('.date-picker-preset-active').text()).toBe('Last 30 Days')

    const cancel = wrapper.findAll('.date-picker-actions .btn').find((node) => node.text() === 'Cancel')!
    await cancel.trigger('click')
    expect(wrapper.emitted('change')).toBeUndefined()
    expect(wrapper.find('.date-picker-dropdown').exists()).toBe(false)

    // 重新打开时回到原来的区间，没有快捷项被选中
    await wrapper.get('.input-trigger').trigger('click')
    expect(wrapper.find('.date-picker-preset-active').exists()).toBe(false)
    expect(wrapper.findAll('.date-picker-day-selected').map((node) => node.text())).toEqual(['10', '12'])
  })

  it('按 Esc 或点外部关闭时同样丢弃改动', async () => {
    const wrapper = mountPicker('2026-01-10', '2026-01-12')
    await wrapper.get('.input-trigger').trigger('click')
    await presetButton(wrapper, 'Today').trigger('click')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.date-picker-dropdown').exists()).toBe(false)

    await wrapper.get('.input-trigger').trigger('click')
    expect(wrapper.find('.date-picker-preset-active').exists()).toBe(false)
    document.body.click()
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.date-picker-dropdown').exists()).toBe(false)
    expect(wrapper.emitted('change')).toBeUndefined()
  })

  it('明天之后的日期和下个月不可选', async () => {
    const wrapper = mountPicker(daysAgo(0), daysAgo(0))
    expect(wrapper.get('.input-trigger').text()).toBe('Today')
    await wrapper.get('.input-trigger').trigger('click')

    const nav = wrapper.findAll('.date-picker-nav')
    expect(nav[1].attributes('aria-label')).toBe('Next month')
    // 月末最后一天时明天已在下个月，允许翻过去
    const tomorrow = new Date()
    tomorrow.setDate(tomorrow.getDate() + 1)
    if (tomorrow.getMonth() === new Date().getMonth()) {
      expect(nav[1].attributes('disabled')).toBeDefined()
    }
    const dayAfterTomorrow = new Date()
    dayAfterTomorrow.setDate(dayAfterTomorrow.getDate() + 2)
    if (dayAfterTomorrow.getMonth() === new Date().getMonth()) {
      const cell = wrapper.findAll('.date-picker-day').find((node) => node.text() === String(dayAfterTomorrow.getDate()))!
      expect(cell.attributes('disabled')).toBeDefined()
    }
  })

  it('父组件换算出新的起止日期后重新匹配快捷项', async () => {
    const wrapper = mountPicker('2026-01-10', '2026-01-12')
    await wrapper.setProps({ startDate: daysAgo(1), endDate: daysAgo(1) })
    expect(wrapper.get('.input-trigger').text()).toBe('Yesterday')
  })
})
