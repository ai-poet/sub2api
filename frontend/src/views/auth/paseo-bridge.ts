export interface PaseoCallbackPayload {
  accessToken: string
  refreshToken: string
  expiresAt: number
  apiKey: string
  claudeApiKey?: string | null
  codexApiKey?: string | null
  endpoint: string
}

export function normalizePaseoEndpoint(endpoint: string): string {
  return endpoint.trim().replace(/\/+$/, '')
}

export function resolveExpiresInSeconds(expiresAt: number, now: number = Date.now()): number {
  const remainingMs = expiresAt - now
  return Math.max(Math.floor(remainingMs / 1000), 0)
}

/** Where the session gets delivered after the browser login completes. */
export interface CallbackTarget {
  /** URL the fragment is appended to and navigated first. */
  base: string
  /**
   * Second attempt for desktop builds that registered the old URL scheme.
   * Only present when the primary is itself a guessed scheme — an explicit
   * `redirect_to` never falls back.
   */
  legacyFallback?: string
}

/** The scheme current desktop builds register. */
export const CALLBACK_SCHEME = 'agentdesk://auth/callback'
/** The scheme the previous desktop generation registered. */
export const LEGACY_CALLBACK_SCHEME = 'paseo://auth/callback'

/**
 * Validate and resolve the client-requested delivery target.
 *
 * The fragment carries the whole session (tokens and API keys), so
 * `redirect_to` is strictly allow-listed — a crafted
 * `/auth/paseo?redirect_to=https://evil` link must never be able to walk away
 * with it. Allowed:
 *
 * - loopback HTTP (`http://127.0.0.1:*` / `http://localhost:*`) — the native
 *   desktop's local listener;
 * - the app's own URL schemes, current and legacy.
 *
 * Anything else — including any non-loopback http(s) — is ignored, and the
 * flow falls back to the scheme pair: the current scheme first, the legacy
 * one as a delayed second attempt for older installs.
 */
export function resolveCallbackTarget(redirectTo?: string | null): CallbackTarget {
  const requested = (redirectTo ?? '').trim()
  if (requested) {
    if (
      requested.startsWith(`${CALLBACK_SCHEME.split('://')[0]}://`) ||
      requested.startsWith(`${LEGACY_CALLBACK_SCHEME.split('://')[0]}://`)
    ) {
      return { base: requested }
    }
    if (isLoopbackCallbackUrl(requested)) {
      return { base: requested }
    }
  }
  return { base: CALLBACK_SCHEME, legacyFallback: LEGACY_CALLBACK_SCHEME }
}

/**
 * Whether `url` is the native desktop's local listener: plain http on
 * 127.0.0.1, localhost or [::1]. Anything unparseable is not.
 */
export function isLoopbackCallbackUrl(url: string): boolean {
  try {
    const parsed = new URL(url.trim())
    return (
      parsed.protocol === 'http:' &&
      (parsed.hostname === '127.0.0.1' ||
        parsed.hostname === 'localhost' ||
        parsed.hostname === '[::1]')
    )
  } catch {
    // Not a URL at all.
    return false
  }
}

/** A token pair minted for the desktop by `POST /auth/desktop-session`. */
export interface DesktopSession {
  access_token: string
  refresh_token: string
  /** Access-token lifetime in seconds. */
  expires_in: number
}

/** The gateway keys the bridge prepared for the desktop's CLIs. */
export interface DesktopSessionKeys {
  apiKey: string
  claudeApiKey?: string | null
  codexApiKey?: string | null
}

/**
 * The callback payload from a desktop session of its own - never from this
 * browser's tokens. Rejects an incomplete answer instead of delivering it,
 * because the desktop would store empty tokens and sign out on next launch
 * with nothing to explain why.
 */
export function payloadFromDesktopSession(
  session: DesktopSession,
  keys: DesktopSessionKeys,
  endpoint: string,
  now: number = Date.now()
): PaseoCallbackPayload {
  const accessToken = (session.access_token ?? '').trim()
  const refreshToken = (session.refresh_token ?? '').trim()
  const expiresIn = Number(session.expires_in)
  if (!accessToken || !refreshToken || !Number.isFinite(expiresIn) || expiresIn <= 0) {
    throw new Error('The service did not return a complete desktop session.')
  }
  if (!keys.apiKey?.trim()) {
    throw new Error('Missing API key state after browser login.')
  }
  return {
    accessToken,
    refreshToken,
    expiresAt: now + expiresIn * 1000,
    apiKey: keys.apiKey.trim(),
    claudeApiKey: keys.claudeApiKey ?? null,
    codexApiKey: keys.codexApiKey ?? null,
    endpoint: normalizePaseoEndpoint(endpoint)
  }
}

export function buildPaseoCallbackUrl(
  payload: PaseoCallbackPayload,
  options?: {
    now?: number
    callbackBase?: string
  }
): string {
  const params = new URLSearchParams()
  params.set('access_token', payload.accessToken)
  params.set('refresh_token', payload.refreshToken)
  params.set('expires_in', String(resolveExpiresInSeconds(payload.expiresAt, options?.now)))
  params.set('api_key', payload.apiKey)
  if (payload.claudeApiKey?.trim()) {
    params.set('claude_api_key', payload.claudeApiKey.trim())
  }
  if (payload.codexApiKey?.trim()) {
    params.set('codex_api_key', payload.codexApiKey.trim())
  }
  params.set('endpoint', normalizePaseoEndpoint(payload.endpoint))

  return `${options?.callbackBase ?? CALLBACK_SCHEME}#${params.toString()}`
}

