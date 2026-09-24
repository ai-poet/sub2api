<template>
  <section data-test="home-closing" class="relative overflow-hidden bg-primary-100 dark:bg-primary-950">
    <div aria-hidden="true" class="pointer-events-none absolute inset-0 hidden md:block">
      <span class="absolute left-[9%] top-[22%] h-6 w-6 rotate-12 bg-amber-300"></span>
      <span class="absolute right-[11%] top-[30%] h-10 w-10 -rotate-12 rounded-xl bg-white/80 dark:bg-white/10"></span>
      <span class="absolute bottom-[18%] left-[18%] h-4 w-4 rotate-45 bg-primary-500"></span>
      <span class="absolute bottom-[24%] right-[22%] h-5 w-5 rotate-6 bg-orange-400"></span>
    </div>

    <div class="relative mx-auto flex max-w-[1200px] flex-col items-center px-4 py-20 text-center md:px-6 md:py-28">
      <p class="text-[11px] font-bold uppercase tracking-[0.24em] text-primary-800 dark:text-primary-300">
        {{ t('home.landing.closing.overline') }}
      </p>
      <h2
        class="mt-5 text-[clamp(2.2rem,5.6vw,4.4rem)] font-black leading-[1.04] tracking-[-0.045em] text-gray-900 [text-wrap:balance] dark:text-white"
      >
        <template v-if="clientDownloadOptions.length > 0">
          <span class="block">{{ t('home.landing.closing.clientTitleLead', { siteName }) }}</span>
          <span class="block">{{ t('home.landing.closing.clientTitleTail') }}</span>
        </template>
        <template v-else>
          <span class="block">{{ t('home.landing.closing.apiTitleLead') }}</span>
          <span class="block">{{ t('home.landing.closing.apiTitleTail') }}</span>
        </template>
      </h2>

      <div v-if="clientDownloadOptions.length > 0" class="mt-10 flex w-full flex-col items-center gap-4">
        <HomeDownloadButton :options="clientDownloadOptions" />
        <router-link
          :to="dashboardPath"
          class="inline-flex items-center gap-1.5 text-sm font-semibold text-gray-800 underline-offset-4 hover:underline dark:text-white/75"
        >
          {{ t('home.landing.hero.useApi') }}
          <Icon name="arrowRight" size="xs" />
        </router-link>
      </div>
      <div v-else class="mt-10 flex w-full flex-col justify-center gap-3 sm:w-auto sm:flex-row">
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="inline-flex h-14 items-center justify-center gap-2 rounded-xl bg-gray-900 px-7 text-[15px] font-bold text-white transition hover:bg-black dark:bg-white dark:text-gray-900 dark:hover:bg-gray-100"
        >
          {{ t('home.landing.hero.startApi') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex h-14 items-center justify-center gap-2 rounded-xl border border-gray-900/15 bg-white/60 px-7 text-[15px] font-semibold text-gray-900 transition hover:bg-white dark:border-white/15 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
        >
          {{ t('home.landing.hero.viewDocs') }}
          <Icon name="externalLink" size="sm" />
        </a>
      </div>

      <p class="mt-8 text-sm text-gray-700 dark:text-white/55">{{ t('home.landing.closing.note') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import HomeDownloadButton from '@/components/home/HomeDownloadButton.vue'
import Icon from '@/components/icons/Icon.vue'
import type { ClientDownloadOption } from '@/utils/clientDownloads'

defineProps<{
  siteName: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  clientDownloadOptions: ClientDownloadOption[]
}>()

const { t } = useI18n()
</script>
