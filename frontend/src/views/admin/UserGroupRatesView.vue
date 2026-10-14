<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-72">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.userGroupRates.searchPlaceholder')"
                class="input pl-10"
                data-test="ugr-search"
                @input="handleSearch"
              />
            </div>
            <div class="w-full sm:w-56">
              <Select
                v-model="groupFilter"
                :options="groupFilterOptions"
                searchable
                @change="handleFilterChange"
              />
            </div>
            <div class="w-full sm:w-36">
              <Select
                v-model="statusFilter"
                :options="statusFilterOptions"
                @change="handleFilterChange"
              />
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadRates"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              v-if="selectedCount > 0"
              type="button"
              class="btn btn-secondary"
              data-test="ugr-clear-selection"
              @click="clearSelection"
            >
              {{ t('admin.userGroupRates.clearSelection') }}
            </button>
            <button
              v-if="canSelectAllFiltered"
              type="button"
              class="btn btn-secondary"
              :disabled="selectingAll"
              data-test="ugr-select-all"
              @click="selectAllFiltered"
            >
              {{ t('admin.userGroupRates.selectAllFiltered', { count: pagination.total }) }}
            </button>
            <button
              v-if="selectedCount > 0"
              type="button"
              class="btn btn-secondary"
              :disabled="selectedCount > USER_GROUP_RATE_BATCH_MAX"
              data-test="ugr-batch"
              @click="showBatchDialog = true"
            >
              {{ t('admin.userGroupRates.batchAdjust', { count: selectedCount }) }}
            </button>
            <button
              type="button"
              class="btn btn-primary"
              data-test="ugr-add"
              @click="showAddDialog = true"
            >
              <Icon name="plus" size="md" class="mr-1" />
              {{ t('admin.userGroupRates.add') }}
            </button>
          </div>
        </div>
        <p class="mt-3 text-sm text-gray-500 dark:text-gray-400" data-test="ugr-summary">
          {{ t('admin.userGroupRates.summary', { users: userCount, entries: pagination.total }) }}
          <span class="ml-2 text-xs text-gray-400 dark:text-gray-500">{{ t('admin.userGroupRates.effectHint') }}</span>
        </p>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="rows"
          :loading="loading"
          :row-key="rowKey"
          selectable
          :selected-keys="selectedKeys"
          :selection-label="selectionLabel"
          :server-side-sort="true"
          default-sort-key="updated_at"
          default-sort-order="desc"
          :actions-count="2"
          @sort="handleSort"
          @update:selected-keys="handleSelectedKeys"
        >
          <template #cell-user="{ row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium text-gray-900 dark:text-white">{{ row.user_email }}</span>
                <span
                  v-if="row.user_status !== 'active'"
                  class="badge badge-danger shrink-0"
                >{{ statusLabel(row.user_status) }}</span>
              </div>
              <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                #{{ row.user_id }}<template v-if="row.username"> · {{ row.username }}</template><template v-if="row.user_notes"> · {{ row.user_notes }}</template>
              </div>
            </div>
          </template>

          <template #cell-group_name="{ row }">
            <GroupBadge
              :name="row.group_name"
              :platform="row.group_platform"
              :rate-multiplier="row.group_rate_multiplier"
              :user-rate-multiplier="row.rate_multiplier"
            />
          </template>

          <template #cell-group_rate_multiplier="{ row }">
            <span class="text-gray-600 dark:text-gray-300">×{{ formatRate(row.group_rate_multiplier) }}</span>
          </template>

          <template #cell-rate_multiplier="{ row }">
            <div v-if="editingKey === rowKey(row)" class="flex items-center gap-1">
              <input
                ref="editInput"
                v-model.number="editingValue"
                type="number"
                step="0.0001"
                min="0.0001"
                class="input hide-spinner w-24 py-1 text-sm"
                data-test="ugr-edit-input"
                @keydown.enter.prevent="saveEdit(row)"
                @keydown.esc.prevent="cancelEdit"
              />
              <button
                type="button"
                class="rounded p-1 text-emerald-600 hover:bg-emerald-50 disabled:opacity-50 dark:hover:bg-emerald-900/20"
                :disabled="saving"
                :title="t('common.save')"
                data-test="ugr-edit-save"
                @click="saveEdit(row)"
              >
                <Icon name="check" size="sm" />
              </button>
              <button
                type="button"
                class="rounded p-1 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700"
                :title="t('common.cancel')"
                @click="cancelEdit"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
            <button
              v-else
              type="button"
              class="group inline-flex items-center gap-1 rounded px-1 font-semibold text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
              :title="t('admin.userGroupRates.edit')"
              data-test="ugr-edit"
              @click="startEdit(row)"
            >
              ×{{ formatRate(row.rate_multiplier) }}
              <Icon name="edit" size="xs" class="opacity-0 transition-opacity group-hover:opacity-100" />
            </button>
          </template>

          <template #cell-discount="{ row }">
            <span
              :class="discountClass(row)"
              :title="isNotBelowGroup(row) ? t('admin.userGroupRates.higherThanGroup') : undefined"
            >{{ formatPercent(discountPercent(row.rate_multiplier, row.group_rate_multiplier)) }}</span>
          </template>

          <template #cell-usage="{ row }">
            <span v-if="usageLoading" class="text-gray-400">…</span>
            <span v-else-if="usageFailed" class="text-gray-400">—</span>
            <template v-else-if="usageFor(row)">
              <div class="text-gray-900 dark:text-white">{{ formatCurrency(usageFor(row)!.actual_cost) }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.userGroupRates.requests', { count: formatNumber(usageFor(row)!.requests) }) }}
              </div>
            </template>
            <span v-else class="text-xs text-gray-400">{{ t('admin.userGroupRates.noUsage') }}</span>
          </template>

          <template #cell-rpm_override="{ row }">
            <span class="text-gray-600 dark:text-gray-300">{{ row.rpm_override ?? '-' }}</span>
          </template>

          <template #cell-updated_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ formatDateTimeToMinute(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                @click="startEdit(row)"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('admin.userGroupRates.edit') }}</span>
              </button>
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                data-test="ugr-reset"
                @click="resetTarget = row"
              >
                <Icon name="refresh" size="sm" />
                <span class="text-xs">{{ t('admin.userGroupRates.reset') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="hasActiveFilters ? t('admin.userGroupRates.emptyFiltered') : t('admin.userGroupRates.empty')"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BatchAdjustRatesDialog
      :show="showBatchDialog"
      :rows="selectedRows"
      :submitting="saving"
      @close="showBatchDialog = false"
      @submit="submitBatch"
    />

    <AddUserGroupRateDialog
      :show="showAddDialog"
      :groups="groups"
      :submitting="saving"
      @close="showAddDialog = false"
      @submit="submitAdd"
    />

    <ConfirmDialog
      :show="resetTarget !== null"
      :title="t('admin.userGroupRates.resetConfirmTitle')"
      :message="resetMessage"
      danger
      @confirm="confirmReset"
      @cancel="resetTarget = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
// fork：专属倍率用户管理——跨用户列出 / 调整用户在各分组的专属计费倍率。
// 运维管理员与用户管理同一规则：看得到全部内容，写操作被后端截获成审批（202），全局提示由 App.vue 弹出。
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import {
  USER_GROUP_RATE_BATCH_MAX,
  type UserGroupRateBatchItem,
  type UserGroupRateGroupOption,
  type UserGroupRateRow,
  type UserGroupRateSortField,
  type UserGroupRateUsage
} from '@/api/admin/userGroupRates'
import { isApprovalQueued } from '@/utils/approval'
import { extractApiErrorMessage } from '@/utils/apiError'
import { discountPercent, formatPercent, formatRate, isValidRate, roundRate, rowKey } from '@/utils/userGroupRates'
import { formatCurrency, formatDateTimeToMinute, formatNumber } from '@/utils/format'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import BatchAdjustRatesDialog from '@/components/admin/userGroupRates/BatchAdjustRatesDialog.vue'
import AddUserGroupRateDialog from '@/components/admin/userGroupRates/AddUserGroupRateDialog.vue'

const { t } = useI18n()
const appStore = useAppStore()

const columns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.userGroupRates.columns.user') },
  { key: 'group_name', label: t('admin.userGroupRates.columns.group'), sortable: true },
  { key: 'group_rate_multiplier', label: t('admin.userGroupRates.columns.groupRate') },
  { key: 'rate_multiplier', label: t('admin.userGroupRates.columns.rate'), sortable: true },
  { key: 'discount', label: t('admin.userGroupRates.columns.discount'), sortable: true },
  { key: 'usage', label: t('admin.userGroupRates.columns.usage30d') },
  { key: 'rpm_override', label: t('admin.userGroupRates.columns.rpmOverride') },
  { key: 'updated_at', label: t('admin.userGroupRates.columns.updatedAt'), sortable: true },
  { key: 'actions', label: t('admin.userGroupRates.columns.actions') }
])

const SORTABLE_FIELDS: UserGroupRateSortField[] = ['updated_at', 'rate_multiplier', 'discount', 'user_id', 'group_name']

// ---------- 列表 ----------

const rows = ref<UserGroupRateRow[]>([])
const loading = ref(false)
const userCount = ref(0)
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
const sortState = reactive<{ sort_by: UserGroupRateSortField; sort_order: 'asc' | 'desc' }>({
  sort_by: 'updated_at',
  sort_order: 'desc'
})
const searchQuery = ref('')
const groupFilter = ref<number | ''>('')
const statusFilter = ref<string>('')
const groups = ref<UserGroupRateGroupOption[]>([])

const hasActiveFilters = computed(() => searchQuery.value.trim() !== '' || groupFilter.value !== '' || statusFilter.value !== '')

const groupFilterOptions = computed(() => [
  { value: '', label: t('admin.userGroupRates.allGroups') },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])
const statusFilterOptions = computed(() => [
  { value: '', label: t('admin.userGroupRates.allStatuses') },
  { value: 'active', label: t('admin.userGroupRates.status.active') },
  { value: 'disabled', label: t('admin.userGroupRates.status.disabled') }
])

const statusLabel = (status: string) =>
  status === 'active' || status === 'disabled' ? t(`admin.userGroupRates.status.${status}`) : status

function currentQuery(page: number, pageSize: number) {
  return {
    page,
    page_size: pageSize,
    search: searchQuery.value.trim() || undefined,
    group_id: groupFilter.value === '' ? undefined : groupFilter.value,
    user_status: statusFilter.value || undefined,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order
  }
}

let listController: AbortController | null = null

async function loadRates() {
  listController?.abort()
  const controller = new AbortController()
  listController = controller
  loading.value = true
  try {
    const data = await adminAPI.userGroupRates.list(currentQuery(pagination.page, pagination.page_size), {
      signal: controller.signal
    })
    if (controller.signal.aborted) return
    rows.value = data.items
    pagination.total = data.total
    userCount.value = data.user_count
    void loadUsage(data.items)
  } catch (error: unknown) {
    if (controller.signal.aborted) return
    appStore.showError(extractApiErrorMessage(error, t('admin.userGroupRates.loadFailed')))
  } finally {
    if (listController === controller) {
      loading.value = false
      listController = null
    }
  }
}

async function loadGroups() {
  try {
    groups.value = await adminAPI.userGroupRates.groups()
  } catch {
    // 分组下拉只是辅助：失败时列表照常可用
  }
}

// ---------- 近 30 天用量（列表之后再取，失败不影响列表） ----------

const usageMap = ref<Map<string, UserGroupRateUsage>>(new Map())
const usageLoading = ref(false)
const usageFailed = ref(false)
let usageSeq = 0

async function loadUsage(items: UserGroupRateRow[]) {
  const seq = ++usageSeq
  const userIds = [...new Set(items.map((r) => r.user_id))]
  if (userIds.length === 0) {
    usageMap.value = new Map()
    return
  }
  usageLoading.value = true
  usageFailed.value = false
  try {
    const usage = await adminAPI.userGroupRates.usage(userIds)
    if (seq !== usageSeq) return
    usageMap.value = new Map(usage.map((u) => [rowKey(u), u]))
  } catch {
    if (seq !== usageSeq) return
    usageFailed.value = true
  } finally {
    if (seq === usageSeq) usageLoading.value = false
  }
}

const usageFor = (row: UserGroupRateRow) => usageMap.value.get(rowKey(row))

// ---------- 折扣 ----------

const isNotBelowGroup = (row: UserGroupRateRow) =>
  row.group_rate_multiplier > 0 && roundRate(row.rate_multiplier) >= roundRate(row.group_rate_multiplier)

function discountClass(row: UserGroupRateRow) {
  const pct = discountPercent(row.rate_multiplier, row.group_rate_multiplier)
  if (pct === null) return 'text-gray-400'
  if (isNotBelowGroup(row)) return 'font-medium text-amber-600 dark:text-amber-400'
  return 'font-medium text-emerald-600 dark:text-emerald-400'
}

// ---------- 筛选 / 排序 / 分页 ----------

let searchTimer: ReturnType<typeof setTimeout> | undefined

function handleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    clearSelection()
    pagination.page = 1
    loadRates()
  }, 300)
}

