import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { PaymentNotification } from '@/lib/payment/types';

const mockOrderFindUnique = vi.fn();
const mockAuditLogCreate = vi.fn();
const mockAuditLogFindFirst = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    order: {
      findUnique: (...args: unknown[]) => mockOrderFindUnique(...args),
    },
    auditLog: {
      create: (...args: unknown[]) => mockAuditLogCreate(...args),
      findFirst: (...args: unknown[]) => mockAuditLogFindFirst(...args),
    },
  },
}));

import {
  observeEasyPayNotify,
  PAYMENT_NOTIFY_ANOMALY,
  PAYMENT_NOTIFY_BLOCKED,
  recordBlockedEasyPayNotify,
} from '@/lib/easy-pay/notify-audit';
import { EasyPayNotifyRejectedError } from '@/lib/easy-pay/notify-errors';

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

describe('recordBlockedEasyPayNotify', () => {
  const rejected = new EasyPayNotifyRejectedError(
    'EasyPay notification rejected: unexpected param "clientip" (out_trade_no="order-001")',
    'unexpected_param',
    'order-001',
    'clientip',
  );

  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    mockOrderFindUnique.mockResolvedValue({ id: 'order-001' });
    mockAuditLogFindFirst.mockResolvedValue(null);
    mockAuditLogCreate.mockResolvedValue({});
  });

  it('验签被拒且订单存在：写一条拦截记录', async () => {
    await recordBlockedEasyPayNotify(rejected, 'easy-pay:inst-1');

    expect(mockOrderFindUnique).toHaveBeenCalledWith({ where: { id: 'order-001' }, select: { id: true } });
    expect(mockAuditLogCreate).toHaveBeenCalledTimes(1);
    const { data } = mockAuditLogCreate.mock.calls[0][0];
    expect(data).toMatchObject({
      orderId: 'order-001',
      action: PAYMENT_NOTIFY_BLOCKED,
      operator: 'easy-pay:inst-1:notify',
    });
    expect(JSON.parse(data.detail)).toEqual({
      reason: 'unexpected_param',
      param: 'clientip',
      message: rejected.message,
    });
  });

  it('签名错误没有 param 字段', async () => {
    await recordBlockedEasyPayNotify(
      new EasyPayNotifyRejectedError('EasyPay notification signature verification failed', 'bad_signature', 'order-001'),
      'easy-pay',
    );

    expect(JSON.parse(mockAuditLogCreate.mock.calls[0][0].data.detail)).toEqual({
      reason: 'bad_signature',
      message: 'EasyPay notification signature verification failed',
    });
  });

  it('十分钟内已有拦截记录：不重复写', async () => {
    mockAuditLogFindFirst.mockResolvedValue({ id: 'audit-1' });

    await recordBlockedEasyPayNotify(rejected, 'easy-pay');

    const where = mockAuditLogFindFirst.mock.calls[0][0].where;
    expect(where).toMatchObject({ orderId: 'order-001', action: PAYMENT_NOTIFY_BLOCKED });
    expect(Date.now() - where.createdAt.gte.getTime()).toBeGreaterThanOrEqual(10 * 60 * 1000 - 1000);
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('不是验签拒绝的错误（如入账时数据库出错）：不记录', async () => {
    await recordBlockedEasyPayNotify(new Error('db down'), 'easy-pay');

    expect(mockOrderFindUnique).not.toHaveBeenCalled();
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('订单号为空、过长或对不上订单：不记录', async () => {
    await recordBlockedEasyPayNotify(new EasyPayNotifyRejectedError('x', 'bad_signature', ''), 'easy-pay');
    await recordBlockedEasyPayNotify(new EasyPayNotifyRejectedError('x', 'bad_signature', 'a'.repeat(65)), 'easy-pay');
    expect(mockOrderFindUnique).not.toHaveBeenCalled();

    mockOrderFindUnique.mockResolvedValue(null);
    await recordBlockedEasyPayNotify(rejected, 'easy-pay');
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('查库或写库出错：吞掉异常', async () => {
    mockOrderFindUnique.mockRejectedValue(new Error('db down'));
    await expect(recordBlockedEasyPayNotify(rejected, 'easy-pay')).resolves.toBeUndefined();

    mockOrderFindUnique.mockResolvedValue({ id: 'order-001' });
    mockAuditLogCreate.mockRejectedValue(new Error('db down'));
    await expect(recordBlockedEasyPayNotify(rejected, 'easy-pay')).resolves.toBeUndefined();
  });
});
