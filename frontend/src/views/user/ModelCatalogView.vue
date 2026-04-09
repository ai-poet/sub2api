<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] space-y-6">
      <section class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div class="space-y-2">
          <div class="inline-flex items-center gap-2 rounded-full bg-cyan-50 px-3 py-1 text-xs font-semibold text-cyan-700 dark:bg-cyan-950/30 dark:text-cyan-300">
            <Icon name="grid" size="sm" />
            {{ t('modelCatalog.caption') }}
          </div>
          <p class="max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
            {{ t('modelCatalog.intro') }}
          </p>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('modelCatalog.sortNotice') }}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <div class="rounded-xl border border-gray-200 bg-white/85 px-4 py-2 text-right shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/75">
            <div class="text-[11px] uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
              {{ t('modelCatalog.lastUpdated') }}
            </div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
              {{ lastUpdatedLabel }}
            </div>
          </div>

          <button type="button" class="btn btn-secondary" :disabled="loading || refreshing" @click="refreshCatalog">
            <span
              v-if="loading || refreshing"
              class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
            ></span>
            <Icon name="refresh" size="sm" class="mr-2" />
            {{ t('common.refresh') }}
          </button>
        </div>
      </section>

      <div
        v-if="paymentConfigNotice"
        class="rounded-2xl border border-dashed border-amber-300 bg-amber-50/80 px-4 py-3 text-sm text-amber-800 dark:border-amber-700/60 dark:bg-amber-950/20 dark:text-amber-200"
      >
        <div class="flex items-start gap-3">
          <Icon name="infoCircle" size="sm" class="mt-0.5 flex-shrink-0" />
          <div>
            <div class="font-medium">{{ t('modelCatalog.paymentNoticeTitle') }}</div>
            <div class="mt-1 text-amber-700/90 dark:text-amber-200/90">
              {{ paymentConfigNotice }}
            </div>
          </div>
        </div>
      </div>

      <section class="grid gap-4 xl:grid-cols-4">
        <article class="summary-card summary-card-cyan">
          <div class="summary-kicker">{{ t('modelCatalog.summary.visibleCards') }}</div>
          <div class="summary-value">{{ visibleItems.length }}</div>
          <div class="summary-note">
            {{ t('modelCatalog.summary.totalCards', { total: summary?.total_models ?? allItems.length }) }}
          </div>
        </article>

        <article class="summary-card summary-card-emerald">
          <div class="summary-kicker">{{ t('modelCatalog.summary.lowestInput') }}</div>
          <div class="summary-value summary-value-sm">
            {{ lowestInputSummary?.item.display_name || '—' }}
          </div>
          <div class="summary-note">
            {{
              lowestInputSummary
                ? `${lowestInputSummary.item.best_group.name} · ${formatUsd(lowestInputSummary.price)}`
                : '—'
            }}
          </div>
        </article>

        <article class="summary-card summary-card-rose">
          <div class="summary-kicker">{{ t('modelCatalog.summary.maxSavings') }}</div>
          <div class="summary-value">
            {{ maxSavingsSummary ? formatPercent(maxSavingsSummary.savings) : '—' }}
          </div>
          <div class="summary-note">
            {{
              maxSavingsSummary
                ? `${maxSavingsSummary.item.display_name} · ${maxSavingsSummary.item.best_group.name}`
                : t('modelCatalog.summary.noSavingsReference')
            }}
          </div>
        </article>

        <article class="summary-card summary-card-amber">
          <div class="summary-kicker">{{ t('modelCatalog.summary.cachingCount') }}</div>
          <div class="summary-value">{{ cachingCount }}</div>
          <div class="summary-note">
            {{ t('modelCatalog.summary.tokenAndNonToken', { token: tokenCount, nonToken: nonTokenCount }) }}
          </div>
        </article>
      </section>

      <section class="rounded-3xl border border-gray-200/80 bg-white/90 p-4 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80">
        <div class="flex flex-col gap-2 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <div class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('modelCatalog.groupTabs.title') }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('modelCatalog.groupTabs.description') }}
            </div>
          </div>

          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('modelCatalog.groupTabs.currentGroup', { group: activeGroupTabLabel }) }}
          </div>
        </div>

        <div class="mt-4 flex flex-wrap gap-2">
          <button
            v-for="tab in groupTabs"
            :key="tab.id == null ? 'all' : tab.id"
            type="button"
            class="group-tab"
            :class="selectedGroupId === tab.id ? 'group-tab-active' : 'group-tab-inactive'"
            @click="selectedGroupId = tab.id"
          >
            <span class="truncate">{{ tab.name }}</span>
            <span class="group-tab-count">
              {{ tab.count }}
            </span>
          </button>
        </div>
      </section>

      <section class="rounded-3xl border border-gray-200/80 bg-white/90 p-4 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80">
        <div class="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_repeat(4,minmax(0,0.8fr))]">
          <div>
            <label class="input-label">{{ t('modelCatalog.filters.search') }}</label>
            <SearchInput
              v-model="search"
              :placeholder="t('modelCatalog.filters.searchPlaceholder')"
              :debounce-ms="150"
            />
          </div>

          <div>
            <label class="input-label">{{ t('modelCatalog.filters.platform') }}</label>
            <select v-model="selectedPlatform" class="input">
              <option value="all">{{ t('modelCatalog.filters.allPlatforms') }}</option>
              <option v-for="platform in platformOptions" :key="platform" :value="platform">
                {{ platformLabel(platform) }}
              </option>
            </select>
          </div>

          <div>
            <label class="input-label">{{ t('modelCatalog.filters.billingMode') }}</label>
            <select v-model="selectedBillingMode" class="input">
              <option value="all">{{ t('modelCatalog.filters.allBillingModes') }}</option>
              <option value="token">{{ t('modelCatalog.billingMode.token') }}</option>
              <option value="per_request">{{ t('modelCatalog.billingMode.perRequest') }}</option>
              <option value="image">{{ t('modelCatalog.billingMode.image') }}</option>
            </select>
          </div>

          <div class="flex items-end">
            <label class="flex w-full items-center gap-3 rounded-2xl border border-gray-200 bg-gray-50/90 px-4 py-3 text-sm font-medium text-gray-700 dark:border-dark-700 dark:bg-dark-800/80 dark:text-gray-200">
              <input
                v-model="onlySavings"
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <span>{{ t('modelCatalog.filters.onlySavings') }}</span>
            </label>
          </div>

          <div>
            <label class="input-label">{{ t('modelCatalog.filters.sortBy') }}</label>
            <select v-model="sortBy" class="input">
              <option value="savings_desc">{{ t('modelCatalog.sorting.savingsDesc') }}</option>
              <option value="effective_price_asc">{{ t('modelCatalog.sorting.effectivePriceAsc') }}</option>
              <option value="model_asc">{{ t('modelCatalog.sorting.modelAsc') }}</option>
            </select>
          </div>
        </div>

        <div class="mt-4 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2.5 py-1 dark:bg-dark-800">
            <Icon name="sort" size="xs" />
            {{ t('modelCatalog.filterResult', { visible: visibleItems.length, total: allItems.length }) }}
          </span>
          <span class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2.5 py-1 dark:bg-dark-800">
            <Icon name="dollar" size="xs" />
            {{ t('modelCatalog.priceBasis') }}
          </span>
          <span
            v-if="balanceCreditCnyPerUsd"
            class="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-1 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300"
          >
            <Icon name="calculator" size="xs" />
            {{ t('modelCatalog.cnyRateReady', { rate: balanceCreditCnyPerUsd.toFixed(2) }) }}
          </span>
        </div>
      </section>

      <div
        v-if="loading && allItems.length === 0"
        class="card flex items-center justify-center py-20"
      >
        <div class="h-10 w-10 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <section
        v-else-if="visibleItems.length === 0"
        class="card p-12 text-center"
      >
        <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
          <Icon name="grid" size="lg" class="text-gray-400" />
        </div>
        <h3 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t(loadError ? 'modelCatalog.loadFailedTitle' : 'modelCatalog.emptyTitle') }}
        </h3>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ loadError || t('modelCatalog.emptyDescription') }}
        </p>
      </section>

      <section v-else class="grid gap-4 xl:grid-cols-2 2xl:grid-cols-3">
        <article
          v-for="item in visibleItems"
          :key="getCatalogItemKey(item)"
          class="model-card"
          :class="getCardClass(item)"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <span class="platform-pill" :class="getPlatformBadgeClass(item.platform)">
                <PlatformIcon :platform="item.platform as any" size="xs" />
                {{ platformLabel(item.platform) }}
              </span>
              <span class="mode-pill" :class="getModeBadgeClass(item.billing_mode)">
                {{ billingModeLabel(item.billing_mode) }}
              </span>
              <span class="group-pill">
                {{ item.best_group.name }}
              </span>
            </div>

            <span class="savings-pill" :class="getSavingsBadgeClass(item)">
              {{ savingsBadgeText(item) }}
            </span>
          </div>

          <div class="mt-4 flex items-start justify-between gap-4">
            <div class="min-w-0">
              <h3 class="truncate text-xl font-semibold tracking-tight text-gray-900 dark:text-white">
                {{ item.display_name }}
              </h3>
              <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-gray-500 dark:text-gray-400">
                <span>{{ t('modelCatalog.groupRateLabel', { rate: formatRate(item.best_group.rate_multiplier) }) }}</span>
                <span class="text-gray-300 dark:text-dark-500">•</span>
                <span>{{ rateSourceLabel(item.best_group.rate_source) }}</span>
                <template v-if="item.available_group_count > 1">
                  <span class="text-gray-300 dark:text-dark-500">•</span>
                  <span>{{ t('modelCatalog.peerGroupsLabel', { count: item.available_group_count }) }}</span>
                </template>
              </div>
            </div>

            <div class="hidden rounded-2xl border border-gray-200/80 bg-gray-50/90 px-4 py-3 text-right shadow-inner dark:border-dark-700 dark:bg-dark-800/80 md:block">
              <div class="text-[11px] uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
                {{ t('modelCatalog.primaryPrice') }}
              </div>
              <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                {{ formatUsd(getPrimaryEffectivePrice(item)) }}
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ billingUnitLabel(item.billing_mode) }}
              </div>
            </div>
          </div>

          <div class="mt-5 space-y-3">
            <div
              v-for="row in buildPriceRows(item)"
              :key="row.key"
              class="rounded-2xl border border-gray-200/80 bg-gray-50/80 p-3 dark:border-dark-700 dark:bg-dark-800/75"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ row.label }}
                  </div>
                  <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ row.unit }}
                  </div>
                </div>

                <div
                  class="grid min-w-0 flex-1 gap-2"
                  :class="balanceCreditCnyPerUsd ? 'md:grid-cols-3' : 'md:grid-cols-2'"
                >
                  <div class="price-stat">
                    <div class="price-stat-label">{{ t('modelCatalog.priceColumns.official') }}</div>
                    <div class="price-stat-value">{{ formatUsd(row.officialUsd) }}</div>
                  </div>

                  <div class="price-stat price-stat-strong">
                    <div class="price-stat-label">{{ t('modelCatalog.priceColumns.balance') }}</div>
                    <div class="price-stat-value">{{ formatUsd(row.balanceUsd) }}</div>
                  </div>

                  <div v-if="balanceCreditCnyPerUsd" class="price-stat price-stat-cny">
                    <div class="price-stat-label">{{ t('modelCatalog.priceColumns.cash') }}</div>
                    <div class="price-stat-value">{{ formatCny(row.actualCny) }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-2">
            <span v-if="item.pricing_details.supports_prompt_caching" class="cap-badge cap-badge-cyan">
              {{ t('modelCatalog.capabilities.promptCaching') }}
            </span>
            <span v-if="item.pricing_details.has_long_context_multiplier" class="cap-badge cap-badge-amber">
              {{ t('modelCatalog.capabilities.longContext', { threshold: formatThreshold(item.pricing_details.long_context_input_threshold) }) }}
            </span>
            <span v-if="item.pricing_details.intervals.length" class="cap-badge cap-badge-slate">
              {{ t('modelCatalog.capabilities.tieredPricing', { count: item.pricing_details.intervals.length }) }}
            </span>
            <span v-if="item.official_pricing.has_reference" class="cap-badge cap-badge-emerald">
              {{ officialSourceLabel(item.official_pricing.source) }}
            </span>
            <span v-if="item.best_group.rate_source === 'user_override'" class="cap-badge cap-badge-rose">
              {{ t('modelCatalog.capabilities.userRateOverride') }}
            </span>
          </div>

          <div v-if="hasExpandableContent(item)" class="mt-4 border-t border-gray-100 pt-4 dark:border-dark-800">
            <button
              type="button"
              class="inline-flex items-center gap-2 text-sm font-medium text-gray-700 transition-colors hover:text-gray-900 dark:text-gray-300 dark:hover:text-white"
              @click="toggleExpanded(item)"
            >
              <Icon :name="isExpanded(item) ? 'chevronUp' : 'chevronDown'" size="sm" />
              {{ isExpanded(item) ? t('modelCatalog.collapseDetails') : t('modelCatalog.expandDetails') }}
            </button>

            <div v-if="isExpanded(item)" class="mt-4 grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
              <div
                v-if="item.pricing_details.intervals.length"
                class="rounded-2xl border border-gray-200/80 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800/70"
              >
                <div class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('modelCatalog.intervalSectionTitle') }}
                </div>
                <div class="mt-3 space-y-3">
                  <div
                    v-for="interval in item.pricing_details.intervals"
                    :key="`${interval.tier_label}-${interval.min_tokens}-${interval.max_tokens}`"
                    class="rounded-2xl border border-white/70 bg-white/80 p-3 dark:border-dark-700 dark:bg-dark-900/75"
                  >
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <div class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ formatIntervalRange(interval) }}
                      </div>
                      <div class="text-xs text-gray-500 dark:text-gray-400">
                        {{ interval.tier_label || t('modelCatalog.intervalDefaultLabel') }}
                      </div>
                    </div>
                    <div class="mt-3 grid gap-2 sm:grid-cols-2">
                      <div
                        v-for="detail in buildIntervalDetails(interval, item.billing_mode)"
                        :key="detail.key"
                        class="rounded-xl bg-gray-50 px-3 py-2 text-xs dark:bg-dark-800"
                      >
                        <div class="text-gray-500 dark:text-gray-400">{{ detail.label }}</div>
                        <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                          {{ formatUsd(detail.value) }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div
                v-if="item.other_groups.length"
                class="rounded-2xl border border-gray-200/80 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800/70"
              >
                <div class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('modelCatalog.otherGroupsTitle') }}
                </div>
                <div class="mt-3 space-y-3">
                  <div
                    v-for="other in item.other_groups"
                    :key="`${other.group.id}-${item.model}`"
                    class="rounded-2xl border border-white/70 bg-white/80 p-3 dark:border-dark-700 dark:bg-dark-900/75"
                  >
                    <div class="flex items-start justify-between gap-3">
                      <div>
                        <div class="font-medium text-gray-900 dark:text-white">{{ other.group.name }}</div>
                        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                          {{ t('modelCatalog.groupRateLabel', { rate: formatRate(other.group.rate_multiplier) }) }}
                        </div>
                      </div>
                      <span
                        class="savings-pill text-[11px]"
                        :class="getPeerSavingsBadgeClass(getPeerDisplaySavingsPercent(item, other.effective_pricing_usd))"
                      >
                        {{ peerSavingsText(getPeerDisplaySavingsPercent(item, other.effective_pricing_usd)) }}
                      </span>
                    </div>

                    <div class="mt-3 rounded-xl bg-gray-50 px-3 py-2 text-sm dark:bg-dark-800">
                      <div class="text-xs text-gray-500 dark:text-gray-400">
                        {{ t('modelCatalog.peerEffectivePrice') }}
                      </div>
                      <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                        {{ formatUsd(getPrimaryPrice(other.effective_pricing_usd, item.billing_mode)) }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import {
  convertUsdAmountToCny,
  fetchBalanceCreditCnyPerUsd,
  filterModelCatalogItems,
  getCatalogItemKey,
  getPrimaryEffectivePrice,
  getPrimaryPrice,
  modelCatalogAPI,
  sortModelCatalogItems,
  type ModelCatalogBillingMode,
  type ModelCatalogItem,
  type ModelCatalogPriceInterval,
  type ModelCatalogPricing,
  type ModelCatalogSortKey,
  type ModelCatalogSummary,
} from '@/api/modelCatalog'

interface PriceRow {
  key: string
  label: string
  unit: string
  officialUsd: number | null
  balanceUsd: number | null
  actualCny: number | null
}

interface IntervalDetail {
  key: string
  label: string
  value: number | null
}

interface GroupTab {
  id: number | null
  name: string
  count: number
}

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const allItems = ref<ModelCatalogItem[]>([])
const summary = ref<ModelCatalogSummary | null>(null)
const loading = ref(true)
const refreshing = ref(false)
const loadError = ref('')
const balanceCreditCnyPerUsd = ref<number | null>(null)
const paymentConfigError = ref<string | null>(null)
const lastUpdatedAt = ref<Date | null>(null)
const expandedKeys = ref<Set<string>>(new Set())

const search = ref('')
const selectedGroupId = ref<number | null>(null)
const selectedPlatform = ref('all')
const selectedBillingMode = ref('all')
const onlySavings = ref(false)
const sortBy = ref<ModelCatalogSortKey>('savings_desc')

const groupTabs = computed<GroupTab[]>(() => {
  const counts = new Map<number, GroupTab>()

  for (const item of allItems.value) {
    const current = counts.get(item.best_group.id)
    if (current) {
      current.count += 1
      continue
    }

    counts.set(item.best_group.id, {
      id: item.best_group.id,
      name: item.best_group.name,
      count: 1,
    })
  }

  const groups = [...counts.values()].sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))

  return [
    {
      id: null,
      name: t('modelCatalog.groupTabs.allGroups'),
      count: allItems.value.length,
    },
    ...groups,
  ]
})

