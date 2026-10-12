<template>
  <div ref="cardRef" class="card relative p-4" :aria-busy="loading" data-testid="usage-heatmap">
    <div class="mb-4 flex items-center justify-between gap-2">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-dark-50">{{ t('dashboard.fork.heatmap.title') }}</h3>
      <!-- 色阶图例 -->
      <div class="flex shrink-0 items-center gap-1 text-xs text-gray-500 dark:text-dark-400" aria-hidden="true">
        <span>{{ t('dashboard.fork.heatmap.less') }}</span>
        <span v-for="level in 5" :key="level" class="heatmap-cell h-3 w-3" :class="levelClass(level - 1)" />
        <span>{{ t('dashboard.fork.heatmap.more') }}</span>
      </div>
    </div>

    <!-- 宽屏时网格在左、本段汇总在右；放不下时汇总换到网格下方 -->
    <div ref="bodyRef" class="flex flex-wrap items-stretch gap-x-6 gap-y-4" data-testid="heatmap-body">
      <!--
        格子边长随宽度在 12-18px 之间变化、间距 3px，周列数随容器宽度自适应，最多 26 周。
      -->
      <div
        ref="gridWrapRef"
        class="min-w-0"
        data-testid="heatmap-grid-wrap"
        @mouseleave="hoveredDay = null"
        @focusout="onGridFocusOut"
        @keydown="onGridKeydown"
        @animationend="onWaveEnd"
      >
        <!--
          统一网格：第 1 列是星期标签，第 1 行是月份标签，其余为日期格子。
          格子显式指定 gridColumn/gridRow，保证标签与格子严格对齐。
        -->
        <div
          class="grid w-max"
          :style="{
            gridTemplateColumns: `auto repeat(${visibleWeeks}, ${cellPx}px)`,
            gridTemplateRows: `16px repeat(7, ${cellPx}px)`,
            gap: `${GAP_PX}px`,
            '--cell-size': `${cellPx}px`,
          }"
        >
          <!-- 首次取数还没有日期数据，也要按当前可见周数填满占位格。 -->
          <template v-if="loading && days.length === 0">
            <div
              aria-hidden="true"
              class="h-4 w-8 animate-pulse rounded-md bg-gray-200 dark:bg-dark-700 motion-reduce:animate-none"
              :style="{ gridColumn: 2, gridRow: 1 }"
            ></div>
            <div
              v-for="cell in visibleWeeks * 7"
              :key="`loading-${cell}`"
              data-testid="heatmap-skeleton-cell"
              aria-hidden="true"
              class="heatmap-cell animate-pulse bg-gray-200 dark:bg-dark-700 motion-reduce:animate-none"
              :style="{ gridColumn: Math.floor((cell - 1) / 7) + 2, gridRow: ((cell - 1) % 7) + 2 }"
            ></div>
          </template>
          <!-- 月份标签：本周首格月份与上一列不同才显示 -->
          <div
            v-for="m in monthItems"
            :key="`m-${m.weekIndex}`"
            class="overflow-visible whitespace-nowrap text-xs leading-4 text-gray-400 dark:text-gray-500"
            :style="{ gridColumn: m.weekIndex + 2, gridRow: 1 }"
          >{{ m.label }}</div>

          <!-- 星期标签：只标周一/周三/周五 -->
          <div
            v-for="w in weekdayLabels"
            :key="`w-${w.row}`"
            class="flex items-center pr-1 text-xs leading-none text-gray-400 dark:text-gray-500"
            :style="{ gridColumn: 1, gridRow: w.row + 2 }"
          >{{ w.label }}</div>

          <!--
            格子可点击查看当天按小时的用量。键盘使用漫游焦点：只有一个格子可 Tab 进入，
            方向键在日期间移动，Enter / Space 选中。
          -->
          <div
            v-for="day in visibleDays"
            :key="day.date"
            data-testid="heatmap-cell"
            :data-date="day.date"
            :data-level="day.future ? undefined : day.level"
            :data-wave-last="day.weekIndex === visibleWeeks - 1 ? 'true' : undefined"
            class="heatmap-cell"
            :class="cellClasses(day)"
            :style="{ gridColumn: day.weekIndex + 2, gridRow: day.dayOfWeek + 2, '--wave-delay': waveDelay(day.weekIndex) }"
            :role="day.future ? undefined : 'button'"
            :tabindex="day.future ? undefined : day.date === focusDate ? 0 : -1"
            :aria-label="day.future ? undefined : cellLabel(day)"
            :aria-pressed="day.future ? undefined : day.date === selectedDay"
            @mouseenter="!loading && onCellHover(day, $event)"
            @focus="!loading && onCellHover(day, $event)"
            @click="selectCell(day)"
          />
        </div>
      </div>

      <!-- 本段汇总：只统计网格里可见的日期 -->
      <dl
        class="grid min-w-[13rem] flex-1 grid-cols-2 content-center gap-3"
        data-testid="heatmap-summary"
      >
        <div v-for="item in summaryItems" :key="item.key" class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700/50">
          <dt class="truncate text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</dt>
          <dd class="mt-0.5 truncate text-sm font-semibold tabular-nums text-gray-900 dark:text-dark-50" :data-testid="`heatmap-summary-${item.key}`">
            <Skeleton v-if="loading && days.length === 0" :width="56" :height="16" />
            <template v-else>{{ item.value }}</template>
          </dd>
        </div>
      </dl>
    </div>

    <!--
      悬停提示：卡片内绝对定位，避免被滚动容器裁剪。
      前两行格子改为下方弹出，避免超出卡片顶部。
    -->
    <div
      data-testid="heatmap-tooltip"
      class="pointer-events-none absolute z-20 whitespace-nowrap rounded-lg bg-gray-900 px-2 py-1 text-xs text-white shadow-lg ring-1 ring-white/10 dark:bg-dark-900 dark:text-dark-100 dark:ring-dark-600"
      :style="tooltipStyle"
      :aria-hidden="hoveredDay ? 'false' : 'true'"
    >
      <div ref="tooltipContentRef" class="w-max">
        <template v-if="hoveredDay">
          <div class="font-medium">{{ formatDayLabel(hoveredDay.date) }}</div>
          <template v-if="hoveredDay.requests > 0">
            <div>{{ t('dashboard.fork.heatmap.requests') }}: {{ formatNumber(hoveredDay.requests) }}</div>
            <div>{{ t('dashboard.fork.heatmap.tokens') }}: {{ formatTokens(hoveredDay.tokens) }}</div>
            <div>{{ t('dashboard.fork.heatmap.cost') }}: {{ formatUsd(hoveredDay.actualCost, 4) }}</div>
          </template>
          <div v-else>{{ t('dashboard.fork.heatmap.noUsage') }}</div>
          <div class="mt-1 opacity-70">{{ t('dashboard.fork.heatmap.selectDay') }}</div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：用量热力图（移植自 TokenRouter UserDashboardHeatmap.vue）。
