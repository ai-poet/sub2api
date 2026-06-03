import { describe, it, expect } from 'vitest'

// Mirror of the formatDate implementation from ChangelogView.vue
// for adversarial validation without modifying the component source.
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

describe('formatDate adversarial tests', () => {
  it('returns empty string for empty input', () => {
    expect(formatDate('')).toBe('')
  })

  it('returns original string for non-YYYY-MM-DD format', () => {
    expect(formatDate('invalid')).toBe('invalid')
    expect(formatDate('2026/01/01')).toBe('2026/01/01')
    expect(formatDate('2026-1-1')).toBe('2026-1-1')
    expect(formatDate('01-01-2026')).toBe('01-01-2026')
  })

  it('returns original string for out-of-range values that pass regex', () => {
    // Regex passes but Date.parse would be invalid
    expect(formatDate('2026-13-01')).toBe('2026-13-01')
    expect(formatDate('2026-01-32')).toBe('2026-01-32')
  })

  it('formats valid YYYY-MM-DD date to localized string', () => {
    const result = formatDate('2026-01-15')
    expect(result).not.toBe('2026-01-15')
    expect(result).toContain('2026')
  })

  it('formats date that passes regex even if calendar-invalid (JS Date auto-overflows)', () => {
    // 2023-02-29 passes the regex; JS Date auto-overflows to Mar 1.
    // The function does not validate calendar correctness — that is backend's job.
    const result = formatDate('2023-02-29')
    expect(result).not.toBe('2023-02-29')
    expect(result).toContain('2023')
  })

  it('formats valid leap-year date', () => {
    const result = formatDate('2024-02-29')
    expect(result).not.toBe('2024-02-29')
    expect(result).toContain('2024')
  })
})