const activeGroupTabLabel = computed(() => {
  return groupTabs.value.find(tab => tab.id === selectedGroupId.value)?.name || t('modelCatalog.groupTabs.allGroups')
})

const platformOptions = computed(() => {
  return [...new Set(allItems.value.map(item => item.platform).filter(Boolean))].sort((a, b) =>
    a.localeCompare(b, undefined, { sensitivity: 'base' }),
  )
})

const filteredItems = computed(() =>
  filterModelCatalogItems(allItems.value, {
    search: search.value,
    groupId: selectedGroupId.value,
    platform: selectedPlatform.value,
    billingMode: selectedBillingMode.value,
    onlySavings: onlySavings.value,
  }),
)

const visibleItems = computed(() => sortModelCatalogItems(filteredItems.value, sortBy.value))

const lowestInputSummary = computed(() => {
  const tokenItems = visibleItems.value
    .filter(item => item.billing_mode === 'token' && item.effective_pricing_usd.input_per_mtok_usd != null)
    .map(item => ({
      item,
      price: item.effective_pricing_usd.input_per_mtok_usd as number,
    }))

  if (tokenItems.length === 0) {
    return null
  }

  tokenItems.sort((a, b) => a.price - b.price)
  return tokenItems[0]
})

const maxSavingsSummary = computed(() => {
  const itemsWithSavings = visibleItems.value
    .map(item => ({
      item,
      savings: getDisplaySavingsPercent(item),
    }))
    .filter((entry): entry is { item: ModelCatalogItem; savings: number } => entry.savings != null)
    .map(item => ({
      item: item.item,
      savings: item.savings,
    }))

  if (itemsWithSavings.length === 0) {
    return null
  }

  itemsWithSavings.sort((a, b) => b.savings - a.savings)
  return itemsWithSavings[0]
})

