import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Prisma } from '@prisma/client';

const mockPromotionFindMany = vi.fn();
const mockOrderGroupBy = vi.fn();
const mockQueryRaw = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    rechargePromotion: { findMany: (...args: unknown[]) => mockPromotionFindMany(...args) },
    order: { groupBy: (...args: unknown[]) => mockOrderGroupBy(...args) },
  },
  getConfiguredDatabaseSchema: () => 'sub2apipay',
}));

vi.mock('@/lib/config', () => ({
  getEnv: () => ({ DATABASE_URL: 'postgresql://localhost/test?schema=sub2apipay' }),
}));

import {
  getPromotionUsageStats,
  listAvailablePromotionsForUser,
  resolvePromotionForOrder,
  type PromotionRow,
} from '@/lib/promotion/service';

const NOW = new Date('2026-09-02T12:00:00Z');

function row(overrides: Partial<PromotionRow> = {}): PromotionRow {
  return {
    id: 'p1',
    name: '充100送10',
    description: null,
    minAmount: new Prisma.Decimal('100.00'),
    bonusType: 'fixed',
    bonusValue: new Prisma.Decimal('10.00'),
    maxBonus: null,
    startsAt: null,
    endsAt: null,
    perUserLimit: 0,
    totalLimit: 0,
    enabled: true,
    sortOrder: 0,
    ...overrides,
  };
}

function makeTx() {
  return {
    rechargePromotion: { findMany: (...args: unknown[]) => mockPromotionFindMany(...args) },
    order: { groupBy: (...args: unknown[]) => mockOrderGroupBy(...args) },
    $queryRaw: (...args: unknown[]) => mockQueryRaw(...args),
  } as unknown as Prisma.TransactionClient;
}

beforeEach(() => {
  vi.clearAllMocks();
  mockPromotionFindMany.mockResolvedValue([]);
  mockOrderGroupBy.mockResolvedValue([]);
  mockQueryRaw.mockResolvedValue([]);
});

