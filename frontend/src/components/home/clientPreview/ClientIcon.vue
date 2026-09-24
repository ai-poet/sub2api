<template>
  <!-- 图形数据是 icons.ts 里的静态常量，不含任何外部输入 -->
  <svg
    :viewBox="shape.viewBox"
    :fill="shape.stroke ? 'none' : 'currentColor'"
    :stroke="shape.stroke ? 'currentColor' : undefined"
    :stroke-width="shape.stroke"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
    v-html="shape.body"
  ></svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CLIENT_ICONS, type ClientIconName, type ClientIconShape } from './icons'

const props = defineProps<{ name: ClientIconName }>()

const shape = computed<ClientIconShape>(() => CLIENT_ICONS[props.name])
</script>