const cachingCount = computed(() =>
  visibleItems.value.filter(item => item.pricing_details.supports_prompt_caching).length,
)

const tokenCount = computed(() => visibleItems.value.filter(item => item.billing_mode === 'token').length)
const nonTokenCount = computed(() => visibleItems.value.length - tokenCount.value)

const lastUpdatedLabel = computed(() => {
  if (!lastUpdatedAt.value) {
    return t('modelCatalog.neverUpdated')
  }
  return lastUpdatedAt.value.toLocaleString()
})

const paymentConfigNotice = computed(() => {
  if (loading.value || refreshing.value) {
    return ''
  }
  if (!paymentConfigError.value) {
    return ''
  }
  return t('modelCatalog.paymentNoticeDescription')
})

onMounted(async () => {
  await loadCatalog()
})

async function loadCatalog(options: { refresh?: boolean } = {}) {
  if (options.refresh) {
    refreshing.value = true
  } else {
    loading.value = true
  }

  loadError.value = ''

  try {
    if (!appStore.publicSettingsLoaded) {
      await appStore.fetchPublicSettings()
    }

    const response = await modelCatalogAPI.getCatalog()
    allItems.value = response.items || []
    if (selectedGroupId.value != null && !allItems.value.some(item => item.best_group.id === selectedGroupId.value)) {
      selectedGroupId.value = null
    }
    summary.value = response.summary || null
    lastUpdatedAt.value = new Date()
    await loadPaymentConfig()
  } catch (error) {
    console.error('Failed to load model catalog:', error)
    loadError.value = t('modelCatalog.loadFailedDescription')
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function loadPaymentConfig() {
  const result = await fetchBalanceCreditCnyPerUsd({
    purchaseSubscriptionUrl: appStore.cachedPublicSettings?.purchase_subscription_url,
    userId: authStore.user?.id,
    token: authStore.token,
    locale: locale.value,
  })

  balanceCreditCnyPerUsd.value = result.balanceCreditCnyPerUsd
  paymentConfigError.value = result.error
}

async function refreshCatalog() {
  await loadCatalog({ refresh: true })
}

function isExpanded(item: ModelCatalogItem): boolean {
  return expandedKeys.value.has(getCatalogItemKey(item))
}

function toggleExpanded(item: ModelCatalogItem) {
  const key = getCatalogItemKey(item)
  const next = new Set(expandedKeys.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  expandedKeys.value = next
}

function hasExpandableContent(item: ModelCatalogItem): boolean {
  return item.pricing_details.intervals.length > 0 || item.other_groups.length > 0
}

function buildPriceRows(item: ModelCatalogItem): PriceRow[] {
  const toCny = (value: number | null) => convertUsdAmountToCny(value, balanceCreditCnyPerUsd.value)

  if (item.billing_mode === 'per_request') {
    return [
      {
        key: 'per_request',
        label: t('modelCatalog.priceLabels.perRequest'),
        unit: t('modelCatalog.units.perRequest'),
        officialUsd: item.official_pricing.per_request_usd,
        balanceUsd: item.effective_pricing_usd.per_request_usd,
        actualCny: toCny(item.effective_pricing_usd.per_request_usd),
      },
    ]
  }

  if (item.billing_mode === 'image') {
    return [
      {
        key: 'per_image',
        label: t('modelCatalog.priceLabels.perImage'),
        unit: t('modelCatalog.units.perImage'),
        officialUsd: item.official_pricing.per_image_usd,
        balanceUsd: item.effective_pricing_usd.per_image_usd,
        actualCny: toCny(item.effective_pricing_usd.per_image_usd),
      },
    ]
  }

  return [
    {
      key: 'input',
      label: t('modelCatalog.priceLabels.input'),
      unit: t('modelCatalog.units.perMillionTokens'),
      officialUsd: item.official_pricing.input_per_mtok_usd,
      balanceUsd: item.effective_pricing_usd.input_per_mtok_usd,
      actualCny: toCny(item.effective_pricing_usd.input_per_mtok_usd),
    },
    {
      key: 'output',
      label: t('modelCatalog.priceLabels.output'),
      unit: t('modelCatalog.units.perMillionTokens'),
      officialUsd: item.official_pricing.output_per_mtok_usd,
      balanceUsd: item.effective_pricing_usd.output_per_mtok_usd,
      actualCny: toCny(item.effective_pricing_usd.output_per_mtok_usd),
    },
    {
      key: 'cache_write',
      label: t('modelCatalog.priceLabels.cacheWrite'),
      unit: t('modelCatalog.units.perMillionTokens'),
      officialUsd: item.official_pricing.cache_write_per_mtok_usd,
      balanceUsd: item.effective_pricing_usd.cache_write_per_mtok_usd,
      actualCny: toCny(item.effective_pricing_usd.cache_write_per_mtok_usd),
    },
    {
      key: 'cache_read',
      label: t('modelCatalog.priceLabels.cacheRead'),
      unit: t('modelCatalog.units.perMillionTokens'),
      officialUsd: item.official_pricing.cache_read_per_mtok_usd,
      balanceUsd: item.effective_pricing_usd.cache_read_per_mtok_usd,
      actualCny: toCny(item.effective_pricing_usd.cache_read_per_mtok_usd),
    },
  ]
}

function buildIntervalDetails(
  interval: ModelCatalogPriceInterval,
  billingMode: ModelCatalogBillingMode,
): IntervalDetail[] {
  if (billingMode === 'per_request') {
    return [{
      key: 'per_request',
      label: t('modelCatalog.priceLabels.perRequest'),
      value: interval.per_request_usd,
    }]
  }

  if (billingMode === 'image') {
    return [{
      key: 'per_image',
      label: t('modelCatalog.priceLabels.perImage'),
      value: interval.per_image_usd,
    }]
  }

  return [
    {
      key: 'input',
      label: t('modelCatalog.priceLabels.input'),
      value: interval.input_per_mtok_usd,
    },
    {
      key: 'output',
      label: t('modelCatalog.priceLabels.output'),
      value: interval.output_per_mtok_usd,
    },
    {
      key: 'cache_write',
      label: t('modelCatalog.priceLabels.cacheWrite'),
      value: interval.cache_write_per_mtok_usd,
    },
    {
      key: 'cache_read',
      label: t('modelCatalog.priceLabels.cacheRead'),
      value: interval.cache_read_per_mtok_usd,
    },
  ].filter(detail => detail.value != null)
}

function billingModeLabel(mode: ModelCatalogBillingMode): string {
  if (mode === 'per_request') return t('modelCatalog.billingMode.perRequest')
  if (mode === 'image') return t('modelCatalog.billingMode.image')
  return t('modelCatalog.billingMode.token')
}

function billingUnitLabel(mode: ModelCatalogBillingMode): string {
  if (mode === 'per_request') return t('modelCatalog.units.perRequest')
  if (mode === 'image') return t('modelCatalog.units.perImage')
  return t('modelCatalog.units.perMillionTokens')
}

function platformLabel(platform: string): string {
  return t(`admin.groups.platforms.${platform}`)
}

function rateSourceLabel(source: string): string {
  if (source === 'user_override') {
    return t('modelCatalog.rateSource.userOverride')
  }
  return t('modelCatalog.rateSource.groupDefault')
}

function officialSourceLabel(source: string): string {
  if (source === 'litellm') {
    return t('modelCatalog.referenceSource.litellm')
  }
  if (source === 'fallback') {
    return t('modelCatalog.referenceSource.fallback')
  }
  return t('modelCatalog.referenceSource.none')
}

function formatUsd(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '—'
  }
  return `$${formatNumber(value)}`
}

function formatCny(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '—'
  }
  return `¥${formatNumber(value)}`
}

function formatPercent(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '—'
  }
  return `${(value * 100).toFixed(Math.abs(value) >= 0.1 ? 1 : 2)}%`
}

