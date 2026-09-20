<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="card p-4 sm:p-6">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex flex-wrap items-center gap-2">
              <button
                v-for="tab in tabs"
                :key="tab.key"
                type="button"
                class="rounded-lg px-3 py-1.5 text-sm font-medium transition-colors"
                :class="activeTab === tab.key
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'"
                :data-test="`tickets-tab-${tab.key}`"
                @click="switchTab(tab.key)"
              >
                {{ tab.label }}
                <span
                  v-if="tab.key === 'open' && ticketsStore.openCount > 0"
                  class="ml-1 rounded-full bg-red-500 px-1.5 text-[10px] font-bold text-white"
                >
                  {{ ticketsStore.openCount }}
                </span>
              </button>
            </div>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
              {{ t('common.refresh') }}
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="card p-4">
          <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
            <input
              v-model="search"
              type="text"
              class="input md:col-span-2"
              :placeholder="t('tickets.admin.searchPlaceholder')"
              data-test="tickets-search"
            />
            <Select v-model="category" :options="categoryOptions" :placeholder="t('tickets.admin.categoryAll')" />
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="items"
          :loading="loading"
          row-key="id"
          :sticky-right-columns="['status']"
          clickable-rows
          @row-click="openDetail"
        >
          <template #cell-user="{ row }">
            <div class="min-w-0 max-w-[220px] truncate text-gray-900 dark:text-white" :title="row.user?.email">
              {{ row.user?.email || `#${row.user?.id ?? '-'}` }}
            </div>
          </template>
          <template #cell-title="{ row }">
            <div class="min-w-[220px] max-w-md truncate font-medium text-gray-900 dark:text-white" :title="row.title">
              {{ row.title }}
            </div>
          </template>
          <template #cell-category="{ value }">
            <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ ticketCategoryLabel(t, value) }}</span>
          </template>
          <template #cell-last_message_at="{ value }">
            <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-status="{ value, row }">
            <span :class="ticketStatusBadgeClass(value)" :data-test="`ticket-status-${row.id}`">{{ ticketStatusLabel(t, value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end">
              <button type="button" class="btn btn-primary btn-sm" :data-test="`ticket-view-${row.id}`" @click.stop="openDetail(row)">
                {{ t('tickets.actions.view') }}
              </button>
            </div>
          </template>
          <template #empty>
            <EmptyState :title="t('tickets.admin.empty')" />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          :total="total"
          :page="page"
          :page-size="pageSize"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- 工单详情（admin 与 operator 同权，按钮不区分角色） -->
    <BaseDialog :show="detailVisible" :title="detailTitle" width="wide" @close="closeDetail">
      <div v-if="detail" class="space-y-4">
        <div class="flex flex-wrap items-center gap-3 text-sm text-gray-500 dark:text-gray-400">
          <span :class="ticketStatusBadgeClass(detail.status)" data-test="ticket-detail-status">{{ ticketStatusLabel(t, detail.status) }}</span>
          <span>{{ ticketCategoryLabel(t, detail.category) }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-200" data-test="ticket-detail-user">{{ detail.user?.email }}</span>
          <span>{{ formatDateTime(detail.created_at) }}</span>
        </div>
        <TicketThread
          ref="threadRef"
          :ticket="detail"
          :messages="messages"
          viewer="staff"
          :submitting="replying"
          :loading="detailLoading"
          @reply="sendReply"
        />
      </div>
      <div v-else class="py-6 text-center text-sm text-gray-400">{{ t('tickets.loading') }}</div>
      <template #footer>
        <button
          v-if="detail && detail.status !== 'closed'"
          type="button"
          class="btn btn-danger"
          :disabled="acting"
          data-test="ticket-close"
          @click="closeVisible = true"
        >
          {{ t('tickets.actions.close') }}
        </button>
        <button
          v-else-if="detail"
          type="button"
          class="btn btn-secondary"
          :disabled="acting"
          data-test="ticket-reopen"
          @click="reopenTicket"
        >
          {{ t('tickets.actions.reopen') }}
        </button>
        <button type="button" class="btn btn-secondary" @click="closeDetail">{{ t('tickets.actions.dismiss') }}</button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="closeVisible"
      :title="t('tickets.confirmClose.title')"
      :message="t('tickets.confirmClose.message')"
      danger
      @confirm="closeTicket"
      @cancel="closeVisible = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import TicketThread from '@/components/tickets/TicketThread.vue'
import type { Column } from '@/components/common/types'
import { adminAPI } from '@/api/admin'
import type { AdminSupportTicket } from '@/api/admin/tickets'
import type { SupportTicketMessage, TicketCategory, TicketStatus } from '@/api/tickets'
import { useAppStore } from '@/stores/app'
import { useTicketsStore } from '@/stores/tickets'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  TICKET_CATEGORIES,
  parseTicketDeepLinkId,
  ticketCategoryLabel,
  ticketStatusBadgeClass,
  ticketStatusLabel
} from '@/utils/tickets'

type TabKey = TicketStatus | 'all'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const ticketsStore = useTicketsStore()

// ---------- 列表 ----------
const tabs = computed<Array<{ key: TabKey; label: string }>>(() => [
  { key: 'open', label: t('tickets.admin.tabs.open') },
  { key: 'replied', label: t('tickets.admin.tabs.replied') },
  { key: 'closed', label: t('tickets.admin.tabs.closed') },
  { key: 'all', label: t('tickets.admin.tabs.all') }
])
const activeTab = ref<TabKey>('open')
const search = ref('')
const category = ref<TicketCategory | ''>('')

const categoryOptions = computed(() => [
  { value: '', label: t('tickets.admin.categoryAll') },
  ...TICKET_CATEGORIES.map((value) => ({ value, label: ticketCategoryLabel(t, value) }))
])

const items = ref<AdminSupportTicket[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
let abortController: AbortController | null = null
let searchTimer: number | null = null

const columns = computed<Column[]>(() => [
  { key: 'id', label: t('tickets.columns.id') },
  { key: 'user', label: t('tickets.columns.user') },
  { key: 'title', label: t('tickets.columns.title'), class: 'w-full' },
  { key: 'category', label: t('tickets.columns.category') },
  { key: 'message_count', label: t('tickets.columns.messages') },
  { key: 'last_message_at', label: t('tickets.columns.lastMessage') },
  { key: 'status', label: t('tickets.columns.status') },
  { key: 'actions', label: t('tickets.columns.actions') }
])

const load = async () => {
  abortController?.abort()
  abortController = new AbortController()
  loading.value = true
  try {
    const res = await adminAPI.tickets.list(
      page.value,
      pageSize.value,
      {
        status: activeTab.value === 'all' ? '' : activeTab.value,
        category: category.value,
        search: search.value
      },
      { signal: abortController.signal }
    )
    items.value = res.items ?? []
    total.value = res.total ?? 0
  } catch (err: any) {
    if (err?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('tickets.loadFailed')))
  } finally {
    loading.value = false
  }
}

const switchTab = (key: TabKey) => {
  if (activeTab.value === key) return
  activeTab.value = key
  page.value = 1
  void load()
}

const onPageChange = (next: number) => {
  page.value = next
  void load()
}

const onPageSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  void load()
}

watch(category, () => {
  page.value = 1
  void load()
})

watch(search, () => {
  if (searchTimer !== null) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    searchTimer = null
    page.value = 1
    void load()
  }, 300)
})

