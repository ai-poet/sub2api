import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, ref } from 'vue'

import UserDashboardHeatmap from '../UserDashboardHeatmap.vue'
import { provideUsageChartState, type UsageChartState } from '../usageChartState'
import { usageAPI } from '@/api/usage'
import { formatDateLocalInput } from '@/utils/format'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${Object.values(params).join(',')}` : key),
      locale: { value: 'en' },
    }),
  }
})

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardTrend: vi.fn(),
    getDashboardModels: vi.fn().mockResolvedValue({ models: [], start_date: '', end_date: '' }),
  },
}))

// usage_logs 默认只保留 90 天，热力图最多显示 26 周：今天往前 25 周再对齐到周日
const MAX_WEEKS = 26
const FETCH_DAYS = (MAX_WEEKS - 1) * 7

const expectedRange = () => {
  const end = new Date()
  const start = new Date(end)
  start.setDate(start.getDate() - FETCH_DAYS)
  start.setDate(start.getDate() - start.getDay())
  return { start, end }
}

// 热力图读取页面提供的用量状态，用一层外壳提供并暴露给断言。
let usageState: UsageChartState
const selectedDays: string[] = []
const heatmapRef = ref<{ reload: () => Promise<void> } | null>(null)
const Harness = defineComponent({
  setup() {
    usageState = provideUsageChartState()
    return () => h(UserDashboardHeatmap, { ref: heatmapRef, onSelectDay: (date: string) => selectedDays.push(date) })
  },
})

const mountHeatmap = async () => {
  const wrapper = mount(Harness)
  await flushPromises()
  return wrapper
}

const emptyTrend = {
  trend: [],
  start_date: '',
  end_date: '',
  granularity: 'day',
}

const trendPoint = (date: string, tokens = 30) => ({
  date,
  requests: 3,
  input_tokens: 10,
  output_tokens: 20,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  total_tokens: tokens,
  cost: 0.1,
  actual_cost: 0.2,
})

const shiftDays = (base: Date, days: number) => {
  const date = new Date(base)
  date.setDate(date.getDate() + days)
  return formatDateLocalInput(date)
}

describe('UserDashboardHeatmap', () => {
  beforeEach(() => {
    vi.mocked(usageAPI.getDashboardTrend).mockReset()
    selectedDays.length = 0
    localStorage.clear()
  })

  it('首次请求未返回时也有占位格，成功后替换为日期数据', async () => {
    let resolveTrend!: (value: typeof emptyTrend) => void
    vi.mocked(usageAPI.getDashboardTrend).mockImplementationOnce(() => new Promise(resolve => { resolveTrend = resolve }))
    const wrapper = await mountHeatmap()
    expect(wrapper.findAll('[data-testid="heatmap-skeleton-cell"]').length).toBe(MAX_WEEKS * 7)
    expect(wrapper.find('[data-testid="heatmap-cell"]').exists()).toBe(false)
    expect(wrapper.attributes('aria-busy')).toBe('true')

    resolveTrend(emptyTrend)
    await flushPromises()
    expect(wrapper.find('[data-testid="heatmap-skeleton-cell"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="heatmap-cell"]').exists()).toBe(true)
    expect(wrapper.attributes('aria-busy')).toBe('false')
    wrapper.unmount()
  })

  it('按 26 个整周的范围请求按日趋势数据', async () => {
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    await mountHeatmap()

    const { start, end } = expectedRange()
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledWith({
      start_date: formatDateLocalInput(start),
      end_date: formatDateLocalInput(end),
      granularity: 'day',
    })
  })

  it('渲染 26 周网格，接口未返回的日期补零为无用量档', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [trendPoint(today)],
    })

    const wrapper = await mountHeatmap()

    const cells = wrapper.findAll('[data-testid="heatmap-cell"]')
    expect(cells.length).toBe(MAX_WEEKS * 7)

    // 有用量的今天为最高档且只有一天；未来占位格不可见
    const greenCells = cells.filter((c) => c.classes().includes('bg-green-700'))
    expect(greenCells.length).toBe(1)
    const futureCount = cells.filter((c) => c.classes().includes('invisible')).length
    expect(cells.filter((c) => c.classes().includes('bg-gray-100')).length).toBe(cells.length - futureCount - 1)
  })

  it('按非零用量的四分位分档', async () => {
    const { end } = expectedRange()
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [
        trendPoint(shiftDays(end, 0), 4000),
        trendPoint(shiftDays(end, -1), 100),
        trendPoint(shiftDays(end, -2), 2000),
        trendPoint(shiftDays(end, -3), 3000),
      ],
    })
    const wrapper = await mountHeatmap()
    expect(wrapper.get(`[data-date="${shiftDays(end, 0)}"]`).attributes('data-level')).toBe('4')
    expect(wrapper.get(`[data-date="${shiftDays(end, -1)}"]`).attributes('data-level')).toBe('1')
    expect(wrapper.get(`[data-date="${shiftDays(end, -10)}"]`).attributes('data-level')).toBe('0')
  })

  it('格子按列错峰入场，最晚一列不超过 480ms', async () => {
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)
    const wrapper = await mountHeatmap()
    const delays = wrapper.findAll('[data-testid="heatmap-cell"]')
      .map(cell => parseInt((cell.attributes('style') ?? '').match(/--wave-delay:\s*(\d+)ms/)?.[1] ?? '0', 10))
    expect(Math.max(...delays)).toBeLessThanOrEqual(480)
    expect(Math.max(...delays)).toBeGreaterThan(0)
  })

  it('汇总活跃天数、Token 合计、用量最多的一天和连续使用天数', async () => {
    const { end } = expectedRange()
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [
        trendPoint(shiftDays(end, 0), 1000),
        trendPoint(shiftDays(end, -1), 3000),
        trendPoint(shiftDays(end, -5), 500),
      ],
    })
    const wrapper = await mountHeatmap()
    expect(wrapper.get('[data-testid="heatmap-summary-activeDays"]').text()).toBe('dashboard.fork.heatmap.summary.days:3')
    expect(wrapper.get('[data-testid="heatmap-summary-tokens"]').text()).toBe('4.5K')
    expect(wrapper.get('[data-testid="heatmap-summary-busiest"]').text()).toContain('3.0K')
    expect(wrapper.get('[data-testid="heatmap-summary-streak"]').text()).toBe('dashboard.fork.heatmap.summary.days:2')
  })

  it('悬停格子时显示当天用量，无用量日期显示无用量', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [trendPoint(today)],
    })

    const wrapper = await mountHeatmap()
    const cells = wrapper.findAll('[data-testid="heatmap-cell"]')
    const todayIndex = cells.findIndex((c) => c.classes().includes('bg-green-700'))
    expect(todayIndex).toBeGreaterThan(0)

    await cells[todayIndex].trigger('mouseenter')
    const tooltip = wrapper.get('[data-testid="heatmap-tooltip"]')
    expect(tooltip.text()).toContain('dashboard.fork.heatmap.requests')
    expect(tooltip.text()).toContain('dashboard.fork.heatmap.tokens')
    expect(tooltip.text()).toContain('dashboard.fork.heatmap.cost')
    expect(tooltip.text()).toContain('$0.2000')
    expect(tooltip.attributes('aria-hidden')).toBe('false')
    expect(tooltip.attributes('style')).toContain('opacity var(--motion-fast) var(--motion-ease)')
    expect(tooltip.attributes('style')).not.toMatch(/transition:.*(?:width|left) /)

    await wrapper.get('[data-testid="heatmap-grid-wrap"]').trigger('mouseleave')
    expect(wrapper.get('[data-testid="heatmap-tooltip"]').attributes('aria-hidden')).toBe('true')

    // 前一天无用量
    await cells[todayIndex - 1].trigger('mouseenter')
    expect(wrapper.get('[data-testid="heatmap-tooltip"]').text()).toContain('dashboard.fork.heatmap.noUsage')
  })

  it('在相邻格子之间移动时保持 tooltip 可见，不触发闪烁隐藏', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [trendPoint(today)],
    })

    const wrapper = await mountHeatmap()
    const cells = wrapper.findAll('[data-testid="heatmap-cell"]')
    const todayIndex = cells.findIndex((c) => c.classes().includes('bg-green-700'))
    await cells[todayIndex].trigger('mouseenter')
    const tooltipElement = wrapper.get('[data-testid="heatmap-tooltip"]').element

    await cells[todayIndex - 1].trigger('mouseenter')

    expect(wrapper.get('[data-testid="heatmap-tooltip"]').element).toBe(tooltipElement)
    expect(wrapper.get('[data-testid="heatmap-tooltip"]').attributes('aria-hidden')).toBe('false')
  })

  it('左侧渲染周一/周三/周五的星期标签', async () => {
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    const wrapper = await mountHeatmap()
    // 周一/周三/周五用窄格式渲染（英文为 M/W/F），放在网格第 1 列的第 2/4/6 行
    const labels = [1, 3, 5].map((d) =>
      new Date(2024, 0, d).toLocaleDateString('en', { weekday: 'narrow' })
    )
    for (const [i, dayOfWeek] of [1, 3, 5].entries()) {
      const el = wrapper.find(`[style*="grid-column: 1"][style*="grid-row: ${dayOfWeek + 2}"]`)
      expect(el.exists()).toBe(true)
      expect(el.text()).toBe(labels[i])
    }
  })

  it('月份标签间隔不足 3 列时跳过显示', async () => {
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    const wrapper = await mountHeatmap()
    // 月份标签都渲染在网格第 1 行，列号从 2 开始
    const labelCols = wrapper
      .findAll('[style*="grid-row: 1"]')
      .map((el) => Number(/grid-column:\s*(\d+)/.exec(el.attributes('style') || '')?.[1]))
      .filter((n) => Number.isFinite(n))
    expect(labelCols.length).toBeGreaterThan(0)
    for (let i = 1; i < labelCols.length; i++) {
      expect(labelCols[i] - labelCols[i - 1]).toBeGreaterThanOrEqual(3)
    }
  })

  it('reload 会重新请求数据', async () => {
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    await mountHeatmap()
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledTimes(1)

    await heatmapRef.value?.reload()
    await flushPromises()
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledTimes(2)
  })

  it('点击格子把趋势图切到当天按小时查看，并标记为选中', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [trendPoint(today)],
    })

    const wrapper = await mountHeatmap()
    const cell = wrapper.get(`[data-date="${today}"]`)
    expect(cell.attributes('role')).toBe('button')
    expect(cell.attributes('tabindex')).toBe('0')
    expect(cell.classes()).toContain('heatmap-cell-today')

    await cell.trigger('click')
    await flushPromises()

    expect(selectedDays).toEqual([today])
    expect(usageState.rangePreset.value).toBe('custom')
    expect(usageState.activeRange.value.granularity).toBe('hour')
    expect(wrapper.get(`[data-date="${today}"]`).attributes('aria-pressed')).toBe('true')
    expect(wrapper.get(`[data-date="${today}"]`).classes()).toContain('heatmap-cell-selected')
  })

  it('方向键在日期之间移动焦点，不会移到未来日期', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    const wrapper = await mountHeatmap()
    const yesterdayKey = shiftDays(end, -1)

    await wrapper.get(`[data-date="${today}"]`).trigger('keydown', { key: 'ArrowUp' })
    expect(wrapper.get(`[data-date="${yesterdayKey}"]`).attributes('tabindex')).toBe('0')
    expect(wrapper.get(`[data-date="${today}"]`).attributes('tabindex')).toBe('-1')

    await wrapper.get(`[data-date="${yesterdayKey}"]`).trigger('keydown', { key: 'ArrowDown' })
    await wrapper.get(`[data-date="${today}"]`).trigger('keydown', { key: 'ArrowDown' })
    expect(wrapper.get(`[data-date="${today}"]`).attributes('tabindex')).toBe('0')
  })
})
