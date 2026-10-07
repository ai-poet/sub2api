import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockAuditCount = vi.fn();
const mockAuditFindMany = vi.fn();
const mockAuditFindFirst = vi.fn();
const mockAuditCreate = vi.fn();
const mockAuditUpdateMany = vi.fn();
const mockGetInstanceConfig = vi.fn();
const mockEasyPayConstructed = vi.fn();
const mockEasyPayQueryOrder = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    auditLog: {
      count: (...args: unknown[]) => mockAuditCount(...args),
      findMany: (...args: unknown[]) => mockAuditFindMany(...args),
      findFirst: (...args: unknown[]) => mockAuditFindFirst(...args),
      create: (...args: unknown[]) => mockAuditCreate(...args),
      updateMany: (...args: unknown[]) => mockAuditUpdateMany(...args),
    },
  },
  getConfiguredDatabaseSchema: () => 'public',
}));

const mockIsGatewayDbConfigured = vi.fn();
const mockFindGatewayOrder = vi.fn();

vi.mock('@/lib/easy-pay/gateway-db', () => ({
  isGatewayDbConfigured: () => mockIsGatewayDbConfigured(),
  findGatewayOrder: (...args: unknown[]) => mockFindGatewayOrder(...args),
  gatewayMerchantIds: () => new Set(['1001']),
}));

vi.mock('@/lib/payment/load-balancer', () => ({
  getInstanceConfig: (...args: unknown[]) => mockGetInstanceConfig(...args),
}));

vi.mock('@/lib/easy-pay/provider', () => ({
  EasyPayProvider: class {
    readonly name: string;
    constructor(instanceId?: string, config?: Record<string, string>) {
      mockEasyPayConstructed(instanceId, config);
      this.name = instanceId ? `easy-pay:${instanceId}` : 'easy-pay';
    }
    queryOrder(orderId: string) {
      return mockEasyPayQueryOrder(this.name, orderId);
    }
  },
}));

import { computeRiskWindowStart } from '@/lib/payment-risk/overview';
import {
  instanceIdFromOperator,
  parseRecheckRequest,
  recheckCandidateWhere,
  recheckEasyPayOrders,
} from '@/lib/payment-risk/recheck';
import type { QueryOrderResponse } from '@/lib/payment/types';

const NOW = new Date('2026-10-07T04:00:00.000Z');
const SINCE = new Date('2026-09-08T16:00:00.000Z');

function row(id: string, orderId: string, overrides: Record<string, unknown> = {}, operator = 'easy-pay:inst-1:notify') {
  return {
    id,
    operator,
    order: {
      id: orderId,
      userId: 7,
      userName: 'mallory',
      userEmail: 'm@example.com',
      amount: 100,
      payAmount: 10,
      paymentType: 'alipay',
      status: 'COMPLETED',
      paidAt: new Date('2026-10-01T02:00:00.000Z'),
      paymentTradeNo: `T-${orderId}`,
      providerInstanceId: 'inst-1',
      ...overrides,
    },
  };
}

const paid = (orderId: string, overrides: Partial<QueryOrderResponse> = {}): QueryOrderResponse => ({
  tradeNo: `T-${orderId}`,
  status: 'paid',
  amount: 10,
  ...overrides,
});

describe('parseRecheckRequest', () => {
  it('默认复核最近 30 天、从头开始、每批 10 笔', () => {
    expect(parseRecheckRequest({}, NOW)).toEqual({ since: computeRiskWindowStart(30, NOW), cursor: null, batchSize: 10 });
    expect(parseRecheckRequest(null, NOW)).toEqual({ since: computeRiskWindowStart(30, NOW), cursor: null, batchSize: 10 });
  });

  it('第一批按 days 计算窗口，之后各批沿用 since 与 cursor', () => {
    expect(parseRecheckRequest({ days: 365, batch_size: 20 }, NOW)).toEqual({
      since: computeRiskWindowStart(365, NOW),
      cursor: null,
      batchSize: 20,
    });
    expect(parseRecheckRequest({ since: SINCE.toISOString(), cursor: 'cm1abc_DEF-9' }, NOW)).toEqual({
      since: SINCE,
      cursor: 'cm1abc_DEF-9',
      batchSize: 10,
    });
  });

  it('参数不合法返回 null', () => {
    for (const body of [
      { days: 0 },
      { days: 366 },
      { days: 1.5 },
      { days: '7' },
      { batch_size: 21 },
      { cursor: "x' OR 1=1" },
      { since: 'yesterday' },
      { since: '2026-10-08T00:00:00.000Z' },
      { since: '2025-10-01T00:00:00.000Z' },
    ]) {
      expect(parseRecheckRequest(body, NOW)).toBeNull();
    }
  });
});

