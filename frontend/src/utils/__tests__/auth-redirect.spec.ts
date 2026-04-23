import { describe, it, expect, beforeEach } from 'vitest'
import {
  normalizeOAuthRedirectParam,
  parseAppInternalRedirect,
  OAUTH_RETURN_PATH_STORAGE_KEY,
  readStoredOAuthReturnPath,
  rememberOAuthReturnPath,
  rememberPaseoBridgeTargetIfApplicable,
  getPendingPaseoBridgeRoute,
} from '../auth-redirect'

describe('auth-redirect', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('normalizeOAuthRedirectParam handles string[]', () => {
    expect(normalizeOAuthRedirectParam(['/auth/paseo?x=1', 'other'])).toBe('/auth/paseo?x=1')
  })

  it('parseAppInternalRedirect splits path and query', () => {
    expect(parseAppInternalRedirect('/auth/paseo?endpoint=https%3A%2F%2Fexample.com')).toEqual({
      path: '/auth/paseo',
      query: { endpoint: 'https://example.com' },
    })
  })

  it('remember + read round-trip for OAuth return path', () => {
    rememberOAuthReturnPath('/auth/paseo?endpoint=x')
    expect(readStoredOAuthReturnPath()).toBe('/auth/paseo?endpoint=x')
  })

  it('readStoredOAuthReturnPath drops invalid JSON', () => {
    sessionStorage.setItem(OAUTH_RETURN_PATH_STORAGE_KEY, 'not-json')
    expect(readStoredOAuthReturnPath()).toBeUndefined()
  })

  it('rememberPaseoBridgeTargetIfApplicable only stores /auth/paseo', () => {
    rememberPaseoBridgeTargetIfApplicable('/dashboard')
    expect(readStoredOAuthReturnPath()).toBeUndefined()
    rememberPaseoBridgeTargetIfApplicable('/auth/paseo?endpoint=https%3A%2F%2Fx')
    expect(getPendingPaseoBridgeRoute()).toEqual({
      path: '/auth/paseo',
      query: { endpoint: 'https://x' },
    })
  })
})
