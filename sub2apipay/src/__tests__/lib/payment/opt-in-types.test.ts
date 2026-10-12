import { describe, expect, it } from 'vitest';
import {
  instanceSupportsPaymentType,
  isOptInPaymentType,
  parsePaymentTypeList,
  withoutOptInPaymentTypes,
} from '@/lib/payment/opt-in-types';

describe('opt-in payment types', () => {
  it('treats PayNow as opt-in and leaves the existing types alone', () => {
    expect(isOptInPaymentType('paynow')).toBe(true);
    for (const type of ['alipay', 'wxpay', 'usdt.plasma', 'usdt.polygon', 'usdc.solana', 'bank', 'stripe']) {
      expect(isOptInPaymentType(type)).toBe(false);
    }
  });

  it('parses comma lists', () => {
    expect(parsePaymentTypeList(' alipay, ,paynow ')).toEqual(['alipay', 'paynow']);
    expect(parsePaymentTypeList(null)).toEqual([]);
    expect(parsePaymentTypeList('')).toEqual([]);
  });

  it('lets an empty instance list cover every type except opt-in ones', () => {
    expect(instanceSupportsPaymentType('', 'alipay')).toBe(true);
    expect(instanceSupportsPaymentType(null, 'bank')).toBe(true);
    expect(instanceSupportsPaymentType('', 'paynow')).toBe(false);
    expect(instanceSupportsPaymentType(' , ', 'paynow')).toBe(false);
  });

  it('honours an explicit instance list for every type', () => {
    expect(instanceSupportsPaymentType('alipay,paynow', 'paynow')).toBe(true);
    expect(instanceSupportsPaymentType('alipay,paynow', 'alipay')).toBe(true);
    expect(instanceSupportsPaymentType('alipay', 'paynow')).toBe(false);
    expect(instanceSupportsPaymentType('alipay', 'wxpay')).toBe(false);
  });

  it('drops opt-in types from select-all defaults', () => {
    expect(withoutOptInPaymentTypes(['alipay', 'wxpay', 'bank', 'paynow'])).toEqual(['alipay', 'wxpay', 'bank']);
  });
});