describe('recheckCandidateWhere / instanceIdFromOperator', () => {
  it('只挑易支付入账、且不是查单确认的', () => {
    expect(recheckCandidateWhere(SINCE)).toEqual({
      action: 'ORDER_PAID',
      createdAt: { gte: SINCE },
      operator: { startsWith: 'easy-pay' },
      NOT: [
        { operator: { endsWith: ':poll' } },
        { operator: { endsWith: ':sweep' } },
        { operator: { endsWith: ':cancel' } },
        { operator: { endsWith: ':expire' } },
      ],
    });
  });

  it('从 operator 取实例 id', () => {
    expect(instanceIdFromOperator('easy-pay:inst-1:notify')).toBe('inst-1');
    expect(instanceIdFromOperator('easy-pay:inst-1')).toBe('inst-1');
    expect(instanceIdFromOperator('easy-pay:notify')).toBeNull();
    expect(instanceIdFromOperator('easy-pay')).toBeNull();
    expect(instanceIdFromOperator('alipay:notify')).toBeNull();
    expect(instanceIdFromOperator(null)).toBeNull();
  });
});

describe('recheckEasyPayOrders', () => {
  const queryOrder = vi.fn();
  const provider = { name: 'easy-pay:inst-1', queryOrder: (...args: unknown[]) => queryOrder(...args) };
  const resolveProvider = vi.fn();
  const sleep = vi.fn(() => Promise.resolve());

  beforeEach(() => {
    vi.clearAllMocks();
    resolveProvider.mockImplementation((instanceId: string | null) =>
      Promise.resolve(instanceId === 'gone' ? null : provider),
    );
    mockAuditCount.mockResolvedValue(42);
    mockAuditFindFirst.mockResolvedValue(null);
    mockAuditCreate.mockResolvedValue({});
    mockAuditUpdateMany.mockResolvedValue({ count: 0 });
    mockIsGatewayDbConfigured.mockReturnValue(false);
  });

  it('查实一致的订单撤销它之前的不符记录，并另记一条撤销记录', async () => {
    mockAuditFindMany.mockResolvedValue([row('a1', 'o1')]);
    queryOrder.mockResolvedValue(paid('o1'));
    mockAuditUpdateMany.mockResolvedValue({ count: 2 });

    const batch = await recheckEasyPayOrders({ since: SINCE, cursor: null, batchSize: 10 }, { resolveProvider, sleep });

    expect(batch.results[0]).toMatchObject({ orderId: 'o1', outcome: 'ok', resolved: 2 });
    expect(mockAuditUpdateMany).toHaveBeenCalledWith({
      where: { orderId: 'o1', action: 'PAYMENT_UPSTREAM_MISMATCH' },
      data: { action: 'PAYMENT_UPSTREAM_MISMATCH_RESOLVED' },
    });
    expect(mockAuditCreate).toHaveBeenCalledTimes(1);
    const { data } = mockAuditCreate.mock.calls[0][0];
    expect(data).toMatchObject({ orderId: 'o1', action: 'PAYMENT_UPSTREAM_RESOLVED', operator: 'easy-pay:inst-1:recheck' });
    expect(JSON.parse(data.detail)).toMatchObject({ resolved: 2, checkedVia: 'merchant_api', upstreamStatus: 'paid' });
  });

  it('没有旧的不符记录时不写撤销记录；撤销失败也不改变"一致"的结论', async () => {
    mockAuditFindMany.mockResolvedValue([row('a1', 'o1'), row('a2', 'o2')]);
    queryOrder.mockImplementation((orderId: string) => Promise.resolve(paid(orderId)));
    mockAuditUpdateMany.mockResolvedValueOnce({ count: 0 }).mockRejectedValueOnce(new Error('db down'));

    const batch = await recheckEasyPayOrders({ since: SINCE, cursor: null, batchSize: 10 }, { resolveProvider, sleep });

    expect(batch.results[0]).toEqual(expect.objectContaining({ orderId: 'o1', outcome: 'ok' }));
    expect(batch.results[0].resolved).toBeUndefined();
    expect(batch.results[1]).toMatchObject({ orderId: 'o2', outcome: 'ok' });
    expect(batch.results[1].message).toContain('撤销旧的不符记录失败');
    expect(mockAuditCreate).not.toHaveBeenCalled();
  });

  it('配置了网关库：商户换号后查不到的老订单经网关库确认，不再记为不符', async () => {
    mockIsGatewayDbConfigured.mockReturnValue(true);
    mockAuditFindMany.mockResolvedValue([row('a1', 'o1'), row('a2', 'o2')]);
    queryOrder.mockRejectedValue(new Error('EasyPay query order failed: 订单号不存在'));
    mockFindGatewayOrder.mockImplementation((orderId: string) =>
      Promise.resolve(
        orderId === 'o1'
          ? { outTradeNo: 'o1', tradeNo: 'T-o1', merchantId: '1001', amount: 10, status: '1', paid: true }
          : null,
      ),
    );

    const batch = await recheckEasyPayOrders({ since: SINCE, cursor: null, batchSize: 10 }, { resolveProvider, sleep });

    expect(batch.results.map((item) => [item.orderId, item.outcome, item.reason ?? null])).toEqual([
      ['o1', 'ok', null],
      ['o2', 'mismatch', 'upstream_order_not_found'],
    ]);
    expect(mockAuditCreate).toHaveBeenCalledTimes(1);
    expect(JSON.parse(mockAuditCreate.mock.calls[0][0].data.detail)).toMatchObject({
      reason: 'upstream_order_not_found',
      checkedVia: 'gateway_db',
      upstreamMessage: '订单号不存在；网关库中也没有这笔订单',
    });
  });

  it('一批里的各种结果：一致、不符、查无此单、查单失败、实例不存在', async () => {
    mockAuditFindMany.mockResolvedValue([
      row('a1', 'o1'),
      row('a2', 'o2'),
      row('a3', 'o3'),
      row('a4', 'o4'),
      row('a5', 'o5', { providerInstanceId: 'gone' }),
    ]);
    queryOrder.mockImplementation((orderId: string) => {
      if (orderId === 'o1') return Promise.resolve(paid('o1'));
      if (orderId === 'o2') return Promise.resolve(paid('o2', { status: 'pending' }));
      if (orderId === 'o3') return Promise.reject(new Error('EasyPay query order failed: 订单编号不存在'));
      return Promise.reject(new Error('request to https://pay.example.com/api.php?act=order&key=SECRET failed'));
    });

    const batch = await recheckEasyPayOrders(
      { since: SINCE, cursor: null, batchSize: 10 },
      { resolveProvider, sleep, queryGapMs: 300 },
    );

    expect(batch).toMatchObject({ since: SINCE.toISOString(), total: 42, processed: 5, done: true, nextCursor: null });
    expect(batch.results.map((item) => [item.orderId, item.outcome, item.reason ?? null])).toEqual([
      ['o1', 'ok', null],
      ['o2', 'mismatch', 'upstream_not_paid'],
      ['o3', 'mismatch', 'upstream_order_not_found'],
      ['o4', 'failed', null],
      ['o5', 'skipped', 'provider_unavailable'],
    ]);
    expect(batch.results[0]).toMatchObject({
      userId: 7,
      userName: 'mallory',
      amount: 10,
      paymentType: 'alipay',
      orderStatus: 'COMPLETED',
      paidAt: '2026-10-01T02:00:00.000Z',
    });
    expect(batch.results[1].recorded).toBe(true);
    expect(batch.results[2].message).toBe('订单编号不存在');
    // 查单失败的原因不能带出商户密钥
    expect(batch.results[3].message).toContain('key=***');
    expect(batch.results[3].message).not.toContain('SECRET');

    // 用入账时记下的支付金额与交易号比对
    expect(queryOrder).toHaveBeenCalledWith('o1');
    // 两笔不符各写一条，operator 标明来自复核
    expect(mockAuditCreate).toHaveBeenCalledTimes(2);
    expect(mockAuditCreate.mock.calls.map((call) => call[0].data.orderId)).toEqual(['o2', 'o3']);
    expect(mockAuditCreate.mock.calls[0][0].data.operator).toBe('easy-pay:inst-1:recheck');
    expect(JSON.parse(mockAuditCreate.mock.calls[0][0].data.detail)).toMatchObject({
      reason: 'upstream_not_paid',
      source: 'recheck',
      notifyAmount: 10,
      notifyTradeNo: 'T-o2',
    });
    // 笔与笔之间留间隔
    expect(sleep).toHaveBeenCalledTimes(4);
    expect(sleep).toHaveBeenCalledWith(300);
    // 同一实例只解析一次
    expect(resolveProvider.mock.calls).toEqual([['inst-1'], ['gone']]);
  });

  it('同一订单同一原因之前已记录：不重复写', async () => {
    mockAuditFindMany.mockResolvedValue([row('a2', 'o2')]);
    queryOrder.mockResolvedValue(paid('o2', { status: 'pending' }));
    mockAuditFindFirst.mockResolvedValue({ id: 'audit-1' });

    const batch = await recheckEasyPayOrders({ since: SINCE, cursor: null, batchSize: 10 }, { resolveProvider, sleep });

    expect(batch.results[0]).toMatchObject({ outcome: 'mismatch', recorded: false });
    expect(mockAuditCreate).not.toHaveBeenCalled();
  });

  it('按游标续跑，按时间正序', async () => {
    mockAuditFindMany.mockResolvedValue([]);

    const batch = await recheckEasyPayOrders({ since: SINCE, cursor: 'a10', batchSize: 5 }, { resolveProvider, sleep });

    expect(mockAuditFindMany).toHaveBeenCalledWith({
      where: recheckCandidateWhere(SINCE),
      orderBy: [{ createdAt: 'asc' }, { id: 'asc' }],
      cursor: { id: 'a10' },
      skip: 1,
      take: 5,
      select: expect.objectContaining({ id: true, operator: true }),
    });
    expect(batch).toMatchObject({ processed: 0, done: true, nextCursor: null });
  });

  it('取满一批：未完成，下一批从最后一笔之后继续', async () => {
    mockAuditFindMany.mockResolvedValue([row('a1', 'o1'), row('a2', 'o2')]);
    queryOrder.mockImplementation((orderId: string) => Promise.resolve(paid(orderId)));

    const batch = await recheckEasyPayOrders({ since: SINCE, cursor: null, batchSize: 2 }, { resolveProvider, sleep });

    expect(batch).toMatchObject({ processed: 2, done: false, nextCursor: 'a2' });
  });

  it('超过单次时间上限：剩下的留给下一批，但每批至少处理一笔', async () => {
    mockAuditFindMany.mockResolvedValue([row('a1', 'o1'), row('a2', 'o2'), row('a3', 'o3')]);
    let now = 0;
    const clock = () => now;
    queryOrder.mockImplementation((orderId: string) => {
      now += 15_000;
      return Promise.resolve(paid(orderId));
    });

    const batch = await recheckEasyPayOrders(
      { since: SINCE, cursor: null, batchSize: 10 },
      { resolveProvider, sleep, clock, timeBudgetMs: 20_000 },
    );

    expect(batch).toMatchObject({ processed: 2, done: false, nextCursor: 'a2' });
  });

  it('默认按订单的支付实例或环境变量配置查单', async () => {
    mockAuditFindMany.mockResolvedValue([
      row('a1', 'o1'),
      row('a2', 'o2', { providerInstanceId: null }, 'easy-pay:notify'),
      row('a3', 'o3', { providerInstanceId: null }, 'easy-pay:inst-2:notify'),
      row('a4', 'o4', { providerInstanceId: 'deleted' }),
    ]);
    mockGetInstanceConfig.mockImplementation((id: string) =>
      Promise.resolve(id === 'deleted' ? null : { pid: '1001', pkey: 'k' }),
    );
    mockEasyPayQueryOrder.mockImplementation((_name: string, orderId: string) => Promise.resolve(paid(orderId)));

    const batch = await recheckEasyPayOrders({ since: SINCE, cursor: null, batchSize: 10 }, { sleep });

    expect(mockGetInstanceConfig.mock.calls).toEqual([['inst-1'], ['inst-2'], ['deleted']]);
    expect(mockEasyPayConstructed.mock.calls).toEqual([
      ['inst-1', { pid: '1001', pkey: 'k' }],
      [undefined, undefined],
      ['inst-2', { pid: '1001', pkey: 'k' }],
    ]);
    expect(mockEasyPayQueryOrder.mock.calls).toEqual([
      ['easy-pay:inst-1', 'o1'],
      ['easy-pay', 'o2'],
      ['easy-pay:inst-2', 'o3'],
    ]);
    expect(batch.results.map((item) => item.outcome)).toEqual(['ok', 'ok', 'ok', 'skipped']);
  });
});
