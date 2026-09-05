<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="group" class="space-y-6">
      <div
        class="flex flex-col gap-3 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/80 md:flex-row md:items-center md:justify-between"
      >
        <div>
          <div class="text-base font-semibold text-gray-900 dark:text-white">{{ group.name }}</div>
          <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t(`admin.groups.platforms.${group.platform}`) }}
            <span v-if="group.description" class="ml-2">{{ group.description }}</span>
          </div>
        </div>
        <span :class="['badge', summaryBadgeClass]">{{ summaryStatusText }}</span>
      </div>

      <div
        v-if="loading"
        class="flex items-center justify-center py-10"
      >
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else class="space-y-6">
        <div v-if="loadError" class="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-900/40 dark:bg-rose-950/20 dark:text-rose-300">
          {{ loadError }}
        </div>

        <div class="flex items-center justify-between rounded-xl border border-gray-200 px-4 py-4 dark:border-dark-700">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.enabled') }}
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.runtimeStatus.enabledHint') }}
            </p>
          </div>
          <Toggle v-model="form.enabled" />
        </div>

        <div class="flex items-center justify-between rounded-xl border border-gray-200 px-4 py-4 dark:border-dark-700">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.notifyEnabled') }}
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.groups.runtimeStatus.notifyEnabledHint') }}
            </p>
          </div>
          <Toggle v-model="form.notify_enabled" />
        </div>

        <div class="grid gap-5 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.probeModel') }}</label>
            <input
              v-model.trim="form.probe_model"
              type="text"
              class="input"
              :placeholder="t('admin.groups.runtimeStatus.probeModelPlaceholder')"
            />
          </div>

          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.validationMode') }}</label>
            <select v-model="form.validation_mode" class="input">
              <option
                v-for="option in validationModeOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.groups.runtimeStatus.probePrompt') }}</label>
          <textarea
            v-model="form.probe_prompt"
            rows="4"
            class="input"
            :placeholder="t('admin.groups.runtimeStatus.probePromptPlaceholder')"
          ></textarea>
        </div>

        <div v-if="showKeywordEditor">
          <label class="input-label">{{ t('admin.groups.runtimeStatus.expectedKeywords') }}</label>
          <textarea
            v-model="expectedKeywordsText"
            rows="4"
            class="input"
            :placeholder="t('admin.groups.runtimeStatus.expectedKeywordsPlaceholder')"
          ></textarea>
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.groups.runtimeStatus.expectedKeywordsHint') }}
          </p>
        </div>

        <div class="grid gap-5 md:grid-cols-3">
          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.intervalSeconds') }}</label>
            <input
              v-model.number="form.interval_seconds"
              type="number"
              min="10"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.timeoutSeconds') }}</label>
            <input
              v-model.number="form.timeout_seconds"
              type="number"
              min="1"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">{{ t('admin.groups.runtimeStatus.slowLatencyMs') }}</label>
            <input
              v-model.number="form.slow_latency_ms"
              type="number"
              min="100"
              step="100"
              class="input"
            />
          </div>
        </div>

        <div
          v-if="group.platform === 'openai'"
          class="space-y-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
        >
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.solJuice.title') }}
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.solJuice.hint') }}
              </p>
            </div>
            <Toggle v-model="form.sol_juice_enabled" />
          </div>

          <div class="grid gap-5 md:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.solJuice.intervalSeconds') }}</label>
              <input
                v-model.number="form.sol_juice_interval_seconds"
                type="number"
                min="300"
                step="60"
                class="input"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.groups.runtimeStatus.solJuice.model') }}</label>
              <input
                v-model.trim="form.sol_juice_model"
                type="text"
                class="input"
                :placeholder="t('admin.groups.runtimeStatus.solJuice.modelPlaceholder')"
              />
            </div>
          </div>

          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.groups.runtimeStatus.solJuice.latestResult') }}
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="loading || saving || probing || solJuiceProbing"
              @click="handleSolJuiceProbe"
            >
              <span
                v-if="solJuiceProbing"
                class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              ></span>
              {{ solJuiceProbing ? t('admin.groups.runtimeStatus.solJuice.probing') : t('admin.groups.runtimeStatus.solJuice.probeNow') }}
            </button>
          </div>

          <div v-if="summary.sol_juice_checked_at" class="space-y-3">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.status') }}
                </div>
                <div class="mt-1">
                  <span :class="['badge', getSolJuiceBadgeClass(solJuiceDisplayStatus)]">
                    {{ t(`admin.groups.runtimeStatus.solJuice.statuses.${solJuiceDisplayStatus}`) }}
                  </span>
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.value') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.sol_juice_value || '-' }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.checkedAt') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(summary.sol_juice_checked_at) }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(summary.sol_juice_checked_at) }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.tokens') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.sol_juice_input_tokens }} / {{ summary.sol_juice_output_tokens }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.reasoningTokens') }}: {{ summary.sol_juice_reasoning_tokens }}
                </div>
              </div>
            </div>

            <div class="grid gap-3 md:grid-cols-2">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.lastCost') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatUsd(summary.sol_juice_last_cost_usd) }}
                </div>
              </div>
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.solJuice.monthlyEstimate') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatUsd(solJuiceMonthlyCost, 2) }}
                </div>
              </div>
            </div>

            <div
              v-if="summary.sol_juice_detail"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.solJuice.detail') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-200">{{ summary.sol_juice_detail }}</pre>
            </div>
          </div>
          <div
            v-else
            class="rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            {{ t('admin.groups.runtimeStatus.solJuice.latestResultEmpty') }}
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.groups.runtimeStatus.latestResult') }}
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.latestResultHint') }}
              </p>
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="loading || saving || probing"
              @click="handleProbe"
            >
              <span
                v-if="probing"
                class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
              ></span>
              {{ probing ? t('admin.groups.runtimeStatus.probing') : t('admin.groups.runtimeStatus.probeNow') }}
            </button>
          </div>

          <div v-if="summary.observed_at" class="mt-4 space-y-4">
            <div class="grid gap-3 md:grid-cols-4">
              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.currentStatus') }}
                </div>
                <div class="mt-1">
                  <span :class="['badge', summaryBadgeClass]">{{ summaryStatusText }}</span>
                </div>
              </div>

              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.checkedAt') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatDateTime(summary.observed_at) }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ formatRelativeTime(summary.observed_at) }}
                </div>
              </div>

              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.latency') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatGroupRuntimeLatency(summary.latency_ms) }}
                </div>
              </div>

              <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.runtimeStatus.httpCode') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ summary.http_code ?? '-' }}
                </div>
              </div>
            </div>

            <div
              v-if="summary.sub_status"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200"
            >
              <span class="font-medium">{{ t('admin.groups.runtimeStatus.subStatus') }}:</span>
              <span class="ml-2">{{ summary.sub_status }}</span>
            </div>

            <div
              v-if="summary.response_excerpt"
              class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.groups.runtimeStatus.responseExcerpt') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-200">{{ summary.response_excerpt }}</pre>
            </div>

            <div
              v-if="summary.error_detail"
              class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-3 dark:border-rose-900/40 dark:bg-rose-950/20"
            >
              <div class="text-xs font-medium text-rose-700 dark:text-rose-300">
                {{ t('admin.groups.runtimeStatus.errorDetail') }}
              </div>
              <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-rose-700 dark:text-rose-300">{{ summary.error_detail }}</pre>
            </div>
          </div>

          <div
            v-else
            class="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
          >
            {{ t('admin.groups.runtimeStatus.latestResultEmpty') }}
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex w-full items-center justify-between gap-3">
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.runtimeStatus.footerHint') }}
        </div>
        <div class="flex items-center gap-3">
          <button type="button" class="btn btn-secondary" :disabled="saving || probing" @click="emit('close')">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="loading || saving || probing || !group" @click="handleSave">
            <span
              v-if="saving"
              class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
            ></span>
            {{ saving ? t('common.saving') : t('admin.groups.runtimeStatus.save') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores'
import type {
  AdminGroup,
  GroupStatusAdminView,
  GroupStatusSummary,
  GroupStatusValidationMode,
} from '@/types'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import {
  estimateSolJuiceMonthlyCostUsd,
  formatGroupRuntimeLatency,
  formatUsd,
  getGroupRuntimeStatusBadgeClass,
  getSolJuiceBadgeClass,
  joinRuntimeKeywordsText,
  normalizeSolJuiceStatus,
  normalizeGroupRuntimeStatus,
  shouldShowRuntimeKeywordEditor,
  splitRuntimeKeywordsText,
} from '@/utils/groupStatus'

interface Props {
  show: boolean
  group: AdminGroup | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'updated'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const probing = ref(false)
const loadError = ref('')
const expectedKeywordsText = ref('')
const currentView = ref<GroupStatusAdminView | null>(null)

const form = reactive({
  enabled: false,
  probe_model: '',
  probe_prompt: '',
  validation_mode: 'non_empty' as GroupStatusValidationMode,
  interval_seconds: 60,
  timeout_seconds: 30,
  slow_latency_ms: 15000,
  notify_enabled: true,
  sol_juice_enabled: false,
  sol_juice_interval_seconds: 900,
  sol_juice_model: 'gpt-5.6-sol',
})

const solJuiceProbing = ref(false)

const dialogTitle = computed(() => {
  if (!props.group) {
    return t('admin.groups.runtimeStatus.titlePlain')
  }
  return t('admin.groups.runtimeStatus.title', { name: props.group.name })
})

const validationModeOptions = computed(() => [
  {
    value: 'non_empty' as GroupStatusValidationMode,
    label: t('admin.groups.runtimeStatus.validationModes.nonEmpty')
  },
  {
    value: 'keywords_any' as GroupStatusValidationMode,
    label: t('admin.groups.runtimeStatus.validationModes.keywordsAny')
  },
  {
    value: 'keywords_all' as GroupStatusValidationMode,
    label: t('admin.groups.runtimeStatus.validationModes.keywordsAll')
  }
])

const summary = computed<GroupStatusSummary>(() => {
  return currentView.value?.summary ?? {
    group_id: props.group?.id ?? 0,
    config_id: 0,
    enabled: false,
    probe_model: '',
    latest_status: '',
    stable_status: '',
    response_excerpt: '',
    latency_ms: null,
    http_code: null,
    sub_status: '',
    error_detail: '',
    observed_at: null,
    consecutive_down: 0,
    consecutive_non_down: 0,
    sol_juice_enabled: false,
    sol_juice_model: '',
    sol_juice_interval_seconds: 900,
    sol_juice_status: '',
    sol_juice_stable_status: '',
    sol_juice_value: '',
    sol_juice_detail: '',
    sol_juice_checked_at: null,
    sol_juice_consecutive_mismatch: 0,
    sol_juice_input_tokens: 0,
    sol_juice_output_tokens: 0,
    sol_juice_reasoning_tokens: 0,
    sol_juice_last_cost_usd: 0
  }
})

const solJuiceDisplayStatus = computed(() =>
  normalizeSolJuiceStatus(summary.value.sol_juice_stable_status || summary.value.sol_juice_status)
)

const solJuiceMonthlyCost = computed(() =>
  estimateSolJuiceMonthlyCostUsd(summary.value.sol_juice_last_cost_usd, Number(form.sol_juice_interval_seconds) || 0)
)

const summaryStatus = computed(() => {
  if (!form.enabled) {
    return 'unknown'
  }
  if (summary.value.stable_status) {
    return normalizeGroupRuntimeStatus(summary.value.stable_status)
  }
  if (summary.value.latest_status) {
    return normalizeGroupRuntimeStatus(summary.value.latest_status)
  }
  return 'unknown'
})

const summaryBadgeClass = computed(() => getGroupRuntimeStatusBadgeClass(summaryStatus.value))

const summaryStatusText = computed(() => {
  if (!form.enabled) {
    return t('admin.groups.runtimeStatus.disabled')
  }
  if (!summary.value.observed_at) {
    return t('admin.groups.runtimeStatus.waiting')
  }
  return t(`modelStatus.statuses.${summaryStatus.value}`)
})

const showKeywordEditor = computed(() => shouldShowRuntimeKeywordEditor(form.validation_mode))

function resetForm() {
  form.enabled = false
  form.probe_model = ''
  form.probe_prompt = ''
  form.validation_mode = 'non_empty'
  form.interval_seconds = 60
  form.timeout_seconds = 30
  form.slow_latency_ms = 15000
  form.notify_enabled = true
  form.sol_juice_enabled = false
  form.sol_juice_interval_seconds = 900
  form.sol_juice_model = 'gpt-5.6-sol'
  expectedKeywordsText.value = ''
}

function applyView(view: GroupStatusAdminView) {
  currentView.value = view
  form.enabled = view.config.enabled
  form.probe_model = view.config.probe_model
  form.probe_prompt = view.config.probe_prompt
  form.validation_mode = view.config.validation_mode
  form.interval_seconds = view.config.interval_seconds
  form.timeout_seconds = view.config.timeout_seconds
  form.slow_latency_ms = view.config.slow_latency_ms
  form.notify_enabled = view.config.notify_enabled !== false
  form.sol_juice_enabled = view.config.sol_juice_enabled === true
  form.sol_juice_interval_seconds = view.config.sol_juice_interval_seconds || 900
  form.sol_juice_model = view.config.sol_juice_model || 'gpt-5.6-sol'
  expectedKeywordsText.value = joinRuntimeKeywordsText(view.config.expected_keywords)
}

async function loadRuntimeStatus(groupId: number) {
  loading.value = true
  loadError.value = ''
  try {
    const view = await adminAPI.groups.getRuntimeStatus(groupId)
    applyView(view)
  } catch (error: any) {
    loadError.value = error?.message || t('admin.groups.runtimeStatus.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function saveRuntimeStatus(showToast = true): Promise<GroupStatusAdminView | null> {
  if (!props.group) {
    return null
  }

  saving.value = true
  loadError.value = ''
  try {
    const view = await adminAPI.groups.updateRuntimeStatus(props.group.id, {
      enabled: form.enabled,
      probe_model: form.probe_model.trim(),
      probe_prompt: form.probe_prompt.trim(),
      validation_mode: form.validation_mode,
      expected_keywords: splitRuntimeKeywordsText(expectedKeywordsText.value),
      interval_seconds: Math.max(10, Math.round(Number(form.interval_seconds) || 60)),
      timeout_seconds: Math.max(1, Math.round(Number(form.timeout_seconds) || 30)),
      slow_latency_ms: Math.max(100, Math.round(Number(form.slow_latency_ms) || 15000)),
      notify_enabled: form.notify_enabled,
      sol_juice_enabled: form.sol_juice_enabled,
      sol_juice_interval_seconds: Math.max(300, Math.round(Number(form.sol_juice_interval_seconds) || 900)),
      sol_juice_model: form.sol_juice_model.trim() || 'gpt-5.6-sol',
    })
    applyView(view)
    if (showToast) {
      appStore.showSuccess(t('admin.groups.runtimeStatus.saved'))
    }
    emit('updated')
    return view
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.failedToSave'))
    return null
  } finally {
    saving.value = false
  }
}

async function handleSave() {
  const saved = await saveRuntimeStatus(true)
  if (saved) {
    emit('close')
  }
}

async function handleProbe() {
  if (!props.group) {
    return
  }

  const saved = await saveRuntimeStatus(false)
  if (!saved) {
    return
  }

  probing.value = true
  try {
    const view = await adminAPI.groups.probeRuntimeStatus(props.group.id)
    applyView(view)
    appStore.showSuccess(t('admin.groups.runtimeStatus.probeSucceeded'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.probeFailed'))
  } finally {
    probing.value = false
  }
}

async function handleSolJuiceProbe() {
  if (!props.group) {
    return
  }

  const saved = await saveRuntimeStatus(false)
  if (!saved) {
    return
  }

  solJuiceProbing.value = true
  try {
    const view = await adminAPI.groups.probeRuntimeStatusSolJuice(props.group.id)
    applyView(view)
    appStore.showSuccess(t('admin.groups.runtimeStatus.solJuice.probeSucceeded'))
    emit('updated')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.groups.runtimeStatus.solJuice.probeFailed'))
  } finally {
    solJuiceProbing.value = false
  }
}

watch(
  () => [props.show, props.group?.id] as const,
  ([show, groupId]) => {
    if (!show || !groupId) {
      if (!show) {
        resetForm()
        currentView.value = null
        loadError.value = ''
      }
      return
    }
    void loadRuntimeStatus(groupId)
  },
  { immediate: true }
)
</script>
