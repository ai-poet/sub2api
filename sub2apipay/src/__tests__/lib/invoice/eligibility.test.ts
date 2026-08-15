import { describe, expect, it } from 'vitest';
import { evaluateInvoiceEligibility, type InvoiceEligibilityOrder } from '@/lib/invoice/eligibility';
import { INVOICE_STATUS } from '@/lib/invoice/types';

const NOW = new Date('2026-08-15T00:00:00Z');

function order(overrides: Partial<InvoiceEligibilityOrder> = {}): InvoiceEligibilityOrder {
  return {
    status: 'COMPLETED',
    paymentType: 'alipay',
    paidAt: new Date('2026-08-01T00:00:00Z'),
    payAmount: '128.00',
    refundAmount: null,
    ...overrides,
  };
}

const DEFAULTS = { featureEnabled: true, maxAgeDays: 180, now: NOW };

describe('evaluateInvoiceEligibility', () => {
  it('accepts a completed, paid, CNY order', () => {
    expect(evaluateInvoiceEligibility(order(), DEFAULTS)).toEqual({ eligible: true });
  });

  it('rejects everything when the feature is disabled', () => {
    expect(evaluateInvoiceEligibility(order(), { ...DEFAULTS, featureEnabled: false })).toEqual({
      eligible: false,
      reason: 'FEATURE_DISABLED',
    });
  });

  it.each(['PENDING', 'PAID', 'RECHARGING', 'FAILED', 'CANCELLED', 'EXPIRED'])(
    'rejects order status %s',
    (status) => {
      expect(evaluateInvoiceEligibility(order({ status }), DEFAULTS)).toEqual({
        eligible: false,
        reason: 'NOT_COMPLETED',
      });
    },
  );

  it('rejects a COMPLETED order that somehow has no paidAt', () => {
    expect(evaluateInvoiceEligibility(order({ paidAt: null }), DEFAULTS)).toEqual({
      eligible: false,
      reason: 'NOT_COMPLETED',
    });
  });

  // USDT/USDC settle in USD, so there is no CNY figure to put on a 发票.
  it.each(['usdt.plasma', 'usdt.polygon', 'usdc.solana'])('rejects stablecoin payment type %s', (paymentType) => {
    expect(evaluateInvoiceEligibility(order({ paymentType }), DEFAULTS)).toEqual({
      eligible: false,
      reason: 'STABLECOIN_NOT_SUPPORTED',
    });
  });

  // Legacy orders predate the fee/payAmount columns. Never back-compute the CNY
  // figure from the USD credit — refuse and send the user to support.
  it.each([null, undefined, 0, '0.00'])('rejects missing or zero payAmount (%s)', (payAmount) => {
    expect(evaluateInvoiceEligibility(order({ payAmount }), DEFAULTS)).toEqual({
      eligible: false,
      reason: 'MISSING_PAY_AMOUNT',
    });
  });

  it('rejects a refunded order', () => {
    expect(evaluateInvoiceEligibility(order({ refundAmount: '10.00' }), DEFAULTS)).toEqual({
      eligible: false,
      reason: 'ORDER_REFUNDED',
    });
  });

  it('rejects an order older than the invoicing window', () => {
    const stale = order({ paidAt: new Date('2026-01-01T00:00:00Z') });
    expect(evaluateInvoiceEligibility(stale, DEFAULTS)).toEqual({ eligible: false, reason: 'TOO_OLD' });
  });

  it('accepts an old order when maxAgeDays is 0 (unlimited)', () => {
    const stale = order({ paidAt: new Date('2020-01-01T00:00:00Z') });
    expect(evaluateInvoiceEligibility(stale, { ...DEFAULTS, maxAgeDays: 0 })).toEqual({ eligible: true });
  });

  it.each([INVOICE_STATUS.PENDING, INVOICE_STATUS.ISSUED])(
    'rejects a re-request while an invoice is %s',
    (existingStatus) => {
      expect(evaluateInvoiceEligibility(order(), { ...DEFAULTS, existingStatus })).toEqual({
        eligible: false,
        reason: 'ALREADY_REQUESTED',
      });
    },
  );

  it.each([INVOICE_STATUS.REJECTED, INVOICE_STATUS.CANCELLED])(
    'allows re-requesting after %s',
    (existingStatus) => {
      expect(evaluateInvoiceEligibility(order(), { ...DEFAULTS, existingStatus })).toEqual({ eligible: true });
    },
  );

  // The feature flag is checked before anything else so a disabled deployment
  // never leaks per-order reasons.
  it('reports FEATURE_DISABLED even for an otherwise ineligible order', () => {
    expect(
      evaluateInvoiceEligibility(order({ status: 'PENDING' }), { ...DEFAULTS, featureEnabled: false }),
    ).toEqual({ eligible: false, reason: 'FEATURE_DISABLED' });
  });
});
