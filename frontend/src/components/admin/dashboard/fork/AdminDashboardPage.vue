<template>
  <!-- 首次进入时各区块按 --rise-i 依次上移淡入，最后一块播放完后移除入场类 -->
  <div
    class="space-y-4"
    :class="{ 'dash-rise': entering }"
    :style="dashboardMotionVars"
    data-testid="admin-dashboard-root"
    @animationend="onRiseEnd"
  >
    <!--
      标题行：TokenRouter 把刷新按钮放在页面标题右侧。本项目的 AppLayout 没有标题行插槽，
      宽屏时顶栏已显示标题和说明，这里只在窄屏显示文字，刷新按钮始终靠右。
    -->
    <div class="dash-rise-item flex items-center justify-between gap-3" :style="{ '--rise-i': 0 }">
      <div class="min-w-0 lg:hidden">
        <h2 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('admin.dashboard.title') }}</h2>
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.dashboard.description') }}</p>
      </div>
      <button
        type="button"
        class="btn btn-secondary btn-icon ml-auto shrink-0"
        :disabled="chartsLoading"
        :title="t('common.refresh')"
        :aria-label="t('common.refresh')"
        data-testid="admin-dashboard-refresh"
        @click="loadDashboardStats"
      >
        <Icon name="refresh" size="sm" :class="chartsLoading ? 'animate-spin' : ''" />
      </button>
    </div>

    <AdminDashboardSkeleton v-if="loading && !stats" />

    <template v-else-if="stats">
      <!-- 第一行：密钥、账号、今日请求、今日新增用户 -->
      <div class="dash-rise-item grid grid-cols-2 gap-4 lg:grid-cols-4" :style="{ '--rise-i': 1 }">
        <AdminStatCard icon="key" tone="blue" :label="t('admin.dashboard.apiKeys')" testid="stat-api-keys">
          <p :class="VALUE_CLASS"><AnimatedNumber :value="stats.total_api_keys" :from="0" /></p>
          <template #sub>
            <span class="text-green-600 dark:text-green-400">{{ stats.active_api_keys }} {{ t('common.active') }}</span>
          </template>
        </AdminStatCard>

        <AdminStatCard icon="server" tone="purple" :label="t('admin.dashboard.accounts')" testid="stat-accounts">
          <p :class="VALUE_CLASS"><AnimatedNumber :value="stats.total_accounts" :from="0" /></p>
          <template #sub>
            <span class="whitespace-nowrap text-green-600 dark:text-green-400">{{ stats.normal_accounts }} {{ t('common.active') }}</span>
            <span v-if="stats.error_accounts > 0" class="ml-1 whitespace-nowrap text-red-500">{{ stats.error_accounts }} {{ t('common.error') }}</span>
          </template>
        </AdminStatCard>

        <AdminStatCard icon="chart" tone="green" :label="t('admin.dashboard.todayRequests')" testid="stat-today-requests">
          <p :class="VALUE_CLASS"><AnimatedNumber :value="stats.today_requests" :from="0" /></p>
          <template #sub>
            <span class="text-gray-500 dark:text-gray-400">{{ t('common.total') }}: {{ formatNumber(stats.total_requests) }}</span>
          </template>
        </AdminStatCard>

        <AdminStatCard icon="userPlus" tone="emerald" :label="t('admin.dashboard.users')" testid="stat-users">
          <p class="mt-0.5 whitespace-nowrap text-lg font-bold text-emerald-600 dark:text-emerald-400 lg:text-xl">
            +<AnimatedNumber :value="stats.today_new_users" :from="0" />
          </p>
          <template #sub>
            <span class="text-gray-500 dark:text-gray-400">{{ t('common.total') }}: {{ formatNumber(stats.total_users) }}</span>
          </template>
        </AdminStatCard>
      </div>

      <!-- 第二行：今日 / 累计 Token（实际 / 成本 / 标准）、性能、平均响应 -->
      <div class="dash-rise-item grid grid-cols-2 gap-4 lg:grid-cols-4" :style="{ '--rise-i': 2 }">
        <AdminStatCard icon="cube" tone="amber" :label="t('admin.dashboard.todayTokens')" testid="stat-today-tokens">
          <p :class="VALUE_CLASS">
            <AnimatedNumber :value="stats.today_tokens" :from="0" :format="compactFormatterFor(stats.today_tokens, 2)" />
          </p>
          <template #sub>
            <!-- 金额与分隔符在各段内不换行，段与段之间可以换行 -->
            <div class="flex flex-wrap items-center gap-x-1" data-testid="today-costs">
              <span class="whitespace-nowrap text-green-600 dark:text-green-400" :title="t('admin.dashboard.fork.actualDescription')">{{ formatCost(stats.today_actual_cost) }}</span>
              <span class="text-gray-400 dark:text-gray-500">/</span>
              <span class="whitespace-nowrap text-orange-500 dark:text-orange-400" :title="t('admin.dashboard.fork.accountCostDescription')">{{ formatCost(stats.today_account_cost) }}</span>
              <span class="text-gray-400 dark:text-gray-500">/</span>
              <span class="whitespace-nowrap text-gray-400 dark:text-gray-500" :title="t('admin.dashboard.fork.standardDescription')">{{ formatCost(stats.today_cost) }}</span>
            </div>
          </template>
        </AdminStatCard>

        <AdminStatCard icon="database" tone="indigo" :label="t('admin.dashboard.totalTokens')" testid="stat-total-tokens">
          <p :class="VALUE_CLASS">
            <AnimatedNumber :value="stats.total_tokens" :from="0" :format="compactFormatterFor(stats.total_tokens, 2)" />
          </p>
          <template #sub>
            <div class="flex flex-wrap items-center gap-x-1" data-testid="total-costs">
              <span class="whitespace-nowrap text-green-600 dark:text-green-400" :title="t('admin.dashboard.fork.actualDescription')">{{ formatCost(stats.total_actual_cost) }}</span>
              <span class="text-gray-400 dark:text-gray-500">/</span>
              <span class="whitespace-nowrap text-orange-500 dark:text-orange-400" :title="t('admin.dashboard.fork.accountCostDescription')">{{ formatCost(stats.total_account_cost) }}</span>
              <span class="text-gray-400 dark:text-gray-500">/</span>
              <span class="whitespace-nowrap text-gray-400 dark:text-gray-500" :title="t('admin.dashboard.fork.standardDescription')">{{ formatCost(stats.total_cost) }}</span>
            </div>
          </template>
        </AdminStatCard>

        <AdminStatCard icon="bolt" tone="violet" :label="t('admin.dashboard.performance')" testid="stat-performance">
          <div class="mt-0.5 flex items-baseline gap-2">
            <p class="whitespace-nowrap text-lg font-bold text-gray-900 dark:text-white lg:text-xl">
              <AnimatedNumber :value="stats.rpm" :from="0" :format="compactFormatterFor(stats.rpm, 2)" />
            </p>
            <span class="text-xs text-gray-500 dark:text-gray-400">RPM</span>
          </div>
          <div class="mt-0.5 flex items-baseline gap-2">
            <p class="whitespace-nowrap text-sm font-semibold text-violet-600 dark:text-violet-400">
              <AnimatedNumber :value="stats.tpm" :from="0" :format="compactFormatterFor(stats.tpm, 2)" />
            </p>
            <span class="text-xs text-gray-500 dark:text-gray-400">TPM</span>
          </div>
        </AdminStatCard>

        <AdminStatCard icon="clock" tone="rose" :label="t('admin.dashboard.avgResponse')" testid="stat-avg-response">
          <p :class="VALUE_CLASS">
            <AnimatedNumber :value="stats.average_duration_ms" :from="0" :format="formatDuration" />
          </p>
          <template #sub>
            <span class="text-gray-500 dark:text-gray-400">{{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}</span>
          </template>
        </AdminStatCard>
      </div>

      <!-- 快捷操作（本项目特有） -->
      <AdminQuickActions class="dash-rise-item" :style="{ '--rise-i': 3 }" />

      <!-- 时间范围与粒度 -->
      <div class="dash-rise-item card relative z-10 p-4" :style="{ '--rise-i': 4 }">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex min-w-0 items-center gap-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.dashboard.timeRange') }}:</span>
            <DateRangePicker v-model:start-date="startDate" v-model:end-date="endDate" @change="onDateRangeChange" />
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.dashboard.granularity') }}:</span>
            <div class="w-28">
              <Select v-model="granularity" :options="granularityOptions" @change="loadChartData" />
            </div>
          </div>
        </div>
      </div>

      <div class="dash-rise-item grid grid-cols-1 gap-4 lg:grid-cols-2" :style="{ '--rise-i': 5 }">
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

      <AdminUserTrend
        class="dash-rise-item"
        :style="{ '--rise-i': 6 }"
        data-rise-last="true"
        :trend="userTrend"
        :metric="userTrendMetric"
        :loading="userTrendLoading"
        :granularity="granularity"
        @update:metric="setUserTrendMetric"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘整页（布局移植自 TokenRouter views/admin/DashboardView.vue）。
// 上游的 views/admin/DashboardView.vue 只负责套 AppLayout 并挂载本组件；上游的
// ModelDistributionChart / TokenUsageTrend 仍给用量页使用，这里换成同目录的移植版。
// 本项目保留：快捷操作、账号成本、用户趋势按 Token / 实际消费切换；另加统计数字滚动和区块依次入场。
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { DashboardStats, ModelStat, TrendDataPoint, UserSpendingRankingItem, UserUsageTrendPoint } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import { dashboardMotionVars } from '@/components/user/dashboard/fork/dashboardMotion'
import { compactFormatterFor } from '@/utils/compactNumber'
import AdminDashboardSkeleton from './AdminDashboardSkeleton.vue'
import AdminModelDistribution from './AdminModelDistribution.vue'
import AdminQuickActions from './AdminQuickActions.vue'
import AdminStatCard from './AdminStatCard.vue'
import AdminTokenTrend from './AdminTokenTrend.vue'
import AdminUserTrend from './AdminUserTrend.vue'
import { formatCost, formatDuration, formatNumber } from './format'

const VALUE_CLASS = 'mt-0.5 whitespace-nowrap text-lg font-bold text-gray-900 dark:text-white lg:text-xl'

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()

// 入场动画只在首次进入时播放
const entering = ref(true)

const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const userTrendMetric = ref<'tokens' | 'actual_cost'>('tokens')
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
// 每类请求一个序号，只采用最后一次请求的结果
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const RANKING_LIMIT = 12
const USER_TREND_LIMIT = 12

const formatLocalDate = (date: Date): string =>
  `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`

// 默认看最近 24 小时，按小时
const now = new Date()
const granularity = ref<'day' | 'hour'>('hour')
const startDate = ref(formatLocalDate(new Date(now.getTime() - 24 * 60 * 60 * 1000)))
const endDate = ref(formatLocalDate(now))

const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') },
])

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: { user_id: String(item.user_id), start_date: startDate.value, end_date: endDate.value },
  })
}

