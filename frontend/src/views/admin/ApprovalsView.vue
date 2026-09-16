<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="card p-4 sm:p-6">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex items-center gap-2">
              <button
                v-for="tab in tabs"
                :key="tab.key"
                type="button"
                class="rounded-lg px-3 py-1.5 text-sm font-medium transition-colors"
                :class="activeTab === tab.key
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'"
                :data-test="`approvals-tab-${tab.key}`"
                @click="switchTab(tab.key)"
              >
                {{ tab.label }}
                <span v-if="tab.key === 'pending' && approvalsStore.pendingCount > 0" class="ml-1 rounded-full bg-red-500 px-1.5 text-[10px] font-bold text-white">
                  {{ approvalsStore.pendingCount }}
                </span>
              </button>
            </div>
            <div class="flex flex-wrap items-center gap-3">
              <p v-if="!isAdmin" class="text-xs text-gray-500 dark:text-gray-400">{{ t('operator.approval.operatorHint') }}</p>
              <template v-if="isAdmin && activeTab === 'pending'">
                <button
                  type="button"
                  class="btn btn-primary btn-sm"
                  :disabled="batchRunning || selectedIds.length === 0"
                  data-test="approve-selected"
                  @click="openBatch('selected')"
                >
                  {{ t('operator.approval.actions.approveSelected', { count: selectedIds.length }) }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="batchRunning || total === 0"
                  data-test="approve-all"
                  @click="openBatch('all')"
                >
                  {{ batchRunning ? t('operator.approval.approving') : t('operator.approval.actions.approveAll') }}
                </button>
              </template>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
                {{ t('common.refresh') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="items"
          :loading="loading"
          row-key="id"
          :selectable="isAdmin && activeTab === 'pending'"
          :selected-keys="selectedIds"
          @update:selected-keys="onSelectionChange"
        >
          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-requester="{ row }">
            <div class="min-w-0 max-w-[220px] truncate font-medium text-gray-900 dark:text-white" :title="row.requester?.email">
              {{ row.requester?.email || `#${row.requester?.id ?? '-'}` }}
            </div>
          </template>

          <!-- 操作内容：把脱敏后的请求体翻译成一句话 + 逐项字段，直接摆在列表里，审批时不用点开详情 -->
          <template #cell-action="{ row }">
            <div class="min-w-[260px] max-w-2xl whitespace-normal" :data-test="`summary-${row.id}`">
              <div class="text-sm font-semibold leading-snug text-gray-900 dark:text-white">{{ describe(row).summary }}</div>
              <ul v-if="describe(row).lines.length" class="mt-1 space-y-0.5">
                <li v-for="(line, idx) in describe(row).lines" :key="idx" class="text-xs leading-snug text-gray-700 dark:text-gray-300">
                  <span class="text-gray-400 dark:text-gray-500">{{ line.label }}：</span><span class="break-all">{{ line.value }}</span>
                </li>
              </ul>
              <div class="mt-1 text-[11px] text-gray-400" :title="`${row.method} ${row.request_path}`">
                {{ actionLabel(row.action) }}
              </div>
            </div>
          </template>

          <template #cell-status="{ row }">
            <span :class="statusBadgeClass(row.status)" :data-test="`approval-status-${row.id}`">{{ statusLabel(row.status) }}</span>
            <div v-if="activeTab === 'pending'" class="mt-0.5 whitespace-nowrap text-[11px] text-gray-400">
              {{ t('operator.approval.decision.expiresAt', { time: formatDateTime(row.expires_at) }) }}
            </div>
          </template>

          <!-- 已处理：处理人 / 时间 / 拒绝理由 / 执行结果直接进表格，不用点详情 -->
          <template #cell-decision="{ row }">
            <div class="min-w-[200px] max-w-sm whitespace-normal text-xs leading-snug" :data-test="`decision-${row.id}`">
              <div
                v-for="(line, idx) in decisionLines(row)"
                :key="idx"
                :class="line.tone === 'error' ? 'text-red-600 dark:text-red-400' : line.tone === 'ok' ? 'text-green-700 dark:text-green-400' : 'text-gray-600 dark:text-gray-300'"
              >
                <span class="text-gray-400 dark:text-gray-500">{{ line.label }}：</span><span class="break-words">{{ line.value }}</span>
              </div>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-2">
              <button
                v-if="isAdmin && row.status === 'pending'"
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="actingId === row.id"
                :data-test="`approve-${row.id}`"
                @click="approve(row)"
              >
                {{ actingId === row.id ? t('operator.approval.approving') : t('operator.approval.actions.approve') }}
              </button>
              <button
                v-if="isAdmin && row.status === 'pending'"
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="actingId === row.id"
                :data-test="`reject-${row.id}`"
                @click="openReject(row)"
              >
                {{ t('operator.approval.actions.reject') }}
              </button>
              <!-- 撤回：申请人撤回自己的申请；管理员也可以不给理由地撤回任意待审申请 -->
              <button
                v-if="row.status === 'pending'"
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="actingId === row.id"
                :data-test="`cancel-${row.id}`"
                @click="openCancel(row)"
              >
                {{ t('operator.approval.actions.cancel') }}
              </button>
              <button
                type="button"
                class="inline-flex items-center gap-1 text-sm font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                :data-test="`detail-${row.id}`"
                @click="openDetail(row)"
              >
                {{ t('operator.approval.actions.detail') }}
              </button>
            </div>
          </template>

          <template #empty>
            <div class="flex flex-col items-center py-8">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('operator.approval.empty') }}</p>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="total > 0"
          :total="total"
          :page="page"
          :page-size="pageSize"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- 详情 -->
    <BaseDialog
      :show="detailVisible"
      :title="detail ? t('operator.approval.detail.titleWithId', { id: detail.id }) : t('operator.approval.detail.title')"
      width="wide"
      :close-on-click-outside="true"
      @close="detailVisible = false"
    >
      <div v-if="detailLoading" class="flex items-center justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>
      <div v-else-if="detail" class="space-y-5 py-2">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
            <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('operator.approval.columns.action') }}</div>
            <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ actionLabel(detail.action) }}</div>
            <div class="mt-1 break-all font-mono text-xs text-gray-500">{{ detail.method }} {{ detail.request_path }}<span v-if="detail.request_query">?{{ detail.request_query }}</span></div>
          </div>
          <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
            <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('operator.approval.columns.target') }}</div>
            <div class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-white">{{ detail.target_summary || '—' }}</div>
          </div>
          <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
            <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('operator.approval.columns.requester') }}</div>
            <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ detail.requester?.email || `#${detail.requester?.id ?? '-'}` }}</div>
            <div class="mt-1 text-xs text-gray-500">{{ formatDateTime(detail.created_at) }}<span v-if="detail.requester_ip"> · {{ detail.requester_ip }}</span></div>
          </div>
          <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
            <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('operator.approval.columns.status') }}</div>
            <div class="mt-1"><span :class="statusBadgeClass(detail.status)">{{ statusLabel(detail.status) }}</span></div>
            <div class="mt-1 text-xs text-gray-500">{{ t('operator.approval.detail.expiresAt') }}：{{ formatDateTime(detail.expires_at) }}</div>
          </div>
        </div>

        <div class="rounded-xl border border-primary-100 bg-primary-50/60 p-4 dark:border-primary-900/40 dark:bg-primary-900/10" data-test="detail-summary">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ describe(detail).summary }}</div>
          <ul v-if="describe(detail).lines.length" class="mt-2 space-y-1">
            <li v-for="(line, idx) in describe(detail).lines" :key="idx" class="text-sm text-gray-700 dark:text-gray-300">
              <span class="text-gray-500 dark:text-gray-400">{{ line.label }}：</span><span class="break-all">{{ line.value }}</span>
            </li>
          </ul>
          <p v-else class="mt-1 text-xs text-gray-500">{{ t('operator.approval.noFields') }}</p>
        </div>

        <details class="group">
          <summary class="cursor-pointer select-none text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('operator.approval.rawToggle') }}</summary>
          <pre class="mt-2 max-h-[320px] overflow-auto rounded-xl border border-gray-200 bg-white p-4 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100"><code>{{ prettyJSON(detail.request_body) }}</code></pre>
        </details>

        <div v-if="detail.decided_by || detail.decision_reason" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
          <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('operator.approval.detail.decision') }}</div>
          <div class="mt-1 text-sm text-gray-900 dark:text-white">
            {{ detail.decided_by?.email || '—' }}<span v-if="detail.decided_at"> · {{ formatDateTime(detail.decided_at) }}</span>
          </div>
          <div v-if="detail.decision_reason" class="mt-1 break-words text-sm text-gray-600 dark:text-gray-300">{{ detail.decision_reason }}</div>
        </div>

        <div v-if="detail.executed_at || detail.result_status_code" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
          <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('operator.approval.detail.result') }}</div>
          <div class="mt-1 text-sm text-gray-900 dark:text-white">
            {{ t('operator.approval.detail.statusCode') }}：{{ detail.result_status_code ?? '—' }}
            <span v-if="detail.result_error" class="ml-2 font-mono text-red-500">{{ detail.result_error }}</span>
          </div>
          <pre v-if="detail.result_body" class="mt-2 max-h-[240px] overflow-auto rounded-xl border border-gray-200 bg-white p-3 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100"><code>{{ prettyJSON(detail.result_body) }}</code></pre>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <template v-if="detail && isAdmin && detail.status === 'pending'">
            <button type="button" class="btn btn-secondary" :disabled="actingId === detail.id" @click="openReject(detail)">
              {{ t('operator.approval.actions.reject') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="actingId === detail.id" data-test="detail-approve" @click="approve(detail)">
              {{ actingId === detail.id ? t('operator.approval.approving') : t('operator.approval.actions.approve') }}
            </button>
          </template>
          <button type="button" class="btn btn-secondary" @click="detailVisible = false">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- 拒绝理由 -->
    <BaseDialog :show="rejectVisible" :title="t('operator.approval.rejectTitle')" width="narrow" @close="rejectVisible = false">
      <div class="space-y-3">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ rejectTarget ? describe(rejectTarget).summary : '' }}</p>
        <textarea v-model="rejectReason" rows="3" class="input" maxlength="500" :placeholder="t('operator.approval.rejectReasonPlaceholder')" data-test="reject-reason"></textarea>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="rejectVisible = false">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-danger" :disabled="rejectTarget !== null && actingId === rejectTarget.id" data-test="confirm-reject" @click="confirmReject">
            {{ t('operator.approval.actions.reject') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- 撤回 -->
    <ConfirmDialog
      :show="cancelVisible"
      :title="t('operator.approval.cancelTitle')"
      :message="t('operator.approval.cancelConfirm')"
      danger
      @confirm="confirmCancel"
      @cancel="cancelVisible = false"
    />

    <!-- 批量一键通过（会立即执行，需要确认一次） -->
    <ConfirmDialog
      :show="batchVisible"
      :title="t('operator.approval.actions.approveAll')"
      :message="t('operator.approval.batchConfirm', { count: batchCount })"
      @confirm="confirmBatch"
      @cancel="batchVisible = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type { AdminApprovalRequest, AdminApprovalStatus, ApprovalStatusFilter } from '@/api/admin/approvals'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useApprovalsStore } from '@/stores/approvals'
import { formatDateTime } from '@/utils/format'
import { APPROVAL_QUEUED_EVENT, approvalActionLabelKey } from '@/utils/approval'
import { describeApproval, type ApprovalDescribeContext, type ApprovalDescription } from '@/utils/approvalDescribe'

type TabKey = 'pending' | 'processed'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const approvalsStore = useApprovalsStore()

const isAdmin = computed(() => authStore.isAdmin)

const tabs = computed<Array<{ key: TabKey; label: string }>>(() => [
  { key: 'pending', label: t('operator.approval.tabs.pending') },
  { key: 'processed', label: t('operator.approval.tabs.processed') }
])
const activeTab = ref<TabKey>('pending')

const items = ref<AdminApprovalRequest[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
let abortController: AbortController | null = null

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('operator.approval.columns.time') },
  ...(isAdmin.value ? [{ key: 'requester', label: t('operator.approval.columns.requester') }] : []),
  { key: 'action', label: t('operator.approval.columns.action'), class: 'w-full' },
  { key: 'status', label: t('operator.approval.columns.status') },
  ...(activeTab.value === 'processed' ? [{ key: 'decision', label: t('operator.approval.columns.decision') }] : []),
  { key: 'actions', label: t('operator.approval.columns.actions') }
])

