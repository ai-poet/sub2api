<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <section class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <div class="inline-flex items-center gap-2 rounded-full bg-amber-50 px-3 py-1 text-xs font-semibold text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
            <Icon name="trophy" size="sm" />
            {{ t('leaderboard.caption') }}
          </div>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('leaderboard.intro') }}</p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <div v-if="tabs.length > 1" v-segmented class="segmented" role="tablist" :aria-label="t('leaderboard.tabsLabel')">
            <button
              v-for="tab in tabs"
              :key="tab"
              type="button"
              role="tab"
              class="segmented-item px-3 py-1.5 text-sm"
              :class="{ 'segmented-item-active': activeTab === tab }"
              :aria-selected="activeTab === tab"
              :data-tab="tab"
              @click="selectTab(tab)"
            >
              {{ t(`leaderboard.tabs.${tab}`) }}
            </button>
          </div>

          <div v-segmented class="segmented" role="group" :aria-label="t('leaderboard.windowLabel')">
            <button
              v-for="option in LEADERBOARD_WINDOWS"
              :key="option"
              type="button"
              class="segmented-item px-3 py-1.5 text-sm"
              :class="{ 'segmented-item-active': period === option }"
              :aria-pressed="period === option"
              :data-window="option"
              @click="period = option"
            >
              {{ t(`leaderboard.windows.${option}`) }}
            </button>
          </div>

          <button
            type="button"
            class="btn btn-secondary px-3"
            :disabled="loading"
            :title="t('common.refresh')"
            :aria-label="t('common.refresh')"
            @click="load"
          >
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          </button>
        </div>
      </section>

      <section v-if="tabs.length === 0" class="card p-12 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('leaderboard.disabled') }}
      </section>

      <template v-else-if="activeTab === 'models'">
        <div v-if="loading && !models" class="card space-y-3 p-5" aria-busy="true" data-testid="leaderboard-skeleton">
          <div v-for="row in 8" :key="row" class="h-10 animate-pulse rounded-xl bg-gray-100 motion-reduce:animate-none dark:bg-dark-800"></div>
        </div>
        <section v-else-if="error" class="card p-10 text-center text-sm text-rose-600 dark:text-rose-400">{{ error }}</section>
        <section v-else-if="models && !models.enabled" class="card p-10 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('leaderboard.disabled') }}
        </section>
        <section v-else-if="models && models.items.length === 0" class="empty-state">
          <Icon name="chartBar" size="xl" class="text-gray-300 dark:text-dark-600" />
          <p>{{ t('leaderboard.empty') }}</p>
        </section>
        <ModelRankingList
          v-else-if="models"
          :items="models.items"
          :period="period"
          :total-tokens="models.total_tokens"
          :other-share-percent="models.other_share_percent"
        />
      </template>

      <template v-else>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div v-segmented class="segmented" role="group" :aria-label="t('leaderboard.metricLabel')">
            <button
              v-for="option in metricOptions"
              :key="option"
              type="button"
              class="segmented-item px-3 py-1.5 text-sm"
              :class="{ 'segmented-item-active': metric === option }"
              :aria-pressed="metric === option"
              :data-metric="option"
              @click="metric = option"
            >
              {{ t(`leaderboard.metrics.${option}`) }}
            </button>
          </div>
          <span v-if="generatedLabel" class="text-xs text-gray-500 dark:text-gray-400">{{ generatedLabel }}</span>
        </div>

        <div v-if="loading && !users" class="space-y-4" aria-busy="true" data-testid="leaderboard-skeleton">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div v-for="card in 3" :key="card" class="h-52 animate-pulse rounded-3xl bg-gray-100 motion-reduce:animate-none dark:bg-dark-800"></div>
          </div>
        </div>
        <section v-else-if="error" class="card p-10 text-center text-sm text-rose-600 dark:text-rose-400">{{ error }}</section>
        <section v-else-if="users && !users.enabled" class="card p-10 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('leaderboard.disabled') }}
        </section>
        <section v-else-if="users && users.insufficient_data" class="empty-state" data-testid="leaderboard-insufficient">
          <Icon name="users" size="xl" class="text-gray-300 dark:text-dark-600" />
          <p>{{ t('leaderboard.users.insufficient') }}</p>
        </section>
        <section v-else-if="users && users.items.length === 0" class="empty-state">
          <Icon name="chartBar" size="xl" class="text-gray-300 dark:text-dark-600" />
          <p>{{ t('leaderboard.empty') }}</p>
        </section>
        <template v-else-if="users">
          <UserPodium :items="users.items" :metric="users.metric" :period="period" :show-cost="users.show_cost" />
          <UserRankingList :items="users.items" :metric="users.metric" :period="period" :show-cost="users.show_cost" />
        </template>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
