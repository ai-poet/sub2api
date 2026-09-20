import { prisma } from '@/lib/db';
import { ORDER_STATUS } from '@/lib/constants';
import { reconcilePendingOrderPayment } from './service';

/**
 * 待支付订单周期对账。
 *
 * 订单完成有三条路：支付平台的异步通知、用户停留在页面时的轮询查单、到期扫描前的查单。
 * 通知一旦失效（回调地址不可达、平台不重试、瞬时失败被记成功……），付款时已经离开页面的
 * 用户就会卡在"未支付"，直到订单到期或管理员点取消。这里补上第四条：每 30 秒主动向平台
 * 查一遍创建满 1 分钟、尚未到期的 PENDING 订单，让通知变成"快路径"而不是唯一路径。
 *
 * 1 分钟的门槛是避开二维码页每 2 秒的轮询窗口，减少对平台的重复查询；到期后的订单交给
 * timeout.ts 的到期扫描（它也会先查单）。
 */
const INTERVAL_MS = 30_000;
const MIN_AGE_MS = 60_000;
const BATCH_SIZE = 50;
let timer: ReturnType<typeof setInterval> | null = null;
let running = false;

export async function reconcilePendingOrders(now: Date = new Date()): Promise<number> {
  const orders = await prisma.order.findMany({
    where: {
      status: ORDER_STATUS.PENDING,
      createdAt: { lt: new Date(now.getTime() - MIN_AGE_MS) },
      expiresAt: { gt: now },
    },
    select: { id: true },
    take: BATCH_SIZE,
    orderBy: { createdAt: 'asc' },
  });

  let confirmed = 0;
  for (const order of orders) {
    // 内部已吞掉平台 / 数据库错误并返回 false，一单失败不影响其它
    if (await reconcilePendingOrderPayment(order.id, 'sweep')) {
      confirmed++;
    }
  }

  if (confirmed > 0) {
    console.log(`Reconciled ${confirmed} pending order(s) already paid on the payment platform`);
  }
  return confirmed;
}

async function runOnce(): Promise<void> {
  // 平台慢时一轮可能超过 30 秒，不允许两轮重叠
  if (running) return;
  running = true;
  try {
    await reconcilePendingOrders();
  } catch (err) {
    console.error('Pending order reconciliation failed:', err);
  } finally {
    running = false;
  }
}

export function startReconcileScheduler(): void {
  if (timer) return;

  void runOnce();
  timer = setInterval(() => {
    void runOnce();
  }, INTERVAL_MS);

  console.log('Pending order reconciliation scheduler started');
}

export function stopReconcileScheduler(): void {
  if (timer) {
    clearInterval(timer);
    timer = null;
    console.log('Pending order reconciliation scheduler stopped');
  }
}
