import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref } from 'vue'

import UserDashboardHeatmap from '../UserDashboardHeatmap.vue'
import { provideUsageChartState, type UsageChartState } from '../usageChartState'
import { usageAPI } from '@/api/usage'
import { formatDateLocalInput } from '@/utils/format'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
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

// 与 TokenRouter 一致：固定拉取近三年数据，今天向前 3*364 天再对齐到周日
const FETCH_DAYS = 3 * 364
// 156 个整周再加今天所在的一周
const TOTAL_WEEKS = FETCH_DAYS / 7 + 1
// 测不出容器宽度时回退显示最近 53 周
const FALLBACK_WEEKS = 53
// 格子 12px、间距 3px，星期标签列按 24px 估算
const weeksForWidth = (width: number) => Math.max(4, Math.min(Math.floor((width - 24 + 3) / 15), TOTAL_WEEKS))

// 色阶 0-4 的浅色 / 暗色类（0 档暗色比 TokenRouter 亮一档，匹配本项目 dark-800 的卡片底）
const LEVEL_CLASSES = [
  ['bg-gray-100', 'dark:bg-dark-600'],
  ['bg-green-200', 'dark:bg-green-800'],
  ['bg-green-300', 'dark:bg-green-600'],
  ['bg-green-500', 'dark:bg-green-400'],
  ['bg-green-700', 'dark:bg-green-300'],
]

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

// 今天及之前最近的一个星期 dayOfWeek（0 为周日）
const latestWeekday = (end: Date, dayOfWeek: number) => shiftDays(end, -((end.getDay() - dayOfWeek + 7) % 7))

const rect = (left: number, top: number, width: number, height: number) => ({
  left, top, width, height, x: left, y: top, right: left + width, bottom: top + height, toJSON: () => ({}),
}) as DOMRect

const gridOf = (wrapper: Awaited<ReturnType<typeof mountHeatmap>>) => wrapper.get('[data-testid="heatmap-grid-wrap"] > div')

