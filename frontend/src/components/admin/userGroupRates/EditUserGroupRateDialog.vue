<template>
  <BaseDialog :show="show" :title="t('admin.userGroupRates.editDialog.title')" width="narrow" @close="handleClose">
    <div v-if="row" class="space-y-4">
      <dl class="grid grid-cols-[auto,1fr] gap-x-4 gap-y-2 text-sm">
        <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.userGroupRates.columns.user') }}</dt>
        <dd class="min-w-0">
          <div class="truncate font-medium text-gray-900 dark:text-white" :title="row.user_email">{{ row.user_email }}</div>
          <div class="truncate text-xs text-gray-500 dark:text-gray-400">
            #{{ row.user_id }}<template v-if="row.username"> · {{ row.username }}</template>
          </div>
        </dd>
        <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.userGroupRates.columns.group') }}</dt>
        <dd class="min-w-0 truncate text-gray-900 dark:text-white">{{ row.group_name }}</dd>
        <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.userGroupRates.columns.groupRate') }}</dt>
        <dd class="text-gray-900 dark:text-white">×{{ formatRate(row.group_rate_multiplier) }}</dd>
        <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.userGroupRates.editDialog.currentRate') }}</dt>
        <dd class="text-gray-900 dark:text-white">×{{ formatRate(row.rate_multiplier) }}</dd>
      </dl>

      <div>
        <label class="input-label" for="ugr-edit-rate">{{ t('admin.userGroupRates.editDialog.newRate') }}</label>
        <input
          id="ugr-edit-rate"
          ref="inputRef"
          v-model.number="value"
          type="number"
          step="0.0001"
          min="0.0001"
          class="input w-full"
          data-test="ugr-edit-input"
          @keydown.enter.prevent="submit"
        />
        <p v-if="previewPercent !== null" class="mt-1.5 text-xs" :class="notBelowGroup ? 'text-amber-600 dark:text-amber-400' : 'text-gray-500 dark:text-gray-400'">
          {{ t('admin.userGroupRates.columns.discount') }} {{ formatPercent(previewPercent) }}
          <template v-if="notBelowGroup"> · {{ t('admin.userGroupRates.higherThanGroup') }}</template>
        </p>
        <p v-if="error" class="mt-1.5 text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="submitting" data-test="ugr-edit-save" @click="submit">
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
// fork：专属倍率用户管理——修改单个（用户, 分组）的专属倍率。
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { UserGroupRateBatchItem, UserGroupRateRow } from '@/api/admin/userGroupRates'
import { discountPercent, formatPercent, formatRate, isValidRate, roundRate } from '@/utils/userGroupRates'

const props = defineProps<{
  show: boolean
  row: UserGroupRateRow | null
  submitting?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [item: UserGroupRateBatchItem]
}>()

const { t } = useI18n()

const value = ref<number | ''>('')
const error = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

watch(
  () => [props.show, props.row] as const,
  async ([open, row]) => {
    if (!open || !row) return
    value.value = row.rate_multiplier
    error.value = ''
    await nextTick()
    // BaseDialog 在同一个 tick 里把焦点放到关闭按钮上，这里晚一拍再聚焦输入框
    setTimeout(() => {
      inputRef.value?.focus()
      inputRef.value?.select()
    }, 0)
  },
  { immediate: true }
)

const previewPercent = computed(() =>
  props.row && isValidRate(value.value) ? discountPercent(roundRate(value.value), props.row.group_rate_multiplier) : null
)

const notBelowGroup = computed(
  () =>
    !!props.row &&
    isValidRate(value.value) &&
    props.row.group_rate_multiplier > 0 &&
    roundRate(value.value) >= roundRate(props.row.group_rate_multiplier)
)

function submit() {
  if (!props.row || props.submitting) return
  if (!isValidRate(value.value)) {
    error.value = t('admin.userGroupRates.invalidRate')
    return
  }
  const rate = roundRate(value.value)
  if (rate === roundRate(props.row.rate_multiplier)) {
    emit('close')
    return
  }
  emit('submit', { user_id: props.row.user_id, group_id: props.row.group_id, rate_multiplier: rate })
}

function handleClose() {
  if (props.submitting) return
  emit('close')
}
</script>
