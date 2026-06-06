import type { SelectMenuOption } from './select-menu-types'

// 数据形状参照 client/packages/app 真客户端：
// - Mode 来自 utils/agent-mode-localization.ts MODE_TEXT_BY_ID
// - Thinking 来自 i18n/sub2api.ts thinkingEffortLabels
// - Model 取最新的几代模型，默认值与 i18n 中 home.clientWorkflow.model 对齐

export const MODE_OPTIONS: SelectMenuOption[] = [
  { id: 'always-ask', label: 'Always Ask', description: 'Ask before using a tool the first time', indicatorClass: 'dot-amber' },
  { id: 'acceptEdits', label: 'Accept file edits', description: 'Auto-approve file edits without prompts', indicatorClass: 'dot-blue' },
  { id: 'plan', label: 'Plan', description: 'Analyze the codebase, no edits or commands', indicatorClass: 'dot-violet' },
  { id: 'bypassPermissions', label: 'Bypass', description: 'Skip every permission prompt (use with care)', indicatorClass: 'dot-red' },
]

export const THINKING_OPTIONS: SelectMenuOption[] = [
  { id: 'low', label: 'Low' },
  { id: 'medium', label: 'Medium' },
  { id: 'high', label: 'High' },
  { id: 'extraHigh', label: 'Extra high' },
]

export const MODEL_OPTIONS: SelectMenuOption[] = [
  { id: 'opus-4-7-1m', label: 'Opus 4.7 1M', description: 'Claude · deepest reasoning' },
  { id: 'opus-4-8-1m', label: 'Opus 4.8 1M', description: 'Claude · default for the workflow demo' },
  { id: 'sonnet-4-6', label: 'Sonnet 4.6', description: 'Claude · balanced default' },
  { id: 'haiku-4-5', label: 'Haiku 4.5', description: 'Claude · fastest, cheapest' },
  { id: 'gpt-5-codex', label: 'GPT-5 Codex', description: 'OpenAI · for Codex sessions' },
]

export const DEFAULT_MODE_ID = 'bypassPermissions'
export const DEFAULT_THINKING_ID = 'medium'
export const DEFAULT_MODEL_ID = 'opus-4-8-1m'
