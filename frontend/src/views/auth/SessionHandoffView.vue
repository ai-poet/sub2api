<template>
  <AuthLayout>
    <div class="space-y-6" data-test="session-handoff">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('auth.sessionHandoff.pageTitle') }}
        </h2>
        <p v-if="!errorMessage" class="mt-2 text-sm text-gray-500 dark:text-dark-400" data-test="session-handoff-status">
          {{ statusText }}
        </p>
      </div>

      <div v-if="!errorMessage" class="flex justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <div
        v-else
        class="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
        data-test="session-handoff-error"
      >
        <div class="flex items-start gap-3">
          <Icon name="exclamationCircle" size="md" class="flex-shrink-0 text-red-500" />
          <div class="space-y-3">
            <p class="text-sm text-red-700 dark:text-red-400">{{ errorMessage }}</p>
            <div class="flex flex-wrap gap-2">
              <router-link to="/login" class="btn btn-primary">{{ t('auth.sessionHandoff.backToLogin') }}</router-link>
              <router-link to="/" class="btn btn-secondary">{{ t('auth.sessionHandoff.backHome') }}</router-link>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
// 多域名登录交接（fork 本地功能）：一个页面三种模式，按路由 meta.handoffMode 区分。
//  give     /auth/handoff           提供方：已登录就签交接码送回接收方；未登录先登录（必要时替用户发起第三方登录）
//  complete /auth/handoff/complete  接收方：拿片段里的码 + 本地 verifier 兑换 token
//  pull     /auth/handoff/pull      接收方：由别的域名请过来，本域生成 verifier 后去对方那里取会话
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@/components/layout/AuthLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildOAuthLoginStartURL, persistOAuthTokenContext } from '@/api/auth'
import { createSessionHandoffCode, exchangeSessionHandoffCode } from '@/api/sessionHandoff'
import { useAppStore, useAuthStore } from '@/stores'
import { extractApiErrorCode } from '@/utils/apiError'
import { storeOAuthAffiliateCode } from '@/utils/oauthAffiliate'
import {
  SESSION_HANDOFF_GIVE_PATH,
  buildCompleteURL,
  clearGiveState,
  clearReceiveState,
  isHandoffOrigin,
  normalizeOrigin,
  parseGiveQuery,
  readGiveState,
  readReceiveState,
  readSessionHandoffConfig,
  sanitizeHandoffRedirect,
  saveGiveState,
  startReceive,
  type SessionHandoffConfig,
  type SessionHandoffGiveState,
} from '@/utils/sessionHandoff'

type HandoffMode = 'give' | 'complete' | 'pull'

/** 与 EmailOAuthButtons / OAuthCallbackView 约定的键：回调页据此区分 GitHub / Google。 */
const EMAIL_OAUTH_PENDING_PROVIDER_KEY = 'email_oauth_pending_provider'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const mode = computed<HandoffMode>(() => (route.meta.handoffMode as HandoffMode) ?? 'give')
const errorMessage = ref('')
const targetOrigin = ref('')
const redirectingToLogin = ref(false)

const statusText = computed(() => {
  if (mode.value === 'complete') return t('auth.sessionHandoff.receiving')
  if (redirectingToLogin.value) return t('auth.sessionHandoff.redirecting')
  return t('auth.sessionHandoff.giving', { origin: targetOrigin.value || '…' })
})

function fail(key: string): void {
  errorMessage.value = t(`auth.sessionHandoff.${key}`)
}

async function loadConfig(): Promise<SessionHandoffConfig | null> {
  if (!appStore.publicSettingsLoaded) {
    try {
      await appStore.fetchPublicSettings()
    } catch {
      // 读不到设置就当未配置
    }
  }
  return readSessionHandoffConfig(appStore.cachedPublicSettings)
}

function actionCaptchaEnabled(): boolean {
  const s = appStore.cachedPublicSettings
  return Boolean(
    (s?.tencent_captcha_enabled && s.tencent_captcha_app_id) ||
      (s?.aliyun_captcha_enabled && s.aliyun_captcha_scene_id && s.aliyun_captcha_prefix)
  )
}

