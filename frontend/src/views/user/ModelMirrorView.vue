<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div class="grid gap-4 md:grid-cols-3">
        <div class="card p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelMirror.statusLabel') }}</div>
          <div class="mt-3 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ statusTitle }}
          </div>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ statusSubtitle }}</p>
        </div>

        <div class="card p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelMirror.verdictLabel') }}</div>
          <div
            class="mt-3 inline-flex rounded-full px-3 py-1 text-sm font-medium"
            :class="verdictBadgeClass"
          >
            {{ verdictLabel }}
          </div>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ verdictDescription }}</p>
        </div>

        <div class="card p-5">
          <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelMirror.scoreLabel') }}</div>
          <div class="mt-3 flex items-end gap-2">
            <span class="text-4xl font-semibold text-gray-900 dark:text-white">{{ score }}</span>
            <span class="pb-1 text-sm text-gray-500 dark:text-gray-400">/ 100</span>
          </div>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ t('modelMirror.scoreHint', { passed: passedChecks, total: results.length }) }}
          </p>
        </div>
      </div>

      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('modelMirror.configTitle') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('modelMirror.configDescription') }}
          </p>
        </div>
        <div class="space-y-5 p-6">
          <div
            v-if="blockedByBackendMode"
            class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300"
          >
            {{ t('modelMirror.backendModeBlocked') }}
          </div>

          <div class="grid gap-4 lg:grid-cols-[1.4fr_1fr]">
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('modelMirror.endpoint') }}
              </label>
              <input
                v-model="form.api_endpoint"
                type="text"
                class="input"
                :placeholder="t('modelMirror.endpointPlaceholder')"
                :disabled="running"
              />
            </div>

            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('modelMirror.model') }}
              </label>
              <input
                v-model="form.api_model"
                list="model-mirror-models"
                type="text"
                class="input"
                :placeholder="t('modelMirror.modelPlaceholder')"
                :disabled="running"
              />
              <datalist id="model-mirror-models">
                <option v-for="item in recommendedModels" :key="item" :value="item"></option>
              </datalist>
            </div>
          </div>

          <div>
            <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('modelMirror.apiKey') }}
            </label>
            <input
              v-model="form.api_key"
              type="password"
              class="input"
              :placeholder="t('modelMirror.apiKeyPlaceholder')"
              :disabled="running"
              autocomplete="off"
            />
          </div>

          <div class="grid gap-3 lg:grid-cols-2">
            <div class="rounded-xl border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-700 dark:border-blue-900/50 dark:bg-blue-950/30 dark:text-blue-300">
              {{ t('modelMirror.privacyHint') }}
            </div>
            <div class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300">
              {{ t('modelMirror.securityHint') }}
            </div>
          </div>

          <div
            v-if="stepMessage"
            class="rounded-xl border border-primary-200 bg-primary-50 px-4 py-3 text-sm text-primary-700 dark:border-primary-900/50 dark:bg-primary-950/30 dark:text-primary-300"
          >
            {{ stepMessage }}
          </div>

          <div class="flex flex-wrap justify-end gap-3">
            <button type="button" class="btn btn-secondary" :disabled="running" @click="resetState">
              {{ t('modelMirror.reset') }}
            </button>
            <button v-if="running" type="button" class="btn btn-secondary" @click="handleStopVerification">
              {{ t('modelMirror.stop') }}
            </button>
            <button
              v-else
              type="button"
              class="btn btn-primary"
              :disabled="blockedByBackendMode"
              @click="runVerification"
            >
              {{ t('modelMirror.start') }}
            </button>
          </div>
        </div>
      </div>

      <div class="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('modelMirror.checksTitle') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('modelMirror.checksDescription') }}
            </p>
          </div>

          <div v-if="results.length === 0" class="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('modelMirror.emptyChecks') }}
          </div>

          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <div
              v-for="result in results"
              :key="result.id"
              class="flex flex-col gap-3 px-6 py-4 md:flex-row md:items-start md:justify-between"
            >
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <div class="font-medium text-gray-900 dark:text-white">{{ result.label }}</div>
                  <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-300">
                    {{ t('modelMirror.weight', { weight: result.weight }) }}
                  </span>
                </div>
                <p class="mt-2 break-words text-sm text-gray-500 dark:text-gray-400">
                  {{ result.detail }}
                </p>
              </div>
              <span
                class="inline-flex shrink-0 rounded-full px-3 py-1 text-xs font-medium"
                :class="checkBadgeClass(result)"
              >
                {{ checkStatusLabel(result) }}
              </span>
            </div>
          </div>
        </div>

        <div class="space-y-6">
          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('modelMirror.evidenceTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('modelMirror.evidenceDescription') }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div>
                <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('modelMirror.upstreamModel') }}
                </div>
                <div class="rounded-xl bg-gray-50 px-4 py-3 font-mono text-sm text-gray-700 dark:bg-dark-800 dark:text-gray-200">
                  {{ donePayload?.upstream_model || '-' }}
                </div>
              </div>

              <div>
                <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('modelMirror.responseExcerpt') }}
                </div>
                <pre class="max-h-64 overflow-auto rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-700 dark:bg-dark-800 dark:text-gray-200">{{ donePayload?.response_excerpt || '-' }}</pre>
              </div>

              <div>
                <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('modelMirror.thinkingExcerpt') }}
                </div>
                <pre class="max-h-64 overflow-auto rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-700 dark:bg-dark-800 dark:text-gray-200">{{ donePayload?.thinking_excerpt || '-' }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  verifyModelMirror,
  type ModelMirrorCheckResult,
  type ModelMirrorDonePayload,
  type ModelMirrorVerdict
} from '@/api/modelMirror'
import { useAppStore, useAuthStore } from '@/stores'

