<template>
  <div class="card p-4" data-testid="admin-token-trend">
    <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
      {{ t('admin.dashboard.tokenUsageTrend') }}
    </h3>
    <DashboardChartSkeleton v-if="loading" variant="plot" />
    <div v-else-if="chartData" class="h-48">
      <Line :data="chartData" :options="lineOptions" />
    </div>
    <div v-else class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的 Token 使用趋势（移植自 TokenRouter components/charts/TokenUsageTrend.vue）。
// 加载骨架、随主题切换的刻度色、外置提示；小时粒度横轴只显示时分，完整时间留在提示标题里。
// 上游的 TokenUsageTrend.vue 仍给用量页使用，不改它。
import { computed, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import type { TrendDataPoint } from '@/types'
import {
  CHART_LEGEND_FONT_SIZE,
  CHART_SERIES_COLORS,
  CHART_TICK_FONT_SIZE,
  useChartTheme,
} from '@/components/user/dashboard/fork/chartTheme'
import { externalTooltipHandler, hideExternalTooltip } from '@/components/user/dashboard/fork/chartExternalTooltip'
import { formatUsd } from '@/components/user/dashboard/fork/format'
import DashboardChartSkeleton from './DashboardChartSkeleton.vue'
import { formatDayLabel, formatHourLabel, formatTokens } from './format'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)
onBeforeUnmount(hideExternalTooltip)

const props = withDefaults(defineProps<{
  trendData: TrendDataPoint[]
  loading?: boolean
  granularity?: 'day' | 'hour'
}>(), {
  loading: false,
  granularity: 'day',
})

const { t } = useI18n()
const { colors: themeColors } = useChartTheme()

const formatPercent = (value: number): string => {
  const displayValue = value < 100 ? Math.floor(value * 100) / 100 : value
  return `${displayValue.toFixed(2)}%`
}

const series = (label: string, color: string, pick: (point: TrendDataPoint) => number) => ({
  label,
  data: props.trendData.map(pick),
  borderColor: color,
  backgroundColor: `${color}20`,
  fill: true,
  tension: 0.3,
})

const chartData = computed(() => {
  if (!props.trendData?.length) return null
  return {
    labels: props.trendData.map((point) =>
      props.granularity === 'hour' ? formatHourLabel(point.date) : formatDayLabel(point.date),
    ),
    datasets: [
      series('Input', CHART_SERIES_COLORS.input, (d) => d.input_tokens),
      series('Output', CHART_SERIES_COLORS.output, (d) => d.output_tokens),
      series('Cache Creation', CHART_SERIES_COLORS.cacheCreation, (d) => d.cache_creation_tokens),
      series('Cache Read', CHART_SERIES_COLORS.cacheRead, (d) => d.cache_read_tokens),
      {
        label: 'Cached Input %',
        data: props.trendData.map((d) => {
          // 后端的 input_tokens 已扣除 cache_read，命中率的分母取输入侧的全部 token
          const inputSideTokens = d.input_tokens + d.cache_creation_tokens + d.cache_read_tokens
          return inputSideTokens > 0 ? (d.cache_read_tokens / inputSideTokens) * 100 : 0
        }),
        borderColor: CHART_SERIES_COLORS.cacheHitRate,
        backgroundColor: `${CHART_SERIES_COLORS.cacheHitRate}20`,
        fill: false,
        tension: 0.3,
        yAxisID: 'yPercent',
      },
    ],
  }
})

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
      callbacks: {
        title: (items: Array<{ dataIndex: number }>) => {
          const index = items[0]?.dataIndex
          return index !== undefined ? props.trendData[index]?.date || '' : ''
        },
        label: (context: { dataset: { label?: string; yAxisID?: string }; raw: unknown }) => {
          const value = Number(context.raw)
          if (context.dataset.yAxisID === 'yPercent') return `${context.dataset.label}: ${formatPercent(value)}`
          return `${context.dataset.label}: ${formatTokens(value)}`
        },
        footer: (items: Array<{ dataIndex: number }>) => {
          const point = props.trendData[items[0]?.dataIndex ?? -1]
          if (!point) return ''
          return `Actual: ${formatUsd(point.actual_cost, 4)} | Standard: ${formatUsd(point.cost, 4)}`
        },
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
        callback: (value: string | number) => formatTokens(Number(value)),
      },
    },
    yPercent: {
      position: 'right' as const,
      min: 0,
      max: 100,
      grid: { drawOnChartArea: false },
      ticks: {
        color: CHART_SERIES_COLORS.cacheHitRate,
        font: { size: CHART_TICK_FONT_SIZE },
        callback: (value: string | number) => `${value}%`,
      },
    },
  },
}))
</script>
