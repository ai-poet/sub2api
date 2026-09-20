/**
 * 工单（fork 本地功能）：状态 / 分类的展示辅助。
 *
 * 标签一律通过静态字面量 t('...') 取——i18n 完整性检查只扫描字面量键，
 * 拼接键（t('tickets.status.' + s)）会漏检。
 */

import type { TicketCategory, TicketStatus } from '@/api/tickets'

type TranslateFn = (key: string, ...args: any[]) => string

export const TICKET_CATEGORIES: TicketCategory[] = ['account', 'billing', 'api', 'other']
export const TICKET_STATUSES: TicketStatus[] = ['open', 'replied', 'closed']
export const TICKET_TITLE_MAX = 200
export const TICKET_BODY_MAX = 5000
export const TICKET_OPEN_LIMIT = 5

export function ticketStatusLabel(t: TranslateFn, status: string): string {
  switch (status) {
    case 'open':
      return t('tickets.status.open')
    case 'replied':
      return t('tickets.status.replied')
    case 'closed':
      return t('tickets.status.closed')
    default:
      return status
  }
}

export function ticketCategoryLabel(t: TranslateFn, category: string): string {
  switch (category) {
    case 'account':
      return t('tickets.category.account')
    case 'billing':
      return t('tickets.category.billing')
    case 'api':
      return t('tickets.category.api')
    case 'other':
      return t('tickets.category.other')
    default:
      return category
  }
}

export function ticketStatusBadgeClass(status: string): string {
  const base = 'inline-flex items-center whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-medium'
  switch (status) {
    case 'open':
      return `${base} bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300`
    case 'replied':
      return `${base} bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300`
    case 'closed':
      return `${base} bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300`
    default:
      return `${base} bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300`
  }
}

/** 解析 ?id= 深链（Server酱³ 推送里的链接），非法时返回 null。 */
export function parseTicketDeepLinkId(raw: unknown): number | null {
  const text = typeof raw === 'string' ? raw : Array.isArray(raw) ? String(raw[0] ?? '') : ''
  const id = Number.parseInt(text, 10)
  return Number.isFinite(id) && id > 0 ? id : null
}
