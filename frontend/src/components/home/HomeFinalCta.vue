<template>
  <section class="mx-auto max-w-[1380px]">
    <div class="overflow-hidden rounded-[36px] bg-[#121316] px-6 py-10 text-white shadow-[0_35px_120px_rgba(12,12,12,0.18)] md:px-10 md:py-14">
      <div class="relative">
        <div class="pointer-events-none absolute right-[-6rem] top-[-5rem] h-40 w-40 rounded-full bg-primary-400/30 blur-3xl"></div>
        <div class="pointer-events-none absolute bottom-[-6rem] left-[-4rem] h-40 w-40 rounded-full bg-white/10 blur-3xl"></div>

        <div class="relative max-w-[760px]">
          <p class="text-[11px] uppercase tracking-[0.26em] text-primary-300">
            {{ t('home.cta.eyebrow') }}
          </p>
          <h2 class="mt-5 max-w-[14ch] text-4xl font-semibold leading-[1.1] tracking-[-0.04em] md:text-5xl [text-wrap:balance]">
            {{ t('home.cta.title') }}
          </h2>
          <p class="mt-5 max-w-[44rem] text-lg leading-8 text-white/72">
            {{ t('home.cta.description') }}
          </p>

          <div class="mt-10 flex flex-col gap-3 sm:flex-row">
            <router-link
              :to="primaryTo"
              class="inline-flex h-14 items-center justify-center gap-2 rounded-full bg-white px-6 text-[15px] font-semibold text-[#111318] transition hover:translate-y-[-1px] hover:bg-[#ede9e3]"
            >
              <span>{{ primaryLabel }}</span>
              <Icon name="arrowRight" size="sm" />
            </router-link>

            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex h-14 items-center justify-center gap-2 rounded-full border border-white/12 bg-white/5 px-6 text-[15px] font-semibold text-white transition hover:translate-y-[-1px] hover:bg-white/10"
            >
              <span>{{ t('home.docs') }}</span>
              <Icon name="externalLink" size="sm" />
            </a>
            <router-link
              v-else
              to="/login"
              class="inline-flex h-14 items-center justify-center gap-2 rounded-full border border-white/12 bg-white/5 px-6 text-[15px] font-semibold text-white transition hover:translate-y-[-1px] hover:bg-white/10"
            >
              <span>{{ t('home.login') }}</span>
              <Icon name="arrowRight" size="sm" />
            </router-link>
          </div>

          <p class="mt-5 text-sm leading-6 text-white/55">
            {{ t('home.cta.stat') }}
          </p>
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
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
}>()

const { t } = useI18n()

const primaryTo = computed(() => (props.isAuthenticated ? props.dashboardPath : '/register'))
const primaryLabel = computed(() => (props.isAuthenticated ? t('home.goToDashboard') : t('home.cta.button')))
</script>
