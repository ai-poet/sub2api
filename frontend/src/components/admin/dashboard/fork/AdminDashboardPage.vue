<template>
  <div class="space-y-4">
    <!--
      TokenRouter 把刷新按钮放在 AppLayout 的 page-heading-actions 插槽（标题右侧）。
      本项目的 AppLayout 没有标题行插槽，宽屏时顶栏已显示标题和说明，
      这里只在窄屏显示文字，刷新按钮始终靠右。
    -->
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div class="lg:hidden">
        <h2 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('admin.dashboard.title') }}</h2>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.dashboard.description') }}</p>
      </div>
      <button
        type="button"
        class="btn btn-secondary ml-auto shrink-0 btn-icon"
        :disabled="chartsLoading"
        :title="t('common.refresh')"
        data-testid="admin-dashboard-refresh"
        @click="loadDashboardStats"
      >
        <Icon name="refresh" size="sm" :class="chartsLoading ? 'animate-spin' : ''" />
      </button>
    </div>

    <!-- Loading State -->
    <AdminDashboardSkeleton v-if="loading && !stats" />

    <template v-else-if="stats">
      <!-- Row 1: Core Stats -->
      <!-- 移动端卡片纵向排列图标和文字，文字占满卡宽，桌面端横向排列。 -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <!-- Total API Keys -->
        <div class="card p-4" data-testid="stat-api-keys">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
              <Icon name="key" size="md" class="text-blue-600 dark:text-blue-400" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.apiKeys') }}
              </p>
              <p class="mt-0.5 whitespace-nowrap text-lg font-bold tabular-nums text-gray-900 dark:text-white lg:text-xl">
                {{ stats.total_api_keys }}
              </p>
              <p class="mt-0.5 text-xs text-green-600 dark:text-green-400">
                {{ stats.active_api_keys }} {{ t('common.active') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Service Accounts -->
        <div class="card p-4" data-testid="stat-accounts">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-purple-100 p-2 dark:bg-purple-900/30">
              <Icon name="server" size="md" class="text-purple-600 dark:text-purple-400" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.accounts') }}
              </p>
              <p class="mt-0.5 whitespace-nowrap text-lg font-bold tabular-nums text-gray-900 dark:text-white lg:text-xl">
                {{ stats.total_accounts }}
              </p>
              <p class="mt-0.5 text-xs">
                <span class="whitespace-nowrap text-green-600 dark:text-green-400"
                  >{{ stats.normal_accounts }} {{ t('common.active') }}</span
                >
                <span v-if="stats.error_accounts > 0" class="ml-1 whitespace-nowrap text-red-500"
                  >{{ stats.error_accounts }} {{ t('common.error') }}</span
                >
              </p>
            </div>
          </div>
        </div>

        <!-- Today Requests -->
        <div class="card p-4" data-testid="stat-today-requests">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
              <Icon name="chart" size="md" class="text-green-600 dark:text-green-400" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.todayRequests') }}
              </p>
              <p class="mt-0.5 whitespace-nowrap text-lg font-bold tabular-nums text-gray-900 dark:text-white lg:text-xl">
                {{ stats.today_requests }}
              </p>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('common.total') }}: {{ formatNumber(stats.total_requests) }}
              </p>
            </div>
          </div>
        </div>

        <!-- New Users Today -->
        <div class="card p-4" data-testid="stat-users">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-emerald-100 p-2 dark:bg-emerald-900/30">
              <Icon name="userPlus" size="md" class="text-emerald-600 dark:text-emerald-400" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.users') }}
              </p>
              <p class="mt-0.5 whitespace-nowrap text-lg font-bold tabular-nums text-emerald-600 dark:text-emerald-400 lg:text-xl">
                +{{ stats.today_new_users }}
              </p>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('common.total') }}: {{ formatNumber(stats.total_users) }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Row 2: Token Stats -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <!-- Today Tokens -->
        <div class="card p-4" data-testid="stat-today-tokens">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
              <Icon name="cube" size="md" class="text-amber-600 dark:text-amber-400" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.todayTokens') }}
              </p>
              <p class="mt-0.5 whitespace-nowrap text-lg font-bold tabular-nums text-gray-900 dark:text-white lg:text-xl">
                {{ formatTokens(stats.today_tokens) }}
              </p>
              <!-- 金额与符号在各段内保持同一行，段间可以换行。 -->
              <div class="mt-0.5 flex flex-wrap items-center gap-x-1 text-xs" data-testid="today-costs">
                <span
                  class="whitespace-nowrap text-green-600 dark:text-green-400"
                  :title="t('admin.dashboard.fork.actualDescription')"
                  >{{ formatCost(stats.today_actual_cost) }}</span
                >
                <span class="text-gray-400 dark:text-gray-500"> / </span>
                <span
                  class="whitespace-nowrap text-orange-500 dark:text-orange-400"
                  :title="t('admin.dashboard.fork.accountCostDescription')"
                  >{{ formatCost(stats.today_account_cost) }}</span
                >
                <span class="text-gray-400 dark:text-gray-500"> / </span>
                <span
                  class="whitespace-nowrap text-gray-400 dark:text-gray-500"
                  :title="t('admin.dashboard.fork.standardDescription')"
                  >{{ formatCost(stats.today_cost) }}</span
                >
              </div>
            </div>
          </div>
        </div>

        <!-- Total Tokens -->
        <div class="card p-4" data-testid="stat-total-tokens">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30">
              <Icon name="database" size="md" class="text-indigo-600 dark:text-indigo-400" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.totalTokens') }}
              </p>
              <p class="mt-0.5 whitespace-nowrap text-lg font-bold tabular-nums text-gray-900 dark:text-white lg:text-xl">
                {{ formatTokens(stats.total_tokens) }}
              </p>
              <!-- 金额与符号在各段内保持同一行，段间可以换行。 -->
              <div class="mt-0.5 flex flex-wrap items-center gap-x-1 text-xs" data-testid="total-costs">
                <span
                  class="whitespace-nowrap text-green-600 dark:text-green-400"
                  :title="t('admin.dashboard.fork.actualDescription')"
                  >{{ formatCost(stats.total_actual_cost) }}</span
                >
                <span class="text-gray-400 dark:text-gray-500"> / </span>
                <span
                  class="whitespace-nowrap text-orange-500 dark:text-orange-400"
                  :title="t('admin.dashboard.fork.accountCostDescription')"
                  >{{ formatCost(stats.total_account_cost) }}</span
                >
                <span class="text-gray-400 dark:text-gray-500"> / </span>
                <span
                  class="whitespace-nowrap text-gray-400 dark:text-gray-500"
                  :title="t('admin.dashboard.fork.standardDescription')"
                  >{{ formatCost(stats.total_cost) }}</span
                >
              </div>
            </div>
          </div>
        </div>

        <!-- Performance (RPM/TPM) -->
        <div class="card p-4" data-testid="stat-performance">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-violet-100 p-2 dark:bg-violet-900/30">
              <Icon name="bolt" size="md" class="text-violet-600 dark:text-violet-400" :stroke-width="2" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.performance') }}
              </p>
              <div class="mt-0.5 flex items-baseline gap-2">
                <p class="whitespace-nowrap text-lg font-bold tabular-nums text-gray-900 dark:text-white lg:text-xl">
                  {{ formatTokens(stats.rpm) }}
                </p>
                <span class="text-xs text-gray-500 dark:text-gray-400">RPM</span>
              </div>
              <div class="mt-0.5 flex items-baseline gap-2">
                <p class="whitespace-nowrap text-sm font-semibold tabular-nums text-violet-600 dark:text-violet-400">
                  {{ formatTokens(stats.tpm) }}
                </p>
                <span class="text-xs text-gray-500 dark:text-gray-400">TPM</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Avg Response Time -->
        <div class="card p-4" data-testid="stat-avg-response">
          <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
            <div class="shrink-0 self-start rounded-lg bg-rose-100 p-2 dark:bg-rose-900/30">
              <Icon name="clock" size="md" class="text-rose-600 dark:text-rose-400" :stroke-width="2" />
            </div>
            <div class="min-w-0">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.dashboard.avgResponse') }}
              </p>
              <p class="mt-0.5 whitespace-nowrap text-lg font-bold tabular-nums text-gray-900 dark:text-white lg:text-xl">
                {{ formatDuration(stats.average_duration_ms) }}
              </p>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                {{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Charts Section -->
      <div class="space-y-4">
        <!-- Date Range Filter -->
        <div class="card p-4">
          <div class="time-controls flex flex-wrap items-center justify-between gap-2">
            <div class="flex min-w-0 items-center gap-2">
              <span class="time-control-label text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.dashboard.timeRange') }}:</span>
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="onDateRangeChange"
              />
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <span class="time-control-label text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.dashboard.granularity') }}:</span>
              <div class="w-28">
                <Select
                  v-model="granularity"
                  :options="granularityOptions"
                  @change="loadChartData"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Charts Grid -->
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <AdminModelDistribution
            :model-stats="modelStats"
            :ranking-items="rankingItems"
            :ranking-total-actual-cost="rankingTotalActualCost"
            :ranking-total-requests="rankingTotalRequests"
            :ranking-total-tokens="rankingTotalTokens"
            :loading="chartsLoading"
            :ranking-loading="rankingLoading"
            :ranking-error="rankingError"
            :start-date="startDate"
            :end-date="endDate"
            @ranking-click="goToUserUsage"
          />
          <AdminTokenTrend :trend-data="trendData" :loading="chartsLoading" :granularity="granularity" />
        </div>

        <!-- User Usage Trend (Full Width) -->
        <AdminUserTrend
          :trend="userTrend"
          :metric="userTrendMetric"
          :loading="userTrendLoading"
          @update:metric="setUserTrendMetric"
        />
      </div>

      <!-- 本项目特有：快捷操作（批量生图、分组定价），TokenRouter 的管理员仪表盘没有这一块，放在最后 -->
      <AdminQuickActions />
    </template>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘整页，模板与取数逐项移植自 TokenRouter views/admin/DashboardView.vue。
// 上游的 views/admin/DashboardView.vue 只负责套 AppLayout 并挂载本组件；上游的
// ModelDistributionChart / TokenUsageTrend 仍给用量页使用，这里换成同目录的移植版。
// 与 TokenRouter 的差别：「账号」卡（TokenRouter 叫服务商）、金额以美元显示、
// 用户趋势可按 Token / 实际消费切换、页尾的快捷操作——都是本项目原有功能。
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import AdminDashboardSkeleton from './AdminDashboardSkeleton.vue'
import AdminModelDistribution from './AdminModelDistribution.vue'
import AdminQuickActions from './AdminQuickActions.vue'
import AdminTokenTrend from './AdminTokenTrend.vue'
import AdminUserTrend from './AdminUserTrend.vue'
import { formatCost, formatDuration, formatNumber, formatTokens } from './format'

const { t } = useI18n()

const appStore = useAppStore()
const router = useRouter()
const stats = ref<DashboardStats | null>(null)
const loading = ref(true)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const userTrendMetric = ref<'tokens' | 'actual_cost'>('tokens')
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity options for Select component
const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

// 本项目保留：用户趋势按 Token / 实际消费切换，切换后清空旧数据并重新取数
const setUserTrendMetric = (metric: 'tokens' | 'actual_cost') => {
  if (userTrendMetric.value === metric) return
  userTrendMetric.value = metric
  userTrend.value = []
  loadUsersTrend()
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12,
      metric: userTrendMetric.value
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  loadDashboardStats()
})
</script>

<style scoped>
/* 移植自 TokenRouter style.css：筛选卡按自身宽度隐藏「时间范围 / 粒度」文字标签，窄于 460px 只留控件。 */
.time-controls {
  container-type: inline-size;
}

.time-control-label {
  display: none;
}

@container (min-width: 460px) {
  .time-control-label {
    display: inline;
  }
}
</style>
