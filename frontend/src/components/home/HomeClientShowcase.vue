<template>
  <section class="mx-auto max-w-[1380px]">
    <div
      class="relative overflow-hidden rounded-[34px] border border-black/10 bg-[#0a0a0c] p-6 shadow-[0_20px_70px_rgba(15,15,15,0.05)] md:p-10"
    >
      <!-- Ambient glow behind screenshot -->
      <div
        class="pointer-events-none absolute left-1/2 top-1/2 h-[500px] w-[800px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary-500/8 blur-[120px]"
      ></div>
      <div
        class="pointer-events-none absolute right-[-4rem] top-[-4rem] h-64 w-64 rounded-full bg-primary-400/6 blur-[80px]"
      ></div>
      <div
        class="pointer-events-none absolute bottom-[-4rem] left-[-4rem] h-64 w-64 rounded-full bg-primary-600/6 blur-[80px]"
      ></div>

      <!-- Header -->
      <div class="relative z-10 max-w-[720px]">
        <div
          class="inline-flex items-center gap-2 rounded-full border border-primary-800/40 bg-primary-900/20 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-primary-400"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-primary-400"></span>
          {{ t('home.clientShowcase.badge') }}
        </div>

        <h2
          class="mt-5 max-w-[18ch] text-3xl font-semibold leading-[1.12] tracking-[-0.04em] text-white md:text-4xl [text-wrap:balance]"
        >
          {{ t('home.clientShowcase.title') }}
        </h2>

        <p class="mt-4 max-w-[44rem] text-base leading-7 text-white/55">
          {{ t('home.clientShowcase.description') }}
        </p>
      </div>

      <!-- Feature pills -->
      <div class="relative z-10 mt-8 flex flex-wrap gap-2.5">
        <span
          v-for="pill in pills"
          :key="pill"
          class="rounded-full border border-white/8 bg-white/[0.04] px-3.5 py-1.5 text-sm text-white/70"
        >
          {{ pill }}
        </span>
      </div>

      <!-- Screenshot -->
      <div class="relative z-10 mt-10">
        <div
          class="relative overflow-hidden rounded-2xl border border-white/10 shadow-[0_0_80px_rgba(13,148,136,0.12)]"
        >
          <!-- Browser chrome bar -->
          <div
            class="flex items-center gap-2 border-b border-white/8 bg-white/[0.03] px-4 py-2.5"
          >
            <div class="flex gap-1.5">
              <span class="h-3 w-3 rounded-full bg-[#ff5f57]/80"></span>
              <span class="h-3 w-3 rounded-full bg-[#febc2e]/80"></span>
              <span class="h-3 w-3 rounded-full bg-[#28c840]/80"></span>
            </div>
            <div
              class="ml-4 flex flex-1 items-center justify-center rounded-md border border-white/8 bg-white/[0.04] px-3 py-1 text-xs text-white/30"
            >
              cheaprouter.cloud
            </div>
          </div>

          <!-- Product image -->
          <div class="relative bg-[#0d0d0f]">
            <img
              src="/product.png"
              alt="CheapRouter Client"
              class="w-full"
              loading="lazy"
            />
            <!-- Bottom gradient fade -->
            <div
              class="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-[#0a0a0c] to-transparent"
            ></div>
          </div>
        </div>

        <!-- Caption -->
        <p class="mt-4 text-center text-xs text-white/25">
          {{ t('home.clientShowcase.caption') }}
        </p>
      </div>

      <!-- CTA row -->
      <div class="relative z-10 mt-10 flex flex-col items-start gap-4 sm:flex-row sm:items-center">
        <router-link
          to="/register"
          class="inline-flex h-12 items-center gap-2 rounded-full bg-white px-6 text-sm font-semibold text-[#0a0a0c] transition hover:bg-white/90"
        >
          <span>{{ t('home.clientShowcase.cta') }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>

        <p class="text-sm text-white/30">
          {{ t('home.clientShowcase.ctaNote') }}
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const pills = computed(() => [
  t('home.clientShowcase.pills.darkMode'),
  t('home.clientShowcase.pills.workspace'),
  t('home.clientShowcase.pills.terminal'),
  t('home.clientShowcase.pills.crossDevice'),
])
</script>
