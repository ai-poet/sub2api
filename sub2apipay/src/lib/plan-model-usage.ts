import { estimatePlanRequests, planWindowLimit, type PlanLimits, type PlanWindow } from './plan-usage';

export interface PlanModelPricing {
  input_per_mtok_usd: number | null;
  output_per_mtok_usd: number | null;
  cache_read_per_mtok_usd: number | null;
  per_request_usd: number | null;
  per_image_usd: number | null;
  per_second_usd: number | null;
}

export interface PlanModel {
  model: string;
  display_name: string;
  billing_mode: string;
  /** Already includes the group rate. Do not multiply again. */
  effective_pricing_usd: PlanModelPricing;
  pricing_details: {
    intervals: Array<PlanModelPricing & { min_tokens: number; max_tokens: number | null }>;
    media_tiers: Array<{ tier: string; effective_usd: number | null; is_default_tier: boolean }>;
    has_long_context_multiplier: boolean;
    long_context_input_threshold: number;
  };
}

export interface ModelUsageAssumptions {
  inputTokens: number;
  outputTokens: number;
}

export function modelRequestCost(model: PlanModel, usage: ModelUsageAssumptions): number | null {
  const valid = (value: number | null | undefined): value is number =>
    typeof value === 'number' && Number.isFinite(value) && value >= 0;
  const pricing = model.effective_pricing_usd;
  if (model.billing_mode === 'per_request') {
    // A tiered request price needs the biller's tier-selection rules.
    if (model.pricing_details.intervals.length > 0) return null;
    return valid(pricing.per_request_usd) ? pricing.per_request_usd : null;
  }
  if (model.billing_mode === 'image' || model.billing_mode === 'video') {
    const price = model.billing_mode === 'image' ? pricing.per_image_usd : pricing.per_second_usd;
    return valid(price) ? price : null;
  }
  if (model.billing_mode !== 'token') return null;
  if (!Number.isSafeInteger(usage.inputTokens) || usage.inputTokens < 0 ||
      !Number.isSafeInteger(usage.outputTokens) || usage.outputTokens < 0 ||
      usage.inputTokens + usage.outputTokens === 0) return null;
  const tiers = model.pricing_details.intervals;
  // Gateway context schedules use (min, max], including the lower-price tier
  // at the exact threshold. Zero-input requests use the first tier.
  const tier = tiers.find((row) => (usage.inputTokens > row.min_tokens ||
    (usage.inputTokens === 0 && row.min_tokens === 0)) &&
    (row.max_tokens === null || usage.inputTokens <= row.max_tokens));
  if (tiers.length > 0 && !tier) return null;
  const prices = tier ?? pricing;
  if (!valid(prices.input_per_mtok_usd) || !valid(prices.output_per_mtok_usd)) return null;
  const cost = (usage.inputTokens * prices.input_per_mtok_usd +
    usage.outputTokens * prices.output_per_mtok_usd) / 1_000_000;
  return Number.isFinite(cost) ? cost : null;
}

export function modelWindowEstimate(
  model: PlanModel, limits: PlanLimits | null, window: PlanWindow, usage: ModelUsageAssumptions,
) {
  const cost = modelRequestCost(model, usage);
  if (cost === null) return { kind: 'unknown' } as const;
  // Zero effective price is explicit; absent prices remain unknown.
  return estimatePlanRequests(planWindowLimit(limits, window), cost === 0 ? 0 : 1, cost === 0 ? 1 : cost);
}
