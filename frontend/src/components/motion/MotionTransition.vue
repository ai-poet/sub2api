<template>
  <Transition v-bind="$attrs" @before-enter="restoreEnteringElement" @before-leave="isolateLeavingElement">
    <slot />
  </Transition>
</template>

<script setup lang="ts">
// fork：显隐入口（移植自 TokenRouter components/common/MotionTransition.vue）。
// 属性与事件原样传给 Vue Transition；退出时给旧节点设置 inert，旧节点不再响应操作。
import { isolateLeavingElement, restoreEnteringElement } from '@/utils/leavingElement'

defineOptions({ inheritAttrs: false })
</script>
