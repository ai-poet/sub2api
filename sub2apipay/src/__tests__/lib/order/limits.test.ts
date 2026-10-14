import { vi, describe, it, expect, beforeEach } from 'vitest';

// mock getSystemConfig：默认返回 undefined（无 DB 配置）
const mockGetSystemConfig = vi.fn<(key: string) => Promise<string | undefined>>().mockResolvedValue(undefined);
vi.mock('@/lib/system-config', () => ({
  getSystemConfig: (...args: unknown[]) => mockGetSystemConfig(...(args as [string])),
}));

vi.mock('@/lib/config', () => ({
  getEnv: () => ({
    BALANCE_CREDIT_CNY_PER_USD: 1,
    BALANCE_CREDIT_USD_PER_CNY: undefined,
  }),
}));

const mockOrderGroupBy = vi.fn();
const mockFindManyProviderInstances = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    order: { groupBy: (...args: unknown[]) => mockOrderGroupBy(...args) },
    paymentProviderInstance: {
      findMany: (...args: unknown[]) => mockFindManyProviderInstances(...args),
    },
  },
}));

// fee.ts 用到 payment 模块（手续费率）；真实模块初始化要读 PAYMENT_PROVIDERS 等环境变量
vi.mock('@/lib/payment', () => ({
  initPaymentProviders: vi.fn(),
  paymentRegistry: { getProviderKey: vi.fn() },
}));

import { getMethodDailyLimit, getMethodSingleLimit, queryMethodLimits } from '@/lib/order/limits';

beforeEach(() => {
  vi.clearAllMocks();
  mockGetSystemConfig.mockResolvedValue(undefined);
  mockOrderGroupBy.mockResolvedValue([]);
  mockFindManyProviderInstances.mockResolvedValue([]);
  delete process.env.FEE_RATE_ALIPAY;
});

// 渠道级限额只认显式配置，代码里不写死默认值（没配置 = 不限）。
describe.each([
  ['getMethodDailyLimit', getMethodDailyLimit, 'MAX_DAILY_AMOUNT'],
  ['getMethodSingleLimit', getMethodSingleLimit, 'MAX_SINGLE_AMOUNT'],
] as const)('%s', (_name, getLimit, prefix) => {
  const withConfig = (values: Record<string, string>) =>
    mockGetSystemConfig.mockImplementation(async (key) => values[key]);

  it('没有配置时返回 0（不限），不套用任何写死的默认值', async () => {
    for (const type of ['alipay', 'wxpay', 'alipay_direct', 'wxpay_direct', 'usdc.solana', 'stripe']) {
      expect(await getLimit(type)).toBe(0);
    }
  });

  it('读取渠道配置', async () => {
    withConfig({ [`${prefix}_WXPAY`]: '500' });
    expect(await getLimit('wxpay')).toBe(500);
  });

  it('配置为 0 表示不限', async () => {
    withConfig({ [`${prefix}_ALIPAY`]: '0' });
    expect(await getLimit('alipay')).toBe(0);
  });

  it('无效配置视为未配置', async () => {
    withConfig({ [`${prefix}_ALIPAY`]: 'abc' });
    expect(await getLimit('alipay')).toBe(0);
    withConfig({ [`${prefix}_ALIPAY`]: '-5' });
    expect(await getLimit('alipay')).toBe(0);
  });

  it('OVERRIDE_ENV_ENABLED 不再影响结果', async () => {
    withConfig({ OVERRIDE_ENV_ENABLED: 'false', [`${prefix}_ALIPAY`]: '300' });
    expect(await getLimit('alipay')).toBe(300);
  });

  it('paymentType 大小写不敏感（key 构造用 toUpperCase）', async () => {
    withConfig({ [`${prefix}_USDC.SOLANA`]: '700' });
    expect(await getLimit('usdc.solana')).toBe(700);
  });
});

describe('queryMethodLimits', () => {
  it('将支付渠道单笔上限换算成到账 USD', async () => {
    mockGetSystemConfig.mockImplementation(async (key) => {
      if (key === 'BALANCE_CREDIT_CNY_PER_USD') return '6.6667';
      if (key === 'MAX_SINGLE_AMOUNT_ALIPAY') return '1000';
      if (key === 'MAX_DAILY_AMOUNT_ALIPAY') return '10000';
      return undefined;
    });

    const result = await queryMethodLimits(['alipay']);

    expect(result.alipay.singleMax).toBe(150);
    expect(result.alipay.dailyLimit).toBe(1499.99);
    expect(result.alipay.available).toBe(true);
  });

  it('在有手续费时按网关实付上限反推到账上限', async () => {
    process.env.FEE_RATE_ALIPAY = '10';
    mockGetSystemConfig.mockImplementation(async (key) => {
      if (key === 'MAX_SINGLE_AMOUNT_ALIPAY') return '110';
      return undefined;
    });

    const result = await queryMethodLimits(['alipay']);

    expect(result.alipay.singleMax).toBe(100);
  });

  // 线上问题：最大到账金额设成 5000，易支付写死的单笔 1000 仍把付款上限卡在 1000。
  it('什么都没配时渠道不设单笔与每日上限，前端按全局最大到账金额', async () => {
    mockGetSystemConfig.mockImplementation(async (key) => (key === 'BALANCE_CREDIT_CNY_PER_USD' ? '1.0000' : undefined));

    const result = await queryMethodLimits(['alipay', 'wxpay', 'usdc.solana']);

    for (const type of ['alipay', 'wxpay', 'usdc.solana']) {
      expect(result[type].singleMax).toBe(0);
      expect(result[type].dailyLimit).toBe(0);
      expect(result[type].available).toBe(true);
    }
  });
});
