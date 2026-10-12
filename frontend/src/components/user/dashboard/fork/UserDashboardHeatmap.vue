<template>
  <section class="card p-5" :aria-busy="loading" data-testid="usage-heatmap">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.fork.heatmap.title') }}</h3>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ summaryLabel }}</p>
      </div>
      <div class="flex items-center gap-1 text-[11px] text-gray-500 dark:text-gray-400" aria-hidden="true">
        <span>{{ t('dashboard.fork.heatmap.less') }}</span>
        <span v-for="level in 5" :key="level" class="h-3 w-3 rounded-[3px]" :class="LEVEL_CLASSES[level - 1]"></span>
        <span>{{ t('dashboard.fork.heatmap.more') }}</span>
      </div>
    </div>

    <div class="overflow-x-auto pb-1">
      <div
        class="grid w-max gap-[3px]"
        :style="{ gridTemplateColumns: `auto repeat(${WEEKS}, 12px)`, gridTemplateRows: `14px repeat(7, 12px)` }"
        @mouseleave="hovered = null"
      >
        <span
          v-for="month in monthLabels"
          :key="`m-${month.column}`"
          class="whitespace-nowrap text-[10px] leading-[14px] text-gray-400 dark:text-gray-500"
          :style="{ gridColumn: month.column + 2, gridRow: 1 }"
        >{{ month.label }}</span>
        <span
          v-for="weekday in WEEKDAY_ROWS"
          :key="`w-${weekday}`"
          class="pr-1.5 text-[10px] leading-3 text-gray-400 dark:text-gray-500"
          :style="{ gridColumn: 1, gridRow: weekday + 2 }"
        >{{ weekdayLabel(weekday) }}</span>
        <span
          v-for="day in days"
          :key="day.date"
          class="heatmap-cell"
          :class="[day.future ? 'invisible' : LEVEL_CLASSES[day.level], { 'heatmap-cell-today': day.today }]"
          :style="{ gridColumn: day.column + 2, gridRow: day.row + 2, '--wave-delay': `${Math.min(day.column * 30, 480)}ms` }"
          :data-date="day.date"
          :data-level="day.level"
          :title="day.future ? undefined : cellTitle(day)"
          @mouseenter="hovered = day.future ? null : day"
        ></span>
      </div>
    </div>

    <p class="mt-3 h-4 text-xs text-gray-500 dark:text-gray-400" aria-live="polite">
      <template v-if="hovered">{{ cellTitle(hovered) }}</template>
    </p>
  </section>
</template>

<script setup lang="ts">
// fork：用量热力图（参考 TokenRouter UserDashboardHeatmap）。最近 13 周按天的 token 用量，
// 颜色按非零日用量的四分位分档；首次取数后格子按列错峰入场，总延迟不超过 480ms。
// usage_logs 默认只保留 90 天，所以只看 13 周。
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { usageAPI } from '@/api/usage'
import type { TrendDataPoint } from '@/types'
import { formatDateLocalInput } from '@/utils/format'
import { formatCompact } from '@/utils/compactNumber'

interface HeatmapDay {
  date: string
  column: number
  row: number
  tokens: number
  requests: number
  level: number
  future: boolean
  today: boolean
}

const WEEKS = 13
const WEEKDAY_ROWS = [0, 2, 4]
const LEVEL_CLASSES = [
  'bg-gray-100 dark:bg-dark-800',
  'bg-primary-200 dark:bg-primary-900/70',
  'bg-primary-300 dark:bg-primary-700',
  'bg-primary-500 dark:bg-primary-500',
  'bg-primary-700 dark:bg-primary-300',
]

const { t, locale } = useI18n()
const loading = ref(true)
const trend = ref<TrendDataPoint[]>([])
const hovered = ref<HeatmapDay | null>(null)

// 网格从 13 周前的周一开始，到本周日结束；今天之后的格子留空
const range = computed(() => {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const mondayOffset = (today.getDay() + 6) % 7
  const start = new Date(today)
  start.setDate(today.getDate() - mondayOffset - (WEEKS - 1) * 7)
  return { start, today }
})