const STORAGE_KEY = 'sub2api-model-mirror-config'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const recommendedModels = ['claude-opus-4-6', 'claude-sonnet-4-6']

const form = reactive({
  api_endpoint: '',
  api_key: '',
  api_model: 'claude-opus-4-6'
})

const running = ref(false)
const stepMessage = ref('')
const results = ref<ModelMirrorCheckResult[]>([])
const donePayload = ref<ModelMirrorDonePayload | null>(null)
const status = ref<'idle' | 'testing' | 'pass' | 'fail'>('idle')
const customStatusTitle = ref(t('modelMirror.idleTitle'))
const customStatusSubtitle = ref(t('modelMirror.idleSubtitle'))
const controller = ref<AbortController | null>(null)

const blockedByBackendMode = computed(() => appStore.backendModeEnabled && !authStore.isAdmin)
const score = computed(() => donePayload.value?.score ?? 0)
const passedChecks = computed(() => results.value.filter((item) => item.pass).length)
const verdict = computed<ModelMirrorVerdict>(() => donePayload.value?.verdict ?? 'pending')

const statusTitle = computed(() => customStatusTitle.value)
const statusSubtitle = computed(() => customStatusSubtitle.value)

const verdictLabel = computed(() => t(`modelMirror.verdicts.${verdict.value}.label`))
const verdictDescription = computed(() => t(`modelMirror.verdicts.${verdict.value}.description`))

const verdictBadgeClass = computed(() => {
  switch (verdict.value) {
    case 'max_pure':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
    case 'official_api':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'
    case 'reverse_proxy':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
    case 'likely_not_claude':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
})

function loadSavedConfig() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return
    }
    const parsed = JSON.parse(raw) as Partial<typeof form>
    if (typeof parsed.api_endpoint === 'string') {
      form.api_endpoint = parsed.api_endpoint
    }
    if (typeof parsed.api_model === 'string' && parsed.api_model.trim()) {
      form.api_model = parsed.api_model
    }
  } catch {
    // ignore malformed local cache
  }
}

