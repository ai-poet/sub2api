<template>
  <header class="sticky top-0 z-40 px-4 pt-4 md:px-6">
    <nav
      class="mx-auto flex max-w-[1380px] items-center justify-between gap-4 rounded-full border border-black/8 bg-[rgba(255,251,245,0.82)] px-4 py-3 shadow-[0_20px_80px_rgba(18,18,18,0.07)] backdrop-blur-xl dark:border-white/10 dark:bg-[rgba(17,19,24,0.82)]"
    >
      <router-link to="/home" class="flex min-w-0 items-center gap-3">
        <div class="flex h-11 w-11 items-center justify-center overflow-hidden rounded-2xl bg-white shadow-[0_10px_25px_rgba(15,15,15,0.12)] dark:bg-[#191d24]">
          <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain p-1.5" />
        </div>
        <div class="min-w-0">
          <div class="home-font-serif truncate text-2xl leading-none tracking-[-0.04em] text-[#111111] dark:text-white">
            {{ siteName }}
          </div>
          <div class="hidden text-[11px] uppercase tracking-[0.24em] text-[#6c665d] dark:text-white/55 md:block">
            {{ t('home.headerTagline') }}
          </div>
        </div>
      </router-link>

      <div class="flex items-center gap-2 md:gap-3">
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="hidden rounded-full px-3 py-2 text-sm font-medium text-[#5f5850] transition hover:bg-black/5 hover:text-[#111111] dark:text-white/60 dark:hover:bg-white/10 dark:hover:text-white lg:inline-flex"
        >
          {{ t('home.docs') }}
        </a>

        <a
          :href="githubUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="hidden rounded-full px-3 py-2 text-sm font-medium text-[#5f5850] transition hover:bg-black/5 hover:text-[#111111] dark:text-white/60 dark:hover:bg-white/10 dark:hover:text-white xl:inline-flex"
        >
          GitHub
        </a>

        <LocaleSwitcher />

        <button
          type="button"
          class="inline-flex h-10 w-10 items-center justify-center rounded-full text-[#5f5850] transition hover:bg-black/5 hover:text-[#111111] dark:text-white/65 dark:hover:bg-white/10 dark:hover:text-white"
          :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          @click="$emit('toggleTheme')"
        >
          <Icon v-if="isDark" name="sun" size="md" />
          <Icon v-else name="moon" size="md" />
        </button>

        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="inline-flex items-center gap-2 rounded-full bg-[#141518] px-4 py-2 text-sm font-semibold text-white transition hover:translate-y-[-1px] hover:bg-black dark:bg-white dark:text-[#0f1114] dark:hover:bg-[#e9e7e3]"
        >
          <span
            v-if="isAuthenticated && userInitial"
            class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-white/10 text-[11px] dark:bg-black/10"
          >
            {{ userInitial }}
          </span>
          <span>{{ isAuthenticated ? t('home.dashboard') : t('home.login') }}</span>
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
  siteLogo: string
  docUrl: string
  isDark: boolean
  isAuthenticated: boolean
  dashboardPath: string
  userInitial: string
  githubUrl: string
}>()

defineEmits<{
  toggleTheme: []
}>()

const { t } = useI18n()
</script>
