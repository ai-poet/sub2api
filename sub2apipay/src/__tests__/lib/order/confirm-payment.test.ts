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

import { confirmPayment, handlePaymentNotify } from '@/lib/order/service';

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
    promotionId: null,
    promotionName: null,
    ...overrides,
  };
}

/** 让 confirmPayment 之后的履约链路（executeFulfillment → executeRecharge）全部成功 */
function arrangeSuccessfulFulfillment(initial: ReturnType<typeof makeOrder>) {
  mockOrderFindUnique
    .mockResolvedValueOnce(initial) // confirmPayment 读订单
    .mockResolvedValueOnce({ orderType: 'balance' }) // executeFulfillment 读 orderType
    .mockResolvedValueOnce(makeOrder({ status: ORDER_STATUS.PAID })); // executeRecharge 读订单
  mockOrderUpdateMany.mockResolvedValue({ count: 1 });
  mockCreateAndRedeem.mockResolvedValue({ id: 1 });
}

function auditActions(): string[] {
  return mockAuditLogCreate.mock.calls.map((call) => (call[0] as { data: { action: string } }).data.action);
}

function auditByAction(action: string) {
  const call = mockAuditLogCreate.mock.calls.find((c) => (c[0] as { data: { action: string } }).data.action === action);
  return call ? (call[0] as { data: { action: string; operator: string; detail: string } }).data : undefined;
}

describe('confirmPayment', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuditLogCreate.mockResolvedValue({});
    mockOrderUpdate.mockResolvedValue({});
  });

  it('PENDING 订单：转 PAID 并履约，ORDER_PAID 的 operator 带触发来源', async () => {
    arrangeSuccessfulFulfillment(makeOrder());

    const ok = await confirmPayment({
      orderId: 'order-001',
      tradeNo: 'T-1',
      paidAmount: 100,
      providerName: 'easy-pay',
      source: 'notify',
    });

    expect(ok).toBe(true);
    const firstUpdate = mockOrderUpdateMany.mock.calls[0][0] as {
      where: { status: { in: string[] } };
      data: { status: string };
    };
    expect(firstUpdate.data.status).toBe(ORDER_STATUS.PAID);
    expect(firstUpdate.where.status.in).toEqual([ORDER_STATUS.PENDING, ORDER_STATUS.EXPIRED, ORDER_STATUS.CANCELLED]);
    const paid = auditByAction('ORDER_PAID');
    expect(paid?.operator).toBe('easy-pay:notify');
    expect(JSON.parse(paid!.detail)).toMatchObject({ previous_status: ORDER_STATUS.PENDING, source: 'notify' });
    expect(mockCreateAndRedeem).toHaveBeenCalledTimes(1);
  });

  it('未指定来源时 operator 只写 provider 名（兼容旧调用）', async () => {
    arrangeSuccessfulFulfillment(makeOrder());

    await confirmPayment({ orderId: 'order-001', tradeNo: 'T-1', paidAmount: 100, providerName: 'easy-pay' });

    expect(auditByAction('ORDER_PAID')?.operator).toBe('easy-pay');
  });

  it.each([ORDER_STATUS.EXPIRED, ORDER_STATUS.CANCELLED])('%s 订单收到已验签的付款也照收并履约', async (status) => {
    arrangeSuccessfulFulfillment(makeOrder({ status }));

    const ok = await confirmPayment({
      orderId: 'order-001',
      tradeNo: 'T-1',
      paidAmount: 100,
      providerName: 'easy-pay',
      source: 'notify',
    });

    expect(ok).toBe(true);
    expect(JSON.parse(auditByAction('ORDER_PAID')!.detail)).toMatchObject({ previous_status: status });
    expect(mockCreateAndRedeem).toHaveBeenCalledTimes(1);
  });

  it('金额不是有限正数：写 PAYMENT_NOTIFY_REJECTED 审计并返回 false，不动订单', async () => {
    mockOrderFindUnique.mockResolvedValue(makeOrder());

    for (const paidAmount of [Number.NaN, Number.POSITIVE_INFINITY, 0, -1]) {
      mockAuditLogCreate.mockClear();
      const ok = await confirmPayment({
        orderId: 'order-001',
        tradeNo: 'T-1',
        paidAmount,
        providerName: 'easy-pay',
        source: 'notify',
      });
      expect(ok).toBe(false);
      expect(auditActions()).toEqual(['PAYMENT_NOTIFY_REJECTED']);
      expect(auditByAction('PAYMENT_NOTIFY_REJECTED')?.operator).toBe('easy-pay:notify');
    }
    expect(mockOrderUpdateMany).not.toHaveBeenCalled();
  });

  it('金额相差超过 0.01：写 PAYMENT_AMOUNT_MISMATCH 并返回 false', async () => {
    mockOrderFindUnique.mockResolvedValue(makeOrder());

    const ok = await confirmPayment({
      orderId: 'order-001',
      tradeNo: 'T-1',
      paidAmount: 99,
      providerName: 'easy-pay',
      source: 'cancel',
    });

    expect(ok).toBe(false);
    expect(auditByAction('PAYMENT_AMOUNT_MISMATCH')?.operator).toBe('easy-pay:cancel');
    expect(mockOrderUpdateMany).not.toHaveBeenCalled();
  });

  it('订单已被另一路径推进到 PAID / RECHARGING：返回 false 让平台稍后重试', async () => {
    mockOrderFindUnique.mockResolvedValueOnce(makeOrder()).mockResolvedValueOnce({ status: ORDER_STATUS.RECHARGING });
    mockOrderUpdateMany.mockResolvedValue({ count: 0 });

    const ok = await confirmPayment({
      orderId: 'order-001',
      tradeNo: 'T-1',
      paidAmount: 100,
      providerName: 'easy-pay',
    });

    expect(ok).toBe(false);
    expect(mockCreateAndRedeem).not.toHaveBeenCalled();
  });

  it('订单已 COMPLETED：直接返回 true 停止平台重试', async () => {
    mockOrderFindUnique
      .mockResolvedValueOnce(makeOrder({ status: ORDER_STATUS.COMPLETED }))
      .mockResolvedValueOnce({ status: ORDER_STATUS.COMPLETED });
    mockOrderUpdateMany.mockResolvedValue({ count: 0 });

    const ok = await confirmPayment({
      orderId: 'order-001',
      tradeNo: 'T-1',
      paidAmount: 100,
      providerName: 'easy-pay',
    });

    expect(ok).toBe(true);
  });
});

describe('handlePaymentNotify', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuditLogCreate.mockResolvedValue({});
  });

  it('成功通知以 notify 来源进入 confirmPayment', async () => {
    arrangeSuccessfulFulfillment(makeOrder());

    const ok = await handlePaymentNotify(
      { orderId: 'order-001', tradeNo: 'T-1', amount: 100, status: 'success', rawData: {} },
      'easy-pay',
    );

    expect(ok).toBe(true);
    expect(auditByAction('ORDER_PAID')?.operator).toBe('easy-pay:notify');
  });

  it('非成功状态的通知不入账、不报错，只记日志', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});

    const ok = await handlePaymentNotify(
      { orderId: 'order-001', tradeNo: 'T-1', amount: 100, status: 'failed', rawData: { trade_status: 'WAIT' } },
      'easy-pay',
    );

    expect(ok).toBe(true);
    expect(mockOrderFindUnique).not.toHaveBeenCalled();
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('order-001'), expect.anything());
    warn.mockRestore();
  });
});
