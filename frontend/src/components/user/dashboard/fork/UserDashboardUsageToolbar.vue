<template>
  <!-- 放在页面标题行右侧；窄屏时整行换到标题下方，控件在内部继续换行 -->
  <div class="flex max-w-full flex-wrap items-center gap-2" data-testid="usage-toolbar">
    <UserDashboardFilterDropdown :active-count="activeCount" @reset="resetFilters">
      <div class="min-w-0">
        <span class="input-label">{{ t('usage.apiKeyFilter') }}</span>
        <Select v-model="filters.api_key_id" :options="apiKeyOptions" searchable data-testid="usage-filter-api-key" @change="applyFilters" />
      </div>
      <div class="min-w-0">
        <span class="input-label">{{ t('usage.model') }}</span>
        <Select v-model="filters.model" :options="modelOptions" searchable data-testid="usage-filter-model" @change="applyFilters" />
      </div>
      <div class="min-w-0">
        <span class="input-label">{{ t('admin.usage.group') }}</span>
        <Select v-model="filters.group_id" :options="groupOptions" searchable data-testid="usage-filter-group" @change="applyFilters" />
      </div>
      <div class="min-w-0">
        <span class="input-label">{{ t('usage.type') }}</span>
        <Select v-model="filters.request_type" :options="requestTypeOptions" data-testid="usage-filter-request-type" @change="applyFilters" />
      </div>
      <div v-if="subscriptionEnabled" class="min-w-0">
        <span class="input-label">{{ t('admin.usage.billingType') }}</span>
        <Select v-model="filters.billing_type" :options="billingTypeOptions" data-testid="usage-filter-billing-type" @change="applyFilters" />
      </div>
      <div class="min-w-0">
        <span class="input-label">{{ t('admin.usage.billingMode') }}</span>
        <Select v-model="filters.billing_mode" :options="billingModeOptions" data-testid="usage-filter-billing-mode" @change="applyFilters" />
      </div>
    </UserDashboardFilterDropdown>
    <button
      type="button"
      class="btn btn-secondary btn-icon shrink-0"
      :disabled="refreshing"
      :title="t('common.refresh')"
      :aria-label="t('common.refresh')"
      data-testid="usage-refresh"
      @click="emit('refresh')"
    >
      <Icon name="refresh" size="sm" :class="refreshing ? 'animate-spin motion-reduce:animate-none' : ''" />
    </button>
    <DateRangePicker
      :start-date="pickerStart"
      :end-date="pickerEnd"
      @change="selectCustomRange"
    />
  </div>
</template>

<script setup lang="ts">
// fork：仪表盘标题行的工具栏（移植自 TokenRouter UserDashboardUsageToolbar.vue）：筛选、刷新、时间范围。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import UserDashboardFilterDropdown from './UserDashboardFilterDropdown.vue'
import { injectUsageChartState } from './usageChartState'

// 刷新范围覆盖整个仪表盘，由页面统一处理。
defineProps<{ refreshing?: boolean }>()
const emit = defineEmits<{ (event: 'refresh'): void }>()

const { t } = useI18n()
const appStore = useAppStore()
const {
  pickerStart,
  pickerEnd,
  filterState,
  applyFilters,
  resetFilters,
  selectCustomRange,
} = injectUsageChartState()
const {
  filters,
  activeCount,
  apiKeyOptions,
  modelOptions,
  groupOptions,
  requestTypeOptions,
  billingTypeOptions,
  billingModeOptions,
} = filterState

// 订阅功能关闭后只剩余额计费，「计费类型」筛选失去意义，与使用记录页一样隐藏。
const subscriptionEnabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.subscription))
</script>
