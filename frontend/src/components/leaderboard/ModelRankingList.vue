<template>
  <section class="card overflow-hidden p-0" data-testid="model-ranking">
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.models.title') }}</h2>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.models.description') }}</p>
      </div>
      <div class="text-right">
        <div class="text-[11px] uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">{{ t('leaderboard.models.totalTokens') }}</div>
        <AnimatedNumber
          :value="totalTokens"
          :from="0"
          :format="totalFormat"
          class="text-xl font-semibold text-gray-900 dark:text-white"
        />
      </div>
    </header>

    <ol class="divide-y divide-gray-100 dark:divide-dark-800">
      <li
        v-for="(item, index) in items"
        :key="`${period}-${item.model}`"
        class="model-row group"
        :style="{ '--i': Math.min(index, 14), '--bar-w': `${Math.max(item.share_percent, 0.5)}%` }"
      >
        <span class="rank-badge" :class="rankClass(item.rank)">{{ item.rank }}</span>

        <span class="vendor-tile">
          <ModelIcon :model="item.model" size="20px" />
        </span>

        <div class="min-w-0 flex-1">
          <div class="flex min-w-0 items-center gap-2">
            <router-link
              :to="{ path: '/models', query: { q: item.model } }"
              class="truncate font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
              :title="t('leaderboard.models.viewInCatalog')"
            >
              {{ item.model }}
            </router-link>
            <span v-if="item.platform" class="hidden shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-500 dark:bg-dark-800 dark:text-gray-400 sm:inline">
              {{ platformLabel(item.platform) }}
            </span>
          </div>
          <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
            <div class="share-bar h-full rounded-full" :class="barClass(item.rank)"></div>
          </div>
        </div>

        <div class="w-20 shrink-0 text-right sm:w-28">
          <div class="text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatCompact(item.tokens) }}</div>
          <div class="text-[11px] tabular-nums text-gray-500 dark:text-gray-400">
            {{ t('leaderboard.models.requests', { count: formatCompact(item.requests) }) }}
          </div>
        </div>
        <div class="hidden w-16 shrink-0 text-right text-sm font-medium tabular-nums text-gray-600 dark:text-gray-300 sm:block">
          {{ formatShare(item.share_percent) }}
        </div>
      </li>
    </ol>

    <footer v-if="otherSharePercent >= 0.05" class="flex items-center justify-between border-t border-gray-100 px-5 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
      <span>{{ t('leaderboard.models.others') }}</span>
      <span class="tabular-nums">{{ formatShare(otherSharePercent) }}</span>
    </footer>
  </section>
</template>

<script setup lang="ts">
// fork：模型热度榜。占比条依次从左伸展（参考 TokenRouter UserDashboardTopModels），
// 换时间窗时按 period 重新挂载行，动画重放。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import type { LeaderboardModelItem, LeaderboardWindow } from '@/api/leaderboard'
import { compactFormatterFor, formatCompact } from '@/utils/compactNumber'

const props = defineProps<{
  items: LeaderboardModelItem[]
  period: LeaderboardWindow
  totalTokens: number
  otherSharePercent: number
}>()

const { t } = useI18n()

const totalFormat = computed(() => compactFormatterFor(props.totalTokens))

function formatShare(value: number): string {
  if (value > 0 && value < 0.1) return '<0.1%'
  return `${value.toFixed(1)}%`
}

function platformLabel(platform: string): string {
  return t(`admin.groups.platforms.${platform}`)
}

function rankClass(rank: number): string {
  if (rank === 1) return 'rank-badge-gold'
  if (rank === 2) return 'rank-badge-silver'
  if (rank === 3) return 'rank-badge-bronze'
  return 'rank-badge-plain'
}

function barClass(rank: number): string {
  if (rank === 1) return 'bg-gradient-to-r from-amber-400 to-orange-500'
  if (rank === 2) return 'bg-gradient-to-r from-sky-400 to-indigo-500'
  if (rank === 3) return 'bg-gradient-to-r from-emerald-400 to-teal-500'
  return 'bg-primary-500/70 dark:bg-primary-400/70'
}
</script>

<style scoped>
.model-row {
  @apply flex items-center gap-3 px-5 py-3.5;
  transition: background-color var(--motion-fast) var(--motion-ease);
}

.model-row:hover {
  @apply bg-gray-50/80 dark:bg-dark-800/40;
}

.rank-badge {
  @apply flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-xs font-bold tabular-nums;
}

.rank-badge-gold {
  @apply bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300;
}

.rank-badge-silver {
  @apply bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300;
}

.rank-badge-bronze {
  @apply bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300;
}

.rank-badge-plain {
  @apply bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-gray-400;
}

.vendor-tile {
  @apply flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-gray-200/80 bg-white dark:border-dark-700 dark:bg-dark-800;
  transition: transform var(--motion-layout) var(--motion-ease);
}

.model-row:hover .vendor-tile {
  transform: scale(1.08) rotate(-4deg);
}

/* 占比条从 0 伸到 --bar-w，相邻行错开 50ms，越往后越淡 */
.share-bar {
  width: var(--bar-w);
  animation: share-bar-grow 600ms var(--motion-ease) backwards;
  animation-delay: calc(var(--i, 0) * 50ms);
}

@keyframes share-bar-grow {
  from {
    width: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .share-bar {
    animation: none;
  }

  .model-row:hover .vendor-tile {
    transform: none;
  }
}
</style>
