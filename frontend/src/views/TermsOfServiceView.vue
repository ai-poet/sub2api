<template>
  <div class="home-font-sans min-h-screen bg-[#f8fafb] text-[#161616] dark:bg-[#0f1114] dark:text-[#f3f1ed]">
    <!-- Header -->
    <div class="border-b border-black/8 bg-white/80 backdrop-blur dark:border-white/10 dark:bg-white/5">
      <div class="mx-auto flex max-w-[860px] items-center justify-between px-6 py-4">
        <RouterLink
          to="/home"
          class="flex items-center gap-2 text-sm text-[#666] transition hover:text-[#111] dark:text-white/50 dark:hover:text-white"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M10 13L5 8l5-5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          {{ t('home.privacy.backHome') }}
        </RouterLink>
        <span class="text-sm font-medium text-[#111] dark:text-white">{{ t('home.terms.title') }}</span>
      </div>
    </div>

    <!-- Content -->
    <article class="mx-auto max-w-[860px] px-6 py-14">
      <h1 class="text-3xl font-semibold tracking-[-0.03em] text-[#111] dark:text-white">
        {{ t('home.terms.title') }}
      </h1>
      <p class="mt-2 text-sm text-[#999] dark:text-white/38">{{ t('home.terms.lastUpdated') }}</p>

      <p class="mt-6 text-base leading-8 text-[#5f5850] dark:text-white/65">
        {{ t('home.terms.intro') }}
      </p>

      <div class="mt-10 space-y-10">
        <section v-for="section in sections" :key="section.id">
          <h2 class="text-lg font-semibold tracking-[-0.02em] text-[#111] dark:text-white">
            {{ section.title }}
          </h2>
          <div class="mt-3 space-y-3 text-base leading-8 text-[#5f5850] dark:text-white/65">
            <p v-for="(para, i) in section.paragraphs" :key="i">{{ para }}</p>
          </div>
          <ul v-if="section.items" class="mt-3 space-y-2 pl-5 text-base leading-7 text-[#5f5850] dark:text-white/65">
            <li v-for="(item, i) in section.items" :key="i" class="list-disc">{{ item }}</li>
          </ul>
        </section>
      </div>

      <!-- Contact -->
      <div class="mt-10 border-t border-black/8 pt-8 dark:border-white/10">
        <p class="text-sm leading-7 text-[#999] dark:text-white/38">
          {{ t('home.terms.contact') }}
        </p>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const sections = computed(() => [
  {
    id: 'eligibility',
    title: t('home.terms.sections.eligibility.title'),
    paragraphs: [t('home.terms.sections.eligibility.p1')],
  },
  {
    id: 'account',
    title: t('home.terms.sections.account.title'),
    paragraphs: [t('home.terms.sections.account.p1')],
    items: [
      t('home.terms.sections.account.i1'),
      t('home.terms.sections.account.i2'),
      t('home.terms.sections.account.i3'),
    ],
  },
  {
    id: 'service',
    title: t('home.terms.sections.service.title'),
    paragraphs: [
      t('home.terms.sections.service.p1'),
      t('home.terms.sections.service.p2'),
    ],
  },
  {
    id: 'billing',
    title: t('home.terms.sections.billing.title'),
    paragraphs: [t('home.terms.sections.billing.p1')],
    items: [
      t('home.terms.sections.billing.i1'),
      t('home.terms.sections.billing.i2'),
      t('home.terms.sections.billing.i3'),
    ],
  },
  {
    id: 'prohibited',
    title: t('home.terms.sections.prohibited.title'),
    paragraphs: [t('home.terms.sections.prohibited.p1')],
    items: [
      t('home.terms.sections.prohibited.i1'),
      t('home.terms.sections.prohibited.i2'),
      t('home.terms.sections.prohibited.i3'),
      t('home.terms.sections.prohibited.i4'),
    ],
  },
  {
    id: 'ip',
    title: t('home.terms.sections.ip.title'),
    paragraphs: [t('home.terms.sections.ip.p1')],
  },
  {
    id: 'disclaimer',
    title: t('home.terms.sections.disclaimer.title'),
    paragraphs: [t('home.terms.sections.disclaimer.p1')],
  },
  {
    id: 'termination',
    title: t('home.terms.sections.termination.title'),
    paragraphs: [t('home.terms.sections.termination.p1')],
  },
  {
    id: 'changes',
    title: t('home.terms.sections.changes.title'),
    paragraphs: [t('home.terms.sections.changes.p1')],
  },
])
</script>
