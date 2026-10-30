import { describe, expect, it } from 'vitest';
import { estimatePlanRequests, planWindowLimit } from '@/lib/plan-usage';

describe('plan quota estimates', () => {
  it('applies the billing multiplier and rounds down without combining windows', () => {
    expect(estimatePlanRequests(10, 0.5, 0.3)).toEqual({ kind: 'count', count: 66 });
    expect(estimatePlanRequests(10, 2, 0.3)).toEqual({ kind: 'count', count: 16 });
    expect(estimatePlanRequests(0.01, 1, 1)).toEqual({ kind: 'count', count: 0 });
    expect(estimatePlanRequests(0.3, 1, 0.1)).toEqual({ kind: 'count', count: 3 });
    expect(estimatePlanRequests(1.2, 1, 0.1)).toEqual({ kind: 'count', count: 12 });
  });

  it('distinguishes an uncapped window from unavailable group data', () => {
    expect(planWindowLimit(null, 'daily')).toBeUndefined();
    expect(planWindowLimit({ daily_limit_usd: null, weekly_limit_usd: 0, monthly_limit_usd: 50 }, 'daily')).toBeNull();
    expect(planWindowLimit({ daily_limit_usd: null, weekly_limit_usd: 0, monthly_limit_usd: 50 }, 'weekly')).toBeNull();
    expect(planWindowLimit({ daily_limit_usd: null, weekly_limit_usd: 0, monthly_limit_usd: 50 }, 'monthly')).toBe(50);
    expect(estimatePlanRequests(null, 1, 0.1)).toEqual({ kind: 'unlimited' });
    expect(estimatePlanRequests(undefined, 1, 0.1)).toEqual({ kind: 'unknown' });
  });

  it('does not turn missing rates or invalid input into a request promise', () => {
    expect(estimatePlanRequests(10, null, 0.1)).toEqual({ kind: 'unknown' });
    expect(estimatePlanRequests(10, -1, 0.1)).toEqual({ kind: 'unknown' });
    for (const cost of [0, -1, NaN, Infinity, Number.MIN_VALUE]) {
      expect(estimatePlanRequests(10, 1, cost)).toEqual({ kind: 'unknown' });
    }
    expect(estimatePlanRequests(10, 0, 0.1)).toEqual({ kind: 'free' });
  });
});
