<template>
  <section class="relative w-full overflow-hidden bg-white px-4 pb-12 pt-12 md:px-6 md:pb-16 md:pt-16 dark:bg-[#0f1114]">

    <div class="mx-auto w-full max-w-[1380px] min-w-0">

      <!-- Category tag pills -->
      <div class="flex flex-wrap gap-2">
        <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.coding') }}</span>
        <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.agent') }}</span>
        <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.tools') }}</span>
      </div>

      <!-- Headline -->
      <h1 class="mt-6 max-w-full text-[clamp(1.95rem,7.8vw,5rem)] font-black leading-[1.06] tracking-[-0.04em] [overflow-wrap:anywhere] [text-wrap:balance] sm:text-[clamp(2.6rem,6vw,5rem)]">
        <span class="block text-[#111] dark:text-white">
          <span class="block sm:inline">{{ t('home.hero.titleLeadPrimary') }}</span>
          <span class="block sm:ml-[0.18em] sm:inline">{{ t('home.hero.titleLeadSecondary') }}</span>
        </span>
        <span v-if="titleAccent" class="block text-primary-600 dark:text-primary-400">{{ titleAccent }}</span>
        <span v-if="titleTail" class="block text-[#111] dark:text-white">{{ titleTail }}</span>
      </h1>

      <!-- CTAs -->
      <div class="mt-6 flex flex-col gap-3 sm:flex-row">
        <a
          v-for="(option, index) in clientDownloadOptions"
          :key="option.id"
          :href="option.url"
          :data-platform="option.id"
          :data-test="index === 0 ? 'hero-primary-download' : 'hero-platform-download'"
          target="_blank"
          rel="noopener noreferrer"
          :class="[
            'inline-flex h-14 w-full items-center justify-center gap-2 rounded-full px-8 text-[15px] transition hover:-translate-y-[1px] active:translate-y-0 sm:w-auto',
            index === 0
              ? 'bg-[#111] font-bold text-white hover:bg-black dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5]'
              : 'border border-gray-200 bg-white font-semibold text-[#111] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10',
          ]"
        >
          <span>{{ index === 0 ? t('home.hero.downloadPrimary') : t('home.clientShowcase.downloadCta', { platform: option.name }) }}</span>
          <Icon name="download" size="sm" />
        </a>
        <router-link
          v-if="clientDownloadOptions.length > 0"
          :to="dashboardPath"
          data-test="hero-connect-api"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 active:translate-y-0 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
        >
          <span>{{ t('home.hero.connectApi') }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>
        <router-link
          v-if="clientDownloadOptions.length === 0"
          :to="primaryTo"
          data-test="hero-primary-fallback"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full bg-[#111] px-8 text-[15px] font-bold text-white transition hover:-translate-y-[1px] hover:bg-black active:translate-y-0 dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5] sm:w-auto"
        >
          <span>{{ primaryLabel }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>

        <a
          v-if="clientDownloadOptions.length === 0 && docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
        >
          <span>{{ t('home.viewDocs') }}</span>
          <Icon name="externalLink" size="sm" />
        </a>
        <router-link
          v-else-if="clientDownloadOptions.length === 0"
          to="/login"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
        >
          <span>{{ t('home.login') }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>

      <!-- Note -->
      <p class="mt-2 max-w-full text-sm leading-6 text-gray-400 [overflow-wrap:anywhere] dark:text-white/35">
        {{ t('home.hero.primaryNote') }}
      </p>
    </div>

    <!-- ===== Agent Workflow Preview ===== -->
    <div class="mx-auto mt-12 max-w-[1380px]">
      <!-- Title & description -->
      <div class="mb-6">
        <h2 class="text-2xl font-semibold tracking-[-0.02em] text-[#111] dark:text-white md:text-3xl">
          {{ t('home.clientShowcase.title') }}
        </h2>
        <p class="mt-2 max-w-[44rem] text-base leading-7 text-gray-500 dark:text-white/55">
          {{ t('home.clientShowcase.description') }}
        </p>
      </div>

      <!-- Feature pills -->
      <div class="mb-6 flex flex-wrap gap-2">
        <span v-for="pill in pills" :key="pill" class="rounded-full border border-gray-200 bg-white px-3.5 py-1.5 text-sm text-gray-600 dark:border-white/10 dark:bg-white/5 dark:text-white/70">
          {{ pill }}
        </span>
      </div>

      <HomeAgentWorkflowPreview />

    </div>

  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeAgentWorkflowPreview from '@/components/home/HomeAgentWorkflowPreview.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  detectPreferredClientPlatform,
  getClientDownloadOptions,
} from '@/utils/clientDownloads'

const props = defineProps<{
  siteSubtitle: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  windowsUrl?: string
  macosUrl?: string
}>()

const { t } = useI18n()

const primaryTo = computed(() => (props.isAuthenticated ? props.dashboardPath : '/login'))
const primaryLabel = computed(() => (props.isAuthenticated ? t('home.goToDashboard') : t('home.cta.button')))
const titleAccent = computed(() => t('home.hero.titleAccent').trim())
const titleTail = computed(() => t('home.hero.titleTail').trim())
const preferredClientPlatform = computed(() => detectPreferredClientPlatform())
const clientDownloadOptions = computed(() =>
  getClientDownloadOptions(
    {
      windowsUrl: props.windowsUrl,
      macosUrl: props.macosUrl,
    },
    preferredClientPlatform.value,
  ),
)

const pills = computed(() => [
  t('home.clientShowcase.pills.darkMode'),
  t('home.clientShowcase.pills.workspace'),
  t('home.clientShowcase.pills.terminal'),
  t('home.clientShowcase.pills.parallel'),
])
</script>
