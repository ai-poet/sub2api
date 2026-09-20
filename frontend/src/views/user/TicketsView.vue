<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] space-y-6">
      <section class="card p-4 sm:p-6">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('tickets.caption') }}</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('tickets.intro') }}</p>
          </div>
          <div class="flex items-center gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
              {{ t('common.refresh') }}
            </button>
            <button type="button" class="btn btn-primary btn-sm" data-test="ticket-create" @click="openCreate">
              {{ t('tickets.create') }}
            </button>
          </div>
        </div>
      </section>

      <section class="card p-4 sm:p-6">
        <div class="mb-4 flex flex-wrap items-center gap-2">
          <button
            v-for="tab in statusTabs"
            :key="tab.key"
            type="button"
            class="rounded-lg px-3 py-1.5 text-sm font-medium transition-colors"
            :class="activeStatus === tab.key
              ? 'bg-primary-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'"
            :data-test="`tickets-tab-${tab.key}`"
            @click="switchStatus(tab.key)"
          >
            {{ tab.label }}
          </button>
        </div>

        <DataTable :columns="columns" :data="items" :loading="loading" row-key="id" clickable-rows @row-click="openDetail">
          <template #cell-title="{ row }">
            <div class="flex min-w-[200px] items-center gap-2">
              <span
                v-if="row.user_unread"
                class="h-2 w-2 flex-shrink-0 rounded-full bg-red-500"
                :data-test="`ticket-unread-${row.id}`"
                :title="t('tickets.thread.unread')"
              />
              <span class="truncate font-medium text-gray-900 dark:text-white">{{ row.title }}</span>
            </div>
          </template>
          <template #cell-category="{ value }">
            <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ ticketCategoryLabel(t, value) }}</span>
          </template>
          <template #cell-status="{ value, row }">
            <span :class="ticketStatusBadgeClass(value)" :data-test="`ticket-status-${row.id}`">{{ ticketStatusLabel(t, value) }}</span>
          </template>
          <template #cell-last_message_at="{ value }">
            <span class="whitespace-nowrap text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end">
              <button type="button" class="btn btn-secondary btn-sm" :data-test="`ticket-view-${row.id}`" @click.stop="openDetail(row)">
                {{ t('tickets.actions.view') }}
              </button>
            </div>
          </template>
          <template #empty>
            <EmptyState :title="t('tickets.empty')" :description="t('tickets.intro')" />
          </template>
        </DataTable>

        <div class="mt-4">
          <Pagination
            :total="total"
            :page="page"
            :page-size="pageSize"
            @update:page="onPageChange"
            @update:pageSize="onPageSizeChange"
          />
        </div>
      </section>
    </div>

    <!-- 新建工单 -->
    <BaseDialog :show="createVisible" :title="t('tickets.createTitle')" width="normal" @close="createVisible = false">
      <div class="space-y-4">
        <Input
          v-model="form.title"
          :label="t('tickets.form.title')"
          :placeholder="t('tickets.form.titlePlaceholder')"
          :hint="`${form.title.length} / ${TICKET_TITLE_MAX}`"
          required
        />
        <div>
          <label class="input-label">{{ t('tickets.form.category') }}</label>
          <Select v-model="form.category" :options="categoryOptions" />
        </div>
        <TextArea
          v-model="form.body"
          :label="t('tickets.form.body')"
          :placeholder="t('tickets.form.bodyPlaceholder')"
          :hint="`${t('tickets.form.bodyHint', { max: TICKET_BODY_MAX })} · ${form.body.length} / ${TICKET_BODY_MAX}`"
          :rows="8"
          required
          @paste="onCreatePaste"
          @drop="onCreateDrop"
          @dragover.prevent
        />
        <div class="flex items-center gap-3">
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :title="t('tickets.attachments.hint')"
            data-test="ticket-create-insert-image"
            @click="pickCreateImage"
          >
            {{ t('tickets.attachments.insertImage') }}
          </button>
          <span v-if="createUploadingCount > 0" class="text-xs text-gray-400 dark:text-gray-500" data-test="ticket-create-uploading">
            {{ t('tickets.attachments.uploading') }}
          </span>
        </div>
        <input
          ref="createFileInput"
          type="file"
          accept="image/*"
          multiple
          class="hidden"
          data-test="ticket-create-attachment-input"
          @change="onCreateFileChange"
        />
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="createVisible = false">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="!canSubmit" data-test="ticket-submit" @click="submitCreate">
          {{ creating ? t('tickets.form.submitting') : t('tickets.form.submit') }}
        </button>
      </template>
    </BaseDialog>

    <!-- 工单详情 -->
    <BaseDialog :show="detailVisible" :title="detailTitle" width="wide" @close="closeDetail">
      <div v-if="detail" class="space-y-4">
        <div class="flex flex-wrap items-center gap-3 text-sm text-gray-500 dark:text-gray-400">
          <span :class="ticketStatusBadgeClass(detail.status)" data-test="ticket-detail-status">{{ ticketStatusLabel(t, detail.status) }}</span>
          <span>{{ ticketCategoryLabel(t, detail.category) }}</span>
          <span>{{ formatDateTime(detail.created_at) }}</span>
        </div>
        <TicketThread
          ref="threadRef"
          :ticket="detail"
          :messages="messages"
          viewer="user"
          side="user"
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
import { computed, onBeforeUnmount, onMounted, reactive, ref, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Input from '@/components/common/Input.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import TicketThread from '@/components/tickets/TicketThread.vue'
import { useTicketAttachments } from '@/components/tickets/useTicketAttachments'
import type { Column } from '@/components/common/types'
import ticketsAPI, { type SupportTicket, type SupportTicketMessage, type TicketCategory, type TicketStatus } from '@/api/tickets'
import { useAppStore } from '@/stores/app'
import { useTicketsStore } from '@/stores/tickets'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  TICKET_BODY_MAX,
  TICKET_CATEGORIES,
  TICKET_TITLE_MAX,
  parseTicketDeepLinkId,
  ticketCategoryLabel,
  ticketStatusBadgeClass,
  ticketStatusLabel
} from '@/utils/tickets'