describe('UserDashboardHeatmap', () => {
  beforeEach(() => {
    vi.mocked(usageAPI.getDashboardTrend).mockReset()
    selectedDays.length = 0
    localStorage.clear()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('首次请求未返回时按回退周数铺满占位格，成功后替换为日期数据', async () => {
    let resolveTrend!: (value: typeof emptyTrend) => void
    vi.mocked(usageAPI.getDashboardTrend).mockImplementationOnce(() => new Promise(resolve => { resolveTrend = resolve }))
    const wrapper = await mountHeatmap()
    const skeletons = wrapper.findAll('[data-testid="heatmap-skeleton-cell"]')
    expect(skeletons.length).toBe(FALLBACK_WEEKS * 7)
    expect(skeletons[0].classes()).toEqual(expect.arrayContaining(['h-3', 'w-3', 'animate-pulse', 'dark:bg-dark-600']))
    expect(wrapper.find('[data-testid="heatmap-cell"]').exists()).toBe(false)
    expect(wrapper.attributes('aria-busy')).toBe('true')

    resolveTrend(emptyTrend)
    await flushPromises()
    expect(wrapper.find('[data-testid="heatmap-skeleton-cell"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="heatmap-cell"]').exists()).toBe(true)
    expect(wrapper.attributes('aria-busy')).toBe('false')
    wrapper.unmount()
  })

  it('按近三年整周范围请求按日趋势数据', async () => {
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    await mountHeatmap()

    const { start, end } = expectedRange()
    expect(usageAPI.getDashboardTrend).toHaveBeenCalledWith({
      start_date: formatDateLocalInput(start),
      end_date: formatDateLocalInput(end),
      granularity: 'day',
    })
  })

  it('回退时渲染 53 周的固定 12px 网格，接口未返回的日期补零为无用量档', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [trendPoint(today)],
    })

    const wrapper = await mountHeatmap()

    // 网格横向铺满卡片：列宽固定 12px、间距 3px，零头由 justify-between 均摊
    const grid = gridOf(wrapper)
    expect(grid.classes()).toEqual(expect.arrayContaining(['grid', 'justify-between']))
    expect(grid.attributes('style')).toContain(`repeat(${FALLBACK_WEEKS}, 12px)`)
    expect(grid.attributes('style')).toContain('gap: 3px')

    const cells = wrapper.findAll('[data-testid="heatmap-cell"]')
    expect(cells.length).toBe(FALLBACK_WEEKS * 7)
    expect(cells[0].classes()).toEqual(expect.arrayContaining(['heatmap-cell', 'h-3', 'w-3']))

    // 有用量的今天为最高档且只有一天；未来占位格不可见
    const greenCells = cells.filter((c) => c.classes().includes('bg-green-700'))
    expect(greenCells.length).toBe(1)
    const futureCount = cells.filter((c) => c.classes().includes('invisible')).length
    expect(cells.filter((c) => c.classes().includes('bg-gray-100')).length).toBe(cells.length - futureCount - 1)
  })

  it('可见周数按网格容器宽度自适应，窗口缩放时跟着变化', async () => {
    const callbacks: ResizeObserverCallback[] = []
    const disconnect = vi.fn()
    vi.stubGlobal('ResizeObserver', class {
      constructor(callback: ResizeObserverCallback) { callbacks.push(callback) }
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = disconnect
    })
    const widthSpy = vi.spyOn(Element.prototype, 'clientWidth', 'get').mockReturnValue(600)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    // 组件只在挂载时读一次 clientWidth，之后靠 ResizeObserver 更新
    const wrapper = await mountHeatmap()
    widthSpy.mockRestore()
    // 挂载时读取容器宽度：600px 放得下 38 周
    expect(weeksForWidth(600)).toBe(38)
    expect(wrapper.findAll('[data-testid="heatmap-cell"]').length).toBe(38 * 7)
    expect(gridOf(wrapper).attributes('style')).toContain('repeat(38, 12px)')

    const resize = async (width: number) => {
      callbacks[0]([{ contentRect: { width } } as ResizeObserverEntry], {} as ResizeObserver)
      await nextTick()
    }

    await resize(300)
    expect(wrapper.findAll('[data-testid="heatmap-cell"]').length).toBe(weeksForWidth(300) * 7)

    // 很窄时至少保留 4 周
    await resize(30)
    expect(wrapper.findAll('[data-testid="heatmap-cell"]').length).toBe(4 * 7)

    // 很宽时显示拉取到的全部周，第一格就是拉取范围的起点（周日）
    await resize(5000)
    const cells = wrapper.findAll('[data-testid="heatmap-cell"]')
    expect(cells.length).toBe(TOTAL_WEEKS * 7)
    expect(cells[0].attributes('data-date')).toBe(formatDateLocalInput(expectedRange().start))

    wrapper.unmount()
    expect(disconnect).toHaveBeenCalled()
  })

  it('按非零用量的四分位分档，图例和格子使用同一套色阶', async () => {
    const { end } = expectedRange()
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue({
      ...emptyTrend,
      trend: [
        trendPoint(shiftDays(end, 0), 4000),
        trendPoint(shiftDays(end, -1), 100),
        trendPoint(shiftDays(end, -2), 2000),
        trendPoint(shiftDays(end, -3), 3000),
        trendPoint(shiftDays(end, -4), 3500),
      ],
    })
    const wrapper = await mountHeatmap()

    const levelOf = (offset: number) => wrapper.get(`[data-date="${shiftDays(end, offset)}"]`).classes()
    expect(levelOf(0)).toEqual(expect.arrayContaining(LEVEL_CLASSES[4]))
    expect(levelOf(-4)).toEqual(expect.arrayContaining(LEVEL_CLASSES[3]))
    expect(levelOf(-3)).toEqual(expect.arrayContaining(LEVEL_CLASSES[2]))
    expect(levelOf(-2)).toEqual(expect.arrayContaining(LEVEL_CLASSES[1]))
    expect(levelOf(-1)).toEqual(expect.arrayContaining(LEVEL_CLASSES[1]))
    expect(levelOf(-10)).toEqual(expect.arrayContaining(LEVEL_CLASSES[0]))

    const legend = wrapper.findAll('span.heatmap-cell')
    expect(legend.length).toBe(5)
    legend.forEach((swatch, level) => {
      expect(swatch.classes()).toEqual(expect.arrayContaining(['h-3', 'w-3', ...LEVEL_CLASSES[level]]))
    })
  })

  it('格子按列错峰入场，最晚一列不超过 480ms，最后一列播完后去掉入场类', async () => {
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)
    const wrapper = await mountHeatmap()
    const cells = wrapper.findAll('[data-testid="heatmap-cell"]')
    const delays = cells
      .map(cell => parseInt((cell.attributes('style') ?? '').match(/--wave-delay:\s*(\d+)ms/)?.[1] ?? '0', 10))
    expect(delays[0]).toBe(0)
    expect(Math.max(...delays)).toBeLessThanOrEqual(480)
    expect(Math.max(...delays)).toBeGreaterThan(0)

    const visible = cells.filter((c) => !c.classes().includes('invisible'))
    expect(visible.every((c) => c.classes().includes('heatmap-cell-enter'))).toBe(true)

    // 前面列的动画结束不影响，最后一列结束后整体去掉入场类
    await cells[0].trigger('animationend')
    expect(wrapper.findAll('[data-testid="heatmap-cell"]')[0].classes()).toContain('heatmap-cell-enter')
    const last = wrapper.findAll('[data-wave-last="true"]')
    expect(last.length).toBe(7)
    await last[0].trigger('animationend')
    expect(wrapper.find('.heatmap-cell-enter').exists()).toBe(false)
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
    expect(tooltip.text()).toContain('dashboard.fork.heatmap.selectDay')
    expect(tooltip.attributes('aria-hidden')).toBe('false')
    expect(tooltip.attributes('style')).toContain('opacity var(--motion-fast) var(--motion-ease)')
    expect(tooltip.attributes('style')).not.toMatch(/transition:.*(?:width|left) /)

    await wrapper.get('[data-testid="heatmap-grid-wrap"]').trigger('mouseleave')
    expect(wrapper.get('[data-testid="heatmap-tooltip"]').attributes('aria-hidden')).toBe('true')

    // 前一天无用量
    await cells[todayIndex - 1].trigger('mouseenter')
    expect(wrapper.get('[data-testid="heatmap-tooltip"]').text()).toContain('dashboard.fork.heatmap.noUsage')
  })

  it('提示框以格子中心定位：前两行在格子下方 18px，其余行在格子上方', async () => {
    const { end } = expectedRange()
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)
    const wrapper = await mountHeatmap()
    vi.spyOn(wrapper.element as HTMLElement, 'getBoundingClientRect').mockReturnValue(rect(10, 20, 800, 200))

    const hoverAt = async (date: string) => {
      const cell = wrapper.get(`[data-date="${date}"]`)
      vi.spyOn(cell.element, 'getBoundingClientRect').mockReturnValue(rect(110, 100, 12, 12))
      await cell.trigger('mouseenter')
      return wrapper.get('[data-testid="heatmap-tooltip"]').attributes('style') ?? ''
    }

    // 周一在第 2 行：向下弹出，格子顶部 80px + 18px
    const below = await hoverAt(latestWeekday(end, 1))
    expect(below).toContain('left: 106px')
    expect(below).toContain('top: 98px')
    expect(below).toContain('transform: translateX(-50%)')

    // 周三在第 4 行：向上弹出，格子顶部上方 6px
    const above = await hoverAt(latestWeekday(end, 3))
    expect(above).toContain('top: 74px')
    expect(above).toContain('transform: translate(-50%, -100%)')
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

  it('reload 进行中保留已有格子但改为占位色，期间不响应悬停和点击', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValueOnce({ ...emptyTrend, trend: [trendPoint(today)] })
    const wrapper = await mountHeatmap()

    let resolveTrend!: (value: typeof emptyTrend) => void
    vi.mocked(usageAPI.getDashboardTrend).mockImplementationOnce(() => new Promise(resolve => { resolveTrend = resolve }))
    const pending = heatmapRef.value!.reload()
    await nextTick()

    expect(wrapper.attributes('aria-busy')).toBe('true')
    expect(wrapper.find('[data-testid="heatmap-skeleton-cell"]').exists()).toBe(false)
    const cell = wrapper.get(`[data-date="${today}"]`)
    expect(cell.classes()).toEqual(expect.arrayContaining(['animate-pulse', 'bg-gray-200', 'dark:bg-dark-600']))
    expect(cell.classes()).not.toContain('bg-green-700')

    await cell.trigger('mouseenter')
    await cell.trigger('click')
    expect(wrapper.get('[data-testid="heatmap-tooltip"]').attributes('aria-hidden')).toBe('true')
    expect(selectedDays).toEqual([])

    resolveTrend({ ...emptyTrend, trend: [trendPoint(today)] } as typeof emptyTrend)
    await pending
    await flushPromises()
    expect(wrapper.get(`[data-date="${today}"]`).classes()).toContain('bg-green-700')
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
    const lastWeekKey = shiftDays(end, -7)

    await wrapper.get(`[data-date="${today}"]`).trigger('keydown', { key: 'ArrowUp' })
    expect(wrapper.get(`[data-date="${yesterdayKey}"]`).attributes('tabindex')).toBe('0')
    expect(wrapper.get(`[data-date="${today}"]`).attributes('tabindex')).toBe('-1')

    await wrapper.get(`[data-date="${yesterdayKey}"]`).trigger('keydown', { key: 'ArrowDown' })
    await wrapper.get(`[data-date="${today}"]`).trigger('keydown', { key: 'ArrowDown' })
    expect(wrapper.get(`[data-date="${today}"]`).attributes('tabindex')).toBe('0')

    // 左右按周移动；今天往右是下一周，已越界，不移动
    await wrapper.get(`[data-date="${today}"]`).trigger('keydown', { key: 'ArrowLeft' })
    expect(wrapper.get(`[data-date="${lastWeekKey}"]`).attributes('tabindex')).toBe('0')
    await wrapper.get(`[data-date="${lastWeekKey}"]`).trigger('keydown', { key: 'ArrowRight' })
    await wrapper.get(`[data-date="${today}"]`).trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.get(`[data-date="${today}"]`).attributes('tabindex')).toBe('0')
  })

  it('Enter 和空格选中当前格子', async () => {
    const { end } = expectedRange()
    const today = formatDateLocalInput(end)
    const yesterdayKey = shiftDays(end, -1)
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    const wrapper = await mountHeatmap()
    await wrapper.get(`[data-date="${yesterdayKey}"]`).trigger('keydown', { key: 'Enter' })
    expect(selectedDays).toEqual([yesterdayKey])
    expect(wrapper.get(`[data-date="${yesterdayKey}"]`).attributes('aria-pressed')).toBe('true')
    expect(wrapper.get(`[data-date="${yesterdayKey}"]`).attributes('tabindex')).toBe('0')

    await wrapper.get(`[data-date="${today}"]`).trigger('keydown', { key: ' ' })
    expect(selectedDays).toEqual([yesterdayKey, today])
    expect(wrapper.get(`[data-date="${yesterdayKey}"]`).attributes('aria-pressed')).toBe('false')
    expect(wrapper.get(`[data-date="${today}"]`).attributes('aria-pressed')).toBe('true')
  })

  it('今天之后补齐本周的格子不可见，也不能悬停、点击或获得焦点', async () => {
    // 2026-10-12 是周一，本周二到周六共 5 个未来占位格
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 9, 12, 12, 0, 0))
    vi.mocked(usageAPI.getDashboardTrend).mockResolvedValue(emptyTrend)

    const wrapper = await mountHeatmap()
    const cells = wrapper.findAll('[data-testid="heatmap-cell"]')
    const future = cells.filter((c) => c.classes().includes('invisible'))
    expect(future.map((c) => c.attributes('data-date'))).toEqual([
      '2026-10-13', '2026-10-14', '2026-10-15', '2026-10-16', '2026-10-17',
    ])
    for (const cell of future) {
      expect(cell.attributes('role')).toBeUndefined()
      expect(cell.attributes('tabindex')).toBeUndefined()
      expect(cell.attributes('aria-label')).toBeUndefined()
      expect(cell.attributes('aria-pressed')).toBeUndefined()
    }

    const today = wrapper.get('[data-date="2026-10-12"]')
    expect(today.classes()).toContain('heatmap-cell-today')
    expect(today.attributes('tabindex')).toBe('0')
    expect(wrapper.get('[data-date="2026-10-11"]').classes()).not.toContain('heatmap-cell-today')

    await future[0].trigger('mouseenter')
    await future[0].trigger('click')
    expect(wrapper.get('[data-testid="heatmap-tooltip"]').attributes('aria-hidden')).toBe('true')
    expect(selectedDays).toEqual([])

    // 今天往下是明天（未来），焦点留在今天
    await today.trigger('keydown', { key: 'ArrowDown' })
    expect(wrapper.get('[data-date="2026-10-12"]').attributes('tabindex')).toBe('0')
  })
})
