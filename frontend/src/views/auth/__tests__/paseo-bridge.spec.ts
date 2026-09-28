import { describe, expect, it } from 'vitest'
import {
  buildDesktopCodeCallbackUrl,
  buildPaseoCallbackUrl,
  CALLBACK_SCHEME,
  desktopLoginEntryFromGrant,
  desktopLoginStorageKey,
  formatCountdown,
  isLoopbackCallbackUrl,
  LEGACY_CALLBACK_SCHEME,
  loadDesktopLoginEntry,
  payloadFromDesktopSession,
  readDesktopCodeRequest,
  resolveCallbackTarget,
  resolveExpiresInSeconds,
  saveDesktopLoginEntry
} from '../paseo-bridge'

describe('paseo-bridge', () => {
  it('builds the payload from a desktop session of its own, not the browser tokens', () => {
    const payload = payloadFromDesktopSession(
      { access_token: ' desktop-at ', refresh_token: 'rt_desktop', expires_in: 900 },
      { apiKey: 'sk-live', claudeApiKey: 'sk-claude', codexApiKey: null },
      'https://api.example.com/',
      1_710_000_000_000
    )

    expect(payload).toEqual({
      accessToken: 'desktop-at',
      refreshToken: 'rt_desktop',
      expiresAt: 1_710_000_900_000,
      apiKey: 'sk-live',
      claudeApiKey: 'sk-claude',
      codexApiKey: null,
      endpoint: 'https://api.example.com'
    })
    expect(
      buildPaseoCallbackUrl(payload, {
        now: 1_710_000_000_000,
        callbackBase: 'http://127.0.0.1:5/callback'
      })
    ).toContain('refresh_token=rt_desktop&expires_in=900')
  })

  it('refuses an incomplete desktop session rather than delivering empty tokens', () => {
    const keys = { apiKey: 'sk-live' }
    const origin = 'https://a.org'
    expect(() =>
      payloadFromDesktopSession({ access_token: '', refresh_token: 'rt', expires_in: 900 }, keys, origin)
    ).toThrow(/complete desktop session/)
    expect(() =>
      payloadFromDesktopSession({ access_token: 'at', refresh_token: '', expires_in: 900 }, keys, origin)
    ).toThrow(/complete desktop session/)
    expect(() =>
      payloadFromDesktopSession({ access_token: 'at', refresh_token: 'rt', expires_in: 0 }, keys, origin)
    ).toThrow(/complete desktop session/)
    expect(() =>
      payloadFromDesktopSession(
        { access_token: 'at', refresh_token: 'rt', expires_in: 900 },
        { apiKey: ' ' },
        origin
      )
    ).toThrow(/API key/)
  })
  it('builds a callback url with tokens, scoped api keys, and endpoint', () => {
    const url = buildPaseoCallbackUrl(
      {
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        expiresAt: 1_710_000_090_000,
        apiKey: 'sk-live-example',
        claudeApiKey: 'sk-claude',
        codexApiKey: 'sk-codex',
        endpoint: 'https://api.example.com/'
      },
      { now: 1_710_000_000_000 }
    )

    expect(url).toBe(
      'agentdesk://auth/callback#access_token=access-token&refresh_token=refresh-token&expires_in=90&api_key=sk-live-example&claude_api_key=sk-claude&codex_api_key=sk-codex&endpoint=https%3A%2F%2Fapi.example.com'
    )
  })

  it('clamps expires_in at zero when the token is already expired', () => {
    expect(resolveExpiresInSeconds(1000, 2000)).toBe(0)
  })

  it('defaults to the current scheme with the legacy scheme as fallback', () => {
    for (const absent of [undefined, null, '', '   ']) {
      expect(resolveCallbackTarget(absent)).toEqual({
        base: CALLBACK_SCHEME,
        legacyFallback: LEGACY_CALLBACK_SCHEME
      })
    }
  })

  it('honours an explicit scheme without adding a fallback', () => {
    expect(resolveCallbackTarget('agentdesk://auth/callback')).toEqual({
      base: 'agentdesk://auth/callback'
    })
    // An old client that names itself keeps working unchanged.
    expect(resolveCallbackTarget('paseo://auth/callback')).toEqual({
      base: 'paseo://auth/callback'
    })
  })

  it('accepts the native desktop loopback listener', () => {
    for (const loopback of [
      'http://127.0.0.1:51789/callback',
      'http://localhost:8123/callback',
      'http://[::1]:9000/callback'
    ]) {
      expect(resolveCallbackTarget(loopback)).toEqual({ base: loopback })
    }
  })

  it('refuses to deliver the session anywhere else', () => {
    // The fragment carries the whole session, so a crafted redirect_to must
    // fall back to the app schemes instead of walking away with the tokens.
    for (const hostile of [
      'https://evil.example.com/steal',
      'http://evil.example.com/steal',
      'http://127.0.0.1.evil.example.com/steal',
      'https://127.0.0.1:8443/callback',
      'javascript:alert(1)',
      'file:///tmp/x',
      'not a url'
    ]) {
      expect(resolveCallbackTarget(hostile)).toEqual({
        base: CALLBACK_SCHEME,
        legacyFallback: LEGACY_CALLBACK_SCHEME
      })
    }
  })
})

