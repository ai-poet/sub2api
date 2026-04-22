export interface PaseoCallbackPayload {
  accessToken: string
  refreshToken: string
  expiresAt: number
  apiKey: string
  endpoint: string
}

export function normalizePaseoEndpoint(endpoint: string): string {
  return endpoint.trim().replace(/\/+$/, '')
}

export function resolveExpiresInSeconds(expiresAt: number, now: number = Date.now()): number {
  const remainingMs = expiresAt - now
  return Math.max(Math.floor(remainingMs / 1000), 0)
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
  params.set('endpoint', normalizePaseoEndpoint(payload.endpoint))

  return `${options?.callbackBase ?? 'paseo://auth/callback'}#${params.toString()}`
}
