import { describe, expect, it } from 'vitest'
import {
  computeAdjustedRate,
  discountPercent,
  formatPercent,
  formatRate,
  isValidRate,
  rowKey,
  toBatchItems
} from '../userGroupRates'

describe('userGroupRates utils (fork)', () => {
  it('builds a stable row key from user and group', () => {
    expect(rowKey({ user_id: 7, group_id: 3 })).toBe('7:3')
  })

  it('computes the new rate for each mode, rounded to 4 decimals', () => {
    expect(computeAdjustedRate(0.8, 'set', 0.75)).toBe(0.75)
    expect(computeAdjustedRate(0.8, 'factor', 1.1)).toBe(0.88)
    expect(computeAdjustedRate(0.8, 'percent', 10)).toBe(0.88)
    expect(computeAdjustedRate(0.8, 'percent', -20)).toBe(0.64)
    expect(computeAdjustedRate(0.33333, 'factor', 1)).toBe(0.3333)
    expect(computeAdjustedRate(0.8, 'reset', null)).toBeNull()
  })

  it('floors at 0.0001 and rejects values that cannot be a rate', () => {
    expect(computeAdjustedRate(0.0001, 'factor', 0.1)).toBe(0.0001)
    expect(computeAdjustedRate(0.8, 'set', 0)).toBeUndefined()
    expect(computeAdjustedRate(0.8, 'percent', -100)).toBeUndefined()
    expect(computeAdjustedRate(0.8, 'factor', -1)).toBeUndefined()
    expect(computeAdjustedRate(0.8, 'set', null)).toBeUndefined()
    expect(computeAdjustedRate(0.8, 'set', Number.NaN)).toBeUndefined()
    expect(computeAdjustedRate(0.8, 'set', 2_000_000)).toBeUndefined()
  })

  it('validates rates against the DECIMAL(10,4) range', () => {
    expect(isValidRate(0.8)).toBe(true)
    expect(isValidRate(0.0001)).toBe(true)
    expect(isValidRate(0.00004)).toBe(false)
    expect(isValidRate(0)).toBe(false)
    expect(isValidRate('0.8')).toBe(false)
    expect(isValidRate(1_000_000)).toBe(false)
  })

  it('reports the change against the group rate in percent', () => {
    expect(discountPercent(0.8, 1)).toBe(-20)
    expect(discountPercent(1.2, 1)).toBe(20)
    expect(discountPercent(1, 1.2)).toBe(-16.7)
    expect(discountPercent(0.8, 0)).toBeNull()
    expect(formatPercent(-20)).toBe('-20%')
    expect(formatPercent(20)).toBe('+20%')
    expect(formatPercent(0)).toBe('0%')
    expect(formatPercent(null)).toBe('-')
    expect(formatRate(0.123456)).toBe('0.1235')
    expect(formatRate(null)).toBe('-')
  })

  it('turns a preview into batch items, skipping invalid and unchanged rows', () => {
    const a = { user_id: 1, group_id: 3 }
    const b = { user_id: 2, group_id: 3 }
    const c = { user_id: 3, group_id: 3 }
    const d = { user_id: 4, group_id: 3 }
    expect(
      toBatchItems([
        { row: a, from: 0.8, to: 0.88 },
        { row: b, from: 0.8, to: undefined },
        { row: c, from: 0.8, to: 0.8 },
        { row: d, from: 0.8, to: null }
      ])
    ).toEqual([
      { user_id: 1, group_id: 3, rate_multiplier: 0.88 },
      { user_id: 4, group_id: 3, rate_multiplier: null }
    ])
  })
})
