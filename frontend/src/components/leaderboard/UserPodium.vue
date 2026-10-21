<template>
  <!-- 前三名卡片：桌面端按第二、第一、第三名排列，第一名居中且更高（items-end 让三张卡底部对齐） -->
  <section class="grid grid-cols-1 gap-4 md:grid-cols-3 md:items-end" data-testid="user-podium">
    <article
      v-for="item in podium"
      :key="`${rangeKey}-${item.rank}`"
      class="podium-card relative overflow-hidden rounded-lg border p-5 shadow-sm"
      :class="[
        rankTheme(item.rank).card,
        item.rank === 1 ? 'md:min-h-[260px] md:p-6' : 'md:min-h-[220px]',
        orderClass(item.rank),
        podium.length === 1 && item.rank === 1 ? 'md:col-start-2' : '',
        item.is_self ? 'ring-2 ring-primary-400/60 dark:ring-primary-500/50' : '',
      ]"
      :style="{ '--rise-delay': `${RISE_DELAY[item.rank - 1] ?? 0}ms` }"
      :data-rank="item.rank"
      :data-self="item.is_self || undefined"
    >
      <div class="pointer-events-none absolute -right-10 -top-10 h-28 w-28 rounded-full blur-2xl" :class="rankTheme(item.rank).glow"></div>

      <div class="relative flex items-start justify-between gap-2">
        <span class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-semibold ring-1" :class="rankTheme(item.rank).badge">
          <Icon :name="rankTheme(item.rank).icon" size="xs" />
          #{{ item.rank }}
        </span>
        <span v-if="item.is_self" class="rounded-full bg-primary-500 px-2 py-0.5 text-[11px] font-semibold text-white">
          {{ t('leaderboard.users.you') }}
        </span>
      </div>

      <div class="relative mt-7 flex flex-col items-center text-center">
        <IdenticonAvatar :seed="item.avatar_seed" size-class="h-16 w-16" />
        <LeaderboardUserName
          :item="item"
          tag="h3"
          center
          class="mt-4 max-w-full"
          name-class="text-lg font-semibold text-gray-900 dark:text-white"
        />
        <AnimatedNumber
          :value="metricValue(item, primaryMetric)"
          :from="0"
          :duration="1200"
          :format="metricFormatterFor(primaryMetric, metricValue(item, primaryMetric))"
          class="mt-2 text-3xl font-semibold text-gray-900 dark:text-white"
          data-testid="podium-primary"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t(`leaderboard.metrics.${primaryMetric}`) }}</p>
      </div>

      <div
        v-if="secondaryMetrics.length > 0"
        class="relative mt-6 grid gap-2 text-center text-xs text-gray-500 dark:text-gray-400"
        :style="metricGridStyle(secondaryMetrics.length)"
      >
        <div v-for="metric in secondaryMetrics" :key="metric" :data-metric="metric">
          <p class="font-medium text-gray-900 dark:text-white">{{ formatMetric(metric, metricValue(item, metric)) }}</p>
          <p>{{ t(`leaderboard.metrics.${metric}`) }}</p>
        </div>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
// fork：用户榜前三名（对齐 TokenRouter UsageRankingView 的 TopRankCard）。主指标由后台设定并滚动计数，
// 三张卡按 2 → 1 → 3 的顺序依次升起，减少动态效果时直接显示。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import IdenticonAvatar from './IdenticonAvatar.vue'
import LeaderboardUserName from './LeaderboardUserName.vue'
import type { LeaderboardMetric, LeaderboardUserItem } from '@/api/leaderboard'
import { formatMetric, metricFormatterFor, metricGridStyle, metricValue, rankTheme } from './leaderboardFormat'

const props = defineProps<{
  items: LeaderboardUserItem[]
  primaryMetric: LeaderboardMetric
  metrics: LeaderboardMetric[]
  /** 日期范围变化时重新挂载卡片，升起动画和滚动数字重放 */
  rangeKey: string
}>()

const { t } = useI18n()

const podium = computed(() => props.items.slice(0, 3))
const secondaryMetrics = computed(() => props.metrics.filter(metric => metric !== props.primaryMetric))

// 升起顺序：第二名、第一名、第三名
const RISE_DELAY = [120, 0, 240]

function orderClass(rank: number): string {
  if (rank === 1) return 'md:order-2'
  if (rank === 2) return 'md:order-1'
  return 'md:order-3'
}
</script>

<style scoped>
.podium-card {
  animation: podium-rise 520ms var(--motion-ease) backwards;
  animation-delay: var(--rise-delay, 0ms);
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
}
</style>
