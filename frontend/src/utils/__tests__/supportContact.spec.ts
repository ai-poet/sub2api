import { describe, expect, it } from 'vitest'
import { supportContactHref } from '../supportContact'

describe('supportContactHref', () => {
  it('links a bare email address with mailto', () => {
    expect(supportContactHref('  support@example.com ')).toBe('mailto:support@example.com')
  })

  it('links a bare http(s) address', () => {
    expect(supportContactHref('https://t.me/example')).toBe('https://t.me/example')
  })

  it('keeps free text and other schemes as plain text', () => {
    expect(supportContactHref('QQ: 123456789')).toBeNull()
    expect(supportContactHref('邮箱 support@example.com')).toBeNull()
    expect(supportContactHref('javascript:alert(1)')).toBeNull()
    expect(supportContactHref('https://example.com and more')).toBeNull()
    expect(supportContactHref('')).toBeNull()
    expect(supportContactHref(undefined)).toBeNull()
  })
})
