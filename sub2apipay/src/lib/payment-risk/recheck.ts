import type { Prisma } from '@prisma/client';
import { z } from 'zod';
import { prisma } from '@/lib/db';
import { EasyPayProvider } from '@/lib/easy-pay/provider';
import { checkOrderUpstream, recordUpstreamMismatch, type CreditedPayment } from '@/lib/easy-pay/upstream-verify';
import { getInstanceConfig } from '@/lib/payment/load-balancer';
import type { PaymentProvider } from '@/lib/payment/types';
import { computeRiskWindowStart } from './overview';
import type { PaymentRecheckBatch, PaymentRecheckItem } from './shared';

/**
 * 批量复核历史订单：把一段时间里由易支付回调入账的订单逐笔拿到平台上查单，看平台是否真的收到了钱。
 *
 * - 只读订单，不改订单状态；对不上的写一条 PAYMENT_UPSTREAM_MISMATCH（operator 以 `:recheck` 结尾），
 *   同一订单同一原因只写一次，可以放心反复执行。
 * - 由管理员在面板上手动触发，按批推进：每次请求最多处理一小批，笔与笔之间留间隔，单次请求有时间上限，
 *   前端拿 nextCursor 续跑。服务端不保存任何任务状态，关掉页面就停。
 */

export const RECHECK_DEFAULT_BATCH_SIZE = 10;
export const RECHECK_MAX_BATCH_SIZE = 20;
/** 一次请求最多花多久；超过就把剩下的留给下一批。单笔查单自带 10 秒超时。 */
const DEFAULT_TIME_BUDGET_MS = 20_000;
/** 两次查单之间的间隔，避免给支付平台造成压力。 */
const DEFAULT_QUERY_GAP_MS = 300;

/** 这些来源的入账本来就是向平台查单确认的，平台已经核对过，不必再查。 */
const QUERY_CONFIRMED_SOURCES = ['poll', 'sweep', 'cancel', 'expire'] as const;
const PAYMENT_SOURCES = new Set<string>(['notify', ...QUERY_CONFIRMED_SOURCES]);
const EASY_PAY_OPERATOR_PREFIX = 'easy-pay';

type QueryProvider = Pick<PaymentProvider, 'name' | 'queryOrder'>;

export interface RecheckRequest {
  since: Date;
  cursor: string | null;
  batchSize: number;
}

export interface RecheckDeps {
  resolveProvider?: (instanceId: string | null) => Promise<QueryProvider | null>;
  sleep?: (ms: number) => Promise<void>;
  clock?: () => number;
  timeBudgetMs?: number;
  queryGapMs?: number;
}

const ORDER_SELECT = {
  id: true,
  userId: true,
  userName: true,
  userEmail: true,
  amount: true,
  payAmount: true,
  paymentType: true,
  status: true,
  paidAt: true,
  paymentTradeNo: true,
  providerInstanceId: true,
} satisfies Prisma.OrderSelect;

type CandidateRow = {
  id: string;
  operator: string | null;
  order: Prisma.OrderGetPayload<{ select: typeof ORDER_SELECT }>;
};

const DAY_MS = 24 * 60 * 60 * 1000;
/** 复核窗口最长一年，多留一天容纳业务时区的整日对齐。 */
const MAX_WINDOW_MS = 366 * DAY_MS;

const recheckBodySchema = z.object({
  days: z.number().int().min(1).max(365).optional(),
  since: z.string().max(64).optional(),
  cursor: z
    .string()
    .regex(/^[A-Za-z0-9_-]{1,64}$/)
    .nullable()
    .optional(),
  batch_size: z.number().int().min(1).max(RECHECK_MAX_BATCH_SIZE).optional(),
});

/**
 * 解析批量复核请求。第一批传 days（默认 30），之后各批原样带回上一批返回的 since 与 nextCursor。
 * 参数不合法返回 null。
 */
export function parseRecheckRequest(body: unknown, now: Date = new Date()): RecheckRequest | null {
  const parsed = recheckBodySchema.safeParse(body ?? {});
  if (!parsed.success) return null;
  const { days, since: sinceRaw, cursor, batch_size: batchSize } = parsed.data;

  let since: Date;
  if (sinceRaw !== undefined) {
    since = new Date(sinceRaw);
    const age = now.getTime() - since.getTime();
    if (Number.isNaN(since.getTime()) || age < 0 || age > MAX_WINDOW_MS) return null;
  } else {
    since = computeRiskWindowStart(days ?? 30, now);
  }

  return { since, cursor: cursor ?? null, batchSize: batchSize ?? RECHECK_DEFAULT_BATCH_SIZE };
}

/** 待复核的订单：窗口内由易支付入账、且不是查单确认的那些（回调入账，以及早期没记来源的）。 */
export function recheckCandidateWhere(since: Date): Prisma.AuditLogWhereInput {
  return {
    action: 'ORDER_PAID',
    createdAt: { gte: since },
    operator: { startsWith: EASY_PAY_OPERATOR_PREFIX },
    NOT: QUERY_CONFIRMED_SOURCES.map((source) => ({ operator: { endsWith: `:${source}` } })),
  };
}

