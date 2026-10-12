<template>
  <div class="card p-4" data-testid="admin-model-distribution">
    <div class="mb-4 flex items-center justify-between gap-3">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ activeView === 'model_distribution'
          ? t('admin.dashboard.modelDistribution')
          : t('admin.dashboard.spendingRankingTitle') }}
      </h3>
      <div v-segmented class="segmented" role="group" :aria-label="t('admin.dashboard.modelDistribution')">
        <button
          type="button"
          class="segmented-item px-2.5 py-1 text-xs"
          :class="{ 'segmented-item-active': activeView === 'model_distribution' }"
          :aria-pressed="activeView === 'model_distribution'"
          @click="activeView = 'model_distribution'"
        >
          {{ t('admin.dashboard.viewModelDistribution') }}
        </button>
        <button
          type="button"
          class="segmented-item px-2.5 py-1 text-xs"
          :class="{ 'segmented-item-active': activeView === 'spending_ranking' }"
          :aria-pressed="activeView === 'spending_ranking'"
          @click="activeView = 'spending_ranking'"
        >
          {{ t('admin.dashboard.viewSpendingRanking') }}
        </button>
      </div>
    </div>

    <DashboardChartSkeleton v-if="activeView === 'model_distribution' && loading" variant="distribution" />
    <div
      v-else-if="activeView === 'model_distribution' && sortedModels.length > 0 && chartData"
      class="flex flex-col items-center gap-4 sm:flex-row sm:items-start sm:gap-6"
    >
      <div class="h-48 w-48 shrink-0">
        <Doughnut :data="chartData" :options="doughnutOptions" />
      </div>
      <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto">
        <table class="w-full text-xs tabular-nums">
          <thead>
            <tr class="text-gray-500 dark:text-gray-400">
              <th class="pb-2 text-left">{{ t('admin.dashboard.model') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.requests') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.tokens') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.actual') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.accountCost') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.standard') }}</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="model in sortedModels" :key="model.model">
              <!-- 整行可点击展开按用户拆分；模型名随悬停着色，箭头旋转 -->
              <tr
                class="group/breakdown cursor-pointer border-t border-gray-100 transition-colors hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/40"
                :aria-expanded="expandedModel === model.model"
                data-testid="model-row"
                @click="toggleBreakdown(model.model)"
              >
                <td
                  class="max-w-[100px] truncate py-1.5 font-medium text-gray-900 group-hover/breakdown:text-primary-600 dark:text-white dark:group-hover/breakdown:text-primary-400"
                  :title="model.model"
                >
                  <span class="inline-flex items-center gap-1">
                    <Icon
                      name="chevronRight"
                      size="xs"
                      class="h-3 w-3 shrink-0 transition-transform duration-normal ease-standard"
                      :class="{ 'rotate-90': expandedModel === model.model }"
                    />
                    {{ model.model }}
                  </span>
                </td>
                <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">{{ formatNumber(model.requests) }}</td>
                <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">{{ formatTokens(model.total_tokens) }}</td>
                <td class="py-1.5 text-right text-green-600 dark:text-green-400">{{ formatCost(model.actual_cost) }}</td>
                <td class="py-1.5 text-right text-orange-500 dark:text-orange-400">{{ formatCost(model.account_cost) }}</td>
                <td class="py-1.5 text-right text-gray-400 dark:text-gray-500">{{ formatCost(model.cost) }}</td>
              </tr>
              <ExpandableTableRow :open="expandedModel === model.model" :colspan="6">
                <UserBreakdownSubTable :items="breakdownItems" :loading="breakdownLoading" />
              </ExpandableTableRow>
            </template>
          </tbody>
        </table>
      </div>
    </div>
    <div
      v-else-if="activeView === 'model_distribution'"
      class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
    >
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>

    <DashboardChartSkeleton v-else-if="rankingLoading" variant="distribution" />
    <div
      v-else-if="rankingError"
      class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
    >
      {{ t('admin.dashboard.failedToLoad') }}
    </div>
    <!-- 两个视图在宽屏都顶部对齐 -->
    <div
      v-else-if="rankingDisplayItems.length > 0 && rankingChartData"
      class="flex flex-col items-center gap-4 sm:flex-row sm:items-start sm:gap-6"
    >
      <div class="h-48 w-48 shrink-0">
        <Doughnut :data="rankingChartData" :options="rankingDoughnutOptions" />
      </div>
      <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto">
        <table class="w-full text-xs tabular-nums">
          <thead>
            <tr class="text-gray-500 dark:text-gray-400">
              <th class="pb-2 text-left">{{ t('admin.dashboard.spendingRankingUser') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.spendingRankingRequests') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.spendingRankingTokens') }}</th>
              <th class="pb-2 text-right">{{ t('admin.dashboard.spendingRankingSpend') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(item, index) in rankingDisplayItems"
              :key="item.isOther ? 'others' : `${item.user_id}-${index}`"
              class="border-t border-gray-100 transition-colors dark:border-dark-700"
              :class="item.isOther
                ? 'bg-gray-50/70 dark:bg-dark-700/20'
                : 'cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-700/40'"
              data-testid="ranking-row"
              @click="item.isOther ? undefined : emit('ranking-click', item)"
            >
              <td class="py-1.5">
                <div class="flex min-w-0 items-center gap-2">
                  <span class="shrink-0 text-xs font-semibold text-gray-500 dark:text-gray-400">
                    {{ item.isOther ? 'Σ' : `#${index + 1}` }}
                  </span>
                  <span
                    class="block max-w-[140px] truncate font-medium text-gray-900 dark:text-white"
                    :title="rankingRowLabel(item)"
                  >
                    {{ rankingRowLabel(item) }}
                  </span>
                </div>
              </td>
              <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">{{ formatNumber(item.requests) }}</td>
              <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">{{ formatTokens(item.tokens) }}</td>
              <td class="py-1.5 text-right text-green-600 dark:text-green-400">{{ formatCost(item.actual_cost) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div v-else class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的「模型分布 / 用户消费榜」卡片。
// 移植自 TokenRouter components/charts/ModelDistributionChart.vue：分段切换带滑块、加载骨架、
// 展开行带收起动画、圆环提示挂在 body 上。上游的 ModelDistributionChart.vue 仍给用量页使用，不改它。
// 本项目保留「成本」（账号成本）列。
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js'
import { Doughnut } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import UserBreakdownSubTable from '@/components/charts/UserBreakdownSubTable.vue'
import ExpandableTableRow from '@/components/motion/ExpandableTableRow.vue'
import { vSegmented } from '@/directives/segmented'
import { getUserBreakdown } from '@/api/admin/dashboard'
import type { ModelStat, UserBreakdownItem, UserSpendingRankingItem } from '@/types'
import { CHART_OTHER_COLOR, CHART_PALETTE } from '@/components/user/dashboard/fork/chartTheme'
import { externalTooltipHandler, hideExternalTooltip } from '@/components/user/dashboard/fork/chartExternalTooltip'
import DashboardChartSkeleton from './DashboardChartSkeleton.vue'
import { formatCost, formatNumber, formatTokens, toFiniteNumber } from './format'

ChartJS.register(ArcElement, Tooltip, Legend)
onBeforeUnmount(hideExternalTooltip)

type RankingDisplayItem = UserSpendingRankingItem & { isOther?: boolean }

const props = withDefaults(defineProps<{
  modelStats: ModelStat[]
  rankingItems?: UserSpendingRankingItem[]
  rankingTotalActualCost?: number
  rankingTotalRequests?: number
  rankingTotalTokens?: number
  loading?: boolean
  rankingLoading?: boolean
  rankingError?: boolean
  startDate?: string
  endDate?: string
}>(), {
  rankingItems: () => [],
  rankingTotalActualCost: 0,
  rankingTotalRequests: 0,
  rankingTotalTokens: 0,
  loading: false,
  rankingLoading: false,
  rankingError: false,
  startDate: undefined,
  endDate: undefined,
})

const emit = defineEmits<{
  'ranking-click': [item: UserSpendingRankingItem]
}>()

const { t } = useI18n()
const activeView = ref<'model_distribution' | 'spending_ranking'>('model_distribution')

// ---- 按用户拆分 ----
const expandedModel = ref<string | null>(null)
const breakdownItems = ref<UserBreakdownItem[]>([])
const breakdownLoading = ref(false)
let breakdownSeq = 0

const toggleBreakdown = async (model: string) => {
  if (expandedModel.value === model) {
    expandedModel.value = null
    return
  }
  const seq = ++breakdownSeq
  expandedModel.value = model
  breakdownLoading.value = true
  breakdownItems.value = []
  try {
    const res = await getUserBreakdown({
      start_date: props.startDate,
      end_date: props.endDate,
      model,
      model_source: 'requested',
    })
    if (seq !== breakdownSeq) return
    breakdownItems.value = res.users || []
  } catch {
    if (seq !== breakdownSeq) return
    breakdownItems.value = []
  } finally {
    if (seq === breakdownSeq) breakdownLoading.value = false
  }
}

// ---- 模型分布 ----
const sortedModels = computed(() =>
  [...(props.modelStats || [])].sort((a, b) => toFiniteNumber(b.total_tokens) - toFiniteNumber(a.total_tokens)),
)

// 扇区与提示共用同一组数值，保证扇区大小和占比一致
const chartValues = computed(() => sortedModels.value.map((m) => toFiniteNumber(m.total_tokens)))

const chartData = computed(() => {
  if (!sortedModels.value.length) return null
  return {
    labels: sortedModels.value.map((m) => m.model),
    datasets: [{
      data: chartValues.value,
      backgroundColor: sortedModels.value.map((_, i) => CHART_PALETTE[i % CHART_PALETTE.length]),
      borderWidth: 0,
    }],
  }
})

// ---- 用户消费榜 ----
const rankingUserLabel = (item: UserSpendingRankingItem): string => {
  if (item.username?.trim()) return item.username.trim()
  if (item.email?.trim()) return item.email.trim()
  return t('admin.redeem.userPrefix', { id: item.user_id })
}

const rankingRowLabel = (item: RankingDisplayItem): string =>
  item.isOther ? t('admin.dashboard.spendingRankingOther') : rankingUserLabel(item)

// 榜单之外的消费合成一行「其他」
const otherRankingItem = computed<RankingDisplayItem | null>(() => {
  if (!props.rankingItems?.length) return null
  const sum = (pick: (item: UserSpendingRankingItem) => unknown) =>
    props.rankingItems.reduce((total, item) => total + toFiniteNumber(pick(item)), 0)

  const otherActualCost = Math.max(props.rankingTotalActualCost - sum((item) => item.actual_cost), 0)
  const otherRequests = Math.max(props.rankingTotalRequests - sum((item) => item.requests), 0)
  const otherTokens = Math.max(props.rankingTotalTokens - sum((item) => item.tokens), 0)
  if (otherActualCost <= 0.000001 && otherRequests <= 0 && otherTokens <= 0) return null

  return {
    user_id: 0,
    email: '',
    username: '',
    actual_cost: otherActualCost,
    requests: otherRequests,
    tokens: otherTokens,
    isOther: true,
  }
})

const rankingDisplayItems = computed<RankingDisplayItem[]>(() => {
  if (!props.rankingItems?.length) return []
  return otherRankingItem.value ? [...props.rankingItems, otherRankingItem.value] : [...props.rankingItems]
})

const rankingValues = computed(() => rankingDisplayItems.value.map((item) => toFiniteNumber(item.actual_cost)))

const rankingChartData = computed(() => {
  if (!props.rankingItems?.length) return null
  return {
    labels: rankingDisplayItems.value.map((item, index) =>
      item.isOther ? t('admin.dashboard.spendingRankingOther') : `#${index + 1} ${rankingUserLabel(item)}`,
    ),
    datasets: [{
      data: rankingValues.value,
      backgroundColor: rankingDisplayItems.value.map((item, i) =>
        item.isOther ? CHART_OTHER_COLOR : CHART_PALETTE[i % CHART_PALETTE.length],
      ),
      borderWidth: 0,
    }],
  }
})

// ---- 提示 ----
const share = (values: number[], index: number): string => {
  const total = values.reduce((a, b) => a + b, 0)
  return total > 0 ? ((values[index] / total) * 100).toFixed(1) : '0.0'
}

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      enabled: false,
      external: externalTooltipHandler,
      callbacks: {
        label: (context: { label: string; dataIndex: number }) => {
          const value = chartValues.value[context.dataIndex] ?? 0
          return `${context.label}: ${formatTokens(value)} (${share(chartValues.value, context.dataIndex)}%)`
        },
      },
    },
  },
}))

const rankingDoughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      enabled: false,
      external: externalTooltipHandler,
      callbacks: {
        label: (context: { label: string; dataIndex: number }) => {
          const value = rankingValues.value[context.dataIndex] ?? 0
          return `${context.label}: ${formatCost(value)} (${share(rankingValues.value, context.dataIndex)}%)`
        },
      },
    },
  },
}))
</script>
