/**
 * 封禁申诉会话（fork 本地功能）：令牌的保存与识别。
 *
 * 令牌只放 sessionStorage（关掉标签页就没了），绝不放 localStorage 的 auth_token：
 * 路由守卫和 apiClient 会把那个键当成登录态。
 */

export const APPEAL_SESSION_KEY = 'sub2api_appeal_session'
export const APPEAL_LOGIN_NOTICE_KEY = 'sub2api_appeal_notice'

export type AppealLoginNotice = 'restored' | 'expired'

export interface AppealSessionState {
  token: string
  /** 毫秒时间戳 */
  expiresAt: number
}

function safeSession(): Storage | null {
  try {
    return typeof window !== 'undefined' ? window.sessionStorage : null
  } catch {
    return null
  }
}

export function storeAppealSession(token: string, expiresIn: number | string | undefined): AppealSessionState | null {
  const value = String(token || '').trim()
  if (!value.startsWith('apl_')) return null
  const seconds = Number(expiresIn)
  const ttl = Number.isFinite(seconds) && seconds > 0 ? seconds : 7200
  const state: AppealSessionState = { token: value, expiresAt: Date.now() + ttl * 1000 }
  safeSession()?.setItem(APPEAL_SESSION_KEY, JSON.stringify(state))
  return state
}

export function readAppealSession(): AppealSessionState | null {
  const storage = safeSession()
  const raw = storage?.getItem(APPEAL_SESSION_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Partial<AppealSessionState>
    if (typeof parsed.token !== 'string' || typeof parsed.expiresAt !== 'number' || parsed.expiresAt <= Date.now()) {
      storage?.removeItem(APPEAL_SESSION_KEY)
      return null
    }
    return { token: parsed.token, expiresAt: parsed.expiresAt }
  } catch {
    storage?.removeItem(APPEAL_SESSION_KEY)
    return null
  }
}

export function clearAppealSession() {
  safeSession()?.removeItem(APPEAL_SESSION_KEY)
}

/** 登录接口的 403 USER_NOT_ACTIVE 里带的申诉令牌（apiClient 已保留 reason / metadata）。 */
export function extractAppealToken(err: unknown): { token: string; expiresIn?: string } | null {
  if (!err || typeof err !== 'object') return null
  const e = err as { reason?: unknown; metadata?: Record<string, unknown> }
  if (e.reason !== 'USER_NOT_ACTIVE' || !e.metadata) return null
  const token = typeof e.metadata.appeal_token === 'string' ? e.metadata.appeal_token : ''
  if (!token.startsWith('apl_')) return null
  const expiresIn = e.metadata.expires_in
  return { token, expiresIn: typeof expiresIn === 'string' || typeof expiresIn === 'number' ? String(expiresIn) : undefined }
}

/** 拿到了申诉令牌就保存，返回 true 表示调用方应跳转 /appeal。 */
export function captureAppealToken(err: unknown): boolean {
  const found = extractAppealToken(err)
  if (!found) return false
  return storeAppealSession(found.token, found.expiresIn) !== null
}

/**
 * 第三方登录回调页：fragment 里有 appeal_token 时保存并清掉地址栏里的令牌。
 * params 是 fragment 解析出的 URLSearchParams；返回 true 表示调用方应跳转 /appeal。
 */
export function consumeAppealFragment(params: URLSearchParams): boolean {
  const token = params.get('appeal_token')
  if (!token) return false
  const stored = storeAppealSession(token, params.get('expires_in') ?? undefined)
  try {
    if (typeof window !== 'undefined' && window.location.hash) {
      window.history.replaceState(window.history.state, '', window.location.pathname + window.location.search)
    }
  } catch {
    // ignore
  }
  return stored !== null
}

export function setAppealLoginNotice(notice: AppealLoginNotice) {
  safeSession()?.setItem(APPEAL_LOGIN_NOTICE_KEY, notice)
}

export function consumeAppealLoginNotice(): AppealLoginNotice | null {
  const storage = safeSession()
  const value = storage?.getItem(APPEAL_LOGIN_NOTICE_KEY)
  storage?.removeItem(APPEAL_LOGIN_NOTICE_KEY)
  return value === 'restored' || value === 'expired' ? value : null
}
