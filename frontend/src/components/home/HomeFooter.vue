<template>
  <footer class="relative z-10 px-4 pb-10 pt-12 md:px-6 md:pb-12">
    <div
      class="mx-auto flex max-w-[1380px] flex-col gap-6 border-t border-black/10 pt-6 text-sm text-[#645d54] dark:border-white/10 dark:text-white/52 md:flex-row md:items-end md:justify-between"
    >
      <div>
        <p class="home-font-serif text-2xl tracking-[-0.04em] text-[#111111] dark:text-white">
          {{ siteName }}
        </p>
        <p class="mt-2 max-w-[30rem] leading-6">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-4 md:gap-6">
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="transition hover:text-[#111111] dark:hover:text-white"
        >
          {{ t('home.docs') }}
        </a>
        <a
          :href="githubUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="transition hover:text-[#111111] dark:hover:text-white"
        >
          GitHub
        </a>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  siteName: string
  docUrl: string
  githubUrl: string
  currentYear: number
}>()

const { t } = useI18n()
</script>
