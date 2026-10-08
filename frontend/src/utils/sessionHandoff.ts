/**
 * 多域名登录交接（fork 本地功能）。
 *
 * 同一套服务挂在登录主域名（第三方 OAuth 回调登记在这里）和别名域名（例如 CDN 加速域名）上。
 * OAuth 的 state cookie 与登录 token 都只属于各自的域名，所以：
 *  - 别名域名上点第三方登录 / 绑定时，转到登录主域名去做；
 *  - 登录完成后，登录主域名用一次性交接码把会话交还给别名域名，别名域名自己兑换一套 token。
 *
 * 角色：接收方（要拿到会话的域名）生成 PKCE verifier 并留在本域 sessionStorage，只把 challenge
 * 交给提供方；提供方（/auth/handoff）用当前会话申请交接码，再把浏览器带到接收方的
 * /auth/handoff/complete。密码登录不经过这里，留在当前域名完成。
 */

import type { OAuthLoginStart } from '@/api/auth'
import type { PublicSettings } from '@/types'
import { sanitizeOAuthFrontendRedirect } from '@/utils/oauth-redirect-sanitize'

export const SESSION_HANDOFF_GIVE_PATH = '/auth/handoff'
export const SESSION_HANDOFF_COMPLETE_PATH = '/auth/handoff/complete'
export const SESSION_HANDOFF_PULL_PATH = '/auth/handoff/pull'

const RECEIVE_STATE_KEY = 'sub2api_session_handoff_receive'
const GIVE_STATE_KEY = 'sub2api_session_handoff_give'
/** 一轮交接（含第三方授权页停留）的最长时间。 */
const STATE_MAX_AGE_MS = 15 * 60 * 1000
const CHALLENGE_RE = /^[A-Za-z0-9_-]{43}$/
/** 随第三方登录一起转交的参数：其余（如 intent）一律丢弃。 */
const FORWARDED_OAUTH_PARAMS = ['aff_code', 'mode'] as const
const OAUTH_PROVIDERS = ['github', 'google', 'linuxdo', 'dingtalk', 'wechat', 'oidc'] as const

export interface SessionHandoffConfig {
  loginOrigin: string
  aliasOrigins: string[]
}

/** 接收方本地保存的一轮交接。 */
export interface SessionHandoffReceiveState {
  verifier: string
  from: string
  redirect: string
  t: number
}

/** 提供方本地保存的一轮交接（跨第三方登录往返时用来恢复）。 */
export interface SessionHandoffGiveState {
  origin: string
  challenge: string
  redirect: string
  provider?: OAuthLoginStart['provider']
  params?: Record<string, string>
  /** 已经替用户发起过一次第三方登录：回来时仍未登录就不再自动发起，避免循环。 */
  oauthStarted?: boolean
  t: number
}

/** 规范成 location.origin 的形式；不是 http(s) 源时返回空串。 */
export function normalizeOrigin(raw: unknown): string {
  if (typeof raw !== 'string' || raw.trim() === '') return ''
  try {
    const url = new URL(raw.trim())
    if (url.protocol !== 'https:' && url.protocol !== 'http:') return ''
    if ((url.pathname !== '/' && url.pathname !== '') || url.search || url.hash || url.username || url.password) return ''
    return url.origin
  } catch {
    return ''
  }
}

export function readSessionHandoffConfig(
  settings: Partial<PublicSettings> | null | undefined
): SessionHandoffConfig | null {
  const loginOrigin = normalizeOrigin(settings?.session_handoff_login_origin)
  const aliasOrigins = (settings?.session_handoff_alias_origins ?? [])
    .map(normalizeOrigin)
    .filter((origin) => origin !== '' && origin !== loginOrigin)
  if (!loginOrigin || aliasOrigins.length === 0) return null
  return { loginOrigin, aliasOrigins }
}

export function isHandoffOrigin(config: SessionHandoffConfig, origin: string): boolean {
  return origin === config.loginOrigin || config.aliasOrigins.includes(origin)
}

/** 当前页面是否在别名域名上（需要把第三方登录转到登录主域名）。 */
export function isOnAliasOrigin(config: SessionHandoffConfig | null, origin = currentOrigin()): boolean {
  return config !== null && origin !== config.loginOrigin && config.aliasOrigins.includes(origin)
}

export function isHandoffChallenge(value: unknown): value is string {
  return typeof value === 'string' && CHALLENGE_RE.test(value)
}

function currentOrigin(): string {
  return typeof window === 'undefined' ? '' : window.location.origin
}

function base64Url(bytes: Uint8Array): string {
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

/** 生成一对 PKCE：verifier 是 32 字节随机数的 base64url，challenge 是其 SHA-256 的 base64url。 */
export async function createHandoffPkcePair(): Promise<{ verifier: string; challenge: string }> {
  const random = new Uint8Array(32)
  crypto.getRandomValues(random)
  const verifier = base64Url(random)
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier))
  return { verifier, challenge: base64Url(new Uint8Array(digest)) }
}

function writeState(key: string, value: unknown): void {
  try {
    sessionStorage.setItem(key, JSON.stringify(value))
  } catch {
    // 隐私模式 / 配额：交接无法跨页面恢复，最多回退到普通登录
  }
}

function readState<T extends { t: number }>(key: string): T | null {
  try {
    const raw = sessionStorage.getItem(key)
    if (!raw) return null
    const parsed = JSON.parse(raw) as T
    if (typeof parsed?.t !== 'number' || Date.now() - parsed.t > STATE_MAX_AGE_MS) {
      sessionStorage.removeItem(key)
      return null
    }
    return parsed
  } catch {
    return null
  }
}

function clearState(key: string): void {
  try {
    sessionStorage.removeItem(key)
  } catch {
    // ignore
  }
}

