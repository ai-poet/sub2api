<template>
  <div class="card p-4" data-testid="admin-model-distribution">
    <div class="mb-4 flex items-center justify-between gap-3">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ activeView === 'model_distribution'
          ? t('admin.dashboard.modelDistribution')
          : t('admin.dashboard.spendingRankingTitle') }}
      </h3>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <div v-segmented class="segmented">
          <button
            type="button"
            class="segmented-item px-2.5 py-1 text-xs"
            :class="{ 'segmented-item-active': activeView === 'model_distribution' }"
            @click="activeView = 'model_distribution'"
          >
            {{ t('admin.dashboard.viewModelDistribution') }}
          </button>
          <button
            type="button"
            class="segmented-item px-2.5 py-1 text-xs"
            :class="{ 'segmented-item-active': activeView === 'spending_ranking' }"
            @click="activeView = 'spending_ranking'"
          >
            {{ t('admin.dashboard.viewSpendingRanking') }}
          </button>
        </div>
      </div>
    </div>

    <DashboardChartSkeleton v-if="activeView === 'model_distribution' && loading" variant="distribution" />
    <div
      v-else-if="activeView === 'model_distribution' && displayModelStats.length > 0 && chartData"
      class="flex flex-col items-center gap-4 sm:flex-row sm:items-start sm:gap-6"
    >
      <div class="h-48 w-48 shrink-0">
        <Doughnut :data="chartData" :options="doughnutOptions" />
      </div>
      <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto">
        <table class="w-full text-xs">
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
            <template v-for="model in displayModelStats" :key="model.model">
              <tr
                class="group/breakdown cursor-pointer border-t border-gray-100 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:hover:bg-dark-700/40"
                :aria-expanded="expandedKey === `model-${model.model}`"
                data-testid="model-row"
                @click="toggleBreakdown('model', model.model)"
              >
                <!-- 模型名称随可展开行的悬停着色，箭头继承文字颜色。 -->
                <td
                  class="max-w-[100px] truncate py-1.5 font-medium text-gray-900 group-hover/breakdown:text-primary-600 dark:text-white dark:group-hover/breakdown:text-primary-500"
                  :title="model.model"
                >
                  <span class="inline-flex items-center gap-1">
                    <Icon
                      name="chevronRight"
                      size="xs"
                      class="h-3 w-3 shrink-0 transition-transform duration-normal"
                      :class="{ 'rotate-90': expandedKey === `model-${model.model}` }"
                    />
                    {{ model.model }}
                  </span>
                </td>
                <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                  {{ formatNumber(model.requests) }}
                </td>
                <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                  {{ formatTokens(model.total_tokens) }}
                </td>
                <td class="py-1.5 text-right text-green-600 dark:text-green-400">
                  ${{ formatCost(model.actual_cost) }}
                </td>
                <td class="py-1.5 text-right text-orange-500 dark:text-orange-400">
                  ${{ formatCost(model.account_cost) }}
                </td>
                <td class="py-1.5 text-right text-gray-400 dark:text-gray-500">
                  ${{ formatCost(model.cost) }}
                </td>
              </tr>
              <ExpandableTableRow :open="expandedKey === `model-${model.model}`" :colspan="6">
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
    <!-- 两个视图保持相同的桌面端顶部对齐规则。 -->
    <div v-else-if="rankingDisplayItems.length > 0 && rankingChartData" class="flex flex-col items-center gap-4 sm:flex-row sm:items-start sm:gap-6">
      <div class="h-48 w-48 shrink-0">
        <Doughnut :data="rankingChartData" :options="rankingDoughnutOptions" />
      </div>
      <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto">
        <table class="w-full text-xs">
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
              class="border-t border-gray-100 transition-colors dark:border-dark-600"
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
                    :title="getRankingRowLabel(item)"
                  >
                    {{ getRankingRowLabel(item) }}
                  </span>
                </div>
              </td>
              <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                {{ formatNumber(item.requests) }}
              </td>
              <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                {{ formatTokens(item.tokens) }}
              </td>
              <td class="py-1.5 text-right text-green-600 dark:text-green-400">
                ${{ formatCost(item.actual_cost) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div
      v-else
      class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
    >
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的「模型分布 / 用户消费榜」卡片，逐项移植自 TokenRouter components/charts/ModelDistributionChart.vue
// （仪表盘用到的部分：分段切换、加载骨架、可展开的按用户拆分、圆环外置提示）。
// 与 TokenRouter 的差别只有数据口径：「成本」列是本项目的账号成本（TokenRouter 叫服务商成本），金额以美元显示。
// 上游的 components/charts/ModelDistributionChart.vue 仍给用量页使用，不改它。
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
import { formatNumber, formatTokens, toFiniteNumber } from './format'

ChartJS.register(ArcElement, Tooltip, Legend)

onBeforeUnmount(hideExternalTooltip)

const { t } = useI18n()

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

const expandedKey = ref<string | null>(null)
const breakdownItems = ref<UserBreakdownItem[]>([])
const breakdownLoading = ref(false)

const toggleBreakdown = async (type: string, id: string) => {
  const key = `${type}-${id}`
  if (expandedKey.value === key) {
    expandedKey.value = null
    return
  }
  expandedKey.value = key
  breakdownLoading.value = true
  breakdownItems.value = []
  try {
    const res = await getUserBreakdown({
      start_date: props.startDate,
      end_date: props.endDate,
      model: id,
      model_source: 'requested',
    })
    // 拆分结果回来时用户可能已经展开了另一个模型
    if (expandedKey.value !== key) return
    breakdownItems.value = res.users || []
  } catch {
    if (expandedKey.value !== key) return
    breakdownItems.value = []
  } finally {
    if (expandedKey.value === key) breakdownLoading.value = false
  }
}

const emit = defineEmits<{
  'ranking-click': [item: UserSpendingRankingItem]
}>()

const activeView = ref<'model_distribution' | 'spending_ranking'>('model_distribution')

const chartColors = CHART_PALETTE

const displayModelStats = computed(() => {
  if (!props.modelStats?.length) return []
  return [...props.modelStats].sort((a, b) => toFiniteNumber(b.total_tokens) - toFiniteNumber(a.total_tokens))
})

// 圆环扇区与 tooltip 共用原始指标值，保证扇区大小与占比一致。
const chartValues = computed(() => displayModelStats.value.map((m) => toFiniteNumber(m.total_tokens)))

const chartData = computed(() => {
  if (!displayModelStats.value.length) return null

  return {
    labels: displayModelStats.value.map((m) => m.model),
    datasets: [
      {
        data: chartValues.value,
        backgroundColor: chartColors.slice(0, displayModelStats.value.length),
        borderWidth: 0
      }
    ]
  }
})

const rankingValues = computed(() => {
  if (!props.rankingItems?.length) return []

  const values = props.rankingItems.map((item) => toFiniteNumber(item.actual_cost))
  if (otherRankingItem.value) values.push(toFiniteNumber(otherRankingItem.value.actual_cost))
  return values
})

const rankingChartData = computed(() => {
  if (!props.rankingItems?.length) return null

  const labels = props.rankingItems.map((item, index) => `#${index + 1} ${getRankingUserLabel(item)}`)
  const backgroundColor: string[] = [...chartColors.slice(0, props.rankingItems.length)]

  if (otherRankingItem.value) {
    labels.push(t('admin.dashboard.spendingRankingOther'))
    backgroundColor.push(CHART_OTHER_COLOR)
  }

  return {
    labels,
    datasets: [
      {
        data: rankingValues.value,
        backgroundColor,
        borderWidth: 0
      }
    ]
  }
})

const otherRankingItem = computed<RankingDisplayItem | null>(() => {
  if (!props.rankingItems?.length) return null

  const rankedActualCost = props.rankingItems.reduce((sum, item) => sum + toFiniteNumber(item.actual_cost), 0)
  const rankedRequests = props.rankingItems.reduce((sum, item) => sum + toFiniteNumber(item.requests), 0)
  const rankedTokens = props.rankingItems.reduce((sum, item) => sum + toFiniteNumber(item.tokens), 0)

  const otherActualCost = Math.max((props.rankingTotalActualCost || 0) - rankedActualCost, 0)
  const otherRequests = Math.max((props.rankingTotalRequests || 0) - rankedRequests, 0)
  const otherTokens = Math.max((props.rankingTotalTokens || 0) - rankedTokens, 0)

  if (otherActualCost <= 0.000001 && otherRequests <= 0 && otherTokens <= 0) return null

  return {
    user_id: 0,
    email: '',
    username: '',
    actual_cost: otherActualCost,
    requests: otherRequests,
    tokens: otherTokens,
    isOther: true
  }
})

const rankingDisplayItems = computed<RankingDisplayItem[]>(() => {
  if (!props.rankingItems?.length) return []
  return otherRankingItem.value
    ? [...props.rankingItems, otherRankingItem.value]
    : [...props.rankingItems]
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    },
    tooltip: {
      enabled: false,
      external: externalTooltipHandler,
      callbacks: {
        label: (context: { label: string; dataIndex: number }) => {
          const value = chartValues.value[context.dataIndex] ?? 0
          const total = chartValues.value.reduce((a: number, b: number) => a + b, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return `${context.label}: ${formatTokens(value)} (${percentage}%)`
        }
      }
    }
  }
}))

const rankingDoughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    },
    tooltip: {
      enabled: false,
      external: externalTooltipHandler,
      callbacks: {
        label: (context: { label: string; dataIndex: number }) => {
          const value = rankingValues.value[context.dataIndex] ?? 0
          const total = rankingValues.value.reduce((a: number, b: number) => a + b, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return `${context.label}: $${formatCost(value)} (${percentage}%)`
        }
      }
    }
  }
}))

const getRankingUserLabel = (item: UserSpendingRankingItem): string => {
  // 排行标签优先使用用户名，并忽略仅包含空白字符的身份字段。
  if (item.username?.trim()) return item.username.trim()
  if (item.email?.trim()) return item.email.trim()
  return t('admin.redeem.userPrefix', { id: item.user_id })
}

const getRankingRowLabel = (item: RankingDisplayItem): string => {
  if (item.isOther) return t('admin.dashboard.spendingRankingOther')
  return getRankingUserLabel(item)
}

// 金额的小数位随大小变化（与 TokenRouter 的 formatCost 相同）：千以上缩写 K。
const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}
</script>