const actionLabel = (action: string): string => {
  const key = approvalActionLabelKey(action)
  return key ? t(key) : action
}

// ---------- 文字化描述 ----------
// 分组 / 用户属性只在请求体里以 id 出现，这里拉一份 id → 名称 映射（两个接口管理员和运维都能读）。
const groupNames = ref<Map<number, string>>(new Map())
const attributeNames = ref<Map<number, string>>(new Map())
const loadLookups = async () => {
  const [groups, attrs] = await Promise.allSettled([
    adminAPI.usage.listFilterGroups(),
    adminAPI.userAttributes.listDefinitions()
  ])
  if (groups.status === 'fulfilled') groupNames.value = new Map((groups.value || []).map((g) => [g.id, g.name]))
  if (attrs.status === 'fulfilled') attributeNames.value = new Map((attrs.value || []).map((a) => [a.id, a.name]))
}
const describeCtx = computed<ApprovalDescribeContext>(() => ({
  t: (key, params) => (params ? t(key, params) : t(key)),
  groupName: (id) => groupNames.value.get(id),
  attributeName: (id) => attributeNames.value.get(id)
}))
const describe = (row: AdminApprovalRequest): ApprovalDescription => describeApproval(row, describeCtx.value)

// ---------- 已处理：处理结果 ----------
interface DecisionLine {
  label: string
  value: string
  tone?: 'ok' | 'error'
}

