<template>
  <!-- fork：首页公开统计，放在首屏客户端演示之后的第二屏。后台开关关闭时整块不渲染、不发请求；
       数据不可用（聚合未开启或落后）时也不显示。 -->
  <section
    v-if="enabled && !unavailable"
    ref="root"
    class="px-4 pt-16 md:px-6 md:pt-24"
    :class="{ 'live-stats-visible': started }"
    data-test="home-live-stats"
    :aria-label="t('home.liveStats.label')"
  >
    <div class="mx-auto max-w-[1200px]">
      <div class="text-center">
        <p class="inline-flex items-center gap-2 text-[11px] uppercase tracking-[0.24em] text-gray-500 dark:text-white/42">
          <span class="relative flex h-2 w-2" aria-hidden="true">
            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-60 motion-reduce:animate-none"></span>
            <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
          </span>
          {{ t('home.liveStats.overline') }}
        </p>
        <h2 class="mt-3 text-3xl font-semibold leading-[1.12] tracking-[-0.04em] text-gray-900 dark:text-white md:text-4xl [text-wrap:balance]">
          {{ t('home.liveStats.title') }}
        </h2>
        <p class="mx-auto mt-3 max-w-[40rem] text-base leading-7 text-gray-600 dark:text-white/68">
          {{ t('home.liveStats.description') }}
        </p>
      </div>

      <div class="mt-10 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div
          v-for="(card, index) in cards"
          :key="card.key"
          class="live-stat-card"
          :style="{ '--card-i': index }"
          :data-stat="card.key"
        >
          <div class="text-sm font-medium text-gray-500 dark:text-white/55">
            {{ t(`home.liveStats.${card.key}`) }}
          </div>
          <div class="mt-3 min-h-[1.1em] text-4xl font-extrabold leading-none tracking-[-0.04em] text-gray-900 dark:text-white md:text-5xl">
            <AnimatedNumber
              v-if="stats"
              :value="started ? card.value : 0"
              :from="0"
              :duration="HOME_COUNT_UP_MS"
              :format="card.format"
            /><span v-else aria-hidden="true">—</span><span v-if="stats && card.plus" class="text-primary-500" aria-hidden="true">+</span>
          </div>
        </div>
      </div>
      <p v-if="asOfLabel" class="mt-4 text-center text-xs text-gray-400 dark:text-white/35">{{ asOfLabel }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
// fork：首页「总 token 数」滚动计数（参考 TokenRouter HomeView 的首页计数）。
// 进入视口后从 0 滚到目标值（3.2 秒，四次方缓出）；单位锁定为目标值的单位，1.2B 从 0.0B 滚起。
// 减少动态效果时直接显示终值；读屏只读最终值（AnimatedNumber）。
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import { getPublicStats, type PublicStats } from '@/api/publicStats'
import { useAppStore } from '@/stores'
import { HOME_COUNT_UP_MS } from '@/constants/motion'
import { compactFormatterFor } from '@/utils/compactNumber'
import { ForkFeatureFlags } from '@/utils/forkFeatureFlags'
import { resolveFeatureFlag } from '@/utils/featureFlags'

const { t } = useI18n()
const appStore = useAppStore()

const enabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, ForkFeatureFlags.publicStats))
const stats = ref<PublicStats | null>(null)
const unavailable = ref(false)
const started = ref(false)
const root = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null
let loaded = false

const cards = computed(() => {
  const data = stats.value
  const total = data?.total_tokens ?? 0
  const today = data?.today_tokens ?? 0
  const requests = data?.total_requests ?? 0
  return [
    { key: 'total', value: total, format: compactFormatterFor(total), plus: true },
    { key: 'today', value: today, format: compactFormatterFor(today), plus: false },
    { key: 'requests', value: requests, format: compactFormatterFor(requests), plus: true },
  ]
})

const asOfLabel = computed(() => {
  if (!stats.value?.as_of) return ''
  const date = new Date(stats.value.as_of)
  return Number.isNaN(date.getTime()) ? '' : t('home.liveStats.asOf', { time: date.toLocaleString() })
})

async function load() {
  if (loaded || !enabled.value) return
  loaded = true
  const result = await getPublicStats()
  if (!result.enabled || !result.available) {
    unavailable.value = true
    return
  }
  stats.value = result
}

function startWhenVisible(element: HTMLElement | null) {
  if (!element || started.value) return
  if (typeof IntersectionObserver === 'undefined') {
    started.value = true
    return
  }
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some(entry => entry.isIntersecting)) {
        started.value = true
        observer?.disconnect()
        observer = null
      }
    },
    { threshold: 0.35 },
  )
  observer.observe(element)
}

// 开关可能在公开设置加载完成后才变成 true
watch(enabled, (value) => {
  if (value) void load()
})

watch(root, (element) => startWhenVisible(element))

onMounted(() => {
  void load()
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
})
</script>

<style scoped>
.live-stat-card {
  @apply rounded-[26px] border border-black/10 bg-white/70 px-6 py-7 text-center shadow-[0_20px_70px_rgba(15,15,15,0.05)] backdrop-blur dark:border-white/10 dark:bg-white/5;
}

/* 第二屏在首屏之下：卡片在进入视口时才依次升起，和数字滚动同时开始，而不是在屏幕外提前播完 */
.live-stats-visible .live-stat-card {
  animation: live-stat-rise 560ms var(--motion-ease) backwards;
  animation-delay: calc(var(--card-i, 0) * 90ms);
}

@keyframes live-stat-rise {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .live-stats-visible .live-stat-card {
    animation: none;
  }
}
</style>
