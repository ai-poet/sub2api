import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { PaymentNotification } from '@/lib/payment/types';

const mockOrderFindUnique = vi.fn();
const mockAuditLogCreate = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    order: {
      findUnique: (...args: unknown[]) => mockOrderFindUnique(...args),
    },
    auditLog: {
      create: (...args: unknown[]) => mockAuditLogCreate(...args),
    },
  },
}));

import { observeEasyPayNotify, PAYMENT_NOTIFY_ANOMALY } from '@/lib/easy-pay/notify-audit';

const STANDARD_RAW = {
  pid: '1001',
  trade_no: 'EP-001',
  out_trade_no: 'order-001',
  type: 'alipay',
  name: 'Balance recharge',
  money: '10.00',
  trade_status: 'TRADE_SUCCESS',
  sign: 'x',
  sign_type: 'MD5',
};

function notification(overrides: Partial<PaymentNotification> = {}): PaymentNotification {
  return {
    orderId: 'order-001',
    tradeNo: 'EP-001',
    amount: 10,
    status: 'success',
    rawData: STANDARD_RAW,
    ...overrides,
  };
}

describe('observeEasyPayNotify', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    mockAuditLogCreate.mockResolvedValue({});
  });

  it('交易号与下单时平台返回的一致：不写审计', async () => {
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001', status: 'PENDING', paymentTradeNo: 'EP-001' });

    await observeEasyPayNotify(notification(), 'easy-pay');

    expect(mockOrderFindUnique).toHaveBeenCalledWith({
      where: { id: 'order-001' },
      select: { id: true, status: true, paymentTradeNo: true },
    });
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('订单上没有存交易号时无从比较：不写审计', async () => {
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001', status: 'PENDING', paymentTradeNo: null });

    await observeEasyPayNotify(notification(), 'easy-pay');

    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('成功通知没带交易号：写 missing_trade_no 审计', async () => {
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001', status: 'PENDING', paymentTradeNo: 'EP-001' });

    await observeEasyPayNotify(notification({ tradeNo: '' }), 'easy-pay:inst-1');

    expect(mockAuditLogCreate).toHaveBeenCalledTimes(1);
    const { data } = mockAuditLogCreate.mock.calls[0][0];
    expect(data).toMatchObject({ orderId: 'order-001', action: PAYMENT_NOTIFY_ANOMALY, operator: 'easy-pay:inst-1:notify' });
    expect(JSON.parse(data.detail)).toEqual({
      reason: 'missing_trade_no',
      notifyTradeNo: '',
      storedTradeNo: 'EP-001',
      orderStatus: 'PENDING',
      amount: 10,
    });
  });

  it('通知交易号与下单时平台返回的不同：写 trade_no_mismatch 审计', async () => {
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001', status: 'EXPIRED', paymentTradeNo: 'EP-001' });

    await observeEasyPayNotify(notification({ tradeNo: 'EP-999' }), 'easy-pay');

    expect(mockAuditLogCreate).toHaveBeenCalledTimes(1);
    expect(JSON.parse(mockAuditLogCreate.mock.calls[0][0].data.detail)).toMatchObject({
      reason: 'trade_no_mismatch',
      notifyTradeNo: 'EP-999',
      storedTradeNo: 'EP-001',
      orderStatus: 'EXPIRED',
    });
  });

  it('订单不存在：不写审计、不抛错', async () => {
    mockOrderFindUnique.mockResolvedValue(null);

    await expect(observeEasyPayNotify(notification({ tradeNo: '' }), 'easy-pay')).resolves.toBeUndefined();
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('非成功通知：不查库', async () => {
    await observeEasyPayNotify(notification({ status: 'failed', tradeNo: '' }), 'easy-pay');

    expect(mockOrderFindUnique).not.toHaveBeenCalled();
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('查库出错：吞掉异常', async () => {
    mockOrderFindUnique.mockRejectedValue(new Error('db down'));

    await expect(observeEasyPayNotify(notification(), 'easy-pay')).resolves.toBeUndefined();
  });

  it('写审计出错：吞掉异常', async () => {
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001', status: 'PENDING', paymentTradeNo: 'EP-001' });
    mockAuditLogCreate.mockRejectedValue(new Error('db down'));

    await expect(observeEasyPayNotify(notification({ tradeNo: 'EP-999' }), 'easy-pay')).resolves.toBeUndefined();
  });

  it('标准字段 + inst：不告警', async () => {
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001', status: 'PENDING', paymentTradeNo: 'EP-001' });

    await observeEasyPayNotify(notification({ rawData: { ...STANDARD_RAW, inst: 'inst-1', param: '' } }), 'easy-pay');

    expect(console.warn).not.toHaveBeenCalled();
  });

  it('出现标准字段以外的参数名：只告警，不写审计', async () => {
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001', status: 'PENDING', paymentTradeNo: 'EP-001' });

    await observeEasyPayNotify(
      notification({ rawData: { ...STANDARD_RAW, buyer: 'b@example.com', addtime: '2026-10-06 10:00:00' } }),
      'easy-pay',
    );

    expect(console.warn).toHaveBeenCalledWith(expect.stringContaining('buyer,addtime'));
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });
});
