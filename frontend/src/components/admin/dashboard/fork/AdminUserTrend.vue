<template>
  <!-- User Usage Trend (Full Width) -->
  <div class="card p-4" data-testid="admin-user-trend">
    <!-- 本项目保留：标题右侧可切换按 Token / 按实际消费排出前 12 名 -->
    <div class="mb-4 flex items-center justify-between gap-3">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ t('admin.dashboard.recentUsage') }} (Top 12)
      </h3>
      <div v-segmented class="segmented">
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
      <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
      <div
        v-else
        class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('admin.dashboard.noDataAvailable') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的用户使用趋势（Top 12），图表逐项移植自 TokenRouter views/admin/DashboardView.vue 的
// lineOptions / userTrendChartData（品牌色刻度与网格、外置提示按数值从大到小排序）。
// 本项目保留按 Token / 实际消费切换；数据由页面加载，这里只负责展示。
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
  type TooltipItem,
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
import DashboardChartSkeleton from '@/components/user/dashboard/fork/DashboardChartSkeleton.vue'
import { formatTokens } from './format'

// Register Chart.js components
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

onBeforeUnmount(hideExternalTooltip)

type UserTrendMetric = 'tokens' | 'actual_cost'
const METRICS: readonly UserTrendMetric[] = ['tokens', 'actual_cost']

const props = withDefaults(defineProps<{
  trend: UserUsageTrendPoint[]
  metric: UserTrendMetric
  loading?: boolean
}>(), {
  loading: false,
})

const emit = defineEmits<{
  'update:metric': [value: UserTrendMetric]
}>()

const { t } = useI18n()

// 与 TokenRouter 相同：浅色时刻度/网格用品牌色系（TokenRouter 的 primary-900 / primary-100），
// 这里换成本项目同档位的 primary-900 / primary-100；深色网格按卡片底色取同样反差的 #3d3d3d。主题切换即刻重绘。
const { isDark } = useChartTheme()
const chartColors = computed(() => ({
  text: isDark.value ? '#D4D4D8' : '#134e4a',
  grid: isDark.value ? '#3d3d3d' : '#ccfbf1',
}))

// 按实际消费时沿用原页面的金额格式：$ + 两到四位小数，千以上缩写 K
const formatCost = (value: number): string => {
  if (value >= 1000) return `$${(value / 1000).toFixed(2)}K`
  if (value >= 1) return `$${value.toFixed(2)}`
  if (value >= 0.01) return `$${value.toFixed(3)}`
  return `$${value.toFixed(4)}`
}

const formatUserTrendValue = (value: number): string =>
  props.metric === 'tokens' ? formatTokens(value) : formatCost(value)

const tooltipValue = (item: TooltipItem<'line'>): number =>
  typeof item?.raw === 'number' ? item.raw : Number(item?.parsed?.y ?? 0)

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: CHART_LEGEND_FONT_SIZE
        }
      }
    },
    tooltip: {
      enabled: false,
      external: externalTooltipHandler,
      itemSort: (a: TooltipItem<'line'>, b: TooltipItem<'line'>) => tooltipValue(b) - tooltipValue(a),
      callbacks: {
        label: (context: TooltipItem<'line'>) => {
          return `${context.dataset.label}: ${formatUserTrendValue(Number(context.raw))}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: CHART_TICK_FONT_SIZE
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: CHART_TICK_FONT_SIZE
        },
        callback: (value: string | number) => formatUserTrendValue(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!props.trend?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  props.trend.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, props.metric === 'tokens' ? point.tokens : point.actual_cost)
  })

  const sortedDates = Array.from(allDates).sort()

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: CHART_PALETTE[idx % CHART_PALETTE.length],
    backgroundColor: `${CHART_PALETTE[idx % CHART_PALETTE.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return {
    labels: sortedDates,
    datasets
  }
})
</script>
