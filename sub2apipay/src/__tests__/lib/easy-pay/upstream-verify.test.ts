import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { PaymentNotification, QueryOrderResponse } from '@/lib/payment/types';

const mockAuditLogCreate = vi.fn();
const mockAuditLogFindFirst = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    auditLog: {
      create: (...args: unknown[]) => mockAuditLogCreate(...args),
      findFirst: (...args: unknown[]) => mockAuditLogFindFirst(...args),
    },
  },
}));

import {
  checkOrderUpstream,
  compareWithUpstream,
  isUpstreamOrderNotFound,
  PAYMENT_UPSTREAM_MISMATCH,
  recordUpstreamMismatch,
  verifyEasyPayCreditUpstream,
} from '@/lib/easy-pay/upstream-verify';

const notification: PaymentNotification = {
  orderId: 'order-001',
  tradeNo: 'EP-001',
  amount: 10,
  status: 'success',
  rawData: {},
};
const credited = { amount: 10, tradeNo: 'EP-001' };

const paid: QueryOrderResponse = { tradeNo: 'EP-001', status: 'paid', amount: 10 };
const unpaid: QueryOrderResponse = { tradeNo: 'EP-001', status: 'pending', amount: 10 };
const notFound = new Error('EasyPay query order failed: 订单编号不存在');

const mockQueryOrder = vi.fn();
const provider = { name: 'easy-pay:inst-1', queryOrder: (...args: unknown[]) => mockQueryOrder(...args) };
const sleep = vi.fn(() => Promise.resolve());
const options = { delaysMs: [5_000, 60_000], sleep };

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  vi.spyOn(console, 'error').mockImplementation(() => {});
  mockAuditLogCreate.mockResolvedValue({});
  mockAuditLogFindFirst.mockResolvedValue(null);
});

describe('compareWithUpstream', () => {
  it('一致时无差异', () => {
    expect(compareWithUpstream(credited, paid)).toBeNull();
  });

  it('金额在一分钱以内视为一致', () => {
    expect(compareWithUpstream(credited, { ...paid, amount: 10.004 })).toBeNull();
  });

  it('任何一方没有交易号时不比较交易号', () => {
    expect(compareWithUpstream(credited, { ...paid, tradeNo: '' })).toBeNull();
    expect(compareWithUpstream({ ...credited, tradeNo: '' }, { ...paid, tradeNo: 'EP-999' })).toBeNull();
  });

  it('依次识别未付款、金额不符、交易号不符', () => {
    expect(compareWithUpstream(credited, unpaid)).toBe('upstream_not_paid');
    expect(compareWithUpstream(credited, { ...paid, amount: 650 })).toBe('upstream_amount_mismatch');
    expect(compareWithUpstream(credited, { ...paid, amount: Number.NaN })).toBe('upstream_amount_mismatch');
    expect(compareWithUpstream(credited, { ...paid, tradeNo: 'EP-999' })).toBe('upstream_trade_no_mismatch');
  });
});

describe('isUpstreamOrderNotFound', () => {
  it('认得平台各种"查无此单"的说法', () => {
    for (const msg of ['订单编号不存在', '订单不存在', '该订单号不存在', '查无此单', '无此订单', 'order not exist', 'Order does not exist', 'order not found']) {
      expect(isUpstreamOrderNotFound(new Error(`EasyPay query order failed: ${msg}`))).toBe(true);
    }
  });

  it('商户配置错误、网络错误不算查无此单，避免整批误报', () => {
    for (const msg of ['商户不存在', 'KEY校验失败', 'unknown error']) {
      expect(isUpstreamOrderNotFound(new Error(`EasyPay query order failed: ${msg}`))).toBe(false);
    }
    expect(isUpstreamOrderNotFound(new Error('fetch failed'))).toBe(false);
    expect(isUpstreamOrderNotFound(new Error('订单不存在'))).toBe(false);
    expect(isUpstreamOrderNotFound('EasyPay query order failed: 订单不存在')).toBe(false);
  });
});

describe('checkOrderUpstream', () => {
  it('一致 / 不符 / 查无此单 / 查单出错', async () => {
    mockQueryOrder.mockResolvedValueOnce(paid);
    expect(await checkOrderUpstream(provider, 'order-001', credited)).toEqual({ kind: 'ok', upstream: paid });

    mockQueryOrder.mockResolvedValueOnce(unpaid);
    expect(await checkOrderUpstream(provider, 'order-001', credited)).toEqual({
      kind: 'mismatch',
      reason: 'upstream_not_paid',
      upstream: unpaid,
    });

    mockQueryOrder.mockRejectedValueOnce(notFound);
    expect(await checkOrderUpstream(provider, 'order-001', credited)).toEqual({
      kind: 'mismatch',
      reason: 'upstream_order_not_found',
      upstream: null,
      message: '订单编号不存在',
    });

    const networkError = new Error('fetch failed');
    mockQueryOrder.mockRejectedValueOnce(networkError);
    expect(await checkOrderUpstream(provider, 'order-001', credited)).toEqual({ kind: 'error', error: networkError });
  });
});

