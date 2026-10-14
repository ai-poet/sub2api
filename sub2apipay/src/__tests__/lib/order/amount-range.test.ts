import { describe, expect, it } from 'vitest';
import { amountRangePlaceholder, resolveAmountMax } from '@/lib/order/amount-range';

describe('resolveAmountMax', () => {
  it('没有渠道上限时用全局最大到账金额', () => {
    expect(resolveAmountMax(5000)).toBe(5000);
    expect(resolveAmountMax(5000, 0)).toBe(5000);
  });

  it('渠道上限更小时按渠道上限收紧', () => {
    expect(resolveAmountMax(5000, 1000)).toBe(1000);
  });

  it('渠道上限更大时不放宽全局最大到账金额', () => {
    // 当天剩余日额度 10000 换算出的上限不能越过后台设的 5000
    expect(resolveAmountMax(5000, 10000)).toBe(5000);
  });

  it('全局最大到账金额为 0 时不限，只剩渠道上限', () => {
    expect(resolveAmountMax(0)).toBe(0);
    expect(resolveAmountMax(0, 0)).toBe(0);
    expect(resolveAmountMax(0, 800)).toBe(800);
  });
});

describe('amountRangePlaceholder', () => {
  it('只列出设置了的那一侧', () => {
    expect(amountRangePlaceholder(10, 5000)).toBe('10 - 5000');
    expect(amountRangePlaceholder(10, 0)).toBe('≥ 10');
    expect(amountRangePlaceholder(0, 5000)).toBe('≤ 5000');
    expect(amountRangePlaceholder(0, 0)).toBe('');
  });
});
