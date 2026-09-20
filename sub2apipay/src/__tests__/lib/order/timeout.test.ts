import { vi, describe, it, expect, beforeEach } from 'vitest';
import { ORDER_STATUS } from '@/lib/constants';

const mockOrderFindMany = vi.fn();
const mockOrderUpdateMany = vi.fn();
const mockAuditLogCreate = vi.fn();
const mockCancelOrderCore = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    order: {
      findMany: (...args: unknown[]) => mockOrderFindMany(...args),
      updateMany: (...args: unknown[]) => mockOrderUpdateMany(...args),
    },
    auditLog: {
      create: (...args: unknown[]) => mockAuditLogCreate(...args),
    },
  },
  getConfiguredDatabaseSchema: () => 'public',
}));

vi.mock('@/lib/order/service', () => ({
  cancelOrderCore: (...args: unknown[]) => mockCancelOrderCore(...args),
}));

import { expireOrders } from '@/lib/order/timeout';

const NOW = new Date('2026-09-21T10:00:00.000Z');

function pendingOrder(id: string, expiredAgoMs: number) {
  return {
    id,
    paymentTradeNo: `T-${id}`,
    paymentType: 'alipay',
    providerInstanceId: null,
    expiresAt: new Date(NOW.getTime() - expiredAgoMs),
  };
}

describe('expireOrders', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    mockAuditLogCreate.mockResolvedValue({});
    mockOrderUpdateMany.mockResolvedValue({ count: 1 });
  });

  it('到期未满 1 小时：平台查单失败时跳过（skip），满 1 小时后才允许本地过期', async () => {
    mockOrderFindMany.mockResolvedValue([pendingOrder('fresh', 5 * 60 * 1000), pendingOrder('stale', 2 * 60 * 60 * 1000)]);
    mockCancelOrderCore.mockResolvedValueOnce('platform_unavailable').mockResolvedValueOnce('cancelled');

    const expired = await expireOrders(NOW);

    expect(expired).toBe(1);
    expect(mockCancelOrderCore).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ orderId: 'fresh', finalStatus: ORDER_STATUS.EXPIRED, onPlatformError: 'skip' }),
    );
    expect(mockCancelOrderCore).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ orderId: 'stale', finalStatus: ORDER_STATUS.EXPIRED, onPlatformError: 'cancel' }),
    );
    // platform_unavailable 不动订单
    expect(mockOrderUpdateMany).not.toHaveBeenCalled();
  });

  it('平台已付款但入账失败：CAS 从 PENDING 置 FAILED 并写 PAYMENT_CONFIRM_FAILED，移出待支付池', async () => {
    mockOrderFindMany.mockResolvedValue([pendingOrder('mismatch', 60 * 1000)]);
    mockCancelOrderCore.mockResolvedValue('paid_unconfirmed');

    const expired = await expireOrders(NOW);

    expect(expired).toBe(0);
    expect(mockOrderUpdateMany).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { id: 'mismatch', status: ORDER_STATUS.PENDING },
        data: expect.objectContaining({
          status: ORDER_STATUS.FAILED,
          failedReason: expect.stringContaining('PAID_ON_PLATFORM_BUT_UNCONFIRMED'),
        }),
      }),
    );
    expect(mockAuditLogCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({ orderId: 'mismatch', action: 'PAYMENT_CONFIRM_FAILED', operator: 'timeout' }),
      }),
    );
  });

  it('paid_unconfirmed 但订单已被其它路径推进（CAS 未命中）：不写审计', async () => {
    mockOrderFindMany.mockResolvedValue([pendingOrder('racing', 60 * 1000)]);
    mockCancelOrderCore.mockResolvedValue('paid_unconfirmed');
    mockOrderUpdateMany.mockResolvedValue({ count: 0 });

    await expireOrders(NOW);

    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('already_paid 不计入过期数，单笔异常不影响其它订单', async () => {
    mockOrderFindMany.mockResolvedValue([
      pendingOrder('paid', 60 * 1000),
      pendingOrder('boom', 60 * 1000),
      pendingOrder('gone', 60 * 1000),
    ]);
    mockCancelOrderCore
      .mockResolvedValueOnce('already_paid')
      .mockRejectedValueOnce(new Error('db down'))
      .mockResolvedValueOnce('cancelled');

    const expired = await expireOrders(NOW);

    expect(expired).toBe(1);
    expect(mockCancelOrderCore).toHaveBeenCalledTimes(3);
  });
});
