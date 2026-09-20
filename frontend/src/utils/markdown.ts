import { marked } from 'marked'
import DOMPurify from 'dompurify'

marked.setOptions({
  breaks: true,
  gfm: true,
})

// 同一段正文在两种净化配置下的结果不同，所以两套配置各用各的缓存，
// 免得为了区分而往缓存键里拼分隔符（正文本身可能含任何字符）。
const renderedMarkdownCache = new Map<string, string>()
const renderedMarkdownBlobCache = new Map<string, string>()

/**
 * DOMPurify 默认的 ALLOWED_URI_REGEXP 不含 blob:，而工单图片附件只能走 blob: URL：
 * 附件内容端点和其它接口一样只认 Authorization 头，浏览器给 <img src> 发请求时不带
 * 这个头，所以必须先用 apiClient 取回字节、再 URL.createObjectURL 交给渲染。
 *
 * 这里在默认方案之上只多放一个 blob，其余 scheme 与 DOMPurify 默认一致。blob: URL 按源
 * 隔离且只能由本源的 JS 创建——正文里伪造的 blob:https://evil/… 在本源解析不到，只会是
 * 一张坏图，拿不到任何数据。
 */
const BLOB_ALLOWED_URI_REGEXP =
  /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|sms|cid|xmpp|blob):|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i

export interface RenderMarkdownOptions {
  /** 允许 <img src="blob:…">。只给工单线程用，别的调用点保持 DOMPurify 默认。 */
  allowBlobImages?: boolean
}

export function renderSafeMarkdown(content: string, options: RenderMarkdownOptions = {}): string {
  if (!content) return ''
  const allowBlob = options.allowBlobImages === true
  const cache = allowBlob ? renderedMarkdownBlobCache : renderedMarkdownCache
  const cached = cache.get(content)
  if (cached !== undefined) return cached

  const html = marked.parse(content) as string
  const sanitized = allowBlob
    ? DOMPurify.sanitize(html, { ALLOWED_URI_REGEXP: BLOB_ALLOWED_URI_REGEXP })
    : DOMPurify.sanitize(html)
  cache.set(content, sanitized)
  return sanitized
}

export function clearRenderedMarkdownCache(): void {
  renderedMarkdownCache.clear()
  renderedMarkdownBlobCache.clear()
}