// 与 TokenRouter 的差别：usage_logs 默认只保留 90 天，所以最多显示 26 周（TokenRouter 取三年）；
// 宽屏时格子放大到 18px，并在网格右侧放一块本段汇总，避免满宽卡片大片留白。
// 颜色按非零日用量的四分位分档；首次取数后格子按列错峰入场，最晚一列不超过 480ms。
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Skeleton from '@/components/common/Skeleton.vue'
import { usageAPI } from '@/api/usage'
import { formatDateLocalInput, formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { HEATMAP_WAVE_MAX_MS, HEATMAP_WAVE_STEP_MS } from './dashboardMotion'
import { formatUsd } from './format'
import { injectUsageChartState } from './usageChartState'

// 最多显示的周数；测不出宽度（如测试环境）时也按这个数渲染。
const MAX_WEEKS = 26
// 拉取范围：今天往前 MAX_WEEKS - 1 周，再对齐到周日，正好 MAX_WEEKS 整列。
const FETCH_DAYS = (MAX_WEEKS - 1) * 7
// 格子边长范围、间距，以及星期标签列的近似宽度（px）。
const MIN_CELL_PX = 12
const MAX_CELL_PX = 18
const GAP_PX = 3
const LABEL_COL_PX = 24
// 汇总区放在右侧时至少占的宽度，以及与网格的间距（px）。
const SUMMARY_MIN_PX = 208
const SUMMARY_GAP_PX = 24
// 格子分档色：0 为无用量，1-4 按用量分位递增
const LEVEL_CLASSES = [
  'bg-gray-100 dark:bg-dark-700',
  'bg-green-200 dark:bg-green-900',
  'bg-green-300 dark:bg-green-700',
  'bg-green-500 dark:bg-green-500',
  'bg-green-700 dark:bg-green-300',
]
// 方向键在日期间移动的步长：上下为前后一天，左右为前后一周
const KEY_OFFSETS: Record<string, number> = { ArrowUp: -1, ArrowDown: 1, ArrowLeft: -7, ArrowRight: 7 }

interface HeatmapDay {
  date: string // YYYY-MM-DD（本地时区）
  weekIndex: number // 所在列
  dayOfWeek: number // 0-6，周日为 0
  tokens: number
  requests: number
  actualCost: number
  level: number
  future?: boolean // 补齐最后一周的未来占位格，不展示不响应悬停
}

const emit = defineEmits<{
  // 用户选中某一天，页面据此把趋势图滚动到可见区域
  (event: 'select-day', date: string): void
}>()

const { t, locale } = useI18n()
const { selectDay, selectedDay } = injectUsageChartState()

const loading = ref(true)
const days = ref<HeatmapDay[]>([])
const hoveredDay = ref<HeatmapDay | null>(null)
const cardRef = ref<HTMLElement | null>(null)
const bodyRef = ref<HTMLElement | null>(null)
const gridWrapRef = ref<HTMLElement | null>(null)
const tooltipContentRef = ref<HTMLElement | null>(null)
// 悬停格子中心相对卡片左上角的坐标，用于 tooltip 定位
const hoverPos = ref({ left: 0, top: 0 })
// 记录 tooltip 上一次弹出方向，离开格子时仍保留位置以完成淡出动画。
const tooltipAbove = ref(false)
const tooltipSize = ref({ width: 0, height: 0 })
// 卡片内容区宽度，驱动格子尺寸和可见周数
const bodyWidth = ref(0)
let resizeObserver: ResizeObserver | null = null
// 今天的日期键，每次取数时更新，跨午夜后刷新即可对上
const todayKey = ref(formatDateLocalInput(new Date()))
// 首次拿到数据时格子按列错峰入场，最后一列播放完后关闭
const waveActive = ref(true)
// 键盘焦点所在日期；未设置时落在选中日或今天
const focusOverride = ref<string | null>(null)

const levelClass = (level: number) => LEVEL_CLASSES[level] ?? LEVEL_CLASSES[0]

// 拉取范围：向前对齐到周日，保证整周列
// start 从 end 克隆，保证两者时分秒一致，最后一天比较不会出现毫秒级漂移
const buildDateRange = () => {
  const end = new Date()
  const start = new Date(end)
  start.setDate(start.getDate() - FETCH_DAYS)
  start.setDate(start.getDate() - start.getDay())
  return { start, end }
}

// 按非零用量的 25/50/75 分位划分 1-4 档，无用量为 0 档；峰值日固定为最高档
const computeLevel = (tokens: number, sortedNonZero: number[]): number => {
  if (tokens <= 0 || sortedNonZero.length === 0) return 0
  if (tokens >= sortedNonZero[sortedNonZero.length - 1]) return 4
  const percentile = (p: number) => sortedNonZero[Math.min(sortedNonZero.length - 1, Math.floor(sortedNonZero.length * p))]
  if (tokens <= percentile(0.25)) return 1
  if (tokens <= percentile(0.5)) return 2
  if (tokens <= percentile(0.75)) return 3
  return 4
}

const load = async () => {
  loading.value = true
  hoveredDay.value = null
  todayKey.value = formatDateLocalInput(new Date())
  try {
    const { start, end } = buildDateRange()
    const res = await usageAPI.getDashboardTrend({
      start_date: formatDateLocalInput(start),
      end_date: formatDateLocalInput(end),
      granularity: 'day',
    })
    // 趋势接口只返回有用量的日期，按日期建索引后补零
    const byDate = new Map((res.trend || []).map((p) => [p.date, p]))
    const sortedNonZero = (res.trend || [])
      .map((p) => p.total_tokens)
      .filter((v) => v > 0)
      .sort((a, b) => a - b)

    const result: HeatmapDay[] = []
    const cursor = new Date(start)
    while (cursor <= end) {
      const date = formatDateLocalInput(cursor)
      const point = byDate.get(date)
      const tokens = point?.total_tokens ?? 0
      result.push({
        date,
        weekIndex: Math.floor(result.length / 7),
        dayOfWeek: cursor.getDay(),
        tokens,
        requests: point?.requests ?? 0,
        actualCost: point?.actual_cost ?? 0,
        level: computeLevel(tokens, sortedNonZero),
      })
      cursor.setDate(cursor.getDate() + 1)
    }
    // 用不可见的未来占位格补齐最后一周，保证网格按整周列对齐
    while (cursor.getDay() !== 0) {
      result.push({
        date: formatDateLocalInput(cursor),
        weekIndex: Math.floor(result.length / 7),
        dayOfWeek: cursor.getDay(),
        tokens: 0,
        requests: 0,
        actualCost: 0,
        level: 0,
        future: true,
      })
      cursor.setDate(cursor.getDate() + 1)
    }
    days.value = result
  } catch (error) {
    console.error('Failed to load usage heatmap:', error)
  } finally {
    loading.value = false
  }
}

// 数据覆盖的总周数（含占位格）
const totalWeeks = computed(() => Math.ceil(days.value.length / 7))

// 汇总区能否放在网格右侧：最小格子排满 MAX_WEEKS 周后还剩得下汇总区
const summaryBeside = computed(() => (
  bodyWidth.value >= LABEL_COL_PX + MAX_WEEKS * (MIN_CELL_PX + GAP_PX) + SUMMARY_MIN_PX + SUMMARY_GAP_PX
))

// 留给网格的宽度
const gridAvailable = computed(() => (
  summaryBeside.value ? bodyWidth.value - SUMMARY_MIN_PX - SUMMARY_GAP_PX : bodyWidth.value
))

// 格子边长：宽度富余时放大，最大 18px；测不出宽度时用最小值
const cellPx = computed(() => {
  if (gridAvailable.value <= 0) return MIN_CELL_PX
  const fit = Math.floor((gridAvailable.value - LABEL_COL_PX + GAP_PX) / MAX_WEEKS) - GAP_PX
  return Math.min(MAX_CELL_PX, Math.max(MIN_CELL_PX, fit))
})

// 可见周数随容器宽度自适应，最多 MAX_WEEKS；测不出宽度（如测试环境）时按 MAX_WEEKS 渲染
const visibleWeeks = computed(() => {
  const limit = Math.min(MAX_WEEKS, totalWeeks.value || MAX_WEEKS)
  if (gridAvailable.value <= 0) return limit
  const fit = Math.floor((gridAvailable.value - LABEL_COL_PX + GAP_PX) / (cellPx.value + GAP_PX))
  return Math.max(4, Math.min(fit, limit))
})

// 只渲染最近 visibleWeeks 周，weekIndex 重新从 0 编排
const visibleDays = computed(() =>
  days.value.slice(-visibleWeeks.value * 7).map((day, i) => ({
    ...day,
    weekIndex: Math.floor(i / 7),
  }))
)

// 月份标签：每周首格（周日）月份与上一列不同才显示；
// 相邻月份标签间距小于 MIN_MONTH_LABEL_GAP 列时保留较新的标签。
const MIN_MONTH_LABEL_GAP = 3
const monthItems = computed(() => {
  const items: { weekIndex: number; label: string }[] = []
  let prevMonth = -1
  for (const day of visibleDays.value) {
    if (day.dayOfWeek !== 0) continue
    const month = new Date(`${day.date}T00:00:00`).getMonth()
    if (month !== prevMonth) {
      const prev = items[items.length - 1]
      if (prev && day.weekIndex - prev.weekIndex < MIN_MONTH_LABEL_GAP) {
        items.pop()
      }
      items.push({ weekIndex: day.weekIndex, label: formatMonth(day.date) })
    }
    prevMonth = month
  }
  return items
})

// 星期标签：只标周一/周三/周五，用窄格式（如英文 M/W/F、中文 一/三/五）
const weekdayLabels = computed(() =>
  [1, 3, 5].map((dayOfWeek) => ({
    row: dayOfWeek,
    // 2024-01-01 是周一，依次偏移得到对应星期
    label: new Date(2024, 0, dayOfWeek).toLocaleDateString(locale.value, { weekday: 'narrow' }),
  }))
)

const formatMonth = (date: string) =>
  new Date(`${date}T00:00:00`).toLocaleDateString(locale.value, { month: 'short' })

const formatDayLabel = (date: string) =>
  new Date(`${date}T00:00:00`).toLocaleDateString(locale.value, { year: 'numeric', month: 'short', day: 'numeric' })

const formatShortDay = (date: string) =>
  new Date(`${date}T00:00:00`).toLocaleDateString(locale.value, { month: 'short', day: 'numeric' })

// 本段汇总：活跃天数、Token 合计、用量最多的一天、截至今天的连续使用天数
const summaryItems = computed(() => {
  const shown = visibleDays.value.filter((day) => !day.future)
  const active = shown.filter((day) => day.requests > 0 || day.tokens > 0)
  const tokens = shown.reduce((sum, day) => sum + day.tokens, 0)
  const busiest = active.reduce<HeatmapDay | null>((best, day) => (!best || day.tokens > best.tokens ? day : best), null)
  // 今天还没用时从昨天开始数，避免一早打开时连续天数被清零
  let streak = 0
  const ordered = [...shown].reverse()
  const startIndex = ordered[0] && ordered[0].requests === 0 && ordered[0].tokens === 0 ? 1 : 0
  for (let i = startIndex; i < ordered.length; i++) {
    if (ordered[i].requests === 0 && ordered[i].tokens === 0) break
    streak++
  }
  return [
    { key: 'activeDays', label: t('dashboard.fork.heatmap.summary.activeDays'), value: t('dashboard.fork.heatmap.summary.days', { count: active.length }, active.length) },
    { key: 'tokens', label: t('dashboard.fork.heatmap.summary.tokens'), value: formatTokens(tokens) },
    { key: 'busiest', label: t('dashboard.fork.heatmap.summary.busiest'), value: busiest ? `${formatShortDay(busiest.date)} · ${formatTokens(busiest.tokens)}` : '—' },
    { key: 'streak', label: t('dashboard.fork.heatmap.summary.streak'), value: t('dashboard.fork.heatmap.summary.days', { count: streak }, streak) },
  ]
})

// 漫游焦点所在的日期：优先手动移动到的日期，其次是选中日，最后是今天
const focusDate = computed(() => {
  const visible = new Set(visibleDays.value.filter((day) => !day.future).map((day) => day.date))
  for (const candidate of [focusOverride.value, selectedDay.value, todayKey.value]) {
    if (candidate && visible.has(candidate)) return candidate
  }
  return null
})

// waveDelay 按列计算入场延迟；列数很多时压缩步长，最后一列不晚于上限
const waveDelay = (weekIndex: number): string => {
  const step = Math.min(HEATMAP_WAVE_STEP_MS, HEATMAP_WAVE_MAX_MS / Math.max(1, visibleWeeks.value))
  return `${Math.round(weekIndex * step)}ms`
}

const cellClasses = (day: HeatmapDay) => {
  if (day.future) return 'invisible'
  if (loading.value) return 'animate-pulse bg-gray-200 dark:bg-dark-700 motion-reduce:animate-none'
  return [
    levelClass(day.level),
    'heatmap-cell-interactive',
    {
      'heatmap-cell-enter': waveActive.value,
      'heatmap-cell-today': day.date === todayKey.value,
      'heatmap-cell-selected': day.date === selectedDay.value,
    },
  ]
}

const cellLabel = (day: HeatmapDay): string => {
  const usage = day.requests > 0
    ? `${t('dashboard.fork.heatmap.requests')} ${formatNumber(day.requests)}`
    : t('dashboard.fork.heatmap.noUsage')
  return `${formatDayLabel(day.date)}, ${usage}`
}

// selectCell 把趋势图切到这一天，按小时查看
const selectCell = (day: HeatmapDay) => {
  if (day.future || loading.value) return
  focusOverride.value = day.date
  selectDay(day.date)
  emit('select-day', day.date)
}

// focusCell 把焦点移到指定日期的格子
const focusCell = (date: string) => {
  focusOverride.value = date
  gridWrapRef.value?.querySelector<HTMLElement>(`[data-date="${date}"]`)?.focus()
}

// onGridKeydown 处理格子上的方向键和选中键，越界或落到未来日期时不移动
const onGridKeydown = (event: KeyboardEvent) => {
  const date = (event.target as HTMLElement).dataset?.date
  if (!date || loading.value) return
  const index = visibleDays.value.findIndex((day) => day.date === date)
  if (index < 0) return
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    selectCell(visibleDays.value[index])
    return
  }
  const offset = KEY_OFFSETS[event.key]
  if (offset === undefined) return
  event.preventDefault()
  const next = visibleDays.value[index + offset]
  if (next && !next.future) focusCell(next.date)
}

