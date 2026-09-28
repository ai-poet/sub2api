<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('auth.desktopBridge.title') }}
        </h2>
        <p v-if="statusMessage" class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ statusMessage }}
        </p>
      </div>

      <transition name="fade">
        <div
          v-if="errorMessage"
          class="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
        >
          <p class="text-sm text-red-700 dark:text-red-400">
            {{ errorMessage }}
          </p>
        </div>
      </transition>

      <div
        v-if="desktopCode"
        data-testid="desktop-login-code"
        class="rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-300"
      >
        <p v-if="codeExpired" class="text-red-700 dark:text-red-400">
          {{ t('auth.desktopBridge.code.expired') }}
        </p>
        <template v-else>
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
            {{ t('auth.desktopBridge.code.label') }}
          </p>
          <div class="mt-2 flex flex-wrap items-center justify-between gap-3">
            <span
              data-testid="desktop-login-code-value"
              class="select-all font-mono text-3xl font-bold tracking-[0.2em] text-gray-900 dark:text-white"
            >
              {{ desktopCode.code }}
            </span>
            <button
              type="button"
              class="inline-flex rounded-lg border border-gray-300 bg-white px-3 py-1.5 font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700"
              @click="copyCode"
            >
              {{ copied ? t('auth.desktopBridge.code.copied') : t('auth.desktopBridge.code.copy') }}
            </button>
          </div>
          <p class="mt-3">{{ t('auth.desktopBridge.code.hint') }}</p>
          <p
            data-testid="desktop-login-code-warning"
            class="mt-2 text-xs text-amber-700 dark:text-amber-400"
          >
            {{ t('auth.desktopBridge.code.warning') }}
          </p>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            {{ t('auth.desktopBridge.code.expiresIn', { time: countdown }) }}
          </p>
          <a
            v-if="codeCallbackUrl"
            :href="codeCallbackUrl"
            class="mt-3 inline-flex rounded-lg bg-gray-900 px-4 py-2 font-medium text-white dark:bg-white dark:text-gray-900"
          >
            {{ t('auth.desktopBridge.code.returnToApp') }}
          </a>
        </template>
      </div>

      <div
        v-if="callbackUrl"
        class="rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-300"
      >
        <p>{{ t('auth.desktopBridge.manualHint') }}</p>
        <a
          :href="callbackUrl"
          class="mt-3 inline-flex rounded-lg bg-gray-900 px-4 py-2 font-medium text-white dark:bg-white dark:text-gray-900"
        >
          {{ t('auth.desktopBridge.openApp') }}
        </a>
      </div>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import { keysAPI, userGroupsAPI } from '@/api'
import { authAPI, getAuthToken } from '@/api/auth'
import { clearStoredOAuthReturnPath, rememberOAuthReturnPath } from '@/utils/auth-redirect'
import { useClipboard } from '@/composables/useClipboard'
import {
  buildDesktopCodeCallbackUrl,
  buildPaseoCallbackUrl,
  desktopLoginEntryFromGrant,
  formatCountdown,
  isLoopbackCallbackUrl,
  loadDesktopLoginEntry,
  normalizePaseoEndpoint,
  payloadFromDesktopSession,
  readDesktopCodeRequest,
  resolveCallbackTarget,
  safeSessionStorage,
  saveDesktopLoginEntry,
  type DesktopLoginEntry
} from './paseo-bridge'
import type { ApiKey, Group, GroupPlatform } from '@/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

// The stage, not its text, so the message follows a language switch. Null
// while a sign-in code is on screen: the code card says what to do.
type BridgeStage =
  | 'preparing'
  | 'preparingRoutes'
  | 'creatingSession'
  | 'creatingCode'
  | 'opening'
  | 'failedStatus'
const stage = ref<BridgeStage | null>('preparing')
const statusMessage = computed(() => (stage.value ? t(`auth.desktopBridge.${stage.value}`) : ''))
const errorMessage = ref('')
const callbackUrl = ref('')

// Code mode: the one-time sign-in code on screen, and where it is handed over.
const desktopCode = ref<DesktopLoginEntry | null>(null)
const codeCallbackUrl = ref('')
const nowMs = ref(Date.now())
const remainingMs = computed(() =>
  desktopCode.value ? Math.max(desktopCode.value.expiresAt - nowMs.value, 0) : 0
)
const codeExpired = computed(() => desktopCode.value !== null && remainingMs.value <= 0)
const countdown = computed(() => formatCountdown(remainingMs.value))
const { copied, copyToClipboard } = useClipboard()

/** Long enough to see the code before the page gives way to the app's listener. */
const CODE_OPEN_APP_DELAY_MS = 1500
let countdownTimer: number | undefined
let openAppTimer: number | undefined

