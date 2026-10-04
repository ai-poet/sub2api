<template>
  <!-- 内容自动翻译（fork 本地功能）：独立加载 / 保存，不属于 SettingsView 的大表单 -->
  <div class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-600" data-test="content-translation-card">
    <div class="flex items-center justify-between gap-4">
      <div>
        <label class="font-medium text-gray-900 dark:text-white">
          {{ t('admin.settings.site.contentTranslation.title') }}
        </label>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.contentTranslation.description') }}
        </p>
      </div>
      <Toggle v-model="form.enabled" data-test="content-translation-enabled" />
    </div>

    <p v-if="loadFailed" class="text-sm text-red-600 dark:text-red-400" data-test="content-translation-load-failed">
      {{ t('admin.settings.site.contentTranslation.loadFailed') }}
    </p>

    <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300" for="content-translation-key">
          {{ t('admin.settings.site.contentTranslation.apiKey') }}
        </label>
        <select
          id="content-translation-key"
          v-model="form.api_key_id"
          class="input text-sm"
          data-test="content-translation-key"
        >
          <option :value="null">{{ t('admin.settings.site.contentTranslation.apiKeyPlaceholder') }}</option>
          <option v-for="option in keyOptions" :key="option.id" :value="option.id">
            {{ option.label }}
          </option>
        </select>
        <p v-if="keysLoadFailed" class="mt-1.5 text-xs text-red-600 dark:text-red-400">
          {{ t('admin.settings.site.contentTranslation.apiKeysLoadFailed') }}
        </p>
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.contentTranslation.apiKeyHint') }}
        </p>
      </div>
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300" for="content-translation-model">
          {{ t('admin.settings.site.contentTranslation.model') }}
        </label>
        <input
          id="content-translation-model"
          v-model.trim="form.model"
          type="text"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.site.contentTranslation.modelPlaceholder')"
          data-test="content-translation-model"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.contentTranslation.modelHint') }}
        </p>
      </div>
      <div class="md:col-span-2">
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300" for="content-translation-base-url">
          {{ t('admin.settings.site.contentTranslation.baseUrl') }}
        </label>
        <input
          id="content-translation-base-url"
          v-model.trim="form.base_url"
          type="url"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.site.contentTranslation.baseUrlPlaceholder')"
          data-test="content-translation-base-url"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.contentTranslation.baseUrlHint', { path: '/v1/chat/completions' }) }}
        </p>
      </div>
      <div class="md:col-span-2">
        <span class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.site.contentTranslation.languages') }}
        </span>
        <div class="flex flex-wrap gap-4">
          <label v-for="lang in LANGUAGES" :key="lang" class="flex cursor-pointer items-center gap-2">
            <input
              v-model="form.languages"
              type="checkbox"
              :value="lang"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :data-test="`content-translation-lang-${lang}`"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ t(`admin.settings.site.contentTranslation.languageNames.${lang}`) }}
            </span>
          </label>
        </div>
      </div>
    </div>

    <!-- 运行状态 -->
    <div
      class="flex flex-col gap-1 rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
      data-test="content-translation-status"
    >
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <template v-if="status">
          <span>
            {{
              t('admin.settings.site.contentTranslation.status.summary', {
                sources: status.sources,
                translated: status.translated,
                pending: status.pending
              })
            }}
          </span>
          <span>·</span>
          <span>
            {{
              status.last_run_at
                ? t('admin.settings.site.contentTranslation.status.lastRun', { time: formatDateTime(status.last_run_at) })
                : t('admin.settings.site.contentTranslation.status.neverRun')
            }}
          </span>
          <span v-if="status.running" class="font-medium text-primary-600 dark:text-primary-400">
            · {{ t('admin.settings.site.contentTranslation.status.running') }}
          </span>
        </template>
        <span v-else-if="statusLoadFailed" class="text-red-600 dark:text-red-400">
          {{ t('admin.settings.site.contentTranslation.status.loadFailed') }}
        </span>
        <button
          type="button"
          class="ml-auto text-primary-600 hover:underline disabled:opacity-50 dark:text-primary-400"
          :disabled="statusLoading"
          @click="loadStatus"
        >
          {{ t('admin.settings.site.contentTranslation.refreshStatus') }}
        </button>
      </div>
      <p v-if="status?.last_error" class="break-words text-red-600 dark:text-red-400" data-test="content-translation-last-error">
        {{
          t('admin.settings.site.contentTranslation.status.lastError', {
            time: status.last_error_at ? formatDateTime(status.last_error_at) : '-',
            error: status.last_error
          })
        }}
      </p>
    </div>

    <!-- 测试结果 -->
    <div
      v-if="testResult"
      class="rounded-lg border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-800 dark:border-green-900/40 dark:bg-green-950/20 dark:text-green-300"
      data-test="content-translation-test-result"
    >
      <p class="text-xs font-medium">
        {{ t('admin.settings.site.contentTranslation.testResult', { ms: testResult.latency_ms }) }}
      </p>
      <p class="mt-1 whitespace-pre-wrap break-words">{{ testResult.translated }}</p>
    </div>

    <div class="flex flex-wrap justify-end gap-2">
      <button type="button" class="btn btn-secondary btn-sm" data-test="content-translation-manage" @click="dialogOpen = true">
        {{ t('admin.settings.site.contentTranslation.manage') }}
      </button>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="syncing"
        data-test="content-translation-sync"
        @click="syncNow"
      >
        {{
          syncing
            ? t('admin.settings.site.contentTranslation.syncing')
            : t('admin.settings.site.contentTranslation.sync')
        }}
      </button>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="testing"
        data-test="content-translation-test"
        @click="runTest"
      >
        {{
          testing
            ? t('admin.settings.site.contentTranslation.testing')
            : t('admin.settings.site.contentTranslation.test')
        }}
      </button>
      <button
        type="button"
        class="btn btn-primary btn-sm"
        :disabled="saving || loading"
        data-test="content-translation-save"
        @click="save"
      >
        {{
          saving
            ? t('admin.settings.site.contentTranslation.saving')
            : t('admin.settings.site.contentTranslation.save')
        }}
      </button>
    </div>

    <ContentTranslationsDialog :show="dialogOpen" @close="dialogOpen = false" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import ContentTranslationsDialog from '@/components/admin/settings/ContentTranslationsDialog.vue'
