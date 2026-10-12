<template>
  <!-- 筛选按钮 + 浮层（参考 TokenRouter FilterDropdown）：有筛选时按钮变主题色并显示数量，浮层里有条件标签和重置 -->
  <div ref="root" class="relative shrink-0" @keydown.esc.stop.prevent="close">
    <button
      type="button"
      class="filter-trigger"
      :class="{ 'filter-trigger-active': activeCount > 0 }"
      :aria-label="t('modelCatalog.filters.title')"
      :title="t('modelCatalog.filters.title')"
      :aria-expanded="open"
      aria-haspopup="dialog"
      data-testid="catalog-filter-trigger"
      @click="open = !open"
    >
      <Icon name="filter" size="sm" />
      <span v-if="activeCount" class="filter-trigger-count">{{ activeCount }}</span>
    </button>

    <MotionTransition name="dropdown-fade">
      <div v-if="open" class="filter-panel" role="dialog" :aria-label="t('modelCatalog.filters.title')" data-testid="catalog-filter-panel">
        <div class="flex items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelCatalog.filters.title') }}</span>
            <span v-if="activeCount" class="rounded-full bg-primary-50 px-1.5 text-xs font-semibold text-primary-600 dark:bg-primary-500/15 dark:text-primary-300">
              {{ activeCount }}
            </span>
          </div>
          <button
            type="button"
            class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 disabled:opacity-40 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-white"
            :disabled="!activeCount"
            data-testid="catalog-filter-reset"
            @click="emit('reset')"
          >
            <Icon name="refresh" size="xs" />
            {{ t('modelCatalog.filters.reset') }}
          </button>
        </div>

        <div v-if="chips.length" class="flex flex-wrap gap-1.5 border-b border-gray-100 px-4 py-2.5 dark:border-dark-700">
          <span v-for="chip in chips" :key="chip.key" class="inline-flex max-w-full items-center gap-1 rounded-md bg-gray-100 py-0.5 pl-2 pr-1 text-xs dark:bg-dark-700">
            <span class="shrink-0 text-gray-500 dark:text-gray-400">{{ chip.label }}</span>
            <span class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ chip.text }}</span>
            <button
              type="button"
              class="rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-white"
              :aria-label="t('modelCatalog.filters.remove', { label: chip.label })"
              @click="chip.clear()"
            >
              <Icon name="x" size="xs" />
            </button>
          </span>
        </div>

        <div class="grid grid-cols-1 gap-3 p-4 sm:grid-cols-3">
          <label class="block">
            <span class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('modelCatalog.filters.brand') }}</span>
            <select :value="platform" class="input text-sm" data-testid="catalog-filter-platform" @change="emit('update:platform', ($event.target as HTMLSelectElement).value)">
              <option value="all">{{ t('modelCatalog.filters.allBrands') }}</option>
              <option v-for="option in platformOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('modelCatalog.filters.pricingType') }}</span>
            <select :value="pricing" class="input text-sm" data-testid="catalog-filter-pricing" @change="emit('update:pricing', ($event.target as HTMLSelectElement).value as CatalogPricingFilter)">
              <option v-for="option in pricingOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="block">
            <span class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('modelCatalog.filters.group') }}</span>
            <select :value="String(groupId)" class="input text-sm" data-testid="catalog-filter-group" @change="onGroupChange">
              <option value="all">{{ t('modelCatalog.groupTabs.allGroups') }}</option>
              <option v-for="option in groupOptions" :key="option.value" :value="String(option.value)">{{ option.label }}</option>
            </select>
          </label>
        </div>
      </div>
    </MotionTransition>
  </div>
</template>

<script setup lang="ts">
// fork：模型广场筛选浮层（参考 TokenRouter components/common/FilterDropdown.vue）。点击外部或按 Esc 关闭。
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import MotionTransition from '@/components/motion/MotionTransition.vue'
import type { CatalogPricingFilter } from './catalogGroups'

const props = defineProps<{
  platform: string
  pricing: CatalogPricingFilter
  groupId: number | 'all'
  platformOptions: Array<{ value: string; label: string }>
  pricingOptions: Array<{ value: CatalogPricingFilter; label: string }>
  groupOptions: Array<{ value: number; label: string }>
}>()

const emit = defineEmits<{
  'update:platform': [value: string]
  'update:pricing': [value: CatalogPricingFilter]
  'update:groupId': [value: number | 'all']
  reset: []
}>()

const { t } = useI18n()
const open = ref(false)
const root = ref<HTMLElement | null>(null)

const activeCount = computed(() =>
  [props.platform !== 'all', props.pricing !== 'all', props.groupId !== 'all'].filter(Boolean).length,
)

const chips = computed(() => {
  const list: Array<{ key: string; label: string; text: string; clear: () => void }> = []
  if (props.platform !== 'all') {
    list.push({
      key: 'platform',
      label: t('modelCatalog.filters.brand'),
      text: props.platformOptions.find(option => option.value === props.platform)?.label ?? props.platform,
      clear: () => emit('update:platform', 'all'),
    })
  }
  if (props.pricing !== 'all') {
    list.push({
      key: 'pricing',
      label: t('modelCatalog.filters.pricingType'),
      text: props.pricingOptions.find(option => option.value === props.pricing)?.label ?? props.pricing,
      clear: () => emit('update:pricing', 'all'),
    })
  }
  if (props.groupId !== 'all') {
    list.push({
      key: 'group',
      label: t('modelCatalog.filters.group'),
      text: props.groupOptions.find(option => option.value === props.groupId)?.label ?? String(props.groupId),
      clear: () => emit('update:groupId', 'all'),
    })
  }
  return list
})

function onGroupChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  emit('update:groupId', value === 'all' ? 'all' : Number(value))
}

function close() {
  open.value = false
}

function onDocumentPointerDown(event: PointerEvent) {
  if (open.value && root.value && !root.value.contains(event.target as Node)) close()
}

onMounted(() => document.addEventListener('pointerdown', onDocumentPointerDown))
onBeforeUnmount(() => document.removeEventListener('pointerdown', onDocumentPointerDown))
</script>

<style scoped>
.filter-trigger {
  @apply relative inline-flex h-10 w-10 items-center justify-center rounded-xl border border-gray-200 bg-white text-gray-600 transition-colors hover:bg-gray-50 hover:text-gray-900 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white;
}

.filter-trigger-active {
  @apply border-primary-300 text-primary-600 dark:border-primary-500/50 dark:text-primary-300;
}

.filter-trigger-count {
  @apply absolute -right-1.5 -top-1.5 flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-primary-500 px-1 text-[10px] font-bold text-white;
}

.filter-panel {
  @apply absolute right-0 top-full z-30 mt-2 w-[min(40rem,calc(100vw-2rem))] rounded-xl border border-gray-200 bg-white shadow-xl dark:border-dark-600 dark:bg-dark-800;
}
</style>
