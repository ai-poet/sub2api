<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- 标题与筛选工具共用一行（对齐 TokenRouter 用量排行） -->
      <section class="flex flex-wrap items-end justify-between gap-3">
        <div class="min-w-0">
          <h1 class="page-title">{{ t('leaderboard.title') }}</h1>
          <p class="page-description">{{ activeBoard === 'models' ? t('leaderboard.models.pageDescription') : t('leaderboard.description') }}</p>
        </div>

        <!-- 手机上工具栏单独一行、日期选择器在最左，下拉框向右展开；sm 以上靠右，下拉框向左展开 -->
        <div class="leaderboard-toolbar flex w-full flex-wrap items-center gap-2 sm:ml-auto sm:w-auto sm:shrink-0">
          <div v-if="boards.length > 1" v-segmented class="segmented" role="tablist" :aria-label="t('leaderboard.boardsLabel')">
            <button
              v-for="board in boards"
              :key="board"
              type="button"
              role="tab"
              class="segmented-item px-3 py-1.5 text-sm"
              :class="{ 'segmented-item-active': activeBoard === board }"
              :aria-selected="activeBoard === board"
              :data-board="board"
              @click="selectBoard(board)"
            >
              {{ t(`leaderboard.boards.${board}`) }}
            </button>
          </div>
          <DateRangePicker
            v-if="boards.length > 0"
            class="order-first sm:order-none"
            v-model:start-date="startDate"
            v-model:end-date="endDate"
            @change="onDateRangeChange"
          />
          <button
            v-if="boards.length > 0"
            type="button"
            class="btn btn-secondary btn-icon"
            :disabled="loading"
            :title="t('common.refresh')"
            :aria-label="t('common.refresh')"
            data-testid="leaderboard-refresh"
            @click="load"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </section>

      <section v-if="boards.length === 0" class="empty-box" data-testid="leaderboard-disabled">
        <div>
          <div class="empty-icon"><Icon name="trophy" size="lg" /></div>
          <h2 class="empty-title">{{ t('leaderboard.disabled') }}</h2>
        </div>
      </section>

      <div v-else-if="loading" class="space-y-4" role="status" :aria-label="t('common.loading')" aria-busy="true" data-testid="leaderboard-skeleton">
        <div v-if="activeBoard === 'users'" class="grid grid-cols-1 gap-4 md:grid-cols-3" aria-hidden="true">
          <div v-for="card in 3" :key="card" class="card min-w-0 space-y-3 p-4">
            <Skeleton width="55%" :height="12" />
            <Skeleton width="75%" :height="28" />
            <Skeleton width="65%" :height="12" />
          </div>
        </div>
        <div class="card space-y-4 p-6" aria-hidden="true">
          <div v-for="row in 5" :key="row" class="flex items-center gap-3">
            <Skeleton :width="48" :height="32" />
            <Skeleton variant="circle" :width="40" :height="40" />
            <Skeleton width="40%" :height="14" />
          </div>
        </div>
      </div>

      <div v-else-if="error" class="rounded-lg border border-red-200 bg-red-50 p-5 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-200" data-testid="leaderboard-error">
        {{ error }}
      </div>

      <template v-else-if="activeBoard === 'users'">
        <section v-if="users && !users.enabled" class="empty-box" data-testid="leaderboard-disabled">
          <div>
            <div class="empty-icon"><Icon name="trophy" size="lg" /></div>
            <h2 class="empty-title">{{ t('leaderboard.disabled') }}</h2>
          </div>
        </section>
        <section v-else-if="users && users.insufficient_data" class="empty-box" data-testid="leaderboard-insufficient">
          <div>
            <div class="empty-icon"><Icon name="users" size="lg" /></div>
            <h2 class="empty-title">{{ t('leaderboard.users.insufficientTitle') }}</h2>
            <p class="empty-description">{{ t('leaderboard.users.insufficient') }}</p>
          </div>
        </section>
        <template v-else-if="users && users.items.length > 0">
          <UserPodium :items="users.items" :primary-metric="primaryMetric" :metrics="userMetrics" :range-key="rangeKey" />
          <UserRankingList
            :items="users.items"
            :primary-metric="primaryMetric"
            :metrics="userMetrics"
            :limit="users.limit"
            :range-label="rangeLabel"
            :range-key="rangeKey"
          />
        </template>
        <section v-else-if="users" class="empty-box" data-testid="leaderboard-empty">
          <div>
            <div class="empty-icon"><Icon name="chart" size="lg" /></div>
            <h2 class="empty-title">{{ t('leaderboard.emptyTitle') }}</h2>
            <p class="empty-description">{{ t('leaderboard.emptyDescription') }}</p>
          </div>
        </section>
      </template>

      <template v-else>
        <section v-if="models && !models.enabled" class="empty-box" data-testid="leaderboard-disabled">
          <div>
            <div class="empty-icon"><Icon name="trophy" size="lg" /></div>
            <h2 class="empty-title">{{ t('leaderboard.disabled') }}</h2>
          </div>
        </section>
        <ModelRankingList
          v-else-if="models && models.items.length > 0"
          :items="models.items"
          :total-tokens="models.total_tokens"
          :other-share-percent="models.other_share_percent"
          :range-label="rangeLabel"
          :range-key="rangeKey"
        />
        <section v-else-if="models" class="empty-box" data-testid="leaderboard-empty">
          <div>
            <div class="empty-icon"><Icon name="chart" size="lg" /></div>
            <h2 class="empty-title">{{ t('leaderboard.emptyTitle') }}</h2>
            <p class="empty-description">{{ t('leaderboard.models.emptyDescription') }}</p>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
