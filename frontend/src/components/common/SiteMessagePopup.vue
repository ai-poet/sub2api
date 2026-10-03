<template>
  <Teleport to="body">
    <Transition name="popup-fade">
      <div
        v-if="message"
        data-testid="site-message-popup"
        class="fixed inset-0 z-[110] flex items-start justify-center overflow-y-auto bg-black/60 p-4 pt-[8vh] backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
      >
        <div
          class="w-full max-w-[640px] overflow-hidden rounded-2xl bg-white shadow-2xl ring-1 ring-black/5 dark:bg-dark-800 dark:ring-white/10"
          @click.stop
        >
          <div :class="['border-b px-6 py-5', headerClass]">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center gap-1.5 rounded-lg bg-white/70 px-2 py-0.5 text-xs font-medium text-gray-700 dark:bg-dark-900/50 dark:text-gray-200">
                <Icon name="mail" size="xs" />
                {{ t('siteMessages.popup.badge') }}
              </span>
              <span class="rounded-lg bg-white/70 px-2 py-0.5 text-xs font-medium text-gray-700 dark:bg-dark-900/50 dark:text-gray-200">
                {{ siteMessageCategoryLabel(t, message.category) }}
              </span>
              <span v-if="remaining > 0" class="ml-auto text-xs text-gray-500 dark:text-gray-400">
                {{ t('siteMessages.popup.more', { count: remaining }) }}
              </span>
            </div>
            <h2 class="text-xl font-bold leading-tight text-gray-900 dark:text-white">{{ message.title }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ siteMessageFromLabel(t, message.from) }} · {{ formatRelativeWithDateTime(message.created_at) }}
            </p>
          </div>

          <div class="max-h-[55vh] overflow-y-auto px-6 py-5">
            <MarkdownRenderer :content="message.content" class-name="text-sm text-gray-800 dark:text-gray-200" />
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2 border-t border-gray-100 bg-gray-50/60 px-6 py-4 dark:border-dark-700 dark:bg-dark-900/30">
            <button type="button" class="btn btn-secondary btn-sm" @click="store.openInboxFromPopup()">
              {{ t('siteMessages.popup.viewAll') }}
            </button>
            <button
              v-if="remaining > 0"
              type="button"
              data-testid="site-message-popup-read-all"
              class="btn btn-secondary btn-sm"
              @click="handleMarkAll"
            >
              {{ t('siteMessages.markAllRead') }}
            </button>
            <button
              type="button"
              data-testid="site-message-popup-ack"
              class="btn btn-primary btn-sm"
              @click="store.acknowledgePopup()"
            >
              {{ t('siteMessages.popup.acknowledge') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import MarkdownRenderer from '@/components/common/MarkdownRenderer.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAppStore } from '@/stores/app'
import { useSiteMessagesStore } from '@/stores/siteMessages'
import { formatRelativeWithDateTime } from '@/utils/format'
import { siteMessageCategoryLabel, siteMessageFromLabel } from '@/utils/siteMessages'
import { extractApiErrorMessage } from '@/utils/apiError'

// 站内信默认自动弹窗（fork 本地）。公告弹窗优先：公告关掉之后才显示站内信；申诉页不弹。
const { t } = useI18n()
const route = useRoute()
const store = useSiteMessagesStore()
const announcementStore = useAnnouncementStore()
const appStore = useAppStore()

const suppressed = computed(() => route.path.startsWith('/appeal') || !!announcementStore.currentPopup)
const message = computed(() => (suppressed.value ? null : store.currentPopup))
const remaining = computed(() => store.popupQueue.length)

const headerClass = computed(() => {
  switch (message.value?.category) {
    case 'security':
      return 'border-red-100 bg-gradient-to-br from-red-50 to-orange-50 dark:border-dark-700 dark:from-red-900/20 dark:to-orange-900/10'
    case 'admin':
      return 'border-blue-100 bg-gradient-to-br from-blue-50 to-indigo-50 dark:border-dark-700 dark:from-blue-900/20 dark:to-indigo-900/10'
    default:
      return 'border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/30'
  }
})

async function handleMarkAll() {
  try {
    await store.markAllRead()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}
</script>
