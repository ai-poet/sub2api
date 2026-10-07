import { Prisma } from '@prisma/client';
import { getConfiguredDatabaseSchema, prisma } from '@/lib/db';
import { BIZ_TZ_NAME, getBizDayStartUTC, toBizDateStr } from '@/lib/time/biz-day';
import {
  PAYMENT_RISK_ACTIONS,
  isPaymentRiskAction,
  type PaymentRiskAction,
  type PaymentRiskDailyPoint,
  type PaymentRiskOrderBrief,
  type PaymentRiskOverview,
} from './shared';

const DAY_MS = 24 * 60 * 60 * 1000;
const PAID_WITHOUT_TRADE_NO_LIMIT = 50;
const DUPLICATE_GROUP_LIMIT = 20;
const TOP_USER_LIMIT = 10;

export interface PaymentRiskQuery {
  days: number;
  action: PaymentRiskAction | null;
  page: number;
  pageSize: number;
}

function clampInt(raw: string | null, fallback: number, min: number, max: number): number {
  if (raw === null || raw.trim() === '') return fallback;
  const value = Number(raw);
  if (!Number.isFinite(value)) return fallback;
  return Math.min(max, Math.max(min, Math.trunc(value)));
}

export function parsePaymentRiskQuery(searchParams: URLSearchParams): PaymentRiskQuery {
  const action = searchParams.get('action');
  return {
    days: clampInt(searchParams.get('days'), 30, 1, 365),
    action: action && isPaymentRiskAction(action) ? action : null,
    page: clampInt(searchParams.get('page'), 1, 1, 10_000),
    pageSize: clampInt(searchParams.get('page_size'), 20, 1, 100),
  };
}

const ORDER_BRIEF_SELECT = {
  id: true,
  userId: true,
  userName: true,
  userEmail: true,
  amount: true,
  payAmount: true,
  status: true,
  paymentType: true,
  orderType: true,
  paymentTradeNo: true,
  createdAt: true,
  paidAt: true,
} satisfies Prisma.OrderSelect;

type OrderBriefRow = Prisma.OrderGetPayload<{ select: typeof ORDER_BRIEF_SELECT }>;

function toOrderBrief(order: OrderBriefRow): PaymentRiskOrderBrief {
  return {
    id: order.id,
    userId: order.userId,
    userName: order.userName,
    userEmail: order.userEmail,
    amount: Number(order.amount),
    payAmount: order.payAmount == null ? null : Number(order.payAmount),
    status: order.status,
    paymentType: order.paymentType,
    orderType: order.orderType,
    paymentTradeNo: order.paymentTradeNo,
    createdAt: order.createdAt.toISOString(),
    paidAt: order.paidAt ? order.paidAt.toISOString() : null,
  };
}

/** 风控时间窗口的起点：按业务时区的自然日，days=7 表示今天和之前 6 天。 */
export function computeRiskWindowStart(days: number, now: Date = new Date()): Date {
  return new Date(getBizDayStartUTC(now).getTime() - (days - 1) * DAY_MS);
}

function emptyActionCounts(): Record<PaymentRiskAction, number> {
  return Object.fromEntries(PAYMENT_RISK_ACTIONS.map((action) => [action, 0])) as Record<PaymentRiskAction, number>;
}

/**
 * 支付风控面板的数据：只读查询，不改任何订单或审计记录。
 *
 * 时间窗口按业务时区的自然日计算，见 computeRiskWindowStart。
 */
