<template>
  <div>
    <!--
      标题行与用户仪表盘相同（TokenRouter 用户仪表盘的布局）：左侧标题、说明和全站实时状态，右侧筛选 / 刷新 / 时间范围。
      本项目的 AppLayout 没有标题行插槽，所以写在正文第一行；顶栏按路由的 pageHeadingInBody 不再重复标题。
    -->
    <div class="mb-4 flex flex-shrink-0 flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.dashboard.title') }}</h1>
        <p class="mt-1 text-sm text-primary-900/90 dark:text-dark-300">{{ t('admin.dashboard.description') }}</p>
        <!-- 标题说明下方的补充信息：全站实时状态 -->
        <div class="mt-2">
          <UserDashboardLiveStats ref="liveStatsRef" @update="onStats" />
        </div>
      </div>
      <div class="shrink-0">
        <UserDashboardUsageToolbar :refreshing="refreshing" @refresh="refreshAll" />
      </div>
    </div>

    <!-- 首次进入时各区块按 --rise-i 依次上移淡入，最后一块播放完后移除入场类 -->
    <div
      class="space-y-4"
      :class="{ 'dash-rise': entering }"
      :style="dashboardMotionVars"
      data-testid="admin-dashboard-root"
      @animationend="onRiseEnd"
    >
      <!-- 用量区：四个指标卡（全站合计、迷你走势、环比）+ 趋势图 + 模型排行，与用户仪表盘同一组件 -->
      <UserDashboardUsageChart ref="usageChartRef" />

      <!-- 全站概览（管理员特有）：账号、用户、API 密钥、累计用量与实付 / 成本 / 标准 -->
      <AdminOverviewCards class="dash-rise-item" :style="{ '--rise-i': 2 }" :stats="stats" />

      <div class="dash-rise-item" :style="{ '--rise-i': 3 }">
        <UserDashboardHeatmap ref="heatmapRef" @select-day="revealChart" />
      </div>

      <!-- 与用户仪表盘的「公告 | 快捷操作」同一行布局：模型分布 / 用户消费榜 | 快捷操作 -->
      <div class="dash-rise-item grid grid-cols-1 gap-4 lg:grid-cols-3" :style="{ '--rise-i': 4 }">
        <div class="min-w-0 lg:col-span-2">
          <AdminModelDistribution
            :model-stats="usageState.models.value"
            :loading="!usageState.modelsLoaded.value"
            :ranking-items="rankingItems"
            :ranking-total-actual-cost="rankingTotalActualCost"
            :ranking-total-requests="rankingTotalRequests"
            :ranking-total-tokens="rankingTotalTokens"
            :ranking-loading="rankingLoading"
            :ranking-error="rankingError"
            :start-date="usageState.pickerStart.value"
            :end-date="usageState.pickerEnd.value"
            @ranking-click="goToUserUsage"
          />
        </div>
        <div class="min-w-0 lg:col-span-1"><AdminQuickActions /></div>
      </div>

      <!-- 用户使用趋势（Top 12），跟随工具栏的时间范围和筛选条件 -->
      <AdminUserTrend
        class="dash-rise-item"
        :style="{ '--rise-i': 5 }"
        data-rise-last="true"
        :trend="userTrend"
        :metric="userTrendMetric"
        :loading="userTrendLoading"
        @update:metric="setUserTrendMetric"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘整页。设计与用户仪表盘相同（TokenRouter 用户仪表盘：实时状态、指标卡、趋势图、模型排行、热力图），
// 取数换成全站口径（provideDashboardDataSource(adminDashboardDataSource)），下面再接管理员特有的
// 全站概览、模型分布 / 用户消费榜、快捷操作和用户使用趋势，它们都跟随工具栏的时间范围。
// 上游的 views/admin/DashboardView.vue 只负责套 AppLayout 并挂载本组件。
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { getUserSpendingRanking, getUserUsageTrend } from '@/api/admin/dashboard'
import type { DashboardStats, UserSpendingRankingItem, UserUsageTrendPoint } from '@/types'
import UserDashboardHeatmap from '@/components/user/dashboard/fork/UserDashboardHeatmap.vue'
import UserDashboardLiveStats from '@/components/user/dashboard/fork/UserDashboardLiveStats.vue'
import UserDashboardUsageChart from '@/components/user/dashboard/fork/UserDashboardUsageChart.vue'
import UserDashboardUsageToolbar from '@/components/user/dashboard/fork/UserDashboardUsageToolbar.vue'
import {
  adminDashboardDataSource,
  provideDashboardDataSource,
  type DashboardLiveStats,
} from '@/components/user/dashboard/fork/dashboardDataSource'
import { dashboardMotionVars } from '@/components/user/dashboard/fork/dashboardMotion'
import { queryDates } from '@/components/user/dashboard/fork/usageChartData'
import { provideUsageChartState } from '@/components/user/dashboard/fork/usageChartState'
import AdminModelDistribution from './AdminModelDistribution.vue'
import AdminOverviewCards from './AdminOverviewCards.vue'
import AdminQuickActions from './AdminQuickActions.vue'
import AdminUserTrend from './AdminUserTrend.vue'

