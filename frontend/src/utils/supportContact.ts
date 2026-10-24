/**
 * 客服联系方式（管理后台「客服联系方式」，公开设置 contact_info）的链接化（fork 本地）。
 *
 * contact_info 是自由文本（"QQ: 123456"、邮箱、网址都有人填），只有整段就是一个邮箱
 * 或一个 http(s) 网址时才变成链接，其余照原文显示。
 */

const EMAIL_RE = /^[^\s@<>()"']+@[^\s@<>()"']+\.[^\s@<>()"']+$/

export function supportContactHref(value: string | null | undefined): string | null {
  const text = (value ?? '').trim()
  if (!text) return null
  if (EMAIL_RE.test(text)) return `mailto:${text}`
  if (/^https?:\/\//i.test(text) && !/\s/.test(text)) {
    try {
      const url = new URL(text)
      return url.protocol === 'http:' || url.protocol === 'https:' ? url.href : null
    } catch {
      return null
    }
  }
  return null
}