function handleFilterChange() {
  clearSelection()
  pagination.page = 1
  loadRates()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  if (!SORTABLE_FIELDS.includes(key as UserGroupRateSortField)) return
  sortState.sort_by = key as UserGroupRateSortField
  sortState.sort_order = order
  pagination.page = 1
  loadRates()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadRates()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  loadRates()
}

// ---------- 勾选（跨页保留，筛选变化时清空） ----------

const selected = ref<Map<string, UserGroupRateRow>>(new Map())
const selectedKeys = computed(() => [...selected.value.keys()])
const selectedRows = computed(() => [...selected.value.values()])
const selectedCount = computed(() => selected.value.size)
const selectingAll = ref(false)

const selectionLabel = (row: UserGroupRateRow) => `${row.user_email} · ${row.group_name}`

function handleSelectedKeys(keys: Array<string | number>) {
  const byKey = new Map(rows.value.map((r) => [rowKey(r), r]))
  const next = new Map<string, UserGroupRateRow>()
  for (const key of keys) {
    const k = String(key)
    const row = byKey.get(k) ?? selected.value.get(k)
    if (row) next.set(k, row)
  }
  selected.value = next
}

function clearSelection() {
  selected.value = new Map()
}

const canSelectAllFiltered = computed(
  () => pagination.total > 0 && pagination.total <= USER_GROUP_RATE_BATCH_MAX && selectedCount.value < pagination.total
)

