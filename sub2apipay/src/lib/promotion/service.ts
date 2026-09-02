/**
 * 充值活动的数据访问与名额校验（仅服务端）。
 *
 * 名额语义：引用该活动且状态不为 CANCELLED/EXPIRED 的订单都占用名额（下单即占用，
 * 取消/超时/网关失败删单自动释放）。管理端“已使用”统计与这里的限流条件保持一致。
 */
import { Prisma } from '@prisma/client';
import { prisma, getConfiguredDatabaseSchema } from '@/lib/db';
import { ORDER_STATUS } from '@/lib/constants';
import {
  computeBonus,
  isPromotionInWindow,
  pickBestPromotion,
  type BonusType,
  type PromotionRule,
  type PublicPromotion,
} from './calc';

export const PROMOTION_USAGE_EXCLUDED_STATUSES = [ORDER_STATUS.CANCELLED, ORDER_STATUS.EXPIRED];

/** 事务客户端或全局客户端都可以 */
type Db = Prisma.TransactionClient | typeof prisma;

/** 与 Prisma RechargePromotion 模型结构一致（用结构类型便于测试构造） */
export interface PromotionRow {
  id: string;
  name: string;
  description: string | null;
  minAmount: Prisma.Decimal | number;
  bonusType: string;
  bonusValue: Prisma.Decimal | number;
  maxBonus: Prisma.Decimal | number | null;
  startsAt: Date | null;
  endsAt: Date | null;
  perUserLimit: number;
  totalLimit: number;
  enabled: boolean;
  sortOrder: number;
}

export interface PromotionUsage {
  total: number;
  byUser: number;
}

export interface PromotionUsageStats {
  usedCount: number;
  bonusTotal: number;
}

export interface ResolvedPromotion {
  promotionId: string;
  promotionName: string;
  bonusAmount: number;
}

export function toPromotionRule(row: PromotionRow): PromotionRule {
  return {
    id: row.id,
    name: row.name,
    description: row.description ?? null,
    minAmount: Number(row.minAmount),
    bonusType: (row.bonusType === 'percent' ? 'percent' : 'fixed') as BonusType,
    bonusValue: Number(row.bonusValue),
    maxBonus: row.maxBonus != null ? Number(row.maxBonus) : null,
    sortOrder: row.sortOrder,
  };
}

function toIso(value: Date | null): string | null {
  return value ? new Date(value).toISOString() : null;
}

/** 已启用且在有效期内的活动，按 sortOrder、门槛升序 */
export async function listActivePromotions(db: Db = prisma, now: Date = new Date()): Promise<PromotionRow[]> {
  const rows = await db.rechargePromotion.findMany({
    where: { enabled: true },
    orderBy: [{ sortOrder: 'asc' }, { minAmount: 'asc' }],
  });
  return rows.filter((row) => isPromotionInWindow(row, now));
}

/** 各活动的占用名额（总量 + 指定用户量） */
export async function getPromotionUsage(
  db: Db,
  promotionIds: string[],
  userId?: number,
): Promise<Map<string, PromotionUsage>> {
  const usage = new Map<string, PromotionUsage>();
  if (promotionIds.length === 0) return usage;

  const where: Prisma.OrderWhereInput = {
    promotionId: { in: promotionIds },
    status: { notIn: PROMOTION_USAGE_EXCLUDED_STATUSES },
  };

  const totals = await db.order.groupBy({
    by: ['promotionId'],
    where,
    _count: { _all: true },
  });
  for (const group of totals) {
    if (!group.promotionId) continue;
    usage.set(group.promotionId, { total: group._count._all, byUser: 0 });
  }

  if (userId != null) {
    const mine = await db.order.groupBy({
      by: ['promotionId'],
      where: { ...where, userId },
      _count: { _all: true },
    });
    for (const group of mine) {
      if (!group.promotionId) continue;
      const entry = usage.get(group.promotionId) ?? { total: 0, byUser: 0 };
      entry.byUser = group._count._all;
      usage.set(group.promotionId, entry);
    }
  }

  return usage;
}

