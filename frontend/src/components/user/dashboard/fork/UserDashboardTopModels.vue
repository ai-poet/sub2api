<template>
  <section class="card p-5" data-testid="top-models">
    <div class="mb-4 flex items-center justify-between gap-2">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.fork.topModels.title') }}</h3>
      <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.fork.topModels.byTokens') }}</span>
    </div>

    <ol v-if="rows.length" class="space-y-3">
      <li
        v-for="(row, index) in rows"
        :key="`${version}-${row.model}`"
        class="top-model-row"
        :style="{ '--i': index, '--bar-w': `${row.percent}%` }"
      >
        <div class="flex items-center gap-2">
          <span class="w-4 shrink-0 text-xs font-semibold tabular-nums text-gray-400">{{ index + 1 }}</span>
          <ModelIcon :model="row.model" size="16px" />
          <span class="min-w-0 flex-1 truncate text-sm text-gray-800 dark:text-gray-100" :title="row.model">{{ row.model }}</span>
          <span class="shrink-0 text-sm font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatCompact(row.tokens) }}</span>
        </div>
        <div class="ml-6 mt-1.5 flex items-center gap-2">
          <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
            <div class="top-model-bar h-full rounded-full bg-primary-500 dark:bg-primary-400"></div>
          </div>
          <span class="w-9 shrink-0 text-right text-[11px] tabular-nums text-gray-500 dark:text-gray-400">{{ row.percent.toFixed(0) }}%</span>
        </div>
      </li>
    </ol>
    <p v-else class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.fork.topModels.empty') }}</p>
  </section>
</template>

<script setup lang="ts">
// fork：用量最多的模型（参考 TokenRouter UserDashboardTopModels）。复用仪表盘已经取到的模型统计，
// 不另发请求。占比条从 0 依次伸展、越往后越淡；数据变化时按版本号重挂载行，动画重放。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import type { ModelStat } from '@/types'
import { formatCompact } from '@/utils/compactNumber'

const props = withDefaults(defineProps<{ models: ModelStat[]; limit?: number }>(), { limit: 5 })

const { t } = useI18n()
const version = ref(0)

const rows = computed(() => {
  const sorted = [...(props.models ?? [])].sort((a, b) => b.total_tokens - a.total_tokens)
  const total = sorted.reduce((sum, model) => sum + model.total_tokens, 0)
  return sorted.slice(0, props.limit).map(model => ({
    model: model.model,
    tokens: model.total_tokens,
    percent: total > 0 ? (model.total_tokens / total) * 100 : 0,
  }))
})

watch(() => props.models, () => {
  version.value++
})
</script>

<style scoped>
.top-model-row {
  opacity: calc(1 - var(--i, 0) * 0.12);
}

.top-model-bar {
  width: var(--bar-w);
  animation: top-model-bar 600ms var(--motion-ease) backwards;
  animation-delay: calc(var(--i, 0) * 60ms);
}

@keyframes top-model-bar {
  from {
    width: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .top-model-bar {
    animation: none;
  }
}
</style>
