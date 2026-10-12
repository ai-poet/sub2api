<template>
  <!-- 放在页面标题行；窄屏时标题行换行，宽度按页面内边距封顶，控件在内部继续换行 -->
  <div class="flex max-w-[calc(100vw-2rem)] flex-wrap items-center gap-2 md:max-w-[calc(100vw-3rem)]" data-testid="usage-toolbar">
    <UserDashboardFilterDropdown :active-count="activeCount" :columns="3" @reset="resetFilters">
      <UserDashboardFilterField
        :label="t('usage.apiKeyFilter')"
        :value-text="chipText(filters.api_key_id, apiKeyOptions)"
        @clear="clearFilter('api_key_id')"
      >
        <Select v-model="filters.api_key_id" :options="apiKeyOptions" searchable data-testid="usage-filter-api-key" @change="applyFilters" />
      </UserDashboardFilterField>
      <UserDashboardFilterField
        :label="t('usage.model')"
        :value-text="chipText(filters.model, modelOptions)"
        @clear="clearFilter('model')"
      >
        <Select v-model="filters.model" :options="modelOptions" searchable data-testid="usage-filter-model" @change="applyFilters" />
      </UserDashboardFilterField>
      <UserDashboardFilterField
        :label="t('admin.usage.group')"
        :value-text="chipText(filters.group_id, groupOptions)"
        @clear="clearFilter('group_id')"
      >
        <Select v-model="filters.group_id" :options="groupOptions" searchable data-testid="usage-filter-group" @change="applyFilters" />
      </UserDashboardFilterField>
      <UserDashboardFilterField
        :label="t('usage.type')"
        :value-text="chipText(filters.request_type, requestTypeOptions)"
        @clear="clearFilter('request_type')"
      >
        <Select v-model="filters.request_type" :options="requestTypeOptions" data-testid="usage-filter-request-type" @change="applyFilters" />
      </UserDashboardFilterField>
      <UserDashboardFilterField
        v-if="subscriptionEnabled"
        :label="t('admin.usage.billingType')"
        :value-text="chipText(filters.billing_type, billingTypeOptions)"
        @clear="clearFilter('billing_type')"
      >
        <Select v-model="filters.billing_type" :options="billingTypeOptions" data-testid="usage-filter-billing-type" @change="applyFilters" />
      </UserDashboardFilterField>
      <UserDashboardFilterField
        :label="t('admin.usage.billingMode')"
        :value-text="chipText(filters.billing_mode, billingModeOptions)"
        @clear="clearFilter('billing_mode')"
      >
        <Select v-model="filters.billing_mode" :options="billingModeOptions" data-testid="usage-filter-billing-mode" @change="applyFilters" />
      </UserDashboardFilterField>
      <UserDashboardFilterField
        :label="t('usage.compactionFilter')"
        :value-text="chipText(filters.native_compaction_v2, compactionOptions)"
        @clear="clearFilter('native_compaction_v2')"
      >
        <Select v-model="filters.native_compaction_v2" :options="compactionOptions" data-testid="usage-filter-compaction" @change="applyFilters" />
      </UserDashboardFilterField>
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
    <DashboardDateRangePicker
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
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import DashboardDateRangePicker from './DashboardDateRangePicker.vue'
import UserDashboardFilterDropdown from './UserDashboardFilterDropdown.vue'
import UserDashboardFilterField from './UserDashboardFilterField.vue'
import { injectUsageChartState } from './usageChartState'
import type { UsageChartFilters } from './useUsageChartFilters'

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
  compactionOptions,
} = filterState

// 订阅功能关闭后只剩余额计费，「计费类型」筛选失去意义，与使用记录页一样隐藏。
const subscriptionEnabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.subscription))

// 本项目的 Select 不向筛选字段上报选项，已选条件标签的文案按当前取值在候选项里查，null 表示不限、不出标签。
const chipText = (value: SelectOption['value'], options: SelectOption[]): string => {
  if (value === null) return ''
  return options.find((option) => option.value === value)?.label ?? String(value)
}

// 移除标签与在 Select 里选回“全部”一样：清空该条件并按新条件取数。
const clearFilter = (key: keyof UsageChartFilters) => {
  filters.value[key] = null
  applyFilters()
}
</script>
