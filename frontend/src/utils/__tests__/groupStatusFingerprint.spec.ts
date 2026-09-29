import { describe, expect, it } from 'vitest'
import { normalizeAstraStateStatus } from '../groupStatus'

describe('normalizeAstraStateStatus', () => {
  it('keeps the stable verdict for Juice like the former pure-Sol check', () => {
    // 已稳定一致：单次不符（等复测）或读数无法判定都不改变绿色
    expect(normalizeAstraStateStatus({ method: 'sol_juice', stable_status: 'pass', verdict: 'mismatch' })).toBe('pass')
    expect(normalizeAstraStateStatus({ method: 'sol_juice', stable_status: 'pass', verdict: 'insufficient' })).toBe('pass')
    expect(normalizeAstraStateStatus({ method: 'sol_juice', stable_status: 'mismatch', verdict: 'insufficient' })).toBe('mismatch')
    // 还没有稳定结论时看最近一次
    expect(normalizeAstraStateStatus({ method: 'sol_juice', stable_status: '', verdict: 'match' })).toBe('pass')
    expect(normalizeAstraStateStatus({ method: 'sol_juice', stable_status: '', verdict: 'insufficient' })).toBe('insufficient')
  })

  it('shows the latest result for meow and ModelTrace', () => {
    expect(normalizeAstraStateStatus({ method: 'meow', stable_status: 'pass', verdict: 'mismatch' })).toBe('suspect')
    expect(normalizeAstraStateStatus({ method: 'modeltrace', stable_status: 'pass', verdict: 'insufficient' })).toBe('insufficient')
    expect(normalizeAstraStateStatus({ stable_status: 'mismatch', verdict: 'match' })).toBe('mismatch')
  })
})
