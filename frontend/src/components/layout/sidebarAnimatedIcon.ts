// fork：侧栏动态图标的懒加载。AnimatedIcon 连同 motion-v 单独分包（vite.config.ts 的
// feature-motion-icons），第一次挂载侧栏图标时才加载，所有实例共用一次加载。
// 加载完成前和失败时显示静态图标；失败后下次挂载再试。侧栏随页面重挂载时直接复用已加载的组件。
import { markRaw, shallowRef, type Component, type ShallowRef } from 'vue'

export const animatedIcon: ShallowRef<Component | null> = shallowRef(null)

let pending: Promise<void> | null = null

export function loadAnimatedIcon(): Promise<void> {
  if (animatedIcon.value) return Promise.resolve()
  if (pending) return pending
  pending = import('@/components/motion-icons/AnimatedIcon.vue')
    .then((module) => {
      animatedIcon.value = markRaw(module.default as Component)
    })
    .catch(() => {
      pending = null
    })
  return pending
}
