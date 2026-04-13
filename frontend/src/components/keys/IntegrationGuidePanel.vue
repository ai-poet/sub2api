<template>
  <section class="rounded-3xl border border-gray-200/80 bg-white/90 p-5 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80">
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div class="space-y-2">
          <div class="flex flex-wrap items-center gap-2">
            <span class="inline-flex items-center gap-2 rounded-full bg-gray-100 px-3 py-1 text-xs font-semibold text-gray-700 dark:bg-dark-800 dark:text-gray-200">
              <PlatformIcon :platform="props.platform" size="sm" />
              {{ platformLabel }}
            </span>
            <span
              class="inline-flex items-center rounded-full px-3 py-1 text-xs font-medium"
              :class="exampleMode
                ? 'bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300'
                : 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300'"
            >
              {{ exampleMode ? t('integrationGuide.exampleBadge') : t('integrationGuide.liveBadge') }}
            </span>
          </div>
          <p class="max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
            {{ platformDescription }}
          </p>
        </div>

        <div class="rounded-2xl border border-gray-200/80 bg-gray-50/90 px-4 py-3 text-xs text-gray-500 shadow-inner dark:border-dark-700 dark:bg-dark-800/80 dark:text-gray-400">
          <div class="font-semibold text-gray-700 dark:text-gray-200">
            {{ t('integrationGuide.keyLabel') }}
          </div>
          <div v-if="selectedKey" class="mt-1.5 max-w-[18rem] truncate font-mono text-[11px] text-gray-600 dark:text-gray-300">
            {{ maskKey(selectedKey.key) }}
          </div>
          <div v-else class="mt-1.5 max-w-[18rem] text-[11px] leading-5">
            {{ t('integrationGuide.noKeysForPlatform') }}
          </div>
        </div>
      </div>

      <div class="rounded-2xl border border-gray-200/80 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-800/60">
        <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
          <div>
            <label class="input-label">{{ t('integrationGuide.selectKey') }}</label>
            <select
              :value="selectedKeyId"
              class="input"
              :disabled="props.apiKeys.length === 0"
              @change="onKeyChange"
            >
              <option
                v-for="apiKeyItem in props.apiKeys"
                :key="apiKeyItem.id"
                :value="String(apiKeyItem.id)"
              >
                {{ formatKeyOption(apiKeyItem) }}
              </option>
              <option v-if="props.apiKeys.length === 0" value="">
                {{ t('integrationGuide.exampleOption') }}
              </option>
            </select>
          </div>

          <div
            class="rounded-xl px-3 py-2 text-xs"
            :class="exampleMode
              ? 'bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300'
              : 'bg-white text-gray-600 shadow-sm ring-1 ring-gray-200 dark:bg-dark-900 dark:text-gray-300 dark:ring-dark-700'"
          >
            {{ exampleMode ? t('integrationGuide.exampleDescription') : t('integrationGuide.keyHelp') }}
          </div>
        </div>
      </div>

      <div v-if="clientTabs.length" class="border-b border-gray-200 dark:border-dark-700">
        <nav class="-mb-px flex flex-wrap gap-x-6 gap-y-2" aria-label="Client">
          <button
            v-for="tab in clientTabs"
            :key="tab.id"
            type="button"
            @click="activeClientTab = tab.id"
            :class="[
              'inline-flex items-center gap-2 border-b-2 py-2.5 text-sm font-medium transition-colors',
              activeClientTab === tab.id
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
            ]"
          >
            <component :is="tab.icon" class="h-4 w-4" />
            {{ tab.label }}
          </button>
        </nav>
      </div>

      <div v-if="showShellTabs" class="border-b border-gray-200 dark:border-dark-700">
        <nav class="-mb-px flex flex-wrap gap-x-4 gap-y-2" aria-label="Shell">
          <button
            v-for="tab in currentTabs"
            :key="tab.id"
            type="button"
            @click="activeTab = tab.id"
            :class="[
              'inline-flex items-center gap-2 border-b-2 py-2.5 text-sm font-medium transition-colors',
              activeTab === tab.id
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
            ]"
          >
            <component :is="tab.icon" class="h-4 w-4" />
            {{ tab.label }}
          </button>
        </nav>
      </div>

      <div class="space-y-4">
        <div
          v-for="(file, index) in currentFiles"
          :key="`${file.path}-${index}`"
          class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-950 shadow-sm dark:border-dark-700"
        >
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-800 bg-gray-900/90 px-4 py-3">
            <div class="space-y-1">
              <div class="text-xs font-mono text-gray-300">
                {{ file.path }}
              </div>
              <p v-if="file.hint" class="text-[11px] leading-5 text-amber-300/90">
                {{ file.hint }}
              </p>
            </div>

            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-medium transition-colors"
              :class="copiedIndex === index
                ? 'bg-emerald-500/20 text-emerald-300'
                : 'bg-gray-800 text-gray-200 hover:bg-gray-700'"
              @click="copyContent(file.content, index)"
            >
              <svg v-if="copiedIndex === index" class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              <svg v-else class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
              </svg>
              {{ copiedIndex === index ? t('keys.useKeyModal.copied') : t('keys.useKeyModal.copy') }}
            </button>
          </div>

          <pre class="overflow-x-auto p-4 text-sm leading-6 text-gray-100"><code>{{ file.content }}</code></pre>
        </div>
      </div>

      <div
        v-if="showPlatformNote"
        class="flex items-start gap-3 rounded-2xl border border-blue-100 bg-blue-50/80 p-4 text-sm text-blue-700 dark:border-blue-900/40 dark:bg-blue-950/20 dark:text-blue-200"
      >
        <Icon name="infoCircle" size="md" class="mt-0.5 flex-shrink-0" />
        <p class="leading-6">{{ platformNote }}</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, h, ref, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { ApiKey, GroupPlatform } from '@/types'

