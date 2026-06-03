import { describe, it, expect } from 'vitest'

// Replicate the formatDate logic from ChangelogView.vue for regression testing.
// If the source implementation changes, this test should be updated to match.
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

  it('returns formatted date for valid YYYY-MM-DD', () => {
    const result = formatDate('2026-01-01')
    expect(result).toContain('2026')
    expect(result).toContain('1')
  })

  it('returns original string for invalid format', () => {
    expect(formatDate('invalid')).toBe('invalid')
  })

  it('returns original string for slash-separated date', () => {
    expect(formatDate('2026/01/01')).toBe('2026/01/01')
  })

  it('returns original string for single-digit month/day', () => {
    expect(formatDate('2026-1-1')).toBe('2026-1-1')
  })

  it('returns original string for non-YYYY-MM-DD patterns', () => {
    expect(formatDate('01-01-2026')).toBe('01-01-2026')
    expect(formatDate('2026-01')).toBe('2026-01')
    expect(formatDate('2026-01-01-extra')).toBe('2026-01-01-extra')
  })
})
