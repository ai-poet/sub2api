<template>
  <section class="mx-auto max-w-[1380px]">
    <div class="bg-white border border-gray-100 rounded-[28px] p-8 shadow-[0_2px_20px_rgba(0,0,0,0.04)] dark:bg-white/[0.03] dark:border-white/8">

      <div class="grid gap-8 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)] lg:items-center">

        <!-- Left: header text -->
        <div>
          <div class="text-[11px] uppercase tracking-[0.24em] text-primary-600 dark:text-primary-400">
            {{ t('home.proofStrip.overline') }}
          </div>
          <h2 class="mt-4 max-w-[18ch] text-3xl font-semibold leading-[1.12] tracking-[-0.04em] text-[#111] md:text-4xl [text-wrap:balance] dark:text-white">
            {{ t('home.pricing.barTitle') }}
          </h2>
          <p class="mt-4 max-w-[42rem] text-base leading-7 text-gray-500 dark:text-white/60">
            {{ t('home.pricing.description') }}
          </p>
        </div>

        <!-- Right: 3 stat cards -->
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div
            v-for="point in points"
            :key="point.label"
            class="rounded-2xl border border-gray-100 bg-white p-5 shadow-[0_1px_6px_rgba(0,0,0,0.04)] dark:border-white/8 dark:bg-white/5"
          >
            <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
              <Icon :name="point.icon" size="md" />
            </div>
            <div class="mt-4 text-sm font-semibold text-[#111] dark:text-white">
              {{ point.label }}
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

const { t } = useI18n()

const points = computed(() => [
  { icon: 'dollar' as const,   label: t('home.pricing.points.metered') },
  { icon: 'sync' as const,     label: t('home.pricing.points.routing') },
  { icon: 'chartBar' as const, label: t('home.pricing.points.visibility') },
])
</script>
