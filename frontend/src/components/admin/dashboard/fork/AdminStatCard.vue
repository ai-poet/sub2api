<template>
  <!-- 窄屏图标在上、文字占满卡宽；宽屏图标与文字横排 -->
  <div class="card p-4" :data-testid="testid">
    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="shrink-0 self-start rounded-lg p-2 lg:self-center" :class="tone.bg">
        <Icon :name="icon" size="md" :class="tone.text" :stroke-width="2" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ label }}</p>
        <slot />
        <div v-if="$slots.sub" class="mt-0.5 text-xs">
          <slot name="sub" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的统计卡（版式移植自 TokenRouter views/admin/DashboardView.vue）。
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { IconName } from '@/components/user/dashboard/fork/iconName'

type AdminStatTone = 'blue' | 'purple' | 'green' | 'emerald' | 'amber' | 'indigo' | 'violet' | 'rose'

// 完整类名写死在表里，Tailwind 才能扫描到
const TONES: Record<AdminStatTone, { bg: string; text: string }> = {
  blue: { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-600 dark:text-blue-400' },
  purple: { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-600 dark:text-purple-400' },
  green: { bg: 'bg-green-100 dark:bg-green-900/30', text: 'text-green-600 dark:text-green-400' },
  emerald: { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-600 dark:text-emerald-400' },
  amber: { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-600 dark:text-amber-400' },
  indigo: { bg: 'bg-indigo-100 dark:bg-indigo-900/30', text: 'text-indigo-600 dark:text-indigo-400' },
  violet: { bg: 'bg-violet-100 dark:bg-violet-900/30', text: 'text-violet-600 dark:text-violet-400' },
  rose: { bg: 'bg-rose-100 dark:bg-rose-900/30', text: 'text-rose-600 dark:text-rose-400' },
}

const props = defineProps<{
  icon: IconName
  tone: AdminStatTone
  label: string
  testid?: string
}>()

const tone = computed(() => TONES[props.tone])
</script>
