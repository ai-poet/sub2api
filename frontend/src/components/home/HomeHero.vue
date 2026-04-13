<template>
  <section class="px-4 pb-16 pt-10 md:px-6 md:pb-20 md:pt-16">
    <div class="mx-auto grid max-w-[1380px] gap-14 lg:grid-cols-[minmax(0,1.05fr)_minmax(460px,0.95fr)] lg:items-end">
      <div class="max-w-[760px]">
        <div class="inline-flex items-center gap-2 rounded-full border border-black/8 bg-white/70 px-4 py-2 text-[11px] font-semibold uppercase tracking-[0.24em] text-[#5f5850] shadow-[0_12px_40px_rgba(20,20,20,0.06)] backdrop-blur dark:border-white/10 dark:bg-white/5 dark:text-white/65">
          <Icon name="sparkles" size="sm" class="text-primary-600 dark:text-primary-300" />
          <span>{{ t('home.hero.badge') }}</span>
        </div>

        <p class="home-font-serif mt-8 text-[clamp(3.4rem,10vw,7rem)] leading-[0.88] tracking-[-0.06em] text-[#111111] dark:text-white">
          {{ siteName }}
        </p>

        <h1 class="mt-6 max-w-[10.5ch] text-[clamp(2.8rem,7vw,5.6rem)] font-semibold leading-[0.95] tracking-[-0.06em] text-[#111111] dark:text-white">
          <span class="block">{{ t('home.hero.titleLead') }}</span>
          <span class="block text-primary-700 dark:text-primary-300">{{ t('home.hero.titleAccent') }}</span>
          <span class="block">{{ t('home.hero.titleTail') }}</span>
        </h1>

        <p class="mt-6 max-w-[38rem] text-lg leading-8 text-[#5c554d] dark:text-white/72">
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

        <div class="mt-10 flex flex-wrap gap-3">
          <span
            v-for="chip in compatibilityChips"
            :key="chip"
            class="inline-flex items-center rounded-full border border-black/10 bg-white/65 px-4 py-2 text-sm font-medium text-[#3d372f] shadow-[0_10px_30px_rgba(15,15,15,0.05)] backdrop-blur dark:border-white/10 dark:bg-white/5 dark:text-white/72"
          >
            {{ chip }}
          </span>
        </div>
      </div>

      <div class="relative lg:pb-4">
        <div class="pointer-events-none absolute inset-x-10 top-[-2rem] h-32 rounded-full bg-primary-300/30 blur-3xl dark:bg-primary-500/15"></div>
        <div
          class="animate-home-float relative overflow-hidden rounded-[34px] border border-black/10 bg-[#15171c] text-white shadow-[0_30px_120px_rgba(15,15,15,0.18)] dark:border-white/10"
        >
          <div class="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(45,212,191,0.24),transparent_38%),linear-gradient(180deg,rgba(255,255,255,0.03),transparent_30%)]"></div>
          <div class="relative border-b border-white/10 px-6 py-5">
            <div class="flex items-center justify-between gap-4">
              <div class="flex items-center gap-2">
                <span class="h-3 w-3 rounded-full bg-[#ff7a59]"></span>
                <span class="h-3 w-3 rounded-full bg-[#ffbd2f]"></span>
                <span class="h-3 w-3 rounded-full bg-[#28c840]"></span>
              </div>
              <div class="text-[11px] uppercase tracking-[0.24em] text-white/45">
                {{ t('home.hero.panel.overline') }}
              </div>
            </div>

            <div class="mt-8">
              <p class="text-[12px] uppercase tracking-[0.22em] text-primary-300">
                {{ t('home.hero.panel.title') }}
              </p>
              <p class="mt-3 max-w-[28rem] text-base leading-7 text-white/72">
                {{ t('home.hero.panel.subtitle') }}
              </p>
            </div>
          </div>

          <div class="relative px-6 py-6">
            <div class="overflow-hidden rounded-[24px] border border-white/8 bg-[#0d1014] shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]">
              <div class="border-b border-white/8 bg-white/5 px-5 py-3 text-[12px] uppercase tracking-[0.2em] text-white/40">
                agent-request.ts
              </div>
              <div class="space-y-3 px-5 py-5 font-mono text-[13px] leading-6 text-white/88 md:text-[14px]">
                <div class="text-[#6ee7cf]">const response = await client.chat.completions.create({`{`}</div>
                <div class="pl-4 text-white/74">model: <span class="text-[#f9d98f]">"gpt-5.4"</span>,</div>
                <div class="pl-4 text-white/74">metadata: <span class="text-[#f9d98f]">"coding-agent"</span>,</div>
                <div class="pl-4 text-white/74">fallback: <span class="text-[#f9d98f]">"claude-sonnet-4.6"</span>,</div>
                <div class="pl-4 text-white/74">billing_mode: <span class="text-[#f9d98f]">"metered"</span>,</div>
                <div class="text-[#6ee7cf]">{`})`}</div>
              </div>
            </div>

            <div class="mt-5 grid gap-3 sm:grid-cols-2">
              <div
                v-for="row in panelRows"
                :key="row.label"
                class="rounded-[22px] border border-white/8 bg-white/5 p-4 backdrop-blur"
              >
                <div class="text-[11px] uppercase tracking-[0.2em] text-white/42">
                  {{ row.label }}
                </div>
                <div class="mt-2 text-sm leading-6 text-white/88">
                  {{ row.value }}
                </div>
              </div>
            </div>
          </div>
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
  siteName: string
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
  t('home.providers.gemini'),
  t('home.providers.openaiCompatible'),
])

const panelRows = computed(() => [
  { label: t('home.hero.panel.requestLabel'), value: t('home.hero.panel.requestValue') },
  { label: t('home.hero.panel.modelLabel'), value: t('home.hero.panel.modelValue') },
  { label: t('home.hero.panel.routeLabel'), value: t('home.hero.panel.routeValue') },
  { label: t('home.hero.panel.billLabel'), value: t('home.hero.panel.billValue') },
])
</script>
