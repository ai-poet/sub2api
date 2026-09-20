import { vi, describe, it, expect, beforeEach } from 'vitest';
import { ORDER_STATUS } from '@/lib/constants';

const mockOrderFindMany = vi.fn();
const mockReconcilePendingOrderPayment = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    order: {
      findMany: (...args: unknown[]) => mockOrderFindMany(...args),
    },
  },
  getConfiguredDatabaseSchema: () => 'public',
}));

vi.mock('@/lib/order/service', () => ({
  reconcilePendingOrderPayment: (...args: unknown[]) => mockReconcilePendingOrderPayment(...args),
}));

import { reconcilePendingOrders } from '@/lib/order/reconcile';

describe('reconcilePendingOrders', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'log').mockImplementation(() => {});
  });

  it('只对账创建满 1 分钟、尚未到期的 PENDING 订单，逐单以 sweep 来源查单', async () => {
    const now = new Date('2026-09-21T10:00:00.000Z');
    mockOrderFindMany.mockResolvedValue([{ id: 'order-a' }, { id: 'order-b' }, { id: 'order-c' }]);
    mockReconcilePendingOrderPayment
      .mockResolvedValueOnce(true)
      .mockResolvedValueOnce(false)
      .mockResolvedValueOnce(true);

    const confirmed = await reconcilePendingOrders(now);

    expect(confirmed).toBe(2);
    expect(mockOrderFindMany).toHaveBeenCalledWith(
      expect.objectContaining({
        where: {
          status: ORDER_STATUS.PENDING,
          createdAt: { lt: new Date('2026-09-21T09:59:00.000Z') },
          expiresAt: { gt: now },
        },
        take: 50,
        orderBy: { createdAt: 'asc' },
      }),
    );
    expect(mockReconcilePendingOrderPayment.mock.calls).toEqual([
      ['order-a', 'sweep'],
      ['order-b', 'sweep'],
      ['order-c', 'sweep'],
    ]);
  });

  it('没有待对账订单时不查单', async () => {
    mockOrderFindMany.mockResolvedValue([]);

    expect(await reconcilePendingOrders()).toBe(0);
    expect(mockReconcilePendingOrderPayment).not.toHaveBeenCalled();
  });
});
