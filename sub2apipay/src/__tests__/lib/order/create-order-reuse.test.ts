import { vi, describe, it, expect, beforeEach } from 'vitest';
import { Prisma } from '@prisma/client';
import { ORDER_STATUS } from '@/lib/constants';

// ── mock 外部依赖（布局同 cancel-core.test.ts） ──

const mockPlanFindUnique = vi.fn();
const mockOrderFindFirst = vi.fn();
const mockAuditLogCreate = vi.fn();
const mockTransaction = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    subscriptionPlan: {
      findUnique: (...args: unknown[]) => mockPlanFindUnique(...args),
    },
    order: {
      findFirst: (...args: unknown[]) => mockOrderFindFirst(...args),
    },
    auditLog: {
      create: (...args: unknown[]) => mockAuditLogCreate(...args),
      count: vi.fn().mockResolvedValue(0),
    },
    $transaction: (...args: unknown[]) => mockTransaction(...args),
  },
  getConfiguredDatabaseSchema: () => 'public',
}));

const mockGetUser = vi.fn();
const mockGetGroup = vi.fn();

vi.mock('@/lib/sub2api/client', () => ({
  getUser: (...args: unknown[]) => mockGetUser(...args),
  getGroup: (...args: unknown[]) => mockGetGroup(...args),
  createAndRedeem: vi.fn(),
  subtractBalance: vi.fn(),
  addBalance: vi.fn(),
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
    NEXT_PUBLIC_APP_URL: 'https://gw.example.com/pay',
    ORDER_TIMEOUT_MINUTES: 5,
    MAX_DAILY_RECHARGE_AMOUNT: 0,
  }),
}));

vi.mock('@/lib/system-config', () => ({
  getSystemConfig: vi.fn().mockResolvedValue(undefined),
  getSystemConfigs: vi.fn().mockResolvedValue({}),
}));

import { createOrder } from '@/lib/order/service';
import { verifyOrderStatusAccessToken } from '@/lib/order/status-access';

const TX_REACHED = new Error('transaction reached');

function pendingOrder(overrides: Record<string, unknown> = {}) {
  return {
    id: 'order-plan-1',
    userId: 42,
    amount: new Prisma.Decimal('29.90'),
    payAmount: new Prisma.Decimal('30.08'),
    feeRate: new Prisma.Decimal('0.0060'),
    status: ORDER_STATUS.PENDING,
    orderType: 'subscription',
    planId: 'plan-1',
    paymentType: 'alipay_direct',
    payUrl: 'https://gw.example.com/pay/order-plan-1',
    qrCode: 'https://gw.example.com/pay/order-plan-1',
    expiresAt: new Date(Date.now() + 4 * 60_000),
    ...overrides,
  };
}

function input(overrides: Record<string, unknown> = {}) {
  return {
    userId: 42,
    amount: 29.9,
    paymentType: 'alipay_direct',
    clientIp: '127.0.0.1',
    orderType: 'subscription' as const,
    planId: 'plan-1',
    ...overrides,
  };
}

describe('createOrder — pending subscription order reuse', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPlanFindUnique.mockResolvedValue({
      id: 'plan-1',
      groupId: 7,
      price: new Prisma.Decimal('29.90'),
      validityDays: 1,
      validityUnit: 'month',
      name: 'Pro',
      productName: null,
      forSale: true,
    });
    mockGetGroup.mockResolvedValue({ id: 7, name: 'Pro', status: 'active', subscription_type: 'subscription' });
    mockGetUser.mockResolvedValue({ id: 42, status: 'active', username: 'u', email: 'u@example.com', balance: 3 });
    mockTransaction.mockRejectedValue(TX_REACHED);
  });

  it('hands back a still-payable pending order with a fresh status token', async () => {
    mockOrderFindFirst.mockResolvedValue(pendingOrder());

    const result = await createOrder(input() as Parameters<typeof createOrder>[0]);

    expect(result.orderId).toBe('order-plan-1');
    expect(result.qrCode).toBe('https://gw.example.com/pay/order-plan-1');
    expect(result.payAmount).toBeCloseTo(30.08);
    expect(verifyOrderStatusAccessToken('order-plan-1', result.statusAccessToken)).toBe(true);
    expect(mockTransaction).not.toHaveBeenCalled();
    expect(mockAuditLogCreate).toHaveBeenCalledWith(
      expect.objectContaining({ data: expect.objectContaining({ action: 'ORDER_REUSED' }) }),
    );
  });

  it('creates a new order when the plan price changed since', async () => {
    mockOrderFindFirst.mockResolvedValue(pendingOrder({ amount: new Prisma.Decimal('19.90') }));

    await expect(createOrder(input() as Parameters<typeof createOrder>[0])).rejects.toBe(TX_REACHED);
  });

  it('never reuses for mobile requests', async () => {
    await expect(createOrder(input({ isMobile: true }) as Parameters<typeof createOrder>[0])).rejects.toBe(
      TX_REACHED,
    );
    expect(mockOrderFindFirst).not.toHaveBeenCalled();
  });

  it('only looks at recent, QR-carrying pending orders for the same plan and method', async () => {
    mockOrderFindFirst.mockResolvedValue(null);

    await expect(createOrder(input() as Parameters<typeof createOrder>[0])).rejects.toBe(TX_REACHED);
    const where = mockOrderFindFirst.mock.calls[0][0].where;
    expect(where).toMatchObject({
      userId: 42,
      orderType: 'subscription',
      planId: 'plan-1',
      paymentType: 'alipay_direct',
      status: ORDER_STATUS.PENDING,
      qrCode: { not: null },
    });
  });
});
