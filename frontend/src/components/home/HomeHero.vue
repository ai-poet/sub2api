<template>
  <section class="relative overflow-hidden bg-white px-4 pb-16 pt-12 md:px-6 md:pb-20 md:pt-16 dark:bg-[#0f1114]">

    <div class="mx-auto grid max-w-[1380px] gap-12 lg:grid-cols-[minmax(0,1.1fr)_minmax(460px,0.9fr)] lg:items-center">

      <!-- ===== Left: Copy ===== -->
      <div class="relative max-w-[700px]">

        <!-- Category tag pills -->
        <div class="flex flex-wrap gap-2">
          <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.coding') }}</span>
          <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.agent') }}</span>
          <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.tools') }}</span>
        </div>

        <!-- Headline -->
        <h1 class="mt-7 text-[clamp(2.6rem,6vw,5rem)] font-black leading-[1.05] tracking-[-0.04em] [text-wrap:balance]">
          <span class="block text-[#111] dark:text-white">{{ t('home.hero.titleLead') }}</span>
          <span class="block text-primary-600 dark:text-primary-400">{{ t('home.hero.titleAccent') }}</span>
          <span class="block text-[#111] dark:text-white">{{ t('home.hero.titleTail') }}</span>
        </h1>

        <!-- Description -->
        <p class="mt-6 max-w-[38rem] text-lg leading-8 text-gray-500 dark:text-white/60">
          {{ t('home.hero.description') }}
        </p>

        <!-- CTAs -->
        <div class="mt-8 flex flex-col gap-3 sm:flex-row">
          <router-link
            :to="primaryTo"
            class="inline-flex h-14 items-center justify-center gap-2 rounded-full bg-[#111] px-8 text-[15px] font-bold text-white transition hover:-translate-y-[1px] hover:bg-black active:translate-y-0 dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5]"
          >
            <span>{{ primaryLabel }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex h-14 items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
          >
            <span>{{ t('home.viewDocs') }}</span>
            <Icon name="externalLink" size="sm" />
          </a>
          <router-link
            v-else
            to="/login"
            class="inline-flex h-14 items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
          >
            <span>{{ t('home.login') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>

        <!-- Note -->
        <p class="mt-3 text-sm text-gray-400 dark:text-white/35">
          {{ t('home.hero.primaryNote') }}
        </p>

        <!-- Stats strip -->
        <div class="mt-10 grid grid-cols-3 divide-x divide-gray-100 border-t border-gray-100 pt-8 dark:divide-white/8 dark:border-white/8">
          <div class="pr-5">
            <div class="text-[2.4rem] font-black leading-none tracking-tight text-[#111] dark:text-white">
              97<span class="text-primary-600 dark:text-primary-400">%</span>
            </div>
            <div class="mt-2 text-sm leading-5 text-gray-400 dark:text-white/50">{{ t('home.hero.stats.savings') }}</div>
          </div>
          <div class="px-5">
            <div class="text-[2.4rem] font-black leading-none tracking-tight text-[#111] dark:text-white">
              14<span class="text-primary-600 dark:text-primary-400">+</span>
            </div>
            <div class="mt-2 text-sm leading-5 text-gray-400 dark:text-white/50">{{ t('home.hero.stats.models') }}</div>
          </div>
          <div class="pl-5">
            <div class="text-[2.4rem] font-black leading-none tracking-tight text-[#111] dark:text-white">
              ¥<span>0</span>
            </div>
            <div class="mt-2 text-sm leading-5 text-gray-400 dark:text-white/50">{{ t('home.hero.stats.minCost') }}</div>
          </div>
        </div>
      </div>

      <!-- ===== Right: Scrolling Price Cards ===== -->
      <div class="relative hidden lg:block">
        <div
          class="card-columns-container"
          style="height: 560px; overflow: hidden; -webkit-mask-image: linear-gradient(to bottom, transparent 0%, black 10%, black 90%, transparent 100%); mask-image: linear-gradient(to bottom, transparent 0%, black 10%, black 90%, transparent 100%);"
        >
          <div class="flex gap-4">
            <!-- Column 1: scrolls UP -->
            <div class="card-col flex w-[220px] flex-none flex-col gap-4 scroll-up">
              <template v-for="_ in 2" :key="_">
                <div v-for="card in col1Cards" :key="card.id + _" class="price-card">
                  <div class="card-model-name">{{ card.name }}</div>
                  <div class="card-model-id">{{ card.id }}</div>
                  <div class="card-price-row">
                    <span class="card-price-teal">{{ card.price }}</span>
                    <span class="card-price-strike">{{ card.original }}</span>
                    <span class="card-savings-badge">{{ card.savings }}</span>
                  </div>
                  <div class="card-per-token">{{ t('home.hero.perToken') }}</div>
                </div>
              </template>
            </div>

            <!-- Column 2: scrolls DOWN, offset start -->
            <div class="card-col flex w-[220px] flex-none flex-col gap-4 scroll-down" style="margin-top: -160px;">
              <template v-for="_ in 2" :key="_">
                <div v-for="card in col2Cards" :key="card.id + _" class="price-card">
                  <div class="card-model-name">{{ card.name }}</div>
                  <div class="card-model-id">{{ card.id }}</div>
                  <div class="card-price-row">
                    <span class="card-price-teal">{{ card.price }}</span>
                    <span class="card-price-strike">{{ card.original }}</span>
                    <span class="card-savings-badge">{{ card.savings }}</span>
                  </div>
                  <div class="card-per-token">{{ t('home.hero.perToken') }}</div>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== Model Name Ticker ===== -->
    <div class="mt-16 border-y border-gray-100 py-6 dark:border-white/8">
      <!-- Row 1: scrolls left -->
      <div class="ticker-row overflow-hidden">
        <div class="ticker-track marquee-left flex gap-8 whitespace-nowrap">
          <span v-for="_ in 2" :key="_" class="ticker-inner flex gap-8">
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">claude-haiku-4-5-20251001</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">Claude Sonnet 4.6</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">gpt-5.4</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">Claude Opus 4.6</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">gpt-5.3-codex</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
          </span>
        </div>
      </div>
      <!-- Row 2: scrolls right -->
      <div class="ticker-row mt-3 overflow-hidden">
        <div class="ticker-track marquee-right flex gap-8 whitespace-nowrap">
          <span v-for="_ in 2" :key="_" class="ticker-inner flex gap-8">
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">claude-sonnet-4-5-20250929</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">Claude Opus 4.7</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">gpt-5.2</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">claude-opus-4-5-20251101</span>
            <span class="ticker-sep font-mono text-sm text-gray-300 dark:text-white/20">·</span>
            <span class="ticker-item font-mono text-sm text-gray-400 dark:text-white/30">GPT-5.3 Codex</span>
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

const { t, locale } = useI18n()

const isChinese = computed(() => locale.value === 'zh')

const primaryTo = computed(() => (props.isAuthenticated ? props.dashboardPath : '/register'))
const primaryLabel = computed(() => (props.isAuthenticated ? t('home.goToDashboard') : t('home.cta.button')))

// priceCny: ¥/M tokens (GLM group, 0.25× × ¥0.70/$1)  priceUsd: actual $/M (÷7)
const rawCol1 = [
  { name: 'Claude Haiku 4.5',  id: 'claude-haiku-4-5-20251001',  priceCny: '¥0.175', priceUsd: '$0.025', original: '$1.00', savings: '-97%' },
  { name: 'Claude Sonnet 4.6', id: 'claude-sonnet-4-6',           priceCny: '¥0.525', priceUsd: '$0.075', original: '$3.00', savings: '-97%' },
  { name: 'Claude Opus 4.6',   id: 'claude-opus-4-6',             priceCny: '¥0.875', priceUsd: '$0.125', original: '$5.00', savings: '-97%' },
  { name: 'GPT-5.4',           id: 'gpt-5.4',                     priceCny: '¥1.400', priceUsd: '$0.200', original: '$2.50', savings: '-92%' },
  { name: 'GPT-5.3 Codex',     id: 'gpt-5.3-codex',               priceCny: '¥0.980', priceUsd: '$0.140', original: '$1.75', savings: '-92%' },
]

const rawCol2 = [
  { name: 'Claude Sonnet 4.5', id: 'claude-sonnet-4-5-20250929',  priceCny: '¥0.525', priceUsd: '$0.075', original: '$3.00', savings: '-97%' },
  { name: 'Claude Opus 4.5',   id: 'claude-opus-4-5-20251101',    priceCny: '¥0.875', priceUsd: '$0.125', original: '$5.00', savings: '-97%' },
  { name: 'Claude Opus 4.7',   id: 'claude-opus-4-7',             priceCny: '¥0.875', priceUsd: '$0.125', original: '$5.00', savings: '-97%' },
  { name: 'GPT-5.2',           id: 'gpt-5.2',                     priceCny: '¥0.980', priceUsd: '$0.140', original: '$1.75', savings: '-92%' },
]

const col1Cards = computed(() =>
  rawCol1.map(c => ({ ...c, price: isChinese.value ? c.priceCny : c.priceUsd })),
)
const col2Cards = computed(() =>
  rawCol2.map(c => ({ ...c, price: isChinese.value ? c.priceCny : c.priceUsd })),
)
</script>

<style scoped>
/* ─── Price Cards ─────────────────────────────────── */
.price-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 4px 16px rgba(0,0,0,0.06);
  padding: 14px 16px;
  width: 220px;
  flex-shrink: 0;
}

:global(.dark) .price-card {
  background: rgba(255,255,255,0.04);
  border-color: rgba(255,255,255,0.1);
}

.card-model-name {
  font-size: 14px;
  font-weight: 600;
  color: #111;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:global(.dark) .card-model-name {
  color: rgba(255,255,255,0.9);
}

.card-model-id {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-price-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 10px;
}

.card-price-teal {
  font-size: 20px;
  font-weight: 700;
  color: #0d9488;
  letter-spacing: -0.02em;
}

.card-price-strike {
  font-size: 12px;
  color: #9ca3af;
  text-decoration: line-through;
}

.card-savings-badge {
  font-size: 11px;
  font-weight: 700;
  color: #16a34a;
  background: #dcfce7;
  border-radius: 6px;
  padding: 1px 6px;
}

.card-per-token {
  font-size: 10px;
  color: #d1d5db;
  margin-top: 4px;
}

/* ─── Scroll Animations ───────────────────────────── */
.scroll-up {
  animation: scroll-up 22s linear infinite;
}

.scroll-down {
  animation: scroll-down 22s linear infinite;
}

@keyframes scroll-up {
  0%   { transform: translateY(0); }
  100% { transform: translateY(-50%); }
}

@keyframes scroll-down {
  0%   { transform: translateY(-50%); }
  100% { transform: translateY(0); }
}

/* ─── Ticker ──────────────────────────────────────── */
.marquee-left {
  animation: marquee-left 30s linear infinite;
}

.marquee-right {
  animation: marquee-right 30s linear infinite;
}

@keyframes marquee-left {
  0%   { transform: translateX(0); }
  100% { transform: translateX(-50%); }
}

@keyframes marquee-right {
  0%   { transform: translateX(-50%); }
  100% { transform: translateX(0); }
}

/* ─── Reduced Motion ──────────────────────────────── */
@media (prefers-reduced-motion: reduce) {
  .scroll-up,
  .scroll-down,
  .marquee-left,
  .marquee-right {
    animation: none;
  }
}
</style>
