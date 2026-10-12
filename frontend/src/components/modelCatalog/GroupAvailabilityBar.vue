<template>
  <!-- 分组近 24 小时的可用率：每格一小时，颜色按可用率分档，右侧是整体可用率。 -->
  <div class="flex min-w-0 items-center gap-2" data-testid="group-availability-bar">
    <div
      class="grid h-5 min-w-0 flex-1 items-center gap-[2px]"
      :style="{ gridTemplateColumns: `repeat(${buckets.length}, minmax(0, 1fr))` }"
      role="img"
      :aria-label="ariaLabel"
    >
      <span
        v-for="(bucket, index) in buckets"
        :key="`${bucket.bucket_start}-${index}`"
        class="h-full rounded-[3px] transition-transform duration-fast ease-standard hover:scale-y-110"
        :class="bucketClass(bucket)"
        :title="bucketTitle(bucket)"
        :data-availability="bucket.total_count > 0 ? bucket.availability : 'none'"
      />
    </div>
    <span class="w-[4.25rem] shrink-0 text-right text-xs font-semibold tabular-nums text-gray-700 dark:text-gray-200">
      {{ rateLabel }}
    </span>
  </div>
</template>

<script setup lang="ts">
// fork：分组可用率条（参考 TokenRouter components/marketplace/GroupAvailabilityBar.vue）。
// 数据来自 GET /group-status/history 的分桶，可用率为 0-100。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GroupStatusHistoryBucket } from '@/types'

const props = defineProps<{
  buckets: GroupStatusHistoryBucket[]
  availability: number
  totalCount: number
}>()

const { t } = useI18n()

const rateLabel = computed(() =>
  props.totalCount > 0 ? `${props.availability.toFixed(2)}%` : t('modelCatalog.availability.noData'),
)

const ariaLabel = computed(() => t('modelCatalog.availability.label', { rate: rateLabel.value }))

function bucketClass(bucket: GroupStatusHistoryBucket): string {
  if (!bucket.total_count) return 'bg-gray-200 dark:bg-dark-700'
  if (bucket.availability >= 99.5) return 'bg-emerald-500'
  if (bucket.availability >= 98) return 'bg-lime-500'
  if (bucket.availability >= 90) return 'bg-amber-400'
  return 'bg-rose-400'
}

function bucketTitle(bucket: GroupStatusHistoryBucket): string {
  const start = new Date(bucket.bucket_start)
  const time = Number.isNaN(start.getTime())
    ? ''
    : start.toLocaleString(undefined, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  if (!bucket.total_count) return `${time} · ${t('modelCatalog.availability.noData')}`
  return `${time} · ${bucket.availability.toFixed(1)}%`
}
</script>
