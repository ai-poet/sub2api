<template>
  <section v-if="downloadOptions.length > 0" class="mx-auto max-w-[1180px]">
    <div
      class="rounded-2xl border border-gray-200 bg-white/82 p-6 shadow-[0_16px_50px_rgba(24,22,18,0.08)] backdrop-blur dark:border-white/10 dark:bg-white/[0.04] md:p-8"
    >
      <div class="grid gap-8 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)] lg:items-center">
        <div>
          <div
            class="inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-emerald-700 dark:border-emerald-700/40 dark:bg-emerald-900/20 dark:text-emerald-300"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span>
            {{ t('home.download.badge') }}
          </div>

          <h2
            class="mt-4 max-w-[24ch] text-3xl font-semibold leading-[1.15] text-[#111] dark:text-white md:text-4xl [text-wrap:balance]"
          >
            {{ t('home.download.title') }}
          </h2>

          <p class="mt-3 max-w-[42rem] text-base leading-7 text-gray-600 dark:text-white/60">
            {{ t('home.download.description') }}
          </p>

          <div class="mt-6 flex flex-col gap-3">
            <div class="flex items-center gap-2.5 text-sm text-gray-600 dark:text-white/60">
              <span
                class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >
                <Icon name="shield" size="sm" />
              </span>
              {{ t('home.download.privacyCode') }}
            </div>
            <div class="flex items-center gap-2.5 text-sm text-gray-600 dark:text-white/60">
              <span
                class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
              >
                <Icon name="sync" size="sm" />
              </span>
              {{ t('home.download.privacyKey') }}
            </div>
          </div>
        </div>

        <div class="grid gap-3">
          <a
            v-for="option in downloadOptions"
            :key="option.id"
            :href="option.url"
            :data-platform="option.id"
            target="_blank"
            rel="noopener noreferrer"
            class="group flex items-center justify-between gap-4 rounded-xl border border-gray-200 bg-gray-50/75 p-4 transition hover:-translate-y-0.5 hover:border-primary-200 hover:bg-white hover:shadow-[0_16px_40px_rgba(15,23,42,0.08)] dark:border-white/10 dark:bg-white/[0.04] dark:hover:border-primary-400/40 dark:hover:bg-white/[0.07] sm:p-5"
          >
            <span class="flex min-w-0 items-center gap-4">
              <span
                class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-white text-gray-800 shadow-sm ring-1 ring-gray-100 dark:bg-white/8 dark:text-white dark:ring-white/10"
              >
                <svg class="h-6 w-6" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                  <path :d="option.iconPath" />
                </svg>
              </span>
              <span class="min-w-0">
                <span class="flex flex-wrap items-center gap-2">
                  <span class="font-semibold text-gray-900 dark:text-white">{{ option.name }}</span>
                  <span
                    v-if="option.id === preferredPlatform"
                    class="rounded-full bg-primary-100 px-2.5 py-1 text-[11px] font-semibold text-primary-700 dark:bg-primary-900/35 dark:text-primary-300"
                  >
                    {{ t('home.download.recommended') }}
                  </span>
                </span>
                <span class="mt-1 block text-sm text-gray-500 dark:text-white/45">
                  {{ option.sub }}
                </span>
              </span>
            </span>
            <span
              class="inline-flex shrink-0 items-center gap-2 rounded-full bg-gray-900 px-4 py-2 text-sm font-semibold text-white transition group-hover:bg-primary-600 dark:bg-white dark:text-gray-950 dark:group-hover:bg-primary-300"
            >
              {{ t('home.download.cta') }}
              <Icon name="download" size="sm" />
            </span>
          </a>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  detectPreferredClientPlatform,
  getClientDownloadOptions,
  type ClientDownloadPlatform,
} from '@/utils/clientDownloads'

interface DownloadOption {
  id: ClientDownloadPlatform
  name: string
  sub: string
  url: string
  iconPath: string
}

const props = defineProps<{
  windowsUrl?: string
  macosUrl?: string
}>()

const { t } = useI18n()

const platformIcons: Record<ClientDownloadPlatform, string> = {
  windows:
    'M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801',
  macos:
    'M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z',
}

const preferredPlatform = computed(() => detectPreferredClientPlatform())

const downloadOptions = computed<DownloadOption[]>(() =>
  getClientDownloadOptions(
    {
      windowsUrl: props.windowsUrl,
      macosUrl: props.macosUrl,
    },
    preferredPlatform.value,
  ).map((option) => ({
    ...option,
    sub: t(
      option.id === 'windows'
        ? 'home.download.platforms.windows.sub'
        : 'home.download.platforms.mac.sub',
    ),
    iconPath: platformIcons[option.id],
  })),
)
</script>
