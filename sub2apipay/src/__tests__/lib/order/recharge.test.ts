import { vi, describe, it, expect, beforeEach } from 'vitest';
import { Prisma } from '@prisma/client';
import { ORDER_STATUS } from '@/lib/constants';

// ── mock 外部依赖（布局同 refund.test.ts） ──

const mockOrderFindUnique = vi.fn();
const mockOrderUpdateMany = vi.fn();
const mockOrderUpdate = vi.fn();
const mockAuditLogCreate = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    order: {
      findUnique: (...args: unknown[]) => mockOrderFindUnique(...args),
      updateMany: (...args: unknown[]) => mockOrderUpdateMany(...args),
      update: (...args: unknown[]) => mockOrderUpdate(...args),
    },
    auditLog: {
      create: (...args: unknown[]) => mockAuditLogCreate(...args),
    },
  },
  getConfiguredDatabaseSchema: () => 'public',
}));

const mockCreateAndRedeem = vi.fn();

vi.mock('@/lib/sub2api/client', () => ({
  getUser: vi.fn(),
  createAndRedeem: (...args: unknown[]) => mockCreateAndRedeem(...args),
  subtractBalance: vi.fn(),
  addBalance: vi.fn(),
  getGroup: vi.fn(),
  getUserSubscriptions: vi.fn(),
  extendSubscription: vi.fn(),
}));

vi.mock('@/lib/payment', () => ({
  initPaymentProviders: vi.fn(),
  ensureDBProviders: vi.fn().mockResolvedValue(undefined),
  paymentRegistry: { getProvider: vi.fn() },
}));

vi.mock('@/lib/payment/load-balancer', () => ({
  getInstanceConfig: vi.fn().mockResolvedValue(null),
  selectInstance: vi.fn(),
}));

vi.mock('@/lib/config', () => ({
  getEnv: () => ({
    JWT_SECRET: 'test-jwt-secret-123456',
    ADMIN_TOKEN: 'test-admin-token',
  }),
}));

vi.mock('@/lib/system-config', () => ({
  getSystemConfig: vi.fn(),
  getSystemConfigs: vi.fn(),
}));

import { executeRecharge } from '@/lib/order/service';

function makeOrder(overrides: Record<string, unknown> = {}) {
  return {
    id: 'order-001',
    userId: 42,
    amount: new Prisma.Decimal('100.00'),
    payAmount: new Prisma.Decimal('100.00'),
    rechargeCode: 'RC-ORDER-001',
    status: ORDER_STATUS.PAID,
    orderType: 'balance',
    paymentType: 'alipay',
    ...overrides,
  };
}

describe('executeRecharge', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockOrderUpdateMany.mockResolvedValue({ count: 1 });
    mockOrderUpdate.mockResolvedValue({});
    mockAuditLogCreate.mockResolvedValue({});
    mockCreateAndRedeem.mockResolvedValue({ id: 1, code: 'RC-ORDER-001' });
  });

  it('含活动赠送时按 amount + bonusAmount 一次性入账', async () => {
    mockOrderFindUnique.mockResolvedValue(
      makeOrder({
        bonusAmount: new Prisma.Decimal('10.00'),
        promotionId: 'promo-1',
        promotionName: '充100送10',
      }),
    );

    await executeRecharge('order-001');

    expect(mockCreateAndRedeem).toHaveBeenCalledTimes(1);
    const [code, value, userId, notes] = mockCreateAndRedeem.mock.calls[0];
    expect(code).toBe('RC-ORDER-001');
    expect(value).toBe(110);
    expect(userId).toBe(42);
    expect(notes).toContain('bonus 10.00');
    expect(notes).toContain('充100送10');

    const successLog = mockAuditLogCreate.mock.calls
      .map((call) => call[0].data)
      .find((data) => data.action === 'RECHARGE_SUCCESS');
    expect(JSON.parse(successLog.detail)).toMatchObject({
      amount: 100,
      bonusAmount: 10,
      creditedTotal: 110,
      promotionId: 'promo-1',
      promotionName: '充100送10',
    });
  });

  it('bonusAmount 为 null 时只入账 amount', async () => {
    mockOrderFindUnique.mockResolvedValue(makeOrder({ bonusAmount: null, promotionId: null, promotionName: null }));

    await executeRecharge('order-001');

    const [, value, , notes] = mockCreateAndRedeem.mock.calls[0];
    expect(value).toBe(100);
    expect(notes).toBe('payment center recharge order:order-001');
    const successLog = mockAuditLogCreate.mock.calls
      .map((call) => call[0].data)
      .find((data) => data.action === 'RECHARGE_SUCCESS');
    expect(JSON.parse(successLog.detail)).toEqual({ rechargeCode: 'RC-ORDER-001', amount: 100 });
  });

  it('bonusAmount 缺失（历史订单）时只入账 amount', async () => {
    mockOrderFindUnique.mockResolvedValue(makeOrder());

    await executeRecharge('order-001');

    expect(mockCreateAndRedeem.mock.calls[0][1]).toBe(100);
  });

  it('入账失败时标记 FAILED 并抛出', async () => {
    mockOrderFindUnique.mockResolvedValue(makeOrder({ bonusAmount: new Prisma.Decimal('10.00') }));
    mockCreateAndRedeem.mockRejectedValue(new Error('upstream down'));

    await expect(executeRecharge('order-001')).rejects.toThrow('upstream down');
    expect(mockOrderUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { id: 'order-001' },
        data: expect.objectContaining({ status: ORDER_STATUS.FAILED }),
      }),
    );
  });

  it('已完成订单直接返回', async () => {
    mockOrderFindUnique.mockResolvedValue(makeOrder({ status: ORDER_STATUS.COMPLETED }));
    await executeRecharge('order-001');
    expect(mockCreateAndRedeem).not.toHaveBeenCalled();
  });
});