import contentTranslationsAPI, {
  type ContentTranslationConfig,
  type ContentTranslationConfigInput,
  type ContentTranslationLang,
  type ContentTranslationStatus,
  type ContentTranslationTestResult
} from '@/api/admin/contentTranslations'
import { keysAPI } from '@/api/keys'
import type { ApiKey } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'

// 内容自动翻译（fork 本地功能）的管理卡片。契约见 docs/CONTENT_TRANSLATION.md。
// 管理员 Key 只按 id 引用：这里列出管理员自己的 Key（名称 + 掩码），保存的是 id。

const LANGUAGES: ContentTranslationLang[] = ['zh', 'en', 'ja']
const KEYS_PAGE_SIZE = 100
const KEYS_MAX_PAGES = 10
const STATUS_POLL_MS = 5000

const { t } = useI18n()
const appStore = useAppStore()

const form = reactive<ContentTranslationConfigInput>({
  enabled: false,
  api_key_id: null,
  model: '',
  base_url: '',
  languages: [...LANGUAGES]
})
/** GET 返回的所选 Key 名称：Key 不在列表里（已停用 / 删除）时用于兜底显示 */
const savedKeyName = ref('')

const keys = ref<ApiKey[]>([])
const status = ref<ContentTranslationStatus | null>(null)
const testResult = ref<ContentTranslationTestResult | null>(null)

const loading = ref(false)
const loadFailed = ref(false)
const keysLoadFailed = ref(false)
const statusLoading = ref(false)
const statusLoadFailed = ref(false)
const saving = ref(false)
const testing = ref(false)
const syncing = ref(false)
const dialogOpen = ref(false)

let statusTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false

const keyOptions = computed(() => {
  const options = keys.value.map((key) => {
    const masked = key.key ? maskApiKey(key.key) : key.key_masked || ''
    let label = masked ? `${key.name} (${masked})` : key.name
    if (key.status !== 'active') {
      label += ` · ${t('admin.settings.site.contentTranslation.apiKeyInactive')}`
    }
    return { id: key.id, label }
  })
  const selected = form.api_key_id
  if (selected !== null && !options.some((option) => option.id === selected)) {
    options.unshift({
      id: selected,
      label: `${savedKeyName.value || `#${selected}`} · ${t('admin.settings.site.contentTranslation.apiKeyMissing')}`
    })
  }
  return options
})

