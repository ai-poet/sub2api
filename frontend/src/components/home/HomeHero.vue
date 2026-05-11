<template>
  <section class="px-4 pb-6 pt-10 md:px-6 md:pb-8 md:pt-14">
    <div class="mx-auto max-w-[1380px]">
      <div class="w-full max-w-none">
        <div class="inline-flex items-center gap-2 rounded-full border border-black/8 bg-white/70 px-4 py-2 text-[11px] font-semibold uppercase tracking-[0.24em] text-[#5f5850] shadow-[0_12px_40px_rgba(20,20,20,0.06)] backdrop-blur dark:border-white/10 dark:bg-white/5 dark:text-white/65">
          <Icon name="sparkles" size="sm" class="text-primary-600 dark:text-primary-300" />
          <span>{{ t('home.hero.badge') }}</span>
        </div>

        <h1 class="mt-8 max-w-[12.5ch] text-[clamp(2.9rem,7vw,5.8rem)] font-semibold leading-[1.02] tracking-[-0.05em] text-[#111111] dark:text-white [text-wrap:balance]">
          <span class="block">{{ t('home.hero.titleLead') }}</span>
          <span class="block text-primary-700 dark:text-primary-300">{{ t('home.hero.titleAccent') }}</span>
          <span class="block">{{ t('home.hero.titleTail') }}</span>
        </h1>

        <p class="mt-6 max-w-[42rem] text-lg leading-8 text-[#5c554d] dark:text-white/72">
          {{ t('home.hero.description') }}
        </p>

        <p class="mt-4 text-[12px] uppercase tracking-[0.22em] text-[#837a6f] dark:text-white/45">
          {{ subtitleLine }}
        </p>

        <div class="mt-10 flex flex-col gap-3 sm:flex-row">
          <router-link
            :to="primaryTo"
            class="inline-flex h-14 items-center justify-center gap-2 rounded-full bg-[#121316] px-6 text-[15px] font-semibold text-white transition hover:translate-y-[-1px] hover:bg-black dark:bg-white dark:text-[#111318] dark:hover:bg-[#ece9e5]"
          >
            <span>{{ primaryLabel }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex h-14 items-center justify-center gap-2 rounded-full border border-black/10 bg-white/70 px-6 text-[15px] font-semibold text-[#111111] transition hover:translate-y-[-1px] hover:bg-white dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
          >
            <span>{{ t('home.docs') }}</span>
            <Icon name="externalLink" size="sm" />
          </a>
          <router-link
            v-else
            to="/login"
            class="inline-flex h-14 items-center justify-center gap-2 rounded-full border border-black/10 bg-white/70 px-6 text-[15px] font-semibold text-[#111111] transition hover:translate-y-[-1px] hover:bg-white dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
          >
            <span>{{ t('home.login') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>

        <p class="mt-4 text-sm leading-6 text-[#6d655b] dark:text-white/58">
          {{ t('home.hero.primaryNote') }}
        </p>

        <div class="mt-8 flex flex-wrap gap-3">
          <span
            v-for="chip in compatibilityChips"
            :key="chip"
            class="inline-flex items-center rounded-full border border-black/10 bg-white/65 px-4 py-2 text-sm font-medium text-[#3d372f] shadow-[0_10px_30px_rgba(15,15,15,0.05)] backdrop-blur dark:border-white/10 dark:bg-white/5 dark:text-white/72"
          >
            {{ chip }}
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  siteSubtitle: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
}>()

const { t } = useI18n()

const primaryTo = computed(() => (props.isAuthenticated ? props.dashboardPath : '/register'))
const primaryLabel = computed(() => (props.isAuthenticated ? t('home.goToDashboard') : t('home.cta.button')))
const subtitleLine = computed(() => props.siteSubtitle.trim() || t('home.heroSubtitle'))

const compatibilityChips = computed(() => [
  t('home.providers.claudeCode'),
  t('home.providers.codex'),
  t('home.providers.gpt'),
  t('home.providers.openaiCompatible'),
])
</script>