async function selectAllFiltered() {
  selectingAll.value = true
  try {
    const data = await adminAPI.userGroupRates.list(currentQuery(1, USER_GROUP_RATE_BATCH_MAX))
    selected.value = new Map(data.items.map((r) => [rowKey(r), r]))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.userGroupRates.loadFailed')))
  } finally {
    selectingAll.value = false
  }
}

// ---------- 写操作（全部走批量接口） ----------

const saving = ref(false)
const showBatchDialog = ref(false)
const showAddDialog = ref(false)

/** 提交一批调整。返回 true 表示已生效或已进入审批（调用方应关闭弹窗）。 */
async function applyItems(items: UserGroupRateBatchItem[]): Promise<boolean> {
  if (items.length === 0) return true
  saving.value = true
  try {
    const result = await adminAPI.userGroupRates.applyBatch(items)
    appStore.showSuccess(t('admin.userGroupRates.saved', { updated: result.updated, cleared: result.cleared }))
    clearSelection()
    await loadRates()
    return true
  } catch (error: unknown) {
    // 运维管理员：已排队等待审批（全局提示已弹出）
    if (isApprovalQueued(error)) {
      clearSelection()
      return true
    }
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
    return false
  } finally {
    saving.value = false
  }
}

async function submitBatch(items: UserGroupRateBatchItem[]) {
  if (await applyItems(items)) showBatchDialog.value = false
}

