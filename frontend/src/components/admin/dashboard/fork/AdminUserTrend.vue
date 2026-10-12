<template>
  <div class="card p-4" data-testid="admin-user-trend">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ t('admin.dashboard.recentUsage') }} (Top 12)
      </h3>
      <!-- 本项目保留：按 Token 或按实际消费排出前 12 名 -->
      <div v-segmented class="segmented" role="group" :aria-label="t('admin.dashboard.recentUsage')">
        <button
          v-for="option in METRICS"
          :key="option"
          type="button"
          class="segmented-item px-2.5 py-1 text-xs"
          :class="{ 'segmented-item-active': metric === option }"
          :aria-pressed="metric === option"
          @click="emit('update:metric', option)"
        >{{ t(option === 'tokens' ? 'admin.dashboard.tokens' : 'admin.dashboard.actualSpending') }}</button>
      </div>
    </div>
    <div class="h-64">
      <DashboardChartSkeleton v-if="loading" height="100%" />
      <Line v-else-if="chartData" :data="chartData" :options="lineOptions" />
      <div v-else class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.dashboard.noDataAvailable') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的用户使用趋势（Top 12），图表样式移植自 TokenRouter views/admin/DashboardView.vue：
// 加载骨架、随主题切换的刻度色、外置提示（按数值从大到小排序）。数据由页面加载，这里只负责展示。
import { computed, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler,
  type TooltipItem as ChartTooltipItem,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { vSegmented } from '@/directives/segmented'
import type { UserUsageTrendPoint } from '@/types'
import {
  CHART_LEGEND_FONT_SIZE,
  CHART_PALETTE,
  CHART_TICK_FONT_SIZE,
  useChartTheme,
} from '@/components/user/dashboard/fork/chartTheme'
import { externalTooltipHandler, hideExternalTooltip } from '@/components/user/dashboard/fork/chartExternalTooltip'
import DashboardChartSkeleton from './DashboardChartSkeleton.vue'
import { formatCost, formatDayLabel, formatHourLabel, formatTokens, toFiniteNumber } from './format'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)
onBeforeUnmount(hideExternalTooltip)

type UserTrendMetric = 'tokens' | 'actual_cost'
const METRICS: readonly UserTrendMetric[] = ['tokens', 'actual_cost']

const props = withDefaults(defineProps<{
  trend: UserUsageTrendPoint[]
  metric: UserTrendMetric
  loading?: boolean
  granularity?: 'day' | 'hour'
}>(), {
  loading: false,
  granularity: 'hour',
})

const emit = defineEmits<{
  'update:metric': [value: UserTrendMetric]
}>()

const { t } = useI18n()
const { colors: themeColors } = useChartTheme()

const formatValue = (value: number): string =>
  props.metric === 'tokens' ? formatTokens(value) : formatCost(value)

const displayName = (point: UserUsageTrendPoint): string =>
  point.username?.trim() || point.email?.trim() || t('admin.redeem.userPrefix', { id: point.user_id })

const sortedDates = computed(() => Array.from(new Set(props.trend.map((point) => point.date))).sort())

const chartData = computed(() => {
  if (!props.trend?.length) return null

  // 按 user_id 分组，同名用户不会被合并
  const groups = new Map<number, { name: string; data: Map<string, number> }>()
  for (const point of props.trend) {
    let group = groups.get(point.user_id)
    if (!group) {
      group = { name: displayName(point), data: new Map() }
      groups.set(point.user_id, group)
    }
    group.data.set(point.date, toFiniteNumber(props.metric === 'tokens' ? point.tokens : point.actual_cost))
  }

  return {
    labels: sortedDates.value.map((date) => (props.granularity === 'hour' ? formatHourLabel(date) : formatDayLabel(date))),
    datasets: Array.from(groups.values()).map((group, index) => {
      const color = CHART_PALETTE[index % CHART_PALETTE.length]
      return {
        label: group.name,
        data: sortedDates.value.map((date) => group.data.get(date) || 0),
        borderColor: color,
        backgroundColor: `${color}20`,
        fill: false,
        tension: 0.3,
      }
    }),
  }
})

type TooltipItem = ChartTooltipItem<'line'>
const itemValue = (item: TooltipItem): number =>
  typeof item.raw === 'number' ? item.raw : Number(item.parsed?.y ?? 0)

const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: themeColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: { size: CHART_LEGEND_FONT_SIZE },
      },
    },
    tooltip: {
      enabled: false,
      external: externalTooltipHandler,
      itemSort: (a: TooltipItem, b: TooltipItem) => itemValue(b) - itemValue(a),
      callbacks: {
        title: (items: TooltipItem[]) => sortedDates.value[items[0]?.dataIndex ?? -1] || '',
        label: (context: TooltipItem) => `${context.dataset.label}: ${formatValue(Number(context.raw))}`,
      },
    },
  },
  scales: {
    x: {
      grid: { color: themeColors.value.grid },
      ticks: {
        color: themeColors.value.text,
        autoSkip: true,
        maxRotation: props.granularity === 'hour' ? 0 : 50,
        minRotation: 0,
        font: { size: CHART_TICK_FONT_SIZE },
      },
    },
    y: {
      grid: { color: themeColors.value.grid },
      ticks: {
        color: themeColors.value.text,
        font: { size: CHART_TICK_FONT_SIZE },
        callback: (value: string | number) => formatValue(Number(value)),
      },
    },
  },
}))
</script>
