<template>
  <BaseDialog :show="show" :title="t('admin.userGroupRates.batch.title')" width="wide" @close="handleClose">
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('admin.userGroupRates.batch.selected', { count: rows.length }) }}
      </p>

      <div>
        <label class="input-label">{{ t('admin.userGroupRates.batch.mode') }}</label>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="m in MODES"
            :key="m"
            type="button"
            class="rounded-lg border px-3 py-1.5 text-sm transition-colors"
            :class="mode === m
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-300'
              : 'border-gray-200 text-gray-600 hover:border-gray-300 dark:border-dark-600 dark:text-gray-300 dark:hover:border-dark-500'"
            :data-test="`ugr-mode-${m}`"
            @click="mode = m"
          >
            {{ t(`admin.userGroupRates.batch.modes.${m}`) }}
          </button>
        </div>
      </div>

      <div v-if="mode !== 'reset'">
        <label class="input-label">{{ t(`admin.userGroupRates.batch.valueLabel.${mode}`) }}</label>
        <input
          v-model.number="value"
          type="number"
          :step="mode === 'percent' ? '1' : '0.0001'"
          class="input w-full sm:w-48"
          data-test="ugr-batch-value"
        />
      </div>

      <div>
        <div class="mb-2 flex items-center justify-between">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('admin.userGroupRates.batch.preview') }}</span>
          <div class="flex flex-wrap gap-3 text-xs">
            <span v-if="higherCount > 0" class="text-amber-600 dark:text-amber-400">
              {{ t('admin.userGroupRates.batch.higherWarning', { count: higherCount }) }}
            </span>
            <span v-if="invalidCount > 0" class="text-red-600 dark:text-red-400">
              {{ t('admin.userGroupRates.batch.invalidWarning', { count: invalidCount }) }}
            </span>
          </div>
        </div>
        <div class="max-h-80 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
          <table class="w-full text-sm">
            <thead class="sticky top-0 bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('admin.userGroupRates.columns.user') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.userGroupRates.columns.group') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('admin.userGroupRates.columns.groupRate') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('admin.userGroupRates.batch.current') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('admin.userGroupRates.batch.next') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="change in preview" :key="rowKey(change.row)" data-test="ugr-preview-row">
                <td class="max-w-[14rem] truncate px-3 py-1.5 text-gray-900 dark:text-white">{{ change.row.user_email }}</td>
                <td class="max-w-[10rem] truncate px-3 py-1.5 text-gray-600 dark:text-gray-300">{{ change.row.group_name }}</td>
                <td class="px-3 py-1.5 text-right text-gray-500 dark:text-gray-400">×{{ formatRate(change.row.group_rate_multiplier) }}</td>
                <td class="px-3 py-1.5 text-right text-gray-600 dark:text-gray-300">×{{ formatRate(change.from) }}</td>
                <td class="px-3 py-1.5 text-right font-medium" :class="nextClass(change)">{{ nextLabel(change) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="submitting || items.length === 0"
          data-test="ugr-batch-submit"
          @click="emit('submit', items)"
        >
          {{ items.length === 0 ? t('admin.userGroupRates.batch.nothingToChange') : t('admin.userGroupRates.batch.submit', { count: items.length }) }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
// fork：专属倍率用户管理——批量调价（设为固定值 / 按系数 / 按百分比 / 恢复分组默认），提交前预览 旧 → 新。
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { UserGroupRateBatchItem, UserGroupRateRow } from '@/api/admin/userGroupRates'
import {
  computeAdjustedRate,
  formatRate,
  roundRate,
  rowKey,
  toBatchItems,
  type RateAdjustMode,
  type RateChangePreview
} from '@/utils/userGroupRates'

const props = defineProps<{
  show: boolean
  rows: UserGroupRateRow[]
  submitting?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [items: UserGroupRateBatchItem[]]
}>()

const { t } = useI18n()

const MODES: RateAdjustMode[] = ['set', 'factor', 'percent', 'reset']
const mode = ref<RateAdjustMode>('percent')
const value = ref<number | ''>('')

watch(
  () => props.show,
  (open) => {
    if (open) {
      mode.value = 'percent'
      value.value = ''
    }
  }
)

const preview = computed<RateChangePreview<UserGroupRateRow>[]>(() =>
  props.rows.map((row) => ({
    row,
    from: row.rate_multiplier,
    to: computeAdjustedRate(row.rate_multiplier, mode.value, value.value === '' ? null : value.value)
  }))
)

const items = computed(() => toBatchItems(preview.value))

const isUnchanged = (change: RateChangePreview<UserGroupRateRow>) =>
  typeof change.to === 'number' && roundRate(change.to) === roundRate(change.from)

const isNotBelowGroup = (change: RateChangePreview<UserGroupRateRow>) =>
  typeof change.to === 'number' &&
  change.row.group_rate_multiplier > 0 &&
  roundRate(change.to) >= roundRate(change.row.group_rate_multiplier)

const higherCount = computed(() => preview.value.filter((c) => !isUnchanged(c) && isNotBelowGroup(c)).length)
// 只有填了数值仍算不出来才算无效；还没填时不提示
const invalidCount = computed(() =>
  mode.value === 'reset' || value.value === '' ? 0 : preview.value.filter((c) => c.to === undefined).length
)

function nextLabel(change: RateChangePreview<UserGroupRateRow>) {
  if (change.to === null) return t('admin.userGroupRates.batch.restoreDefault')
  if (change.to === undefined) return value.value === '' ? '-' : t('admin.userGroupRates.batch.invalid')
  if (isUnchanged(change)) return t('admin.userGroupRates.batch.unchanged')
  return `×${formatRate(change.to)}`
}

function nextClass(change: RateChangePreview<UserGroupRateRow>) {
  if (change.to === undefined) return value.value === '' ? 'text-gray-400' : 'text-red-600 dark:text-red-400'
  if (change.to === null) return 'text-gray-600 dark:text-gray-300'
  if (isUnchanged(change)) return 'text-gray-400'
  if (isNotBelowGroup(change)) return 'text-amber-600 dark:text-amber-400'
  return change.to > change.from ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'
}

function handleClose() {
  if (props.submitting) return
  emit('close')
}
</script>
