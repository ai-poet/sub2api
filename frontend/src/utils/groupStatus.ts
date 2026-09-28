import type { GroupRuntimeStatus, GroupStatusValidationMode } from '@/types'

export type NormalizedGroupRuntimeStatus = GroupRuntimeStatus | 'unknown'

export function normalizeGroupRuntimeStatus(status?: string | null): NormalizedGroupRuntimeStatus {
  switch (status) {
    case 'up':
    case 'degraded':
    case 'down':
      return status
    default:
      return 'unknown'
  }
}

export function getGroupRuntimeStatusBadgeClass(status?: string | null): string {
  switch (normalizeGroupRuntimeStatus(status)) {
    case 'up':
      return 'badge-success'
    case 'degraded':
      return 'badge-warning'
    case 'down':
      return 'badge-danger'
    default:
      return 'badge-gray'
  }
}

export function getGroupRuntimeStatusSurfaceClass(status?: string | null): string {
  switch (normalizeGroupRuntimeStatus(status)) {
    case 'up':
      return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/40 dark:bg-emerald-950/20 dark:text-emerald-300'
    case 'degraded':
      return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-300'
    case 'down':
      return 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-900/40 dark:bg-rose-950/20 dark:text-rose-300'
    default:
      return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300'
  }
}

export function getGroupRuntimeStatusBarClass(status?: string | null): string {
  switch (normalizeGroupRuntimeStatus(status)) {
    case 'up':
      return 'bg-emerald-500/85'
    case 'degraded':
      return 'bg-amber-500/85'
    case 'down':
      return 'bg-rose-500/85'
    default:
      return 'bg-gray-300 dark:bg-dark-600'
  }
}

export function formatGroupRuntimeLatency(latencyMS?: number | null): string {
  if (latencyMS === null || latencyMS === undefined || !Number.isFinite(latencyMS)) {
    return '-'
  }
  if (latencyMS < 1000) {
    return `${Math.round(latencyMS)} ms`
  }
  const seconds = latencyMS / 1000
  return `${seconds >= 10 ? seconds.toFixed(0) : seconds.toFixed(1)} s`
}

export function formatGroupRuntimeAvailability(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return '-'
  }
  const digits = value >= 99 ? 2 : 1
  return `${value.toFixed(digits)}%`
}

