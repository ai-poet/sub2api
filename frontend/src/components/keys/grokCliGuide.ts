// Grok CLI (x.ai) 接入教程的共享内容。
//
// UseKeyModal 与 IntegrationGuidePanel 展示的是同一份教程，配置一旦分叉就会有一边过期，
// 所以安装命令、配置路径和 config.toml 模板都放在这里，两边只负责渲染。

export type GrokCliOS = 'unix' | 'windows'

/** Grok CLI 官方安装脚本。 */
export function grokCliInstallCommand(os: GrokCliOS): string {
  return os === 'windows'
    ? 'irm https://x.ai/cli/install.ps1 | iex'
    : 'curl -fsSL https://x.ai/cli/install.sh | bash'
}

/** 安装命令展示用的终端名。 */
export function grokCliInstallShellLabel(os: GrokCliOS): string {
  return os === 'windows' ? 'PowerShell' : 'Terminal'
}

/** Grok CLI 主配置文件路径。 */
export function grokCliConfigPath(os: GrokCliOS): string {
  return os === 'windows' ? '%USERPROFILE%\\.grok\\config.toml' : '~/.grok/config.toml'
}

/**
 * 生成 ~/.grok/config.toml。
 *
 * base_url 必须带 /v1，且 api_backend 固定为 responses —— 网关的 Grok 文本流量只走
 * POST /v1/responses，省略后 Grok CLI 会默认打 /v1/chat/completions。
 */
export function grokCliConfigToml(baseUrl: string, apiKey: string): string {
  return `[cli]
installer = "internal"

[model.grok]
model = "grok-4.6"
base_url = "${baseUrl}"
name = "grok"
api_key = "${apiKey}"
env_key = "${apiKey}"
context_window = 500000
api_backend = "responses"
supports_reasoning_effort = true
reasoning_efforts = [
    "low",
    "medium",
    "high",
]

[models]
default = "grok"
default_reasoning_effort = "high"

[session]
auto_compact_threshold_percent = 85
load_envrc = true

[memory]
enabled = true

[memory.session]
save_on_end = true

[ui]
fork_secondary_model = "grok"
max_thoughts_width = 120
yolo = false
compact_mode = false
permission_mode = "always-approve"

[subagents]
enabled = true`
}