// 焦点离开网格时收起提示
const onGridFocusOut = (event: FocusEvent) => {
  if (!gridWrapRef.value?.contains(event.relatedTarget as Node | null)) hoveredDay.value = null
}

// 最后一列入场结束后去掉入场类，之后新增的格子（如窗口变宽）直接显示
const onWaveEnd = (event: AnimationEvent) => {
  if ((event.target as HTMLElement).dataset?.waveLast) waveActive.value = false
}

// 以格子中心相对卡片的位置定位 tooltip
const onCellHover = (day: HeatmapDay, event: Event) => {
  if (day.future) return
  const cell = event.currentTarget as HTMLElement
  const card = cardRef.value
  if (card) {
    const rect = cell.getBoundingClientRect()
    const cardRect = card.getBoundingClientRect()
    hoverPos.value = {
      left: rect.left - cardRect.left + rect.width / 2,
      top: rect.top - cardRect.top,
    }
  }
  tooltipAbove.value = day.dayOfWeek >= 2
  hoveredDay.value = day
  void updateTooltipSize()
}

// tooltip 与卡片左右边缘保持的最小距离
const TOOLTIP_EDGE_PX = 8

// 提示以格子中心定位，靠近两侧时向卡片内调整。
const tooltipLeft = computed(() => {
  const half = tooltipSize.value.width / 2
  const cardWidth = cardRef.value?.clientWidth ?? 0
  if (half <= 0 || cardWidth <= 0) return hoverPos.value.left
  const min = half + TOOLTIP_EDGE_PX
  const max = cardWidth - half - TOOLTIP_EDGE_PX
  return max < min ? cardWidth / 2 : Math.min(Math.max(hoverPos.value.left, min), max)
})