const { t } = useI18n()
const router = useRouter()

// 全站口径：子组件（工具栏、热力图、实时状态条、模型排行）通过 inject 拿到；
// 本组件 inject 不到自己 provide 的值，所以用量状态直接传入数据源。
const source = provideDashboardDataSource(adminDashboardDataSource)
const usageState = provideUsageChartState(source)

const heatmapRef = ref<InstanceType<typeof UserDashboardHeatmap> | null>(null)
const liveStatsRef = ref<InstanceType<typeof UserDashboardLiveStats> | null>(null)
const usageChartRef = ref<InstanceType<typeof UserDashboardUsageChart> | null>(null)
const refreshing = ref(false)
// 入场动画只在首次挂载时播放
const entering = ref(true)

// 实时状态条每分钟取一次全站统计，概览卡复用同一份；取数失败时保留上一次的结果。
const stats = ref<DashboardStats | null>(null)
const onStats = (value: DashboardLiveStats | null) => {
  // 本页指定了管理员口径，状态条给出的就是完整的 DashboardStats。
  if (value) stats.value = value as DashboardStats
}

// ---- 用户消费榜（只按日期）----
const RANKING_LIMIT = 12
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
const rankingLoading = ref(false)
const rankingError = ref(false)
let rankingSeq = 0

const loadSpendingRanking = async () => {
  const seq = ++rankingSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await getUserSpendingRanking({ ...queryDates(usageState.activeRange.value), limit: RANKING_LIMIT })
    if (seq !== rankingSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (seq !== rankingSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (seq === rankingSeq) rankingLoading.value = false
  }
}

// ---- 用户使用趋势（Top 12，按日期、粒度和筛选条件）----
const USER_TREND_LIMIT = 12
const userTrend = ref<UserUsageTrendPoint[]>([])
const userTrendMetric = ref<'tokens' | 'actual_cost'>('tokens')
const userTrendLoading = ref(false)
let userTrendSeq = 0

const loadUserTrend = async () => {
  const seq = ++userTrendSeq
  userTrendLoading.value = true
  const range = usageState.activeRange.value
  try {
    const response = await getUserUsageTrend({
      ...queryDates(range),
      granularity: range.granularity,
      ...usageState.filterState.queryParams.value,
      limit: USER_TREND_LIMIT,
      metric: userTrendMetric.value,
    })
    if (seq !== userTrendSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (seq !== userTrendSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (seq === userTrendSeq) userTrendLoading.value = false
  }
}

// 切换指标后清空旧数据并重新取数（本项目保留的按 Token / 实际消费切换）
const setUserTrendMetric = (metric: 'tokens' | 'actual_cost') => {
  if (userTrendMetric.value === metric) return
  userTrendMetric.value = metric
  userTrend.value = []
  void loadUserTrend()
}

// 工具栏换了时间范围或筛选条件后，消费榜和用户趋势跟着重新取数。
// 用量区每次加载都会生成新的范围对象，所以比较实际的查询条件，只在条件变化时重新取数。
const chartQueryKey = () => {
  const range = usageState.activeRange.value
  return JSON.stringify([queryDates(range), range.granularity, usageState.filterState.queryParams.value])
}
watch(chartQueryKey, () => {
  void loadSpendingRanking()
  void loadUserTrend()
})

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: { user_id: String(item.user_id), start_date: usageState.pickerStart.value, end_date: usageState.pickerEnd.value },
  })
}

// 刷新按钮覆盖整页：用量区、热力图、实时状态（含概览卡）、消费榜和用户趋势。
const refreshAll = async () => {
  refreshing.value = true
  try {
    // 各区块自行处理错误，这里只等全部结束再恢复刷新按钮。
    await Promise.allSettled([
      usageState.load(),
      heatmapRef.value?.reload(),
      liveStatsRef.value?.reload(),
      loadSpendingRanking(),
      loadUserTrend(),
    ])
  } finally {
    refreshing.value = false
  }
}

// revealChart 在热力图选中某天后，把趋势图滚动到可见区域。
const revealChart = () => {
  const card = usageChartRef.value?.chartCardRef
  if (!card || typeof card.scrollIntoView !== 'function') return
  const reduced = typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  card.scrollIntoView({ behavior: reduced ? 'auto' : 'smooth', block: 'center' })
}

// 最后一块入场结束后移除入场类，之后的重新渲染不会再次播放。
const onRiseEnd = (event: AnimationEvent) => {
  if ((event.target as HTMLElement).dataset?.riseLast) entering.value = false
}

onMounted(() => {
  void usageState.load()
  void usageState.loadFilterOptions()
  void loadSpendingRanking()
  void loadUserTrend()
})
</script>

<style scoped>
/* 区块入场：与用户仪表盘相同，自下方 8px 上移并淡入，相邻区块错开一个步长。 */
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
