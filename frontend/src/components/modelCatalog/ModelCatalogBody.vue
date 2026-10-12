<template>
  <div class="space-y-4">
    <!-- 标题行：左侧标题与统计，右侧搜索 + 筛选（参考 TokenRouter 模型广场工具栏） -->
    <section class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
      <div class="min-w-0">
        <h1 v-if="showTitle" class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{{ t('modelCatalog.title') }}</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400" :class="{ 'mt-1': showTitle }" data-testid="catalog-summary">
          {{ summaryLabel }}
        </p>
      </div>
      <div class="flex min-w-0 items-center gap-2">
        <div class="min-w-0 flex-1 md:w-80 md:flex-none lg:w-96">
          <SearchInput v-model="search" :placeholder="t('modelCatalog.filters.searchPlaceholder')" :debounce-ms="120" />
        </div>
        <CatalogFilterDropdown
          v-model:platform="selectedPlatform"
          v-model:pricing="selectedPricing"
          v-model:group-id="selectedGroupId"
          :platform-options="platformOptions"
          :pricing-options="pricingOptions"
          :group-options="groupOptions"
          @reset="resetPanelFilters"
        />
      </div>
    </section>

    <!-- 匿名视图：引导登录查看专属价格 -->
    <div
      v-if="publicMode"
      class="flex flex-col gap-3 rounded-xl border border-primary-200/70 bg-primary-50/70 px-4 py-3 text-sm text-primary-800 dark:border-primary-800/50 dark:bg-primary-950/20 dark:text-primary-200 sm:flex-row sm:items-center sm:justify-between"
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
      class="rounded-xl border border-dashed border-amber-300 bg-amber-50/80 px-4 py-3 text-sm text-amber-800 dark:border-amber-700/60 dark:bg-amber-950/20 dark:text-amber-200"
    >
      <div class="flex items-start gap-3">
        <Icon name="infoCircle" size="sm" class="mt-0.5 flex-shrink-0" />
        <div>
          <div class="font-medium">{{ t('modelCatalog.paymentNoticeTitle') }}</div>
          <div class="mt-1 text-amber-700/90 dark:text-amber-200/90">{{ paymentConfigNotice }}</div>
        </div>
      </div>
    </div>

    <ModelCatalogSkeleton v-if="loading && items.length === 0" />

    <div v-else-if="loadError && items.length === 0" class="card border-red-200 p-6 dark:border-red-500/30">
      <div class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('modelCatalog.loadFailedTitle') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ loadError }}</p>
        </div>
        <button v-if="!publicMode || items.length" type="button" class="btn btn-primary" @click="emit('refresh')">
          {{ t('common.refresh') }}
        </button>
      </div>
    </div>

    <div v-else-if="visibleSections.length === 0" class="card px-6 py-14" data-testid="model-catalog-empty">
      <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
        <Icon name="inbox" size="xl" />
      </div>
      <h2 class="mt-6 text-center text-2xl font-semibold text-gray-950 dark:text-white">{{ t('modelCatalog.emptyTitle') }}</h2>
      <p class="mx-auto mt-3 max-w-xl text-center text-sm leading-7 text-gray-600 dark:text-gray-300">
        {{ t('modelCatalog.emptyDescription') }}
      </p>
      <div v-if="hasActiveFilters" class="mt-6 text-center">
        <button type="button" class="btn btn-secondary" data-testid="model-catalog-reset-filters" @click="resetFilters">
          {{ t('modelCatalog.filters.reset') }}
        </button>
      </div>
    </div>

    <div v-else class="space-y-4">
      <section
        v-for="(group, index) in visibleSections"
        :key="group.id"
        class="catalog-group card overflow-hidden"
        :style="{ '--group-i': Math.min(index, 6) }"
        data-testid="catalog-group-section"
        :data-group="group.name"
      >
        <!-- 分组头部：品牌、最高优惠、分组倍率，分组 Logo、名称、描述，右侧可用率条 -->
        <div class="flex flex-col gap-4 border-b border-gray-100 px-4 py-4 dark:border-dark-700 md:px-5 xl:flex-row xl:items-center xl:justify-between">
          <div class="min-w-0 flex-1 space-y-3">
            <div class="flex flex-wrap items-center gap-2">
              <span class="brand-pill" :class="brandPillClass(group.platform)">
                <PlatformIcon :platform="group.platform as any" size="xs" />
                {{ platformLabel(group.platform) }}
              </span>
              <HelpTooltip v-if="formatMaxSavings(group.maxSavings)" :content="t('modelCatalog.group.maxSavingsHint')" width-class="w-72">
                <template #trigger>
                  <span class="info-pill border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200" data-testid="group-max-savings">
                    {{ t('modelCatalog.group.maxSavings', { percent: formatMaxSavings(group.maxSavings) }) }}
                  </span>
                </template>
              </HelpTooltip>
              <HelpTooltip :content="t(group.rateSource === 'user_override' ? 'modelCatalog.group.userRateHint' : 'modelCatalog.group.rateHint')" width-class="w-72">
                <template #trigger>
                  <span class="info-pill border-gray-200/80 bg-gray-50 text-gray-600 dark:border-dark-600 dark:bg-dark-700/60 dark:text-gray-200" data-testid="group-rate">
                    {{ t(group.rateSource === 'user_override' ? 'modelCatalog.group.userRate' : 'modelCatalog.group.rate', { rate: formatGroupMultiplier(group.rateMultiplier) }) }}
                  </span>
                </template>
              </HelpTooltip>
            </div>

            <div class="flex items-start gap-3">
              <span class="group-logo">
                <PlatformIcon :platform="group.platform as any" size="lg" class="!h-7 !w-7" :class="brandIconClass(group.platform)" />
              </span>
              <div class="min-w-0">
                <h2 class="text-lg font-semibold text-gray-950 dark:text-white">{{ tx(group.name) }}</h2>
                <p v-if="group.description" class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ tx(group.description) }}</p>
                <p v-else class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('modelCatalog.group.modelCount', { count: group.models.length }) }}</p>
              </div>
            </div>
          </div>
          <div v-if="availabilityFor(group)" class="w-full xl:ml-6 xl:w-[560px] xl:shrink-0" data-testid="catalog-group-availability">
            <GroupAvailabilityBar
              size="lg"
              :buckets="availabilityFor(group)!.buckets"
              :availability="availabilityFor(group)!.availability"
              :total-count="availabilityFor(group)!.total_count"
            />
          </div>
        </div>

        <div class="grid min-w-0 grid-cols-1 items-start gap-4 p-4 md:grid-cols-2 md:p-5 lg:grid-cols-3">
          <CatalogModelCard v-for="item in group.models" :key="`${group.id}-${item.model}`" :item="item" :price-context="priceContext" />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：模型广场主体（参考 TokenRouter ModelMarketplaceView）：按分组分区，分区头部是分组信息，