describe('recordUpstreamMismatch', () => {
  const check = { kind: 'mismatch' as const, reason: 'upstream_not_paid' as const, upstream: unpaid };

  it('写审计，detail 带来源与平台、入账两侧的数据', async () => {
    const recorded = await recordUpstreamMismatch({
      orderId: 'order-001',
      operator: 'easy-pay:recheck',
      check,
      credited,
      source: 'recheck',
    });

    expect(recorded).toBe(true);
    expect(mockAuditLogFindFirst).not.toHaveBeenCalled();
    const { data } = mockAuditLogCreate.mock.calls[0][0];
    expect(data).toMatchObject({ orderId: 'order-001', action: PAYMENT_UPSTREAM_MISMATCH, operator: 'easy-pay:recheck' });
    expect(JSON.parse(data.detail)).toEqual({
      reason: 'upstream_not_paid',
      source: 'recheck',
      upstreamStatus: 'pending',
      upstreamAmount: 10,
      upstreamTradeNo: 'EP-001',
      notifyAmount: 10,
      notifyTradeNo: 'EP-001',
    });
  });

  it('查无此单时记下平台原话', async () => {
    await recordUpstreamMismatch({
      orderId: 'order-001',
      operator: 'easy-pay:recheck',
      check: { kind: 'mismatch', reason: 'upstream_order_not_found', upstream: null, message: '订单编号不存在' },
      credited,
      source: 'recheck',
    });

    expect(JSON.parse(mockAuditLogCreate.mock.calls[0][0].data.detail)).toMatchObject({
      reason: 'upstream_order_not_found',
      upstreamStatus: '',
      upstreamAmount: null,
      upstreamMessage: '订单编号不存在',
    });
  });

  it('dedupe：同一订单同一原因已有记录就不再写', async () => {
    mockAuditLogFindFirst.mockResolvedValue({ id: 'audit-1' });

    const recorded = await recordUpstreamMismatch({
      orderId: 'order-001',
      operator: 'easy-pay:recheck',
      check,
      credited,
      source: 'recheck',
      dedupe: true,
    });

    expect(recorded).toBe(false);
    expect(mockAuditLogFindFirst).toHaveBeenCalledWith({
      where: {
        orderId: 'order-001',
        action: PAYMENT_UPSTREAM_MISMATCH,
        detail: { contains: '"reason":"upstream_not_paid"' },
      },
      select: { id: true },
    });
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });
});

describe('verifyEasyPayCreditUpstream', () => {
  it('平台确认已付且一致：只查一次，不写审计', async () => {
    mockQueryOrder.mockResolvedValue(paid);

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(sleep).toHaveBeenCalledTimes(1);
    expect(sleep).toHaveBeenCalledWith(5_000);
    expect(mockQueryOrder).toHaveBeenCalledTimes(1);
    expect(mockQueryOrder).toHaveBeenCalledWith('order-001');
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('第一次未付款、复查已付：视为平台延迟，不写审计', async () => {
    mockQueryOrder.mockResolvedValueOnce(unpaid).mockResolvedValueOnce(paid);

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(sleep.mock.calls).toEqual([[5_000], [60_000]]);
    expect(mockQueryOrder).toHaveBeenCalledTimes(2);
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('两次都显示未付款：写高危审计', async () => {
    mockQueryOrder.mockResolvedValue(unpaid);

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(mockQueryOrder).toHaveBeenCalledTimes(2);
    expect(mockAuditLogCreate).toHaveBeenCalledTimes(1);
    const { data } = mockAuditLogCreate.mock.calls[0][0];
    expect(data).toMatchObject({
      orderId: 'order-001',
      action: PAYMENT_UPSTREAM_MISMATCH,
      operator: 'easy-pay:inst-1:verify',
    });
    expect(JSON.parse(data.detail)).toEqual({
      reason: 'upstream_not_paid',
      source: 'notify',
      upstreamStatus: 'pending',
      upstreamAmount: 10,
      upstreamTradeNo: 'EP-001',
      notifyAmount: 10,
      notifyTradeNo: 'EP-001',
    });
  });

  it('两次都查无此单：写高危审计', async () => {
    mockQueryOrder.mockRejectedValue(notFound);

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(mockQueryOrder).toHaveBeenCalledTimes(2);
    expect(JSON.parse(mockAuditLogCreate.mock.calls[0][0].data.detail)).toMatchObject({
      reason: 'upstream_order_not_found',
      upstreamMessage: '订单编号不存在',
    });
  });

  it('金额不符：不再复查，立即写审计', async () => {
    mockQueryOrder.mockResolvedValue({ ...paid, amount: 0.01 });

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(mockQueryOrder).toHaveBeenCalledTimes(1);
    expect(JSON.parse(mockAuditLogCreate.mock.calls[0][0].data.detail)).toMatchObject({
      reason: 'upstream_amount_mismatch',
      upstreamAmount: 0.01,
      notifyAmount: 10,
    });
  });

  it('交易号不符：写审计', async () => {
    mockQueryOrder.mockResolvedValue({ ...paid, tradeNo: 'EP-999' });

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(JSON.parse(mockAuditLogCreate.mock.calls[0][0].data.detail)).toMatchObject({
      reason: 'upstream_trade_no_mismatch',
      upstreamTradeNo: 'EP-999',
    });
  });

  it('查单失败一次、复查成功：按复查结果判断', async () => {
    mockQueryOrder.mockRejectedValueOnce(new Error('timeout')).mockResolvedValueOnce(paid);

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(mockQueryOrder).toHaveBeenCalledTimes(2);
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('两次查单都失败：只告警，不下结论', async () => {
    mockQueryOrder.mockRejectedValue(new Error('platform down'));

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(mockAuditLogCreate).not.toHaveBeenCalled();
    expect(console.warn).toHaveBeenCalledWith(expect.stringContaining('no conclusion drawn'), expect.any(Error));
  });

  it('第一次未付款、复查时查单失败：不下结论', async () => {
    mockQueryOrder.mockResolvedValueOnce(unpaid).mockRejectedValueOnce(new Error('timeout'));

    await verifyEasyPayCreditUpstream(notification, provider, options);

    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('写审计出错：吞掉异常', async () => {
    mockQueryOrder.mockResolvedValue(unpaid);
    mockAuditLogCreate.mockRejectedValue(new Error('db down'));

    await expect(verifyEasyPayCreditUpstream(notification, provider, options)).resolves.toBeUndefined();
  });
});
