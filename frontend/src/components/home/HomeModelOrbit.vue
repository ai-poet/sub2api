<template>
  <section
    data-test="home-model-orbit"
    class="relative overflow-hidden bg-[#111214] text-white dark:border-y dark:border-white/5 dark:bg-[#15171b]"
  >
    <div class="mx-auto grid max-w-[1200px] items-center gap-14 px-4 py-20 md:grid-cols-2 md:px-6 md:py-28">
      <div>
        <p class="text-[11px] font-bold uppercase tracking-[0.24em] text-primary-300">
          {{ t('home.landing.models.overline') }}
        </p>
        <h2 class="mt-5 text-[clamp(2.3rem,5.4vw,4.25rem)] font-black leading-[1.04] tracking-[-0.045em] [text-wrap:balance]">
          <span class="block">{{ t('home.landing.models.titleLead') }}</span>
          <span class="block text-white/55">{{ t('home.landing.models.titleTail') }}</span>
        </h2>
        <p class="mt-6 max-w-[30rem] text-base leading-7 text-white/60">
          {{ mode === 'client' ? t('home.landing.models.clientSubtitle') : t('home.landing.models.apiSubtitle') }}
        </p>
        <a
          href="#pricing"
          data-test="orbit-pricing-link"
          class="mt-9 inline-flex h-12 items-center gap-2 rounded-xl bg-white px-5 text-sm font-bold text-gray-900 transition hover:bg-gray-100"
          @click.prevent="goToSection('pricing')"
        >
          {{ t('home.landing.models.cta') }}
          <Icon name="arrowRight" size="sm" />
        </a>
      </div>

      <div class="relative mx-auto aspect-square w-full max-w-[420px]" aria-hidden="true">
        <span class="absolute inset-0 rounded-full border border-dashed border-white/15"></span>
        <span class="absolute inset-[22%] rounded-full border border-dashed border-white/10"></span>

        <div class="orbit-spin absolute inset-0">
          <div
            v-for="node in nodes"
            :key="node.platform"
            class="absolute -translate-x-1/2 -translate-y-1/2"
            :style="{ left: node.left, top: node.top }"
          >
            <div class="orbit-counter-spin flex flex-col items-center gap-2">
              <span class="flex h-14 w-14 items-center justify-center rounded-full bg-white text-gray-900 shadow-[0_10px_30px_rgba(0,0,0,0.35)]">
                <PlatformIcon :platform="node.platform" size="lg" />
              </span>
              <span class="whitespace-nowrap text-xs font-semibold text-white/70">{{ node.label }}</span>
            </div>
          </div>
          <!-- 内圈：国产模型 -->
          <div
            v-for="node in innerNodes"
            :key="node.platform"
            class="absolute -translate-x-1/2 -translate-y-1/2"
            :style="{ left: node.left, top: node.top }"
          >
            <div class="orbit-counter-spin flex flex-col items-center gap-1.5">
              <span class="flex h-10 w-10 items-center justify-center rounded-full bg-white/90 text-gray-900 shadow-[0_8px_24px_rgba(0,0,0,0.3)]">
                <PlatformIcon :platform="node.platform" size="md" />
              </span>
              <span class="whitespace-nowrap text-[11px] font-semibold text-white/60">{{ node.label }}</span>
            </div>
          </div>
        </div>

        <div class="absolute inset-[35%] flex items-center justify-center">
          <span class="absolute inset-0 rotate-6 rounded-[28px] bg-primary-500"></span>
          <HomeBrandMark
            :site-name="siteName"
            :site-logo="siteLogo"
            class="relative h-[72%] w-[72%] -rotate-6 rounded-[22px] text-4xl"
          />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeBrandMark from '@/components/home/HomeBrandMark.vue'
import { useHomeSectionNav } from '@/components/home/useHomeSectionNav'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'

const props = defineProps<{
  mode: 'client' | 'api'
  siteName: string
  siteLogo?: string
}>()

const { t } = useI18n()
const { goToSection } = useHomeSectionNav()

// 圆周上的点：角度从正上方开始顺时针，半径是容器边长的百分比。
function onCircle(angleDeg: number, radiusPct: number) {
  const rad = (angleDeg * Math.PI) / 180
  return {
    left: `${50 + radiusPct * Math.sin(rad)}%`,
    top: `${50 - radiusPct * Math.cos(rad)}%`,
  }
}

const nodes = computed(() => {
  const labels: Array<[GroupPlatform, string]> =
    props.mode === 'client'
      ? [['anthropic', 'Claude Code'], ['openai', 'Codex'], ['grok', 'Grok']]
      : [['anthropic', 'Claude'], ['openai', 'GPT'], ['grok', 'Grok']]
  return labels.map(([platform, label], index) => ({ platform, label, ...onCircle(index * 120, 50) }))
})

const INNER_MODELS: Array<[GroupPlatform, string]> = [['deepseek', 'DeepSeek'], ['zhipu', 'GLM'], ['kimi', 'Kimi']]
const innerNodes = INNER_MODELS.map(([platform, label], index) => ({ platform, label, ...onCircle(60 + index * 120, 28) }))
</script>

<style scoped>
.orbit-spin {
  animation: orbit-spin 60s linear infinite;
}

.orbit-counter-spin {
  animation: orbit-spin 60s linear infinite reverse;
}

@keyframes orbit-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .orbit-spin,
  .orbit-counter-spin {
    animation: none;
  }
}
</style>
