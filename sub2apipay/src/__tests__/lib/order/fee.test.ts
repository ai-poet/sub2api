import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGetSystemConfig = vi.fn<(key: string) => Promise<string | undefined>>();

vi.mock('@/lib/system-config', () => ({
  getSystemConfig: (...args: unknown[]) => mockGetSystemConfig(...(args as [string])),
}));

vi.mock('@/lib/payment', () => ({
  initPaymentProviders: vi.fn(),
  ensureDBProviders: vi.fn().mockResolvedValue(undefined),
  paymentRegistry: {
    getProviderKey: vi.fn().mockReturnValue('easypay'),
  },
}));

import {
  calculatePayAmount,
  calculatePaymentFeeQuote,
  getMaxSettlementAmountForPayLimit,
  getMethodFeeConfigs,
  getMinSettlementAmountForPayLimit,
  parsePaymentMethodFees,
  validatePaymentFeeConfig,
} from '@/lib/order/fee';

describe('calculatePaymentFeeQuote', () => {
  it.each([
    { amount: 100, feeRate: 0, fixedFee: 0, expected: '100.00', fee: '0.00' },
    { amount: 100, feeRate: 10, fixedFee: 0, expected: '111.12', fee: '11.12' },
    { amount: 100, feeRate: 0, fixedFee: 0.5, expected: '100.50', fee: '0.50' },
    { amount: 100, feeRate: 3.9, fixedFee: 3.6, expected: '107.81', fee: '7.81' },
    { amount: 0.01, feeRate: 1, fixedFee: 0, expected: '0.02', fee: '0.01' },
    { amount: 10000, feeRate: 2.5, fixedFee: 0, expected: '10256.42', fee: '256.42' },
  ])('反推实付与手续费（$amount, $feeRate%, ¥$fixedFee）', ({ amount, feeRate, fixedFee, expected, fee }) => {
    const result = calculatePaymentFeeQuote(amount, { enabled: feeRate > 0 || fixedFee > 0, feeRate, fixedFee });
    expect(result.payAmount.toFixed(2)).toBe(expected);
    expect(result.feeAmount.toFixed(2)).toBe(fee);
  });

  it('关闭时保留配置但不计费', () => {
    expect(calculatePaymentFeeQuote(100, { enabled: false, feeRate: 3.9, fixedFee: 3.6 })).toEqual({
      settlementAmount: 100,
      payAmount: 100,
      feeAmount: 0,
    });
  });

  it('无限小数按分向上取整', () => {
    expect(calculatePayAmount(33, 1)).toBe('33.34');
    expect(calculatePayAmount(7, 3)).toBe('7.22');
    expect(calculatePayAmount(1, 0.7)).toBe('1.01');
  });
});

describe('validatePaymentFeeConfig', () => {
  it.each([
    { enabled: true, feeRate: 100, fixedFee: 0 },
    { enabled: true, feeRate: -1, fixedFee: 0 },
    { enabled: true, feeRate: 0.00001, fixedFee: 0 },
    { enabled: true, feeRate: 0, fixedFee: -0.01 },
    { enabled: true, feeRate: 0, fixedFee: 0.001 },
    { enabled: true, feeRate: 0, fixedFee: 0 },
  ])('拒绝非法配置 %j', (config) => {
    expect(validatePaymentFeeConfig(config)).not.toBeNull();
  });

  it('允许禁用且全零的配置', () => {
    expect(validatePaymentFeeConfig({ enabled: false, feeRate: 0, fixedFee: 0 })).toBeNull();
  });
});

describe('配置解析与兼容回退', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetSystemConfig.mockResolvedValue(undefined);
    delete process.env.FEE_RATE_ALIPAY;
    delete process.env.FEE_RATE_PROVIDER_EASYPAY;
  });

  it('解析每种支付方式的开关、费率和固定费', () => {
    expect(parsePaymentMethodFees(JSON.stringify({ alipay: { enabled: true, feeRate: 3.9, fixedFee: 3.6 } }))).toEqual({
      alipay: { enabled: true, feeRate: 3.9, fixedFee: 3.6 },
    });
  });

  it('拒绝非法 JSON 与非法费率', () => {
    expect(() => parsePaymentMethodFees('{')).toThrow();
    expect(() =>
      parsePaymentMethodFees(JSON.stringify({ alipay: { enabled: true, feeRate: 100, fixedFee: 0 } })),
    ).toThrow();
  });

  it('数据库配置优先，未配置的方法回退旧环境变量', async () => {
    process.env.FEE_RATE_WXPAY = '1.6';
    mockGetSystemConfig.mockResolvedValue(JSON.stringify({ alipay: { enabled: true, feeRate: 3.9, fixedFee: 3.6 } }));

    const result = await getMethodFeeConfigs(['alipay', 'wxpay', 'stripe']);
    expect(result.alipay).toEqual({ enabled: true, feeRate: 3.9, fixedFee: 3.6 });
    expect(result.wxpay).toEqual({ enabled: true, feeRate: 1.6, fixedFee: 0 });
    expect(result.stripe).toEqual({ enabled: false, feeRate: 0, fixedFee: 0 });
    expect(mockGetSystemConfig).toHaveBeenCalledTimes(1);
  });

  it('显式禁用覆盖旧环境变量', async () => {
    process.env.FEE_RATE_ALIPAY = '2';
    mockGetSystemConfig.mockResolvedValue(JSON.stringify({ alipay: { enabled: false, feeRate: 2, fixedFee: 0 } }));
    expect((await getMethodFeeConfigs(['alipay'])).alipay.enabled).toBe(false);
  });
});

describe('实付限额反推', () => {
  const fee = { enabled: true, feeRate: 3.9, fixedFee: 3.6 };

  it('最大结算金额不会超过实付上限', () => {
    const max = getMaxSettlementAmountForPayLimit(110, fee);
    expect(max).toBe(102.11);
    expect(calculatePaymentFeeQuote(max, fee).payAmount).toBeLessThanOrEqual(110);
  });

  it('固定费超过上限时没有可支付的正金额', () => {
    expect(getMaxSettlementAmountForPayLimit(3, fee)).toBe(0);
  });

  it('最小结算金额至少达到实付下限', () => {
    const min = getMinSettlementAmountForPayLimit(110, fee);
    expect(min).toBe(102.11);
    expect(calculatePaymentFeeQuote(min, fee).payAmount).toBeGreaterThanOrEqual(110);
  });
});
