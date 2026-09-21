import { describe, expect, it } from 'vitest';
import { formatUsageUsd, usageWindowRatio, usageWindowTone } from '@/lib/subscription-utils';

describe('formatUsageUsd', () => {
  it('keeps two decimals for ordinary amounts', () => {
    expect(formatUsageUsd(5)).toBe('5.00');
    expect(formatUsageUsd(0)).toBe('0.00');
    expect(formatUsageUsd(12.346)).toBe('12.35');
  });

  it('shows four decimals for sub-cent usage so small test requests stay visible', () => {
    expect(formatUsageUsd(0.0034)).toBe('0.0034');
    expect(formatUsageUsd(0.0099)).toBe('0.0099');
    expect(formatUsageUsd(0.01)).toBe('0.01');
  });

  it('treats missing or invalid values as zero', () => {
    expect(formatUsageUsd(null)).toBe('0.00');
    expect(formatUsageUsd(undefined)).toBe('0.00');
    expect(formatUsageUsd(Number.NaN)).toBe('0.00');
  });
});

describe('usageWindowRatio', () => {
  it('returns null when the window has no limit', () => {
    expect(usageWindowRatio(3, null)).toBeNull();
    expect(usageWindowRatio(3, undefined)).toBeNull();
    expect(usageWindowRatio(3, 0)).toBeNull();
  });

  it('clamps the ratio into [0, 1]', () => {
    expect(usageWindowRatio(0, 5)).toBe(0);
    expect(usageWindowRatio(2.5, 5)).toBe(0.5);
    expect(usageWindowRatio(7, 5)).toBe(1);
    expect(usageWindowRatio(-1, 5)).toBe(0);
    expect(usageWindowRatio(null, 5)).toBe(0);
  });
});

describe('usageWindowTone', () => {
  it('turns amber at 80% and red at 100%', () => {
    expect(usageWindowTone(0)).toBe('ok');
    expect(usageWindowTone(0.79)).toBe('ok');
    expect(usageWindowTone(0.8)).toBe('warn');
    expect(usageWindowTone(1)).toBe('over');
  });
});
