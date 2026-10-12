<template>
  <div ref="containerRef" class="relative">
    <button
      type="button"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :class="[
        'input input-trigger',
        // 日期控件文字沿用既有的中性灰(比 .input 默认色略浅),保持现状视觉。
        'text-gray-700 dark:text-gray-300',
        isOpen && 'date-picker-trigger-open'
      ]"
      @click="toggle"
    >
      <span class="date-picker-icon">
        <Icon name="calendar" size="sm" />
      </span>
      <span class="date-picker-value">
        {{ displayValue }}
      </span>
      <span class="date-picker-chevron">
        <Icon
          name="chevronDown"
          size="sm"
          :class="['transition-transform duration-normal', isOpen && 'rotate-180']"
        />
      </span>
    </button>

    <MotionTransition name="dropdown-fade">
      <div v-if="isOpen" class="date-picker-dropdown" :style="dropdownStyle" role="dialog">
        <div class="date-picker-body">
          <!-- 快捷范围：桌面端纵向分组排列，窄屏折成可换行的 chip -->
          <div class="date-picker-presets">
            <div v-for="(group, index) in presetGroups" :key="index" class="date-picker-preset-group">
              <button
                v-for="preset in group"
                :key="preset.value"
                type="button"
                :class="['date-picker-preset', isPresetActive(preset) && 'date-picker-preset-active']"
                @click="selectPreset(preset)"
              >
                <span class="truncate">{{ t(preset.labelKey) }}</span>
                <Icon
                  v-if="isPresetActive(preset)"
                  name="check"
                  size="sm"
                  class="date-picker-preset-check"
                />
              </button>
            </div>
          </div>

          <!-- 自定义区间：单月日历，先点开始日期再点结束日期 -->
          <div class="date-picker-calendar">
            <div class="date-picker-calendar-header">
              <button
                type="button"
                class="date-picker-nav"
                :aria-label="t('dashboard.fork.datePicker.previousMonth')"
                @click="shiftMonth(-1)"
              >
                <Icon name="chevronLeft" size="sm" />
              </button>
              <span class="date-picker-month">{{ monthLabel }}</span>
              <button
                type="button"
                class="date-picker-nav"
                :aria-label="t('dashboard.fork.datePicker.nextMonth')"
                :disabled="!canGoNextMonth"
                @click="shiftMonth(1)"
              >
                <Icon name="chevronRight" size="sm" />
              </button>
            </div>

            <div class="date-picker-grid" @mouseleave="hoverDate = null">
              <span v-for="weekday in weekdayLabels" :key="weekday" class="date-picker-weekday">
                {{ weekday }}
              </span>
              <div
                v-for="cell in calendarCells"
                :key="cell.key"
                :class="[
                  'date-picker-day-cell',
                  cell.inRange && 'date-picker-day-in-range',
                  cell.isRangeStart && 'date-picker-day-range-start',
                  cell.isRangeEnd && 'date-picker-day-range-end'
                ]"
              >
                <button
                  v-if="cell.date"
                  type="button"
                  :disabled="cell.disabled"
                  :aria-label="cell.ariaLabel"
                  :aria-pressed="cell.isRangeStart || cell.isRangeEnd"
                  :class="[
                    'date-picker-day',
                    (cell.isRangeStart || cell.isRangeEnd) && 'date-picker-day-selected',
                    cell.isToday && 'date-picker-day-today'
                  ]"
                  @click="selectDay(cell.date)"
                  @mouseenter="hoverDate = cell.date"
                >
                  {{ cell.day }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 底栏：左侧显示待应用的区间，右侧取消与应用 -->
        <div class="date-picker-actions">
          <div class="date-picker-summary">
            <template v-if="selectingEnd">
              <span class="date-picker-summary-value">{{ formatSummaryDate(localStartDate) }}</span>
              <Icon name="arrowRight" size="xs" class="shrink-0" />
              <span>{{ t('dashboard.fork.datePicker.selectEndDate') }}</span>
            </template>
            <template v-else>
              <span class="date-picker-summary-value">{{ formatSummaryDate(localStartDate) }}</span>
              <Icon name="arrowRight" size="xs" class="shrink-0" />
              <span class="date-picker-summary-value">{{ formatSummaryDate(localEndDate) }}</span>
            </template>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <button type="button" class="btn btn-secondary btn-sm h-8" @click="cancel">
              {{ t('common.cancel') }}
            </button>
            <button type="button" class="btn btn-primary btn-sm h-8" :disabled="selectingEnd" @click="apply">
              {{ t('dates.apply') }}
            </button>
          </div>
        </div>
      </div>
    </MotionTransition>
  </div>
</template>

<script setup lang="ts">
// fork：仪表盘工具栏的日期范围选择器（移植自 TokenRouter components/common/DateRangePicker.vue）。
// 上游共享的 DateRangePicker 仍供其他页面使用，这里单独放一份，只给仪表盘工具栏用。
// 与 TokenRouter 的差异：后端只接受整天（YYYY-MM-DD），去掉了近 15 / 30 分钟两个分钟级快捷项及其时刻匹配逻辑。
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import MotionTransition from '@/components/motion/MotionTransition.vue'
import type { FloatingPanelPosition } from '@/utils/floatingPanel'
import { getAlignedPanelPosition } from './alignedPanelPosition'

interface DatePreset {
  labelKey: string
  value: string
  getRange: () => { start: string; end: string }
}

interface CalendarCell {
  key: string
  date: string | null
  day: number
  ariaLabel: string
  disabled: boolean
  isToday: boolean
  inRange: boolean
  isRangeStart: boolean
  isRangeEnd: boolean
}

// 打开弹层时记录的已生效状态，未应用就关闭时据此还原。
interface RangeSnapshot {
  start: string
  end: string
  preset: string | null
}

const props = defineProps<{
  startDate: string
  endDate: string
}>()

const emit = defineEmits<{
  (e: 'change', range: { startDate: string; endDate: string; preset: string | null }): void
}>()

const { t, locale } = useI18n()

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<FloatingPanelPosition | null>(null)
const localStartDate = ref(props.startDate)
const localEndDate = ref(props.endDate)
const activePreset = ref<string | null>(null)
const snapshot = ref<RangeSnapshot | null>(null)

// 日历当前展示月份的 1 日，格式 YYYY-MM-DD。
const viewMonth = ref('')
// 已点选开始日期、等待点选结束日期。
const selectingEnd = ref(false)
// 等待结束日期时鼠标悬停的日期，用于预览区间。
const hoverDate = ref<string | null>(null)

const dropdownWidth = 456
const dropdownMargin = 12

const dropdownStyle = computed(() => {
  const position = dropdownPosition.value
  if (!position) return {}
  return {
    left: `${position.left}px`,
    top: position.top === null ? 'auto' : `${position.top}px`,
    bottom: position.bottom === null ? 'auto' : `${position.bottom}px`,
    width: `${position.width}px`,
    maxHeight: `${position.maxHeight}px`
  }
})

const dateLocale = computed(() => (locale.value === 'zh' ? 'zh-CN' : 'en-US'))

// 按本地时区格式化为 YYYY-MM-DD。
const formatDateToString = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = () => formatDateToString(new Date())

// 用户时区落后于服务器时，服务器的“今天”可能是本地的明天，所以日历最多放开到明天。
const tomorrow = () => {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return formatDateToString(d)
}

const daysAgo = (days: number) => {
  const d = new Date()
  d.setDate(d.getDate() - days)
  return formatDateToString(d)
}

const presets: DatePreset[] = [
  {
    labelKey: 'dates.today',
    value: 'today',
    getRange: () => ({ start: today(), end: today() })
  },
  {
    labelKey: 'dates.yesterday',
    value: 'yesterday',
    getRange: () => ({ start: daysAgo(1), end: daysAgo(1) })
  },
  {
    labelKey: 'dates.last24Hours',
    value: 'last24Hours',
    getRange: () => {
      const end = new Date()
      const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
      return { start: formatDateToString(start), end: formatDateToString(end) }
    }
  },
  {
    labelKey: 'dates.last7Days',
    value: '7days',
    getRange: () => ({ start: daysAgo(6), end: today() })
  },
  {
    labelKey: 'dates.last14Days',
    value: '14days',
    getRange: () => ({ start: daysAgo(13), end: today() })
  },
  {
    labelKey: 'dates.last30Days',
    value: '30days',
    getRange: () => ({ start: daysAgo(29), end: today() })
  },
  {
    labelKey: 'dates.thisMonth',
    value: 'thisMonth',
    getRange: () => {
      const now = new Date()
      return { start: formatDateToString(new Date(now.getFullYear(), now.getMonth(), 1)), end: today() }
    }
  },
  {
    labelKey: 'dates.lastMonth',
    value: 'lastMonth',
    getRange: () => {
      const now = new Date()
      const start = formatDateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1))
      const end = formatDateToString(new Date(now.getFullYear(), now.getMonth(), 0))
      return { start, end }
    }
  }
]

