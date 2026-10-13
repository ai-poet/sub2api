/**
 * 「使用教程」页（`components/keys/IntegrationGuidePanel.vue`）给出的客户端配置。
 *
 * 「导入到 CCS」（`utils/ccswitchImport.ts`）把同一份配置原文随深链接带给 CC Switch，
 * 两边必须共用这里的生成函数：各写一份时，CCS 导入的 Codex 推理强度、上下文窗口、
 * 请求头和 Claude Code 环境变量都和教程对不上过。
 */

export const INTEGRATION_GUIDE_CODEX_MODEL = 'gpt-5.6-sol'

export function escapeTomlBasicString(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

/** `~/.claude/settings.json` 的 `env` 块。 */
export function claudeCodeSettingsEnv(baseUrl: string, apiKey: string): Record<string, string> {
  return {
    ANTHROPIC_BASE_URL: baseUrl,
    ANTHROPIC_AUTH_TOKEN: apiKey,
    CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: '1',
    CLAUDE_CODE_ATTRIBUTION_HEADER: '0'
  }
}

/** `~/.codex/config.toml`；`websockets` 对应教程里的「Codex CLI (WebSocket)」。 */
export function codexConfigToml(
  baseUrl: string,
  apiKey: string,
  options: { websockets?: boolean } = {}
): string {
  const websockets = options.websockets === true
  return `model_provider = "OpenAI"
model = "${INTEGRATION_GUIDE_CODEX_MODEL}"
review_model = "gpt-5.6-terra"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true
model_context_window = 1000000
model_auto_compact_token_limit = 900000

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${baseUrl}"
wire_api = "responses"
${websockets ? 'supports_websockets = true\n' : ''}requires_openai_auth = false
experimental_bearer_token = "${escapeTomlBasicString(apiKey)}"

[model_providers.OpenAI.http_headers]
x-openai-actor-authorization = "local-image-extension"

[features]
image_generation = true
remote_compaction_v2 = true${websockets ? '\nresponses_websockets_v2 = true' : ''}`
}

/**
 * 「导入到 CCS」链接的 `config` 参数（JSON 再 Base64），没有教程配置的应用返回 `undefined`。
 *
 * CC Switch 只收白名单里的键：Claude Code 收整个 `env`；Codex 收模型、推理强度、
 * 上下文窗口和供应商表（含 `http_headers`），`[features]`、`network_access` 这类
 * 用户全局设置会被它丢掉，所以导入后仍需按教程手动补上。
 */
export function ccSwitchImportConfigParam(
  app: string,
  endpoint: string,
  apiKey: string
): string | undefined {
  let payload: unknown
  switch (app) {
    case 'claude':
      payload = { env: claudeCodeSettingsEnv(endpoint, apiKey) }
      break
    case 'codex':
      payload = { config: codexConfigToml(endpoint, apiKey) }
      break
    default:
      return undefined
  }
  const bytes = new TextEncoder().encode(JSON.stringify(payload))
  return btoa(Array.from(bytes, (byte) => String.fromCharCode(byte)).join(''))
}
