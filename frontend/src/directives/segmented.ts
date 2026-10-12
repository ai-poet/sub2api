import type { ObjectDirective } from 'vue'

// fork：分段切换的滑动背景（移植自 TokenRouter directives/segmented.ts，样式见 styles/fork-motion.css）。
// 轨道加 class="segmented" 和 v-segmented，选项加 segmented-item，选中项再加 segmented-item-active。
// 点击、键盘和 ARIA 属性由调用方提供，指令不增加可聚焦节点。

interface SegmentedState {
  observer?: ResizeObserver
  items: Set<HTMLElement>
}

const states = new WeakMap<HTMLElement, SegmentedState>()
const properties = ['x', 'y', 'width', 'height'] as const

/** 根据实际按钮位置移动背景，兼容不等宽选项、换行和动态文案。 */
function syncIndicator(element: HTMLElement) {
  const state = states.get(element)
  if (!state) return

  const items = new Set(
    Array.from(element.children).filter(
      (child): child is HTMLElement =>
        child instanceof HTMLElement && child.classList.contains('segmented-item'),
    ),
  )
  for (const item of state.items) {
    if (!items.has(item)) state.observer?.unobserve(item)
  }
  for (const item of items) {
    if (!state.items.has(item)) state.observer?.observe(item)
  }
  state.items = items

  const active = Array.from(items).find(item => item.classList.contains('segmented-item-active'))
  if (!active || !active.offsetWidth || !active.offsetHeight) {
    // 没有选中项或控件被隐藏时，不保留旧位置的背景。
    element.removeAttribute('data-segmented-ready')
    return
  }

  // offset 几何不受祖先弹窗的进入缩放影响，坐标相对于轨道的内边框。
  const values = [active.offsetLeft, active.offsetTop, active.offsetWidth, active.offsetHeight]
  properties.forEach((property, index) => {
    element.style.setProperty(`--segmented-${property}`, `${values[index]}px`)
  })
  element.setAttribute('data-segmented-ready', '')
}

export const vSegmented: ObjectDirective<HTMLElement> = {
  mounted(element) {
    const observer = typeof ResizeObserver === 'undefined'
      ? undefined
      : new ResizeObserver(() => syncIndicator(element))
    states.set(element, { observer, items: new Set() })
    observer?.observe(element)
    syncIndicator(element)
  },
  updated: syncIndicator,
  beforeUnmount(element) {
    states.get(element)?.observer?.disconnect()
    states.delete(element)
    element.removeAttribute('data-segmented-ready')
    properties.forEach(property => element.style.removeProperty(`--segmented-${property}`))
  },
}
