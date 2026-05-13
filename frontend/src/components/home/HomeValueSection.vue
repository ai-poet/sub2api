<template>
  <section class="mx-auto max-w-[1380px]">
    <div class="max-w-[980px]">
      <p class="text-[11px] uppercase tracking-[0.24em] text-[#7a7268] dark:text-white/42">
        {{ t('home.value.overline') }}
      </p>
      <h2 class="mt-4 max-w-[16ch] text-4xl font-semibold leading-[1.1] tracking-[-0.04em] text-[#111111] dark:text-white md:text-5xl [text-wrap:balance]">
        {{ t('home.value.title') }}
      </h2>
      <p class="mt-5 max-w-[48rem] text-lg leading-8 text-[#5f5850] dark:text-white/68">
        {{ t('home.value.description') }}
      </p>
    </div>

    <div class="mt-14 grid gap-10 border-t border-black/10 pt-8 dark:border-white/10 lg:grid-cols-3 lg:gap-8">
      <article
        v-for="item in valueItems"
        :key="item.title"
        class="border-b border-black/10 pb-8 dark:border-white/10 lg:border-b-0 lg:pb-0"
      >
        <div class="flex items-center justify-between gap-4">
          <p class="text-[11px] uppercase tracking-[0.22em] text-[#7a7268] dark:text-white/38">
            {{ item.eyebrow }}
          </p>
          <div class="flex h-11 w-11 items-center justify-center rounded-full border border-black/10 bg-white/70 text-[#161616] dark:border-white/10 dark:bg-white/5 dark:text-white">
            <Icon :name="item.icon" size="md" />
          </div>
        </div>

        <h3 class="mt-8 text-2xl font-semibold leading-tight tracking-[-0.04em] text-[#111111] dark:text-white">
          {{ item.title }}
        </h3>
        <p class="mt-4 text-base leading-7 text-[#5f5850] dark:text-white/65">
          {{ item.description }}
        </p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const valueItems = computed(() => [
  {
    eyebrow: t('home.value.items.economics.eyebrow'),
    title: t('home.value.items.economics.title'),
    description: t('home.value.items.economics.description'),
    icon: 'dollar' as const,
  },
  {
    eyebrow: t('home.value.items.reliability.eyebrow'),
    title: t('home.value.items.reliability.title'),
    description: t('home.value.items.reliability.description'),
    icon: 'server' as const,
  },
  {
    eyebrow: t('home.value.items.control.eyebrow'),
    title: t('home.value.items.control.title'),
    description: t('home.value.items.control.description'),
    icon: 'chartBar' as const,
  },
])
</script>
