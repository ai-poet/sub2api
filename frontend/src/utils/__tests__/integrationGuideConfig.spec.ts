import { describe, expect, it } from 'vitest'
import { OPENAI_CC_SWITCH_CODEX_MODEL, buildCcSwitchImportDeeplink, type CcSwitchClientType } from '@/utils/ccswitchImport'
import {
  INTEGRATION_GUIDE_CODEX_MODEL,
  claudeCodeSettingsEnv,
  codexConfigToml
} from '@/utils/integrationGuideConfig'
import type { GroupPlatform } from '@/types'

const baseUrl = 'https://api.example.com'
const apiKey = 'sk-test'

function importLink(platform: GroupPlatform, clientType: CcSwitchClientType = 'claude') {
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl,
    platform,
    clientType,
    providerName: 'Sub2API',
    apiKey,
    usageScript: 'return true'
  })
  const params = new URLSearchParams(deeplink.split('?')[1] || '')
  const raw = params.get('config')
  const config = raw === null
    ? null
    : JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(raw), (c) => c.charCodeAt(0))))
  return { params, config }
}

describe('CC Switch import carries the integration guide config', () => {
  it('sends the guide Claude Code env for Claude imports', () => {
    const { params, config } = importLink('anthropic')

    expect(params.get('configFormat')).toBe('json')
    expect(config).toEqual({ env: claudeCodeSettingsEnv(baseUrl, apiKey) })
    expect(config.env.CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC).toBe('1')
    expect(config.env.CLAUDE_CODE_ATTRIBUTION_HEADER).toBe('0')
  })

  it('points Antigravity Claude imports at the antigravity endpoint', () => {
    const { params, config } = importLink('antigravity', 'claude')

    expect(params.get('endpoint')).toBe(`${baseUrl}/antigravity`)
    expect(config).toEqual({ env: claudeCodeSettingsEnv(`${baseUrl}/antigravity`, apiKey) })
  })

  it('sends the guide config.toml for Codex imports', () => {
    const { params, config } = importLink('openai')

    expect(config).toEqual({ config: codexConfigToml(params.get('endpoint') as string, apiKey) })
    expect(params.get('model')).toBe(INTEGRATION_GUIDE_CODEX_MODEL)
    expect(OPENAI_CC_SWITCH_CODEX_MODEL).toBe(INTEGRATION_GUIDE_CODEX_MODEL)
  })

  it.each([
    { platform: 'gemini' as GroupPlatform, clientType: 'gemini' as const },
    { platform: 'antigravity' as GroupPlatform, clientType: 'gemini' as const },
    { platform: 'grok' as GroupPlatform, clientType: 'claude' as const }
  ])('sends no config for $platform imports without a guide config', ({ platform, clientType }) => {
    expect(importLink(platform, clientType).config).toBeNull()
  })
})

describe('integration guide Codex config', () => {
  it('adds only the WebSocket lines for the WebSocket variant', () => {
    const plain = codexConfigToml(baseUrl, apiKey)
    const websockets = codexConfigToml(baseUrl, apiKey, { websockets: true })

    expect(plain).not.toContain('websockets')
    expect(websockets).toBe(
      plain
        .replace('wire_api = "responses"\n', 'wire_api = "responses"\nsupports_websockets = true\n')
        .concat('\nresponses_websockets_v2 = true')
    )
  })

  it('escapes the API key as a TOML basic string', () => {
    expect(codexConfigToml(baseUrl, 'sk-"q\\')).toContain('experimental_bearer_token = "sk-\\"q\\\\"')
  })
})
