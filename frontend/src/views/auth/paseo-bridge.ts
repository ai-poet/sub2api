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
    try {
      const parsed = new URL(requested)
      const loopback =
        parsed.protocol === 'http:' &&
        (parsed.hostname === '127.0.0.1' ||
          parsed.hostname === 'localhost' ||
          parsed.hostname === '[::1]')
      if (loopback) {
        return { base: requested }
      }
    } catch {
      // Not a URL at all; treated as absent.
    }
  }
  return { base: CALLBACK_SCHEME, legacyFallback: LEGACY_CALLBACK_SCHEME }
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
