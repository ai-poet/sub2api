import { describe, expect, it } from 'vitest'
import { clearRenderedMarkdownCache, renderSafeMarkdown } from '@/utils/markdown'

describe('renderSafeMarkdown', () => {
  it('renders common Markdown blocks', () => {
    const html = renderSafeMarkdown('## 0.1.78\n\n- Added **Cloud** route\n- Fixed `@` mention')

    expect(html).toContain('<h2>0.1.78</h2>')
    expect(html).toContain('<li>Added <strong>Cloud</strong> route</li>')
    expect(html).toContain('<code>@</code>')
  })

  it('sanitizes unsafe HTML', () => {
    const html = renderSafeMarkdown('[safe](javascript:alert(1))\n\n<img src=x onerror=alert(1)>')

    expect(html).not.toContain('javascript:')
    expect(html).not.toContain('onerror')
  })

  it('returns an empty string for empty content', () => {
    expect(renderSafeMarkdown('')).toBe('')
  })

  it('can clear cached rendered output without changing rendering behavior', () => {
    const source = '## Cached'
    const first = renderSafeMarkdown(source)

    clearRenderedMarkdownCache()

    expect(renderSafeMarkdown(source)).toBe(first)
  })
})