const endpoint = computed(() => {
  const routeEndpoint = typeof route.query.endpoint === 'string' ? route.query.endpoint : ''
  const fallbackOrigin = typeof window !== 'undefined' ? window.location.origin : ''
  return normalizePaseoEndpoint(routeEndpoint || fallbackOrigin)
})

// Where to deliver the session: an allow-listed `redirect_to` (the native
// desktop's loopback listener), or the app URL scheme pair.
const callbackTarget = computed(() =>
  resolveCallbackTarget(typeof route.query.redirect_to === 'string' ? route.query.redirect_to : '')
)

type PaseoScopedPlatform = Extract<GroupPlatform, 'anthropic' | 'openai'>

interface PaseoPreparedKeys {
  apiKey: string
  claudeApiKey: string | null
  codexApiKey: string | null
}

function isRouteReadyApiKey(apiKey: ApiKey, platform: PaseoScopedPlatform): boolean {
  return (
    apiKey.status === 'active' &&
    apiKey.group?.status === 'active' &&
    apiKey.group.platform === platform
  )
}

function pickExistingRouteKey(apiKeys: ApiKey[], platform: PaseoScopedPlatform): ApiKey | null {
  return apiKeys.find((apiKey) => isRouteReadyApiKey(apiKey, platform)) ?? null
}

function pickCreatableGroup(groups: Group[], platform: PaseoScopedPlatform): Group | null {
  return groups.find((group) => group.status === 'active' && group.platform === platform) ?? null
}

async function loadAllApiKeys(): Promise<ApiKey[]> {
  const pageSize = 100
  let page = 1
  let pages = 1
  const items: ApiKey[] = []

  while (page <= pages) {
    const response = await keysAPI.list(page, pageSize)
    items.push(...response.items)
    pages = response.pages || 1
    page += 1
  }

  return items
}

function pickPrimaryApiKey(
  apiKeys: ApiKey[],
  scoped: { claude: ApiKey | null; codex: ApiKey | null }
): string | null {
  const fallback =
    scoped.claude?.key?.trim() ||
    scoped.codex?.key?.trim() ||
    apiKeys.find((apiKey) => apiKey.status === 'active')?.key?.trim() ||
    apiKeys[0]?.key?.trim() ||
    null
  return fallback && fallback.length > 0 ? fallback : null
}

async function ensurePaseoKeys(): Promise<PaseoPreparedKeys> {
  const apiKeys = await loadAllApiKeys()
  let claudeKey = pickExistingRouteKey(apiKeys, 'anthropic')
  let codexKey = pickExistingRouteKey(apiKeys, 'openai')

  if (!claudeKey || !codexKey) {
    const availableGroups = await userGroupsAPI.getAvailable()

    if (!claudeKey) {
      const group = pickCreatableGroup(availableGroups, 'anthropic')
      if (group) {
        claudeKey = await keysAPI.create('Desktop App (Claude Code)', group.id)
      }
    }

    if (!codexKey) {
      const group = pickCreatableGroup(availableGroups, 'openai')
      if (group) {
        codexKey = await keysAPI.create('Desktop App (Codex)', group.id)
      }
    }
  }

  let primaryApiKey = pickPrimaryApiKey(apiKeys, {
    claude: claudeKey,
    codex: codexKey
  })
  if (!primaryApiKey) {
    const created = await keysAPI.create('Desktop App')
    primaryApiKey = created.key.trim()
  }

  return {
    apiKey: primaryApiKey,
    claudeApiKey: claudeKey?.key?.trim() || null,
    codexApiKey: codexKey?.key?.trim() || null
  }
}

function stopCountdown(): void {
  if (countdownTimer !== undefined) {
    window.clearInterval(countdownTimer)
    countdownTimer = undefined
  }
}

function cancelOpenApp(): void {
  if (openAppTimer !== undefined) {
    window.clearTimeout(openAppTimer)
    openAppTimer = undefined
  }
}

function tickCountdown(): void {
  nowMs.value = Date.now()
  if (codeExpired.value) {
    stopCountdown()
    cancelOpenApp()
  }
}

function showDesktopCode(entry: DesktopLoginEntry, loopbackBase: string | null): void {
  desktopCode.value = entry
  codeCallbackUrl.value = loopbackBase
    ? buildDesktopCodeCallbackUrl(loopbackBase, entry.code, endpoint.value)
    : ''
  stage.value = null
  stopCountdown()
  tickCountdown()
  if (!codeExpired.value) {
    countdownTimer = window.setInterval(tickCountdown, 1000)
  }
}