async function give(config: SessionHandoffConfig): Promise<void> {
  const fromQuery = parseGiveQuery(route.query as Record<string, unknown>, config)
  const state: SessionHandoffGiveState | null = fromQuery ? { ...fromQuery, t: Date.now() } : readGiveState()
  if (!state) {
    fail(route.query.origin ? 'originNotAllowed' : 'invalidRequest')
    return
  }
  if (state.origin === window.location.origin || !isHandoffOrigin(config, state.origin)) {
    clearGiveState()
    fail('originNotAllowed')
    return
  }
  targetOrigin.value = state.origin
  saveGiveState(state)
  if (fromQuery) {
    // 去掉地址栏里的 challenge，刷新时从 sessionStorage 恢复
    await router.replace({ path: SESSION_HANDOFF_GIVE_PATH })
  }

  if (authStore.isAuthenticated) {
    try {
      const issued = await createSessionHandoffCode({
        code_challenge: state.challenge,
        code_challenge_method: 'S256',
        target_origin: state.origin,
      })
      clearGiveState()
      window.location.replace(buildCompleteURL(issued.target_origin, issued.code, state.redirect))
    } catch (error) {
      clearGiveState()
      fail(extractApiErrorCode(error) === 'SESSION_HANDOFF_ORIGIN_NOT_ALLOWED' ? 'originNotAllowed' : 'failed')
    }
    return
  }

  // 未登录：先在本域登录，登录完成后回到这里（路由守卫也会兜底拉回来）。
  redirectingToLogin.value = true
  const params = state.params ?? {}
  const provider = state.provider
  if (provider && !state.oauthStarted && !actionCaptchaEnabled()) {
    saveGiveState({ ...state, oauthStarted: true })
    if (params.aff_code) storeOAuthAffiliateCode(params.aff_code)
    if (provider === 'github' || provider === 'google') {
      window.sessionStorage.setItem(EMAIL_OAUTH_PENDING_PROVIDER_KEY, provider)
    }
    window.location.href = buildOAuthLoginStartURL({
      provider,
      params: { ...params, redirect: SESSION_HANDOFF_GIVE_PATH },
    })
    return
  }
  const query: Record<string, string> = { redirect: SESSION_HANDOFF_GIVE_PATH }
  if (params.aff_code) query.aff_code = params.aff_code
  await router.replace({ path: '/login', query })
}

async function complete(): Promise<void> {
  const fragment = new URLSearchParams(window.location.hash.replace(/^#/, ''))
  const code = fragment.get('code') ?? ''
  const fragmentRedirect = sanitizeHandoffRedirect(fragment.get('redirect'))
  // 码只用一次：先从地址栏和历史记录里抹掉
  window.history.replaceState(window.history.state, '', window.location.pathname)

  const state = readReceiveState()
  if (!code || !state) {
    fail('expired')
    return
  }
  try {
    const tokens = await exchangeSessionHandoffCode(code, state.verifier)
    clearReceiveState()
    persistOAuthTokenContext({ refresh_token: tokens.refresh_token, expires_in: tokens.expires_in })
    await authStore.setToken(tokens.access_token)
    await router.replace(fragmentRedirect || state.redirect || authStore.homePath)
  } catch {
    clearReceiveState()
    fail('failed')
  }
}

async function pull(config: SessionHandoffConfig): Promise<void> {
  const first = (value: unknown) => (Array.isArray(value) ? value[0] : value)
  const from = normalizeOrigin(first(route.query.from))
  if (!from || from === window.location.origin || !isHandoffOrigin(config, from)) {
    fail('originNotAllowed')
    return
  }
  targetOrigin.value = window.location.origin
  await startReceive({ from, redirect: sanitizeHandoffRedirect(first(route.query.redirect)) })
}

onMounted(async () => {
  const config = await loadConfig()
  if (!config) {
    if (mode.value === 'give') clearGiveState()
    fail('disabled')
    return
  }
  if (mode.value === 'complete') {
    await complete()
  } else if (mode.value === 'pull') {
    await pull(config)
  } else {
    await give(config)
  }
})
</script>
