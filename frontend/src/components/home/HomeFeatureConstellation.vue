<template>
  <section data-test="home-features" class="mx-auto max-w-[1200px] px-4 py-20 md:px-6 md:py-28">
    <div class="text-center">
      <h2
        class="text-[clamp(2rem,5.2vw,3.75rem)] font-black leading-[1.08] tracking-[-0.04em] text-gray-900 [text-wrap:balance] dark:text-white"
      >
        <span class="block">{{ t(`home.landing.features.${mode}.titleLead`) }}</span>
        <span class="block">{{ t(`home.landing.features.${mode}.titleTail`) }}</span>
      </h2>
      <p class="mx-auto mt-5 max-w-[38rem] text-base leading-7 text-gray-600 dark:text-white/60">
        {{ t(`home.landing.features.${mode}.subtitle`) }}
      </p>
    </div>

    <div class="mt-12 grid gap-4 md:mt-16 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] md:items-center md:gap-8 lg:gap-14">
      <div data-test="features-left">
        <p v-if="columns" class="feature-column-label md:text-right">{{ t(columns.left) }}</p>
        <ul class="grid gap-4 md:gap-7">
          <li
            v-for="(card, index) in leftCards"
            :key="card.key"
            class="feature-card feature-card-left"
            :class="index === 1 ? 'md:-translate-x-8 lg:-translate-x-14' : 'md:translate-x-2'"
          >
            <HomeFeatureCardBody :card="card" />
          </li>
        </ul>
      </div>

      <!-- 中间的站点标志；移动端放在最上面 -->
      <div class="order-first flex justify-center py-4 md:order-none md:py-0">
        <div class="relative flex h-40 w-40 items-center justify-center md:h-52 md:w-52">
          <span class="absolute inset-0 rounded-full border border-dashed border-black/15 dark:border-white/15"></span>
          <span class="absolute inset-5 rotate-6 rounded-[32px] bg-primary-400/90 dark:bg-primary-500/80"></span>
          <HomeBrandMark
            :site-name="siteName"
            :site-logo="siteLogo"
            class="relative h-24 w-24 -rotate-6 rounded-[26px] text-5xl shadow-[0_18px_40px_rgba(15,17,20,0.25)] md:h-28 md:w-28"
          />
        </div>
      </div>

      <div data-test="features-right">
        <p v-if="columns" class="feature-column-label">{{ t(columns.right) }}</p>
        <ul class="grid gap-4 md:gap-7">
          <li
            v-for="(card, index) in rightCards"
            :key="card.key"
            class="feature-card feature-card-right"
            :class="index === 1 ? 'md:translate-x-8 lg:translate-x-14' : 'md:-translate-x-2'"
          >
            <HomeFeatureCardBody :card="card" />
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeBrandMark from '@/components/home/HomeBrandMark.vue'
import Icon from '@/components/icons/Icon.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']

interface FeatureCard {
  key: string
  icon: IconName
  tone: string
}

const props = defineProps<{
  mode: 'client' | 'api'
  siteName: string
  siteLogo?: string
}>()

const { t } = useI18n()

const TONES = [
  'bg-primary-100 text-primary-700 dark:bg-primary-500/15 dark:text-primary-300',
  'bg-amber-100 text-amber-700 dark:bg-amber-400/15 dark:text-amber-300',
  'bg-sky-100 text-sky-700 dark:bg-sky-400/15 dark:text-sky-300',
  'bg-orange-100 text-orange-700 dark:bg-orange-400/15 dark:text-orange-300',
  'bg-emerald-100 text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-300',
  'bg-rose-100 text-rose-700 dark:bg-rose-400/15 dark:text-rose-300',
]

// 左列永远是中转站的质量监控；有客户端时右列讲客户端，没有时讲接入和计费。
const QUALITY_CARDS: Array<[string, IconName]> = [
  ['quality', 'shield'],
  ['probe', 'chartBar'],
  ['failover', 'swap'],
]

const CARDS: Record<'client' | 'api', { left: Array<[string, IconName]>; right: Array<[string, IconName]> }> = {
  client: {
    left: QUALITY_CARDS,
    right: [
      ['builtinAgent', 'bolt'],
      ['allAgents', 'link'],
      ['images', 'sparkles'],
    ],
  },
  api: {
    left: QUALITY_CARDS,
    right: [
      ['compatible', 'cpu'],
      ['metered', 'dollar'],
      ['invoice', 'document'],
    ],
  },
}

// 两列讲的是两件事时才加列标签
const columns = computed(() =>
  props.mode === 'client'
    ? { left: 'home.landing.features.columns.quality', right: 'home.landing.features.columns.client' }
    : null,
)

function toCards(defs: Array<[string, IconName]>, offset: number): FeatureCard[] {
  return defs.map(([key, icon], index) => ({ key, icon, tone: TONES[(offset + index) % TONES.length] }))
}

const leftCards = computed(() => toCards(CARDS[props.mode].left, 0))
const rightCards = computed(() => toCards(CARDS[props.mode].right, 3))

const HomeFeatureCardBody = defineComponent({
  props: {
    card: { type: Object as PropType<FeatureCard>, required: true },
  },
  setup(cardProps) {
    return () =>
      h('div', { class: 'flex items-start gap-4' }, [
        h(
          'span',
          { class: `flex h-11 w-11 shrink-0 items-center justify-center rounded-xl ${cardProps.card.tone}` },
          [h(Icon, { name: cardProps.card.icon, size: 'md' })],
        ),
        h('div', { class: 'min-w-0' }, [
          h(
            'h3',
            { class: 'text-[15px] font-bold text-gray-900 dark:text-white' },
            t(`home.landing.features.cards.${cardProps.card.key}.title`),
          ),
          h(
            'p',
            { class: 'mt-1 text-sm leading-6 text-gray-600 dark:text-white/60' },
            t(`home.landing.features.cards.${cardProps.card.key}.body`),
          ),
        ]),
      ])
  },
})
</script>

<style scoped>
.feature-column-label {
  @apply mb-4 text-[11px] font-bold uppercase tracking-[0.2em] text-primary-700 md:mb-6 dark:text-primary-300;
}

.feature-card {
  @apply relative rounded-2xl border border-black/10 bg-white p-5 shadow-[0_10px_30px_rgba(15,17,20,0.06)] transition-transform duration-500 dark:border-white/10 dark:bg-[#16181c];
}

/* 宽屏时卡片朝中间的标志伸出一个小角，像对话气泡 */
@media (min-width: 768px) {
  .feature-card::after {
    content: '';
    @apply absolute top-1/2 h-3 w-3 -translate-y-1/2 rotate-45 border-black/10 bg-white dark:border-white/10 dark:bg-[#16181c];
  }

  .feature-card-left::after {
    @apply -right-[7px] border-r border-t;
  }

  .feature-card-right::after {
    @apply -left-[7px] border-b border-l;
  }
}
</style>