const actorLabel = (row: AdminApprovalRequest): string => row.decided_by?.email || (row.decided_by ? `#${row.decided_by.id}` : '-')

const decisionLines = (row: AdminApprovalRequest): DecisionLine[] => {
  const D = 'operator.approval.decision'
  const lines: DecisionLine[] = []
  const when = row.decided_at ? formatDateTime(row.decided_at) : ''
  switch (row.status) {
    case 'approved':
    case 'failed':
    case 'executing':
      lines.push({ label: t(`${D}.approvedBy`), value: when ? `${actorLabel(row)} · ${when}` : actorLabel(row) })
      if (row.status === 'approved') {
        lines.push({
          label: t(`${D}.result`),
          value: t(`${D}.executedOk`, { code: row.result_status_code ?? '-', time: row.executed_at ? formatDateTime(row.executed_at) : '' }),
          tone: 'ok'
        })
      } else if (row.status === 'failed') {
        lines.push({
          label: t(`${D}.result`),
          value: t(`${D}.executedFailed`, { error: row.result_error || '-', code: row.result_status_code ?? '-' }),
          tone: 'error'
        })
      } else {
        lines.push({ label: t(`${D}.result`), value: t('operator.approval.status.executing') })
      }
      break
    case 'rejected':
      lines.push({ label: t(`${D}.rejectedBy`), value: when ? `${actorLabel(row)} · ${when}` : actorLabel(row) })
      lines.push({ label: t(`${D}.reason`), value: row.decision_reason || t(`${D}.noReason`) })
      break
    case 'cancelled':
      lines.push({ label: t(`${D}.cancelledBy`), value: when ? `${actorLabel(row)} · ${when}` : actorLabel(row) })
      break
    case 'expired':
      lines.push({ label: t(`${D}.expired`), value: formatDateTime(row.expires_at) })
      break
    default:
      lines.push({ label: t('operator.approval.detail.expiresAt'), value: formatDateTime(row.expires_at) })
  }
  return lines
}

