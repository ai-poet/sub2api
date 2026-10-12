import { computed, inject, provide, ref, type InjectionKey } from 'vue'
import { usageAPI } from '@/api/usage'
import type { ModelStat, TrendDataPoint } from '@/types'
import {
  bucketKeys,
  fillTrendBuckets,
  formatDayKey,
  previousRange,
  queryDates,
  resolveCustomRange,
  resolvePresetRange,
  summarizeTrend,
  type UsageRange,
  type UsageRangePreset,
  type UsageTotals,
} from './usageChartData'
import { useUsageChartFilters } from './useUsageChartFilters'

// fork：仪表盘用量区的共享状态（移植自 TokenRouter usageChartState.ts）。
// 标题行的工具栏负责修改条件，指标卡、趋势图、模型排行和热力图负责展示。

// UsageMetric 是指标卡和趋势图可以切换的指标。
export type UsageMetric = 'requests' | 'tokens' | 'cost' | 'cacheHitRate'

// PresetRange 是工具栏上的快捷时间范围。
export type PresetRange = Exclude<UsageRangePreset, 'custom'>

export const USAGE_METRICS: UsageMetric[] = ['requests', 'tokens', 'cost', 'cacheHitRate']
export const USAGE_RANGE_PRESETS: PresetRange[] = ['24h', '7d', '30d']

// 首次打开默认看近 7 天（与 TokenRouter 代码一致）；之后记住上次的选择。
export const DEFAULT_RANGE: PresetRange = '7d'
export const DEFAULT_METRIC: UsageMetric = 'tokens'

export const USAGE_CHART_STORAGE_KEY = 'dashboard-usage-chart'
// 日期选择器的快捷项能换算成快捷范围时按快捷范围处理，记住选择并按整点或整天对齐。
// 选择器的近 24 小时只给出日期，不映射会被当成两整天。
const PICKER_PRESET_MAP: Record<string, PresetRange> = { last24Hours: '24h', '7days': '7d', '30days': '30d' }

// readSavedState 读取上次选择的指标与快捷范围，存储不可用时使用默认值。
const readSavedState = (): { metric: UsageMetric; range: PresetRange } => {
  const fallback = { metric: DEFAULT_METRIC, range: DEFAULT_RANGE }
  try {
    const saved = JSON.parse(localStorage.getItem(USAGE_CHART_STORAGE_KEY) || '{}')
    return {
      metric: USAGE_METRICS.includes(saved.metric) ? saved.metric : fallback.metric,
      range: USAGE_RANGE_PRESETS.includes(saved.range) ? saved.range : fallback.range,
    }
  } catch {
    return fallback
  }
}

