<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <header class="border-b border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
      <div class="mx-auto flex h-14 max-w-3xl items-center justify-between gap-3 px-4">
        <span class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ appStore.siteName }}</span>
        <div class="flex items-center gap-2">
          <LocaleSwitcher />
          <button type="button" class="btn btn-secondary btn-sm" data-testid="appeal-exit" @click="exitAppeal">
            {{ t('appeal.exit') }}
          </button>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-3xl space-y-6 px-4 py-6">
      <section class="rounded-2xl border border-red-200 bg-red-50 p-5 dark:border-red-900/50 dark:bg-red-950/40">
        <h1 class="text-lg font-semibold text-red-700 dark:text-red-300">{{ t('appeal.banner.title') }}</h1>
        <p class="mt-2 text-sm leading-6 text-red-700/90 dark:text-red-200/90">{{ t('appeal.banner.desc') }}</p>
        <p v-if="session" class="mt-3 text-xs text-red-600/80 dark:text-red-300/80">
          {{ session.email }} · {{ t('appeal.expiresAt', { time: formatDateTime(session.expires_at) }) }}
        </p>
      </section>

      <!-- 站内信：封禁原因 -->
      <section class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('appeal.notices') }}</h2>
        <p v-if="messagesLoaded && notices.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('appeal.noNotices') }}</p>
        <ul class="space-y-2">
          <li
            v-for="notice in notices"
            :key="notice.id"
            data-testid="appeal-notice"
            class="rounded-xl border border-gray-100 dark:border-dark-700"
          >
            <button
              type="button"
              class="flex w-full items-center gap-2 px-4 py-3 text-left"
              @click="toggleNotice(notice)"
            >
              <span class="h-2 w-2 flex-shrink-0 rounded-full" :class="notice.read_at ? 'bg-transparent' : 'bg-red-500'"></span>
              <span class="min-w-0 flex-1 truncate text-sm font-medium text-gray-900 dark:text-gray-100">{{ notice.title }}</span>
              <span class="flex-shrink-0 text-xs text-gray-400">{{ formatDateTime(notice.created_at) }}</span>
            </button>
            <div v-if="expandedNotice === notice.id" class="border-t border-gray-100 px-4 py-3 text-sm dark:border-dark-700">
              <MarkdownRenderer :content="notice.content" class-name="text-gray-800 dark:text-gray-200" />
            </div>
          </li>
        </ul>
      </section>

      <!-- 申诉工单 -->
      <section class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
        <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('appeal.ticketTitle') }}</h2>

        <div v-if="ticket && !showCreate">
          <div class="mb-3 flex flex-wrap items-center gap-2 text-sm">
            <span :class="ticketStatusBadgeClass(ticket.status)">{{ ticketStatusLabel(t, ticket.status) }}</span>
            <span class="font-medium text-gray-800 dark:text-gray-100">{{ ticket.title }}</span>
          </div>
          <TicketThread
            ref="threadRef"
            :ticket="ticket"
            :messages="ticketMessages"
            viewer="user"
            side="appeal"
            :submitting="replying"
            :loading="ticketLoading"
            @reply="sendReply"
          />
          <div v-if="ticket.status === 'closed'" class="mt-4 flex justify-end">
            <button type="button" class="btn btn-primary btn-sm" data-testid="appeal-submit-again" @click="startNewAppeal">
              {{ t('appeal.submitAgain') }}
            </button>
          </div>
        </div>

        <form v-else-if="ticketsLoaded" class="space-y-4" data-testid="appeal-form" @submit.prevent="submitAppeal">
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('appeal.createTitle') }}</p>
          <div>
            <label class="input-label" for="appeal-title">{{ t('appeal.form.title') }}</label>
            <input id="appeal-title" v-model="form.title" type="text" class="input" :maxlength="TICKET_TITLE_MAX" />
          </div>
          <div>
            <label class="input-label" for="appeal-body">{{ t('appeal.form.body') }}</label>
            <textarea
              id="appeal-body"
              v-model="form.body"
              rows="6"
              class="input"
              :maxlength="TICKET_BODY_MAX"
              :placeholder="t('appeal.form.bodyPlaceholder')"
            ></textarea>
          </div>
          <div class="flex justify-end">
            <button type="submit" class="btn btn-primary" :disabled="submitting || !form.title.trim() || !form.body.trim()">
              {{ submitting ? t('common.saving') : t('appeal.form.submit') }}
            </button>
          </div>
        </form>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import MarkdownRenderer from '@/components/common/MarkdownRenderer.vue'
import TicketThread from '@/components/tickets/TicketThread.vue'
import appealAPI, { type AppealApiError, type AppealSessionInfo } from '@/api/appeal'
import type { SiteMessage } from '@/api/siteMessages'
import type { SupportTicket, SupportTicketMessage } from '@/api/tickets'
import { useAppStore } from '@/stores/app'
import { clearAppealSession, readAppealSession, setAppealLoginNotice } from '@/utils/appeal'
import { formatDateTime } from '@/utils/format'
import { TICKET_BODY_MAX, TICKET_TITLE_MAX, ticketStatusBadgeClass, ticketStatusLabel } from '@/utils/tickets'