// 下拉中的展示分组：小时级、单日、滚动天数、自然月。匹配顺序仍以 presets 为准。
const presetGroupValues = [
  ['last24Hours'],
  ['today', 'yesterday'],
  ['7days', '14days', '30days'],
  ['thisMonth', 'lastMonth']
]

const presetGroups = presetGroupValues.map((values) =>
  values
    .map((value) => presets.find((preset) => preset.value === value))
    .filter((preset): preset is DatePreset => !!preset)
)

const parseLocalDate = (dateStr: string): Date => new Date(`${dateStr.slice(0, 10)}T00:00:00`)

const formatDate = (dateStr: string): string => {
  return parseLocalDate(dateStr).toLocaleDateString(dateLocale.value, { month: 'short', day: 'numeric' })
}

// 弹层打开期间触发器继续显示已生效的范围，点应用后才切换文案。
const displayValue = computed(() => {
  const committed = isOpen.value && snapshot.value
    ? snapshot.value
    : { start: localStartDate.value, end: localEndDate.value, preset: activePreset.value }

  if (committed.preset) {
    const preset = presets.find((p) => p.value === committed.preset)
    if (preset) return t(preset.labelKey)
  }

  if (committed.start && committed.end) {
    if (committed.start === committed.end) {
      return formatDate(committed.start)
    }
    return `${formatDate(committed.start)} - ${formatDate(committed.end)}`
  }

  return t('dates.selectDateRange')
})

