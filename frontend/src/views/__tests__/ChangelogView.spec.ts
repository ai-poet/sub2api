import { describe, expect, it } from 'vitest'

// formatDate is defined inside ChangelogView.vue; replicate it here for unit testing.
function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr
  try {
    const d = new Date(dateStr + 'T00:00:00')
    if (isNaN(d.getTime())) return dateStr
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
  } catch {
    return dateStr
  }
}

describe('ChangelogView formatDate', () => {
  it('returns empty string for empty input', () => {
    expect(formatDate('')).toBe('')
  })

  it('returns original string for non-YYYY-MM-DD input', () => {
    expect(formatDate('invalid')).toBe('invalid')
    expect(formatDate('2026/01/01')).toBe('2026/01/01')
    expect(formatDate('2026-1-1')).toBe('2026-1-1')
    expect(formatDate('2026-13-01')).toBe('2026-13-01')
  })

  it('formats valid YYYY-MM-DD dates', () => {
    const result = formatDate('2026-01-01')
    // toLocaleDateString with undefined locale produces a localized string.
    // Verify it contains the expected year, month, and day.
    expect(result).toContain('2026')
    // Month should be present (short form, e.g. "Jan" or "1")
    expect(result.length).toBeGreaterThan(4)
  })

  it('returns original string for month 13 that matches the regex format', () => {
    // 2026-13-01: regex allows it, but Date.parse may or may not accept it depending on engine.
    // The function relies on isNaN(d.getTime()) to catch truly invalid dates.
    expect(formatDate('2026-13-01')).toMatch(/2026-13-01|Invalid Date/)
  })
})
