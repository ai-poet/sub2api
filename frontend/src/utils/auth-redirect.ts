/**
 * Post-login / OAuth return path helpers (Vue Router may expose redirect as string | string[]).
 */

import type { RouteLocationNormalizedLoaded } from 'vue-router'

/** Set by PaseoBridgeView before sending users to /login; short TTL in JSON. */
export const OAUTH_RETURN_PATH_STORAGE_KEY = 'sub2api_oauth_redirect'

const OAUTH_RETURN_PATH_MAX_AGE_MS = 15 * 60 * 1000

export function normalizeOAuthRedirectParam(value: unknown): string | undefined {
  if (typeof value === 'string' && value !== '') {
    return value
  }
  if (Array.isArray(value) && value.length > 0 && typeof value[0] === 'string' && value[0] !== '') {
    return value[0]
  }
  return undefined
}

export function rememberOAuthReturnPath(fullPath: string): void {
  try {
    sessionStorage.setItem(
      OAUTH_RETURN_PATH_STORAGE_KEY,
      JSON.stringify({ p: fullPath, t: Date.now() }),
    )
  } catch {
    // ignore quota / private mode
  }
}

export function readStoredOAuthReturnPath(): string | undefined {
  try {
    const raw = sessionStorage.getItem(OAUTH_RETURN_PATH_STORAGE_KEY)
    if (!raw) return undefined
    const parsed = JSON.parse(raw) as { p?: unknown; t?: unknown }
    const p = typeof parsed.p === 'string' ? parsed.p : ''
    const t = typeof parsed.t === 'number' ? parsed.t : 0
    if (!p.startsWith('/') || Date.now() - t > OAUTH_RETURN_PATH_MAX_AGE_MS) {
      sessionStorage.removeItem(OAUTH_RETURN_PATH_STORAGE_KEY)
      return undefined
    }
    return p
  } catch {
    return undefined
  }
}

export function clearStoredOAuthReturnPath(): void {
  try {
    sessionStorage.removeItem(OAUTH_RETURN_PATH_STORAGE_KEY)
  } catch {
    // ignore
  }
}

/**
 * Target path for OAuth /start?redirect= — query param wins, then recent Paseo-bridge session backup.
 */
export function resolveOAuthStartRedirect(route: RouteLocationNormalizedLoaded): string {
  const fromQuery = normalizeOAuthRedirectParam(route.query.redirect)
  if (fromQuery) return fromQuery
  return readStoredOAuthReturnPath() ?? '/dashboard'
}

/**
 * Parse an internal location like `/auth/paseo?endpoint=...` for router.push/replace.
 */
export function parseAppInternalRedirect(fullPath: string): {
  path: string
  query?: Record<string, string>
} {
  const trimmed = fullPath.trim()
  if (!trimmed.startsWith('/')) {
    return { path: '/dashboard' }
  }
  const u = new URL(trimmed, 'http://localhost')
  const query: Record<string, string> = {}
  u.searchParams.forEach((v, k) => {
    query[k] = v
  })
  if (Object.keys(query).length === 0) {
    return { path: u.pathname }
  }
  return { path: u.pathname, query }
}
