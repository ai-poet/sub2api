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
        <span class="text-sm font-medium text-gray-900 dark:text-white">{{ pt('home.aup.title') }}</span>
      </div>
    </div>

    <!-- Content -->
    <article class="mx-auto max-w-[860px] px-6 py-14">
      <h1 class="text-3xl font-semibold tracking-[-0.03em] text-gray-900 dark:text-white">
        {{ pt('home.aup.title') }}
      </h1>
      <p class="mt-2 text-sm text-[#999] dark:text-white/38">{{ pt('home.aup.lastUpdated') }}</p>

      <p class="mt-6 text-base leading-8 text-gray-600 dark:text-white/65">
        {{ pt('home.aup.intro') }}
      </p>

      <div class="mt-10 space-y-10">
        <section v-for="section in sections" :id="section.id" :key="section.id" :data-test="`aup-${section.id}`">
          <h2 class="text-lg font-semibold tracking-[-0.02em] text-gray-900 dark:text-white">
            {{ section.title }}
          </h2>
          <div class="mt-3 space-y-3 text-base leading-8 text-gray-600 dark:text-white/65">
            <p v-for="(para, i) in section.paragraphs" :key="i">{{ para }}</p>
          </div>
          <ul v-if="section.items" class="mt-3 space-y-2 pl-5 text-base leading-7 text-gray-600 dark:text-white/65">
            <li v-for="(item, i) in section.items" :key="i" class="list-disc">
              <strong v-if="item.label" class="block font-semibold text-gray-900 dark:text-white">{{ item.label }}</strong>{{ item.text }}
              <RouterLink
                v-if="item.link"
                :to="item.link.to"
                :data-test="`aup-link-${section.id}`"
                class="ml-1 font-medium text-primary-600 underline-offset-4 hover:underline dark:text-primary-400"
              >
                {{ item.link.label }}
              </RouterLink>
            </li>
          </ul>
          <div v-if="section.after" class="mt-3 space-y-3 text-base leading-8 text-gray-600 dark:text-white/65">
            <p v-for="(para, i) in section.after" :key="i">{{ para }}</p>
          </div>
        </section>
      </div>

      <LegalContactNote :lead="pt('home.aup.contact')" />
    </article>
  </div>
</template>

<script setup lang="ts">
// AIGC 内容使用规范（fork 本地）：六类禁止内容、平台内容审核机制、举报渠道与违规处置。
// 文案描述的是本站实际行为（内容审计两级核验、工单「违规内容举报」分类、申诉会话），改功能时同步改文案。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LegalContactNote from '@/components/legal/LegalContactNote.vue'
import { useAppStore } from '@/stores/app'

interface PolicyItem {
  label?: string
  text: string
  link?: { to: string; label: string }
}

interface PolicySection {
  id: string
  title: string
  paragraphs: string[]
  items?: PolicyItem[]
  after?: string[]
}

const { t } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => {
  const configuredName = appStore.cachedPublicSettings?.site_name?.trim() || appStore.siteName.trim()
  return configuredName && configuredName !== 'Sub2API' ? configuredName : 'CheapRouter'
})

const pt = (key: string) => t(key, { siteName: siteName.value })

const category = (key: string): PolicyItem => ({
  label: pt(`home.aup.sections.prohibited.${key}.label`),
  text: pt(`home.aup.sections.prohibited.${key}.text`),
})

const sections = computed<PolicySection[]>(() => [
  {
    id: 'prohibited',
    title: pt('home.aup.sections.prohibited.title'),
    paragraphs: [pt('home.aup.sections.prohibited.p1')],
    items: ['nsfw', 'violence', 'hate', 'csam', 'deepfake', 'copyright'].map(category),
  },
  {
    id: 'other',
    title: pt('home.aup.sections.other.title'),
    paragraphs: [pt('home.aup.sections.other.p1')],
    items: [
      { text: pt('home.aup.sections.other.i1') },
      { text: pt('home.aup.sections.other.i2') },
      { text: pt('home.aup.sections.other.i3') },
      { text: pt('home.aup.sections.other.i4') },
    ],
  },
  {
    id: 'moderation',
    title: pt('home.aup.sections.moderation.title'),
    paragraphs: [pt('home.aup.sections.moderation.p1')],
    items: [
      { text: pt('home.aup.sections.moderation.i1') },
      { text: pt('home.aup.sections.moderation.i2') },
      { text: pt('home.aup.sections.moderation.i3') },
      { text: pt('home.aup.sections.moderation.i4') },
      { text: pt('home.aup.sections.moderation.i5') },
      { text: pt('home.aup.sections.moderation.i6') },
    ],
  },
  {
    id: 'enforcement',
    title: pt('home.aup.sections.enforcement.title'),
    paragraphs: [pt('home.aup.sections.enforcement.p1')],
    items: [
      { text: pt('home.aup.sections.enforcement.i1') },
      { text: pt('home.aup.sections.enforcement.i2') },
      { text: pt('home.aup.sections.enforcement.i3') },
      { text: pt('home.aup.sections.enforcement.i4') },
    ],
  },
  {
    id: 'reporting',
    title: pt('home.aup.sections.reporting.title'),
    paragraphs: [pt('home.aup.sections.reporting.p1')],
    items: [
      {
        text: pt('home.aup.sections.reporting.i1'),
        link: { to: '/tickets?new=report', label: pt('home.aup.sections.reporting.i1Link') },
      },
      { text: pt('home.aup.sections.reporting.i2') },
    ],
    after: [pt('home.aup.sections.reporting.p2'), pt('home.aup.sections.reporting.p3')],
  },
  {
    id: 'appeal',
    title: pt('home.aup.sections.appeal.title'),
    paragraphs: [pt('home.aup.sections.appeal.p1')],
  },
  {
    id: 'changes',
    title: pt('home.aup.sections.changes.title'),
    paragraphs: [pt('home.aup.sections.changes.p1')],
  },
])
</script>
