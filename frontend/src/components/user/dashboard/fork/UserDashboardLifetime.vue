<template>
  <!-- 累计用量：开户至今的请求数、Token 与花费（标准价 vs 实付价），原仪表盘的全时段总计保留在这里 -->
  <div class="card p-4" data-testid="lifetime-usage" :aria-busy="!stats">
    <div class="mb-4 flex min-h-7 items-center justify-between gap-2">
      <div class="flex min-w-0 items-baseline gap-2">
        <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-dark-50">
          <Icon name="chartBar" size="sm" class="text-primary-600 dark:text-primary-400" />
          {{ t('dashboard.fork.lifetime.title') }}
        </h3>
        <span class="truncate text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.fork.lifetime.subtitle') }}</span>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
      <!-- 累计请求 -->
      <div class="lifetime-tile" data-testid="lifetime-requests">
        <div class="flex items-center gap-2">
          <span class="lifetime-icon" :class="[METRIC_TONES.requests.tile, METRIC_TONES.requests.icon]">
            <Icon name="swap" size="sm" />
          </span>
          <span class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('dashboard.fork.lifetime.requests') }}</span>
        </div>
        <Skeleton v-if="!stats" class="mt-3" :width="120" :height="30" />
        <template v-else>
          <AnimatedNumber :value="stats.total_requests" :from="0" class="lifetime-value" />
          <p class="lifetime-sub">{{ t('dashboard.fork.lifetime.today', { value: formatNumber(stats.today_requests) }) }}</p>
        </template>
      </div>

      <!-- 累计 Token：输入 / 输出 / 缓存拆分 -->
      <div class="lifetime-tile" data-testid="lifetime-tokens">
        <div class="flex items-center gap-2">
          <span class="lifetime-icon" :class="[METRIC_TONES.tokens.tile, METRIC_TONES.tokens.icon]">
            <Icon name="cube" size="sm" />
          </span>
          <span class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('dashboard.fork.lifetime.tokens') }}</span>
        </div>
        <Skeleton v-if="!stats" class="mt-3" :width="120" :height="30" />
        <template v-else>
          <AnimatedNumber :value="stats.total_tokens" :from="0" :format="compactFormatterFor(stats.total_tokens)" class="lifetime-value" />
          <p class="lifetime-sub">
            {{ t('dashboard.input') }} {{ formatTokensK(stats.total_input_tokens) }}
            · {{ t('dashboard.output') }} {{ formatTokensK(stats.total_output_tokens) }}
            · {{ t('dashboard.cache') }} {{ formatTokensK(stats.total_cache_creation_tokens + stats.total_cache_read_tokens) }}
          </p>
          <p class="lifetime-sub">{{ t('dashboard.fork.lifetime.today', { value: formatTokensK(stats.today_tokens) }) }}</p>
        </template>
      </div>

      <!-- 累计花费：实付价与标准价对比 -->
      <div class="lifetime-tile" data-testid="lifetime-cost">
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <span class="lifetime-icon" :class="[METRIC_TONES.cost.tile, METRIC_TONES.cost.icon]">
              <Icon name="dollar" size="sm" />
            </span>
            <span class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('dashboard.fork.lifetime.cost') }}</span>
          </div>
          <span
            v-if="stats && savings"
            class="rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300"
            data-testid="lifetime-savings"
          >
            {{ t('dashboard.fork.lifetime.saved', { amount: formatUsd(savings.amount), percent: savings.percent }) }}
          </span>
        </div>
        <Skeleton v-if="!stats" class="mt-3" :width="120" :height="30" />
        <template v-else>
          <div class="mt-3 flex items-baseline gap-2">
            <AnimatedNumber :value="stats.total_actual_cost" :from="0" :format="formatUsdAuto" class="lifetime-value !mt-0" />
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.actual') }}</span>
          </div>
          <!-- 实付占标准价的比例：整条是标准价，填充部分是实付 -->
          <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600" :title="t('dashboard.fork.lifetime.barHint')">
            <div class="lifetime-bar h-full rounded-full bg-amber-500" :style="{ '--bar-w': `${actualShare}%` }"></div>
          </div>
          <p class="lifetime-sub">
            {{ t('dashboard.standard') }} {{ formatUsd(stats.total_cost) }}
          </p>
          <p class="lifetime-sub">
            {{ t('dashboard.fork.lifetime.todayCost', { actual: formatUsd(stats.today_actual_cost), standard: formatUsd(stats.today_cost) }) }}
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：累计用量卡片。仪表盘换成 TokenRouter 布局后，原 UserDashboardStats 里的全时段总计
// （累计请求、累计 Token 及拆分、标准价与实付价对比）放在这里，数据复用实时状态条轮询到的统计。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Skeleton from '@/components/common/Skeleton.vue'
import Icon from '@/components/icons/Icon.vue'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import type { UserDashboardStats } from '@/api/usage'
import { compactFormatterFor } from '@/utils/compactNumber'
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { formatUsd } from './format'
import { METRIC_TONES } from './metricTones'

const props = defineProps<{
  stats: UserDashboardStats | null
}>()

const { t } = useI18n()

const formatUsdAuto = (value: number): string => formatUsd(value)

/** 实付占标准价的百分比（0..100）；标准价为 0 时按实付是否为 0 显示满条或空条。 */
const actualShare = computed(() => {
  const standard = props.stats?.total_cost ?? 0
  const actual = props.stats?.total_actual_cost ?? 0
  if (standard <= 0) return actual > 0 ? 100 : 0
  return Math.min(100, Math.max(0, (actual / standard) * 100))
})

/** 相对标准价省下的金额与比例；没有省钱时不显示。 */
const savings = computed(() => {
  const standard = props.stats?.total_cost ?? 0
  const actual = props.stats?.total_actual_cost ?? 0
  const amount = standard - actual
  if (standard <= 0 || amount <= 0) return null
  return { amount, percent: Math.round((amount / standard) * 100) }
})
</script>

<style scoped>
.lifetime-tile {
  @apply min-w-0 rounded-xl bg-gray-50 p-3 dark:bg-dark-700/40;
}

.lifetime-icon {
  @apply flex h-7 w-7 shrink-0 items-center justify-center rounded-md;
}

.lifetime-value {
  @apply mt-3 block truncate text-2xl font-semibold tracking-tight tabular-nums text-gray-900 dark:text-dark-50;
}

.lifetime-sub {
  @apply mt-1 truncate text-xs tabular-nums text-gray-500 dark:text-dark-400;
}

/* 实付比例条从 0 伸展到目标宽度 */
.lifetime-bar {
  width: var(--bar-w);
  animation: lifetime-bar 700ms var(--motion-ease) backwards;
}

@keyframes lifetime-bar {
  from {
    width: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .lifetime-bar {
    animation: none;
  }
}
</style>