// 底栏摘要带年份。
const formatSummaryDate = (value: string): string => {
  if (!value) return '—'
  return parseLocalDate(value).toLocaleDateString(dateLocale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

const monthLabel = computed(() => {
  if (!viewMonth.value) return ''
  return parseLocalDate(viewMonth.value).toLocaleDateString(dateLocale.value, { year: 'numeric', month: 'long' })
})

// 中文日历以周一开头，英文以周日开头。
const weekStartsOn = computed(() => (locale.value === 'zh' ? 1 : 0))

const weekdayLabels = computed(() => {
  // 2023-01-01 是周日，以它为基准依次生成一周的星期缩写。
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(2023, 0, 1 + ((index + weekStartsOn.value) % 7))
    return date.toLocaleDateString(dateLocale.value, { weekday: 'narrow' })
  })
})

const canGoNextMonth = computed(() => {
  if (!viewMonth.value) return false
  const next = parseLocalDate(viewMonth.value)
  next.setMonth(next.getMonth() + 1)
  return formatDateToString(next) <= tomorrow()
})

// 当前高亮的区间。等待结束日期时按悬停位置预览，起止顺序自动对调。
const highlightRange = computed(() => {
  const start = localStartDate.value
  const end = localEndDate.value
  if (selectingEnd.value && hoverDate.value) {
    return start <= hoverDate.value
      ? { start, end: hoverDate.value }
      : { start: hoverDate.value, end: start }
  }
  return { start, end }
})