type StatusTab = 'all' | TicketStatus

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const ticketsStore = useTicketsStore()

// ---------- 列表 ----------
const statusTabs = computed<Array<{ key: StatusTab; label: string }>>(() => [
  { key: 'all', label: t('tickets.statusFilter.all') },
  { key: 'open', label: t('tickets.status.open') },
  { key: 'replied', label: t('tickets.status.replied') },
  { key: 'closed', label: t('tickets.status.closed') }
])
const activeStatus = ref<StatusTab>('all')

const items = ref<SupportTicket[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
let abortController: AbortController | null = null

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('tickets.columns.title'), class: 'w-full' },
  { key: 'category', label: t('tickets.columns.category') },
  { key: 'status', label: t('tickets.columns.status') },
  { key: 'message_count', label: t('tickets.columns.messages') },
  { key: 'last_message_at', label: t('tickets.columns.lastMessage') },
  { key: 'actions', label: t('tickets.columns.actions') }
])

const load = async () => {
  abortController?.abort()
  abortController = new AbortController()
  loading.value = true
  try {
    const res = await ticketsAPI.list(
      page.value,
      pageSize.value,
      { status: activeStatus.value === 'all' ? '' : activeStatus.value },
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

const switchStatus = (key: StatusTab) => {
  if (activeStatus.value === key) return
  activeStatus.value = key
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

// 详情里的变更同步回列表行，避免再拉一次列表
const syncRow = (ticket: SupportTicket) => {
  const idx = items.value.findIndex((item) => item.id === ticket.id)
  if (idx >= 0) items.value.splice(idx, 1, { ...items.value[idx], ...ticket })
}

// ---------- 新建 ----------
const createVisible = ref(false)
const creating = ref(false)
const form = reactive<{ title: string; category: TicketCategory; body: string }>({ title: '', category: 'other', body: '' })

// 新建表单的图片附件上传（side='user'），与线程回复框共用同一套逻辑
const {
  fileInput: createFileInput,
  uploadingCount: createUploadingCount,
  pickImage: pickCreateImage,
  onFileChange: onCreateFileChange,
  onPaste: onCreatePaste,
  onDrop: onCreateDrop
} = useTicketAttachments({ side: 'user', draft: toRef(form, 'body') })

const categoryOptions = computed(() => TICKET_CATEGORIES.map((value) => ({ value, label: ticketCategoryLabel(t, value) })))

const canSubmit = computed(() => {
  const title = form.title.trim()
  const body = form.body.trim()
  return (
    !creating.value &&
    createUploadingCount.value === 0 &&
    title.length > 0 &&
    title.length <= TICKET_TITLE_MAX &&
    body.length > 0 &&
    body.length <= TICKET_BODY_MAX
  )
})

const openCreate = () => {
  form.title = ''
  form.category = 'other'
  form.body = ''
  createVisible.value = true
}

const submitCreate = async () => {
  if (!canSubmit.value) return
  creating.value = true
  try {
    const created = await ticketsAPI.create({ title: form.title.trim(), category: form.category, body: form.body.trim() })
    createVisible.value = false
    appStore.showSuccess(t('tickets.toast.created'))
    page.value = 1
    await load()
    await openDetailById(created.id)
  } catch (err: any) {
    appStore.showError(
      extractApiErrorMessage(err, t('tickets.loadFailed'), { TICKET_OPEN_LIMIT: t('tickets.errors.openLimit') })
    )
  } finally {
    creating.value = false
  }
}

// ---------- 详情 / 回复 / 关闭 / 重开 ----------
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<SupportTicket | null>(null)
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
    const res = await ticketsAPI.get(id)
    detail.value = res.ticket
    messages.value = res.messages ?? []
    syncRow(res.ticket)
    // 服务端在返回详情时已清掉未读标记，刷新角标
    void ticketsStore.fetchCounts()
  } catch (err: any) {
    detailVisible.value = false
    detail.value = null
    appStore.showError(extractApiErrorMessage(err, t('tickets.loadFailed')))
  } finally {
    detailLoading.value = false
  }
}

const openDetail = (row: SupportTicket) => {
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
    const res = await ticketsAPI.reply(current.id, body)
    messages.value.push(res.message)
    detail.value = res.ticket
    syncRow(res.ticket)
    threadRef.value?.clearDraft()
    appStore.showSuccess(t('tickets.toast.replied'))
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
    const updated = await ticketsAPI.close(current.id)
    detail.value = updated
    syncRow(updated)
    appStore.showSuccess(t('tickets.toast.closed'))
    void load()
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
    const updated = await ticketsAPI.reopen(current.id)
    detail.value = updated
    syncRow(updated)
    appStore.showSuccess(t('tickets.toast.reopened'))
    void load()
  } catch (err: any) {
    appStore.showError(
      extractApiErrorMessage(err, t('tickets.loadFailed'), {
        TICKET_NOT_CLOSED: t('tickets.errors.notClosed'),
        TICKET_OPEN_LIMIT: t('tickets.errors.openLimit')
      })
    )
  } finally {
    acting.value = false
  }
}

// ---------- 深链 ----------
const openDeepLink = () => {
  const id = parseTicketDeepLinkId(route.query.id)
  if (id === null) return
  void openDetailById(id)
}

onMounted(() => {
  void load()
  openDeepLink()
})

onBeforeUnmount(() => {
  abortController?.abort()
})
</script>