const statusLabel = (status: AdminApprovalStatus | string): string => {
  const known: AdminApprovalStatus[] = ['pending', 'executing', 'approved', 'failed', 'rejected', 'cancelled', 'expired']
  return known.includes(status as AdminApprovalStatus) ? t(`operator.approval.status.${status}`) : String(status)
}

const statusBadgeClass = (status: string): string => {
  const base = 'inline-flex items-center rounded-lg px-2 py-1 text-xs font-bold ring-1 ring-inset '
  switch (status) {
    case 'approved':
      return base + 'bg-green-50 text-green-700 ring-green-600/20 dark:bg-green-900/30 dark:text-green-400'
    case 'failed':
    case 'rejected':
      return base + 'bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-900/30 dark:text-red-400'
    case 'pending':
    case 'executing':
      return base + 'bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-900/30 dark:text-amber-400'
    default:
      return base + 'bg-gray-50 text-gray-600 ring-gray-500/20 dark:bg-dark-700 dark:text-gray-300'
  }
}

const prettyJSON = (raw?: string): string => {
  if (!raw) return '—'
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

const load = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const status: ApprovalStatusFilter = activeTab.value
    const res = await adminAPI.approvals.list(page.value, pageSize.value, { status }, { signal: controller.signal })
    if (controller.signal.aborted) return
    items.value = res.items || []
    total.value = res.total || 0
  } catch (err: any) {
    if (controller.signal.aborted || err?.name === 'CanceledError' || err?.code === 'ERR_CANCELED') return
    console.error('[ApprovalsView] failed to load approvals', err)
    appStore.showError(err?.message || t('operator.approval.loadFailed'))
  } finally {
    if (abortController === controller) {
      loading.value = false
      abortController = null
    }
  }
}

