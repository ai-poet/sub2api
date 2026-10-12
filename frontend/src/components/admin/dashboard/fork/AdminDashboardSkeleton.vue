<script setup lang="ts">
// fork：管理员仪表盘首屏骨架（移植自 TokenRouter components/common/DashboardSkeleton.vue）。
import { useI18n } from 'vue-i18n'
import SkeletonBlock from './SkeletonBlock.vue'
import DashboardChartSkeleton from './DashboardChartSkeleton.vue'

const { t } = useI18n()
</script>

<template>
  <!-- 首屏骨架为统计卡、筛选控件和图表预留位置。 -->
  <div class="space-y-4" role="status" :aria-label="t('common.loading')" aria-busy="true" data-testid="dashboard-skeleton">
    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4" aria-hidden="true">
      <div v-for="card in 8" :key="card" class="card p-4">
        <div class="flex flex-col gap-2 lg:flex-row">
          <SkeletonBlock :width="36" :height="36" class="shrink-0" />
          <div class="min-w-0 flex-1 space-y-3">
            <SkeletonBlock width="65%" :height="12" />
            <SkeletonBlock width="80%" :height="24" />
            <SkeletonBlock width="55%" :height="12" />
          </div>
        </div>
      </div>
    </div>
    <div class="card flex items-center justify-between gap-2 p-4" aria-hidden="true">
      <SkeletonBlock :width="128" :height="36" />
      <SkeletonBlock :width="112" :height="36" />
    </div>
    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2" aria-hidden="true">
      <div v-for="chart in 2" :key="chart" class="card space-y-4 p-4">
        <SkeletonBlock :width="96" :height="16" />
        <DashboardChartSkeleton :variant="chart === 1 ? 'distribution' : 'plot'" />
      </div>
    </div>
  </div>
</template>
