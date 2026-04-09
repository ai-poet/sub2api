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