// ---------------------------------------------------------------------------
// One-time sign-in code (PKCE)
//
// A desktop that cannot rely on its loopback listener being reachable sends a
// PKCE `code_challenge`. The page then never puts tokens in a URL: it asks the
// service for a short, single-use code bound to that challenge, shows it for
// the user to paste into the app, and — only when the listener is loopback —
// also hands it over as `#code=...`. The app redeems it with its verifier.
// ---------------------------------------------------------------------------

/** An S256 challenge: base64url of a SHA-256 digest, unpadded — 43 chars. */
const CODE_CHALLENGE_PATTERN = /^[A-Za-z0-9_-]{43}$/

/**
 * The desktop's code-mode request, or null for the legacy token-fragment flow.
 * Both parameters must be present and valid; anything else is legacy, so an
 * old client (or a mangled link) keeps the flow it always had.
 */
export function readDesktopCodeRequest(query: {
  code_challenge?: unknown
  code_challenge_method?: unknown
}): { challenge: string } | null {
  const challenge = query.code_challenge
  const method = query.code_challenge_method
  if (typeof challenge !== 'string' || typeof method !== 'string') {
    return null
  }
  if (method !== 'S256' || !CODE_CHALLENGE_PATTERN.test(challenge)) {
    return null
  }
  return { challenge }
}

/** `<base>#code=<code>&endpoint=<endpoint>` for the desktop's listener. */
export function buildDesktopCodeCallbackUrl(base: string, code: string, endpoint: string): string {
  const params = new URLSearchParams()
  params.set('code', code)
  params.set('endpoint', normalizePaseoEndpoint(endpoint))
  return `${base}#${params.toString()}`
}

/** A sign-in code this tab already obtained for one challenge. */
export interface DesktopLoginEntry {
  code: string
  /** Epoch milliseconds after which the code is no longer accepted. */
  expiresAt: number
  /** Whether the page already sent the browser to the loopback listener. */
  redirected: boolean
}

/** What `POST /auth/desktop-session/code` answers. */
export interface DesktopLoginCodeGrant {
  code: string
  /** Seconds until the code expires. */
  expires_in: number
}

/**
 * The stored entry for a freshly issued code. Rejects an incomplete answer
 * rather than showing the user a blank or already-dead code.
 */
export function desktopLoginEntryFromGrant(
  grant: DesktopLoginCodeGrant,
  now: number = Date.now()
): DesktopLoginEntry {
  const code = (grant?.code ?? '').trim()
  const expiresIn = Number(grant?.expires_in)
  if (!code || !Number.isFinite(expiresIn) || expiresIn <= 0) {
    throw new Error('The service did not return a sign-in code.')
  }
  return { code, expiresAt: now + expiresIn * 1000, redirected: false }
}

export function desktopLoginStorageKey(challenge: string): string {
  return `sub2api_desktop_login:${challenge}`
}

type EntryStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

/** `window.sessionStorage`, or null where touching it throws (sandboxed, disabled). */
export function safeSessionStorage(): EntryStorage | null {
  try {
    return typeof window !== 'undefined' ? window.sessionStorage : null
  } catch {
    return null
  }
}

/**
 * The unexpired entry for `challenge`, or null. An expired or malformed entry
 * is removed on the way. Storage failures read as "nothing stored".
 */
export function loadDesktopLoginEntry(
  storage: EntryStorage | null | undefined,
  challenge: string,
  now: number = Date.now()
): DesktopLoginEntry | null {
  if (!storage) {
    return null
  }
  const key = desktopLoginStorageKey(challenge)
  try {
    const raw = storage.getItem(key)
    if (!raw) {
      return null
    }
    const parsed = JSON.parse(raw) as Partial<DesktopLoginEntry> | null
    const code = typeof parsed?.code === 'string' ? parsed.code.trim() : ''
    const expiresAt = Number(parsed?.expiresAt)
    if (!code || !Number.isFinite(expiresAt) || expiresAt <= now) {
      storage.removeItem(key)
      return null
    }
    return { code, expiresAt, redirected: parsed?.redirected === true }
  } catch {
    return null
  }
}

/** Store `entry` for `challenge`; a storage failure is ignored. */
export function saveDesktopLoginEntry(
  storage: EntryStorage | null | undefined,
  challenge: string,
  entry: DesktopLoginEntry
): void {
  if (!storage) {
    return
  }
  try {
    storage.setItem(desktopLoginStorageKey(challenge), JSON.stringify(entry))
  } catch {
    // Quota or disabled storage: the code still shows, it just won't survive Back.
  }
}

/** `mm:ss` for a remaining duration, rounded up so it reaches 00:00 only at expiry. */
export function formatCountdown(remainingMs: number): string {
  const totalSeconds = Math.max(Math.ceil(remainingMs / 1000), 0)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}