interface Props {
  platform: GroupPlatform
  apiKeys: ApiKey[]
  baseUrl: string
}

interface TabConfig {
  id: string
  label: string
  icon: Component
}

interface FileConfig {
  path: string
  content: string
  hint?: string
}

const props = defineProps<Props>()

const { t } = useI18n()
const { copyToClipboard: clipboardCopy } = useClipboard()

const copiedIndex = ref<number | null>(null)
const activeTab = ref<string>('unix')
const activeClientTab = ref<string>('claude')
const selectedKeyId = ref<string>('')
const exampleApiKey = 'YOUR_API_KEY'

const defaultClientTab = computed(() => {
  switch (props.platform) {
    case 'openai':
      return 'codex'
    default:
      return 'claude'
  }
})

watch(
  () => props.apiKeys,
  (keys) => {
    if (!keys.length) {
      selectedKeyId.value = ''
      return
    }

    if (!keys.some((key) => String(key.id) === selectedKeyId.value)) {
      selectedKeyId.value = String(keys[0].id)
    }
  },
  { immediate: true }
)

watch(
  () => props.platform,
  () => {
    activeTab.value = 'unix'
    activeClientTab.value = defaultClientTab.value
  },
  { immediate: true }
)

const selectedKey = computed(() =>
  props.apiKeys.find((key) => String(key.id) === selectedKeyId.value) ?? null
)

const exampleMode = computed(() => !selectedKey.value)
const effectiveApiKey = computed(() => selectedKey.value?.key || exampleApiKey)
const allowMessagesDispatch = computed(() => selectedKey.value?.group?.allow_messages_dispatch ?? false)

function onKeyChange(event: Event) {
  const target = event.target as HTMLSelectElement
  selectedKeyId.value = target.value
}

const AppleIcon = {
  render() {
    return h('svg', {
      fill: 'currentColor',
      viewBox: '0 0 24 24',
      class: 'h-4 w-4'
    }, [
      h('path', { d: 'M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z' })
    ])
  }
}

const WindowsIcon = {
  render() {
    return h('svg', {
      fill: 'currentColor',
      viewBox: '0 0 24 24',
      class: 'h-4 w-4'
    }, [
      h('path', { d: 'M3 12V6.75l6-1.32v6.48L3 12zm17-9v8.75l-10 .15V5.21L20 3zM3 13l6 .09v6.81l-6-1.15V13zm7 .25l10 .15V21l-10-1.91v-5.84z' })
    ])
  }
}

const TerminalIcon = {
  render() {
    return h('svg', {
      fill: 'none',
      stroke: 'currentColor',
      viewBox: '0 0 24 24',
      'stroke-width': '1.5',
      class: 'h-4 w-4'
    }, [
      h('path', {
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        d: 'm6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 17.25V6.75A2.25 2.25 0 0 0 18.75 4.5H5.25A2.25 2.25 0 0 0 3 6.75v10.5A2.25 2.25 0 0 0 5.25 20.25Z'
      })
    ])
  }
}

