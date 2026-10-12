<template>
  <!-- 模型卡片（参考 TokenRouter 模型广场）：标题 + 能力图标，ID 行，价格预览，展开查看详细定价 -->
  <article class="catalog-model-card group" data-testid="catalog-model-card" :data-model="item.model">
    <div class="flex min-w-0 flex-wrap items-start justify-between gap-2">
      <div class="flex min-w-0 flex-1 basis-32 items-center">
        <h3 class="min-w-0 truncate text-base font-semibold text-gray-950 dark:text-white" :title="title">{{ title }}</h3>
        <HelpTooltip v-if="infoLines.length" width-class="w-64" class="shrink-0">
          <template #trigger>
            <span
              class="inline-flex h-5 w-5 cursor-help items-center justify-center rounded-full text-gray-400 transition-colors hover:text-gray-700 dark:text-gray-500 dark:hover:text-gray-200"
              data-testid="model-info-trigger"
            >
              <Icon name="infoCircle" size="sm" />
            </span>
          </template>
          <dl class="space-y-1.5">
            <div v-for="line in infoLines" :key="line.label" class="flex items-baseline justify-between gap-3">
              <dt class="text-gray-300">{{ line.label }}</dt>
              <dd class="text-right font-medium tabular-nums text-white">{{ line.value }}</dd>
            </div>
          </dl>
        </HelpTooltip>
      </div>
      <ModelModalityIcons :model="item.model" :billing-mode="item.billing_mode" />
    </div>

    <ModelIdLabel :model-id="item.model" class="mt-1" />

    <div class="mt-4">
      <dl v-if="compactRows.length" class="space-y-2">
        <div v-for="row in compactRows" :key="row.key" class="flex items-baseline justify-between gap-3 text-sm">
          <dt class="shrink-0 text-gray-500 dark:text-gray-400">{{ row.label }}</dt>
          <dd class="min-w-0 break-words text-right font-medium tabular-nums text-gray-900 [overflow-wrap:anywhere] dark:text-white">
            {{ row.value }}
          </dd>
        </div>
      </dl>
      <p v-else class="text-sm text-gray-400 dark:text-gray-500">{{ t('modelCatalog.card.pricingUnavailable') }}</p>

      <!-- 展开 / 收起详细定价：箭头随状态旋转，内容用 Collapse 平滑展开 -->
      <button
        type="button"
        class="mt-3 flex w-full items-center justify-between gap-2 border-t border-gray-100 pt-3 text-sm font-medium text-primary-600 transition-colors hover:text-primary-700 dark:border-dark-700 dark:text-primary-300 dark:hover:text-primary-200"
        data-testid="model-pricing-toggle"
        :aria-expanded="expanded"
        @click="expanded = !expanded"
      >
        <span class="inline-flex items-center gap-1.5">
          <Icon name="eye" size="sm" />
          {{ expanded ? t('modelCatalog.card.collapsePricing') : t('modelCatalog.card.viewPricing') }}
        </span>
        <Icon name="chevronDown" size="sm" class="transition-transform duration-normal ease-standard" :class="{ 'rotate-180': expanded }" />
      </button>

      <Collapse :open="expanded">
        <div class="pt-3" data-testid="model-pricing-panel">
          <div v-if="intervals.length > 1" class="mb-3 flex justify-end">
            <div v-segmented class="segmented max-w-full flex-wrap" data-testid="pricing-interval-switch">
              <button
                v-for="(interval, index) in intervals"
                :key="`${interval.min_tokens}-${interval.max_tokens}`"
                type="button"
                class="segmented-item px-2 py-0.5 text-xs font-semibold tabular-nums"
                :class="{ 'segmented-item-active': index === activeInterval }"
                @click="activeInterval = index"
              >
                {{ formatTokenRange(interval.min_tokens, interval.max_tokens) }}
              </button>
            </div>
          </div>

          <div class="space-y-2.5">
            <div
              v-for="row in detailRows"
              :key="row.key"
              class="flex items-baseline justify-between gap-3 border-b border-gray-100 pb-2 text-sm dark:border-dark-700"
            >
              <span class="min-w-0 max-w-[45%] shrink-0 break-words text-gray-500 dark:text-gray-400">
                {{ row.label }}
                <span v-if="row.hint" class="block text-[11px] text-gray-400 dark:text-gray-500">{{ row.hint }}</span>
              </span>
              <span class="min-w-0 text-right">
                <span class="block font-medium tabular-nums text-gray-900 [overflow-wrap:anywhere] dark:text-white">{{ row.value }}</span>
                <span v-if="row.official" class="block text-[11px] tabular-nums text-gray-400 line-through dark:text-gray-500">
                  {{ t('modelCatalog.card.official', { price: row.official }) }}
                </span>
              </span>
            </div>
          </div>

          <div v-if="badges.length" class="mt-3 flex flex-wrap gap-1.5">
            <span v-for="badge in badges" :key="badge.key" class="rounded-full px-2 py-0.5 text-[11px] font-semibold" :class="badge.tone">
              {{ badge.text }}
            </span>
          </div>
        </div>
      </Collapse>
    </div>
  </article>
