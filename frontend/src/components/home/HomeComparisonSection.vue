<template>
  <section class="mx-auto max-w-[1380px]">
    <div class="rounded-[34px] border border-black/10 bg-white/70 p-6 shadow-[0_20px_70px_rgba(15,15,15,0.05)] backdrop-blur dark:border-white/10 dark:bg-white/5 md:p-8">
      <div class="max-w-[900px]">
        <p class="text-[11px] uppercase tracking-[0.24em] text-[#7a7268] dark:text-white/42">
          {{ t('home.comparison.overline') }}
        </p>
        <h2 class="mt-4 max-w-[17ch] text-3xl font-semibold leading-[1.12] tracking-[-0.04em] text-[#111111] dark:text-white md:text-4xl [text-wrap:balance]">
          {{ t('home.comparison.title') }}
        </h2>
        <p class="mt-4 max-w-[42rem] text-base leading-7 text-[#5f5850] dark:text-white/68">
          {{ t('home.comparison.description') }}
        </p>
      </div>

      <div class="mt-8 overflow-hidden rounded-[26px] border border-black/8 dark:border-white/10">
        <div class="hidden grid-cols-[180px_minmax(0,1fr)_minmax(0,1fr)] bg-[#16181d] px-6 py-4 text-sm font-semibold text-white md:grid">
          <div>{{ t('home.comparison.headers.feature') }}</div>
          <div class="text-white/62">{{ t('home.comparison.headers.official') }}</div>
          <div class="text-primary-300">{{ t('home.comparison.headers.us') }}</div>
        </div>

        <div class="divide-y divide-black/8 bg-white/82 dark:divide-white/10 dark:bg-[#14181f]/82">
          <div
            v-for="row in rows"
            :key="row.feature"
            class="grid gap-4 px-5 py-5 md:grid-cols-[180px_minmax(0,1fr)_minmax(0,1fr)] md:px-6"
          >
            <div class="text-sm font-semibold uppercase tracking-[0.18em] text-[#7a7268] dark:text-white/38">
              {{ row.feature }}
            </div>
            <div class="text-sm leading-7 text-[#60594f] dark:text-white/62">
              <div class="mb-1 text-[11px] uppercase tracking-[0.18em] text-[#8b8378] md:hidden">
                {{ t('home.comparison.headers.official') }}
              </div>
              {{ row.official }}
            </div>
            <div class="text-sm leading-7 text-[#161616] dark:text-white">
              <div class="mb-1 text-[11px] uppercase tracking-[0.18em] text-primary-600 dark:text-primary-300 md:hidden">
                {{ t('home.comparison.headers.us') }}
              </div>
              {{ row.us }}
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

const { t } = useI18n()

const rows = computed(() => [
  {
    feature: t('home.comparison.items.pricing.feature'),
    official: t('home.comparison.items.pricing.official'),
    us: t('home.comparison.items.pricing.us'),
  },
  {
    feature: t('home.comparison.items.models.feature'),
    official: t('home.comparison.items.models.official'),
    us: t('home.comparison.items.models.us'),
  },
  {
    feature: t('home.comparison.items.stability.feature'),
    official: t('home.comparison.items.stability.official'),
    us: t('home.comparison.items.stability.us'),
  },
])
</script>
