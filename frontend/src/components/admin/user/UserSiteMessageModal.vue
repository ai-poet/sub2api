<template>
  <BaseDialog
    :show="show"
    :title="user ? t('admin.users.siteMessage.title', { email: user.email }) : t('admin.users.siteMessage.menuItem')"
    width="wide"
    @close="$emit('close')"
  >
    <div v-if="user" class="space-y-5">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100">
          <span class="text-lg font-medium text-primary-700">{{ user.email.charAt(0).toUpperCase() }}</span>
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.users.siteMessage.recipient') }}</p>
          <p class="truncate font-medium text-gray-900 dark:text-gray-100">{{ user.email }}</p>
        </div>
      </div>

      <p v-if="user.status === 'disabled'" class="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('admin.users.siteMessage.disabledHint') }}
      </p>
      <p v-if="readonly" data-testid="site-message-operator-hint" class="rounded-lg bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:bg-blue-900/20 dark:text-blue-300">
        {{ t('admin.users.siteMessage.operatorHint') }}
      </p>

      <form id="site-message-form" class="space-y-4" @submit.prevent="handleSubmit">
        <div>
          <label class="input-label" for="site-message-title">{{ t('admin.users.siteMessage.titleLabel') }}</label>
          <input
            id="site-message-title"
            v-model="form.title"
            type="text"
            class="input"
            :maxlength="TITLE_MAX"
            :placeholder="t('admin.users.siteMessage.titlePlaceholder')"
          />
          <p class="mt-1 text-right text-xs text-gray-400">{{ runeLength(form.title) }} / {{ TITLE_MAX }}</p>
        </div>
        <div>
          <div class="mb-1 flex items-center justify-between">
            <label class="input-label mb-0" for="site-message-content">{{ t('admin.users.siteMessage.contentLabel') }}</label>
            <div class="inline-flex rounded-lg bg-gray-100 p-0.5 text-xs dark:bg-dark-800">
              <button
                type="button"
                class="rounded-md px-2 py-0.5"
                :class="!previewing ? 'bg-white shadow-sm dark:bg-dark-700' : 'text-gray-500'"
                @click="previewing = false"
              >
                {{ t('admin.users.siteMessage.edit') }}
              </button>
              <button
                type="button"
                data-testid="site-message-preview-toggle"
                class="rounded-md px-2 py-0.5"
                :class="previewing ? 'bg-white shadow-sm dark:bg-dark-700' : 'text-gray-500'"
                @click="previewing = true"
              >
                {{ t('admin.users.siteMessage.preview') }}
              </button>
            </div>
          </div>
          <textarea
            v-show="!previewing"
            id="site-message-content"
            v-model="form.content"
            rows="8"
            class="input font-mono text-sm"
            :maxlength="CONTENT_MAX"
            :placeholder="t('admin.users.siteMessage.contentPlaceholder')"
          ></textarea>
          <div
            v-if="previewing"
            class="min-h-[10rem] rounded-xl border border-gray-200 p-4 text-sm dark:border-dark-600"
          >
            <MarkdownRenderer v-if="form.content.trim()" :content="form.content" />
            <p v-else class="text-gray-400">{{ t('admin.users.siteMessage.previewEmpty') }}</p>
          </div>
          <p class="mt-1 text-right text-xs text-gray-400">{{ runeLength(form.content) }} / {{ CONTENT_MAX }}</p>
        </div>
      </form>

      <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
        <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.users.siteMessage.history') }}</h4>
        <p v-if="historyLoading && history.length === 0" class="text-sm text-gray-400">{{ t('common.loading') }}</p>
        <p v-else-if="history.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.users.siteMessage.historyEmpty') }}</p>
        <ul v-else class="max-h-72 space-y-2 overflow-y-auto">
          <li
            v-for="item in history"
            :key="item.id"
            data-testid="site-message-history-row"
            class="rounded-lg border border-gray-100 px-3 py-2 dark:border-dark-700"
          >
            <div class="flex flex-wrap items-center gap-2 text-sm">
              <span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ siteMessageCategoryLabel(t, item.category) }}
              </span>
              <span class="min-w-0 flex-1 truncate font-medium text-gray-900 dark:text-gray-100">{{ item.title }}</span>
              <span
                class="text-xs"
                :class="item.read_at ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'"
                :title="item.read_at ? t('admin.users.siteMessage.readAt', { time: formatDateTime(item.read_at) }) : ''"
              >
                {{ item.read_at ? t('admin.users.siteMessage.read') : t('admin.users.siteMessage.unread') }}
              </span>
            </div>
            <div class="mt-1 flex flex-wrap items-center gap-x-2 text-xs text-gray-400">
              <span>{{ senderLabel(item) }}</span>
              <span v-if="item.approval_id">· {{ t('admin.users.siteMessage.viaApproval', { id: item.approval_id }) }}</span>
              <span>· {{ formatDateTime(item.created_at) }}</span>
              <button type="button" class="ml-auto text-primary-600 hover:underline dark:text-primary-400" @click="toggleExpanded(item.id)">
                {{ expanded.has(item.id) ? t('admin.users.siteMessage.hideContent') : t('admin.users.siteMessage.showContent') }}
              </button>
            </div>
            <div v-if="expanded.has(item.id)" class="mt-2 rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800">
              <MarkdownRenderer :content="item.content" />
            </div>
          </li>
        </ul>
        <div v-if="history.length < historyTotal" class="mt-2 text-center">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="historyLoading" @click="loadHistory(false)">
            {{ t('siteMessages.loadMore') }}
          </button>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" form="site-message-form" class="btn btn-primary" :disabled="submitting">
          {{ submitting ? t('common.saving') : t('admin.users.siteMessage.send') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import MarkdownRenderer from '@/components/common/MarkdownRenderer.vue'
import { adminAPI } from '@/api/admin'
import type { AdminSiteMessage } from '@/api/admin/users'
import { useAppStore } from '@/stores/app'
import { isApprovalQueued } from '@/utils/approval'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { siteMessageCategoryLabel } from '@/utils/siteMessages'
import type { AdminUser } from '@/types'

// 用户管理 → 发送站内信（fork 本地）。运维管理员（readonly）同样能发：后端返回 202 进入审批，
// apiClient 已弹出全局提示，这里只需关闭弹窗。
const TITLE_MAX = 200
const CONTENT_MAX = 5000
const HISTORY_PAGE_SIZE = 10

const props = withDefaults(defineProps<{ show: boolean; user: AdminUser | null; readonly?: boolean }>(), {
  readonly: false
})
const emit = defineEmits<{ close: []; sent: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const form = reactive({ title: '', content: '' })
const previewing = ref(false)
const submitting = ref(false)
const history = ref<AdminSiteMessage[]>([])
const historyTotal = ref(0)
const historyPage = ref(0)
const historyLoading = ref(false)
const expanded = ref(new Set<number>())
let idempotencyKey = ''

function newIdempotencyKey(userId: number) {
  const requestId = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `site-message-${userId}-${requestId}`
}

/** 与后端一致按码点计数（中文与 emoji 各算 1 个） */
function runeLength(value: string): number {
  return [...value.trim()].length
}

function senderLabel(item: AdminSiteMessage): string {
  if (!item.sender_user_id) return t('admin.users.siteMessage.senderSystem')
  const who = item.sender_email || `#${item.sender_user_id}`
  return item.sender_role ? `${who} (${item.sender_role})` : who
}

function toggleExpanded(id: number) {
  const next = new Set(expanded.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expanded.value = next
}

async function loadHistory(reset: boolean) {
  if (!props.user || historyLoading.value) return
  const userId = props.user.id
  const page = reset ? 1 : historyPage.value + 1
  historyLoading.value = true
  try {
    const res = await adminAPI.users.listSiteMessages(userId, page, HISTORY_PAGE_SIZE)
    if (props.user?.id !== userId) return
    const items = res?.items ?? []
    history.value = reset ? items : [...history.value, ...items]
    historyTotal.value = Number(res?.total ?? history.value.length)
    historyPage.value = page
  } catch (e) {
    appStore.showError(extractApiErrorMessage(e, t('admin.users.siteMessage.loadFailed')))
  } finally {
    historyLoading.value = false
  }
}

watch(
  () => [props.show, props.user?.id] as const,
  ([visible]) => {
    if (!visible || !props.user) return
    form.title = ''
    form.content = ''
    previewing.value = false
    expanded.value = new Set()
    history.value = []
    historyTotal.value = 0
    historyPage.value = 0
    idempotencyKey = newIdempotencyKey(props.user.id)
    void loadHistory(true)
  },
  { immediate: true }
)

function validate(): string | null {
  const titleLen = runeLength(form.title)
  const contentLen = runeLength(form.content)
  if (titleLen === 0) return t('admin.users.siteMessage.titleRequired')
  if (titleLen > TITLE_MAX) return t('admin.users.siteMessage.titleTooLong', { max: TITLE_MAX })
  if (contentLen === 0) return t('admin.users.siteMessage.contentRequired')
  if (contentLen > CONTENT_MAX) return t('admin.users.siteMessage.contentTooLong', { max: CONTENT_MAX })
  return null
}

async function handleSubmit() {
  if (!props.user || submitting.value) return
  const invalid = validate()
  if (invalid) {
    appStore.showError(invalid)
    return
  }
  submitting.value = true
  try {
    await adminAPI.users.sendSiteMessage(
      props.user.id,
      { title: form.title.trim(), content: form.content.trim() },
      idempotencyKey
    )
    appStore.showSuccess(t('admin.users.siteMessage.sent'))
    emit('sent')
    form.title = ''
    form.content = ''
    previewing.value = false
    idempotencyKey = newIdempotencyKey(props.user.id)
    void loadHistory(true)
  } catch (e) {
    if (isApprovalQueued(e)) {
      // 运维管理员：已排队等待管理员审批（全局提示已弹出）
      emit('close')
      return
    }
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally {
    submitting.value = false
  }
}
</script>
