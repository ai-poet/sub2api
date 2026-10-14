import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockGetSystemConfig = vi.fn<(key: string) => Promise<string | undefined>>();
vi.mock('@/lib/system-config', () => ({
  getSystemConfig: (...args: unknown[]) => mockGetSystemConfig(...(args as [string])),
}));

const mockEnv: Record<string, number | undefined> = {};
vi.mock('@/lib/config', () => ({ getEnv: () => mockEnv }));

import {
  getRechargeLimits,
  isWithinRechargeRange,
  parseRechargeLimit,
  rechargeRangeMessage,
} from '@/lib/order/recharge-limits';

beforeEach(() => {
  mockGetSystemConfig.mockReset().mockResolvedValue(undefined);
  for (const key of Object.keys(mockEnv)) delete mockEnv[key];
});

describe('parseRechargeLimit', () => {
  it('后台配置优先，其次环境变量', () => {
    expect(parseRechargeLimit('5000', 1000)).toBe(5000);
    expect(parseRechargeLimit(undefined, 1000)).toBe(1000);
  });

  it('都没设置、为空或无效时为 0（不限），不写死默认值', () => {
    expect(parseRechargeLimit(undefined, undefined)).toBe(0);
    expect(parseRechargeLimit('', undefined)).toBe(0);
    expect(parseRechargeLimit('  ', undefined)).toBe(0);
    expect(parseRechargeLimit('abc', undefined)).toBe(0);
    expect(parseRechargeLimit('-1', undefined)).toBe(0);
  });

  it('后台填 0 表示不限，不再退回环境变量', () => {
    expect(parseRechargeLimit('0', 1000)).toBe(0);
  });

  it('后台值无效时退回环境变量', () => {
    expect(parseRechargeLimit('abc', 300)).toBe(300);
  });
});

describe('getRechargeLimits', () => {
  it('什么都没配置时三项都不限', async () => {
    expect(await getRechargeLimits()).toEqual({ minAmount: 0, maxAmount: 0, maxDailyAmount: 0 });
  });

  it('读取后台配置与环境变量', async () => {
    mockEnv.MIN_RECHARGE_AMOUNT = 1;
    mockEnv.MAX_DAILY_RECHARGE_AMOUNT = 20000;
    mockGetSystemConfig.mockImplementation(async (key) => (key === 'RECHARGE_MAX_AMOUNT' ? '5000' : undefined));
    expect(await getRechargeLimits()).toEqual({ minAmount: 1, maxAmount: 5000, maxDailyAmount: 20000 });
  });
});

describe('isWithinRechargeRange / rechargeRangeMessage', () => {
  it('任一侧为 0 时该侧不限', () => {
    expect(isWithinRechargeRange(100000, { minAmount: 0, maxAmount: 0 })).toBe(true);
    expect(isWithinRechargeRange(0.01, { minAmount: 0, maxAmount: 5000 })).toBe(true);
    expect(isWithinRechargeRange(6000, { minAmount: 0, maxAmount: 5000 })).toBe(false);
    expect(isWithinRechargeRange(5, { minAmount: 10, maxAmount: 0 })).toBe(false);
    expect(isWithinRechargeRange(50000, { minAmount: 10, maxAmount: 0 })).toBe(true);
  });

  it('提示只写出设置了的那一侧', () => {
    expect(rechargeRangeMessage('zh', { minAmount: 10, maxAmount: 5000 })).toBe('到账金额需在 $10 - $5000 之间');
    expect(rechargeRangeMessage('zh', { minAmount: 10, maxAmount: 0 })).toBe('到账金额不能低于 $10');
    expect(rechargeRangeMessage('en', { minAmount: 0, maxAmount: 5000 })).toBe('Credited balance must not exceed $5000');
  });
});