function applyConfig(config: ContentTranslationConfig) {
  form.enabled = Boolean(config.enabled)
  form.api_key_id = typeof config.api_key_id === 'number' ? config.api_key_id : null
  form.model = config.model || ''
  form.base_url = config.base_url || ''
  form.languages = Array.isArray(config.languages)
    ? LANGUAGES.filter((lang) => config.languages.includes(lang))
    : [...LANGUAGES]
  savedKeyName.value = config.api_key_name || ''
}

function payload(): ContentTranslationConfigInput {
  return {
    enabled: form.enabled,
    api_key_id: form.api_key_id,
    model: form.model.trim(),
    base_url: form.base_url.trim(),
    languages: LANGUAGES.filter((lang) => form.languages.includes(lang))
  }
}

/** 启用时必须选好 Key、填好模型、至少一种语言；测试同样需要 Key 与模型 */
function validate(requireAll: boolean): boolean {
  if (!requireAll) return true
  const data = payload()
  if (data.api_key_id === null) {
    appStore.showError(t('admin.settings.site.contentTranslation.apiKeyRequired'))
    return false
  }
  if (!data.model) {
    appStore.showError(t('admin.settings.site.contentTranslation.modelRequired'))
    return false
  }
  if (data.languages.length === 0) {
    appStore.showError(t('admin.settings.site.contentTranslation.languagesRequired'))
    return false
  }
  return true
}

async function loadConfig() {
  loading.value = true
  loadFailed.value = false
  try {
    applyConfig(await contentTranslationsAPI.getConfig())
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

async function loadKeys() {
  keysLoadFailed.value = false
  try {
    const collected: ApiKey[] = []
    for (let page = 1; page <= KEYS_MAX_PAGES; page++) {
      const result = await keysAPI.list(page, KEYS_PAGE_SIZE, { sort_by: 'created_at', sort_order: 'desc' })
      collected.push(...(result.items ?? []))
      if (!result.pages || page >= result.pages) break
    }
    keys.value = collected
  } catch {
    keysLoadFailed.value = true
  }
}

function clearStatusTimer() {
  if (statusTimer !== null) {
    clearTimeout(statusTimer)
    statusTimer = null
  }
}

/** 扫描进行中时每 5 秒刷新一次状态，直到结束 */
function scheduleStatusPoll() {
  clearStatusTimer()
  if (disposed || !status.value?.running) return
  statusTimer = setTimeout(() => {
    statusTimer = null
    void loadStatus()
  }, STATUS_POLL_MS)
}

async function loadStatus() {
  statusLoading.value = true
  statusLoadFailed.value = false
  try {
    status.value = await contentTranslationsAPI.getStatus()
  } catch {
    statusLoadFailed.value = true
  } finally {
    statusLoading.value = false
  }
  scheduleStatusPoll()
}

async function save() {
  if (saving.value) return
  if (!validate(form.enabled)) return
  saving.value = true
  try {
    applyConfig(await contentTranslationsAPI.updateConfig(payload()))
    appStore.showSuccess(t('admin.settings.site.contentTranslation.saved'))
    void loadStatus()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.contentTranslation.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function runTest() {
  if (testing.value) return
  if (!validate(true)) return
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await contentTranslationsAPI.testConfig(payload())
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.contentTranslation.testFailed')))
  } finally {
    testing.value = false
  }
}

async function syncNow() {
  if (syncing.value) return
  syncing.value = true
  try {
    status.value = await contentTranslationsAPI.sync()
    statusLoadFailed.value = false
    appStore.showSuccess(t('admin.settings.site.contentTranslation.syncStarted'))
    // 扫描是异步的：返回时可能还没标记 running，稍后再看一次
    clearStatusTimer()
    if (!disposed) {
      statusTimer = setTimeout(() => {
        statusTimer = null
        void loadStatus()
      }, STATUS_POLL_MS)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.contentTranslation.syncFailed')))
  } finally {
    syncing.value = false
  }
}

onMounted(() => {
  void loadConfig()
  void loadKeys()
  void loadStatus()
})

onBeforeUnmount(() => {
  disposed = true
  clearStatusTimer()
})
</script>