// 固定生成 6 周 42 格，不同月份切换时面板高度保持不变。
const calendarCells = computed<CalendarCell[]>(() => {
  if (!viewMonth.value) return []
  const first = parseLocalDate(viewMonth.value)
  const leading = (first.getDay() - weekStartsOn.value + 7) % 7
  const daysInMonth = new Date(first.getFullYear(), first.getMonth() + 1, 0).getDate()
  const { start, end } = highlightRange.value
  const todayKey = today()
  const maxDate = tomorrow()

  return Array.from({ length: 42 }, (_, index) => {
    const day = index - leading + 1
    if (day < 1 || day > daysInMonth) {
      return {
        key: `blank-${index}`,
        date: null,
        day: 0,
        ariaLabel: '',
        disabled: true,
        isToday: false,
        inRange: false,
        isRangeStart: false,
        isRangeEnd: false
      }
    }
    const current = new Date(first.getFullYear(), first.getMonth(), day)
    const date = formatDateToString(current)
    const hasRange = !!start && !!end
    return {
      key: date,
      date,
      day,
      ariaLabel: current.toLocaleDateString(dateLocale.value, { dateStyle: 'full' }),
      disabled: date > maxDate,
      isToday: date === todayKey,
      inRange: hasRange && start !== end && date >= start && date <= end,
      isRangeStart: hasRange && date === start,
      isRangeEnd: hasRange && date === end
    }
  })
})

const showMonthOf = (value: string) => {
  const base = value ? parseLocalDate(value) : new Date()
  viewMonth.value = formatDateToString(new Date(base.getFullYear(), base.getMonth(), 1))
}

const shiftMonth = (offset: number) => {
  const current = parseLocalDate(viewMonth.value)
  viewMonth.value = formatDateToString(new Date(current.getFullYear(), current.getMonth() + offset, 1))
}

// 第一次点击确定开始日期，第二次点击确定结束日期；反向点选时自动对调。
const selectDay = (date: string) => {
  if (!selectingEnd.value) {
    localStartDate.value = date
    localEndDate.value = date
    activePreset.value = null
    selectingEnd.value = true
    return
  }

  const start = localStartDate.value
  if (date < start) {
    localStartDate.value = date
    localEndDate.value = start
  } else {
    localEndDate.value = date
  }
  selectingEnd.value = false
  hoverDate.value = null
  onDateChange()
}

const isPresetActive = (preset: DatePreset): boolean => activePreset.value === preset.value

const selectPreset = (preset: DatePreset) => {
  const range = preset.getRange()
  localStartDate.value = range.start
  localEndDate.value = range.end
  activePreset.value = preset.value
  selectingEnd.value = false
  hoverDate.value = null
  showMonthOf(range.end)
}

// 当前起止日期正好等于某个快捷项时，把它标成选中。
const onDateChange = () => {
  activePreset.value = null
  for (const preset of presets) {
    const range = preset.getRange()
    if (range.start === localStartDate.value && range.end === localEndDate.value) {
      activePreset.value = preset.value
      break
    }
  }
}

const open = () => {
  snapshot.value = {
    start: localStartDate.value,
    end: localEndDate.value,
    preset: activePreset.value
  }
  selectingEnd.value = false
  hoverDate.value = null
  showMonthOf(localEndDate.value)
  isOpen.value = true
  updateDropdownPosition()
}

// 未点应用就关闭时丢弃本次改动，触发器文案保持已生效的范围。
const close = () => {
  if (!isOpen.value) return
  if (snapshot.value) {
    localStartDate.value = snapshot.value.start
    localEndDate.value = snapshot.value.end
    activePreset.value = snapshot.value.preset
  }
  selectingEnd.value = false
  hoverDate.value = null
  isOpen.value = false
}

