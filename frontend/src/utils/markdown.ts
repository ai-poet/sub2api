import { marked } from 'marked'
import DOMPurify from 'dompurify'

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedMarkdownCache = new Map<string, string>()

export function renderSafeMarkdown(content: string): string {
  if (!content) return ''
  const cached = renderedMarkdownCache.get(content)
  if (cached !== undefined) return cached

  const html = marked.parse(content) as string
  const sanitized = DOMPurify.sanitize(html)
  renderedMarkdownCache.set(content, sanitized)
  return sanitized
}

export function clearRenderedMarkdownCache(): void {
  renderedMarkdownCache.clear()
}