export function isWithinLimits(row: PromotionRow, usage: Map<string, PromotionUsage>): boolean {
  const entry = usage.get(row.id) ?? { total: 0, byUser: 0 };
  if (row.totalLimit > 0 && entry.total >= row.totalLimit) return false;
  if (row.perUserLimit > 0 && entry.byUser >= row.perUserLimit) return false;
  return true;
}

/** 充值页展示用：进行中的活动 + 该用户是否还能参与 */
export async function listAvailablePromotionsForUser(
  userId: number,
  now: Date = new Date(),
): Promise<PublicPromotion[]> {
  const active = await listActivePromotions(prisma, now);
  if (active.length === 0) return [];
  const usage = await getPromotionUsage(
    prisma,
    active.map((row) => row.id),
    userId,
  );
  return active.map((row) => ({
    ...toPromotionRule(row),
    startsAt: toIso(row.startsAt),
    endsAt: toIso(row.endsAt),
    available: isWithinLimits(row, usage),
  }));
}

/**
 * 对设置了名额限制的活动行加锁（SELECT ... FOR UPDATE），让并发下单串行地计数。
 * createOrder 的事务是 READ COMMITTED，不加锁的话两笔并发订单都能抢到最后一个名额。
 */
async function lockPromotionRows(tx: Prisma.TransactionClient, ids: string[]): Promise<void> {
  if (ids.length === 0) return;
  const schemaName = getConfiguredDatabaseSchema().replace(/"/g, '""');
  const table = Prisma.raw(`"${schemaName}"."recharge_promotions"`);
  await tx.$queryRaw`SELECT "id" FROM ${table} WHERE "id" IN (${Prisma.join(ids)}) ORDER BY "id" FOR UPDATE`;
}

/**
 * 下单事务内决定订单参与的活动与赠送额。
 * 只有余额充值订单会调用；返回 null 表示没有可用活动。
 */
export async function resolvePromotionForOrder(
  tx: Prisma.TransactionClient,
  userId: number,
  amount: number,
  now: Date = new Date(),
): Promise<ResolvedPromotion | null> {
  const active = await listActivePromotions(tx, now);
  const candidates = active.filter((row) => computeBonus(amount, toPromotionRule(row)) > 0);
  if (candidates.length === 0) return null;

  const limited = candidates.filter((row) => row.perUserLimit > 0 || row.totalLimit > 0);
  let usage = new Map<string, PromotionUsage>();
  if (limited.length > 0) {
    const limitedIds = limited.map((row) => row.id).sort();
    await lockPromotionRows(tx, limitedIds);
    usage = await getPromotionUsage(tx, limitedIds, userId);
  }

  const eligible = candidates.filter((row) => isWithinLimits(row, usage));
  const best = pickBestPromotion(
    amount,
    eligible.map((row) => toPromotionRule(row)),
  );
  if (!best) return null;

  return {
    promotionId: best.rule.id,
    promotionName: best.rule.name,
    bonusAmount: best.bonus,
  };
}

/** 管理端列表：各活动已使用次数与赠送合计（状态过滤与限流器一致） */
export async function getPromotionUsageStats(promotionIds: string[]): Promise<Map<string, PromotionUsageStats>> {
  const stats = new Map<string, PromotionUsageStats>();
  if (promotionIds.length === 0) return stats;

  const groups = await prisma.order.groupBy({
    by: ['promotionId'],
    where: {
      promotionId: { in: promotionIds },
      status: { notIn: PROMOTION_USAGE_EXCLUDED_STATUSES },
    },
    _count: { _all: true },
    _sum: { bonusAmount: true },
  });

  for (const group of groups) {
    if (!group.promotionId) continue;
    stats.set(group.promotionId, {
      usedCount: group._count._all,
      bonusTotal: Number(group._sum.bonusAmount ?? 0),
    });
  }
  return stats;
}