// 封禁申诉页（fork 本地）：只用申诉令牌（sessionStorage），不碰正常登录态。
const SESSION_POLL_MS = 60 * 1000

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const session = ref<AppealSessionInfo | null>(null)
const notices = ref<SiteMessage[]>([])
const messagesLoaded = ref(false)
const expandedNotice = ref<number | null>(null)

const ticket = ref<SupportTicket | null>(null)
const ticketMessages = ref<SupportTicketMessage[]>([])
const ticketsLoaded = ref(false)
const ticketLoading = ref(false)
const showCreate = ref(false)
const replying = ref(false)
const submitting = ref(false)
const threadRef = ref<InstanceType<typeof TicketThread> | null>(null)
const form = reactive({ title: '', body: '' })

let pollTimer: number | null = null
let leaving = false

function leave(notice: 'restored' | 'expired' | null) {
  if (leaving) return
  leaving = true
  clearAppealSession()
  if (notice) setAppealLoginNotice(notice)
  void router.replace('/login')
}

/** 令牌失效 / 账户已恢复时离开申诉页；返回 true 表示已处理。 */
function handleSessionError(err: unknown): boolean {
  const e = err as AppealApiError
  if (e?.reason === 'APPEAL_ACCOUNT_ACTIVE' || e?.status === 409) {
    leave('restored')
    return true
  }
  if (e?.reason === 'APPEAL_TOKEN_INVALID' || e?.status === 401) {
    leave('expired')
    return true
  }
  return false
}

function showError(err: unknown, fallback: string) {
  if (handleSessionError(err)) return
  const e = err as AppealApiError
  appStore.showError(e?.message || fallback)
}

async function loadSession() {
  try {
    session.value = await appealAPI.getSession()
  } catch (err) {
    showError(err, t('appeal.errors.expired'))
  }
}

async function loadNotices() {
  try {
    const res = await appealAPI.listSiteMessages(1, 20)
    notices.value = res?.items ?? []
  } catch (err) {
    showError(err, t('siteMessages.loadFailed'))
  } finally {
    messagesLoaded.value = true
  }
}

async function loadTicket() {
  ticketLoading.value = true
  try {
    const res = await appealAPI.listTickets(1, 20)
    const items = res?.items ?? []
    // 优先显示未关闭的申诉，其次最近一单
    const current = items.find((item) => item.status !== 'closed') ?? items[0] ?? null
    if (current) {
      const detail = await appealAPI.getTicket(current.id)
      ticket.value = detail.ticket
      ticketMessages.value = detail.messages ?? []
    } else {
      ticket.value = null
      ticketMessages.value = []
    }
  } catch (err) {
    showError(err, t('tickets.loadFailed'))
  } finally {
    ticketLoading.value = false
    ticketsLoaded.value = true
  }
}

function toggleNotice(notice: SiteMessage) {
  expandedNotice.value = expandedNotice.value === notice.id ? null : notice.id
  if (!notice.read_at) {
    notice.read_at = new Date().toISOString()
    appealAPI.markSiteMessageRead(notice.id).catch((err) => handleSessionError(err))
  }
}

function startNewAppeal() {
  showCreate.value = true
  form.title = t('appeal.form.defaultTitle')
  form.body = ''
}

async function submitAppeal() {
  if (submitting.value) return
  submitting.value = true
  try {
    await appealAPI.createTicket(form.title.trim(), form.body.trim())
    showCreate.value = false
    appStore.showSuccess(t('appeal.form.submitted'))
    await loadTicket()
  } catch (err) {
    const e = err as AppealApiError
    if (e?.reason === 'TICKET_APPEAL_ACTIVE') {
      appStore.showError(t('appeal.errors.activeExists'))
      showCreate.value = false
      await loadTicket()
    } else if (e?.status === 429) {
      appStore.showError(t('appeal.errors.rateLimited'))
    } else {
      showError(err, t('common.error'))
    }
  } finally {
    submitting.value = false
  }
}

async function sendReply(body: string) {
  const current = ticket.value
  if (!current || replying.value) return
  replying.value = true
  try {
    const res = await appealAPI.reply(current.id, body)
    ticketMessages.value.push(res.message)
    ticket.value = res.ticket
    threadRef.value?.clearDraft()
  } catch (err) {
    const e = err as AppealApiError
    if (e?.status === 429) appStore.showError(t('appeal.errors.rateLimited'))
    else showError(err, t('common.error'))
  } finally {
    replying.value = false
  }
}

async function exitAppeal() {
  try {
    await appealAPI.logout()
  } catch {
    // 退出失败也照样清掉本地令牌
  }
  leave(null)
}

function pollSession() {
  if (typeof document !== 'undefined' && document.hidden) return
  void loadSession()
}

onMounted(async () => {
  if (!readAppealSession()) {
    leave('expired')
    return
  }
  form.title = t('appeal.form.defaultTitle')
  await loadSession()
  if (leaving) return
  await Promise.all([loadNotices(), loadTicket()])
  pollTimer = window.setInterval(pollSession, SESSION_POLL_MS)
})

onBeforeUnmount(() => {
  if (pollTimer !== null) window.clearInterval(pollTimer)
})
</script>
