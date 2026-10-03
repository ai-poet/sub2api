import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  APPEAL_SESSION_KEY,
  captureAppealToken,
  clearAppealSession,
  consumeAppealFragment,
  consumeAppealLoginNotice,
  extractAppealToken,
  readAppealSession,
  setAppealLoginNotice,
  storeAppealSession
} from '../appeal'

describe('appeal session utils (fork)', () => {
  beforeEach(() => {
    sessionStorage.clear()
    localStorage.clear()
  })
  afterEach(() => vi.useRealTimers())

  it('stores the token in sessionStorage only, never as auth_token', () => {
    const state = storeAppealSession('apl_abc', '7200')
    expect(state?.token).toBe('apl_abc')
    expect(readAppealSession()?.token).toBe('apl_abc')
    expect(sessionStorage.getItem(APPEAL_SESSION_KEY)).toContain('apl_abc')
    expect(localStorage.getItem('auth_token')).toBeNull()

    clearAppealSession()
    expect(readAppealSession()).toBeNull()
  })

  it('rejects tokens without the appeal prefix', () => {
    expect(storeAppealSession('eyJhbGciOi.jwt', 60)).toBeNull()
    expect(readAppealSession()).toBeNull()
  })

  it('drops expired sessions', () => {
    vi.useFakeTimers()
    storeAppealSession('apl_abc', 10)
    vi.advanceTimersByTime(11_000)
    expect(readAppealSession()).toBeNull()
    expect(sessionStorage.getItem(APPEAL_SESSION_KEY)).toBeNull()
  })

  it('extracts the token only from USER_NOT_ACTIVE errors', () => {
    expect(extractAppealToken({ reason: 'USER_NOT_ACTIVE', metadata: { appeal_token: 'apl_x', expires_in: '7200' } }))
      .toEqual({ token: 'apl_x', expiresIn: '7200' })
    expect(extractAppealToken({ reason: 'INVALID_CREDENTIALS', metadata: { appeal_token: 'apl_x' } })).toBeNull()
    expect(extractAppealToken({ reason: 'USER_NOT_ACTIVE' })).toBeNull()
    expect(extractAppealToken({ reason: 'USER_NOT_ACTIVE', metadata: { appeal_token: 'not-appeal' } })).toBeNull()
    expect(extractAppealToken(null)).toBeNull()

    expect(captureAppealToken({ reason: 'USER_NOT_ACTIVE', metadata: { appeal_token: 'apl_y' } })).toBe(true)
    expect(readAppealSession()?.token).toBe('apl_y')
  })

  it('consumes the appeal fragment and strips it from the address bar', () => {
    window.history.replaceState({}, '', '/auth/linuxdo/callback#appeal_token=apl_z&expires_in=7200')
    const params = new URLSearchParams(window.location.hash.slice(1))
    expect(consumeAppealFragment(params)).toBe(true)
    expect(readAppealSession()?.token).toBe('apl_z')
    expect(window.location.hash).toBe('')

    expect(consumeAppealFragment(new URLSearchParams('access_token=abc'))).toBe(false)
  })

  it('login notice is read once', () => {
    setAppealLoginNotice('restored')
    expect(consumeAppealLoginNotice()).toBe('restored')
    expect(consumeAppealLoginNotice()).toBeNull()
  })
})
