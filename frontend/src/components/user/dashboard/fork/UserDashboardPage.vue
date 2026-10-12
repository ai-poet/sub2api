<template>
  <!-- 首次进入时各区块按 --rise-i 依次上移淡入，最后一块播放完后移除入场类 -->
  <div
    class="space-y-4"
    :class="{ 'dash-rise': entering }"
    :style="dashboardMotionVars"
    data-testid="dashboard-root"
    @animationend="onRiseEnd"
  >
    <!--
      标题行：左侧标题、说明和实时状态条，右侧筛选 / 刷新 / 时间范围。
      宽屏时顶栏已经显示页面标题和说明，这里只在窄屏显示，避免重复。
    -->
    <div class="dash-rise-item relative z-10 flex flex-wrap items-center justify-between gap-x-6 gap-y-3" :style="{ '--rise-i': 0 }">
      <div class="min-w-0 space-y-1">
        <h2 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white lg:hidden">{{ t('dashboard.title') }}</h2>
        <p class="text-sm text-gray-500 dark:text-dark-400 lg:hidden">{{ t('dashboard.welcomeMessage') }}</p>
        <UserDashboardLiveStats ref="liveStatsRef" @update="onStats" />
      </div>
      <UserDashboardUsageToolbar :refreshing="refreshing" @refresh="refreshAll" />
    </div>

    <UserDashboardUsageChart ref="usageChartRef" />

    <!-- 累计用量（本项目保留）：累计请求、累计 Token、标准价与实付价对比 -->
    <UserDashboardLifetime class="dash-rise-item" :style="{ '--rise-i': 3 }" :stats="stats" />

    <div class="dash-rise-item" :style="{ '--rise-i': 4 }">
      <UserDashboardHeatmap ref="heatmapRef" @select-day="revealChart" />
    </div>

    <!-- 按平台拆分与平台配额（本项目特有）；没有用量也没有限额时整块不渲染 -->
    <UserDashboardPlatforms
      class="dash-rise-item"
      :style="{ '--rise-i': 5 }"
      :stats="stats"
      :platform-quotas="platformQuotas"
      :is-simple="authStore.isSimpleMode"
    />

    <div class="dash-rise-item grid grid-cols-1 gap-4 lg:grid-cols-3" :style="{ '--rise-i': 6 }" data-rise-last="true">
      <div class="min-w-0 lg:col-span-2"><UserDashboardAnnouncements /></div>
      <div class="min-w-0 lg:col-span-1"><UserDashboardQuickActions :api-keys="apiKeys" /></div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：用户仪表盘整页（布局与交互移植自 TokenRouter views/user/DashboardView.vue）。
// 上游的 views/user/DashboardView.vue 只负责套 AppLayout 并挂载本组件；
// 本项目的 AppLayout 没有标题行插槽，所以标题行（实时状态条 + 工具栏）放在正文第一行。
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getMyPlatformQuotas } from '@/api/user'
import type { UserDashboardStats } from '@/api/usage'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAuthStore } from '@/stores/auth'
import type { PlatformQuotaItem } from '@/types'
import UserDashboardAnnouncements from './UserDashboardAnnouncements.vue'
import UserDashboardHeatmap from './UserDashboardHeatmap.vue'
import UserDashboardLifetime from './UserDashboardLifetime.vue'
import UserDashboardLiveStats from './UserDashboardLiveStats.vue'
import UserDashboardPlatforms from './UserDashboardPlatforms.vue'
import UserDashboardQuickActions from './UserDashboardQuickActions.vue'
import UserDashboardUsageChart from './UserDashboardUsageChart.vue'
import UserDashboardUsageToolbar from './UserDashboardUsageToolbar.vue'
import { dashboardMotionVars } from './dashboardMotion'
import { provideUsageChartState } from './usageChartState'

const { t } = useI18n()
const authStore = useAuthStore()
const announcementStore = useAnnouncementStore()

// 用量状态由页面提供，标题行的工具栏和正文的图表共用同一份。
const usageState = provideUsageChartState()
const heatmapRef = ref<InstanceType<typeof UserDashboardHeatmap> | null>(null)
const liveStatsRef = ref<InstanceType<typeof UserDashboardLiveStats> | null>(null)
const usageChartRef = ref<InstanceType<typeof UserDashboardUsageChart> | null>(null)
const refreshing = ref(false)
// 入场动画只在首次挂载时播放
const entering = ref(true)

// 实时统计由状态条轮询，平台拆分和密钥数量复用同一份结果。
const stats = ref<UserDashboardStats | null>(null)
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)

const apiKeys = computed(() => (
  stats.value ? { total: stats.value.total_api_keys, active: stats.value.active_api_keys } : null
))

// 取数失败时保留上一次的统计，平台卡片不会在一次失败的轮询后消失。
const onStats = (value: UserDashboardStats | null) => {
  if (value) stats.value = value
}

const loadPlatformQuotas = async () => {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (error) {
    console.warn('Failed to load platform quotas:', error)
    platformQuotas.value = []
  }
}

// refreshUser 刷新账户信息，顶栏余额随之更新。
const refreshUser = async () => {
  try {
    await authStore.refreshUser()
  } catch (error) {
    console.error('Failed to refresh user:', error)
  }
}

// 刷新按钮覆盖整页；手动刷新时绕过公告节流获取最新内容。
const refreshAll = async () => {
  refreshing.value = true
  try {
    // 各区块自行处理错误，这里只等全部结束再恢复刷新按钮。
    await Promise.allSettled([
      refreshUser(),
      usageState.load(),
      heatmapRef.value?.reload(),
      liveStatsRef.value?.reload(),
      loadPlatformQuotas(),
      announcementStore.fetchAnnouncements(true),
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
  void refreshUser()
  void usageState.load()
  void usageState.loadFilterOptions()
  void loadPlatformQuotas()
})
</script>

<style scoped>
/* 区块入场：自下方 8px 上移并淡入，相邻区块错开一个步长。
   只在延迟期间保持起始帧，结束后不保留 transform，避免影响内部浮层的定位。 */
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
