import { onBeforeUnmount, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { usePreferredReducedMotion } from '@vueuse/core'

// fork：数字滚动（移植自 TokenRouter composables/useCountUp.ts）。
// 返回一个跟随 source 变化、从当前显示值滚动到新值的数字。
// 动画使用四次缓出，前段快、尾段慢；系统减少动态效果或环境不支持 rAF 时直接显示新值。
// initial 是首次显示的值：传 0 时首屏也从 0 滚到目标值，省略时首屏直接显示目标值。
export function useCountUp(
  source: MaybeRefOrGetter<number>,
  duration: number,
  options: { initial?: number } = {},
) {
  const reducedMotion = usePreferredReducedMotion()
  const display = ref(options.initial ?? toValue(source))
  let frame = 0

  const cancel = () => {
    if (frame) cancelAnimationFrame(frame)
    frame = 0
  }

  const animateTo = (target: number) => {
    cancel()
    const from = display.value
    if (
      from === target ||
      !Number.isFinite(target) ||
      reducedMotion.value === 'reduce' ||
      typeof requestAnimationFrame !== 'function'
    ) {
      display.value = target
      return
    }
    const startedAt = performance.now()
    const tick = (now: number) => {
      const progress = Math.min((now - startedAt) / duration, 1)
      const eased = 1 - Math.pow(1 - progress, 4)
      display.value = from + (target - from) * eased
      if (progress < 1) {
        frame = requestAnimationFrame(tick)
        return
      }
      display.value = target
      frame = 0
    }
    frame = requestAnimationFrame(tick)
  }

  watch(() => toValue(source), animateTo, { immediate: options.initial !== undefined })

  onBeforeUnmount(cancel)

  return display
}