export async function getPaymentRiskOverview(query: PaymentRiskQuery, now: Date = new Date()): Promise<PaymentRiskOverview> {
  const since = computeRiskWindowStart(query.days, now);

  const schemaName = getConfiguredDatabaseSchema().replace(/"/g, '""');
  const ordersTable = Prisma.raw(`"${schemaName}"."orders"`);
  const auditTable = Prisma.raw(`"${schemaName}"."audit_logs"`);
  const businessTz = Prisma.raw(`'${BIZ_TZ_NAME}'`);
  const riskActions = Prisma.join([...PAYMENT_RISK_ACTIONS]);

  const eventWhere: Prisma.AuditLogWhereInput = {
    action: { in: [...PAYMENT_RISK_ACTIONS] },
    createdAt: { gte: since },
  };
  const listWhere: Prisma.AuditLogWhereInput = query.action
    ? { action: query.action, createdAt: { gte: since } }
    : eventWhere;
  const noTradeNoWhere: Prisma.OrderWhereInput = {
    paidAt: { gte: since },
    OR: [{ paymentTradeNo: null }, { paymentTradeNo: '' }],
  };

  const [actionCounts, dailyRows, eventRows, eventTotal, noTradeNoRows, noTradeNoTotal, duplicateRows, topUserRows] =
    await Promise.all([
      prisma.auditLog.groupBy({ by: ['action'], where: eventWhere, _count: { _all: true } }),
      prisma.$queryRaw<{ date: string; action: string; count: bigint }[]>`
        SELECT (created_at AT TIME ZONE 'UTC' AT TIME ZONE ${businessTz})::date::text AS date,
               action, COUNT(*) AS count
        FROM ${auditTable}
        WHERE action IN (${riskActions}) AND created_at >= ${since}
        GROUP BY 1, 2
        ORDER BY 1
      `,
      prisma.auditLog.findMany({
        where: listWhere,
        orderBy: { createdAt: 'desc' },
        skip: (query.page - 1) * query.pageSize,
        take: query.pageSize,
        select: {
          id: true,
          action: true,
          detail: true,
          operator: true,
          createdAt: true,
          order: { select: ORDER_BRIEF_SELECT },
        },
      }),
      prisma.auditLog.count({ where: listWhere }),
      prisma.order.findMany({
        where: noTradeNoWhere,
        orderBy: { paidAt: 'desc' },
        take: PAID_WITHOUT_TRADE_NO_LIMIT,
        select: ORDER_BRIEF_SELECT,
      }),
      prisma.order.count({ where: noTradeNoWhere }),
      prisma.$queryRaw<{ trade_no: string; order_ids: string[]; count: bigint }[]>`
        SELECT payment_trade_no AS trade_no, array_agg(id ORDER BY paid_at) AS order_ids, COUNT(*) AS count
        FROM ${ordersTable}
        WHERE paid_at >= ${since} AND COALESCE(payment_trade_no, '') <> ''
        GROUP BY payment_trade_no
        HAVING COUNT(*) > 1
        ORDER BY COUNT(*) DESC, payment_trade_no
        LIMIT ${Prisma.raw(String(DUPLICATE_GROUP_LIMIT))}
      `,
      prisma.$queryRaw<
        {
          user_id: number;
          user_name: string | null;
          user_email: string | null;
          event_count: bigint;
          order_count: bigint;
          last_at: Date;
        }[]
      >`
        SELECT o.user_id, MAX(o.user_name) AS user_name, MAX(o.user_email) AS user_email,
               COUNT(*) AS event_count, COUNT(DISTINCT o.id) AS order_count, MAX(a.created_at) AS last_at
        FROM ${auditTable} a
        JOIN ${ordersTable} o ON o.id = a.order_id
        WHERE a.action IN (${riskActions}) AND a.created_at >= ${since}
        GROUP BY o.user_id
        ORDER BY COUNT(*) DESC, MAX(a.created_at) DESC
        LIMIT ${Prisma.raw(String(TOP_USER_LIMIT))}
      `,
    ]);

  const duplicateIds = duplicateRows.flatMap((row) => row.order_ids);
  const duplicateOrders = duplicateIds.length
    ? await prisma.order.findMany({ where: { id: { in: duplicateIds } }, select: ORDER_BRIEF_SELECT })
    : [];
  const duplicateOrderById = new Map(duplicateOrders.map((order) => [order.id, toOrderBrief(order)]));

  const eventSummary = emptyActionCounts();
  for (const row of actionCounts) {
    if (isPaymentRiskAction(row.action)) eventSummary[row.action] = row._count._all;
  }

  const countsByDate = new Map<string, Record<PaymentRiskAction, number>>();
  for (const row of dailyRows) {
    if (!isPaymentRiskAction(row.action)) continue;
    const counts = countsByDate.get(row.date) ?? emptyActionCounts();
    counts[row.action] += Number(row.count);
    countsByDate.set(row.date, counts);
  }
  const daily: PaymentRiskDailyPoint[] = [];
  const seenDates = new Set<string>();
  for (let t = since.getTime(); t <= now.getTime(); t += DAY_MS) {
    const date = toBizDateStr(new Date(t));
    if (seenDates.has(date)) continue;
    seenDates.add(date);
    daily.push({ date, ...(countsByDate.get(date) ?? emptyActionCounts()) });
  }

  return {
    meta: {
      days: query.days,
      since: since.toISOString(),
      generatedAt: now.toISOString(),
      limits: {
        paidWithoutTradeNo: PAID_WITHOUT_TRADE_NO_LIMIT,
        duplicateTradeNoGroups: DUPLICATE_GROUP_LIMIT,
        topUsers: TOP_USER_LIMIT,
      },
    },
    summary: {
      events: eventSummary,
      paidWithoutTradeNo: noTradeNoTotal,
      duplicateTradeNoGroups: duplicateRows.length,
    },
    daily,
    events: {
      action: query.action,
      items: eventRows
        .filter((row) => isPaymentRiskAction(row.action))
        .map((row) => ({
          id: row.id,
          action: row.action as PaymentRiskAction,
          detail: row.detail,
          operator: row.operator,
          createdAt: row.createdAt.toISOString(),
          order: toOrderBrief(row.order),
        })),
      total: eventTotal,
      page: query.page,
      pageSize: query.pageSize,
      totalPages: Math.max(1, Math.ceil(eventTotal / query.pageSize)),
    },
    paidWithoutTradeNo: noTradeNoRows.map(toOrderBrief),
    duplicateTradeNos: duplicateRows.map((row) => ({
      tradeNo: row.trade_no,
      count: Number(row.count),
      orders: row.order_ids
        .map((id) => duplicateOrderById.get(id))
        .filter((order): order is PaymentRiskOrderBrief => order != null),
    })),
    topUsers: topUserRows.map((row) => ({
      userId: Number(row.user_id),
      userName: row.user_name,
      userEmail: row.user_email,
      eventCount: Number(row.event_count),
      orderCount: Number(row.order_count),
      lastEventAt: new Date(row.last_at).toISOString(),
    })),
  };
}
