<template>
  <section class="mx-auto max-w-[1380px]">
    <div class="max-w-[920px]">
      <p class="text-[11px] uppercase tracking-[0.24em] text-[#7a7268] dark:text-white/42">
        {{ t('home.trust.overline') }}
      </p>
      <h2 class="mt-4 max-w-[15ch] text-4xl font-semibold leading-[1.02] tracking-[-0.06em] text-[#111111] dark:text-white md:text-5xl">
        {{ t('home.trust.title') }}
      </h2>
      <p class="mt-5 max-w-[48rem] text-lg leading-8 text-[#5f5850] dark:text-white/68">
        {{ t('home.trust.description') }}
      </p>
    </div>

    <div class="mt-10 overflow-hidden rounded-full border border-black/8 bg-white/65 py-3 shadow-[0_20px_60px_rgba(15,15,15,0.04)] backdrop-blur dark:border-white/10 dark:bg-white/5">
      <div class="animate-home-marquee flex w-max gap-3 whitespace-nowrap px-4">
        <span
          v-for="(item, index) in marqueeItems"
          :key="`${item}-${index}`"
          class="inline-flex items-center rounded-full border border-black/8 bg-[#f7f2ea] px-4 py-2 text-sm font-medium text-[#2e2923] dark:border-white/8 dark:bg-[#11151b] dark:text-white/75"
        >
          {{ item }}
        </span>
      </div>
    </div>

    <div class="mt-12 grid gap-5 lg:grid-cols-3">
      <article
        v-for="card in cards"
        :key="card.title"
        class="rounded-[28px] border border-black/8 bg-white/70 p-6 shadow-[0_18px_60px_rgba(15,15,15,0.05)] backdrop-blur dark:border-white/10 dark:bg-white/5"
      >
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-[#121316] text-white dark:bg-white dark:text-[#0f1114]">
          <Icon :name="card.icon" size="md" />
        </div>
        <h3 class="mt-6 text-2xl font-semibold tracking-[-0.04em] text-[#111111] dark:text-white">
          {{ card.title }}
        </h3>
        <p class="mt-4 text-base leading-7 text-[#5f5850] dark:text-white/65">
          {{ card.description }}
        </p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const marqueeItems = computed(() => {
  const base = [
    t('home.providers.claudeCode'),
    t('home.providers.codex'),
    t('home.providers.gpt'),
    t('home.providers.gemini'),
    t('home.providers.openaiCompatible'),
    t('home.trust.trackers.routing'),
    t('home.trust.trackers.billing'),
    t('home.trust.trackers.visibility'),
  ]

  return [...base, ...base]
})

const cards = computed(() => [
  {
    title: t('home.trust.cards.gateway.title'),
    description: t('home.trust.cards.gateway.description'),
    icon: 'link' as const,
  },
  {
    title: t('home.trust.cards.resilience.title'),
    description: t('home.trust.cards.resilience.description'),
    icon: 'shield' as const,
  },
  {
    title: t('home.trust.cards.visibility.title'),
    description: t('home.trust.cards.visibility.description'),
    icon: 'chartBar' as const,
  },
])
</script>
