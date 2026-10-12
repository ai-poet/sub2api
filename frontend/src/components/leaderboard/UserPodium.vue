<template>
  <!-- 前三名领奖台：第一名居中且更高，第二名在左、第三名在右（items-end 让三张卡底部对齐） -->
  <section class="grid grid-cols-1 gap-4 md:grid-cols-3 md:items-end" data-testid="user-podium">
    <article
      v-for="item in podium"
      :key="`${period}-${metric}-${item.rank}`"
      class="podium-card"
      :class="[THEMES[item.rank - 1].card, orderClass(item.rank), item.rank === 1 ? 'md:min-h-[260px] md:p-6' : 'md:min-h-[218px]', { 'md:col-start-2': podium.length === 1 }]"
      :style="{ '--rise-delay': `${RISE_DELAY[item.rank - 1]}ms` }"
      :data-rank="item.rank"
    >
      <div class="pointer-events-none absolute -right-10 -top-10 h-28 w-28 rounded-full blur-2xl" :class="THEMES[item.rank - 1].glow"></div>

      <div class="relative flex items-center justify-between">
        <span class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-semibold" :class="THEMES[item.rank - 1].pill">
          <Icon :name="THEMES[item.rank - 1].icon" size="xs" :stroke-width="2" />
          {{ t('leaderboard.users.rankLabel', { rank: item.rank }) }}
        </span>
        <span v-if="item.is_self" class="rounded-full bg-primary-500 px-2 py-0.5 text-[11px] font-semibold text-white">
          {{ t('leaderboard.users.you') }}
        </span>
      </div>

      <div class="relative mt-4 flex flex-col items-center text-center">
        <IdenticonAvatar :seed="item.avatar_seed" :size-class="item.rank === 1 ? 'h-16 w-16' : 'h-14 w-14'" />
        <div class="mt-3 max-w-full truncate text-sm font-semibold text-gray-900 dark:text-white">
          {{ item.display_name || t('leaderboard.users.anonymous') }}
        </div>
        <AnimatedNumber
          :value="primaryValue(item)"
          :from="0"
          :duration="1200"
          :format="primaryFormat(item)"
          class="mt-2 font-semibold tracking-tight text-gray-900 dark:text-white"
          :class="item.rank === 1 ? 'text-3xl' : 'text-2xl'"
        />
        <div class="text-xs text-gray-500 dark:text-gray-400">{{ t(`leaderboard.metrics.${metric}`) }}</div>
      </div>

      <dl class="relative mt-4 grid gap-2 text-center text-xs" :class="secondaryMetrics(item).length > 1 ? 'grid-cols-2' : 'grid-cols-1'">
        <div v-for="entry in secondaryMetrics(item)" :key="entry.key" class="rounded-xl bg-white/60 px-2 py-1.5 dark:bg-dark-900/40">
          <dt class="text-gray-500 dark:text-gray-400">{{ entry.label }}</dt>
          <dd class="mt-0.5 font-semibold tabular-nums text-gray-800 dark:text-gray-100">{{ entry.value }}</dd>
        </div>
      </dl>
    </article>
  </section>
</template>

<script setup lang="ts">
// fork：用户榜领奖台（参考 TokenRouter UsageRankingView 的 TopRankCard）。
// 三张卡按 2 → 1 → 3 的顺序依次升起，减少动态效果时直接显示。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import IdenticonAvatar from './IdenticonAvatar.vue'
import type { LeaderboardMetric, LeaderboardUserItem, LeaderboardWindow } from '@/api/leaderboard'
import { metricValue, secondaryMetricEntries, formatMetric, metricFormatterFor } from './leaderboardFormat'

const props = defineProps<{
  items: LeaderboardUserItem[]
  metric: LeaderboardMetric
  period: LeaderboardWindow
  showCost: boolean
}>()

const { t } = useI18n()

const podium = computed(() => props.items.slice(0, 3))

// 金 / 玫红 / 天蓝（与 TokenRouter 一致，不用金银铜，在深浅色下都清楚）
const THEMES = [
  {
    card: 'border-amber-200/80 bg-gradient-to-b from-amber-50 to-white dark:border-amber-500/20 dark:from-amber-500/10 dark:to-dark-900',
    glow: 'bg-amber-300/50 dark:bg-amber-500/20',
    pill: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    icon: 'fire' as const,
  },
  {
    card: 'border-rose-200/80 bg-gradient-to-b from-rose-50 to-white dark:border-rose-500/20 dark:from-rose-500/10 dark:to-dark-900',
    glow: 'bg-rose-300/50 dark:bg-rose-500/20',
    pill: 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300',
    icon: 'trendingUp' as const,
  },
  {
    card: 'border-sky-200/80 bg-gradient-to-b from-sky-50 to-white dark:border-sky-500/20 dark:from-sky-500/10 dark:to-dark-900',
    glow: 'bg-sky-300/50 dark:bg-sky-500/20',
    pill: 'bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300',
    icon: 'badge' as const,
  },
]

// 升起顺序：第二名、第一名、第三名
const RISE_DELAY = [120, 0, 240]

function orderClass(rank: number): string {
  if (rank === 1) return 'md:order-2'
  if (rank === 2) return 'md:order-1'
  return 'md:order-3'
}

function primaryValue(item: LeaderboardUserItem): number {
  return metricValue(item, props.metric)
}

function primaryFormat(item: LeaderboardUserItem): (value: number) => string {
  return metricFormatterFor(props.metric, metricValue(item, props.metric))
}

function secondaryMetrics(item: LeaderboardUserItem) {
  return secondaryMetricEntries(item, props.metric, props.showCost).map(entry => ({
    key: entry.metric,
    label: t(`leaderboard.metrics.${entry.metric}`),
    value: formatMetric(entry.metric, entry.value),
  }))
}
</script>

<style scoped>
.podium-card {
  @apply relative overflow-hidden rounded-3xl border p-5 shadow-sm;
  animation: podium-rise 520ms var(--motion-ease) backwards;
  animation-delay: var(--rise-delay, 0ms);
  transition: transform var(--motion-layout) var(--motion-ease), box-shadow var(--motion-layout) var(--motion-ease);
}

.podium-card:hover {
  @apply -translate-y-1 shadow-md;
}

@keyframes podium-rise {
  from {
    opacity: 0;
    transform: translateY(16px) scale(0.98);
  }
}

@media (prefers-reduced-motion: reduce) {
  .podium-card {
    animation: none;
  }

  .podium-card:hover {
    transform: none;
  }
}
</style>
