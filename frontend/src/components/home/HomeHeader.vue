<template>
  <header class="sticky top-0 z-40 px-4 pt-4 md:px-6">
    <nav
      class="mx-auto flex max-w-[1380px] items-center justify-between gap-4 rounded-full border border-black/10 bg-white/90 px-5 py-3 shadow-[0_8px_40px_rgba(0,0,0,0.06)] backdrop-blur-xl dark:border-white/10 dark:bg-[rgba(17,19,24,0.88)]"
    >
      <!-- Logo -->
      <router-link to="/home" class="flex min-w-0 items-center gap-2.5">
        <div class="home-font-serif truncate text-[1.35rem] font-black leading-none tracking-[-0.04em] text-[#111] dark:text-white">
          {{ siteName }}
        </div>
      </router-link>

      <!-- Nav links (desktop) -->
      <div class="hidden items-center gap-1 lg:flex">
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="rounded-full px-3.5 py-2 text-[13.5px] font-medium text-[#555] transition hover:bg-black/5 hover:text-[#111] dark:text-white/60 dark:hover:bg-white/10 dark:hover:text-white"
        >
          {{ t('home.docs') }}
        </a>
        <router-link
          to="/models"
          class="rounded-full px-3.5 py-2 text-[13.5px] font-medium text-[#555] transition hover:bg-black/5 hover:text-[#111] dark:text-white/60 dark:hover:bg-white/10 dark:hover:text-white"
        >
          {{ t('home.navModels') }}
        </router-link>
        <a
          href="#pricing"
          class="rounded-full px-3.5 py-2 text-[13.5px] font-medium text-[#555] transition hover:bg-black/5 hover:text-[#111] dark:text-white/60 dark:hover:bg-white/10 dark:hover:text-white"
          @click.prevent="scrollTo('pricing')"
        >
          {{ t('home.navPricing') }}
        </a>
      </div>

      <!-- Right actions -->
      <div class="flex items-center gap-2">
        <LocaleSwitcher />

        <button
          type="button"
          class="inline-flex h-9 w-9 items-center justify-center rounded-full text-[#777] transition hover:bg-black/5 hover:text-[#111] dark:text-white/50 dark:hover:bg-white/10 dark:hover:text-white"
          :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          @click="$emit('toggleTheme')"
        >
          <Icon v-if="isDark" name="sun" size="md" />
          <Icon v-else name="moon" size="md" />
        </button>

        <router-link
          :to="isAuthenticated ? dashboardPath : '/register'"
          class="inline-flex items-center gap-1.5 rounded-full bg-primary-600 px-4 py-2 text-[13.5px] font-semibold text-white transition hover:-translate-y-[1px] hover:bg-primary-700 active:translate-y-0 dark:bg-primary-500 dark:hover:bg-primary-400"
        >
          <span
            v-if="isAuthenticated && userInitial"
            class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-white/20 text-[10px]"
          >
            {{ userInitial }}
          </span>
          <span>{{ isAuthenticated ? t('home.dashboard') : t('home.cta.button') }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  siteName: string
  docUrl: string
  isDark: boolean
  isAuthenticated: boolean
  dashboardPath: string
  userInitial: string
}>()

defineEmits<{
  toggleTheme: []
}>()

const { t } = useI18n()

function scrollTo(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
}
</script>
