<template>
  <Transition
    name="collapse"
    :appear="appear"
    :css="animate && !skipAnimation && !reducedMotion"
    @before-enter="beforeEnter"
    @before-leave="beforeLeave"
    @after-enter="finishEnter"
    @after-leave="finishLeave"
    @enter-cancelled="moving = false"
    @leave-cancelled="moving = false"
  >
    <div
      v-show="open"
      class="motion-collapse"
      :class="{ 'motion-collapse-moving': moving }"
      :inert="!open || undefined"
      :aria-hidden="!open || undefined"
      @form-field-reveal.capture="revealImmediately"
    >
      <div class="motion-collapse-inner">
        <RetainedContent v-if="renderContent" :freeze="unmountOnHide && !open">
          <slot />
        </RetainedContent>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
// fork：纵向展开（移植自 TokenRouter components/common/Collapse.vue）。
// grid-template-rows 从 0fr 过渡到 1fr，高度自动适应内容；过渡期间裁剪溢出，
// 收起的内容设置 inert。unmountOnHide 让按需挂载的内容在退出动画结束后再卸载。
import { computed, nextTick, ref, watch } from 'vue'
import { usePreferredReducedMotion } from '@vueuse/core'
import RetainedContent from './RetainedContent'
import { isolateLeavingElement, restoreEnteringElement } from '@/utils/leavingElement'

const props = withDefaults(defineProps<{
  open: boolean
  unmountOnHide?: boolean
  animate?: boolean
  appear?: boolean
}>(), { unmountOnHide: false, animate: true, appear: false })

const emit = defineEmits<{
  'after-enter': []
  'after-leave': []
}>()
const preference = usePreferredReducedMotion()
const reducedMotion = computed(() => preference.value === 'reduce')
const moving = ref(false)
const skipAnimation = ref(false)
const renderContent = ref(!props.unmountOnHide || props.open)

watch(() => [props.open, props.unmountOnHide], () => {
  // 先恢复内容再计算展开高度，退出完成前保留组件及其表单状态。
  if (props.open || !props.unmountOnHide) renderContent.value = true
}, { flush: 'sync' })

function beforeEnter(element: Element) {
  restoreEnteringElement(element)
  moving.value = true
}

function beforeLeave(element: Element) {
  // 按需卸载的必填字段退出校验；长期保留的折叠表单继续支持跨区定位。
  if (props.unmountOnHide) isolateLeavingElement(element)
  moving.value = true
}

function finishEnter() {
  moving.value = false
  if (props.open) emit('after-enter')
}

function finishLeave() {
  moving.value = false
  if (props.open) return
  if (props.unmountOnHide) renderContent.value = false
  emit('after-leave')
}

function revealImmediately() {
  // 校验需要在 nextTick 后定位字段，此次展开不等待动画。
  skipAnimation.value = true
  void nextTick(() => { skipAnimation.value = false })
}
</script>
