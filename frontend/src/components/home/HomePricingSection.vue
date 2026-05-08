<template>
  <section class="mx-auto max-w-[1380px]">
    <div class="rounded-[34px] border border-black/10 bg-white/70 p-6 shadow-[0_20px_70px_rgba(15,15,15,0.05)] backdrop-blur dark:border-white/10 dark:bg-white/5 md:p-10">
      <!-- Header -->
      <div class="flex flex-col items-start justify-between gap-6 md:flex-row md:items-end">
        <div>
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7a7268] dark:text-white/42">
            {{ t('home.pricingTable.overline') }}
          </p>
          <h2 class="mt-3 text-3xl font-semibold leading-[1.12] tracking-[-0.04em] text-[#111111] dark:text-white md:text-4xl [text-wrap:balance]">
            {{ t('home.pricingTable.title') }}
          </h2>
          <p class="mt-3 max-w-[44rem] text-base leading-7 text-[#5f5850] dark:text-white/68">
            {{ t('home.pricingTable.description') }}
          </p>
        </div>
        <div class="shrink-0 rounded-2xl border border-primary-200 bg-primary-50 px-5 py-3 dark:border-primary-800/50 dark:bg-primary-900/30">
          <div class="text-[11px] uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
            {{ t('home.pricingTable.badge') }}
          </div>
          <div class="mt-0.5 text-2xl font-bold tracking-[-0.04em] text-primary-700 dark:text-primary-300">
            {{ t('home.pricingTable.badgeValue') }}
          </div>
        </div>
      </div>

      <!-- Discount cards -->
      <div class="mt-10 grid gap-5 md:grid-cols-2 lg:grid-cols-3">
        <article
          v-for="card in cards"
          :key="card.key"
          class="relative overflow-hidden rounded-[26px] border border-black/8 bg-white p-6 shadow-[0_2px_20px_rgba(0,0,0,0.04)] transition hover:-translate-y-[2px] hover:shadow-[0_18px_48px_rgba(13,148,136,0.12)] dark:border-white/10 dark:bg-white/[0.03]"
        >
          <!-- Decorative gradient -->
          <div class="pointer-events-none absolute right-[-3rem] top-[-3rem] h-32 w-32 rounded-full bg-primary-500/10 blur-3xl"></div>

          <div class="relative">
            <div class="flex items-center gap-2">
              <span class="rounded-full bg-[#111] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-white dark:bg-white dark:text-[#111]">
                {{ card.tag }}
              </span>
            </div>

            <h3 class="mt-5 text-lg font-semibold tracking-tight text-[#111] dark:text-white">
              {{ card.title }}
            </h3>
            <p class="mt-2 text-sm leading-6 text-[#5f5850] dark:text-white/65">
              {{ card.description }}
            </p>

            <div class="mt-5 flex flex-wrap gap-2">
              <span
                v-for="model in card.models"
                :key="model"
                class="rounded-full border border-black/8 bg-[#f7f3ee] px-2.5 py-1 text-xs text-[#3d362e] dark:border-white/10 dark:bg-white/5 dark:text-white/70"
              >
                {{ model }}
              </span>
            </div>
          </div>
        </article>
      </div>

    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const cards = computed(() => [
  {
    key: 'claude',
    tag: t('home.pricingTable.cards.claude.tag'),
    title: t('home.pricingTable.cards.claude.title'),
    description: t('home.pricingTable.cards.claude.description'),
    models: ['Claude Sonnet 4.6', 'Claude Opus 4.7', 'Claude Haiku 4.5'],
  },
  {
    key: 'codex',
    tag: t('home.pricingTable.cards.codex.tag'),
    title: t('home.pricingTable.cards.codex.title'),
    description: t('home.pricingTable.cards.codex.description'),
    models: ['GPT-5.5', 'GPT-5.5 Codex', 'GPT-5.4', 'GPT-5.3 Codex'],
  },
  {
    key: 'compatible',
    tag: t('home.pricingTable.cards.compatible.tag'),
    title: t('home.pricingTable.cards.compatible.title'),
    description: t('home.pricingTable.cards.compatible.description'),
    models: [t('home.providers.openaiCompatible'), 'Gemini', 'GLM', 'Qwen'],
  },
])
</script>
