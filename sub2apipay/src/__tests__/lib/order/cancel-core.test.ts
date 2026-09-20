import { vi, describe, it, expect, beforeEach } from 'vitest';
import { Prisma } from '@prisma/client';
import { ORDER_STATUS } from '@/lib/constants';

// ── mock 外部依赖（布局同 recharge.test.ts） ──

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

const mockQueryOrder = vi.fn();
const mockCancelPayment = vi.fn();
const stubProvider = {
  name: 'easy-pay',
  providerKey: 'easypay',
  queryOrder: (...args: unknown[]) => mockQueryOrder(...args),
  cancelPayment: (...args: unknown[]) => mockCancelPayment(...args),
};

vi.mock('@/lib/payment', () => ({
  initPaymentProviders: vi.fn(),
  ensureDBProviders: vi.fn().mockResolvedValue(undefined),
  paymentRegistry: { getProvider: () => stubProvider },
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

import { cancelOrderCore } from '@/lib/order/service';

function makeOrder(overrides: Record<string, unknown> = {}) {
  return {
    id: 'order-001',
    userId: 42,
    amount: new Prisma.Decimal('100.00'),
    payAmount: new Prisma.Decimal('100.00'),
    rechargeCode: 'RC-ORDER-001',
    status: ORDER_STATUS.PENDING,
    orderType: 'balance',
    bonusAmount: null,
    ...overrides,
  };
}

const baseOptions = {
  orderId: 'order-001',
  paymentTradeNo: 'T-1',
  paymentType: 'alipay',
  finalStatus: ORDER_STATUS.CANCELLED,
  operator: 'admin',
  auditDetail: '管理员取消订单',
} as const;

function auditActions(): string[] {
  return mockAuditLogCreate.mock.calls.map((call) => (call[0] as { data: { action: string } }).data.action);
}

function auditByAction(action: string) {
  const call = mockAuditLogCreate.mock.calls.find((c) => (c[0] as { data: { action: string } }).data.action === action);
  return call ? (call[0] as { data: { action: string; operator: string; detail: string } }).data : undefined;
}

describe('cancelOrderCore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    mockAuditLogCreate.mockResolvedValue({});
    mockOrderUpdate.mockResolvedValue({});
    mockOrderUpdateMany.mockResolvedValue({ count: 1 });
  });

  it('平台未付款：关平台单、本地置 CANCELLED 并写审计', async () => {
    mockQueryOrder.mockResolvedValue({ tradeNo: 'T-1', status: 'pending', amount: 100 });
    mockCancelPayment.mockResolvedValue(undefined);

    const outcome = await cancelOrderCore(baseOptions);

    expect(outcome).toBe('cancelled');
    expect(mockCancelPayment).toHaveBeenCalledWith('T-1');
    expect(mockOrderUpdateMany).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { id: 'order-001', status: ORDER_STATUS.PENDING },
        data: expect.objectContaining({ status: ORDER_STATUS.CANCELLED }),
      }),
    );
    expect(auditActions()).toEqual(['ORDER_CANCELLED']);
  });

  it('平台已付款且入账成功：按 cancel 来源履约，订单不取消', async () => {
    mockQueryOrder.mockResolvedValue({ tradeNo: 'T-1', status: 'paid', amount: 100 });
    mockOrderFindUnique
      .mockResolvedValueOnce(makeOrder())
      .mockResolvedValueOnce({ orderType: 'balance' })
      .mockResolvedValueOnce(makeOrder({ status: ORDER_STATUS.PAID }));
    mockCreateAndRedeem.mockResolvedValue({ id: 1 });

    const outcome = await cancelOrderCore(baseOptions);

    expect(outcome).toBe('already_paid');
    expect(auditByAction('ORDER_PAID')?.operator).toBe('easy-pay:cancel');
    expect(auditActions()).not.toContain('ORDER_CANCELLED');
    const cancelUpdate = mockOrderUpdateMany.mock.calls.find(
      (c) => (c[0] as { data: { status: string } }).data.status === ORDER_STATUS.CANCELLED,
    );
    expect(cancelUpdate).toBeUndefined();
  });

  it('到期扫描路径的来源是 expire', async () => {
    mockQueryOrder.mockResolvedValue({ tradeNo: 'T-1', status: 'paid', amount: 100 });
    mockOrderFindUnique
      .mockResolvedValueOnce(makeOrder())
      .mockResolvedValueOnce({ orderType: 'balance' })
      .mockResolvedValueOnce(makeOrder({ status: ORDER_STATUS.PAID }));
    mockCreateAndRedeem.mockResolvedValue({ id: 1 });

    const outcome = await cancelOrderCore({ ...baseOptions, finalStatus: ORDER_STATUS.EXPIRED, operator: 'timeout' });

    expect(outcome).toBe('already_paid');
    expect(auditByAction('ORDER_PAID')?.operator).toBe('easy-pay:expire');
  });

  it('平台已付款但入账失败（金额不符）：返回 paid_unconfirmed，订单一动不动', async () => {
    mockQueryOrder.mockResolvedValue({ tradeNo: 'T-1', status: 'paid', amount: 88 });
    mockOrderFindUnique.mockResolvedValue(makeOrder());

    const outcome = await cancelOrderCore(baseOptions);

    expect(outcome).toBe('paid_unconfirmed');
    expect(mockOrderUpdateMany).not.toHaveBeenCalled();
    expect(auditActions()).toEqual(['PAYMENT_AMOUNT_MISMATCH']);
  });

  it('平台查单失败、默认策略：本地照常取消', async () => {
    mockQueryOrder.mockRejectedValue(new Error('gateway timeout'));

    const outcome = await cancelOrderCore(baseOptions);

    expect(outcome).toBe('cancelled');
    expect(mockOrderUpdateMany).toHaveBeenCalledTimes(1);
    expect(auditActions()).toEqual(['ORDER_CANCELLED']);
  });

  it('平台查单失败、skip 策略：本轮跳过，订单保持 PENDING', async () => {
    mockQueryOrder.mockRejectedValue(new Error('gateway timeout'));

    const outcome = await cancelOrderCore({
      ...baseOptions,
      finalStatus: ORDER_STATUS.EXPIRED,
      onPlatformError: 'skip',
    });

    expect(outcome).toBe('platform_unavailable');
    expect(mockOrderUpdateMany).not.toHaveBeenCalled();
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });

  it('没有平台单号时直接本地取消，不查平台', async () => {
    const outcome = await cancelOrderCore({ ...baseOptions, paymentTradeNo: null });

    expect(outcome).toBe('cancelled');
    expect(mockQueryOrder).not.toHaveBeenCalled();
  });

  it('订单已不是 PENDING（CAS 未命中）：幂等返回 cancelled 且不写审计', async () => {
    mockQueryOrder.mockResolvedValue({ tradeNo: 'T-1', status: 'pending', amount: 100 });
    mockOrderUpdateMany.mockResolvedValue({ count: 0 });

    const outcome = await cancelOrderCore(baseOptions);

    expect(outcome).toBe('cancelled');
    expect(mockAuditLogCreate).not.toHaveBeenCalled();
  });
});
