<template>
  <section class="card overflow-hidden p-0" data-testid="user-ranking-list">
    <header class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.users.listTitle') }}</h2>
      <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.users.privacyNote') }}</span>
    </header>
    <ol class="divide-y divide-gray-100 dark:divide-dark-800">
      <li
        v-for="(item, index) in items"
        :key="`${period}-${metric}-${item.rank}`"
        class="user-row"
        :class="{ 'user-row-self': item.is_self }"
        :style="{ '--i': Math.min(index, 14) }"
        :data-self="item.is_self || undefined"
      >
        <span class="w-10 shrink-0 text-center text-sm font-semibold tabular-nums text-gray-400 dark:text-gray-500">#{{ item.rank }}</span>
        <IdenticonAvatar :seed="item.avatar_seed" size-class="h-9 w-9" />
        <div class="flex min-w-0 flex-1 items-center gap-2">
          <span class="truncate text-sm font-medium text-gray-900 dark:text-white">
            {{ item.display_name || t('leaderboard.users.anonymous') }}
          </span>
          <span v-if="item.is_self" class="shrink-0 rounded-full bg-primary-500 px-2 py-0.5 text-[11px] font-semibold text-white">
            {{ t('leaderboard.users.you') }}
          </span>
        </div>
        <div
          v-for="column in columns"
          :key="column"
          class="shrink-0 text-right tabular-nums"
          :class="[column === metric ? 'w-24 text-sm font-semibold text-gray-900 dark:text-white' : 'hidden w-24 text-xs text-gray-500 dark:text-gray-400 sm:block']"
          :data-column="column"
        >
          {{ formatMetric(column, metricValue(item, column)) }}
        </div>
      </li>
    </ol>
  </section>
</template>

<script setup lang="ts">
// fork：用户榜完整列表。数值列定宽右对齐，跨行对齐；当前用户整行高亮。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IdenticonAvatar from './IdenticonAvatar.vue'
import type { LeaderboardMetric, LeaderboardUserItem, LeaderboardWindow } from '@/api/leaderboard'
import { formatMetric, metricValue } from './leaderboardFormat'

const props = defineProps<{
  items: LeaderboardUserItem[]
  metric: LeaderboardMetric
  period: LeaderboardWindow
  showCost: boolean
}>()

const { t } = useI18n()

// 排序指标放第一列，其余跟在后面
const columns = computed<LeaderboardMetric[]>(() => {
  const all: LeaderboardMetric[] = props.showCost ? ['tokens', 'requests', 'cost'] : ['tokens', 'requests']
  return [props.metric, ...all.filter(column => column !== props.metric)]
})
</script>

<style scoped>
.user-row {
  @apply flex items-center gap-3 px-5 py-3;
  animation: user-row-enter 320ms var(--motion-ease) backwards;
  animation-delay: calc(var(--i, 0) * 30ms);
  transition: background-color var(--motion-fast) var(--motion-ease);
}

.user-row:hover {
  @apply bg-gray-50/80 dark:bg-dark-800/40;
}

.user-row-self {
  @apply bg-primary-50/70 dark:bg-primary-900/15;
}

@keyframes user-row-enter {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .user-row {
    animation: none;
  }
}
</style>