const tooltipStyle = computed(() => {
  const above = tooltipAbove.value
  return {
    left: `${tooltipLeft.value}px`,
    top: above ? `${hoverPos.value.top - 6}px` : `${hoverPos.value.top + cellPx.value + 6}px`,
    transform: above ? 'translate(-50%, -100%)' : 'translateX(-50%)',
    width: tooltipSize.value.width > 0 ? `${tooltipSize.value.width}px` : '0px',
    height: tooltipSize.value.height > 0 ? `${tooltipSize.value.height}px` : '0px',
    opacity: hoveredDay.value ? '1' : '0',
    overflow: 'hidden',
    boxSizing: 'border-box' as const,
    // 定位随日期立即更新，提示只做淡入淡出。
    transition: 'opacity var(--motion-fast) var(--motion-ease)',
    willChange: 'opacity',
  }
})

// 内容更新后测量自然尺寸，让 tooltip 的容器匹配当前日期的内容。
const updateTooltipSize = async () => {
  await nextTick()
  const content = tooltipContentRef.value
  if (!content) return
  const width = Math.ceil(content.scrollWidth || content.getBoundingClientRect().width)
  const height = Math.ceil(content.scrollHeight || content.getBoundingClientRect().height)
  if (width <= 0 || height <= 0) return
  tooltipSize.value = {
    width: width + 16,
    height: height + 8,
  }
}

