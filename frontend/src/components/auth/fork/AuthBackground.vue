<template>
  <div
    class="pointer-events-none absolute inset-0 z-0 select-none overflow-hidden transition-opacity duration-700"
    :class="loaded ? 'opacity-100' : 'opacity-0'"
    aria-hidden="true"
    data-testid="auth-background"
  >
    <div
      ref="container"
      class="absolute left-1/2 top-1/2 h-[2160px] w-[3840px] -translate-x-1/2 -translate-y-1/2"
    ></div>
  </div>
</template>

<script setup lang="ts">
// fork：登录 / 注册等认证页的背景动画（移植自 TokenRouter components/auth/AuthBackground.vue）。
// 浅灰点阵图被几组移动的形状遮罩着逐块显现；lottie-web 和两份动画数据都按需加载，
// vite.config.ts 把它们分进 feature-auth-lottie / feature-auth-bg-light / feature-auth-bg-dark 三个分包，
// 不进其他页面的首屏，并且只下载当前主题用到的那份动画。
// 与 TokenRouter 的差别：深色状态来自 useIsDark；分包下载失败时只是不显示背景，不留下未处理的异常。
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { AnimationItem, LottiePlayer } from 'lottie-web'
import { useIsDark } from '@/composables/useIsDark'

const isDark = useIsDark()

const container = ref<HTMLDivElement>()
const loaded = ref(false)

let player: LottiePlayer | null = null
let animation: AnimationItem | null = null
let loadToken = 0

const prefersReducedMotion = typeof window.matchMedia === 'function'
  && window.matchMedia('(prefers-reduced-motion: reduce)').matches

async function loadAnimation(dark: boolean) {
  const token = ++loadToken

  const [lottieModule, dataModule] = await Promise.all([
    player ? Promise.resolve(null) : import('lottie-web'),
    dark
      ? import('@/assets/lottie/auth-bg-dark.json')
      : import('@/assets/lottie/auth-bg-light.json'),
  ])

  // 主题在加载期间又被切换，或组件已卸载时放弃本次结果
  if (token !== loadToken || !container.value) return

  if (lottieModule) {
    player = lottieModule.default
  }

  animation?.destroy()
  animation = player!.loadAnimation({
    container: container.value,
    renderer: 'svg',
    loop: true,
    autoplay: !prefersReducedMotion,
    animationData: dataModule.default,
    rendererSettings: { preserveAspectRatio: 'xMidYMid slice' },
  })

  if (prefersReducedMotion) {
    animation.goToAndStop(0, true)
  }

  loaded.value = true
}

// 分包下载失败（离线、部署切换）时不显示背景动画，页面其余部分照常使用。
const showAnimation = (dark: boolean) => {
  loadAnimation(dark).catch((error) => console.warn('Failed to load auth background animation:', error))
}

onMounted(() => {
  showAnimation(isDark.value)
})

watch(isDark, (dark) => {
  showAnimation(dark)
})

onBeforeUnmount(() => {
  loadToken++
  animation?.destroy()
  animation = null
})
</script>