// 下面是模型卡片网格。登录视图与匿名视图共用，数据由 ModelCatalogView 提供。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SearchInput from '@/components/common/SearchInput.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import type { ModelCatalogItem } from '@/api/modelCatalog'
import type { GroupStatusHistoryOverview } from '@/api/groupStatus'
import { useContentTranslation } from '@/composables/useContentTranslation'
import CatalogFilterDropdown from './CatalogFilterDropdown.vue'
import CatalogModelCard from './CatalogModelCard.vue'
import GroupAvailabilityBar from './GroupAvailabilityBar.vue'
import ModelCatalogSkeleton from './ModelCatalogSkeleton.vue'
import {
  filterCatalogGroups,
  formatGroupMultiplier,
  formatMaxSavings,
  groupCatalogItems,
  type CatalogGroupSection,
  type CatalogPricingFilter,
} from './catalogGroups'
import type { CatalogPriceContext } from './catalogPricing'

const props = withDefaults(defineProps<{
  items: ModelCatalogItem[]
  loading: boolean
  loadError?: string
  balanceCreditCnyPerUsd?: number | null
  usdExchangeRate?: number | null
  paymentConfigNotice?: string
  /** 分组 ID → 可用率分桶；只有登录视图且开启了运行状态时才有 */
  availability?: Record<number, GroupStatusHistoryOverview>
  /** 匿名视图：显示登录引导，不显示可用率 */
  publicMode?: boolean
  /** 标题已由外壳（顶栏）显示时不再重复 */
  showTitle?: boolean
  initialSearch?: string
}>(), {
  loadError: '',
  balanceCreditCnyPerUsd: null,
  usdExchangeRate: null,
  paymentConfigNotice: '',
  availability: () => ({}),
  publicMode: false,
  showTitle: false,
  initialSearch: '',
})

const emit = defineEmits<{
  refresh: []
}>()

const { t, locale } = useI18n()
// fork：内容自动翻译（只换显示的分组名 / 描述；搜索仍按原文）
const { tx } = useContentTranslation()

