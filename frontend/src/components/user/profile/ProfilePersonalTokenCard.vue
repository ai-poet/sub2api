<template>
  <div class="card" data-test="personal-token-card">
    <div class="flex items-start justify-between gap-4 border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div>
        <h2 class="text-lg font-medium text-gray-900 dark:text-white">
          {{ t('operator.personalToken.title') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('operator.personalToken.description') }}
        </p>
      </div>
      <button
        v-if="canGenerate && !status?.token"
        type="button"
        class="btn btn-primary shrink-0"
        :disabled="busy"
        data-test="personal-token-generate"
        @click="openGenerate"
      >
        {{ t('operator.personalToken.generate') }}
      </button>
    </div>

    <div class="space-y-4 px-6 py-6">
      <div v-if="loading" class="flex justify-center py-4">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-500"></div>
      </div>

      <template v-else-if="status">
        <p
          v-if="!status.feature_enabled"
          class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300"
          data-test="personal-token-disabled"
        >
          {{ t('operator.personalToken.featureDisabled') }}
        </p>

        <!-- 明文只在这里出现一次：刷新 / 离开页面后只剩提示串 -->
        <div
          v-if="newToken"
          class="space-y-3 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20"
          data-test="personal-token-plaintext"
        >
          <p class="text-sm font-medium text-green-700 dark:text-green-300">
            {{ t('operator.personalToken.showOnceWarning') }}
          </p>
          <div class="flex items-center gap-2">
            <code
              class="flex-1 select-all break-all rounded border border-green-300 bg-white px-3 py-2 font-mono text-sm dark:border-green-700 dark:bg-dark-800"
            >{{ newToken }}</code>
            <button type="button" class="btn btn-primary btn-sm shrink-0" @click="copyToClipboard(newToken, t('operator.personalToken.copied'))">
              {{ t('operator.personalToken.copy') }}
            </button>
          </div>
          <div class="space-y-1">
            <p class="text-xs text-green-700 dark:text-green-400">{{ t('operator.personalToken.usageHint') }}</p>
            <pre
              class="overflow-x-auto whitespace-pre-wrap break-all rounded bg-white px-3 py-2 font-mono text-xs text-gray-700 dark:bg-dark-800 dark:text-gray-300"
            >{{ curlExample }}</pre>
          </div>
          <div class="flex justify-end">
            <button type="button" class="btn btn-secondary btn-sm" @click="newToken = ''">
              {{ t('operator.personalToken.done') }}
            </button>
          </div>
        </div>

        <div
          v-if="!status.token"
          class="rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
        >
          {{ t('operator.personalToken.empty') }}
        </div>

        <div v-else class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between" data-test="personal-token-current">
          <div class="min-w-0 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <Icon name="key" size="md" class="shrink-0 text-primary-500" />
              <code class="font-mono text-sm text-gray-900 dark:text-white">{{ status.token.hint }}</code>
              <span
                v-if="status.token.expired"
                class="rounded-full bg-red-50 px-2 py-0.5 text-xs text-red-700 dark:bg-red-900/30 dark:text-red-300"
              >
                {{ t('operator.personalToken.expired') }}
              </span>
            </div>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('operator.personalToken.createdAt', { date: formatDateTime(status.token.created_at) }) }}
              ·
              {{
                status.token.expires_at
                  ? t('operator.personalToken.expiresAt', { date: formatDateTime(status.token.expires_at) })
                  : t('operator.personalToken.neverExpires')
              }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{
                status.token.last_used_at
                  ? t('operator.personalToken.lastUsed', {
                      date: formatDateTime(status.token.last_used_at),
                      ip: status.token.last_used_ip || '-'
                    })
                  : t('operator.personalToken.neverUsed')
              }}
            </p>
          </div>
          <div class="flex shrink-0 gap-2">
            <button
              v-if="canGenerate"
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="busy"
              data-test="personal-token-regenerate"
              @click="openGenerate"
            >
              {{ t('operator.personalToken.regenerate') }}
            </button>
            <button
              type="button"
              class="btn btn-ghost btn-sm text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-950/30"
              :disabled="busy"
              data-test="personal-token-revoke"
              @click="showRevoke = true"
            >
              {{ t('operator.personalToken.revoke') }}
            </button>
          </div>
        </div>

        <ul class="list-disc space-y-1 pl-5 text-xs text-gray-500 dark:text-gray-400">
          <li>{{ t('operator.personalToken.notes.scope') }}</li>
          <li>{{ t('operator.personalToken.notes.approval') }}</li>
          <li>{{ t('operator.personalToken.notes.invalidation') }}</li>
          <li>{{ t('operator.personalToken.notes.noSensitive') }}</li>
        </ul>
      </template>
    </div>

    <BaseDialog
      :show="showGenerate"
      :title="t(status?.token ? 'operator.personalToken.regenerateTitle' : 'operator.personalToken.generateTitle')"
      width="narrow"
      @close="closeGenerate"
    >
      <form id="personal-token-generate-form" class="space-y-4" @submit.prevent="submitGenerate">
        <p v-if="status?.token" class="text-sm text-amber-600 dark:text-amber-400">
          {{ t('operator.personalToken.regenerateWarning') }}
        </p>
        <div>
          <label for="personal-token-expiry" class="input-label">{{ t('operator.personalToken.expiry') }}</label>
          <select id="personal-token-expiry" v-model.number="expiresInDays" class="input">
            <option v-for="days in expiryOptions" :key="days" :value="days">
              {{ days === 0 ? t('operator.personalToken.neverExpires') : t('operator.personalToken.days', { days }) }}
            </option>
          </select>
        </div>
        <div>
          <label for="personal-token-password" class="input-label">{{ t('profile.currentPassword') }}</label>
          <input
            id="personal-token-password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            class="input"
            data-test="personal-token-password"
          />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="busy" @click="closeGenerate">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="personal-token-generate-form"
            class="btn btn-primary"
            :disabled="busy || password.length === 0"
            data-test="personal-token-submit"
          >
            {{ busy ? t('common.processing') : t('operator.personalToken.confirmGenerate') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showRevoke"
      :title="t('operator.personalToken.revokeTitle')"
      :message="t('operator.personalToken.revokeConfirm')"
      danger
      @confirm="confirmRevoke"
      @cancel="showRevoke = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { personalTokenAPI, type PersonalTokenStatus } from '@/api/personalToken'
import { buildApiUrl } from '@/api/url'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

// 运维管理员个人令牌（fork 本地）：脚本用 Authorization: Bearer <token> 调管理 API，
// 权限与运维网页登录完全一致。明文只在生成后显示一次，只存在组件内存里。
const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const DEFAULT_EXPIRY_DAYS = 90

const status = ref<PersonalTokenStatus | null>(null)
const loading = ref(false)
const busy = ref(false)
const newToken = ref('')
const showGenerate = ref(false)
const showRevoke = ref(false)
const password = ref('')
const expiresInDays = ref(DEFAULT_EXPIRY_DAYS)

const canGenerate = computed(() => Boolean(status.value?.feature_enabled && status.value?.eligible))
const expiryOptions = computed(() => {
  const options = status.value?.expiry_options?.length ? status.value.expiry_options : [30, 90, 180, 365, 0]
  // 永不过期放最后
  return [...options.filter((d) => d > 0), ...options.filter((d) => d === 0)]
})
const curlExample = computed(() => {
  let url = buildApiUrl('/admin/ops/dashboard/overview')
  try {
    url = new URL(url, window.location.origin).toString()
  } catch {
    // 保持相对地址
  }
  return `curl -H "Authorization: Bearer ${newToken.value}" ${url}`
})

async function load() {
  loading.value = true
  try {
    status.value = await personalTokenAPI.getStatus()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('operator.personalToken.loadFailed')))
  } finally {
    loading.value = false
  }
}