// 选一天以内按小时，更长按天
const onDateRangeChange = (range: { startDate: string; endDate: string; preset: string | null }) => {
  const days = Math.ceil((new Date(range.endDate).getTime() - new Date(range.startDate).getTime()) / (24 * 60 * 60 * 1000))
  granularity.value = days <= 1 ? 'hour' : 'day'
  void loadChartData()
}

const loadDashboardSnapshot = async (includeStats: boolean) => {
  const seq = ++chartLoadSeq
  if (includeStats && !stats.value) loading.value = true
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
      include_users_trend: false,
    })
    if (seq !== chartLoadSeq) return
    if (includeStats && response.stats) stats.value = response.stats
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (seq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (seq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const seq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: USER_TREND_LIMIT,
      metric: userTrendMetric.value,
    })
    if (seq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (seq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (seq === usersTrendLoadSeq) userTrendLoading.value = false
  }
}

const setUserTrendMetric = (metric: 'tokens' | 'actual_cost') => {
  if (userTrendMetric.value === metric) return
  userTrendMetric.value = metric
  userTrend.value = []
  void loadUsersTrend()
}

const loadUserSpendingRanking = async () => {
  const seq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: RANKING_LIMIT,
    })
    if (seq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (seq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (seq === rankingLoadSeq) rankingLoading.value = false
  }
}