function saveConfig() {
  localStorage.setItem(
    STORAGE_KEY,
    JSON.stringify({
      api_endpoint: form.api_endpoint,
      api_model: form.api_model
    })
  )
}

watch(
  () => [form.api_endpoint, form.api_model],
  () => {
    saveConfig()
  },
  { deep: false }
)

function normalizeEndpoint(value: string): string {
  const trimmed = value.trim().replace(/\/+$/, '')
  if (!trimmed) {
    return ''
  }
  return trimmed.endsWith('/v1/messages') ? trimmed : `${trimmed}/v1/messages`
}

function setStatus(next: typeof status.value, title: string, subtitle: string) {
  status.value = next
  customStatusTitle.value = title
  customStatusSubtitle.value = subtitle
}

async function runVerification() {
  if (!form.api_endpoint.trim()) {
    appStore.showError(t('modelMirror.endpointRequired'))
    return
  }
  if (!form.api_key.trim()) {
    appStore.showError(t('modelMirror.apiKeyRequired'))
    return
  }
  if (!form.api_model.trim()) {
    appStore.showError(t('modelMirror.modelRequired'))
    return
  }

  stopVerification(false)

  const abortController = new AbortController()
  controller.value = abortController
  running.value = true
  results.value = []
  donePayload.value = null
  stepMessage.value = t('modelMirror.starting')
  setStatus('testing', t('modelMirror.testingTitle'), t('modelMirror.testingSubtitle'))

  try {
    await verifyModelMirror(
      {
        api_endpoint: normalizeEndpoint(form.api_endpoint),
        api_key: form.api_key.trim(),
        api_model: form.api_model.trim()
      },
      {
        onStep(message) {
          stepMessage.value = message
        },
        onCheck(result) {
          results.value = [...results.value, result]
        },
        onDone(payload) {
          donePayload.value = payload
          stepMessage.value = ''
          form.api_key = ''
          if (payload.verdict === 'likely_not_claude') {
            setStatus('fail', t('modelMirror.verdicts.likely_not_claude.label'), t('modelMirror.verdicts.likely_not_claude.description'))
          } else {
            setStatus('pass', verdictLabel.value, verdictDescription.value)
          }
        },
        onError(message) {
          stepMessage.value = ''
          setStatus('fail', t('modelMirror.failedTitle'), message)
          appStore.showError(message)
        }
      },
      abortController.signal
    )
  } catch (error: any) {
    if (error?.name === 'AbortError') {
      return
    }
    stepMessage.value = ''
    const message = error?.message || t('common.unknownError')
    setStatus('fail', t('modelMirror.failedTitle'), message)
    appStore.showError(message)
  } finally {
    if (controller.value === abortController) {
      controller.value = null
    }
    running.value = false
  }
}

function stopVerification(showToast = true) {
  if (controller.value) {
    controller.value.abort()
    controller.value = null
  }
  if (running.value) {
    running.value = false
    stepMessage.value = ''
    setStatus('idle', t('modelMirror.stoppedTitle'), t('modelMirror.stoppedSubtitle'))
    if (showToast) {
      appStore.showInfo(t('modelMirror.stoppedToast'))
    }
  }
}

function handleStopVerification() {
  stopVerification(true)
}

function resetState() {
  stopVerification(false)
  results.value = []
  donePayload.value = null
  stepMessage.value = ''
  form.api_key = ''
  setStatus('idle', t('modelMirror.idleTitle'), t('modelMirror.idleSubtitle'))
}

function checkBadgeClass(result: ModelMirrorCheckResult) {
  if (result.status === 'info') {
    return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  }
  if (result.pass) {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
  }
  return 'bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300'
}

function checkStatusLabel(result: ModelMirrorCheckResult) {
  if (result.status === 'info') {
    return t('modelMirror.info')
  }
  return result.pass ? t('modelMirror.pass') : t('modelMirror.fail')
}

onBeforeUnmount(() => {
  stopVerification(false)
})

loadSavedConfig()
</script>
