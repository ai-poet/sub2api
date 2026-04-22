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
        <span class="text-sm font-medium text-[#111] dark:text-white">{{ t('home.privacy.title') }}</span>
      </div>
    </div>

    <!-- Content -->
    <article class="mx-auto max-w-[860px] px-6 py-14">
      <h1 class="text-3xl font-semibold tracking-[-0.03em] text-[#111] dark:text-white">
        {{ t('home.privacy.title') }}
      </h1>
      <p class="mt-2 text-sm text-[#999] dark:text-white/38">{{ t('home.privacy.lastUpdated') }}</p>

      <p class="mt-6 text-base leading-8 text-[#5f5850] dark:text-white/65">
        {{ t('home.privacy.intro') }}
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

      <!-- Highlight: code not stored -->
      <div class="mt-10 rounded-2xl border border-emerald-200 bg-emerald-50 p-6 dark:border-emerald-800/40 dark:bg-emerald-900/20">
        <div class="flex items-start gap-3">
          <svg class="mt-0.5 shrink-0 text-emerald-600 dark:text-emerald-400" width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M10 2a8 8 0 1 0 0 16A8 8 0 0 0 10 2zm3.7 6.3-4.5 4.5a.75.75 0 0 1-1.06 0l-2-2a.75.75 0 1 1 1.06-1.06l1.47 1.47 3.97-3.97a.75.75 0 1 1 1.06 1.06z" fill="currentColor"/>
          </svg>
          <div>
            <p class="font-semibold text-emerald-800 dark:text-emerald-300">{{ t('home.privacy.pledge.title') }}</p>
            <p class="mt-1 text-sm leading-6 text-emerald-700 dark:text-emerald-400">{{ t('home.privacy.pledge.body') }}</p>
          </div>
        </div>
      </div>

      <!-- Contact -->
      <div class="mt-10 border-t border-black/8 pt-8 dark:border-white/10">
        <p class="text-sm leading-7 text-[#999] dark:text-white/38">
          {{ t('home.privacy.contact') }}
        </p>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

const isChinese = computed(() => locale.value === 'zh')

const sections = computed(() => [
  {
    id: 'collection',
    title: t('home.privacy.sections.collection.title'),
    paragraphs: [t('home.privacy.sections.collection.p1')],
    items: [
      t('home.privacy.sections.collection.i1'),
      t('home.privacy.sections.collection.i2'),
      t('home.privacy.sections.collection.i3'),
      t('home.privacy.sections.collection.i4'),
    ],
  },
  {
    id: 'use',
    title: t('home.privacy.sections.use.title'),
    paragraphs: [t('home.privacy.sections.use.p1')],
    items: [
      t('home.privacy.sections.use.i1'),
      t('home.privacy.sections.use.i2'),
      t('home.privacy.sections.use.i3'),
      t('home.privacy.sections.use.i4'),
    ],
  },
  {
    id: 'code',
    title: t('home.privacy.sections.code.title'),
    paragraphs: [
      t('home.privacy.sections.code.p1'),
      t('home.privacy.sections.code.p2'),
    ],
  },
  {
    id: 'apikey',
    title: t('home.privacy.sections.apikey.title'),
    paragraphs: [t('home.privacy.sections.apikey.p1')],
  },
  {
    id: 'third',
    title: t('home.privacy.sections.third.title'),
    paragraphs: [t('home.privacy.sections.third.p1')],
    items: [
      t('home.privacy.sections.third.i1'),
      t('home.privacy.sections.third.i2'),
    ],
  },
  {
    id: 'retention',
    title: t('home.privacy.sections.retention.title'),
    paragraphs: [t('home.privacy.sections.retention.p1')],
  },
  {
    id: 'rights',
    title: t('home.privacy.sections.rights.title'),
    paragraphs: [t('home.privacy.sections.rights.p1')],
    items: [
      t('home.privacy.sections.rights.i1'),
      t('home.privacy.sections.rights.i2'),
      t('home.privacy.sections.rights.i3'),
    ],
  },
  {
    id: 'changes',
    title: t('home.privacy.sections.changes.title'),
    paragraphs: [t('home.privacy.sections.changes.p1')],
  },
])
</script>
