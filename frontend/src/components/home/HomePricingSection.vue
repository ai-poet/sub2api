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
            {{ badgeLabel }}
          </div>
          <div class="mt-0.5 text-2xl font-bold tracking-[-0.04em] text-primary-700 dark:text-primary-300">
            {{ badgeValue }}
          </div>
        </div>
      </div>

      <!-- Real pricing table (public endpoint). Falls back to the static cards below
           whenever it is unavailable, so the landing page is never empty or broken. -->
      <div v-if="hasPricing" class="mt-10">
        <div
          v-for="section in platformSections"
          :key="section.platform"
          class="mb-8 last:mb-0"
        >
          <div class="mb-3 flex items-baseline gap-2">
            <h3 class="text-sm font-semibold uppercase tracking-[0.14em] text-[#111] dark:text-white">
              {{ section.platform }}
            </h3>
            <span class="text-xs text-[#7a7268] dark:text-white/45">
              {{ t('home.pricingTable.table.modelCount', { count: section.rows.length }) }}
            </span>
          </div>

          <!-- Wide table scrolls inside its own container; the page body never scrolls sideways. -->
          <div class="overflow-x-auto rounded-2xl border border-black/8 bg-white dark:border-white/10 dark:bg-white/[0.03]">
            <table class="w-full min-w-[46rem] border-collapse text-left text-sm">
              <thead>
                <tr class="border-b border-black/8 text-[11px] uppercase tracking-[0.12em] text-[#7a7268] dark:border-white/10 dark:text-white/45">
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('home.pricingTable.table.model') }}</th>
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('home.pricingTable.table.group') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('home.pricingTable.table.input') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('home.pricingTable.table.output') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('home.pricingTable.table.cacheWrite') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('home.pricingTable.table.cacheRead') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('home.pricingTable.table.vsOfficial') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in section.rows"
                  :key="`${row.platform}-${row.model}-${row.group.name}`"
                  class="border-b border-black/5 last:border-0 dark:border-white/5"
                >
                  <td class="px-4 py-3 font-medium text-[#111] dark:text-white">{{ row.display_name }}</td>
                  <td class="px-4 py-3 text-[#5f5850] dark:text-white/65">
                    <span>{{ row.group.name }}</span>
                    <span
                      v-if="row.group.rate_multiplier !== 1"
                      class="ml-1.5 rounded-full bg-[#f7f3ee] px-1.5 py-0.5 font-mono text-[11px] text-[#3d362e] dark:bg-white/10 dark:text-white/70"
                    >×{{ row.group.rate_multiplier }}</span>
                  </td>

                  <!-- Token billing: four per-1M-token columns. -->
                  <template v-if="isTokenBilling(row)">
                    <td class="px-4 py-3 text-right font-mono text-[#111] dark:text-white">{{ perMTok(row.effective_pricing_usd.input_per_mtok_usd) }}</td>
                    <td class="px-4 py-3 text-right font-mono text-[#111] dark:text-white">{{ perMTok(row.effective_pricing_usd.output_per_mtok_usd) }}</td>
                    <td class="px-4 py-3 text-right font-mono text-[#5f5850] dark:text-white/65">{{ perMTok(row.effective_pricing_usd.cache_write_per_mtok_usd) }}</td>
                    <td class="px-4 py-3 text-right font-mono text-[#5f5850] dark:text-white/65">{{ perMTok(row.effective_pricing_usd.cache_read_per_mtok_usd) }}</td>
                  </template>

                  <!-- Per-request / per-image billing: a single merged price cell. -->
                  <td v-else colspan="4" class="px-4 py-3 text-right font-mono text-[#111] dark:text-white">
                    {{ flatPrice(row) }}
                  </td>

                  <td class="px-4 py-3 text-right">
                    <span
                      v-if="savingsLabel(row)"
                      class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
                    >{{ savingsLabel(row) }}</span>
                    <span v-else class="text-xs text-[#7a7268] dark:text-white/40">—</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <button
          v-if="canExpand"
          type="button"
          class="mt-2 rounded-full border border-black/10 px-4 py-2 text-sm text-[#3d362e] transition hover:bg-black/5 dark:border-white/15 dark:text-white/75 dark:hover:bg-white/5"
          @click="expanded = !expanded"
        >
          {{ expanded ? t('home.pricingTable.table.collapse') : t('home.pricingTable.table.expand', { count: hiddenCount }) }}
        </button>
      </div>

      <!-- Fallback: static copy cards (loading, request failure, feature off, or no data) -->
      <div v-else class="mt-10 grid gap-5 md:grid-cols-2 lg:grid-cols-3">
        <article
          v-for="card in cards"
          :key="card.key"
          class="relative overflow-hidden rounded-[26px] border border-black/8 bg-white p-6 shadow-[0_2px_20px_rgba(0,0,0,0.04)] transition hover:-translate-y-[2px] hover:shadow-[0_18px_48px_rgba(13,148,136,0.12)] dark:border-white/10 dark:bg-white/[0.03]"
        >
          <!-- Decorative gradient -->
          <div class="pointer-events-none absolute right-[-3rem] top-[-3rem] h-32 w-32 rounded-full bg-primary-500/10 blur-3xl"></div>

          <div class="relative">
            <div class="flex items-center gap-2">
              <span class="rounded-full bg-[#111] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-white dark:bg-white dark:text-[#111]">
                {{ card.tag }}
              </span>
            </div>

            <h3 class="mt-5 text-lg font-semibold tracking-tight text-[#111] dark:text-white">
              {{ card.title }}
            </h3>
            <p class="mt-2 text-sm leading-6 text-[#5f5850] dark:text-white/65">
              {{ card.description }}
            </p>

            <div class="mt-5 flex flex-wrap gap-2">
              <span
                v-for="model in card.models"
                :key="model"
                class="rounded-full border border-black/8 bg-[#f7f3ee] px-2.5 py-1 text-xs text-[#3d362e] dark:border-white/10 dark:bg-white/5 dark:text-white/70"
              >
                {{ model }}
              </span>
            </div>
          </div>
        </article>
      </div>

      <p class="mt-8 text-xs leading-6 text-[#7a7268] dark:text-white/45">
        {{ t('home.pricingTable.note') }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPublicPricing, type PublicPricingItem } from '@/api/pricing'
