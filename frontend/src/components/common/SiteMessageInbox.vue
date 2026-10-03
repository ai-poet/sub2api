<template>
  <div class="relative">
    <!-- 信封按钮 + 未读数字角标 -->
    <button
      type="button"
      data-testid="site-message-inbox-button"
      class="relative flex h-9 w-9 items-center justify-center rounded-lg text-gray-600 transition-all hover:scale-105 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-800"
      :class="{ 'text-primary-600 dark:text-primary-400': store.unreadCount > 0 }"
      :aria-label="t('siteMessages.open')"
      :title="t('siteMessages.title')"
      @click="openInbox"
    >
      <Icon name="mail" size="md" />
      <span
        v-if="store.unreadCount > 0"
        data-testid="site-message-badge"
        class="absolute -right-0.5 -top-0.5 flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-semibold leading-none text-white"
      >
        {{ badgeText }}
      </span>
    </button>

    <!-- 列表 -->
    <BaseDialog :show="store.inboxOpen" :title="t('siteMessages.title')" width="wide" @close="closeInbox">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
        <div class="inline-flex rounded-lg bg-gray-100 p-0.5 dark:bg-dark-800">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="rounded-md px-3 py-1 text-sm transition-colors"
            :class="store.unreadOnly === tab.unreadOnly
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
            @click="switchTab(tab.unreadOnly)"
          >
            {{ tab.label }}
          </button>
        </div>
        <button
          type="button"
          data-testid="site-message-mark-all"
          class="btn btn-secondary btn-sm"
          :disabled="store.unreadCount === 0 || markingAll"
          @click="handleMarkAll"
        >
          {{ t('siteMessages.markAllRead') }}
        </button>
      </div>

      <div v-if="store.items.length === 0 && !store.listLoading" class="py-12 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ store.unreadOnly ? t('siteMessages.emptyUnread') : t('siteMessages.empty') }}
      </div>

      <ul v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <li v-for="message in store.items" :key="message.id">
          <button
            type="button"
            data-testid="site-message-row"
            class="flex w-full items-start gap-3 rounded-lg px-2 py-3 text-left transition-colors hover:bg-gray-50 dark:hover:bg-dark-800"
            @click="openDetail(message)"
          >
            <span
              class="mt-1.5 h-2 w-2 flex-shrink-0 rounded-full"
              :class="message.read_at ? 'bg-transparent' : 'bg-red-500'"
            ></span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span :class="['rounded px-1.5 py-0.5 text-xs font-medium', categoryClass(message.category)]">
                  {{ siteMessageCategoryLabel(t, message.category) }}
                </span>
                <span
                  class="truncate text-sm"
                  :class="message.read_at ? 'text-gray-600 dark:text-gray-300' : 'font-semibold text-gray-900 dark:text-white'"
                >
                  {{ message.title }}
                </span>
              </span>
              <span class="mt-1 block text-xs text-gray-400 dark:text-gray-500">
                {{ siteMessageFromLabel(t, message.from) }} · {{ formatRelativeWithDateTime(message.created_at) }}
              </span>
            </span>
          </button>
        </li>
      </ul>

      <div v-if="hasMore" class="mt-4 text-center">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="store.listLoading" @click="loadMore">
          {{ t('siteMessages.loadMore') }}
        </button>
      </div>
    </BaseDialog>

    <!-- 详情 -->
    <BaseDialog
      :show="detail !== null"
      :title="detail?.title ?? ''"
      width="wide"
      :z-index="60"
      @close="detail = null"
    >
      <div v-if="detail">
        <div class="mb-4 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span :class="['rounded px-1.5 py-0.5 font-medium', categoryClass(detail.category)]">
            {{ siteMessageCategoryLabel(t, detail.category) }}
          </span>
          <span>{{ siteMessageFromLabel(t, detail.from) }}</span>
          <span>·</span>
          <time>{{ formatRelativeWithDateTime(detail.created_at) }}</time>
        </div>
        <MarkdownRenderer :content="detail.content" class-name="text-sm text-gray-800 dark:text-gray-200" />
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button type="button" class="btn btn-primary" @click="detail = null">{{ t('siteMessages.back') }}</button>
        </div>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import MarkdownRenderer from '@/components/common/MarkdownRenderer.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useSiteMessagesStore } from '@/stores/siteMessages'
import type { SiteMessage, SiteMessageCategory } from '@/api/siteMessages'
import { formatRelativeWithDateTime } from '@/utils/format'
import { siteMessageCategoryLabel, siteMessageFromLabel } from '@/utils/siteMessages'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const store = useSiteMessagesStore()
const appStore = useAppStore()

const detail = ref<SiteMessage | null>(null)
const markingAll = ref(false)

const badgeText = computed(() => (store.unreadCount > 99 ? '99+' : String(store.unreadCount)))
const hasMore = computed(() => store.items.length < store.total)
const tabs = computed(() => [
  { key: 'all', label: t('siteMessages.tabs.all'), unreadOnly: false },
  { key: 'unread', label: t('siteMessages.tabs.unread'), unreadOnly: true }
])

function categoryClass(category: SiteMessageCategory): string {
  switch (category) {
    case 'security':
      return 'bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-300'
    case 'admin':
      return 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  }
}

async function reload() {
  try {
    await store.loadPage(true)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('siteMessages.loadFailed')))
  }
}

function openInbox() {
  store.inboxOpen = true
}

function closeInbox() {
  store.inboxOpen = false
}

// 顶栏按钮或弹窗「查看全部」打开时刷新列表
watch(
  () => store.inboxOpen,
  (open) => {
    if (open) void reload()
  }
)

async function switchTab(unreadOnly: boolean) {
  try {
    await store.setUnreadOnly(unreadOnly)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('siteMessages.loadFailed')))
  }
}

async function loadMore() {
  try {
    await store.loadPage(false)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('siteMessages.loadFailed')))
  }
}

function openDetail(message: SiteMessage) {
  detail.value = message
  if (!message.read_at) void store.markRead(message.id)
}

async function handleMarkAll() {
  markingAll.value = true
  try {
    await store.markAllRead()
    appStore.showSuccess(t('siteMessages.allMarkedRead'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    markingAll.value = false
  }
}
</script>
