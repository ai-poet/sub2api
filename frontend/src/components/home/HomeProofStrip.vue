<template>
  <section class="mx-auto max-w-[1380px]">
    <div
      class="overflow-hidden rounded-[34px] bg-[#141518] px-6 py-8 text-white shadow-[0_30px_100px_rgba(15,15,15,0.16)] md:px-8 md:py-10"
    >
      <div class="grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)] lg:items-center">
        <div>
          <div class="text-[11px] uppercase tracking-[0.24em] text-primary-300">
            {{ t('home.pricing.title') }}
          </div>
          <h2 class="mt-4 max-w-[16ch] text-3xl font-semibold leading-[1.04] tracking-[-0.05em] md:text-4xl">
            {{ t('home.pricing.barTitle') }}
          </h2>
          <p class="mt-4 max-w-[42rem] text-base leading-7 text-white/68">
            {{ t('home.pricing.description') }}
          </p>
        </div>

        <div class="grid gap-3 sm:grid-cols-3">
          <div class="rounded-[26px] border border-white/10 bg-white/5 p-4 sm:col-span-3">
            <div class="text-[11px] uppercase tracking-[0.22em] text-white/45">
              {{ t('home.pricing.subtitle') }}
            </div>
            <div class="mt-2 text-4xl font-semibold tracking-[-0.05em] text-[#f7e0b5]">
              {{ t('home.pricing.highlight') }}
            </div>
            <div class="mt-2 text-sm text-white/62">
              {{ t('home.pricing.badge') }}
            </div>
          </div>

          <div
            v-for="point in points"
            :key="point.label"
            class="rounded-[24px] border border-white/10 bg-white/5 p-4"
          >
            <div class="flex h-10 w-10 items-center justify-center rounded-full bg-white/8 text-primary-300">
              <Icon :name="point.icon" size="md" />
            </div>
            <div class="mt-4 text-sm font-semibold text-white">
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
  { icon: 'dollar' as const, label: t('home.pricing.points.metered') },
  { icon: 'sync' as const, label: t('home.pricing.points.routing') },
  { icon: 'chartBar' as const, label: t('home.pricing.points.visibility') },
])
</script>
