<template>
  <component :is="animatedIcon" v-if="animatedIcon && iconName" :name="iconName" data-sidebar-icon="animated" />
  <component :is="fallbackComponent" v-else data-sidebar-icon="static" />
</template>

<script setup lang="ts">
// fork：侧栏图标。登记了动态图标的路径在悬停时播放动画（见 sidebarIconMap.ts）；
// 动态图标懒加载，加载完成前和失败时显示原有的静态图标（见 sidebarAnimatedIcon.ts）。
import { computed, onMounted, type Component } from 'vue'
import { resolveSidebarAnimatedIcon } from './sidebarIconMap'
import { animatedIcon, loadAnimatedIcon } from './sidebarAnimatedIcon'

const props = defineProps<{
  path: string
  /** 原有的静态图标组件（NavItem.icon）；未登记或动态图标尚未加载时显示 */
  fallback: unknown
}>()

const iconName = computed(() => resolveSidebarAnimatedIcon(props.path))
const fallbackComponent = computed(() => props.fallback as Component)

onMounted(() => {
  if (iconName.value) void loadAnimatedIcon()
})
</script>