</template>

<script setup lang="ts">
// fork：模型广场的模型卡片（参考 TokenRouter ModelMarketplaceView 的卡片与 ModelPricingPanel）。
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import Collapse from '@/components/motion/Collapse.vue'
import { vSegmented } from '@/directives/segmented'
import type { ModelCatalogItem, ModelCatalogPricing } from '@/api/modelCatalog'
import ModelIdLabel from './ModelIdLabel.vue'
import ModelModalityIcons from './ModelModalityIcons.vue'
import { prettifyModelName } from './catalogGroups'
import { formatCharge, formatTokenCount, formatTokenRange, formatUsd, type CatalogPriceContext } from './catalogPricing'

interface PriceRow {
  key: string
  label: string
  value: string
  hint?: string
  official?: string
}

const props = defineProps<{
  item: ModelCatalogItem
  priceContext: CatalogPriceContext
}>()

const { t } = useI18n()
const expanded = ref(false)
const activeInterval = ref(0)

const title = computed(() => prettifyModelName(props.item.display_name || props.item.model))
const intervals = computed(() => (props.item.billing_mode === 'token' ? props.item.pricing_details.intervals : []))
const mediaTiers = computed(() => props.item.pricing_details.media_tiers ?? [])

const charge = (usd: number | null | undefined) => formatCharge(usd, props.priceContext)
const perMTok = (usd: number | null | undefined) => `${charge(usd)} ${t('modelCatalog.card.perMTok')}`

function mediaUnit(): string {
  const unit = props.item.pricing_details.media_unit || (props.item.billing_mode === 'video' ? 'second' : 'image')
  return unit === 'second' ? t('modelCatalog.card.perSecond') : t('modelCatalog.card.perImage')
}

/** 卡片上的价格预览：token 模型显示输入 / 输出（有上下文分档时每档一行），图片 / 视频按档位，按次计费一行。 */
const compactRows = computed<PriceRow[]>(() => {
  const item = props.item
  const effective = item.effective_pricing_usd

  if (item.billing_mode === 'per_request') {
    return effective.per_request_usd == null
      ? []
      : [{ key: 'per_request', label: t('modelCatalog.card.perRequestLabel'), value: `${charge(effective.per_request_usd)} ${t('modelCatalog.card.perRequest')}` }]
  }

  if (item.billing_mode === 'image' || item.billing_mode === 'video') {
    if (mediaTiers.value.length > 0) {
      return mediaTiers.value.map(tier => ({ key: tier.tier, label: tier.tier, value: `${charge(tier.effective_usd)} ${mediaUnit()}` }))
    }
    const usd = item.billing_mode === 'image' ? effective.per_image_usd : effective.per_second_usd
    return usd == null ? [] : [{ key: 'media', label: t(`modelCatalog.billingMode.${item.billing_mode}`), value: `${charge(usd)} ${mediaUnit()}` }]
  }

  if (intervals.value.length > 0) {
    return intervals.value.map((interval, index) => ({
      key: `interval-${index}`,
      label: formatTokenRange(interval.min_tokens, interval.max_tokens),
      value: [
        interval.input_per_mtok_usd != null ? `${t('modelCatalog.priceLabels.input')} ${charge(interval.input_per_mtok_usd)}` : '',
        interval.output_per_mtok_usd != null ? `${t('modelCatalog.priceLabels.output')} ${charge(interval.output_per_mtok_usd)}` : '',
      ].filter(Boolean).join(' / '),
    }))
  }

  const rows: PriceRow[] = []
  if (effective.input_per_mtok_usd != null) rows.push({ key: 'input', label: t('modelCatalog.priceLabels.input'), value: perMTok(effective.input_per_mtok_usd) })
  if (effective.output_per_mtok_usd != null) rows.push({ key: 'output', label: t('modelCatalog.priceLabels.output'), value: perMTok(effective.output_per_mtok_usd) })
  return rows
})

