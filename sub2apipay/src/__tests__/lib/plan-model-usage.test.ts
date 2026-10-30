import { describe, expect, it } from 'vitest';
import { modelRequestCost, modelWindowEstimate, type PlanModel, type PlanModelPricing } from '@/lib/plan-model-usage';

const pricing: PlanModelPricing = {
  input_per_mtok_usd: 3, output_per_mtok_usd: 15, cache_read_per_mtok_usd: 0.3,
  per_request_usd: null, per_image_usd: null, per_second_usd: null,
};
const model: PlanModel = {
  model: 'claude-sonnet', display_name: 'Claude Sonnet', billing_mode: 'token',
  effective_pricing_usd: pricing,
  pricing_details: { intervals: [], media_tiers: [], has_long_context_multiplier: false, long_context_input_threshold: 0 },
};
const usage = { inputTokens: 10_000, outputTokens: 2_000 };
const limits = { daily_limit_usd: 10, weekly_limit_usd: 50, monthly_limit_usd: 150 };

describe('per-model plan usage', () => {
  it('uses each model’s effective prices once, with separate shared windows', () => {
    expect(modelRequestCost(model, usage)).toBeCloseTo(0.06);
    expect(modelWindowEstimate(model, limits, 'daily', usage)).toEqual({ kind: 'count', count: 166 });
    expect(modelWindowEstimate(model, limits, 'monthly', usage)).toEqual({ kind: 'count', count: 2500 });
    const cheaper = { ...model, effective_pricing_usd: { ...pricing, input_per_mtok_usd: 1.5, output_per_mtok_usd: 7.5 } };
    expect(modelRequestCost(cheaper, usage)).toBeCloseTo(0.03);
    expect(modelWindowEstimate(cheaper, limits, 'daily', usage)).toEqual({ kind: 'count', count: 333 });
  });

  it('selects the applicable context tier and refuses gaps or missing prices', () => {
    const tiered = { ...model, pricing_details: { ...model.pricing_details, intervals: [
      { ...pricing, min_tokens: 0, max_tokens: 200_000 },
      { ...pricing, min_tokens: 200_000, max_tokens: null, input_per_mtok_usd: 6, output_per_mtok_usd: 22.5 },
    ] } };
    expect(modelRequestCost(tiered, { inputTokens: 210_000, outputTokens: 2_000 })).toBeCloseTo(1.305);
    expect(modelRequestCost(tiered, { inputTokens: 200_000, outputTokens: 2_000 })).toBeCloseTo(0.63);
    expect(modelRequestCost(tiered, { inputTokens: 200_001, outputTokens: 2_000 })).toBeCloseTo(1.245006);
    expect(modelRequestCost({ ...model, effective_pricing_usd: { ...pricing, output_per_mtok_usd: null } }, usage)).toBeNull();
    expect(modelWindowEstimate(model, null, 'daily', usage)).toEqual({ kind: 'unknown' });
    expect(modelWindowEstimate(model, { ...limits, monthly_limit_usd: null }, 'monthly', usage)).toEqual({ kind: 'unlimited' });
  });

  it('distinguishes request, image and video units from token-based prices', () => {
    const perRequest = { ...model, billing_mode: 'per_request', effective_pricing_usd: { ...pricing, per_request_usd: 0.2 } };
    expect(modelRequestCost(perRequest, usage)).toBe(0.2);
    expect(modelWindowEstimate(perRequest, limits, 'daily', usage)).toEqual({ kind: 'count', count: 50 });
    expect(modelRequestCost({ ...model, billing_mode: 'image', effective_pricing_usd: { ...pricing, per_image_usd: 0.1 } }, usage)).toBe(0.1);
    expect(modelRequestCost({ ...model, billing_mode: 'video', effective_pricing_usd: { ...pricing, per_second_usd: 0.4 } }, usage)).toBe(0.4);
    expect(modelRequestCost({ ...perRequest, pricing_details: { ...model.pricing_details, intervals: [{ ...pricing, min_tokens: 0, max_tokens: null }] } }, usage)).toBeNull();
  });

  it('never invents prices for empty or invalid token assumptions', () => {
    for (const inputTokens of [NaN, -1, 1.5, Infinity]) {
      expect(modelWindowEstimate(model, limits, 'daily', { inputTokens, outputTokens: 2_000 })).toEqual({ kind: 'unknown' });
    }
    expect(modelRequestCost(model, { inputTokens: 0, outputTokens: 0 })).toBeNull();
  });
});