describe('desktop sign-in code (PKCE)', () => {
  // base64url(SHA-256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")), RFC 7636 appendix B.
  const challenge = 'E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM'

  function memoryStorage(): Storage {
    const values = new Map<string, string>()
    return {
      get length() {
        return values.size
      },
      clear: () => values.clear(),
      getItem: (key: string) => (values.has(key) ? values.get(key)! : null),
      key: (index: number) => Array.from(values.keys())[index] ?? null,
      removeItem: (key: string) => {
        values.delete(key)
      },
      setItem: (key: string, value: string) => {
        values.set(key, String(value))
      }
    }
  }

  it('reads a valid S256 challenge as code mode', () => {
    expect(challenge).toHaveLength(43)
    expect(
      readDesktopCodeRequest({ code_challenge: challenge, code_challenge_method: 'S256' })
    ).toEqual({ challenge })
  })

  it('treats anything but a valid challenge and S256 as the legacy flow', () => {
    for (const query of [
      {},
      { code_challenge: challenge },
      { code_challenge_method: 'S256' },
      { code_challenge: challenge, code_challenge_method: 'plain' },
      { code_challenge: challenge, code_challenge_method: 's256' },
      { code_challenge: challenge, code_challenge_method: '' },
      { code_challenge: challenge.slice(0, 42), code_challenge_method: 'S256' },
      { code_challenge: `${challenge}A`, code_challenge_method: 'S256' },
      { code_challenge: `${challenge.slice(0, 42)}=`, code_challenge_method: 'S256' },
      { code_challenge: `${challenge.slice(0, 42)}+`, code_challenge_method: 'S256' },
      { code_challenge: `${challenge.slice(0, 42)}/`, code_challenge_method: 'S256' },
      { code_challenge: ` ${challenge.slice(0, 42)}`, code_challenge_method: 'S256' },
      { code_challenge: [challenge], code_challenge_method: 'S256' },
      { code_challenge: challenge, code_challenge_method: ['S256'] },
      { code_challenge: null, code_challenge_method: 'S256' }
    ]) {
      expect(readDesktopCodeRequest(query)).toBeNull()
    }
  })

  it('builds the code callback url with the code and normalized endpoint', () => {
    expect(
      buildDesktopCodeCallbackUrl(
        'http://127.0.0.1:51789/callback',
        'K7QM-3XPD',
        'https://api.example.com/'
      )
    ).toBe('http://127.0.0.1:51789/callback#code=K7QM-3XPD&endpoint=https%3A%2F%2Fapi.example.com')
  })

  it('recognizes only plain-http loopback listeners', () => {
    for (const loopback of [
      'http://127.0.0.1:51789/callback',
      'http://localhost:8123/callback',
      'http://[::1]:9000/callback'
    ]) {
      expect(isLoopbackCallbackUrl(loopback)).toBe(true)
    }
    for (const other of [
      CALLBACK_SCHEME,
      LEGACY_CALLBACK_SCHEME,
      'https://127.0.0.1:8443/callback',
      'http://127.0.0.1.evil.example.com/callback',
      'http://evil.example.com/callback',
      'not a url',
      ''
    ]) {
      expect(isLoopbackCallbackUrl(other)).toBe(false)
    }
  })

  it('turns a grant into an entry and refuses an incomplete one', () => {
    expect(desktopLoginEntryFromGrant({ code: ' K7QM-3XPD ', expires_in: 600 }, 1_000)).toEqual({
      code: 'K7QM-3XPD',
      expiresAt: 601_000,
      redirected: false
    })
    expect(() => desktopLoginEntryFromGrant({ code: '', expires_in: 600 })).toThrow(/sign-in code/)
    expect(() => desktopLoginEntryFromGrant({ code: 'K7QM-3XPD', expires_in: 0 })).toThrow(
      /sign-in code/
    )
  })

  it('stores, reloads and expires the entry per challenge', () => {
    const storage = memoryStorage()
    const entry = { code: 'K7QM-3XPD', expiresAt: 10_000, redirected: true }

    expect(desktopLoginStorageKey(challenge)).toBe(`sub2api_desktop_login:${challenge}`)
    expect(loadDesktopLoginEntry(storage, challenge, 0)).toBeNull()

    saveDesktopLoginEntry(storage, challenge, entry)
    expect(storage.getItem(desktopLoginStorageKey(challenge))).toBe(JSON.stringify(entry))
    expect(loadDesktopLoginEntry(storage, challenge, 9_999)).toEqual(entry)
    // Another challenge is another sign-in.
    expect(loadDesktopLoginEntry(storage, 'x'.repeat(43), 9_999)).toBeNull()

    // At expiry the entry is gone, and removed on the way.
    expect(loadDesktopLoginEntry(storage, challenge, 10_000)).toBeNull()
    expect(storage.getItem(desktopLoginStorageKey(challenge))).toBeNull()
  })

  it('reads malformed or unavailable storage as nothing stored', () => {
    const storage = memoryStorage()
    storage.setItem(desktopLoginStorageKey(challenge), '{not json')
    expect(loadDesktopLoginEntry(storage, challenge, 0)).toBeNull()

    storage.setItem(desktopLoginStorageKey(challenge), JSON.stringify({ code: '', expiresAt: 5 }))
    expect(loadDesktopLoginEntry(storage, challenge, 0)).toBeNull()
    expect(storage.getItem(desktopLoginStorageKey(challenge))).toBeNull()

    const throwing = {
      getItem: () => {
        throw new Error('denied')
      },
      setItem: () => {
        throw new Error('denied')
      },
      removeItem: () => {
        throw new Error('denied')
      }
    }
    expect(loadDesktopLoginEntry(throwing, challenge, 0)).toBeNull()
    expect(() =>
      saveDesktopLoginEntry(throwing, challenge, { code: 'A', expiresAt: 1, redirected: false })
    ).not.toThrow()
    expect(loadDesktopLoginEntry(null, challenge, 0)).toBeNull()
  })

  it('formats the countdown as mm:ss, rounding up', () => {
    expect(formatCountdown(600_000)).toBe('10:00')
    expect(formatCountdown(599_001)).toBe('10:00')
    expect(formatCountdown(599_000)).toBe('09:59')
    expect(formatCountdown(61_000)).toBe('01:01')
    expect(formatCountdown(500)).toBe('00:01')
    expect(formatCountdown(0)).toBe('00:00')
    expect(formatCountdown(-5_000)).toBe('00:00')
  })
})