// fork：排行榜（模型榜 + 用户榜）。两个榜的开关由后台 insights_config 控制，
// 用户榜默认关闭；时间窗只有固定档位，后端据此缓存。
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelRankingList from '@/components/leaderboard/ModelRankingList.vue'
import UserPodium from '@/components/leaderboard/UserPodium.vue'
import UserRankingList from '@/components/leaderboard/UserRankingList.vue'
import { vSegmented } from '@/directives/segmented'
import {
  LEADERBOARD_WINDOWS,
  getModelLeaderboard,
  getUserLeaderboard,
  type LeaderboardMetric,
  type LeaderboardModelsResponse,
  type LeaderboardUsersResponse,
  type LeaderboardWindow,
} from '@/api/leaderboard'
import { useAppStore } from '@/stores'
import { ForkFeatureFlags } from '@/utils/forkFeatureFlags'
import { resolveFeatureFlag } from '@/utils/featureFlags'
import { extractApiErrorMessage } from '@/utils/apiError'

type LeaderboardTab = 'models' | 'users'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const tabs = computed<LeaderboardTab[]>(() => {
  const settings = appStore.cachedPublicSettings
  const list: LeaderboardTab[] = []
  if (resolveFeatureFlag(settings, ForkFeatureFlags.modelRanking)) list.push('models')
  if (resolveFeatureFlag(settings, ForkFeatureFlags.usageRanking)) list.push('users')
  return list
})

const activeTab = ref<LeaderboardTab>(route.query.tab === 'users' ? 'users' : 'models')
const period = ref<LeaderboardWindow>('7d')
const metric = ref<LeaderboardMetric>('tokens')
const loading = ref(false)
const error = ref('')
const models = ref<LeaderboardModelsResponse | null>(null)
const users = ref<LeaderboardUsersResponse | null>(null)
let requestSeq = 0

const metricOptions = computed<LeaderboardMetric[]>(() =>
  users.value?.show_cost ? ['tokens', 'requests', 'cost'] : ['tokens', 'requests'],
)

const generatedLabel = computed(() => {
  const at = activeTab.value === 'users' ? users.value?.generated_at : models.value?.generated_at
  if (!at) return ''
  const date = new Date(at)
  return Number.isNaN(date.getTime()) ? '' : t('leaderboard.generatedAt', { time: date.toLocaleString() })
})

// 开关变化（或 ?tab= 指向关闭的榜）时落到第一个可用的榜
watch(tabs, (list) => {
  if (list.length > 0 && !list.includes(activeTab.value)) activeTab.value = list[0]
}, { immediate: true })

function selectTab(tab: LeaderboardTab) {
  activeTab.value = tab
  void router.replace({ query: { ...route.query, tab } })
}

async function load() {
  if (tabs.value.length === 0) return
  const seq = ++requestSeq
  loading.value = true
  error.value = ''
  try {
    if (activeTab.value === 'models') {
      const result = await getModelLeaderboard(period.value)
      if (seq === requestSeq) models.value = result
    } else {
      const result = await getUserLeaderboard(period.value, metric.value)
      if (seq === requestSeq) users.value = result
    }
  } catch (err) {
    // 快速切换时只处理最后一次请求的结果
    if (seq === requestSeq) error.value = extractApiErrorMessage(err, t('leaderboard.loadFailed'))
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

watch([activeTab, period, metric], () => {
  void load()
})

onMounted(() => {
  void load()
})
</script>

<style scoped>
.empty-state {
  @apply card flex min-h-[280px] flex-col items-center justify-center gap-3 border-dashed p-10 text-center text-sm text-gray-500 dark:text-gray-400;
}
</style>