function formatRate(value: number): string {
  return `${value.toFixed(value >= 1 ? 2 : 3)}x`
}

function formatThreshold(value: number): string {
  if (!value) {
    return '—'
  }
  if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(1)}M`
  }
  if (value >= 1_000) {
    return `${(value / 1_000).toFixed(0)}K`
  }
  return `${value}`
}

function formatIntervalRange(interval: ModelCatalogPriceInterval): string {
  if (interval.max_tokens == null) {
    return `≥ ${formatThreshold(interval.min_tokens)}`
  }
  if (interval.min_tokens === 0) {
    return `0 - ${formatThreshold(interval.max_tokens)}`
  }
  return `${formatThreshold(interval.min_tokens)} - ${formatThreshold(interval.max_tokens)}`
}

function getDisplaySavingsPercent(item: ModelCatalogItem): number | null {
  const officialUsd = getPrimaryPrice(item.official_pricing, item.billing_mode)
  const effectiveUsd = getPrimaryPrice(item.effective_pricing_usd, item.billing_mode)

  if (balanceCreditCnyPerUsd.value != null) {
    const officialCny = convertUsdAmountToCny(officialUsd, balanceCreditCnyPerUsd.value)
    const actualCny = convertUsdAmountToCny(effectiveUsd, balanceCreditCnyPerUsd.value)
    return calculateDisplaySavingsPercent(officialCny, actualCny)
  }

  return calculateDisplaySavingsPercent(officialUsd, effectiveUsd)
}

function getPeerDisplaySavingsPercent(
  item: ModelCatalogItem,
  effectivePricingUsd: ModelCatalogPricing,
): number | null {
  const officialUsd = getPrimaryPrice(item.official_pricing, item.billing_mode)
  const effectiveUsd = getPrimaryPrice(effectivePricingUsd, item.billing_mode)

  if (balanceCreditCnyPerUsd.value != null) {
    const officialCny = convertUsdAmountToCny(officialUsd, balanceCreditCnyPerUsd.value)
    const actualCny = convertUsdAmountToCny(effectiveUsd, balanceCreditCnyPerUsd.value)
    return calculateDisplaySavingsPercent(officialCny, actualCny)
  }

  return calculateDisplaySavingsPercent(officialUsd, effectiveUsd)
}

function calculateDisplaySavingsPercent(official: number | null, actual: number | null): number | null {
  if (official == null || actual == null || !Number.isFinite(official) || !Number.isFinite(actual) || official <= 0) {
    return null
  }

  const savings = 1 - actual / official
  if (savings <= 0) {
    return 0
  }
  return savings
}

function savingsBadgeText(item: ModelCatalogItem): string {
  const savings = getDisplaySavingsPercent(item)
  if (savings == null) {
    return t('modelCatalog.badges.noReference')
  }
  if (Math.abs(savings) < 1e-9) {
    return t('modelCatalog.badges.sameAsOfficial')
  }
  return t('modelCatalog.badges.saving', { percent: formatPercent(savings) })
}

function peerSavingsText(savings: number | null): string {
  if (savings == null) {
    return t('modelCatalog.badges.noReference')
  }
  if (Math.abs(savings) < 1e-9) {
    return t('modelCatalog.badges.sameAsOfficial')
  }
  return t('modelCatalog.badges.saving', { percent: formatPercent(savings) })
}

function getCardClass(item: ModelCatalogItem): string {
  if (item.billing_mode === 'image') {
    return 'model-card-image'
  }
  if (item.billing_mode === 'per_request') {
    return 'model-card-request'
  }
  return 'model-card-token'
}

function getPlatformBadgeClass(platform: string): string {
  if (platform === 'anthropic') return 'platform-pill-anthropic'
  if (platform === 'openai') return 'platform-pill-openai'
  if (platform === 'antigravity') return 'platform-pill-antigravity'
  return 'platform-pill-gemini'
}

function getModeBadgeClass(mode: ModelCatalogBillingMode): string {
  if (mode === 'image') return 'mode-pill-image'
  if (mode === 'per_request') return 'mode-pill-request'
  return 'mode-pill-token'
}

function getSavingsBadgeClass(item: ModelCatalogItem): string {
  return getPeerSavingsBadgeClass(getDisplaySavingsPercent(item))
}

function getPeerSavingsBadgeClass(savings: number | null): string {
  if (savings == null) return 'savings-pill-neutral'
  if (Math.abs(savings) < 1e-9) return 'savings-pill-neutral'
  return 'savings-pill-positive'
}

function formatNumber(value: number): string {
  if (value === 0) {
    return '0.0000'
  }
  if (Math.abs(value) >= 100) {
    return value.toFixed(2)
  }
  if (Math.abs(value) >= 10) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}
</script>

<style scoped>
.summary-card {
  @apply relative overflow-hidden rounded-3xl border border-gray-200/80 bg-white/90 p-5 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80;
}

.summary-card::before {
  content: '';
  @apply absolute inset-x-0 top-0 h-1.5;
}

.summary-card-cyan::before {
  @apply bg-gradient-to-r from-cyan-500 to-sky-400;
}

.summary-card-emerald::before {
  @apply bg-gradient-to-r from-emerald-500 to-teal-400;
}

.summary-card-rose::before {
  @apply bg-gradient-to-r from-rose-500 to-orange-400;
}

.summary-card-amber::before {
  @apply bg-gradient-to-r from-amber-500 to-yellow-400;
}

.summary-kicker {
  @apply text-[11px] font-semibold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400;
}

.summary-value {
  @apply mt-3 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white;
}

.summary-value-sm {
  @apply text-2xl leading-tight;
}

.summary-note {
  @apply mt-2 text-sm text-gray-500 dark:text-gray-400;
}

.model-card {
  @apply relative overflow-hidden rounded-3xl border border-gray-200/80 bg-white/95 p-5 shadow-sm backdrop-blur transition-all duration-200 dark:border-dark-700 dark:bg-dark-900/80;
}

.model-card::before {
  content: '';
  @apply absolute inset-x-0 top-0 h-1.5;
}

.model-card-token::before {
  @apply bg-gradient-to-r from-cyan-500 via-sky-400 to-emerald-400;
}

.model-card-request::before {
  @apply bg-gradient-to-r from-amber-500 via-orange-400 to-rose-400;
}

.model-card-image::before {
  @apply bg-gradient-to-r from-fuchsia-500 via-rose-400 to-orange-400;
}

.platform-pill,
.mode-pill,
.group-pill,
.savings-pill,
.cap-badge {
  @apply inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-semibold;
}

.group-pill {
  @apply bg-gray-100 text-gray-700 dark:bg-dark-800 dark:text-gray-200;
}

.platform-pill-anthropic {
  @apply bg-orange-100 text-orange-700 dark:bg-orange-950/40 dark:text-orange-300;
}

.platform-pill-openai {
  @apply bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300;
}

.platform-pill-antigravity {
  @apply bg-violet-100 text-violet-700 dark:bg-violet-950/40 dark:text-violet-300;
}

.platform-pill-gemini {
  @apply bg-blue-100 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300;
}

.mode-pill-token {
  @apply bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200;
}

.mode-pill-request {
  @apply bg-amber-100 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300;
}

.mode-pill-image {
  @apply bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300;
}

.savings-pill-positive {
  @apply bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300;
}

.savings-pill-neutral {
  @apply bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-gray-300;
}

.group-tab {
  @apply inline-flex max-w-full items-center gap-2 rounded-2xl border px-3 py-2 text-sm font-medium transition-colors;
}

.group-tab-active {
  @apply border-cyan-200 bg-cyan-50 text-cyan-700 shadow-sm dark:border-cyan-900/60 dark:bg-cyan-950/30 dark:text-cyan-300;
}

.group-tab-inactive {
  @apply border-gray-200 bg-gray-50/80 text-gray-600 hover:border-gray-300 hover:bg-white hover:text-gray-900 dark:border-dark-700 dark:bg-dark-800/80 dark:text-gray-300 dark:hover:border-dark-600 dark:hover:bg-dark-800 dark:hover:text-white;
}

.group-tab-count {
  @apply rounded-full bg-white/80 px-2 py-0.5 text-[11px] font-semibold text-gray-500 dark:bg-dark-900/80 dark:text-gray-300;
}

.price-stat {
  @apply rounded-2xl border border-gray-200 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-900;
}

.price-stat-strong {
  @apply border-sky-200 bg-sky-50/70 dark:border-sky-900/50 dark:bg-sky-950/20;
}

.price-stat-cny {
  @apply border-amber-200 bg-amber-50/80 dark:border-amber-900/50 dark:bg-amber-950/20;
}

.price-stat-label {
  @apply text-[11px] uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400;
}

.price-stat-value {
  @apply mt-1 text-sm font-semibold text-gray-900 dark:text-white;
}

.cap-badge-cyan {
  @apply bg-cyan-50 text-cyan-700 dark:bg-cyan-950/30 dark:text-cyan-300;
}

.cap-badge-amber {
  @apply bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300;
}

.cap-badge-slate {
  @apply bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200;
}

.cap-badge-emerald {
  @apply bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300;
}

.cap-badge-rose {
  @apply bg-rose-50 text-rose-700 dark:bg-rose-950/30 dark:text-rose-300;
}
</style>
