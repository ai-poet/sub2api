import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockAuditGroupBy = vi.fn();
const mockAuditFindMany = vi.fn();
const mockAuditCount = vi.fn();
const mockOrderFindMany = vi.fn();
const mockOrderCount = vi.fn();
const mockQueryRaw = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    auditLog: {
      groupBy: (...args: unknown[]) => mockAuditGroupBy(...args),
      findMany: (...args: unknown[]) => mockAuditFindMany(...args),
      count: (...args: unknown[]) => mockAuditCount(...args),
    },
    order: {
      findMany: (...args: unknown[]) => mockOrderFindMany(...args),
      count: (...args: unknown[]) => mockOrderCount(...args),
    },
    $queryRaw: (...args: unknown[]) => mockQueryRaw(...args),
  },
  getConfiguredDatabaseSchema: () => 'public',
}));

import { getPaymentRiskOverview, parsePaymentRiskQuery } from '@/lib/payment-risk/overview';
import { PAYMENT_RISK_ACTIONS } from '@/lib/payment-risk/shared';

// 业务时区 2026-10-07 12:00
const NOW = new Date('2026-10-07T04:00:00.000Z');

function orderRow(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    userId: 7,
    userName: 'mallory',
    userEmail: 'm@example.com',
    amount: 650,
    payAmount: 650,
    status: 'COMPLETED',
    paymentType: 'alipay',
    orderType: 'balance',
    paymentTradeNo: '',
    createdAt: new Date('2026-10-07T02:00:00.000Z'),
    paidAt: new Date('2026-10-07T02:01:00.000Z'),
    ...overrides,
  };
}

describe('parsePaymentRiskQuery', () => {
  it('默认 30 天、第一页、每页 20 条、不筛选', () => {
    expect(parsePaymentRiskQuery(new URLSearchParams())).toEqual({ days: 30, action: null, page: 1, pageSize: 20 });
  });

  it('越界和非法值被收紧或回落', () => {
    expect(parsePaymentRiskQuery(new URLSearchParams('days=0&page=-3&page_size=1000'))).toEqual({
      days: 1,
      action: null,
      page: 1,
      pageSize: 100,
    });
    expect(parsePaymentRiskQuery(new URLSearchParams('days=9999')).days).toBe(365);
    expect(parsePaymentRiskQuery(new URLSearchParams('days=abc')).days).toBe(30);
    expect(parsePaymentRiskQuery(new URLSearchParams('days=7.9')).days).toBe(7);
  });

  it('只接受已知的风险事件类型', () => {
    expect(parsePaymentRiskQuery(new URLSearchParams('action=PAYMENT_NOTIFY_BLOCKED')).action).toBe(
      'PAYMENT_NOTIFY_BLOCKED',
    );
    expect(parsePaymentRiskQuery(new URLSearchParams('action=ORDER_PAID')).action).toBeNull();
    expect(parsePaymentRiskQuery(new URLSearchParams("action=x'%20OR%201=1")).action).toBeNull();
  });
});

