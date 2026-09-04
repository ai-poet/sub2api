import { describe, expect, it } from 'vitest'
import {
  buildPaseoCallbackUrl,
  CALLBACK_SCHEME,
  LEGACY_CALLBACK_SCHEME,
  payloadFromDesktopSession,
  resolveCallbackTarget,
  resolveExpiresInSeconds
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