/** 从 ORDER_PAID 的 operator（easy-pay[:<实例>][:<来源>]）里取出实例 id。 */
export function instanceIdFromOperator(operator: string | null): string | null {
  if (!operator) return null;
  const [prefix, ...rest] = operator.split(':');
  if (prefix !== EASY_PAY_OPERATOR_PREFIX) return null;
  if (rest.length > 0 && PAYMENT_SOURCES.has(rest[rest.length - 1])) rest.pop();
  return rest.length > 0 ? rest.join(':') : null;
}

async function defaultResolveProvider(instanceId: string | null): Promise<QueryProvider | null> {
  if (!instanceId) {
    return new EasyPayProvider();
  }
  // 停用的实例也能取到配置，历史订单照样能查；实例被删了才查不了
  const config = await getInstanceConfig(instanceId);
  return config ? new EasyPayProvider(instanceId, config) : null;
}

function defaultSleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** 查单失败的原因给管理员看；去掉可能出现在 URL 里的商户密钥。 */
function describeError(error: unknown): string {
  const message = error instanceof Error ? error.message : String(error);
  return message.replace(/key=[^&\s]+/gi, 'key=***').slice(0, 200);
}

async function recheckOne(
  row: CandidateRow,
  providerFor: (instanceId: string | null) => Promise<QueryProvider | null>,
): Promise<PaymentRecheckItem> {
  const { order } = row;
  const credited: CreditedPayment = {
    amount: Number(order.payAmount ?? order.amount),
    tradeNo: order.paymentTradeNo || '',
  };
  const base = {
    orderId: order.id,
    userId: order.userId,
    userName: order.userName,
    userEmail: order.userEmail,
    amount: credited.amount,
    paymentType: order.paymentType,
    orderStatus: order.status,
    paidAt: order.paidAt ? order.paidAt.toISOString() : null,
  };

  try {
    const provider = await providerFor(order.providerInstanceId ?? instanceIdFromOperator(row.operator));
    if (!provider) {
      return { ...base, outcome: 'skipped', reason: 'provider_unavailable' };
    }

    const check = await checkOrderUpstream(provider, order.id, credited);
    if (check.kind === 'ok') {
      return { ...base, outcome: 'ok' };
    }
    if (check.kind === 'error') {
      return { ...base, outcome: 'failed', message: describeError(check.error) };
    }

    const recorded = await recordUpstreamMismatch({
      orderId: order.id,
      operator: `${provider.name}:recheck`,
      check,
      credited,
      source: 'recheck',
      dedupe: true,
    });
    return {
      ...base,
      outcome: 'mismatch',
      reason: check.reason,
      ...(check.message && { message: check.message.slice(0, 200) }),
      recorded,
    };
  } catch (error) {
    return { ...base, outcome: 'failed', message: describeError(error) };
  }
}

export async function recheckEasyPayOrders(request: RecheckRequest, deps: RecheckDeps = {}): Promise<PaymentRecheckBatch> {
  const resolveProvider = deps.resolveProvider ?? defaultResolveProvider;
  const sleep = deps.sleep ?? defaultSleep;
  const clock = deps.clock ?? Date.now;
  const timeBudgetMs = deps.timeBudgetMs ?? DEFAULT_TIME_BUDGET_MS;
  const queryGapMs = deps.queryGapMs ?? DEFAULT_QUERY_GAP_MS;

  const where = recheckCandidateWhere(request.since);
  const [total, rows] = await Promise.all([
    prisma.auditLog.count({ where }),
    prisma.auditLog.findMany({
      where,
      orderBy: [{ createdAt: 'asc' }, { id: 'asc' }],
      ...(request.cursor ? { cursor: { id: request.cursor }, skip: 1 } : {}),
      take: request.batchSize,
      select: { id: true, operator: true, order: { select: ORDER_SELECT } },
    }),
  ]);

  // 同一批里同一个实例只建一次 provider（实例配置要查库解密）
  const providers = new Map<string, Promise<QueryProvider | null>>();
  const providerFor = (instanceId: string | null) => {
    const key = instanceId ?? '';
    let provider = providers.get(key);
    if (!provider) {
      provider = resolveProvider(instanceId);
      providers.set(key, provider);
    }
    return provider;
  };

  const results: PaymentRecheckItem[] = [];
  const startedAt = clock();
  let lastId: string | null = null;
  for (const row of rows) {
    if (results.length > 0) {
      if (clock() - startedAt >= timeBudgetMs) break;
      await sleep(queryGapMs);
    }
    results.push(await recheckOne(row, providerFor));
    lastId = row.id;
  }

  const done = results.length === rows.length && rows.length < request.batchSize;
  return {
    since: request.since.toISOString(),
    total,
    processed: results.length,
    nextCursor: done ? null : (lastId ?? request.cursor),
    done,
    results,
  };
}