async function submitAdd(items: UserGroupRateBatchItem[]) {
  if (await applyItems(items)) showAddDialog.value = false
}

// 行内编辑
const editingKey = ref<string | null>(null)
const editingValue = ref<number | ''>('')
const editInput = ref<HTMLInputElement | HTMLInputElement[] | null>(null)

async function startEdit(row: UserGroupRateRow) {
  editingKey.value = rowKey(row)
  editingValue.value = row.rate_multiplier
  await nextTick()
  const el = Array.isArray(editInput.value) ? editInput.value[0] : editInput.value
  el?.focus()
  el?.select()
}

function cancelEdit() {
  editingKey.value = null
  editingValue.value = ''
}

async function saveEdit(row: UserGroupRateRow) {
  if (saving.value) return
  const value = editingValue.value
  if (!isValidRate(value)) {
    appStore.showError(t('admin.userGroupRates.invalidRate'))
    return
  }
  if (roundRate(value) === roundRate(row.rate_multiplier)) {
    cancelEdit()
    return
  }
  if (await applyItems([{ user_id: row.user_id, group_id: row.group_id, rate_multiplier: roundRate(value) }])) {
    cancelEdit()
  }
}

// 恢复分组默认
const resetTarget = ref<UserGroupRateRow | null>(null)
const resetMessage = computed(() =>
  resetTarget.value
    ? t('admin.userGroupRates.resetConfirmMessage', {
        user: resetTarget.value.user_email,
        group: resetTarget.value.group_name,
        rate: formatRate(resetTarget.value.group_rate_multiplier)
      })
    : ''
)

async function confirmReset() {
  const row = resetTarget.value
  if (!row) return
  resetTarget.value = null
  await applyItems([{ user_id: row.user_id, group_id: row.group_id, rate_multiplier: null }])
}

onMounted(() => {
  loadGroups()
  loadRates()
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
  listController?.abort()
})
</script>

<style scoped>
.hide-spinner::-webkit-outer-spin-button,
.hide-spinner::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.hide-spinner {
  -moz-appearance: textfield;
}
</style>