const toggle = () => {
  if (isOpen.value) {
    close()
  } else {
    open()
  }
}

const cancel = () => {
  close()
}

const apply = () => {
  emit('change', {
    startDate: localStartDate.value,
    endDate: localEndDate.value,
    preset: activePreset.value
  })
  isOpen.value = false
}

// 触发器在视口右半侧时右缘对齐，否则左缘对齐；空间不足时由浮层定位函数翻转和夹取。
const updateDropdownPosition = () => {
  const trigger = containerRef.value?.getBoundingClientRect()
  if (!trigger) return
  const triggerCenter = trigger.left + trigger.width / 2
  dropdownPosition.value = getAlignedPanelPosition(trigger, window.innerWidth, window.innerHeight, {
    viewportPadding: dropdownMargin,
    maxWidth: dropdownWidth,
    maxHeightRatio: 0.85,
    align: triggerCenter > window.innerWidth / 2 ? 'right' : 'left',
    pinLeftOnMobile: false
  })
}

const handleClickOutside = (event: MouseEvent) => {
  if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
    close()
  }
}

const handleEscape = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && isOpen.value) {
    close()
  }
}

const handleViewportChange = () => {
  if (isOpen.value) {
    updateDropdownPosition()
  }
}

// 首次渲染前匹配初始范围的快捷项，触发器直接显示对应文案。
onDateChange()

// 父组件按应用的范围换算出新的起止日期后，同步到本地状态。
watch(
  () => props.startDate,
  (val) => {
    localStartDate.value = val
    onDateChange()
  }
)

watch(
  () => props.endDate,
  (val) => {
    localEndDate.value = val
    onDateChange()
  }
)

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleEscape)
  window.addEventListener('resize', handleViewportChange)
  window.addEventListener('scroll', handleViewportChange, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleEscape)
  window.removeEventListener('resize', handleViewportChange)
  window.removeEventListener('scroll', handleViewportChange, true)
})
</script>

<style scoped>
/* 触发器沿用 .input 的尺寸与边框配方，这里补 TokenRouter .input-trigger 的布局与交互增量；
   py-2 让高度与旁边的图标按钮（btn-icon）对齐，TokenRouter 的 .input 与 .btn 本就同高。 */
.input-trigger {
  @apply flex cursor-pointer items-center gap-2 py-2;
  @apply hover:border-black/20 dark:hover:border-dark-500;
}

/* 展开态描边与 Select 一致，品牌色只用于选中的日期和快捷范围。 */
.date-picker-trigger-open {
  @apply border-primary-900/10 ring-2 ring-black/10 dark:border-dark-400 dark:ring-white/[0.06];
}

.date-picker-icon {
  @apply text-gray-400 dark:text-dark-400;
}

.date-picker-value {
  @apply font-medium;
}

.date-picker-chevron {
  @apply text-gray-400 dark:text-dark-400;
}

.date-picker-dropdown {
  @apply fixed z-[100] flex flex-col;
  @apply bg-white dark:bg-dark-800;
  @apply rounded-xl;
  @apply border border-primary-900/10 dark:border-dark-600;
  @apply shadow-lg shadow-black/10 dark:shadow-black/30;
  @apply overflow-y-auto;
}

.date-picker-body {
  @apply flex flex-col sm:flex-row;
}

/* 窄屏时快捷范围横排换行，sm 起改为左侧纵向列表。 */
.date-picker-presets {
  @apply flex flex-wrap gap-1 p-2;
  @apply border-b border-primary-900/10 dark:border-dark-600;
  @apply sm:w-36 sm:shrink-0 sm:flex-col sm:flex-nowrap sm:gap-0 sm:border-b-0 sm:border-r;
}

.date-picker-preset-group {
  @apply contents sm:flex sm:flex-col sm:gap-0.5;
}

.date-picker-preset-group + .date-picker-preset-group {
  @apply sm:mt-1 sm:border-t sm:border-primary-900/10 sm:pt-1 sm:dark:border-dark-700;
}