const clientTabs = computed((): TabConfig[] => {
  switch (props.platform) {
    case 'openai': {
      const tabs: TabConfig[] = [
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'codex-ws', label: t('keys.useKeyModal.cliTabs.codexCliWs'), icon: TerminalIcon },
      ]
      if (allowMessagesDispatch.value) {
        tabs.push({ id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon })
      }
      tabs.push({ id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon })
      return tabs
    }
    default:
      return [
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
  }
})

watch(
  clientTabs,
  (tabs) => {
    if (tabs.length === 0) return
    if (!tabs.some((tab) => tab.id === activeClientTab.value)) {
      activeClientTab.value = tabs[0].id
    }
  },
  { immediate: true }
)

watch(activeClientTab, () => {
  activeTab.value = 'unix'
})

const shellTabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: AppleIcon },
  { id: 'cmd', label: 'Windows CMD', icon: WindowsIcon },
  { id: 'powershell', label: 'PowerShell', icon: WindowsIcon }
]

const openAITabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: AppleIcon },
  { id: 'windows', label: 'Windows', icon: WindowsIcon }
]

const showShellTabs = computed(() => activeClientTab.value !== 'opencode')

const currentTabs = computed(() => {
  if (!showShellTabs.value) return []
  if (activeClientTab.value === 'codex' || activeClientTab.value === 'codex-ws') {
    return openAITabs
  }
  return shellTabs
})

const platformLabel = computed(() => t(`integrationGuide.platforms.${props.platform}`))

const platformDescription = computed(() => {
  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.description')
      }
      return t('keys.useKeyModal.openai.description')
    default:
      return t('keys.useKeyModal.description')
  }
})

const platformNote = computed(() => {
  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.note')
      }
      return activeTab.value === 'windows'
        ? t('keys.useKeyModal.openai.noteWindows')
        : t('keys.useKeyModal.openai.note')
    default:
      return t('keys.useKeyModal.note')
  }
})

const showPlatformNote = computed(() => activeClientTab.value !== 'opencode')

const currentFiles = computed((): FileConfig[] => {
  const baseUrl = props.baseUrl || window.location.origin
  const apiKey = effectiveApiKey.value
  const baseRoot = baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  const ensureV1 = (value: string) => {
    const trimmed = value.replace(/\/+$/, '')
    return trimmed.endsWith('/v1') ? trimmed : `${trimmed}/v1`
  }
  const apiBase = ensureV1(baseRoot)

  if (activeClientTab.value === 'opencode') {
    switch (props.platform) {
      case 'anthropic':
        return [generateOpenCodeConfig('anthropic', apiBase, apiKey)]
      case 'openai':
        return [generateOpenCodeConfig('openai', apiBase, apiKey)]
      default:
        return [generateOpenCodeConfig('anthropic', apiBase, apiKey)]
    }
  }

  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return generateAnthropicFiles(baseUrl, apiKey)
      }
      if (activeClientTab.value === 'codex-ws') {
        return generateOpenAIWsFiles(baseUrl, apiKey)
      }
      return generateOpenAIFiles(baseUrl, apiKey)
    default:
      return generateAnthropicFiles(baseUrl, apiKey)
  }
})