const switchTab = (tab: TabKey) => {
  if (activeTab.value === tab) return
  activeTab.value = tab
  page.value = 1
  selectedIds.value = []
  void load()
}

// ---------- 批量一键通过（管理员） ----------
const selectedIds = ref<number[]>([])
const onSelectionChange = (keys: Array<string | number>) => {
  selectedIds.value = keys.map((k) => Number(k)).filter((n) => Number.isFinite(n) && n > 0)
}

type BatchMode = 'selected' | 'all'
const batchVisible = ref(false)
const batchMode = ref<BatchMode>('selected')
const batchCount = ref(0)
const batchRunning = ref(false)
const BATCH_CHUNK = 50
const BATCH_ALL_LIMIT = 100

const openBatch = (mode: BatchMode) => {
  batchMode.value = mode
  batchCount.value = mode === 'selected' ? selectedIds.value.length : total.value
  if (batchCount.value === 0) {
    appStore.showInfo(t('operator.approval.batchEmpty'))
    return
  }
  batchVisible.value = true
}

const collectBatchIds = async (): Promise<number[]> => {
  if (batchMode.value === 'selected') return [...selectedIds.value]
  // 一键通过全部：取当前全部待审（最多 100 条），而不是只看当前页
  const res = await adminAPI.approvals.list(1, BATCH_ALL_LIMIT, { status: 'pending' })
  return (res.items || []).filter((item) => item.status === 'pending').map((item) => item.id)
}

const confirmBatch = async () => {
  if (batchRunning.value) return
  batchVisible.value = false
  batchRunning.value = true
  try {
    const ids = await collectBatchIds()
    if (ids.length === 0) {
      appStore.showInfo(t('operator.approval.batchEmpty'))
      return
    }
    let approved = 0
    let failed = 0
    let skipped = 0
    for (let i = 0; i < ids.length; i += BATCH_CHUNK) {
      const res = await adminAPI.approvals.batchApprove(ids.slice(i, i + BATCH_CHUNK))
      approved += res.approved
      failed += res.failed
      skipped += res.skipped
    }
    const summary = t('operator.approval.batchResult', { approved, failed, skipped })
    if (failed > 0) appStore.showError(summary)
    else appStore.showSuccess(summary)
    selectedIds.value = []
    await Promise.all([load(), approvalsStore.fetchPendingCount()])
  } catch (err: any) {
    appStore.showError(err?.message || t('operator.approval.loadFailed'))
    await load()
  } finally {
    batchRunning.value = false
  }
}

const onPageChange = (next: number) => {
  page.value = next
  void load()
}

const onPageSizeChange = (next: number) => {
  pageSize.value = next
  page.value = 1
  void load()
}

// ---------- 详情 ----------
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<AdminApprovalRequest | null>(null)

