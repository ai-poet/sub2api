import { describe, expect, it } from 'vitest'
import { buildPaseoCallbackUrl, resolveExpiresInSeconds } from '../paseo-bridge'

describe('paseo-bridge', () => {
  it('builds a paseo callback url with tokens, scoped api keys, and endpoint', () => {
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
      'paseo://auth/callback#access_token=access-token&refresh_token=refresh-token&expires_in=90&api_key=sk-live-example&claude_api_key=sk-claude&codex_api_key=sk-codex&endpoint=https%3A%2F%2Fapi.example.com'
    )
  })

  it('clamps expires_in at zero when the token is already expired', () => {
    expect(resolveExpiresInSeconds(1000, 2000)).toBe(0)
  })
})
