<template>
  <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800" data-testid="model-ranking">
    <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
      <div class="min-w-0">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('leaderboard.models.title') }}</h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.models.description') }}</p>
      </div>
      <div class="flex shrink-0 flex-col gap-0.5 text-xs text-gray-500 dark:text-gray-400 sm:items-end">
        <span data-testid="ranking-range-label">{{ rangeLabel }}</span>
        <span>
          {{ t('leaderboard.models.totalTokens') }}
          <AnimatedNumber
            :key="rangeKey"
            :value="totalTokens"
            :from="0"
            :format="totalFormat"
            class="ml-1 text-sm font-semibold text-gray-900 dark:text-white"
          />
        </span>
      </div>
    </div>

    <ol class="divide-y divide-gray-100 dark:divide-dark-700">
      <li
        v-for="(item, index) in items"
        :key="`${rangeKey}-${item.model}`"
        class="model-row group grid grid-cols-[auto_minmax(0,1fr)] gap-3 px-5 py-4 transition sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center"
        :class="rankRowClass(item.rank)"
        :style="{ '--i': Math.min(index, 14), '--bar-w': `${Math.max(item.share_percent, 0.5)}%` }"
        :data-rank="item.rank"
      >
        <span
          class="mt-1 inline-flex h-8 w-12 items-center justify-center rounded-lg text-sm font-semibold ring-1 sm:mt-0"
          :class="rankTheme(item.rank).badge"
        >
          #{{ item.rank }}
        </span>

        <div class="flex min-w-0 items-center gap-3">
          <span class="vendor-tile">
            <ModelIcon :model="item.model" size="20px" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex min-w-0 items-center gap-2">
              <router-link
                :to="{ path: '/models', query: { q: item.model } }"
                class="truncate text-sm font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
                :title="t('leaderboard.models.viewInCatalog')"
              >
                {{ item.model }}
              </router-link>
              <span v-if="item.platform" class="hidden shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-500 dark:bg-dark-700 dark:text-gray-400 sm:inline">
                {{ platformLabel(item.platform) }}
              </span>
            </div>
            <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <div class="share-bar h-full rounded-full" :class="barClass(item.rank)"></div>
            </div>
          </div>
        </div>

        <div
          class="col-span-2 grid gap-3 text-sm sm:col-span-1 sm:flex sm:justify-end sm:text-right"
          :style="metricGridStyle(3)"
        >
          <div :class="metricWidthClass('tokens')" data-column="tokens">
            <p class="font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatCompact(item.tokens) }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.metrics.tokens') }}</p>
          </div>
          <div :class="metricWidthClass('requests')" data-column="requests">
            <p class="font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatCompact(item.requests) }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.metrics.requests') }}</p>
          </div>
          <div :class="metricWidthClass('share')" data-column="share">
            <p class="font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatShare(item.share_percent) }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('leaderboard.models.share') }}</p>
          </div>
        </div>
      </li>
    </ol>

    <div v-if="otherSharePercent >= 0.05" class="flex items-center justify-between border-t border-gray-100 px-5 py-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
      <span>{{ t('leaderboard.models.others') }}</span>
      <span class="tabular-nums">{{ formatShare(otherSharePercent) }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
// fork：模型热度榜，与用户榜同一套列表样式（对齐 TokenRouter UsageRankingView 的 RankingRow），
// 头像换成模型图标，名字下方是占比条；占比条依次从左伸展，换日期范围时按 rangeKey 重新挂载、动画重放。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import type { LeaderboardModelItem } from '@/api/leaderboard'
import { compactFormatterFor, formatCompact } from '@/utils/compactNumber'
import { metricGridStyle, metricWidthClass, rankRowClass, rankTheme } from './leaderboardFormat'

const props = defineProps<{
  items: LeaderboardModelItem[]
  totalTokens: number
  otherSharePercent: number
  rangeLabel: string
  rangeKey: string
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

function barClass(rank: number): string {
  if (rank === 1) return 'bg-amber-400 dark:bg-amber-400/80'
  if (rank === 2) return 'bg-rose-400 dark:bg-rose-400/80'
  if (rank === 3) return 'bg-sky-400 dark:bg-sky-400/80'
  return 'bg-primary-500/70 dark:bg-primary-400/70'
}
</script>

<style scoped>
.model-row {
  animation: model-row-enter 320ms var(--motion-ease) backwards;
  animation-delay: calc(var(--i, 0) * 30ms);
}

.vendor-tile {
  @apply flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-900;
  transition: transform var(--motion-layout) var(--motion-ease);
}

.model-row:hover .vendor-tile {
  transform: scale(1.08) rotate(-4deg);
}

/* 占比条从 0 伸到 --bar-w，相邻行错开 50ms */
.share-bar {
  width: var(--bar-w);
  animation: share-bar-grow 600ms var(--motion-ease) backwards;
  animation-delay: calc(var(--i, 0) * 50ms);
}

@keyframes model-row-enter {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
}

@keyframes share-bar-grow {
  from {
    width: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .model-row,
  .share-bar {
    animation: none;
  }

  .model-row:hover .vendor-tile {
    transform: none;
  }
}
</style>