function openGenerate() {
  password.value = ''
  expiresInDays.value = expiryOptions.value.includes(DEFAULT_EXPIRY_DAYS)
    ? DEFAULT_EXPIRY_DAYS
    : expiryOptions.value[0]
  showGenerate.value = true
}

function closeGenerate() {
  showGenerate.value = false
  password.value = ''
}

async function submitGenerate() {
  if (busy.value || password.value.length === 0) return
  busy.value = true
  try {
    const result = await personalTokenAPI.generate({
      password: password.value,
      expires_in_days: expiresInDays.value
    })
    newToken.value = result.token
    if (status.value) {
      status.value = { ...status.value, token: result.info }
    }
    closeGenerate()
    appStore.showSuccess(t('operator.personalToken.generated'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('operator.personalToken.generateFailed')))
  } finally {
    busy.value = false
  }
}

async function confirmRevoke() {
  if (busy.value) return
  busy.value = true
  try {
    await personalTokenAPI.revoke()
    newToken.value = ''
    if (status.value) {
      status.value = { ...status.value, token: null }
    }
    appStore.showSuccess(t('operator.personalToken.revoked'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('operator.personalToken.revokeFailed')))
  } finally {
    busy.value = false
    showRevoke.value = false
  }
}

onMounted(load)
</script>