const openDetail = async (row: AdminApprovalRequest) => {
  detail.value = row
  detailVisible.value = true
  detailLoading.value = true
  try {
    detail.value = await adminAPI.approvals.get(row.id)
  } catch (err: any) {
    console.error('[ApprovalsView] failed to load approval detail', err)
  } finally {
    detailLoading.value = false
  }
}

// ---------- 通过 / 拒绝 / 撤回 ----------
const actingId = ref<number | null>(null)

const afterDecision = async (updated: AdminApprovalRequest) => {
  if (detail.value && detail.value.id === updated.id) detail.value = updated
  selectedIds.value = selectedIds.value.filter((id) => id !== updated.id)
  await Promise.all([load(), approvalsStore.fetchPendingCount()])
}

const approve = async (row: AdminApprovalRequest) => {
  if (actingId.value !== null) return
  actingId.value = row.id
  try {
    const res = await adminAPI.approvals.approve(row.id)
    if (res.approval.status === 'approved') {
      appStore.showSuccess(t('operator.approval.approveSuccess', { target: res.approval.target_summary }))
    } else {
      appStore.showError(t('operator.approval.approveFailed', { error: res.approval.result_error || res.replay?.status_code || '' }))
    }
    await afterDecision(res.approval)
  } catch (err: any) {
    appStore.showError(err?.message || t('operator.approval.approveFailed', { error: '' }))
    await load()
  } finally {
    actingId.value = null
  }
}

const rejectVisible = ref(false)
const rejectTarget = ref<AdminApprovalRequest | null>(null)
const rejectReason = ref('')

const openReject = (row: AdminApprovalRequest) => {
  rejectTarget.value = row
  rejectReason.value = ''
  rejectVisible.value = true
}

const confirmReject = async () => {
  const target = rejectTarget.value
  if (!target || actingId.value !== null) return
  actingId.value = target.id
  try {
    const res = await adminAPI.approvals.reject(target.id, rejectReason.value.trim())
    rejectVisible.value = false
    appStore.showSuccess(t('operator.approval.rejected'))
    await afterDecision(res.approval)
  } catch (err: any) {
    appStore.showError(err?.message || t('operator.approval.loadFailed'))
  } finally {
    actingId.value = null
  }
}

const cancelVisible = ref(false)
const cancelTarget = ref<AdminApprovalRequest | null>(null)

const openCancel = (row: AdminApprovalRequest) => {
  cancelTarget.value = row
  cancelVisible.value = true
}

const confirmCancel = async () => {
  const target = cancelTarget.value
  if (!target || actingId.value !== null) return
  actingId.value = target.id
  try {
    const res = await adminAPI.approvals.cancel(target.id)
    cancelVisible.value = false
    appStore.showSuccess(t('operator.approval.cancelled'))
    await afterDecision(res.approval)
  } catch (err: any) {
    appStore.showError(err?.message || t('operator.approval.loadFailed'))
  } finally {
    actingId.value = null
    cancelTarget.value = null
  }
}

// ---------- 深链与事件 ----------
const onQueued = () => {
  if (activeTab.value === 'pending') void load()
}

const openDeepLink = async () => {
  const raw = route.query.id
  const id = Number.parseInt(typeof raw === 'string' ? raw : Array.isArray(raw) ? String(raw[0] ?? '') : '', 10)
  if (!Number.isFinite(id) || id <= 0) return
  detailVisible.value = true
  detailLoading.value = true
  try {
    const found = await adminAPI.approvals.get(id)
    detail.value = found
    const tab: TabKey = found.status === 'pending' || found.status === 'executing' ? 'pending' : 'processed'
    if (tab !== activeTab.value) {
      activeTab.value = tab
      page.value = 1
      void load()
    }
  } catch (err: any) {
    detailVisible.value = false
    appStore.showError(err?.message || t('operator.approval.loadFailed'))
  } finally {
    detailLoading.value = false
  }
}

onMounted(() => {
  window.addEventListener(APPROVAL_QUEUED_EVENT, onQueued)
  void loadLookups()
  void load()
  void approvalsStore.fetchPendingCount()
  void openDeepLink()
})

onBeforeUnmount(() => {
  window.removeEventListener(APPROVAL_QUEUED_EVENT, onQueued)
  abortController?.abort()
})
</script>