function generateAnthropicFiles(baseUrl: string, apiKey: string): FileConfig[] {
  let path: string
  let content: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = `export ANTHROPIC_BASE_URL="${baseUrl}"
export ANTHROPIC_AUTH_TOKEN="${apiKey}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
      break
    case 'cmd':
      path = 'Command Prompt'
      content = `set ANTHROPIC_BASE_URL=${baseUrl}
set ANTHROPIC_AUTH_TOKEN=${apiKey}
set CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
      break
    case 'powershell':
      path = 'PowerShell'
      content = `$env:ANTHROPIC_BASE_URL="${baseUrl}"
$env:ANTHROPIC_AUTH_TOKEN="${apiKey}"
$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
      break
    default:
      path = 'Terminal'
      content = ''
  }

  const vscodeSettingsPath = activeTab.value === 'unix'
    ? '~/.claude/settings.json'
    : '%userprofile%\\.claude\\settings.json'

  const vscodeContent = `{
  "env": {
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "ANTHROPIC_AUTH_TOKEN": "${apiKey}",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0"
  }
}`

  return [
    { path, content },
    { path: vscodeSettingsPath, content: vscodeContent, hint: 'VSCode Claude Code' }
  ]
}

function generateOpenAIFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'

  const configContent = `model_provider = "OpenAI"
model = "gpt-5.4"
review_model = "gpt-5.4"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true
model_context_window = 1000000
model_auto_compact_token_limit = 900000

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${baseUrl}"
wire_api = "responses"
requires_openai_auth = true`

  const authContent = `{
  "OPENAI_API_KEY": "${apiKey}"
}`

  return [
    {
      path: `${configDir}/config.toml`,
      content: configContent,
      hint: t('keys.useKeyModal.openai.configTomlHint')
    },
    {
      path: `${configDir}/auth.json`,
      content: authContent
    }
  ]
}

function generateOpenAIWsFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'

  const configContent = `model_provider = "OpenAI"
model = "gpt-5.4"
review_model = "gpt-5.4"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true
model_context_window = 1000000
model_auto_compact_token_limit = 900000

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${baseUrl}"
wire_api = "responses"
supports_websockets = true
requires_openai_auth = true

[features]
responses_websockets_v2 = true`

  const authContent = `{
  "OPENAI_API_KEY": "${apiKey}"
}`

  return [
    {
      path: `${configDir}/config.toml`,
      content: configContent,
      hint: t('keys.useKeyModal.openai.configTomlHint')
    },
    {
      path: `${configDir}/auth.json`,
      content: authContent
    }
  ]
}

function generateOpenCodeConfig(platform: string, baseUrl: string, apiKey: string, pathLabel?: string): FileConfig {
  const provider: Record<string, any> = {
    [platform]: {
      options: {
        baseURL: baseUrl,
        apiKey
      }
    }
  }

  const openaiModels = {
    'gpt-5-codex': { name: 'GPT-5 Codex', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {} } },
    'gpt-5.1-codex': { name: 'GPT-5.1 Codex', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {} } },
    'gpt-5.1-codex-max': { name: 'GPT-5.1 Codex Max', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {} } },
    'gpt-5.1-codex-mini': { name: 'GPT-5.1 Codex Mini', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {} } },
    'gpt-5.2': { name: 'GPT-5.2', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {}, xhigh: {} } },
    'gpt-5.4': { name: 'GPT-5.4', limit: { context: 1050000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {}, xhigh: {} } },
    'gpt-5.4-mini': { name: 'GPT-5.4 Mini', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {}, xhigh: {} } },
    'gpt-5.4-nano': { name: 'GPT-5.4 Nano', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {}, xhigh: {} } },
    'gpt-5.3-codex-spark': { name: 'GPT-5.3 Codex Spark', limit: { context: 128000, output: 32000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {}, xhigh: {} } },
    'gpt-5.3-codex': { name: 'GPT-5.3 Codex', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {}, xhigh: {} } },
    'gpt-5.2-codex': { name: 'GPT-5.2 Codex', limit: { context: 400000, output: 128000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {}, xhigh: {} } },
    'codex-mini-latest': { name: 'Codex Mini', limit: { context: 200000, output: 100000 }, options: { store: false }, variants: { low: {}, medium: {}, high: {} } }
  }

  if (platform === 'anthropic') {
    provider[platform].npm = '@ai-sdk/anthropic'
  } else if (platform === 'openai') {
    provider[platform].models = openaiModels
  }

  const agent =
    platform === 'openai'
      ? {
          build: {
            options: {
              store: false
            }
          },
          plan: {
            options: {
              store: false
            }
          }
        }
      : undefined

  const content = JSON.stringify(
    {
      provider,
      ...(agent ? { agent } : {}),
      $schema: 'https://opencode.ai/config.json'
    },
    null,
    2
  )

  return {
    path: pathLabel ?? 'opencode.json',
    content,
    hint: t('keys.useKeyModal.opencode.hint')
  }
}

function maskKey(key: string): string {
  if (key.length <= 12) return key
  return `${key.slice(0, 8)}...${key.slice(-4)}`
}

function formatKeyOption(apiKeyItem: ApiKey): string {
  const parts = [apiKeyItem.name, maskKey(apiKeyItem.key)]
  if (apiKeyItem.status !== 'active') {
    parts.push(t(`keys.status.${apiKeyItem.status}`))
  }
  return parts.join(' · ')
}

async function copyContent(content: string, index: number) {
  const success = await clipboardCopy(content, t('keys.copied'))
  if (success) {
    copiedIndex.value = index
    window.setTimeout(() => {
      if (copiedIndex.value === index) {
        copiedIndex.value = null
      }
    }, 1800)
  }
}
</script>
