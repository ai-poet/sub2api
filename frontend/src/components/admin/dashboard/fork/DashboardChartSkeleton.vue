<template>
  <div class="h-48" :style="{ height }" role="status" :aria-label="t('common.loading')" aria-busy="true">
    <div v-if="variant === 'distribution'" class="flex h-full items-center gap-6" aria-hidden="true">
      <Skeleton variant="circle" :width="128" :height="128" class="shrink-0" />
      <div class="min-w-0 flex-1 space-y-4">
        <Skeleton v-for="row in 4" :key="row" :width="row % 2 ? '85%' : '65%'" :height="12" />
      </div>
    </div>
    <div v-else class="flex h-full flex-col gap-4" aria-hidden="true">
      <div class="flex justify-center gap-4">
        <Skeleton v-for="legend in 3" :key="legend" :width="48" :height="12" />
      </div>
      <Skeleton height="auto" class="min-h-0 flex-1" />
      <div class="flex justify-between">
        <Skeleton v-for="tick in 5" :key="tick" :width="24" :height="8" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：图表加载骨架（移植自 TokenRouter components/common/ChartSkeleton.vue）。
// 与图表 192px 的绘图区同高，数据到达后卡片高度不变。
import { useI18n } from 'vue-i18n'
import Skeleton from '@/components/common/Skeleton.vue'

withDefaults(defineProps<{ variant?: 'plot' | 'distribution'; height?: string }>(), { variant: 'plot', height: undefined })
const { t } = useI18n()
</script>
