import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  buildCompleteURL,
  clearGiveState,
  createHandoffPkcePair,
  hasPendingGive,
  isOnAliasOrigin,
  normalizeOrigin,
  parseGiveQuery,
  pickForwardedOAuthParams,
  readReceiveState,
  readSessionHandoffConfig,
  saveGiveState,
  startBindingOnLoginOrigin,
  startOAuthOnLoginOrigin,
} from '../sessionHandoff'

const LOGIN = 'https://cheaprouter.cc'
const ALIAS = 'https://cdn.cheaprouter.cc'
const settings = { session_handoff_login_origin: LOGIN, session_handoff_alias_origins: [ALIAS] }
const config = { loginOrigin: LOGIN, aliasOrigins: [ALIAS] }
const CHALLENGE = 'A'.repeat(43)

const originalLocation = window.location

function stubLocation(origin: string) {
  const assign = vi.fn()
  Object.defineProperty(window, 'location', {
    value: { ...originalLocation, origin, assign, href: `${origin}/login` },
    writable: true,
  })
  return assign
}

async function sha256Base64Url(value: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value))
  let binary = ''
  for (const byte of new Uint8Array(digest)) binary += String.fromCharCode(byte)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

describe('session handoff utils', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', { value: originalLocation, writable: true })
    vi.useRealTimers()
  })

  it('reads the config only when both a login origin and an alias are set', () => {
    expect(readSessionHandoffConfig(settings)).toEqual(config)
    expect(readSessionHandoffConfig({ session_handoff_login_origin: LOGIN, session_handoff_alias_origins: [] })).toBeNull()
    expect(readSessionHandoffConfig({ session_handoff_alias_origins: [ALIAS] })).toBeNull()
    expect(readSessionHandoffConfig(null)).toBeNull()
    expect(readSessionHandoffConfig({ session_handoff_login_origin: `${LOGIN}/`, session_handoff_alias_origins: [`${LOGIN}`, 'not a url', ALIAS] }))
      .toEqual(config)
  })

  it('normalizes origins and rejects anything with a path or another scheme', () => {
    expect(normalizeOrigin('HTTPS://CDN.cheaprouter.cc:443/')).toBe(ALIAS)
    expect(normalizeOrigin('https://cdn.cheaprouter.cc/login')).toBe('')
    expect(normalizeOrigin('javascript:alert(1)')).toBe('')
    expect(normalizeOrigin(42)).toBe('')
  })

  it('knows which side of the handoff the page is on', () => {
    expect(isOnAliasOrigin(config, ALIAS)).toBe(true)
    expect(isOnAliasOrigin(config, LOGIN)).toBe(false)
    expect(isOnAliasOrigin(config, 'https://other.example')).toBe(false)
    expect(isOnAliasOrigin(null, ALIAS)).toBe(false)
  })

  it('parses a give request and drops everything it does not trust', () => {
    expect(parseGiveQuery({ origin: ALIAS, challenge: CHALLENGE, redirect: '/keys?x=1', provider: 'linuxdo', aff_code: 'AFF', intent: 'bind_current_user' }, config, LOGIN))
      .toEqual({ origin: ALIAS, challenge: CHALLENGE, redirect: '/keys?x=1', provider: 'linuxdo', params: { aff_code: 'AFF' } })

    expect(parseGiveQuery({ origin: ALIAS, challenge: CHALLENGE, redirect: 'https://evil.example', provider: 'evil' }, config, LOGIN))
      .toEqual({ origin: ALIAS, challenge: CHALLENGE, redirect: '', provider: undefined, params: {} })

    expect(parseGiveQuery({ origin: 'https://evil.example', challenge: CHALLENGE }, config, LOGIN)).toBeNull()
    expect(parseGiveQuery({ origin: LOGIN, challenge: CHALLENGE }, config, LOGIN)).toBeNull()
    expect(parseGiveQuery({ origin: ALIAS, challenge: 'short' }, config, LOGIN)).toBeNull()
  })

  it('forwards only allow-listed OAuth params', () => {
    expect(pickForwardedOAuthParams({ redirect: '/x', aff_code: 'A', mode: 'open', intent: 'bind_current_user', extra: 'x' }))
      .toEqual({ aff_code: 'A', mode: 'open' })
  })

  it('puts the code in the fragment of the receiver complete page', () => {
    expect(buildCompleteURL(ALIAS, 'abc_-123', '/keys')).toBe(`${ALIAS}/auth/handoff/complete#code=abc_-123&redirect=%2Fkeys`)
    expect(buildCompleteURL(ALIAS, 'abc', '')).toBe(`${ALIAS}/auth/handoff/complete#code=abc`)
  })

  it('creates a PKCE pair whose challenge is the SHA-256 of the verifier', async () => {
    const { verifier, challenge } = await createHandoffPkcePair()
    expect(verifier).toMatch(/^[A-Za-z0-9_-]{43}$/)
    expect(challenge).toMatch(/^[A-Za-z0-9_-]{43}$/)
    expect(challenge).toBe(await sha256Base64Url(verifier))
  })

  it('expires a stale give state', () => {
    saveGiveState({ origin: ALIAS, challenge: CHALLENGE, redirect: '' })
    expect(hasPendingGive()).toBe(true)
    clearGiveState()
    expect(hasPendingGive()).toBe(false)

    saveGiveState({ origin: ALIAS, challenge: CHALLENGE, redirect: '', t: Date.now() - 16 * 60 * 1000 })
    expect(hasPendingGive()).toBe(false)
  })

  it('sends OAuth started on the alias to the login origin with a fresh challenge', async () => {
    const assign = stubLocation(ALIAS)
    const started = await startOAuthOnLoginOrigin(settings, { provider: 'github', params: { redirect: '/keys', aff_code: 'AFF' } })

    expect(started).toBe(true)
    const target = new URL(assign.mock.calls[0][0] as string)
    expect(target.origin).toBe(LOGIN)
    expect(target.pathname).toBe('/auth/handoff')
    expect(target.searchParams.get('origin')).toBe(ALIAS)
    expect(target.searchParams.get('provider')).toBe('github')
    expect(target.searchParams.get('redirect')).toBe('/keys')
    expect(target.searchParams.get('aff_code')).toBe('AFF')

    const state = readReceiveState()
    expect(state?.from).toBe(LOGIN)
    expect(state?.redirect).toBe('/keys')
    expect(target.searchParams.get('challenge')).toBe(await sha256Base64Url(state!.verifier))
  })

  it('leaves OAuth alone on the login origin or when handoff is off', async () => {
    const assign = stubLocation(LOGIN)
    expect(await startOAuthOnLoginOrigin(settings, { provider: 'github', params: { redirect: '/keys' } })).toBe(false)
    stubLocation(ALIAS)
    expect(await startOAuthOnLoginOrigin({}, { provider: 'github', params: { redirect: '/keys' } })).toBe(false)
    expect(assign).not.toHaveBeenCalled()
    expect(readReceiveState()).toBeNull()
  })

  it('pulls the session to the login origin before binding from the alias', () => {
    const assign = stubLocation(ALIAS)
    expect(startBindingOnLoginOrigin(settings, '/profile?tab=bind')).toBe(true)
    const target = new URL(assign.mock.calls[0][0] as string)
    expect(target.origin).toBe(LOGIN)
    expect(target.pathname).toBe('/auth/handoff/pull')
    expect(target.searchParams.get('from')).toBe(ALIAS)
    expect(target.searchParams.get('redirect')).toBe('/profile?tab=bind')

    stubLocation(LOGIN)
    expect(startBindingOnLoginOrigin(settings, '/profile')).toBe(false)
  })
})
