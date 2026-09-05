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

// ==================== 纯 Sol 验证（Juice 指纹探测） ====================

export type NormalizedSolJuiceStatus = 'pass' | 'mismatch' | 'inconclusive' | 'unknown'

export function normalizeSolJuiceStatus(status?: string | null): NormalizedSolJuiceStatus {
  switch (status) {
    case 'pass':
    case 'mismatch':
    case 'inconclusive':
      return status
    default:
      return 'unknown'
  }
}

export function getSolJuiceBadgeClass(status?: string | null): string {
  switch (normalizeSolJuiceStatus(status)) {
    case 'pass':
      return 'badge-success'
    case 'mismatch':
      return 'badge-danger'
    case 'inconclusive':
      return 'badge-warning'
    default:
      return 'badge-gray'
  }
}

export function isSolJuiceEvent(eventType?: string | null): boolean {
  return eventType === 'sol_juice_mismatch' || eventType === 'sol_juice_recovered'
}

export function getGroupRuntimeEventBadgeClass(eventType?: string | null): string {
  switch (eventType) {
    case 'down':
    case 'sol_juice_mismatch':
      return 'badge-danger'
    case 'up':
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
export function estimateSolJuiceMonthlyCostUsd(
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