describe('resolvePromotionForOrder', () => {
  it('没有活动时返回 null，且不加锁不计数', async () => {
    const result = await resolvePromotionForOrder(makeTx(), 42, 100, NOW);
    expect(result).toBeNull();
    expect(mockQueryRaw).not.toHaveBeenCalled();
    expect(mockOrderGroupBy).not.toHaveBeenCalled();
  });

  it('未达门槛返回 null', async () => {
    mockPromotionFindMany.mockResolvedValue([row()]);
    expect(await resolvePromotionForOrder(makeTx(), 42, 50, NOW)).toBeNull();
  });

  it('过滤掉不在有效期内的活动', async () => {
    mockPromotionFindMany.mockResolvedValue([
      row({ id: 'future', startsAt: new Date('2026-09-03T00:00:00Z') }),
      row({ id: 'past', endsAt: new Date('2026-09-01T00:00:00Z') }),
      row({ id: 'live', name: '进行中', bonusValue: new Prisma.Decimal('5.00') }),
    ]);
    const result = await resolvePromotionForOrder(makeTx(), 42, 100, NOW);
    expect(result).toEqual({ promotionId: 'live', promotionName: '进行中', bonusAmount: 5 });
  });

  it('无限制的活动不加锁；取赠送最高者', async () => {
    mockPromotionFindMany.mockResolvedValue([
      row({ id: 'a', bonusValue: new Prisma.Decimal('10.00') }),
      row({
        id: 'b',
        name: '充500送80',
        minAmount: new Prisma.Decimal('500.00'),
        bonusValue: new Prisma.Decimal('80.00'),
      }),
    ]);
    const result = await resolvePromotionForOrder(makeTx(), 42, 600, NOW);
    expect(result?.promotionId).toBe('b');
    expect(result?.bonusAmount).toBe(80);
    expect(mockQueryRaw).not.toHaveBeenCalled();
  });

  it('设置了名额限制的活动先加锁再计数', async () => {
    mockPromotionFindMany.mockResolvedValue([row({ id: 'limited', totalLimit: 10 })]);
    mockOrderGroupBy.mockResolvedValue([{ promotionId: 'limited', _count: { _all: 3 } }]);

    const result = await resolvePromotionForOrder(makeTx(), 42, 100, NOW);

    expect(result?.promotionId).toBe('limited');
    expect(mockQueryRaw).toHaveBeenCalledTimes(1);
    // 总量 + 用户量各一次
    expect(mockOrderGroupBy).toHaveBeenCalledTimes(2);
    expect(mockOrderGroupBy.mock.calls[0][0]).toMatchObject({
      by: ['promotionId'],
      where: { promotionId: { in: ['limited'] }, status: { notIn: ['CANCELLED', 'EXPIRED'] } },
    });
    expect(mockOrderGroupBy.mock.calls[1][0]).toMatchObject({ where: { userId: 42 } });
  });

  it('总名额用完时回退到次优活动', async () => {
    mockPromotionFindMany.mockResolvedValue([
      row({ id: 'big', name: '大额', bonusValue: new Prisma.Decimal('50.00'), totalLimit: 1 }),
      row({ id: 'small', name: '小额', bonusValue: new Prisma.Decimal('10.00') }),
    ]);
    mockOrderGroupBy.mockResolvedValue([{ promotionId: 'big', _count: { _all: 1 } }]);

    const result = await resolvePromotionForOrder(makeTx(), 42, 100, NOW);
    expect(result).toEqual({ promotionId: 'small', promotionName: '小额', bonusAmount: 10 });
  });

  it('每人次数用完时该用户不再享受', async () => {
    mockPromotionFindMany.mockResolvedValue([row({ id: 'first', perUserLimit: 1 })]);
    mockOrderGroupBy
      .mockResolvedValueOnce([{ promotionId: 'first', _count: { _all: 5 } }])
      .mockResolvedValueOnce([{ promotionId: 'first', _count: { _all: 1 } }]);

    expect(await resolvePromotionForOrder(makeTx(), 42, 100, NOW)).toBeNull();
  });
});

describe('listAvailablePromotionsForUser', () => {
  it('返回进行中的活动并标记可用性', async () => {
    mockPromotionFindMany.mockResolvedValue([
      row({ id: 'open' }),
      row({ id: 'full', totalLimit: 2, endsAt: new Date('2026-09-30T00:00:00Z') }),
      row({ id: 'ended', endsAt: new Date('2026-09-01T00:00:00Z') }),
    ]);
    mockOrderGroupBy.mockResolvedValueOnce([{ promotionId: 'full', _count: { _all: 2 } }]).mockResolvedValueOnce([]);

    const result = await listAvailablePromotionsForUser(42, NOW);
    expect(result.map((p) => p.id)).toEqual(['open', 'full']);
    expect(result[0]).toMatchObject({ available: true, minAmount: 100, bonusType: 'fixed', bonusValue: 10 });
    expect(result[1]).toMatchObject({ available: false, endsAt: '2026-09-30T00:00:00.000Z' });
  });

  it('没有活动时不查询用量', async () => {
    expect(await listAvailablePromotionsForUser(42, NOW)).toEqual([]);
    expect(mockOrderGroupBy).not.toHaveBeenCalled();
  });
});

describe('getPromotionUsageStats', () => {
  it('汇总次数与赠送合计', async () => {
    mockOrderGroupBy.mockResolvedValue([
      { promotionId: 'a', _count: { _all: 3 }, _sum: { bonusAmount: new Prisma.Decimal('30.00') } },
      { promotionId: 'b', _count: { _all: 1 }, _sum: { bonusAmount: null } },
    ]);
    const stats = await getPromotionUsageStats(['a', 'b']);
    expect(stats.get('a')).toEqual({ usedCount: 3, bonusTotal: 30 });
    expect(stats.get('b')).toEqual({ usedCount: 1, bonusTotal: 0 });
  });
});
