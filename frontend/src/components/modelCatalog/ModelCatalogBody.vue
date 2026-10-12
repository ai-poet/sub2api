<template>
  <div class="mx-auto max-w-[1600px] space-y-6">
    <section class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
      <div class="space-y-2">
        <div class="inline-flex items-center gap-2 rounded-full bg-cyan-50 px-3 py-1 text-xs font-semibold text-cyan-700 dark:bg-cyan-950/30 dark:text-cyan-300">
          <Icon name="grid" size="sm" />
          {{ t('modelCatalog.caption') }}
        </div>
        <p class="max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
          {{ t(publicMode ? 'modelCatalog.public.intro' : 'modelCatalog.intro') }}
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

        <button type="button" class="btn btn-secondary" :disabled="loading || refreshing" @click="emit('refresh')">
          <Icon name="refresh" size="sm" class="mr-2" :class="{ 'animate-spin': loading || refreshing }" />
          {{ t('common.refresh') }}
        </button>
      </div>
    </section>

    <!-- 匿名视图：引导登录查看专属价格 -->
    <div
      v-if="publicMode"
      class="flex flex-col gap-3 rounded-2xl border border-primary-200/70 bg-primary-50/70 px-4 py-3 text-sm text-primary-800 dark:border-primary-800/50 dark:bg-primary-950/20 dark:text-primary-200 sm:flex-row sm:items-center sm:justify-between"
      data-testid="model-catalog-login-cta"
    >
      <div class="flex items-start gap-3">
        <Icon name="sparkles" size="sm" class="mt-0.5 flex-shrink-0" />
        <div>
          <div class="font-medium">{{ t('modelCatalog.public.loginTitle') }}</div>
          <div class="mt-0.5 text-primary-700/90 dark:text-primary-200/80">{{ t('modelCatalog.public.loginDescription') }}</div>
        </div>
      </div>
      <router-link :to="loginTarget" class="btn btn-primary shrink-0 self-start sm:self-auto">
        {{ t('modelCatalog.public.loginAction') }}
      </router-link>
    </div>

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
          {{ t('modelCatalog.groupTabs.currentGroup', { group: tx(activeGroupTabLabel) }) }}
        </div>
      </div>

      <div class="mt-4 flex flex-wrap gap-2">
        <button
          v-for="tab in groupTabs"
          :key="tab.id == null ? 'all' : tab.id"
          type="button"
          class="group-tab"
          :class="selectedGroupId === tab.id ? 'group-tab-active' : 'group-tab-inactive'"
          :aria-pressed="selectedGroupId === tab.id"
          @click="selectedGroupId = tab.id"
        >
          <span class="truncate">{{ tx(tab.name) }}</span>
          <span class="group-tab-count">
            {{ tab.count }}
          </span>
        </button>
      </div>
    </section>

    <section class="rounded-3xl border border-gray-200/80 bg-white/90 p-4 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80">
      <div class="grid gap-4 xl:grid-cols-[minmax(0,1.4fr)_repeat(2,minmax(0,0.8fr))]">
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
          <label class="input-label">{{ t('modelCatalog.filters.sortBy') }}</label>
          <select v-model="sortBy" class="input">
            <option value="effective_price_asc">{{ t('modelCatalog.sorting.effectivePriceAsc') }}</option>
            <option value="model_asc">{{ t('modelCatalog.sorting.modelAsc') }}</option>
          </select>
        </div>
      </div>

      <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
        <!-- 计费模式：分段切换，选中背景在选项之间滑动（directives/segmented.ts） -->
        <div
          v-segmented
          class="segmented max-w-full overflow-x-auto"
          role="group"
          :aria-label="t('modelCatalog.filters.billingMode')"
        >
          <button
            v-for="option in billingModeOptions"
            :key="option.value"
            type="button"
            class="segmented-item whitespace-nowrap px-3 py-1.5 text-xs"
            :class="{ 'segmented-item-active': selectedBillingMode === option.value }"
            :aria-pressed="selectedBillingMode === option.value"
            @click="selectedBillingMode = option.value"
          >
            {{ option.label }}
          </button>
        </div>

        <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2.5 py-1 dark:bg-dark-800">
            <Icon name="sort" size="xs" />
            {{ t('modelCatalog.filterResult', { visible: visibleItems.length, total: items.length }) }}
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
      </div>
    </section>

    <ModelCatalogSkeleton v-if="loading && items.length === 0" />

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
      <button
        v-if="hasActiveFilters"
        type="button"
        class="btn btn-secondary mt-5"
        data-testid="model-catalog-reset-filters"
        @click="resetFilters"
      >
        {{ t('modelCatalog.filters.reset') }}
      </button>
    </section>

    <section v-else class="grid gap-4 xl:grid-cols-2 2xl:grid-cols-3">
      <article
        v-for="(item, index) in visibleItems"
        :key="getCatalogItemKey(item)"
        class="model-card catalog-card-enter"
        :class="getCardClass(item)"
        :style="{ '--enter-i': Math.min(index, 11) }"
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
              {{ tx(item.best_group.name) }}
            </span>
          </div>
          <ModelModalityIcons :model="item.model" :billing-mode="item.billing_mode" class="shrink-0 pt-1" />
        </div>

        <div class="mt-4 flex items-start justify-between gap-4">
          <div class="flex min-w-0 items-start gap-3">
            <span class="vendor-tile">
              <ModelIcon :model="item.model" size="22px" />
            </span>
            <div class="min-w-0">
              <h3 class="truncate text-xl font-semibold tracking-tight text-gray-900 dark:text-white" :title="item.display_name">
                {{ item.display_name }}
              </h3>
              <ModelIdLabel :model-id="item.model" class="mt-1" />
              <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-gray-500 dark:text-gray-400">
                <span>{{ t('modelCatalog.groupRateLabel', { rate: formatRate(item.best_group.rate_multiplier) }) }}</span>
                <template v-if="!publicMode">
                  <span class="text-gray-300 dark:text-dark-500">•</span>
                  <span>{{ rateSourceLabel(item.best_group.rate_source) }}</span>
                </template>
                <template v-if="item.context_window">
                  <span class="text-gray-300 dark:text-dark-500">•</span>
                  <span>{{ t('modelCatalog.contextWindow', { size: formatThreshold(item.context_window) }) }}</span>
                </template>
                <template v-if="item.available_group_count > 1">
                  <span class="text-gray-300 dark:text-dark-500">•</span>
                  <span>{{ t('modelCatalog.peerGroupsLabel', { count: item.available_group_count }) }}</span>
                </template>
              </div>
            </div>
          </div>

          <div class="hidden shrink-0 rounded-2xl border border-gray-200/80 bg-gray-50/90 px-4 py-3 text-right shadow-inner dark:border-dark-700 dark:bg-dark-800/80 md:block">
            <div class="text-[11px] uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
              {{ t('modelCatalog.primaryPrice') }}
            </div>
            <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
              {{ formatPrimaryDisplayPrice(item) }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ primaryPriceSubtitle(item) }}
            </div>
          </div>
        </div>

        <div v-if="availabilityFor(item)" class="mt-4 rounded-2xl border border-gray-200/70 bg-gray-50/60 px-3 py-2 dark:border-dark-700 dark:bg-dark-800/50">
          <div class="mb-1 text-[11px] text-gray-500 dark:text-gray-400">
            {{ t('modelCatalog.availability.title') }}
          </div>
          <GroupAvailabilityBar
            :buckets="availabilityFor(item)!.buckets"
            :availability="availabilityFor(item)!.availability"
            :total-count="availabilityFor(item)!.total_count"
          />
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
                <div v-if="row.hint" class="mt-1 text-[11px] font-medium text-gray-500 dark:text-gray-400">
                  {{ row.hint }}
                </div>
              </div>

              <div class="grid min-w-0 flex-1 gap-2 md:grid-cols-2">
                <div class="price-stat">
                  <div class="price-stat-label">{{ t('modelCatalog.priceColumns.official') }}</div>
                  <div class="price-stat-value">{{ formatUsd(row.officialUsd) }}</div>
                </div>

                <div class="price-stat" :class="displayedChargeStatClass()">
                  <div class="price-stat-label">{{ displayedChargeLabel() }}</div>
                  <div class="price-stat-value">{{ formatDisplayedCharge(row.balanceUsd, row.actualCny) }}</div>
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
          <span v-if="mediaTierCount(item)" class="cap-badge cap-badge-slate">
            {{ t('modelCatalog.capabilities.resolutionTiers', { count: mediaTierCount(item) }) }}
          </span>
          <span v-else-if="item.pricing_details.intervals.length" class="cap-badge cap-badge-slate">
            {{ t('modelCatalog.capabilities.tieredPricing', { count: item.pricing_details.intervals.length }) }}
          </span>
          <span v-if="item.official_pricing.has_reference && item.official_pricing.source" class="cap-badge cap-badge-emerald">
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
            :aria-expanded="isExpanded(item)"
            @click="toggleExpanded(item)"
          >
            <Icon
              name="chevronDown"
              size="sm"
              class="transition-transform duration-normal ease-standard"
              :class="{ 'rotate-180': isExpanded(item) }"
            />
            {{ isExpanded(item) ? t('modelCatalog.collapseDetails') : t('modelCatalog.expandDetails') }}
          </button>

          <Collapse :open="isExpanded(item)" unmount-on-hide>
            <div class="grid gap-4 pt-4 xl:grid-cols-[1.1fr_0.9fr]">
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
                        {{ formatIntervalHeading(interval, item.billing_mode) }}
                      </div>
                      <!-- 非 token 模式下标题已经是档位名，再重复一次没有信息量 -->
                      <div v-if="item.billing_mode === 'token'" class="text-xs text-gray-500 dark:text-gray-400">
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
                    <div class="flex items-start gap-3">
                      <div>
                        <div class="font-medium text-gray-900 dark:text-white">{{ tx(other.group.name) }}</div>
                        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                          {{ t('modelCatalog.groupRateLabel', { rate: formatRate(other.group.rate_multiplier) }) }}
                        </div>
                      </div>
                    </div>

                    <div class="mt-3 rounded-xl bg-gray-50 px-3 py-2 text-sm dark:bg-dark-800">
                      <div class="text-xs text-gray-500 dark:text-gray-400">
                        {{ t('modelCatalog.peerDisplayedPrice') }}
                      </div>
                      <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                        {{
                          formatDisplayedCharge(
                            getPrimaryPrice(other.effective_pricing_usd, item.billing_mode),
                            convertUsdAmountToCny(
                              getPrimaryPrice(other.effective_pricing_usd, item.billing_mode),
                              balanceCreditCnyPerUsd,
                            ),
                          )
                        }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </Collapse>
        </div>
      </article>
    </section>
  </div>
</template>

<script setup lang="ts">
// fork：模型广场主体（筛选、卡片）。登录视图与匿名视图共用，数据由 ModelCatalogView 提供。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SearchInput from '@/components/common/SearchInput.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import Collapse from '@/components/motion/Collapse.vue'
import GroupAvailabilityBar from './GroupAvailabilityBar.vue'
import ModelCatalogSkeleton from './ModelCatalogSkeleton.vue'
import ModelIdLabel from './ModelIdLabel.vue'
import ModelModalityIcons from './ModelModalityIcons.vue'
import { vSegmented } from '@/directives/segmented'
import {
  convertCnyAmountToUsd,
  convertUsdAmountToCny,
  filterModelCatalogItems,
  getCatalogItemKey,
  getPrimaryEffectivePrice,
  getPrimaryPrice,
  sortModelCatalogItems,
  type ModelCatalogBillingMode,
  type ModelCatalogItem,
  type ModelCatalogMediaUnit,
  type ModelCatalogPriceInterval,
  type ModelCatalogSortKey,
} from '@/api/modelCatalog'
import type { GroupStatusHistoryOverview } from '@/api/groupStatus'
import { useContentTranslation } from '@/composables/useContentTranslation'

interface PriceRow {
  key: string
  label: string
  unit: string
  /** 附加说明，如「未指定尺寸时按此档计费」 */
  hint?: string
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

const props = withDefaults(defineProps<{
  items: ModelCatalogItem[]
  loading: boolean
  refreshing?: boolean
  loadError?: string
  lastUpdatedAt?: Date | null
  balanceCreditCnyPerUsd?: number | null
  usdExchangeRate?: number | null
  paymentConfigNotice?: string
  /** 分组 ID → 可用率分桶；只有登录视图且开启了运行状态时才有 */
  availability?: Record<number, GroupStatusHistoryOverview>
  /** 匿名视图：隐藏倍率来源与登录才有的信息，显示登录引导 */
  publicMode?: boolean
  initialSearch?: string
}>(), {
  refreshing: false,
  loadError: '',
  lastUpdatedAt: null,
  balanceCreditCnyPerUsd: null,
  usdExchangeRate: null,
  paymentConfigNotice: '',
  availability: () => ({}),
  publicMode: false,
  initialSearch: '',
})

const emit = defineEmits<{
  refresh: []
}>()

const { t, locale } = useI18n()
// fork：内容自动翻译（只换显示的分组名；排序 / 搜索仍按原文）
const { tx } = useContentTranslation()

const expandedKeys = ref<Set<string>>(new Set())

const search = ref(props.initialSearch)
const selectedGroupId = ref<number | null>(null)
const selectedPlatform = ref('all')
const selectedBillingMode = ref('all')
const sortBy = ref<ModelCatalogSortKey>('effective_price_asc')

watch(() => props.initialSearch, (value) => {
  search.value = value
})

// 数据重新加载后，原来选中的分组可能已经不在了
watch(() => props.items, (items) => {
  if (selectedGroupId.value != null && !items.some(item => item.best_group.id === selectedGroupId.value)) {
    selectedGroupId.value = null
  }
})

const loginTarget = computed(() => ({ path: '/login', query: { redirect: '/models' } }))

const billingModeOptions = computed(() => [
  { value: 'all', label: t('modelCatalog.filters.allBillingModes') },
  { value: 'token', label: t('modelCatalog.billingMode.token') },
  { value: 'per_request', label: t('modelCatalog.billingMode.perRequest') },
  { value: 'image', label: t('modelCatalog.billingMode.image') },
  { value: 'video', label: t('modelCatalog.billingMode.video') },
])

const groupTabs = computed<GroupTab[]>(() => {
  const counts = new Map<number, GroupTab>()

  for (const item of props.items) {
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
      count: props.items.length,
    },
    ...groups,
  ]
})

const activeGroupTabLabel = computed(() => {
  return groupTabs.value.find(tab => tab.id === selectedGroupId.value)?.name || t('modelCatalog.groupTabs.allGroups')
})

const platformOptions = computed(() => {
  return [...new Set(props.items.map(item => item.platform).filter(Boolean))].sort((a, b) =>
    a.localeCompare(b, undefined, { sensitivity: 'base' }),
  )
})

const filteredItems = computed(() =>
  filterModelCatalogItems(props.items, {
    search: search.value,
    groupId: selectedGroupId.value,
    platform: selectedPlatform.value,
    billingMode: selectedBillingMode.value,
  }),
)

const visibleItems = computed(() => sortModelCatalogItems(filteredItems.value, sortBy.value))

const hasActiveFilters = computed(() =>
  search.value.trim() !== '' ||
  selectedGroupId.value != null ||
  selectedPlatform.value !== 'all' ||
  selectedBillingMode.value !== 'all',
)

const lastUpdatedLabel = computed(() => {
  if (!props.lastUpdatedAt) {
    return t('modelCatalog.neverUpdated')
  }
  return props.lastUpdatedAt.toLocaleString()
})

function resetFilters() {
  search.value = ''
  selectedGroupId.value = null
  selectedPlatform.value = 'all'
  selectedBillingMode.value = 'all'
}

function availabilityFor(item: ModelCatalogItem): GroupStatusHistoryOverview | undefined {
  if (props.publicMode) return undefined
  return props.availability[item.best_group.id]
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
  const toCny = (value: number | null) => convertUsdAmountToCny(value, props.balanceCreditCnyPerUsd)

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

  if (isMediaBillingMode(item.billing_mode)) {
    const tiers = item.pricing_details.media_tiers ?? []
    if (tiers.length > 0) {
      return tiers.map(tier => ({
        key: `media_${tier.tier}`,
        label: tier.tier,
        unit: mediaUnitLabel(item.pricing_details.media_unit),
        hint: tier.is_default_tier ? t('modelCatalog.defaultTierHint') : undefined,
        officialUsd: tier.official_usd,
        balanceUsd: tier.effective_usd,
        actualCny: toCny(tier.effective_usd),
      }))
    }

    // 后端没下发档位时退回单行，避免整张卡片空掉。
    const isImage = item.billing_mode === 'image'
    const officialUsd = isImage
      ? item.official_pricing.per_image_usd
      : item.official_pricing.per_second_usd
    const balanceUsd = isImage
      ? item.effective_pricing_usd.per_image_usd
      : item.effective_pricing_usd.per_second_usd
    return [
      {
        key: isImage ? 'per_image' : 'per_second',
        label: isImage ? t('modelCatalog.priceLabels.perImage') : t('modelCatalog.priceLabels.perSecond'),
        unit: billingUnitLabel(item.billing_mode),
        officialUsd,
        balanceUsd,
        actualCny: toCny(balanceUsd),
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

  if (billingMode === 'video') {
    return [{
      key: 'per_second',
      label: t('modelCatalog.priceLabels.perSecond'),
      value: interval.per_request_usd,
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
  if (mode === 'video') return t('modelCatalog.billingMode.video')
  return t('modelCatalog.billingMode.token')
}

function billingUnitLabel(mode: ModelCatalogBillingMode): string {
  if (mode === 'per_request') return t('modelCatalog.units.perRequest')
  if (mode === 'image') return t('modelCatalog.units.perImage')
  if (mode === 'video') return t('modelCatalog.units.perSecond')
  return t('modelCatalog.units.perMillionTokens')
}

/** 图片/视频按分辨率分档，与 token 模式的上下文区间是两条不同的分档轴。 */
function isMediaBillingMode(mode: ModelCatalogBillingMode): boolean {
  return mode === 'image' || mode === 'video'
}

function mediaUnitLabel(unit: ModelCatalogMediaUnit): string {
  return unit === 'second' ? t('modelCatalog.units.perSecond') : t('modelCatalog.units.perImage')
}

function mediaTierCount(item: ModelCatalogItem): number {
  return item.pricing_details.media_tiers?.length ?? 0
}

/**
 * 主价格副标题。图片/视频的主价取的是「未指定尺寸时落账的档位」，不点名档位
 * 的话，用户会以为那是唯一价格。
 */
function primaryPriceSubtitle(item: ModelCatalogItem): string {
  const base = `${displayedChargeLabel()} · ${billingUnitLabel(item.billing_mode)}`
  if (!isMediaBillingMode(item.billing_mode)) {
    return base
  }
  const defaultTier = (item.pricing_details.media_tiers ?? []).find(tier => tier.is_default_tier)
  return defaultTier ? `${base} · ${defaultTier.tier}` : base
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

function displayedChargeLabel(): string {
  return props.balanceCreditCnyPerUsd != null
    ? t('modelCatalog.priceColumns.cash')
    : t('modelCatalog.priceColumns.balance')
}

function displayedChargeStatClass(): string {
  return props.balanceCreditCnyPerUsd != null ? 'price-stat-cny' : 'price-stat-strong'
}

function shouldDisplayCashInUsd(): boolean {
  return locale.value.toLowerCase().startsWith('en')
}

function formatDisplayedCharge(balanceUsd: number | null | undefined, actualCny: number | null | undefined): string {
  if (props.balanceCreditCnyPerUsd != null) {
    if (shouldDisplayCashInUsd()) {
      return formatUsd(convertCnyAmountToUsd(actualCny, props.usdExchangeRate))
    }
    return formatCny(actualCny)
  }
  return formatUsd(balanceUsd)
}

function formatPrimaryDisplayPrice(item: ModelCatalogItem): string {
  const balanceUsd = getPrimaryEffectivePrice(item)
  const actualCny = convertUsdAmountToCny(balanceUsd, props.balanceCreditCnyPerUsd)
  return formatDisplayedCharge(balanceUsd, actualCny)
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

/**
 * 区间标题。非 token 模式的分档靠 tier_label 承载，min/max_tokens 恒为 0/null，
 * 用 token 区间格式化器会渲染成「≥ —」。
 */
function formatIntervalHeading(
  interval: ModelCatalogPriceInterval,
  mode: ModelCatalogBillingMode,
): string {
  if (mode !== 'token') {
    return interval.tier_label || t('modelCatalog.intervalDefaultLabel')
  }
  return formatIntervalRange(interval)
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

function getCardClass(item: ModelCatalogItem): string {
  if (item.billing_mode === 'image') {
    return 'model-card-image'
  }
  if (item.billing_mode === 'video') {
    return 'model-card-video'
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
  if (mode === 'video') return 'mode-pill-video'
  if (mode === 'per_request') return 'mode-pill-request'
  return 'mode-pill-token'
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
.model-card {
  @apply relative overflow-hidden rounded-3xl border border-gray-200/80 bg-white/95 p-5 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80;
  transition:
    transform var(--motion-layout) var(--motion-ease),
    box-shadow var(--motion-layout) var(--motion-ease),
    border-color var(--motion-layout) var(--motion-ease);
}

/* 悬浮上浮：轻抬 2px、描边加深；减少动态效果时只保留颜色变化。 */
.model-card:hover {
  @apply -translate-y-0.5 border-gray-300 shadow-md dark:border-dark-600;
}

@media (prefers-reduced-motion: reduce) {
  .model-card:hover {
    transform: none;
  }
}

/* 首次出现时依次上浮淡入，最多错开 11 张；动画结束后不保留 transform。 */
@keyframes catalog-card-enter {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.catalog-card-enter {
  animation: catalog-card-enter 360ms var(--motion-ease) backwards;
  animation-delay: calc(var(--enter-i, 0) * 35ms);
}

@media (prefers-reduced-motion: reduce) {
  .catalog-card-enter {
    animation: none;
  }
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

.model-card-video::before {
  @apply bg-gradient-to-r from-violet-500 via-indigo-400 to-sky-400;
}

.vendor-tile {
  @apply flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-gray-200/80 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800;
  transition: transform var(--motion-layout) var(--motion-ease);
}

.model-card:hover .vendor-tile {
  transform: scale(1.06) rotate(-3deg);
}

@media (prefers-reduced-motion: reduce) {
  .model-card:hover .vendor-tile {
    transform: none;
  }
}

.platform-pill,
.mode-pill,
.group-pill,
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
  @apply bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-200;
}

.mode-pill-request {
  @apply bg-amber-100 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300;
}

.mode-pill-image {
  @apply bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300;
}

.mode-pill-video {
  @apply bg-indigo-100 text-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-300;
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
  @apply bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-200;
}

.cap-badge-emerald {
  @apply bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300;
}

.cap-badge-rose {
  @apply bg-rose-50 text-rose-700 dark:bg-rose-950/30 dark:text-rose-300;
}
</style>
