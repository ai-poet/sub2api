<template>
  <div class="space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700" data-test="personal-tokens-panel">
    <div class="flex items-center justify-between">
      <h4 class="text-sm font-medium text-gray-900 dark:text-white">
        {{ t('admin.settings.site.personalToken.listTitle') }}
      </h4>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
        {{ t('admin.settings.site.personalToken.refresh') }}
      </button>
    </div>

    <p v-if="loadFailed" class="text-sm text-red-600 dark:text-red-400">
      {{ t('admin.settings.site.personalToken.loadFailed') }}
    </p>
    <div v-else-if="loading && items.length === 0" class="flex justify-center py-4">
      <div class="h-6 w-6 animate-spin rounded-full border-b-2 border-primary-500"></div>
    </div>
    <p
      v-else-if="items.length === 0"
      class="rounded-lg border border-dashed border-gray-200 px-4 py-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
    >
      {{ t('admin.settings.site.personalToken.empty') }}
    </p>

    <ul v-else class="divide-y divide-gray-100 dark:divide-dark-700">
      <li
        v-for="item in items"
        :key="item.user_id"
        class="flex flex-col gap-2 py-3 first:pt-0 last:pb-0 sm:flex-row sm:items-center sm:justify-between"
        data-test="personal-token-row"
      >
        <div class="min-w-0 space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <span class="truncate text-sm font-medium text-gray-900 dark:text-white">
              {{ item.user_email || `#${item.user_id}` }}
            </span>
            <span
              class="rounded-full px-2 py-0.5 text-xs"
              :class="
                item.state === 'active'
                  ? 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                  : 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
              "
            >
              {{ t(`admin.settings.site.personalToken.states.${item.state}`) }}
            </span>
            <code class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ item.hint }}</code>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.site.personalToken.createdAt', { date: formatDateTime(item.created_at) }) }}
            ·
            {{
              item.expires_at
                ? t('admin.settings.site.personalToken.expiresAt', { date: formatDateTime(item.expires_at) })
                : t('admin.settings.site.personalToken.neverExpires')
            }}
            ·
            {{
              item.last_used_at
                ? t('admin.settings.site.personalToken.lastUsed', {
                    date: formatDateTime(item.last_used_at),
                    ip: item.last_used_ip || '-'
                  })
                : t('admin.settings.site.personalToken.neverUsed')
            }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-ghost btn-sm shrink-0 text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-950/30"
          :disabled="revoking"
          @click="revokeTarget = item"
        >
          {{ t('admin.settings.site.personalToken.revoke') }}
        </button>
      </li>
    </ul>

    <ConfirmDialog
      :show="revokeTarget !== null"
      :title="t('admin.settings.site.personalToken.revokeTitle')"
      :message="t('admin.settings.site.personalToken.revokeConfirm', { email: revokeTarget?.user_email ?? '' })"
      danger
      @confirm="confirmRevoke"
      @cancel="revokeTarget = null"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { adminAPI } from '@/api/admin'
import type { AdminPersonalTokenItem } from '@/api/admin/personalTokens'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

// 运维管理员个人令牌的管理员视图（fork 本地）：只展示提示串，吊销立即生效。
const { t } = useI18n()
const appStore = useAppStore()

const items = ref<AdminPersonalTokenItem[]>([])
const loading = ref(false)
const loadFailed = ref(false)
const revoking = ref(false)
const revokeTarget = ref<AdminPersonalTokenItem | null>(null)

async function load() {
  if (loading.value) return
  loading.value = true
  loadFailed.value = false
  try {
    const result = await adminAPI.personalTokens.list()
    items.value = result.items ?? []
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

async function confirmRevoke() {
  const target = revokeTarget.value
  if (!target || revoking.value) return
  revoking.value = true
  try {
    await adminAPI.personalTokens.revoke(target.user_id)
    items.value = items.value.filter((item) => item.user_id !== target.user_id)
    appStore.showSuccess(t('admin.settings.site.personalToken.revoked'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.personalToken.revokeFailed')))
  } finally {
    revoking.value = false
    revokeTarget.value = null
  }
}

onMounted(load)
</script>