function tokenRows(pricing: Pick<ModelCatalogPricing, 'input_per_mtok_usd' | 'output_per_mtok_usd' | 'cache_write_per_mtok_usd' | 'cache_read_per_mtok_usd'>, official?: ModelCatalogPricing): PriceRow[] {
  const fields = [
    ['input', 'input_per_mtok_usd'],
    ['output', 'output_per_mtok_usd'],
    ['cacheWrite', 'cache_write_per_mtok_usd'],
    ['cacheRead', 'cache_read_per_mtok_usd'],
  ] as const
  return fields.flatMap(([label, field]) => {
    const value = pricing[field]
    if (value == null) return []
    const reference = official?.has_reference ? official[field] : null
    return [{
      key: field,
      label: t(`modelCatalog.priceLabels.${label}`),
      value: perMTok(value),
      official: reference != null && reference !== value ? `${formatUsd(reference)} ${t('modelCatalog.card.perMTok')}` : undefined,
    }]
  })
}

/** 展开后的完整定价：所有价格项、官方参考价对比、默认档位说明。 */
const detailRows = computed<PriceRow[]>(() => {
  const item = props.item
  const official = item.official_pricing

  if (item.billing_mode === 'per_request') {
    return [{
      key: 'per_request',
      label: t('modelCatalog.card.perRequestLabel'),
      value: `${charge(item.effective_pricing_usd.per_request_usd)} ${t('modelCatalog.card.perRequest')}`,
      official: official.has_reference && official.per_request_usd != null ? formatUsd(official.per_request_usd) : undefined,
    }]
  }

  if (item.billing_mode === 'image' || item.billing_mode === 'video') {
    if (mediaTiers.value.length > 0) {
      return mediaTiers.value.map(tier => ({
        key: tier.tier,
        label: tier.tier,
        value: `${charge(tier.effective_usd)} ${mediaUnit()}`,
        hint: tier.is_default_tier ? t('modelCatalog.defaultTierHint') : undefined,
        official: tier.official_usd != null && tier.official_usd !== tier.effective_usd ? formatUsd(tier.official_usd) : undefined,
      }))
    }
    return compactRows.value
  }

  if (intervals.value.length > 0) {
    const interval = intervals.value[Math.min(activeInterval.value, intervals.value.length - 1)]
    return tokenRows(interval)
  }
  return tokenRows(item.effective_pricing_usd, official)
})

const badges = computed(() => {
  const item = props.item
  const list: Array<{ key: string; text: string; tone: string }> = []
  const savings = item.official_pricing.has_reference ? item.comparison.savings_percent : null
  if (typeof savings === 'number' && savings > 0.005) {
    list.push({ key: 'savings', text: t('modelCatalog.card.savings', { percent: Math.round(savings * 100) }), tone: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' })
  }
  if (item.pricing_details.supports_prompt_caching) {
    list.push({ key: 'cache', text: t('modelCatalog.capabilities.promptCaching'), tone: 'bg-cyan-50 text-cyan-700 dark:bg-cyan-500/10 dark:text-cyan-300' })
  }
  if (item.best_group.rate_source === 'user_override') {
    list.push({ key: 'override', text: t('modelCatalog.capabilities.userRateOverride'), tone: 'bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300' })
  }
  return list
})

/** 标题旁信息浮层：上下文、长上下文计价、参考价来源。 */
const infoLines = computed(() => {
  const item = props.item
  const lines: Array<{ label: string; value: string }> = []
  if (item.context_window) lines.push({ label: t('modelCatalog.card.contextWindow'), value: formatTokenCount(item.context_window) })
  if (item.pricing_details.has_long_context_multiplier) {
    lines.push({ label: t('modelCatalog.card.longContextFrom'), value: formatTokenCount(item.pricing_details.long_context_input_threshold) })
  }
  lines.push({ label: t('modelCatalog.card.promptCaching'), value: item.pricing_details.supports_prompt_caching ? t('common.yes') : t('common.no') })
  if (item.official_pricing.source === 'litellm' || item.official_pricing.source === 'fallback') {
    lines.push({ label: t('modelCatalog.card.reference'), value: t(`modelCatalog.referenceSource.${item.official_pricing.source}`) })
  }
  return lines
})
</script>

<style scoped>
/* 与 TokenRouter 一致：浅底卡片，悬浮轻抬 2px、描边加深 */
.catalog-model-card {
  @apply min-w-0 max-w-full rounded-xl border border-gray-100 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-950/80;
  transition:
    transform var(--motion-normal) var(--motion-ease),
    border-color var(--motion-normal) var(--motion-ease),
    box-shadow var(--motion-normal) var(--motion-ease);
}

.catalog-model-card:hover {
  @apply -translate-y-0.5 border-black/20 shadow-sm dark:border-primary-500/50;
}

@media (prefers-reduced-motion: reduce) {
  .catalog-model-card:hover {
    transform: none;
  }
}
</style>
