import { prisma } from '@/lib/db';
import type { PaymentNotification, PaymentProvider, QueryOrderResponse } from '@/lib/payment/types';
import { PAYMENT_UPSTREAM_MISMATCH } from '@/lib/payment-risk/shared';

export { PAYMENT_UPSTREAM_MISMATCH };

/**
 * 入账后复核时两次查单前各等多久：第一次给平台一点时间落库；平台仍显示未付款或查无此单时，
 * 再等一分钟复查一次。平台查单略晚于通知是常见现象，二次复查避免误报。
 */
const DEFAULT_DELAYS_MS: readonly number[] = [5_000, 60_000];

/** 金额允许的误差，与入账时 confirmPayment 的容差一致。 */
const AMOUNT_TOLERANCE = 0.01;

/** client.ts 查单时，平台回了 code != 1 的错误就以这个前缀抛出，后面跟平台的 msg。 */
const PLATFORM_ERROR_PREFIX = 'EasyPay query order failed:';

/**
 * 平台明确答复"没有这笔订单"的文案。只认和订单相关的说法："商户不存在"、"KEY 校验失败"
 * 这类配置错误会让每一笔都失败，不能当成查无此单，否则会整批误报。
 */
const ORDER_NOT_FOUND_PATTERN = /订单[^，,。;；]{0,8}不存在|查无此单|无此订单|order[^.;,]{0,20}(?:not\s*exist|not\s*found)/i;

export type UpstreamMismatchReason =
  | 'upstream_not_paid'
  | 'upstream_order_not_found'
  | 'upstream_amount_mismatch'
  | 'upstream_trade_no_mismatch';

/** 入账时记下的金额与交易号，复核就拿它们和平台比。 */
export interface CreditedPayment {
  amount: number;
  tradeNo: string;
}

export type UpstreamCheck =
  | { kind: 'ok'; upstream: QueryOrderResponse }
  | { kind: 'mismatch'; reason: UpstreamMismatchReason; upstream: QueryOrderResponse | null; message?: string }
  | { kind: 'error'; error: unknown };

export interface UpstreamVerifyOptions {
  delaysMs?: readonly number[];
  sleep?: (ms: number) => Promise<void>;
}

function defaultSleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function isUpstreamOrderNotFound(error: unknown): boolean {
  if (!(error instanceof Error) || !error.message.startsWith(PLATFORM_ERROR_PREFIX)) {
    return false;
  }
  return ORDER_NOT_FOUND_PATTERN.test(error.message.slice(PLATFORM_ERROR_PREFIX.length));
}

export function compareWithUpstream(credited: CreditedPayment, upstream: QueryOrderResponse): UpstreamMismatchReason | null {
  if (upstream.status !== 'paid') {
    return 'upstream_not_paid';
  }
  if (!Number.isFinite(upstream.amount) || Math.abs(upstream.amount - credited.amount) > AMOUNT_TOLERANCE) {
    return 'upstream_amount_mismatch';
  }
  if (upstream.tradeNo && credited.tradeNo && upstream.tradeNo !== credited.tradeNo) {
    return 'upstream_trade_no_mismatch';
  }
  return null;
}

/** 向平台查一次单并和入账记录比对。网络错误、超时、商户配置错误归为 error，不下结论。 */
export async function checkOrderUpstream(
  provider: Pick<PaymentProvider, 'queryOrder'>,
  orderId: string,
  credited: CreditedPayment,
): Promise<UpstreamCheck> {
  let upstream: QueryOrderResponse;
  try {
    upstream = await provider.queryOrder(orderId);
  } catch (error) {
    if (isUpstreamOrderNotFound(error)) {
      const message = (error as Error).message.slice(PLATFORM_ERROR_PREFIX.length).trim();
      return { kind: 'mismatch', reason: 'upstream_order_not_found', upstream: null, message };
    }
    return { kind: 'error', error };
  }
  const reason = compareWithUpstream(credited, upstream);
  return reason ? { kind: 'mismatch', reason, upstream } : { kind: 'ok', upstream };
}