import { formatScaled } from '@/utils/pricing'

const { t } = useI18n()

/** 首屏默认展示的模型条数，避免落地页被上百行价目表撑爆。 */
const COLLAPSED_LIMIT = 12

const items = ref<PublicPricingItem[]>([])
const expanded = ref(false)
let controller: AbortController | null = null

/**
 * 同一个模型可能出现在多个公开分组下。后端已按「省得多 → 单价低」全局排序，
 * 因此每个模型第一次出现时对应的就是它的最优分组，去重时保留首个即可。
 */
const bestPerModel = computed(() => {
  const seen = new Set<string>()
  const out: PublicPricingItem[] = []
  for (const item of items.value) {
    const key = `${item.platform}::${item.model}`.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(item)
  }
  return out
})

const hasPricing = computed(() => bestPerModel.value.length > 0)

const visibleItems = computed(() =>
  expanded.value ? bestPerModel.value : bestPerModel.value.slice(0, COLLAPSED_LIMIT)
)

const hiddenCount = computed(() => Math.max(0, bestPerModel.value.length - COLLAPSED_LIMIT))
const canExpand = computed(() => hiddenCount.value > 0)

/** 按平台分组，平台内保持后端给出的排序。 */
const platformSections = computed(() => {
  const buckets = new Map<string, PublicPricingItem[]>()
  for (const item of visibleItems.value) {
    const platform = item.platform || 'other'
    const bucket = buckets.get(platform)
    if (bucket) bucket.push(item)
    else buckets.set(platform, [item])
  }
  return [...buckets.entries()].map(([platform, rows]) => ({ platform, rows }))
})

/** 有真实数据且后端算出了最高省钱比例时，徽标改为展示这个数字。 */
const maxSavings = computed(() => {
  let max = 0
  for (const item of bestPerModel.value) {
    const s = item.comparison.savings_percent
    if (s != null && s > max) max = s
  }
  return max
})

const badgeLabel = computed(() =>
  hasPricing.value && maxSavings.value > 0
    ? t('home.pricingTable.badgeSavings')
    : t('home.pricingTable.badge')
)

const badgeValue = computed(() =>
  hasPricing.value && maxSavings.value > 0
    ? t('home.pricingTable.badgeSavingsValue', { percent: maxSavings.value.toFixed(0) })
    : t('home.pricingTable.badgeValue')
)

function isTokenBilling(row: PublicPricingItem): boolean {
  return row.billing_mode === 'token' || row.billing_mode === ''
}

/** 每 100 万 token 的价格。后端下发的 *_per_mtok_usd 已是「每 100 万 token 美元数」，
 *  这里只做格式化，不再换算（scale = 1）。 */
function perMTok(value: number | null): string {
  return value == null ? '—' : formatScaled(value, 1, 2)
}

/** 按次 / 按张计费模型的单价。 */
function flatPrice(row: PublicPricingItem): string {
  const pricing = row.effective_pricing_usd
  if (pricing.per_image_usd != null) {
    return t('home.pricingTable.table.perImage', { price: formatScaled(pricing.per_image_usd, 1, 2) })
  }
  if (pricing.per_request_usd != null) {
    return t('home.pricingTable.table.perRequest', { price: formatScaled(pricing.per_request_usd, 1, 2) })
  }
  return '—'
}

/** 只有后端确实下发了省钱比例（即存在官方参考价）时才展示对比。 */
function savingsLabel(row: PublicPricingItem): string {
  const percent = row.comparison.savings_percent
  if (percent == null || percent <= 0) return ''
  return t('home.pricingTable.table.savings', { percent: percent.toFixed(0) })
}

onMounted(async () => {
  controller = new AbortController()
  try {
    const res = await getPublicPricing({ signal: controller.signal })
    items.value = res?.items ?? []
  } catch {
    // 落地页对访客必须永远可用：拉取失败就静默回落到静态文案卡片。
    items.value = []
  }
})

onUnmounted(() => {
  controller?.abort()
  controller = null
})

const cards = computed(() => [
  {
    key: 'claude',
    tag: t('home.pricingTable.cards.claude.tag'),
    title: t('home.pricingTable.cards.claude.title'),
    description: t('home.pricingTable.cards.claude.description'),
    models: ['Claude Sonnet 4.6', 'Claude Opus 4.7', 'Claude Haiku 4.5'],
  },
  {
    key: 'codex',
    tag: t('home.pricingTable.cards.codex.tag'),
    title: t('home.pricingTable.cards.codex.title'),
    description: t('home.pricingTable.cards.codex.description'),
    models: ['GPT-5.5', 'GPT-5.4', 'GPT-5.3 Codex'],
  },
  {
    key: 'compatible',
    tag: t('home.pricingTable.cards.compatible.tag'),
    title: t('home.pricingTable.cards.compatible.title'),
    description: t('home.pricingTable.cards.compatible.description'),
    models: [t('home.providers.openaiCompatible'), 'Gemini', 'GLM', 'Qwen'],
  },
])
</script>
