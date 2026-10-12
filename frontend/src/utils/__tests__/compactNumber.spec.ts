import { describe, expect, it } from 'vitest'

import { compactFormatterFor, formatCompact } from '../compactNumber'

describe('compactNumber', () => {
  it('formats with the value’s own unit', () => {
    expect(formatCompact(950)).toBe('950')
    expect(formatCompact(1234)).toBe('1.2K')
    expect(formatCompact(3_500_000)).toBe('3.5M')
    expect(formatCompact(1_200_000_000)).toBe('1.2B')
  })

  it('locks the unit to the target so counting up never jumps between K, M and B', () => {
    const format = compactFormatterFor(1_200_000_000)
    expect(format(0)).toBe('0.0B')
    expect(format(600_000_000)).toBe('0.6B')
    expect(format(1_200_000_000)).toBe('1.2B')

    const small = compactFormatterFor(800)
    expect(small(12.4)).toBe('12')
  })
})