const syncRow = (ticket: AdminSupportTicket) => {
  const idx = items.value.findIndex((item) => item.id === ticket.id)
  if (idx >= 0) items.value.splice(idx, 1, { ...items.value[idx], ...ticket })
}

// 状态变化后工单可能离开当前 tab：刷新列表与角标
const afterMutation = async (ticket: AdminSupportTicket) => {
  detail.value = ticket
  syncRow(ticket)
  await Promise.all([load(), ticketsStore.fetchCounts()])
}

// ---------- 详情 / 回复 / 关闭 / 重开 ----------
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<AdminSupportTicket | null>(null)
const messages = ref<SupportTicketMessage[]>([])
const replying = ref(false)
const acting = ref(false)
const closeVisible = ref(false)
const threadRef = ref<InstanceType<typeof TicketThread> | null>(null)

const detailTitle = computed(() =>
  detail.value ? `${t('tickets.detail.titleWithId', { id: detail.value.id })} · ${detail.value.title}` : t('tickets.detail.title')
)

const openDetailById = async (id: number) => {
  detailVisible.value = true
  detailLoading.value = true
  try {
    const res = await adminAPI.tickets.get(id)
    detail.value = res.ticket
    messages.value = res.messages ?? []
    syncRow(res.ticket)
  } catch (err: any) {
    detailVisible.value = false
    detail.value = null
    appStore.showError(extractApiErrorMessage(err, t('tickets.loadFailed')))
  } finally {
    detailLoading.value = false
  }
}

const openDetail = (row: AdminSupportTicket) => {
  void openDetailById(row.id)
}

const closeDetail = () => {
  detailVisible.value = false
  detail.value = null
  messages.value = []
}

const sendReply = async (body: string) => {
  const current = detail.value
  if (!current || replying.value) return
  replying.value = true
  try {
    const res = await adminAPI.tickets.reply(current.id, body)
    messages.value.push(res.message)
    threadRef.value?.clearDraft()
    appStore.showSuccess(t('tickets.toast.replied'))
    await afterMutation(res.ticket)
  } catch (err: any) {
    appStore.showError(extractApiErrorMessage(err, t('tickets.loadFailed'), { TICKET_CLOSED: t('tickets.errors.closed') }))
    if (err?.reason === 'TICKET_CLOSED' || err?.status === 409) await openDetailById(current.id)
  } finally {
    replying.value = false
  }
}

const closeTicket = async () => {
  closeVisible.value = false
  const current = detail.value
  if (!current || acting.value) return
  acting.value = true
  try {
    const updated = await adminAPI.tickets.close(current.id)
    appStore.showSuccess(t('tickets.toast.closed'))
    await afterMutation(updated)
  } catch (err: any) {
    appStore.showError(extractApiErrorMessage(err, t('tickets.loadFailed'), { TICKET_CLOSED: t('tickets.errors.closed') }))
  } finally {
    acting.value = false
  }
}

const reopenTicket = async () => {
  const current = detail.value
  if (!current || acting.value) return
  acting.value = true
  try {
    const updated = await adminAPI.tickets.reopen(current.id)
    appStore.showSuccess(t('tickets.toast.reopened'))
    await afterMutation(updated)
  } catch (err: any) {
    appStore.showError(extractApiErrorMessage(err, t('tickets.loadFailed'), { TICKET_NOT_CLOSED: t('tickets.errors.notClosed') }))
  } finally {
    acting.value = false
  }
}

// ---------- 深链（Server酱³ 推送里的 /admin/tickets?id=N） ----------
const openDeepLink = async () => {
  const id = parseTicketDeepLinkId(route.query.id)
  if (id === null) return
  await openDetailById(id)
  const found = detail.value
  if (found && activeTab.value !== 'all' && found.status !== activeTab.value) {
    activeTab.value = found.status
    page.value = 1
    void load()
  }
}

onMounted(() => {
  void load()
  void ticketsStore.fetchCounts()
  void openDeepLink()
})

onBeforeUnmount(() => {
  abortController?.abort()
  if (searchTimer !== null) window.clearTimeout(searchTimer)
})
</script>
