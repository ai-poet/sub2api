<template>
  <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800" data-testid="user-ranking-list">
    <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.listTitle') }}</h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('leaderboard.limitHint', { limit: limit || items.length }) }} · {{ identified ? t('leaderboard.users.identifiedNote') : t('leaderboard.users.privacyNote') }}
        </p>
      </div>
      <span class="text-xs text-gray-500 dark:text-gray-400" data-testid="ranking-range-label">{{ rangeLabel }}</span>
    </div>

    <ol class="divide-y divide-gray-100 dark:divide-dark-700">
      <li
        v-for="(item, index) in items"
        :key="`${rangeKey}-${item.rank}`"
        class="ranking-row grid grid-cols-[auto_minmax(0,1fr)] gap-3 px-5 py-4 transition sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center"
        :class="rowClass(item)"
        :style="{ '--i': Math.min(index, 14) }"
        :data-rank="item.rank"
        :data-self="item.is_self || undefined"
      >
        <span
          class="mt-1 inline-flex h-8 w-12 items-center justify-center rounded-lg text-sm font-semibold ring-1 sm:mt-0"
          :class="rankTheme(item.rank).badge"
        >
          #{{ item.rank }}
        </span>

        <div class="flex min-w-0 items-center gap-3">
          <IdenticonAvatar :seed="item.avatar_seed" size-class="h-10 w-10" />
          <div class="flex min-w-0 items-center gap-2">
            <LeaderboardUserName :item="item" name-class="text-sm font-medium text-gray-900 dark:text-white" />
            <span v-if="item.is_self" class="shrink-0 rounded-full bg-primary-500 px-2 py-0.5 text-[11px] font-semibold text-white">
              {{ t('leaderboard.users.you') }}
            </span>
          </div>
        </div>

        <div
          class="col-span-2 grid gap-3 text-sm sm:col-span-1 sm:flex sm:justify-end sm:text-right"
          :style="metricGridStyle(columns.length)"
          data-testid="usage-ranking-row-metrics"
        >
          <div v-for="column in columns" :key="column" :class="metricWidthClass(column)" :data-column="column">
            <p class="font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatMetric(column, metricValue(item, column)) }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t(`leaderboard.metrics.${column}`) }}</p>
          </div>
        </div>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
// fork：用户榜完整列表（对齐 TokenRouter UsageRankingView 的 RankingRow）。前三名整行用主题底色，
// 指标列定宽右对齐、主指标在前；当前用户整行高亮并带「你」标记。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IdenticonAvatar from './IdenticonAvatar.vue'
import LeaderboardUserName from './LeaderboardUserName.vue'
import type { LeaderboardMetric, LeaderboardUserItem } from '@/api/leaderboard'
import {
  formatMetric,
  metricGridStyle,
  metricValue,
  metricWidthClass,
  orderedMetrics,
  rankRowClass,
  rankTheme,
} from './leaderboardFormat'

const props = defineProps<{
  items: LeaderboardUserItem[]
  primaryMetric: LeaderboardMetric
  metrics: LeaderboardMetric[]
  limit: number
  rangeLabel: string
  rangeKey: string
  /** 实名视角（管理员 / 运维管理员）：表头说明换成实名提示 */
  identified?: boolean
}>()

const { t } = useI18n()

const columns = computed(() => orderedMetrics(props.metrics, props.primaryMetric))

/** 前三名用主题底色（自己再加描边）；其余行透明，自己那行用主色浅底。 */
function rowClass(item: LeaderboardUserItem): string {
  if (item.rank <= 3) {
    return item.is_self ? `${rankRowClass(item.rank)} ring-1 ring-inset ring-primary-300 dark:ring-primary-500/40` : rankRowClass(item.rank)
  }
  return item.is_self ? 'bg-primary-50/70 dark:bg-primary-900/15' : rankRowClass(item.rank)
}
</script>

<style scoped>
.ranking-row {
  animation: ranking-row-enter 320ms var(--motion-ease) backwards;
  animation-delay: calc(var(--i, 0) * 30ms);
}

@keyframes ranking-row-enter {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .ranking-row {
    animation: none;
  }
}
</style>