export function readReceiveState(): SessionHandoffReceiveState | null {
  const state = readState<SessionHandoffReceiveState>(RECEIVE_STATE_KEY)
  return state && typeof state.verifier === 'string' && state.verifier.length >= 43 ? state : null
}

export function clearReceiveState(): void {
  clearState(RECEIVE_STATE_KEY)
}

export function saveGiveState(state: Omit<SessionHandoffGiveState, 't'> & { t?: number }): void {
  writeState(GIVE_STATE_KEY, { ...state, t: state.t ?? Date.now() })
}

export function readGiveState(): SessionHandoffGiveState | null {
  const state = readState<SessionHandoffGiveState>(GIVE_STATE_KEY)
  return state && typeof state.origin === 'string' && isHandoffChallenge(state.challenge) ? state : null
}

export function clearGiveState(): void {
  clearState(GIVE_STATE_KEY)
}

/** 提供方路由守卫的兜底：登录完成后若还有一轮没交出去的会话，回到 /auth/handoff。 */
export function hasPendingGive(): boolean {
  return readGiveState() !== null
}

export function sanitizeHandoffRedirect(value: unknown): string {
  return typeof value === 'string' ? sanitizeOAuthFrontendRedirect(value, '') : ''
}

function isOAuthProvider(value: unknown): value is OAuthLoginStart['provider'] {
  return typeof value === 'string' && (OAUTH_PROVIDERS as readonly string[]).includes(value)
}

/** 只保留允许转交的第三方登录参数。 */
export function pickForwardedOAuthParams(params: Record<string, unknown> | undefined): Record<string, string> {
  const out: Record<string, string> = {}
  for (const key of FORWARDED_OAUTH_PARAMS) {
    const value = params?.[key]
    if (typeof value === 'string' && value !== '' && value.length <= 128) out[key] = value
  }
  return out
}

/** 从 /auth/handoff 的查询参数里读出一轮交接请求；不合法时返回 null。 */
export function parseGiveQuery(
  query: Record<string, unknown>,
  config: SessionHandoffConfig,
  selfOrigin = currentOrigin()
): Omit<SessionHandoffGiveState, 't'> | null {
  const first = (value: unknown) => (Array.isArray(value) ? value[0] : value)
  const origin = normalizeOrigin(first(query.origin))
  const challenge = first(query.challenge)
  if (!origin || origin === selfOrigin || !isHandoffOrigin(config, origin) || !isHandoffChallenge(challenge)) {
    return null
  }
  const provider = first(query.provider)
  const params: Record<string, unknown> = {}
  for (const key of FORWARDED_OAUTH_PARAMS) params[key] = first(query[key])
  return {
    origin,
    challenge,
    redirect: sanitizeHandoffRedirect(first(query.redirect)),
    provider: isOAuthProvider(provider) ? provider : undefined,
    params: pickForwardedOAuthParams(params),
  }
}

/**
 * 作为接收方开始一轮交接：本地留下 verifier，把浏览器带到提供方的 /auth/handoff。
 * provider 存在时，提供方未登录会直接替用户发起该第三方登录。
 */
export async function startReceive(options: {
  from: string
  redirect?: string
  provider?: OAuthLoginStart['provider']
  params?: Record<string, string>
}): Promise<void> {
  const { verifier, challenge } = await createHandoffPkcePair()
  const redirect = sanitizeHandoffRedirect(options.redirect)
  writeState(RECEIVE_STATE_KEY, { verifier, from: options.from, redirect, t: Date.now() })
  const query = new URLSearchParams({ origin: currentOrigin(), challenge })
  if (redirect) query.set('redirect', redirect)
  if (options.provider) query.set('provider', options.provider)
  for (const [key, value] of Object.entries(pickForwardedOAuthParams(options.params))) query.set(key, value)
  window.location.assign(`${options.from}${SESSION_HANDOFF_GIVE_PATH}?${query.toString()}`)
}

/** 提供方拿到交接码后要去的接收方地址；码放在片段里，不进任何服务器日志。 */
export function buildCompleteURL(origin: string, code: string, redirect: string): string {
  const fragment = new URLSearchParams({ code })
  if (redirect) fragment.set('redirect', redirect)
  return `${origin}${SESSION_HANDOFF_COMPLETE_PATH}#${fragment.toString()}`
}

/**
 * 登录 / 注册页的第三方登录入口：在别名域名上时转到登录主域名发起并返回 true；
 * 否则返回 false，由调用方照常在本域发起。
 */
export async function startOAuthOnLoginOrigin(
  settings: Partial<PublicSettings> | null | undefined,
  request: OAuthLoginStart
): Promise<boolean> {
  const config = readSessionHandoffConfig(settings)
  if (!config || !isOnAliasOrigin(config)) return false
  await startReceive({
    from: config.loginOrigin,
    redirect: request.params.redirect,
    provider: request.provider,
    params: request.params,
  })
  return true
}

/**
 * 资料页的第三方账号绑定入口：在别名域名上时，先把当前会话交给登录主域名（由登录主域名
 * 作为接收方发起），到那边的资料页再绑定，返回 true；否则返回 false。
 */
export function startBindingOnLoginOrigin(
  settings: Partial<PublicSettings> | null | undefined,
  redirect: string
): boolean {
  const config = readSessionHandoffConfig(settings)
  if (!config || !isOnAliasOrigin(config)) return false
  const query = new URLSearchParams({ from: currentOrigin() })
  const safeRedirect = sanitizeHandoffRedirect(redirect)
  if (safeRedirect) query.set('redirect', safeRedirect)
  window.location.assign(`${config.loginOrigin}${SESSION_HANDOFF_PULL_PATH}?${query.toString()}`)
  return true
}