// fork：用量排行（对齐 TokenRouter UsageRankingView）。主榜是用户用量榜：日期范围默认今天，
// 后台设定的排名指标作为主指标，其余可见指标一起显示；模型热度榜在开启时作为第二个视图。
// 两个榜的开关、排名指标、条数与是否显示花费都由后台 insights_config 控制。
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Skeleton from '@/components/common/Skeleton.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import ModelRankingList from '@/components/leaderboard/ModelRankingList.vue'
import UserPodium from '@/components/leaderboard/UserPodium.vue'
import UserRankingList from '@/components/leaderboard/UserRankingList.vue'
import { formatDateRangeLabel, primaryMetricOf, visibleMetrics } from '@/components/leaderboard/leaderboardFormat'
import { vSegmented } from '@/directives/segmented'
import {
  LEADERBOARD_MAX_RANGE_DAYS,
  getModelLeaderboard,
  getUserLeaderboard,
  type LeaderboardModelsResponse,
  type LeaderboardUsersResponse,
} from '@/api/leaderboard'
import { useAppStore } from '@/stores'
import { ForkFeatureFlags } from '@/utils/forkFeatureFlags'
import { resolveFeatureFlag } from '@/utils/featureFlags'
import { extractApiErrorMessage } from '@/utils/apiError'

type LeaderboardBoard = 'users' | 'models'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

// 用户榜在前、默认显示；只开了一个榜时只显示那一个
const boards = computed<LeaderboardBoard[]>(() => {
  const settings = appStore.cachedPublicSettings
  const list: LeaderboardBoard[] = []
  if (resolveFeatureFlag(settings, ForkFeatureFlags.usageRanking)) list.push('users')
  if (resolveFeatureFlag(settings, ForkFeatureFlags.modelRanking)) list.push('models')
  return list
})

function queryBoard(): LeaderboardBoard | null {
  const tab = route.query.tab
  return tab === 'users' || tab === 'models' ? tab : null
}

// 用户点过或链接里指定了榜单就保持它；否则跟着开关落到第一个可用的榜（设置晚到时也会切到用户榜）
let boardPinned = queryBoard() !== null
const activeBoard = ref<LeaderboardBoard>(queryBoard() ?? boards.value[0] ?? 'users')

// 以浏览器本地时区格式化日期，默认范围就是用户看到的今天
function formatLocalDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = formatLocalDate(new Date())
const startDate = ref(today)
const endDate = ref(today)
const loading = ref(false)
const error = ref('')
const models = ref<LeaderboardModelsResponse | null>(null)
const users = ref<LeaderboardUsersResponse | null>(null)
let requestSeq = 0

const userMetrics = computed(() => visibleMetrics(Boolean(users.value?.show_cost)))
const primaryMetric = computed(() => primaryMetricOf(users.value?.sort_by, userMetrics.value))

const shownRange = computed(() => {
  const response = activeBoard.value === 'users' ? users.value : models.value
  return {
    start: response?.start_date || startDate.value,
    end: response?.end_date || endDate.value,
  }
})
const rangeLabel = computed(() => formatDateRangeLabel(shownRange.value.start, shownRange.value.end))
const rangeKey = computed(() => `${shownRange.value.start}_${shownRange.value.end}`)

watch(boards, (list) => {
  if (list.length === 0) return
  if (!boardPinned || !list.includes(activeBoard.value)) activeBoard.value = list[0]
}, { immediate: true })

function selectBoard(board: LeaderboardBoard) {
  boardPinned = true
  activeBoard.value = board
  void router.replace({ query: { ...route.query, tab: board } })
}

/** 含首尾的天数；格式不对时返回 NaN */
function rangeDays(start: string, end: string): number {
  const from = Date.parse(`${start}T00:00:00Z`)
  const to = Date.parse(`${end}T00:00:00Z`)
  return Math.round((to - from) / 86_400_000) + 1
}

function onDateRangeChange(range: { startDate: string; endDate: string; preset: string | null }) {
  startDate.value = range.startDate
  endDate.value = range.endDate
  void load()
}

async function load() {
  if (boards.value.length === 0) return
  const seq = ++requestSeq
  const range = { start_date: startDate.value, end_date: endDate.value }
  if (rangeDays(range.start_date, range.end_date) > LEADERBOARD_MAX_RANGE_DAYS) {
    error.value = t('leaderboard.rangeTooLong', { days: LEADERBOARD_MAX_RANGE_DAYS })
    loading.value = false
    return
  }
  loading.value = true
  error.value = ''
  try {
    if (activeBoard.value === 'users') {
      const result = await getUserLeaderboard(range)
      if (seq === requestSeq) users.value = result
    } else {
      const result = await getModelLeaderboard(range)
      if (seq === requestSeq) models.value = result
    }
  } catch (err) {
    // 快速切换时只处理最后一次请求的结果
    if (seq === requestSeq) error.value = extractApiErrorMessage(err, t('leaderboard.loadFailed'))
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

watch(activeBoard, () => {
  void load()
})

onMounted(() => {
  void load()
})
</script>

<style scoped>
.empty-box {
  @apply flex min-h-[360px] items-center justify-center rounded-xl border border-dashed border-gray-300 bg-white p-8 text-center dark:border-dark-600 dark:bg-dark-900;
}

.empty-icon {
  @apply mx-auto flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-dark-300;
}

.empty-title {
  @apply mt-4 text-base font-semibold text-gray-900 dark:text-white;
}

.empty-description {
  @apply mt-2 text-sm text-gray-500 dark:text-gray-400;
}

/* sm 以上日期选择器在右侧工具栏里，下拉框向左展开，不被视口右边裁掉 */
@media (min-width: 640px) {
  .leaderboard-toolbar :deep(.date-picker-dropdown) {
    left: auto;
    right: 0;
  }
}
</style>