/** 平台未付款、查无此单或查单出错都可能只是暂时的，值得过一会儿再查一次。 */
function worthRetrying(check: UpstreamCheck): boolean {
  if (check.kind === 'error') return true;
  return check.kind === 'mismatch' && (check.reason === 'upstream_not_paid' || check.reason === 'upstream_order_not_found');
}

/**
 * 写一条 PAYMENT_UPSTREAM_MISMATCH 审计，在"支付风控"面板里标为高危。
 * dedupe 为 true 时，同一订单同一原因已有记录就不再写（批量复核可能被反复执行）。返回是否写入。
 */
export async function recordUpstreamMismatch(input: {
  orderId: string;
  operator: string;
  check: Extract<UpstreamCheck, { kind: 'mismatch' }>;
  credited: CreditedPayment;
  source: 'notify' | 'recheck';
  dedupe?: boolean;
}): Promise<boolean> {
  const { orderId, operator, check, credited, source } = input;
  if (input.dedupe) {
    const existing = await prisma.auditLog.findFirst({
      where: { orderId, action: PAYMENT_UPSTREAM_MISMATCH, detail: { contains: `"reason":"${check.reason}"` } },
      select: { id: true },
    });
    if (existing) {
      return false;
    }
  }

  const detail = {
    reason: check.reason,
    source,
    upstreamStatus: check.upstream?.status ?? '',
    upstreamAmount: check.upstream?.amount ?? null,
    upstreamTradeNo: check.upstream?.tradeNo || '',
    ...(check.message && { upstreamMessage: check.message.slice(0, 200) }),
    notifyAmount: credited.amount,
    notifyTradeNo: credited.tradeNo,
  };
  await prisma.auditLog.create({
    data: { orderId, action: PAYMENT_UPSTREAM_MISMATCH, detail: JSON.stringify(detail), operator },
  });
  return true;
}

/**
 * 易支付通知入账成功后，在后台向支付平台查单复核。只记录、不拦截，不改订单状态，内部吞掉所有异常。
 *
 * 平台查单的结果没法伪造，所以这是唯一能兜住未知攻击路线和密钥泄露的措施：复核不一致就写
 * PAYMENT_UPSTREAM_MISMATCH 审计并打错误日志。查单本身失败不下结论，只打警告。
 */
export async function verifyEasyPayCreditUpstream(
  notification: PaymentNotification,
  provider: Pick<PaymentProvider, 'name' | 'queryOrder'>,
  options: UpstreamVerifyOptions = {},
): Promise<void> {
  const delays = options.delaysMs ?? DEFAULT_DELAYS_MS;
  const sleep = options.sleep ?? defaultSleep;
  const credited: CreditedPayment = { amount: notification.amount, tradeNo: notification.tradeNo || '' };

  try {
    let check: UpstreamCheck | null = null;
    for (const delay of delays) {
      await sleep(delay);
      check = await checkOrderUpstream(provider, notification.orderId, credited);
      if (!worthRetrying(check)) {
        break;
      }
    }

    if (!check || check.kind === 'ok') {
      return;
    }
    if (check.kind === 'error') {
      console.warn(
        `${provider.name}:verify upstream query failed for order ${notification.orderId}, no conclusion drawn:`,
        check.error,
      );
      return;
    }

    console.error(`${provider.name}:verify upstream mismatch for order ${notification.orderId}: ${check.reason}`, {
      upstream: check.upstream,
      message: check.message,
      credited,
    });
    await recordUpstreamMismatch({
      orderId: notification.orderId,
      operator: `${provider.name}:verify`,
      check,
      credited,
      source: 'notify',
    });
  } catch (error) {
    console.warn(`${provider.name}:verify upstream check failed for order ${notification.orderId}:`, error);
  }
}
