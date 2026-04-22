<template>
  <section class="mx-auto max-w-[1380px]">
    <div class="rounded-[34px] border border-black/10 bg-white/70 p-6 shadow-[0_20px_70px_rgba(15,15,15,0.05)] backdrop-blur dark:border-white/10 dark:bg-white/5 md:p-10">
      <!-- Header -->
      <div class="flex flex-col items-start justify-between gap-6 md:flex-row md:items-end">
        <div>
          <p class="text-[11px] uppercase tracking-[0.24em] text-[#7a7268] dark:text-white/42">
            {{ t('home.pricingTable.overline') }}
          </p>
          <h2 class="mt-3 text-3xl font-semibold leading-[1.12] tracking-[-0.04em] text-[#111111] dark:text-white md:text-4xl [text-wrap:balance]">
            {{ t('home.pricingTable.title') }}
          </h2>
          <p class="mt-3 max-w-[44rem] text-base leading-7 text-[#5f5850] dark:text-white/68">
            {{ t('home.pricingTable.description') }}
          </p>
        </div>
        <div class="shrink-0 rounded-2xl border border-primary-200 bg-primary-50 px-5 py-3 dark:border-primary-800/50 dark:bg-primary-900/30">
          <div class="text-[11px] uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
            {{ t('home.pricingTable.badge') }}
          </div>
          <div class="mt-0.5 text-2xl font-bold tracking-[-0.04em] text-primary-700 dark:text-primary-300">
            {{ t('home.pricingTable.badgeValue') }}
          </div>
        </div>
      </div>

      <!-- Tab filter -->
      <div class="mt-8 flex flex-wrap gap-2">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="rounded-full px-4 py-1.5 text-sm font-medium transition"
          :class="
            activeTab === tab.key
              ? 'bg-[#111] text-white dark:bg-white dark:text-[#111]'
              : 'border border-black/12 text-[#666] hover:border-black/20 hover:text-[#111] dark:border-white/15 dark:text-white/52 dark:hover:border-white/30 dark:hover:text-white'
          "
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- Table -->
      <div class="mt-5 overflow-hidden rounded-[22px] border border-black/8 dark:border-white/10">
        <!-- Header row -->
        <div class="hidden grid-cols-[minmax(0,2fr)_120px_120px_120px_100px] bg-[#16181d] px-6 py-3.5 text-xs font-semibold uppercase tracking-[0.14em] text-white/60 md:grid">
          <div>{{ t('home.pricingTable.col.model') }}</div>
          <div class="text-right">{{ t('home.pricingTable.col.officialInput') }}</div>
          <div class="text-right">{{ t('home.pricingTable.col.ourInput') }}</div>
          <div class="text-right">{{ t('home.pricingTable.col.officialOutput') }}</div>
          <div class="text-center">{{ t('home.pricingTable.col.savings') }}</div>
        </div>

        <div class="divide-y divide-black/6 bg-white dark:divide-white/8 dark:bg-[#14181f]">
          <template v-for="group in filteredGroups" :key="group.platform">
            <!-- Platform group header -->
            <div class="flex items-center gap-3 bg-[#f8f6f3]/80 px-6 py-2.5 dark:bg-white/3">
              <span class="text-[11px] font-semibold uppercase tracking-[0.2em] text-[#8b8378] dark:text-white/38">
                {{ group.platform }}
              </span>
              <span class="rounded-full bg-black/6 px-2 py-0.5 text-[10px] text-[#666] dark:bg-white/10 dark:text-white/38">
                {{ group.models.length }} {{ t('home.pricingTable.modelsCount') }}
              </span>
            </div>

            <!-- Model rows -->
            <div
              v-for="model in group.models"
              :key="model.id"
              class="grid gap-y-2 px-5 py-4 transition-colors hover:bg-black/[0.018] md:grid-cols-[minmax(0,2fr)_120px_120px_120px_100px] md:items-center md:px-6 dark:hover:bg-white/[0.02]"
            >
              <!-- Model name + tags -->
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium text-[#111] dark:text-white">{{ model.name }}</span>
                <span
                  v-if="model.tag"
                  class="rounded-full px-2 py-0.5 text-[10px] font-semibold"
                  :class="tagClass(model.tag)"
                >
                  {{ model.tag }}
                </span>
                <span class="hidden text-xs text-[#aaa] dark:text-white/28 md:inline">{{ model.id }}</span>
              </div>

              <!-- Official input price -->
              <div class="flex items-center justify-between md:block md:text-right">
                <span class="text-xs text-[#aaa] dark:text-white/28 md:hidden">{{ t('home.pricingTable.col.officialInput') }}</span>
                <span class="text-sm text-[#888] line-through dark:text-white/38">${{ model.officialInput }}</span>
              </div>

              <!-- Our input price -->
              <div class="flex items-center justify-between md:block md:text-right">
                <span class="text-xs text-[#aaa] dark:text-white/28 md:hidden">{{ t('home.pricingTable.col.ourInput') }}</span>
                <span class="text-sm font-semibold text-primary-700 dark:text-primary-300">{{ isChinese ? '¥' : '$' }}{{ isChinese ? model.ourInput : model.ourInputUsd }}</span>
              </div>

              <!-- Official output price -->
              <div class="flex items-center justify-between md:block md:text-right">
                <span class="text-xs text-[#aaa] dark:text-white/28 md:hidden">{{ t('home.pricingTable.col.officialOutput') }}</span>
                <span class="text-sm text-[#888] line-through dark:text-white/38">${{ model.officialOutput }}</span>
              </div>

              <!-- Savings badge -->
              <div class="flex justify-start md:justify-center">
                <span class="rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-bold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
                  -{{ model.savings }}%
                </span>
              </div>
            </div>
          </template>
        </div>
      </div>

      <!-- Footer note -->
      <p class="mt-4 text-[12px] leading-6 text-[#999] dark:text-white/30">
        {{ t('home.pricingTable.note') }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

const isChinese = computed(() => locale.value === 'zh')

const activeTab = ref('all')

const tabs = computed(() => [
  { key: 'all', label: t('home.pricingTable.tabs.all') },
  { key: 'claude', label: 'Claude' },
  { key: 'openai', label: 'Codex / GPT' },
])

// GLM分组 (0.25× multiplier, ¥0.70/$1 platform rate → actual CNY = official_USD × 0.25 × 0.70)
// ourInput: CNY per M tokens  ourInputUsd: actual USD per M tokens (ourInput ÷ 7)
const allGroups = [
  {
    platform: 'Claude',
    tab: 'claude',
    models: [
      {
        id: 'claude-haiku-4-5-20251001',
        name: 'Claude Haiku 4.5',
        tag: 'fast',
        officialInput: '1.00',
        officialOutput: '5.00',
        ourInput: '0.175',
        ourInputUsd: '0.025',
        savings: 97,
      },
      {
        id: 'claude-sonnet-4-5-20250929',
        name: 'Claude Sonnet 4.5',
        tag: null,
        officialInput: '3.00',
        officialOutput: '15.00',
        ourInput: '0.525',
        ourInputUsd: '0.075',
        savings: 97,
      },
      {
        id: 'claude-sonnet-4-6',
        name: 'Claude Sonnet 4.6',
        tag: 'coding',
        officialInput: '3.00',
        officialOutput: '15.00',
        ourInput: '0.525',
        ourInputUsd: '0.075',
        savings: 97,
      },
      {
        id: 'claude-opus-4-5-20251101',
        name: 'Claude Opus 4.5',
        tag: null,
        officialInput: '5.00',
        officialOutput: '25.00',
        ourInput: '0.875',
        ourInputUsd: '0.125',
        savings: 97,
      },
      {
        id: 'claude-opus-4-6',
        name: 'Claude Opus 4.6',
        tag: 'flagship',
        officialInput: '5.00',
        officialOutput: '25.00',
        ourInput: '0.875',
        ourInputUsd: '0.125',
        savings: 97,
      },
      {
        id: 'claude-opus-4-7',
        name: 'Claude Opus 4.7',
        tag: 'new',
        officialInput: '5.00',
        officialOutput: '25.00',
        ourInput: '0.875',
        ourInputUsd: '0.125',
        savings: 97,
      },
    ],
  },
  {
    platform: 'Codex / GPT',
    tab: 'openai',
    models: [
      {
        id: 'gpt-5.4',
        name: 'GPT-5.4',
        tag: 'flagship',
        officialInput: '2.50',
        officialOutput: '15.00',
        ourInput: '1.400',
        ourInputUsd: '0.200',
        savings: 92,
      },
      {
        id: 'gpt-5.3-codex',
        name: 'GPT-5.3 Codex',
        tag: 'coding',
        officialInput: '1.75',
        officialOutput: '14.00',
        ourInput: '0.980',
        ourInputUsd: '0.140',
        savings: 92,
      },
      {
        id: 'gpt-5.2',
        name: 'GPT-5.2',
        tag: null,
        officialInput: '1.75',
        officialOutput: '14.00',
        ourInput: '0.980',
        ourInputUsd: '0.140',
        savings: 92,
      },
    ],
  },
]

const filteredGroups = computed(() => {
  if (activeTab.value === 'all') return allGroups
  return allGroups.filter(g => g.tab === activeTab.value)
})

function tagClass(tag: string | null) {
  switch (tag) {
    case 'flagship':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'coding':
      return 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
    case 'fast':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'new':
      return 'bg-violet-50 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400'
    default:
      return 'bg-[#f0ede8] text-[#666] dark:bg-white/8 dark:text-white/40'
  }
}
</script>
