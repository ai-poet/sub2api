<template>
  <span
    class="inline-flex shrink-0 items-center justify-center overflow-hidden font-black leading-none"
    :class="siteLogo ? '' : toneClass"
    aria-hidden="true"
  >
    <img v-if="siteLogo" :src="siteLogo" alt="" class="h-full w-full object-contain" />
    <span v-else>{{ initial }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

// 站点标志：配置了 site_logo 就用它，否则用站点名首字母做一个方块字标。
// 尺寸、圆角和字号由调用方通过 class 给出。
const props = withDefaults(defineProps<{
  siteName: string
  siteLogo?: string
  /** 没有 logo 时字标的配色：dark 用于浅色底，light 用于彩色或深色底 */
  tone?: 'dark' | 'light'
}>(), {
  siteLogo: '',
  tone: 'dark',
})

const toneClass = computed(() =>
  props.tone === 'light' ? 'bg-white text-gray-900' : 'bg-gray-900 text-white dark:bg-white dark:text-gray-900',
)
const initial = computed(() => props.siteName.trim().charAt(0).toUpperCase() || 'C')
</script>