.date-picker-preset {
  @apply flex items-center justify-between gap-2 rounded-lg px-2.5 py-1.5 text-left text-sm;
  @apply text-gray-700 dark:text-gray-300;
  @apply hover:bg-gray-100 dark:hover:bg-dark-700 dark:hover:text-primary-500;
  @apply transition-colors duration-fast;
  @apply max-sm:border max-sm:border-primary-900/10 max-sm:dark:border-dark-600;
}

.date-picker-preset-active {
  /* 与 Select 选中项同一配色：浅色灰底品牌字，深色淡品牌青底。 */
  @apply bg-gray-100 text-primary-700 dark:bg-primary-500/[0.08] dark:text-primary-500 dark:hover:bg-primary-500/[0.08];
  @apply font-medium;
}

.date-picker-preset-check {
  @apply hidden shrink-0 sm:block;
}

.date-picker-calendar {
  @apply min-w-0 flex-1 p-3;
}

.date-picker-calendar-header {
  @apply mb-2 flex items-center justify-between;
}

.date-picker-month {
  @apply text-sm font-semibold text-gray-900 dark:text-dark-50;
}

/* TokenRouter 的 .btn-icon-sm（32px 紧凑图标按钮）展开在这里。 */
.date-picker-nav {
  @apply inline-flex h-8 w-8 items-center justify-center rounded-lg p-0;
  @apply text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-dark-50;
  @apply transition-colors duration-fast;
  @apply disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent;
}

.date-picker-grid {
  @apply grid grid-cols-7 gap-y-1;
}

.date-picker-weekday {
  @apply flex h-8 items-center justify-center text-xs font-medium text-gray-400 dark:text-dark-400;
}

/* 区间底色画在单元格的伪元素上，相邻格子连成一条带；起止格只铺向区间内侧的一半。 */
.date-picker-day-cell {
  @apply relative flex h-9 items-center justify-center;
}

.date-picker-day-cell::before {
  content: '';
  @apply absolute inset-y-0 hidden bg-primary-100/70 dark:bg-primary-500/10;
}

.date-picker-day-in-range::before {
  @apply left-0 right-0 block;
}

.date-picker-day-in-range.date-picker-day-range-start::before {
  @apply left-1/2;
}

.date-picker-day-in-range.date-picker-day-range-end::before {
  @apply right-1/2;
}

.date-picker-day {
  @apply relative flex h-9 w-9 items-center justify-center rounded-lg text-sm tabular-nums;
  @apply text-gray-700 dark:text-gray-300;
  @apply hover:bg-gray-100 dark:hover:bg-dark-700;
  @apply transition-colors duration-fast;
  @apply disabled:cursor-not-allowed disabled:text-gray-300 disabled:hover:bg-transparent dark:disabled:text-dark-600;
}

.date-picker-day-in-range .date-picker-day {
  @apply text-primary-800 hover:bg-primary-200/70 dark:text-primary-400 dark:hover:bg-primary-500/15;
}

/* 今天用底部小圆点标记，不与选中态的实心底色冲突。 */
.date-picker-day-today::after {
  content: '';
  @apply absolute bottom-1 left-1/2 h-1 w-1 -translate-x-1/2 rounded-full bg-primary-500;
}

.date-picker-day.date-picker-day-selected {
  /* 起止日期与主按钮同色。 */
  @apply bg-primary-600 font-semibold text-white hover:bg-primary-700;
  @apply dark:text-white dark:hover:bg-primary-700;
}

.date-picker-day-selected.date-picker-day-today::after {
  @apply bg-white;
}

.date-picker-actions {
  @apply flex flex-wrap items-center justify-between gap-2 px-3 py-2.5;
  @apply border-t border-primary-900/10 dark:border-dark-600;
}

.date-picker-summary {
  @apply flex min-w-0 items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400;
}

.date-picker-summary-value {
  @apply truncate font-medium text-gray-700 dark:text-dark-100;
}
</style>
