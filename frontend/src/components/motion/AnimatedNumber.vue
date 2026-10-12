<template>
  <span class="tabular-nums">
    <span aria-hidden="true">{{ format(display) }}</span>
    <span class="sr-only">{{ format(value) }}</span>
  </span>
</template>

<script setup lang="ts">
// fork：滚动数字。动画文字对读屏隐藏，读屏只读最终值；减少动态效果时直接显示终值。
// 数值变化时从当前显示值滚到新值；传 from 时首屏也从 from 滚到目标值。
import { COUNT_UP_MS } from '@/constants/motion'
import { useCountUp } from '@/composables/useCountUp'

const props = withDefaults(defineProps<{
  value: number
  duration?: number
  from?: number
  format?: (value: number) => string
}>(), {
  duration: COUNT_UP_MS,
  from: undefined,
  format: (value: number) => Math.round(value).toLocaleString(),
})

const display = useCountUp(() => props.value, props.duration, { initial: props.from })
</script>