// 刷新按钮：统计卡与图表一起重新取数
const loadDashboardStats = async () => {
  await Promise.all([loadDashboardSnapshot(true), loadUsersTrend(), loadUserSpendingRanking()])
}

// 改时间范围或粒度：只重新取图表
const loadChartData = async () => {
  await Promise.all([loadDashboardSnapshot(false), loadUsersTrend(), loadUserSpendingRanking()])
}

// 最后一块入场结束后移除入场类，之后的重新渲染不会再次播放
const onRiseEnd = (event: AnimationEvent) => {
  if ((event.target as HTMLElement).dataset?.riseLast) entering.value = false
}

onMounted(() => {
  void loadDashboardStats()
})
</script>

<style scoped>
/* 区块入场：自下方 8px 上移并淡入，相邻区块错开一个步长（与用户仪表盘一致）。
   只在延迟期间保持起始帧，结束后不保留 transform，避免影响日期选择器等浮层的定位。 */
.dash-rise :deep(.dash-rise-item) {
  animation: dash-rise var(--dash-rise-ms) var(--motion-ease) backwards;
  animation-delay: calc(var(--rise-i, 0) * var(--dash-rise-step-ms));
}

@keyframes dash-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dash-rise :deep(.dash-rise-item) {
    animation: none;
  }
}
</style>
