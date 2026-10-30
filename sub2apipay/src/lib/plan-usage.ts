export const PLAN_WINDOWS = ['daily', 'weekly', 'monthly'] as const;
export type PlanWindow = (typeof PLAN_WINDOWS)[number];

export interface PlanLimits {
  daily_limit_usd: number | null;
  weekly_limit_usd: number | null;
  monthly_limit_usd: number | null;
}

/** A missing limits object means unavailable data, not an unlimited plan. */
export function planWindowLimit(limits: PlanLimits | null, window: PlanWindow): number | null | undefined {
  if (!limits) return undefined;
  const value = limits[`${window}_limit_usd`];
  if (value === null || value === 0) return null;
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : undefined;
}

export type RequestEstimate =
  | { kind: 'count'; count: number }
  | { kind: 'unlimited' }
  | { kind: 'free' }
  | { kind: 'unknown' };

/** Illustrative model cost is before the group multiplier, like gateway TotalCost. */
export function estimatePlanRequests(
  limit: number | null | undefined,
  rate: number | null,
  requestCost: number,
): RequestEstimate {
  if (limit === undefined || rate === null || !Number.isFinite(rate) || rate < 0 ||
    !Number.isFinite(requestCost) || requestCost <= 0) return { kind: 'unknown' };
  if (rate === 0) return { kind: 'free' };
  if (limit === null) return { kind: 'unlimited' };
  const quotient = limit / (requestCost * rate);
  // Decimal billing amounts can land one floating-point step below an integer.
  const count = Math.floor(quotient + Number.EPSILON * Math.max(1, quotient));
  return Number.isSafeInteger(count) && count >= 0 ? { kind: 'count', count } : { kind: 'unknown' };
}