// createUsageChartState 创建仪表盘用量区的共享状态。
export function createUsageChartState() {
  const saved = readSavedState()
  const metric = ref<UsageMetric>(saved.metric)
  const rangePreset = ref<UsageRangePreset>(saved.range)
  const customRange = ref<UsageRange | null>(null)
  const activeRange = ref<UsageRange>(resolvePresetRange(saved.range))

  const loading = ref(false)
  const loaded = ref(false)
  const current = ref<TrendDataPoint[]>([])
  // 上一周期按下标与本期对齐，用于对比线；取数失败时为空数组，同时隐藏环比。
  const previous = ref<TrendDataPoint[]>([])
  const previousTotals = ref<UsageTotals | null>(null)
  // 当前范围和筛选条件下的模型用量，供模型排行使用。
  const models = ref<ModelStat[]>([])
  const modelsLoaded = ref(false)
  const totals = computed(() => summarizeTrend(current.value))

  const filterState = useUsageChartFilters()

  // 日期选择器显示当前窗口覆盖的日期，结束时间是排他边界，所以回退一毫秒。
  const pickerStart = computed(() => formatDayKey(activeRange.value.startAt))
  const pickerEnd = computed(() => formatDayKey(new Date(activeRange.value.endAt.getTime() - 1)))

  // saveState 只记住指标和快捷范围，自定义日期与筛选条件下次打开时回到默认值。
  const saveState = () => {
    try {
      const range = rangePreset.value === 'custom' ? readSavedState().range : rangePreset.value
      localStorage.setItem(USAGE_CHART_STORAGE_KEY, JSON.stringify({ metric: metric.value, range }))
    } catch {
      // 存储不可用时只在当前页面生效。
    }
  }

  // 用递增序号丢弃过期响应，快速切换范围或筛选时只保留最后一次结果。
  let requestSeq = 0

  // fetchRange 拉取一个窗口的趋势数据，带上当前的筛选条件。
  const fetchRange = (range: UsageRange) => usageAPI.getDashboardTrend({
    ...queryDates(range),
    granularity: range.granularity,
    ...filterState.queryParams.value,
  })

  // fetchModels 拉取一个窗口内的模型用量，带上当前的筛选条件。
  const fetchModels = (range: UsageRange) => usageAPI.getDashboardModels({
    ...queryDates(range),
    ...filterState.queryParams.value,
  })

  // loadTrend 并发拉取当前窗口、上一周期和模型用量；上一周期失败只隐藏环比和对比线，模型用量失败只清空排行。
  const loadTrend = async (range: UsageRange) => {
    const seq = ++requestSeq
    // 没有筛选条件时，模型用量就是候选项的来源，不必再单独请求一次。
    const unfiltered = filterState.activeCount.value === 0
    const prevRange = previousRange(range)
    loading.value = true
    const [currentResult, previousResult, modelsResult] = await Promise.allSettled([
      fetchRange(range),
      fetchRange(prevRange),
      fetchModels(range),
    ])
    if (seq !== requestSeq) return
    if (currentResult.status === 'fulfilled') {
      current.value = fillTrendBuckets(currentResult.value.trend || [], bucketKeys(range))
    } else {
      console.error('Failed to load usage trend:', currentResult.reason)
      current.value = fillTrendBuckets([], bucketKeys(range))
    }
    if (previousResult.status === 'fulfilled') {
      // 查询按整天覆盖窗口，按小时的窗口要裁回窗口内的时段再汇总。
      previous.value = fillTrendBuckets(previousResult.value.trend || [], bucketKeys(prevRange))
      previousTotals.value = summarizeTrend(previous.value)
    } else {
      previous.value = []
      previousTotals.value = null
    }
    if (modelsResult.status === 'fulfilled') {
      models.value = modelsResult.value.models || []
      if (unfiltered) filterState.setModelOptions(models.value)
    } else {
      console.error('Failed to load usage models:', modelsResult.reason)
      models.value = []
    }
    loading.value = false
    loaded.value = true
    modelsLoaded.value = true
  }

  // load 按当前范围重新拉取趋势和模型用量；有筛选条件时另取一次不带筛选的模型候选。
  const load = async () => {
    // 只有自定义范围时 customRange 才有值，其余情况按快捷范围实时换算。
    const range = customRange.value ?? resolvePresetRange(rangePreset.value as PresetRange)
    activeRange.value = range
    const tasks = [loadTrend(range)]
    if (filterState.activeCount.value > 0) tasks.push(filterState.loadModelOptions(range))
    await Promise.all(tasks)
  }

  // 筛选只影响趋势和模型用量，候选项沿用当前范围的结果。
  const applyFilters = () => {
    void loadTrend(activeRange.value)
  }

  const resetFilters = () => {
    filterState.reset()
    applyFilters()
  }

  const selectMetric = (value: UsageMetric) => {
    metric.value = value
    saveState()
  }

  const selectPreset = (value: PresetRange) => {
    rangePreset.value = value
    customRange.value = null
    saveState()
    void load()
  }

  const selectCustomRange = (range: { startDate: string; endDate: string; preset: string | null }) => {
    const mapped = range.preset ? PICKER_PRESET_MAP[range.preset] : undefined
    if (mapped) {
      selectPreset(mapped)
      return
    }
    const resolved = resolveCustomRange(range.startDate, range.endDate)
    if (!resolved) return
    rangePreset.value = 'custom'
    customRange.value = resolved
    void load()
  }

  // selectDay 把范围切到某一天，单日窗口按小时聚合。
  const selectDay = (date: string) => {
    selectCustomRange({ startDate: date, endDate: date, preset: null })
  }

  // selectedDay 是当前自定义范围正好覆盖的单日，其余情况为 null。
  const selectedDay = computed(() => (
    rangePreset.value === 'custom' && pickerStart.value === pickerEnd.value ? pickerStart.value : null
  ))

  // applyModelFilter 按模型筛选整个用量区，传入 null 取消模型筛选。
  const applyModelFilter = (model: string | null) => {
    filterState.filters.value.model = model
    applyFilters()
  }

  return {
    metric,
    rangePreset,
    activeRange,
    loading,
    loaded,
    current,
    previous,
    previousTotals,
    models,
    modelsLoaded,
    totals,
    pickerStart,
    pickerEnd,
    filterState,
    load,
    loadFilterOptions: filterState.loadKeyAndGroupOptions,
    applyFilters,
    resetFilters,
    selectMetric,
    selectPreset,
    selectCustomRange,
    selectDay,
    selectedDay,
    applyModelFilter,
  }
}

export type UsageChartState = ReturnType<typeof createUsageChartState>

const USAGE_CHART_STATE_KEY: InjectionKey<UsageChartState> = Symbol('forkUsageChartState')

// provideUsageChartState 在页面上创建共享状态，供标题行的工具栏和正文里的图表共同使用。
export const provideUsageChartState = (): UsageChartState => {
  const state = createUsageChartState()
  provide(USAGE_CHART_STATE_KEY, state)
  return state
}

// injectUsageChartState 读取页面提供的共享状态。
export const injectUsageChartState = (): UsageChartState => {
  const state = inject(USAGE_CHART_STATE_KEY)
  if (!state) throw new Error('usage chart state is not provided')
  return state
}