onMounted(() => {
  // 监听内容区宽度变化，窗口缩放时实时调整格子尺寸和可见周数
  if (bodyRef.value && typeof ResizeObserver !== 'undefined') {
    bodyWidth.value = bodyRef.value.clientWidth
    resizeObserver = new ResizeObserver((entries) => {
      bodyWidth.value = entries[0]?.contentRect.width ?? 0
    })
    resizeObserver.observe(bodyRef.value)
  }
  void load()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
})

// 供仪表盘刷新按钮联动调用
defineExpose({ reload: load })
</script>

<style scoped>
/* 格子边长由网格的 --cell-size 决定；圆角保持 4px，避免小格子看起来像圆点。 */
.heatmap-cell {
  width: var(--cell-size, 12px);
  height: var(--cell-size, 12px);
  border-radius: 4px;
}

/* 可点击的格子：悬停或键盘聚焦时放大并加外环 */
.heatmap-cell-interactive {
  cursor: pointer;
  transition: transform var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
}

.heatmap-cell-interactive:hover,
.heatmap-cell-interactive:focus-visible {
  transform: scale(1.25);
  outline: none;
  box-shadow: 0 0 0 1px theme('colors.gray.400');
}

:global(.dark) .heatmap-cell-interactive:hover,
:global(.dark) .heatmap-cell-interactive:focus-visible {
  box-shadow: 0 0 0 1px theme('colors.dark.300');
}

/* 今天与选中日的描边用中性色，和绿色色阶搭配；选中日更粗更深 */
.heatmap-cell-today {
  outline: 1px solid theme('colors.gray.400');
  outline-offset: 1px;
}

.heatmap-cell-selected {
  outline: 2px solid theme('colors.gray.900');
  outline-offset: 1px;
}

:global(.dark) .heatmap-cell-today {
  outline-color: theme('colors.dark.400');
}

:global(.dark) .heatmap-cell-selected {
  outline-color: theme('colors.dark.50');
}

/* 首次入场：按列错峰从小到大淡入，延迟由 --wave-delay 提供 */
.heatmap-cell-enter {
  animation: heatmap-cell-enter var(--dash-heatmap-enter-ms, 320ms) var(--motion-ease) var(--wave-delay, 0ms) both;
}

@keyframes heatmap-cell-enter {
  from {
    opacity: 0;
    transform: scale(0.4);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .heatmap-cell-enter {
    animation: none;
  }

  .heatmap-cell-interactive:hover,
  .heatmap-cell-interactive:focus-visible {
    transform: none;
  }
}
</style>
