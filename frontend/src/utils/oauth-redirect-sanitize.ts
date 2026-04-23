/**
 * Validates relative redirect targets after OAuth (open-redirect hardening).
 * Query strings may contain e.g. `endpoint=https://host` — `://` there is valid;
 * only the path segment before `?` is checked, matching the backend
 * (handler.sanitizeFrontendRedirectPath).
 */
export function sanitizeOAuthFrontendRedirect(
  path: string | null | undefined,
  fallback = '/dashboard',
): string {
  if (path == null) {
    return fallback
  }
  const trimmed = path.trim()
  if (trimmed === '') {
    return fallback
  }
  if (!trimmed.startsWith('/')) {
    return fallback
  }
  if (trimmed.startsWith('//')) {
    return fallback
  }
  const q = trimmed.indexOf('?')
  const pathOnly = q >= 0 ? trimmed.slice(0, q) : trimmed
  if (pathOnly.includes('://')) {
    return fallback
  }
  if (trimmed.includes('\n') || trimmed.includes('\r')) {
    return fallback
  }
  return trimmed
}
