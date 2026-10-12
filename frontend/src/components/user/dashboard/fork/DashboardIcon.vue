<template>
  <svg
    v-if="extraPath"
    :class="sizeClass"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    :stroke-width="strokeWidth"
  >
    <path stroke-linecap="round" stroke-linejoin="round" :d="extraPath" />
  </svg>
  <Icon v-else :name="name as IconName" :size="size" :stroke-width="strokeWidth" />
</template>

<script setup lang="ts">
// fork：仪表盘用的图标。TokenRouter 的仪表盘用到 layers / trendingDown / minus，
// 本项目的 Icon（上游文件）没有这三个，这里补上（layers 取 Lucide 原图形），其余名字交给 Icon。
// 尺寸与描边和 Icon 一致。
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { DashboardIconName, IconName } from './iconName'

const EXTRA_ICONS = {
  layers:
    'M12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z' +
    'M22 17.65l-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65' +
    'M22 12.65l-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65',
  trendingDown: 'M13 17h8m0 0V9m0 8l-8-8-4 4-6-6',
  minus: 'M5 12h14',
} as const satisfies Record<Exclude<DashboardIconName, IconName>, string>

const SIZE_CLASSES = {
  xs: 'h-3 w-3',
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-6 w-6',
  xl: 'h-8 w-8',
} as const

const props = withDefaults(defineProps<{
  name: DashboardIconName
  size?: keyof typeof SIZE_CLASSES
  strokeWidth?: number
}>(), {
  size: 'md',
  strokeWidth: 1.5,
})

const extraPath = computed(() =>
  props.name in EXTRA_ICONS ? EXTRA_ICONS[props.name as keyof typeof EXTRA_ICONS] : null,
)
const sizeClass = computed(() => SIZE_CLASSES[props.size])
</script>