describe('getPaymentRiskOverview', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockAuditGroupBy.mockResolvedValue([
      { action: 'PAYMENT_NOTIFY_BLOCKED', _count: { _all: 4 } },
      { action: 'PAYMENT_UPSTREAM_MISMATCH', _count: { _all: 1 } },
      { action: 'ORDER_PAID', _count: { _all: 99 } },
    ]);
    mockAuditFindMany.mockResolvedValue([
      {
        id: 'audit-1',
        action: 'PAYMENT_UPSTREAM_MISMATCH',
        detail: '{"reason":"upstream_not_paid"}',
        operator: 'easy-pay:verify',
        createdAt: new Date('2026-10-07T03:00:00.000Z'),
        order: orderRow('o-1'),
      },
    ]);
    mockAuditCount.mockResolvedValue(5);
    mockOrderFindMany.mockImplementation((args: { where: { id?: unknown } }) =>
      Promise.resolve(
        args.where.id
          ? // 重复交易号的订单，故意倒序返回，结果应按 order_ids 的顺序排列
            [orderRow('o-3', { paymentTradeNo: 'T-DUP' }), orderRow('o-2', { paymentTradeNo: 'T-DUP' })]
          : [orderRow('o-1', { paymentTradeNo: null, paidAt: new Date('2026-10-06T08:00:00.000Z') })],
      ),
    );
    mockOrderCount.mockResolvedValue(1);
    mockQueryRaw
      // 按天分布
      .mockResolvedValueOnce([
        { date: '2026-10-05', action: 'PAYMENT_NOTIFY_BLOCKED', count: BigInt(3) },
        { date: '2026-10-07', action: 'PAYMENT_NOTIFY_BLOCKED', count: BigInt(1) },
        { date: '2026-10-07', action: 'PAYMENT_UPSTREAM_MISMATCH', count: BigInt(1) },
        { date: '2026-10-07', action: 'ORDER_PAID', count: BigInt(50) },
      ])
      // 重复交易号
      .mockResolvedValueOnce([{ trade_no: 'T-DUP', order_ids: ['o-2', 'o-3'], count: BigInt(2) }])
      // 可疑用户
      .mockResolvedValueOnce([
        {
          user_id: 7,
          user_name: 'mallory',
          user_email: 'm@example.com',
          event_count: BigInt(5),
          order_count: BigInt(2),
          last_at: new Date('2026-10-07T03:00:00.000Z'),
        },
      ]);
  });

  it('按业务时区自然日计算窗口，并按天补齐', async () => {
    const overview = await getPaymentRiskOverview({ days: 3, action: null, page: 1, pageSize: 20 }, NOW);

    expect(overview.meta.since).toBe('2026-10-04T16:00:00.000Z');
    expect(overview.meta.generatedAt).toBe(NOW.toISOString());
    expect(overview.daily.map((point) => point.date)).toEqual(['2026-10-05', '2026-10-06', '2026-10-07']);
    expect(overview.daily[0]).toMatchObject({ PAYMENT_NOTIFY_BLOCKED: 3, PAYMENT_UPSTREAM_MISMATCH: 0 });
    expect(overview.daily[1]).toMatchObject({ PAYMENT_NOTIFY_BLOCKED: 0 });
    expect(overview.daily[2]).toMatchObject({ PAYMENT_NOTIFY_BLOCKED: 1, PAYMENT_UPSTREAM_MISMATCH: 1 });
    for (const point of overview.daily) {
      expect(Object.keys(point).sort()).toEqual(['date', ...PAYMENT_RISK_ACTIONS].sort());
    }

    const since = new Date('2026-10-04T16:00:00.000Z');
    expect(mockAuditGroupBy).toHaveBeenCalledWith({
      by: ['action'],
      where: { action: { in: [...PAYMENT_RISK_ACTIONS] }, createdAt: { gte: since } },
      _count: { _all: true },
    });
  });

  it('汇总只统计风险事件，其余审计动作被忽略', async () => {
    const overview = await getPaymentRiskOverview({ days: 3, action: null, page: 1, pageSize: 20 }, NOW);

    expect(overview.summary.events).toEqual({
      PAYMENT_UPSTREAM_MISMATCH: 1,
      PAYMENT_NOTIFY_BLOCKED: 4,
      PAYMENT_NOTIFY_ANOMALY: 0,
      PAYMENT_AMOUNT_MISMATCH: 0,
      PAYMENT_NOTIFY_REJECTED: 0,
      PAYMENT_CONFIRM_FAILED: 0,
    });
    expect(overview.summary.paidWithoutTradeNo).toBe(1);
    expect(overview.summary.duplicateTradeNoGroups).toBe(1);
  });

  it('事件列表分页、按类型筛选，金额和时间转成可序列化的值', async () => {
    const overview = await getPaymentRiskOverview(
      { days: 3, action: 'PAYMENT_UPSTREAM_MISMATCH', page: 2, pageSize: 2 },
      NOW,
    );

    expect(mockAuditFindMany).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { action: 'PAYMENT_UPSTREAM_MISMATCH', createdAt: { gte: new Date('2026-10-04T16:00:00.000Z') } },
        orderBy: { createdAt: 'desc' },
        skip: 2,
        take: 2,
      }),
    );
    expect(overview.events).toMatchObject({
      action: 'PAYMENT_UPSTREAM_MISMATCH',
      total: 5,
      page: 2,
      pageSize: 2,
      totalPages: 3,
    });
    expect(overview.events.items).toEqual([
      {
        id: 'audit-1',
        action: 'PAYMENT_UPSTREAM_MISMATCH',
        detail: '{"reason":"upstream_not_paid"}',
        operator: 'easy-pay:verify',
        createdAt: '2026-10-07T03:00:00.000Z',
        order: {
          id: 'o-1',
          userId: 7,
          userName: 'mallory',
          userEmail: 'm@example.com',
          amount: 650,
          payAmount: 650,
          status: 'COMPLETED',
          paymentType: 'alipay',
          orderType: 'balance',
          paymentTradeNo: '',
          createdAt: '2026-10-07T02:00:00.000Z',
          paidAt: '2026-10-07T02:01:00.000Z',
        },
      },
    ]);
  });

  it('无交易号入账、重复交易号与可疑用户', async () => {
    const overview = await getPaymentRiskOverview({ days: 3, action: null, page: 1, pageSize: 20 }, NOW);

    expect(mockOrderFindMany).toHaveBeenCalledWith(
      expect.objectContaining({
        where: {
          paidAt: { gte: new Date('2026-10-04T16:00:00.000Z') },
          OR: [{ paymentTradeNo: null }, { paymentTradeNo: '' }],
        },
        take: 50,
      }),
    );
    expect(overview.paidWithoutTradeNo).toHaveLength(1);
    expect(overview.paidWithoutTradeNo[0]).toMatchObject({ id: 'o-1', paymentTradeNo: null, paidAt: '2026-10-06T08:00:00.000Z' });

    expect(mockOrderFindMany).toHaveBeenCalledWith(expect.objectContaining({ where: { id: { in: ['o-2', 'o-3'] } } }));
    expect(overview.duplicateTradeNos).toHaveLength(1);
    expect(overview.duplicateTradeNos[0].tradeNo).toBe('T-DUP');
    expect(overview.duplicateTradeNos[0].count).toBe(2);
    expect(overview.duplicateTradeNos[0].orders.map((order) => order.id)).toEqual(['o-2', 'o-3']);

    expect(overview.topUsers).toEqual([
      {
        userId: 7,
        userName: 'mallory',
        userEmail: 'm@example.com',
        eventCount: 5,
        orderCount: 2,
        lastEventAt: '2026-10-07T03:00:00.000Z',
      },
    ]);
  });

  it('没有重复交易号时不额外查订单', async () => {
    mockQueryRaw.mockReset();
    mockQueryRaw.mockResolvedValueOnce([]).mockResolvedValueOnce([]).mockResolvedValueOnce([]);

    const overview = await getPaymentRiskOverview({ days: 1, action: null, page: 1, pageSize: 20 }, NOW);

    expect(mockOrderFindMany).toHaveBeenCalledTimes(1);
    expect(overview.duplicateTradeNos).toEqual([]);
    expect(overview.daily).toHaveLength(1);
    expect(overview.daily[0].date).toBe('2026-10-07');
  });
});