const search = ref(props.initialSearch)
const selectedPlatform = ref('all')
const selectedPricing = ref<CatalogPricingFilter>('all')
const selectedGroupId = ref<number | 'all'>('all')

watch(() => props.initialSearch, (value) => {
  search.value = value
})

const sections = computed(() => groupCatalogItems(props.items))

// 数据重新加载后，原来选中的分组可能已经不在了
watch(sections, (list) => {
  if (selectedGroupId.value !== 'all' && !list.some(section => section.id === selectedGroupId.value)) {
    selectedGroupId.value = 'all'
  }
})

const visibleSections = computed(() =>
  filterCatalogGroups(sections.value, {
    search: search.value,
    platform: selectedPlatform.value,
    pricing: selectedPricing.value,
    groupId: selectedGroupId.value,
  }, platformLabel),
)

const priceContext = computed<CatalogPriceContext>(() => ({
  cnyPerUsd: props.balanceCreditCnyPerUsd,
  usdExchangeRate: props.usdExchangeRate,
  locale: locale.value,
}))

const summaryLabel = computed(() => {
  const models = visibleSections.value.reduce((sum, section) => sum + section.models.length, 0)
  return t('modelCatalog.summary', { groups: visibleSections.value.length, models })
})

const platformOptions = computed(() =>
  [...new Set(sections.value.map(section => section.platform).filter(Boolean))]
    .map(platform => ({ value: platform, label: platformLabel(platform) })),
)

const pricingOptions = computed<Array<{ value: CatalogPricingFilter; label: string }>>(() => [
  { value: 'all', label: t('modelCatalog.filters.allBillingModes') },
  { value: 'token', label: t('modelCatalog.billingMode.token') },
  { value: 'per_request', label: t('modelCatalog.billingMode.perRequest') },
  { value: 'image', label: t('modelCatalog.billingMode.image') },
  { value: 'video', label: t('modelCatalog.billingMode.video') },
])

const groupOptions = computed(() => sections.value.map(section => ({ value: section.id, label: tx(section.name) })))

const hasActiveFilters = computed(() =>
  search.value.trim() !== '' || selectedPlatform.value !== 'all' || selectedPricing.value !== 'all' || selectedGroupId.value !== 'all',
)

const loginTarget = computed(() => ({ path: '/login', query: { redirect: '/models' } }))

function resetPanelFilters() {
  selectedPlatform.value = 'all'
  selectedPricing.value = 'all'
  selectedGroupId.value = 'all'
}

function resetFilters() {
  search.value = ''
  resetPanelFilters()
}

function availabilityFor(group: CatalogGroupSection): GroupStatusHistoryOverview | undefined {
  if (props.publicMode) return undefined
  return props.availability[group.id]
}

function platformLabel(platform: string): string {
  return t(`admin.groups.platforms.${platform}`)
}

function brandIconClass(platform: string): string {
  switch (platform) {
    case 'anthropic':
      return 'text-[#D97757]'
    case 'openai':
      return 'text-gray-900 dark:text-white'
    case 'gemini':
      return 'text-blue-500'
    case 'antigravity':
      return 'text-violet-500'
    default:
      return 'text-gray-700 dark:text-gray-200'
  }
}

function brandPillClass(platform: string): string {
  switch (platform) {
    case 'anthropic':
      return 'bg-orange-50 text-orange-700 ring-orange-200 dark:bg-orange-500/10 dark:text-orange-300 dark:ring-orange-500/30'
    case 'openai':
      return 'bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/30'
    case 'antigravity':
      return 'bg-violet-50 text-violet-700 ring-violet-200 dark:bg-violet-500/10 dark:text-violet-300 dark:ring-violet-500/30'
    case 'gemini':
      return 'bg-blue-50 text-blue-700 ring-blue-200 dark:bg-blue-500/10 dark:text-blue-300 dark:ring-blue-500/30'
    default:
      return 'bg-gray-50 text-gray-700 ring-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-600'
  }
}
</script>

<style scoped>
.brand-pill {
  @apply inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold ring-1 ring-inset;
}

.info-pill {
  @apply cursor-help rounded-full border px-3 py-1 text-xs font-semibold;
}

.group-logo {
  @apply mt-0.5 flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-950;
}

/* 分组分区依次上浮淡入（只在首次出现时播放，结束后不保留 transform） */
.catalog-group {
  animation: catalog-group-enter 420ms var(--motion-ease) backwards;
  animation-delay: calc(var(--group-i, 0) * 60ms);
}

@keyframes catalog-group-enter {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .catalog-group {
    animation: none;
  }
}
</style>
