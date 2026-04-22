<template>
  <section class="mx-auto max-w-[1380px]">
    <div
      class="rounded-[34px] border border-gray-100 bg-white p-8 shadow-[0_2px_20px_rgba(0,0,0,0.04)] dark:border-white/8 dark:bg-white/[0.03] md:p-10"
    >
      <div class="grid gap-10 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)] lg:items-center">

        <!-- Left: copy -->
        <div>
          <!-- Coming-soon badge -->
          <div class="inline-flex items-center gap-2 rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-amber-700 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-400">
            <span class="h-1.5 w-1.5 rounded-full bg-amber-400"></span>
            {{ t('home.download.badge') }}
          </div>

          <h2 class="mt-4 max-w-[20ch] text-3xl font-semibold leading-[1.15] tracking-[-0.04em] text-[#111] dark:text-white md:text-4xl [text-wrap:balance]">
            {{ t('home.download.title') }}
          </h2>

          <p class="mt-3 max-w-[40rem] text-base leading-7 text-gray-500 dark:text-white/55">
            {{ t('home.download.description') }}
          </p>

          <!-- Privacy trust badges -->
          <div class="mt-6 flex flex-col gap-3">
            <div class="flex items-center gap-2.5 text-sm text-gray-600 dark:text-white/60">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
                <Icon name="shield" size="sm" />
              </span>
              {{ t('home.download.privacyCode') }}
            </div>
            <div class="flex items-center gap-2.5 text-sm text-gray-600 dark:text-white/60">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
                <Icon name="lock" size="sm" />
              </span>
              {{ t('home.download.privacyKey') }}
            </div>
          </div>

          <!-- Notify CTA -->
          <div class="mt-8">
            <router-link
              :to="isAuthenticated ? dashboardPath : '/register'"
              class="inline-flex items-center gap-2 rounded-full bg-[#111] px-6 py-3 text-sm font-semibold text-white transition hover:-translate-y-[1px] hover:bg-black active:translate-y-0 dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5]"
            >
              <span>{{ t('home.download.cta') }}</span>
              <Icon name="arrowRight" size="sm" />
            </router-link>
          </div>
        </div>

        <!-- Right: platform cards -->
        <div class="grid grid-cols-3 gap-3">
          <div
            v-for="platform in platforms"
            :key="platform.id"
            class="group flex cursor-default flex-col rounded-2xl border border-gray-100 bg-gray-50/60 p-5 dark:border-white/8 dark:bg-white/[0.03]"
          >
            <!-- Platform icon -->
            <div
              class="flex h-10 w-10 items-center justify-center rounded-xl text-gray-400 dark:text-white/30"
              :class="platform.iconBg"
            >
              <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path :d="platform.iconPath" />
              </svg>
            </div>

            <div class="mt-4 font-semibold text-gray-700 dark:text-white/60">{{ platform.name }}</div>
            <div class="mt-0.5 text-xs text-gray-400 dark:text-white/30">{{ platform.sub }}</div>

            <div class="mt-4">
              <span class="inline-flex items-center gap-1 rounded-full bg-gray-200/80 px-2.5 py-1 text-[11px] font-medium text-gray-500 dark:bg-white/10 dark:text-white/40">
                {{ t('home.download.comingSoon') }}
              </span>
            </div>
          </div>
        </div>

      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  isAuthenticated: boolean
  dashboardPath: string
}>()

const { t } = useI18n()

// Platform icon paths (simplified SVG paths)
const platforms = [
  {
    id: 'macos',
    name: 'macOS',
    sub: 'Apple Silicon & Intel',
    iconBg: 'bg-gray-100 dark:bg-white/5',
    // Apple logo path
    iconPath: 'M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z',
  },
  {
    id: 'windows',
    name: 'Windows',
    sub: 'x64 / ARM64',
    iconBg: 'bg-gray-100 dark:bg-white/5',
    // Windows logo path
    iconPath: 'M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801',
  },
  {
    id: 'linux',
    name: 'Linux',
    sub: '.deb / .rpm / AppImage',
    iconBg: 'bg-gray-100 dark:bg-white/5',
    // Terminal / command line icon (represents Linux CLI)
    iconPath: 'M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 14H4V6h16v12zM6 10l1.41-1.41L10 11.17l-2.59 2.58L6 12.34 7.66 10.75 6 9.17V10zm5 4h5v-2h-5v2z',
  },
]
</script>