async function copyCode(): Promise<void> {
  if (desktopCode.value) {
    await copyToClipboard(desktopCode.value.code, t('auth.desktopBridge.code.copied'))
  }
}

// Back from a loopback page that failed to load can restore this page from
// the back/forward cache instead of mounting it again.
function onPageShow(event: PageTransitionEvent): void {
  if (event.persisted && desktopCode.value) {
    cancelOpenApp()
    stage.value = null
    tickCountdown()
  }
}

/**
 * The PKCE flow: a single-use code bound to the app's challenge instead of
 * tokens in a URL. It is always shown for pasting, and handed to the app's
 * listener only when that is loopback - code mode never guesses a URL scheme,
 * since a customer who cannot reach 127.0.0.1 is why it exists.
 */
async function runCodeMode(challenge: string): Promise<void> {
  const target = callbackTarget.value
  const loopbackBase = isLoopbackCallbackUrl(target.base) ? target.base : null
  const storage = safeSessionStorage()
  window.addEventListener('pageshow', onPageShow)

  // Already issued in this tab: the user came Back after the listener's page
  // failed to load. Show the same code; navigating again would fail again.
  const stored = loadDesktopLoginEntry(storage, challenge)
  if (stored) {
    clearStoredOAuthReturnPath()
    showDesktopCode(stored, loopbackBase)
    return
  }

  stage.value = 'preparingRoutes'
  const preparedKeys = await ensurePaseoKeys()

  stage.value = 'creatingCode'
  const grant = await authAPI.createDesktopLoginCode({
    code_challenge: challenge,
    code_challenge_method: 'S256',
    api_key: preparedKeys.apiKey,
    claude_api_key: preparedKeys.claudeApiKey,
    codex_api_key: preparedKeys.codexApiKey
  })
  const entry = desktopLoginEntryFromGrant(grant)
  saveDesktopLoginEntry(storage, challenge, entry)
  clearStoredOAuthReturnPath()
  showDesktopCode(entry, loopbackBase)

  const url = codeCallbackUrl.value
  if (!url) {
    return
  }
  openAppTimer = window.setTimeout(() => {
    openAppTimer = undefined
    if (codeExpired.value) {
      return
    }
    saveDesktopLoginEntry(storage, challenge, { ...entry, redirected: true })
    stage.value = 'opening'
    window.location.href = url
  }, CODE_OPEN_APP_DELAY_MS)
}

onMounted(async () => {
  try {
    const accessToken = getAuthToken()
    if (!accessToken) {
      rememberOAuthReturnPath(route.fullPath)
      await router.replace({
        path: '/login',
        query: { redirect: route.fullPath },
      })
      return
    }

    const codeRequest = readDesktopCodeRequest(route.query)
    if (codeRequest) {
      await runCodeMode(codeRequest.challenge)
      return
    }

    stage.value = 'preparingRoutes'
    const preparedKeys = await ensurePaseoKeys()

    // A session of the desktop's own. Handing over this tab's tokens left two
    // holders of one refresh token, and the service's rotation then signed
    // out whichever of them renewed second - the desktop, usually.
    stage.value = 'creatingSession'
    const session = await authAPI.createDesktopSession()
    const payload = payloadFromDesktopSession(session, preparedKeys, endpoint.value)
    const target = callbackTarget.value
    callbackUrl.value = buildPaseoCallbackUrl(payload, { callbackBase: target.base })

    stage.value = 'opening'
    clearStoredOAuthReturnPath()
    window.location.href = callbackUrl.value

    // A URL scheme nobody registered navigates nowhere and reports nothing,
    // so when the primary is a guess, retry with the previous generation's
    // scheme after a beat. If the current app took over, this page loses
    // visibility and the retry is cancelled — at worst an install with BOTH
    // generations sees the old app receive a login it also understands.
    if (target.legacyFallback) {
      const legacyUrl = buildPaseoCallbackUrl(payload, { callbackBase: target.legacyFallback })
      const timer = window.setTimeout(() => {
        if (document.visibilityState === 'visible' && document.hasFocus()) {
          callbackUrl.value = legacyUrl
          window.location.href = legacyUrl
        }
      }, 1800)
      document.addEventListener(
        'visibilitychange',
        () => {
          if (document.visibilityState !== 'visible') {
            window.clearTimeout(timer)
          }
        },
        { once: true }
      )
    }
  } catch (error: unknown) {
    errorMessage.value = error instanceof Error ? error.message : t('auth.desktopBridge.failed')
    stage.value = 'failedStatus'
  }
})

onBeforeUnmount(() => {
  stopCountdown()
  cancelOpenApp()
  window.removeEventListener('pageshow', onPageShow)
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
