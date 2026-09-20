<template>
  <div class="flex flex-col gap-4">
    <div class="max-h-[50vh] space-y-3 overflow-y-auto pr-1" data-test="ticket-thread">
      <div v-if="loading" class="py-6 text-center text-sm text-gray-400 dark:text-gray-500">
        {{ t('tickets.loading') }}
      </div>
      <template v-else>
        <div
          v-for="msg in messages"
          :key="msg.id"
          class="flex"
          :class="isMine(msg) ? 'justify-end' : 'justify-start'"
          :data-test="`ticket-message-${msg.id}`"
        >
          <div
            class="max-w-[85%] rounded-2xl px-4 py-3 text-sm shadow-sm"
            :class="isMine(msg)
              ? 'bg-primary-50 text-gray-900 dark:bg-primary-900/20 dark:text-gray-100'
              : 'bg-gray-100 text-gray-900 dark:bg-dark-800 dark:text-gray-100'"
          >
            <div class="mb-1 flex flex-wrap items-center gap-x-2 text-xs text-gray-500 dark:text-gray-400">
              <span class="font-medium text-gray-700 dark:text-gray-200">{{ authorLabel(msg) }}</span>
              <span>{{ formatDateTime(msg.created_at) }}</span>
            </div>
            <MarkdownRenderer :content="msg.body" />
          </div>
        </div>
      </template>
    </div>

    <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
      <p v-if="closed" class="text-sm text-gray-500 dark:text-gray-400" data-test="ticket-closed-hint">
        {{ t('tickets.thread.closedHint') }}
      </p>
      <template v-else>
        <TextArea v-model="draft" :rows="4" :placeholder="t('tickets.thread.placeholder')" :disabled="submitting" />
        <div class="mt-2 flex items-center justify-between gap-3">
          <span class="text-xs" :class="overLimit ? 'text-red-500' : 'text-gray-400 dark:text-gray-500'">
            {{ draft.length }} / {{ TICKET_BODY_MAX }}
          </span>
          <button type="button" class="btn btn-primary btn-sm" :disabled="!canSend" data-test="ticket-send" @click="send">
            {{ submitting ? t('tickets.thread.sending') : t('tickets.thread.send') }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import MarkdownRenderer from '@/components/common/MarkdownRenderer.vue'
import TextArea from '@/components/common/TextArea.vue'
import type { SupportTicket, SupportTicketMessage } from '@/api/tickets'
import type { AdminSupportTicket } from '@/api/admin/tickets'
import { formatDateTime } from '@/utils/format'
import { TICKET_BODY_MAX } from '@/utils/tickets'

// 工单线程（fork 本地功能）：用户页与客服页共用。viewer 决定"我方"是哪一边以及作者怎么称呼。
const props = withDefaults(
  defineProps<{
    ticket: SupportTicket | AdminSupportTicket | null
    messages: SupportTicketMessage[]
    viewer: 'user' | 'staff'
    submitting?: boolean
    loading?: boolean
  }>(),
  { submitting: false, loading: false }
)

const emit = defineEmits<{
  (e: 'reply', body: string): void
}>()

const { t } = useI18n()

const draft = ref('')
const closed = computed(() => props.ticket?.status === 'closed')
const overLimit = computed(() => draft.value.length > TICKET_BODY_MAX)
const canSend = computed(() => !props.submitting && draft.value.trim().length > 0 && !overLimit.value)

function isMine(msg: SupportTicketMessage): boolean {
  const fromUser = msg.author_role === 'user'
  return props.viewer === 'user' ? fromUser : !fromUser
}

function roleLabel(role: string): string {
  switch (role) {
    case 'admin':
      return t('tickets.thread.roleAdmin')
    case 'operator':
      return t('tickets.thread.roleOperator')
    default:
      return t('tickets.thread.staff')
  }
}

function authorLabel(msg: SupportTicketMessage): string {
  if (msg.author_role === 'user') {
    if (props.viewer === 'user') return t('tickets.thread.you')
    const email = (props.ticket as AdminSupportTicket | null)?.user?.email
    return email || t('tickets.thread.user')
  }
  if (props.viewer === 'user') return t('tickets.thread.staff')
  const role = roleLabel(msg.author_role)
  return msg.author_email ? `${msg.author_email} · ${role}` : role
}

function send() {
  if (!canSend.value) return
  emit('reply', draft.value.trim())
}

function clearDraft() {
  draft.value = ''
}

defineExpose({ clearDraft })
</script>
