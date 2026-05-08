<template>
  <section class="relative w-full overflow-hidden bg-white px-4 pb-16 pt-12 md:px-6 md:pb-20 md:pt-16 dark:bg-[#0f1114]">

    <div class="mx-auto grid w-full max-w-[1380px] min-w-0 gap-12 lg:grid-cols-[minmax(0,1.1fr)_minmax(460px,0.9fr)] lg:items-center">

      <!-- ===== Left: Copy ===== -->
      <div class="relative w-full min-w-0 max-w-[700px]">

        <!-- Category tag pills -->
        <div class="flex flex-wrap gap-2">
          <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.coding') }}</span>
          <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.agent') }}</span>
          <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.tools') }}</span>
        </div>

        <!-- Headline -->
        <h1 class="mt-7 max-w-full text-[clamp(1.95rem,7.8vw,5rem)] font-black leading-[1.06] tracking-[-0.04em] [overflow-wrap:anywhere] [text-wrap:balance] sm:text-[clamp(2.6rem,6vw,5rem)]">
          <span class="block text-[#111] dark:text-white">
            <span class="block sm:inline">{{ t('home.hero.titleLeadPrimary') }}</span>
            <span class="block sm:ml-[0.18em] sm:inline">{{ t('home.hero.titleLeadSecondary') }}</span>
          </span>
          <span class="block text-primary-600 dark:text-primary-400">{{ t('home.hero.titleAccent') }}</span>
          <span class="block text-[#111] dark:text-white">{{ t('home.hero.titleTail') }}</span>
        </h1>

        <!-- Description -->
        <p class="mt-6 max-w-full text-base leading-8 text-gray-500 [overflow-wrap:anywhere] dark:text-white/60 sm:max-w-[38rem] sm:text-lg">
          {{ t('home.hero.description') }}
        </p>

        <!-- CTAs -->
        <div class="mt-8 flex flex-col gap-3 sm:flex-row">
          <router-link
            :to="primaryTo"
            class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full bg-[#111] px-8 text-[15px] font-bold text-white transition hover:-translate-y-[1px] hover:bg-black active:translate-y-0 dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5] sm:w-auto"
          >
            <span>{{ primaryLabel }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
          >
            <span>{{ t('home.viewDocs') }}</span>
            <Icon name="externalLink" size="sm" />
          </a>
          <router-link
            v-else
            to="/login"
            class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
          >
            <span>{{ t('home.login') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>

        <!-- Note -->
        <p class="mt-3 max-w-full text-sm leading-6 text-gray-400 [overflow-wrap:anywhere] dark:text-white/35">
          {{ t('home.hero.primaryNote') }}
        </p>

        <!-- Client advantage strip -->
        <div class="mt-10 grid grid-cols-1 gap-5 border-t border-gray-100 pt-8 dark:border-white/8 sm:grid-cols-3 sm:gap-0 sm:divide-x sm:divide-gray-100 sm:dark:divide-white/8">
          <div class="min-w-0 sm:pr-5">
            <div class="truncate text-[clamp(1.85rem,8vw,2.4rem)] font-black leading-none tracking-tight text-[#111] dark:text-white">
              {{ t('home.hero.stats.setupValue') }}
            </div>
            <div class="mt-2 text-sm leading-5 text-gray-400 dark:text-white/50">{{ t('home.hero.stats.savings') }}</div>
          </div>
          <div class="min-w-0 sm:px-5">
            <div class="truncate text-[clamp(1.85rem,8vw,2.4rem)] font-black leading-none tracking-tight text-[#111] dark:text-white">
              {{ t('home.hero.stats.routesValue') }}
            </div>
            <div class="mt-2 text-sm leading-5 text-gray-400 dark:text-white/50">{{ t('home.hero.stats.models') }}</div>
          </div>
          <div class="min-w-0 sm:pl-5">
            <div class="truncate text-[clamp(1.85rem,8vw,2.4rem)] font-black leading-none tracking-tight text-[#111] dark:text-white">
              {{ t('home.hero.stats.workspaceValue') }}
            </div>
            <div class="mt-2 text-sm leading-5 text-gray-400 dark:text-white/50">{{ t('home.hero.stats.minCost') }}</div>
          </div>
        </div>
      </div>

      <!-- ===== Right: Product Screenshot ===== -->
      <div class="relative hidden lg:block">
        <div class="relative overflow-hidden rounded-2xl border border-gray-200/60 shadow-[0_20px_60px_rgba(0,0,0,0.12)] dark:border-white/10 dark:shadow-[0_20px_60px_rgba(0,0,0,0.4)]">
          <!-- Browser chrome -->
          <div class="flex items-center gap-2 border-b border-gray-100 bg-gray-50/80 px-4 py-2.5 dark:border-white/8 dark:bg-white/[0.03]">
            <div class="flex gap-1.5">
              <span class="h-2.5 w-2.5 rounded-full bg-[#ff5f57]/70"></span>
              <span class="h-2.5 w-2.5 rounded-full bg-[#febc2e]/70"></span>
              <span class="h-2.5 w-2.5 rounded-full bg-[#28c840]/70"></span>
            </div>
            <div class="ml-3 flex flex-1 items-center justify-center rounded-md border border-gray-200/60 bg-white px-3 py-0.5 text-[11px] text-gray-400 dark:border-white/8 dark:bg-white/[0.04] dark:text-white/25">
              cheaprouter.cloud
            </div>
          </div>
          <!-- Screenshot -->
          <div class="relative bg-[#0d0d0f]">
            <img
              src="/product.png"
              alt="CheapRouter Client"
              class="w-full"
              loading="eager"
            />
          </div>
        </div>
        <!-- Floating badge -->
        <div class="absolute -bottom-3 -right-3 rounded-xl border border-primary-200 bg-primary-50 px-4 py-2.5 shadow-lg dark:border-primary-800/50 dark:bg-primary-900/40">
          <div class="text-[10px] font-semibold uppercase tracking-[0.16em] text-primary-600 dark:text-primary-400">{{ t('home.hero.badgeDiscount') }}</div>
          <div class="text-xl font-black tracking-tight text-primary-700 dark:text-primary-300">off</div>
        </div>
      </div>
    </div>

    <!-- ===== Model Name Ticker ===== -->
    <div class="mt-16 border-y border-gray-100 py-6 dark:border-white/8">
      <!-- Row 1: scrolls left -->
      <div class="ticker-row overflow-hidden">
        <div class="ticker-track marquee-left flex gap-8 whitespace-nowrap">
          <span v-for="_ in 4" :key="_" class="ticker-inner flex gap-8">
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">claude-haiku-4-5-20251001</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">Claude Sonnet 4.6</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">gpt-5.4</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">Claude Opus 4.6</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">gpt-5.5</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">gpt-5.3-codex</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
          </span>
        </div>
      </div>
      <!-- Row 2: scrolls right -->
      <div class="ticker-row mt-3 overflow-hidden">
        <div class="ticker-track marquee-right flex gap-8 whitespace-nowrap">
          <span v-for="_ in 4" :key="_" class="ticker-inner flex gap-8">
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">claude-sonnet-4-5-20250929</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">Claude Opus 4.7</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">gpt-5.2</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">claude-opus-4-5-20251101</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">GPT-5.5</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-xs text-gray-400 dark:text-white/30 sm:text-sm">GPT-5.3 Codex</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
          </span>
        </div>
      </div>
    </div>

  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  siteSubtitle: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
}>()

const { t } = useI18n()

const primaryTo = computed(() => (props.isAuthenticated ? props.dashboardPath : '/login'))
const primaryLabel = computed(() => (props.isAuthenticated ? t('home.goToDashboard') : t('home.cta.button')))
</script>

<style scoped>
/* ─── Ticker ──────────────────────────────────────── */
.marquee-left {
  animation: marquee-left 30s linear infinite;
}

.marquee-right {
  animation: marquee-right 30s linear infinite;
}

@keyframes marquee-left {
  0%   { transform: translateX(0); }
  100% { transform: translateX(-25%); }
}

@keyframes marquee-right {
  0%   { transform: translateX(-25%); }
  100% { transform: translateX(0); }
}

/* ─── Reduced Motion ──────────────────────────────── */
@media (prefers-reduced-motion: reduce) {
  .marquee-left,
  .marquee-right {
    animation: none;
  }
}
</style>
