<template>
  <div class="home-font-sans min-h-screen bg-gray-50 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
    <!-- Header -->
    <div class="border-b border-black/8 bg-white/80 backdrop-blur dark:border-white/10 dark:bg-white/5">
      <div class="mx-auto flex max-w-[860px] items-center justify-between px-6 py-4">
        <RouterLink
          to="/home"
          class="flex items-center gap-2 text-sm text-[#666] transition hover:text-gray-900 dark:text-white/50 dark:hover:text-white"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M10 13L5 8l5-5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          {{ pt('home.privacy.backHome') }}
        </RouterLink>
        <span class="text-sm font-medium text-gray-900 dark:text-white">{{ pt('home.terms.title') }}</span>
      </div>
    </div>

    <!-- Content -->
    <article class="mx-auto max-w-[860px] px-6 py-14">
      <h1 class="text-3xl font-semibold tracking-[-0.03em] text-gray-900 dark:text-white">
        {{ pt('home.terms.title') }}
      </h1>
      <p class="mt-2 text-sm text-[#999] dark:text-white/38">{{ pt('home.terms.lastUpdated') }}</p>

      <p class="mt-6 text-base leading-8 text-gray-600 dark:text-white/65">
        {{ pt('home.terms.intro') }}
      </p>

      <div class="mt-10 space-y-10">
        <section v-for="section in sections" :key="section.id">
          <h2 class="text-lg font-semibold tracking-[-0.02em] text-gray-900 dark:text-white">
            {{ section.title }}
          </h2>
          <div class="mt-3 space-y-3 text-base leading-8 text-gray-600 dark:text-white/65">
            <p v-for="(para, i) in section.paragraphs" :key="i">{{ para }}</p>
          </div>
          <ul v-if="section.items" class="mt-3 space-y-2 pl-5 text-base leading-7 text-gray-600 dark:text-white/65">
            <li v-for="(item, i) in section.items" :key="i" class="list-disc">{{ item }}</li>
          </ul>
          <div v-if="section.after" class="mt-3 space-y-3 text-base leading-8 text-gray-600 dark:text-white/65">
            <p v-for="(para, i) in section.after" :key="i">{{ para }}</p>
          </div>
          <!-- fork：使用规范入口 -->
          <RouterLink
            v-if="section.link"
            :to="section.link.to"
            :data-test="`terms-link-${section.id}`"
            class="mt-3 inline-block text-base font-medium text-primary-600 underline-offset-4 hover:underline dark:text-primary-400"
          >
            {{ section.link.label }}
          </RouterLink>
        </section>
      </div>

      <!-- Contact -->
      <LegalContactNote :lead="pt('home.terms.contact')" />
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LegalContactNote from '@/components/legal/LegalContactNote.vue'
import { useAppStore } from '@/stores/app'

interface TermsSection {
  id: string
  title: string
  paragraphs: string[]
  items?: string[]
  after?: string[]
  link?: { to: string; label: string }
}

const { t } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => {
  const configuredName = appStore.cachedPublicSettings?.site_name?.trim() || appStore.siteName.trim()
  return configuredName && configuredName !== 'Sub2API' ? configuredName : 'CheapRouter'
})

const pt = (key: string) => t(key, { siteName: siteName.value })

const sections = computed<TermsSection[]>(() => [
  {
    id: 'eligibility',
    title: pt('home.terms.sections.eligibility.title'),
    paragraphs: [pt('home.terms.sections.eligibility.p1')],
  },
  {
    id: 'account',
    title: pt('home.terms.sections.account.title'),
    paragraphs: [pt('home.terms.sections.account.p1')],
    items: [
      pt('home.terms.sections.account.i1'),
      pt('home.terms.sections.account.i2'),
      pt('home.terms.sections.account.i3'),
    ],
  },
  {
    id: 'service',
    title: pt('home.terms.sections.service.title'),
    paragraphs: [
      pt('home.terms.sections.service.p1'),
      pt('home.terms.sections.service.p2'),
    ],
  },
  {
    id: 'billing',
    title: pt('home.terms.sections.billing.title'),
    paragraphs: [pt('home.terms.sections.billing.p1')],
    items: [
      pt('home.terms.sections.billing.i1'),
      pt('home.terms.sections.billing.i2'),
      pt('home.terms.sections.billing.i3'),
    ],
  },
  {
    id: 'prohibited',
    title: pt('home.terms.sections.prohibited.title'),
    paragraphs: [pt('home.terms.sections.prohibited.p1')],
    items: [
      pt('home.terms.sections.prohibited.i1'),
      pt('home.terms.sections.prohibited.i2'),
      pt('home.terms.sections.prohibited.i3'),
      pt('home.terms.sections.prohibited.i4'),
    ],
    after: [pt('home.terms.sections.prohibited.p2')],
    link: { to: '/acceptable-use', label: pt('home.terms.sections.prohibited.link') },
  },
  {
    id: 'ip',
    title: pt('home.terms.sections.ip.title'),
    paragraphs: [pt('home.terms.sections.ip.p1')],
  },
  {
    id: 'disclaimer',
    title: pt('home.terms.sections.disclaimer.title'),
    paragraphs: [pt('home.terms.sections.disclaimer.p1')],
  },
  {
    id: 'termination',
    title: pt('home.terms.sections.termination.title'),
    paragraphs: [pt('home.terms.sections.termination.p1')],
  },
  {
    id: 'changes',
    title: pt('home.terms.sections.changes.title'),
    paragraphs: [pt('home.terms.sections.changes.p1')],
  },
])
</script>
