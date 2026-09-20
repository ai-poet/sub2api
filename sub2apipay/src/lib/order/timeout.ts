import { prisma } from '@/lib/db';
import { ORDER_STATUS } from '@/lib/constants';
import { cancelOrderCore } from './service';

const INTERVAL_MS = 30_000; // 30 seconds
const BATCH_SIZE = 50;
/**
 * 平台查单失败时最多再等多久：期间订单保持 PENDING、下一轮重试，避免把已付款的订单"本地过期"掉；
 * 超过这个时长仍查不到就本地过期，免得平台长期不可用把用户的待支付名额占死。
 */
const PLATFORM_ERROR_GRACE_MS = 60 * 60 * 1000;
let timer: ReturnType<typeof setInterval> | null = null;

/**
 * 平台显示已付款但本地入账失败（通常是金额不符）：把订单移出待支付池并标为 FAILED 留给人工处理，
 * 否则每 30 秒都会重新查单、重复写审计。CAS 只改仍是 PENDING 的订单，绝不覆盖另一条路径正在
 * 进行的 PAID / RECHARGING。
 */
async function markPaidUnconfirmed(orderId: string): Promise<void> {
  const reason = 'PAID_ON_PLATFORM_BUT_UNCONFIRMED: platform reports the trade as paid but local crediting failed (see PAYMENT_AMOUNT_MISMATCH / PAYMENT_NOTIFY_REJECTED audit)';
  const result = await prisma.order.updateMany({
    where: { id: orderId, status: ORDER_STATUS.PENDING },
    data: { status: ORDER_STATUS.FAILED, failedAt: new Date(), failedReason: reason },
  });
  if (result.count > 0) {
    await prisma.auditLog.create({
      data: {
        orderId,
        action: 'PAYMENT_CONFIRM_FAILED',
        detail: reason,
        operator: 'timeout',
      },
    });
    console.error(`Order ${orderId} moved to FAILED for manual handling: ${reason}`);
  }
}

export async function expireOrders(now: Date = new Date()): Promise<number> {
  // 查询到期订单（限制批次大小防止内存爆炸）
  // cancelOrderCore 内部 WHERE status='PENDING' 的 CAS 保证多实例不会重复处理同一订单
  const orders = await prisma.order.findMany({
    where: {
      status: ORDER_STATUS.PENDING,
      expiresAt: { lt: now },
    },
    select: {
      id: true,
      paymentTradeNo: true,
      paymentType: true,
      providerInstanceId: true,
      expiresAt: true,
    },
    take: BATCH_SIZE,
    orderBy: { expiresAt: 'asc' },
  });

  if (orders.length === 0) return 0;

  let expiredCount = 0;

  for (const order of orders) {
    try {
      const withinGrace = now.getTime() - order.expiresAt.getTime() < PLATFORM_ERROR_GRACE_MS;
      const outcome = await cancelOrderCore({
        orderId: order.id,
        paymentTradeNo: order.paymentTradeNo,
        paymentType: order.paymentType,
        providerInstanceId: order.providerInstanceId,
        finalStatus: ORDER_STATUS.EXPIRED,
        operator: 'timeout',
        auditDetail: 'Order expired',
        onPlatformError: withinGrace ? 'skip' : 'cancel',
      });

      if (outcome === 'cancelled') {
        expiredCount++;
      } else if (outcome === 'paid_unconfirmed') {
        await markPaidUnconfirmed(order.id);
      }
      // already_paid：已履约；platform_unavailable：保持 PENDING，下一轮再试
    } catch (err) {
      console.error(`Error expiring order ${order.id}:`, err);
    }
  }

  if (expiredCount > 0) {
    console.log(`Expired ${expiredCount} orders`);
  }

  return expiredCount;
}

export function startTimeoutScheduler(): void {
  if (timer) return;

  // Run immediately on startup
  expireOrders().catch(console.error);

  // Then run every 30 seconds
  timer = setInterval(() => {
    expireOrders().catch(console.error);
  }, INTERVAL_MS);

  console.log('Order timeout scheduler started');
}

export function stopTimeoutScheduler(): void {
  if (timer) {
    clearInterval(timer);
    timer = null;
    console.log('Order timeout scheduler stopped');
  }
}
