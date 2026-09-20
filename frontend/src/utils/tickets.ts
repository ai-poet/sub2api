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
/** 前端预检与后端一致：单张图片不超过 5MiB */
export const TICKET_ATTACHMENT_MAX_SIZE = 5 * 1024 * 1024

export type TicketSide = 'user' | 'admin'

/** 消息体里图片附件的 Markdown 写法：![image](ticket-attachment://<key>) */
export function ticketAttachmentMarkdown(key: string): string {
  return `![image](ticket-attachment://${key})`
}

/**
 * 附件内容端点的路径（用户侧与客服侧各有一套鉴权路由），给 apiClient 用。
 *
 * 注意这里只回路径、不回可以直接挂到 <img src> 上的地址：该端点和其它接口一样只认
 * Authorization 头，而浏览器给 <img> 发请求时不带这个头，同源地址直挂必然 401。
 * 取字节要走 api 层的 fetchAttachment，再由 useTicketAttachmentImages 换成 blob: URL。
 */
export function ticketAttachmentContentPath(side: TicketSide): string {
  return side === 'admin' ? '/admin/tickets/attachments/content' : '/tickets/attachments/content'
}

/** 附件还没取回来时的占位：1×1 透明 PNG，免得先闪一张裂图 */
export const TICKET_ATTACHMENT_PLACEHOLDER =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII='

// key 字符集 [A-Za-z0-9/._-]，且后面必须跟 Markdown 图片 URL 的结束符（）/ 空白 / 文本末尾），
// 否则（如 ticket-attachment://a?b）整段不改写。
const TICKET_ATTACHMENT_URL_RE = /ticket-attachment:\/\/([A-Za-z0-9/._-]+)(?=[\s)]|$)/g

/** 正文里引用到的全部附件 key（去重），用来决定要预取哪些图。 */
export function ticketAttachmentKeys(markdown: string): string[] {
  if (!markdown) return []
  const keys = new Set<string>()
  for (const match of markdown.matchAll(TICKET_ATTACHMENT_URL_RE)) keys.add(match[1])
  return [...keys]
}

/**
 * 渲染前把 ticket-attachment://<key> 换成 resolve 给出的地址（已取回的 blob: URL，
 * 或未就绪时的占位）。必须在交给 MarkdownRenderer 之前调用：DOMPurify 会剥掉自定义 scheme。
 * resolve 返回 undefined 表示这张图取不回来，保持原样——渲染成没有 src 的 img，alt 仍可见。
 */
export function rewriteTicketAttachmentUrls(
  markdown: string,
  resolve: (key: string) => string | undefined
): string {
  if (!markdown) return markdown
  return markdown.replace(TICKET_ATTACHMENT_URL_RE, (match, key: string) => resolve(key) ?? match)
}

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