const days = computed<HeatmapDay[]>(() => {
  const byDate = new Map(trend.value.map(point => [point.date, point]))
  const nonZero = trend.value.map(point => point.total_tokens).filter(value => value > 0).sort((a, b) => a - b)
  const quantile = (q: number) => nonZero[Math.floor(q * (nonZero.length - 1))] ?? 0
  const levelFor = (tokens: number) => {
    if (tokens <= 0) return 0
    if (tokens <= quantile(0.25)) return 1
    if (tokens <= quantile(0.5)) return 2
    if (tokens <= quantile(0.75)) return 3
    return 4
  }

  const { start, today } = range.value
  const todayKey = formatDateLocalInput(today)
  const result: HeatmapDay[] = []
  const cursor = new Date(start)
  for (let index = 0; index < WEEKS * 7; index++) {
    const date = formatDateLocalInput(cursor)
    const point = byDate.get(date)
    const tokens = point?.total_tokens ?? 0
    result.push({
      date,
      column: Math.floor(index / 7),
      row: index % 7,
      tokens,
      requests: point?.requests ?? 0,
      level: levelFor(tokens),
      future: cursor > today,
      today: date === todayKey,
    })
    cursor.setDate(cursor.getDate() + 1)
  }
  return result
})

const monthLabels = computed(() => {
  const labels: Array<{ column: number; label: string }> = []
  let lastMonth = -1
  for (const day of days.value) {
    if (day.row !== 0) continue
    const month = new Date(`${day.date}T00:00:00`).getMonth()
    if (month !== lastMonth) {
      labels.push({ column: day.column, label: new Date(`${day.date}T00:00:00`).toLocaleDateString(locale.value, { month: 'short' }) })
      lastMonth = month
    }
  }
  return labels
})

const summaryLabel = computed(() => {
  const active = days.value.filter(day => day.tokens > 0).length
  const total = days.value.reduce((sum, day) => sum + day.tokens, 0)
  return t('dashboard.fork.heatmap.summary', { days: active, tokens: formatCompact(total) })
})

function weekdayLabel(row: number): string {
  // 2026-01-05 是周一，按行号往后推得到本地化的星期名
  const date = new Date(2026, 0, 5 + row)
  return date.toLocaleDateString(locale.value, { weekday: 'short' })
}

function cellTitle(day: HeatmapDay): string {
  const label = new Date(`${day.date}T00:00:00`).toLocaleDateString(locale.value, { month: 'short', day: 'numeric', weekday: 'short' })
  if (day.tokens <= 0) return t('dashboard.fork.heatmap.noUsage', { date: label })
  return t('dashboard.fork.heatmap.cell', { date: label, tokens: formatCompact(day.tokens), requests: day.requests })
}

onMounted(async () => {
  try {
    const { start, today } = range.value
    const res = await usageAPI.getDashboardTrend({
      start_date: formatDateLocalInput(start),
      end_date: formatDateLocalInput(today),
      granularity: 'day',
    })
    trend.value = res.trend || []
  } catch {
    trend.value = []
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.heatmap-cell {
  @apply h-3 w-3 rounded-[3px];
  animation: heatmap-cell-enter 320ms var(--motion-ease) backwards;
  animation-delay: var(--wave-delay, 0ms);
  transition: transform var(--motion-fast) var(--motion-ease);
}

.heatmap-cell:hover {
  transform: scale(1.25);
  @apply ring-1 ring-gray-400 dark:ring-gray-300;
}

.heatmap-cell-today {
  @apply ring-1 ring-primary-500/70;
}

@keyframes heatmap-cell-enter {
  from {
    opacity: 0;
    transform: scale(0.4);
  }
}

@media (prefers-reduced-motion: reduce) {
  .heatmap-cell {
    animation: none;
  }

  .heatmap-cell:hover {
    transform: none;
  }
}
</style>
