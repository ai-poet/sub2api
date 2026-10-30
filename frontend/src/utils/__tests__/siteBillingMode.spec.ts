import { describe, it, expect } from 'vitest'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'

describe('resolveSiteBillingMode', () => {
  it('defaults to recharge & subscription when settings are missing or empty', () => {
    expect(resolveSiteBillingMode(undefined)).toBe('recharge_and_subscription')
    expect(resolveSiteBillingMode(null)).toBe('recharge_and_subscription')
    expect(resolveSiteBillingMode({})).toBe('recharge_and_subscription')
  })

  it('derives the purchase label mode only from the subscription switch', () => {
    expect(resolveSiteBillingMode({ subscription_enabled: true })).toBe('recharge_and_subscription')
    expect(resolveSiteBillingMode({ subscription_enabled: false })).toBe('recharge_only')
  })
})