export function splitRuntimeKeywordsText(raw: string): string[] {
  return raw
    .split(/[\n,，]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function joinRuntimeKeywordsText(keywords: string[] | null | undefined): string {
  if (!Array.isArray(keywords) || keywords.length === 0) {
    return ''
  }
  return keywords.join('\n')
}

export function shouldShowRuntimeKeywordEditor(mode: GroupStatusValidationMode): boolean {
  return mode === 'keywords_any' || mode === 'keywords_all'
}

const runtimeRequestPrefixPattern = /\b(?:Get|Post|Put|Patch|Delete|Head)\s+"https?:\/\/[^"]*":\s*/gi
const runtimeUpstreamUrlPattern = /https?:\/\/[^\s"'<>]+/gi

// 后端新版探测结果已不含上游地址，这里兜底处理历史记录：
// 去掉 Go url.Error 形如 `Post "https://host/path": reason` 的请求前缀，
// 并把残留的 URL 替换成占位符，避免上游地址暴露在状态页上。
export function sanitizeRuntimeErrorDetail(text?: string | null): string {
  const trimmed = (text || '').trim()
  if (!trimmed) {
    return ''
  }
  return trimmed
    .replace(runtimeRequestPrefixPattern, '')
    .replace(runtimeUpstreamUrlPattern, '[upstream]')
    .trim()
}

export function shortenRuntimeExcerpt(text?: string | null, maxLength: number = 140): string {
  const trimmed = (text || '').trim()
  if (!trimmed) {
    return ''
  }
  if (trimmed.length <= maxLength) {
    return trimmed
  }
  return `${trimmed.slice(0, maxLength).trimEnd()}...`
}

// ==================== meow 指纹验证（v3 基准，行为指纹，多模型） ====================

export type NormalizedAstraCheckStatus = 'pass' | 'mismatch' | 'suspect' | 'insufficient' | 'unknown'

// 稳定 mismatch（连续 2 次确认）优先显示红色；否则显示最近一次结果：
// match → 一致，单次 mismatch → 疑似（等复测），insufficient → 证据不足。旧的绿色不会盖住新结果。
export function normalizeAstraCheckStatus(
  stable?: string | null,
  verdict?: string | null
): NormalizedAstraCheckStatus {
  if (stable === 'mismatch') {
    return 'mismatch'
  }
  switch (verdict) {
    case 'match':
      return 'pass'
    case 'mismatch':
      return 'suspect'
    case 'insufficient':
      return 'insufficient'
    default:
      return stable === 'pass' ? 'pass' : 'unknown'
  }
}

export function getAstraCheckBadgeClass(status?: string | null): string {
  switch (status) {
    case 'pass':
      return 'badge-success'
    case 'mismatch':
      return 'badge-danger'
    case 'suspect':
    case 'insufficient':
      return 'badge-warning'
    default:
      return 'badge-gray'
  }
}

export function isAstraCheckEvent(eventType?: string | null): boolean {
  return eventType === 'astra_mismatch' || eventType === 'astra_recovered'
}

// 已下线的纯 Sol 验证（Juice）与 ModelTrace 留下的历史事件
export function isLegacyFingerprintEvent(eventType?: string | null): boolean {
  switch (eventType) {
    case 'sol_juice_mismatch':
    case 'sol_juice_recovered':
    case 'modeltrace_mismatch':
    case 'modeltrace_recovered':
      return true
    default:
      return false
  }
}

// 与后端 astraModelLabels 一致；另收 ModelTrace 时代的 id，供历史事件显示
const ASTRA_MODEL_LABELS: Record<string, string> = {
  'gpt-6-astra': 'GPT-6 Astra',
  'gpt-6-sol': 'GPT-6 Sol',
  'gpt-6-luna': 'GPT-6 Luna',
  'gpt-5.6-sol': 'GPT-5.6 Sol',
  'gpt-5.6-terra': 'GPT-5.6 Terra',
  'gpt-5.6-luna': 'GPT-5.6 Luna',
  'gpt-5.5': 'GPT-5.5',
  'gpt-5.4-mini': 'GPT-5.4 mini',
  'claude-opus-5.5': 'Claude Opus 5.5',
  'claude-opus-5-5': 'Claude Opus 5.5',
  'claude-fable-5.1': 'Claude Fable 5.1',
  'claude-sonnet-5': 'Claude Sonnet 5',
  'claude-haiku-4.5': 'Claude Haiku 4.5',
  'gpt-5.5/5.4': 'GPT-5.5 / GPT-5.4',
  // ModelTrace 指纹库里的模型 id
  'gpt-5.4': 'GPT-5.4',
  'claude-haiku-4-5-20251001': 'Claude Haiku 4.5',
  'claude-sonnet-4-6': 'Claude Sonnet 4.6',
  'claude-opus-4-6': 'Claude Opus 4.6',
  'claude-opus-4-7': 'Claude Opus 4.7',
  'claude-opus-4-8': 'Claude Opus 4.8',
  'claude-opus-5': 'Claude Opus 5',
  other_known_external: 'other'
}

// 候选 id → 可读名；未知 id 原样返回。other_known_external 返回 'other'，由调用方换成本地化文案
export function astraModelLabel(model?: string | null): string {
  const id = (model || '').trim()
  if (!id) {
    return '?'
  }
  return ASTRA_MODEL_LABELS[id] || id
}

export const ASTRA_OTHER_MODEL = 'other_known_external'

export interface AstraEventModels {
  expected: string
  winner: string
}

// 事件 sub_status 形如 <expected>:winner_<winner>（winner 缺失时为 unknown）；
// 单模型时代的旧事件只有 winner_<winner>，预期模型固定是 GPT-6 Astra。
export function parseAstraEventSubStatus(subStatus?: string | null): AstraEventModels {
  const raw = (subStatus || '').trim()
  const marker = ':winner_'
  const idx = raw.lastIndexOf(marker)
  let expected = ''
  let winner = ''
  if (idx >= 0) {
    expected = raw.slice(0, idx)
    winner = raw.slice(idx + marker.length)
  } else if (raw.startsWith('winner_')) {
    expected = 'gpt-6-astra'
    winner = raw.slice('winner_'.length)
  }
  if (winner === 'unknown') {
    winner = ''
  }
  return { expected, winner }
}

export function getGroupRuntimeEventBadgeClass(eventType?: string | null): string {
  switch (eventType) {
    case 'down':
    case 'astra_mismatch':
    case 'modeltrace_mismatch':
    case 'sol_juice_mismatch':
      return 'badge-danger'
    case 'up':
    case 'astra_recovered':
    case 'modeltrace_recovered':
    case 'sol_juice_recovered':
      return 'badge-success'
    default:
      return 'badge-gray'
  }
}

export function formatUsd(value?: number | null, digits: number = 4): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return '-'
  }
  return `$${value.toFixed(digits)}`
}

// 按当前间隔把最近一次成本折算成每月（30 天）估算；没有样本时返回 null
export function estimateMonthlyProbeCostUsd(
  lastCostUsd?: number | null,
  intervalSeconds?: number | null
): number | null {
  if (
    lastCostUsd === null ||
    lastCostUsd === undefined ||
    !Number.isFinite(lastCostUsd) ||
    lastCostUsd <= 0 ||
    !intervalSeconds ||
    intervalSeconds <= 0
  ) {
    return null
  }
  return lastCostUsd * ((30 * 86400) / intervalSeconds)
}

export function formatMatchPercent(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return '-'
  }
  return `${(value * 100).toFixed(1)}%`
}
