import { prisma } from '@/lib/db';
import type { PaymentNotification } from '@/lib/payment/types';
import { EASY_PAY_NOTIFY_FIELDS, EASY_PAY_ROUTING_PARAMS } from './notify-params';

export const PAYMENT_NOTIFY_ANOMALY = 'PAYMENT_NOTIFY_ANOMALY';

/**
 * 易支付异步通知的旁路观测：只记录、不拦截，内部吞掉所有异常，绝不改变通知的处理结果。
 *
 * 必须在验签通过后、handlePaymentNotify 之前调用：入账（confirmPayment）会用通知里的 trade_no
 * 覆盖 orders.payment_trade_no，之后就读不到下单时 mapi.php 返回的平台交易号了。
 *
 * - 成功通知没带 trade_no，或与下单时平台返回的交易号不同：写一条 PAYMENT_NOTIFY_ANOMALY 审计。
 *   真实通知总带平台交易号，且与 mapi.php 返回的一致，不一致是伪造回调的强信号。
 * - 出现标准字段以外的参数名：只打告警日志（克隆平台常见），用来判断以后能否收紧成严格白名单。
 */
export async function observeEasyPayNotify(notification: PaymentNotification, providerName: string): Promise<void> {
  try {
    const raw = notification.rawData;
    const keys = raw && typeof raw === 'object' ? Object.keys(raw) : [];
    const nonStandard = keys.filter((key) => !EASY_PAY_NOTIFY_FIELDS.has(key) && !EASY_PAY_ROUTING_PARAMS.has(key));
    if (nonStandard.length > 0) {
      console.warn(
        `${providerName}:notify carries non-standard params for order ${JSON.stringify(notification.orderId)}: ${nonStandard.join(',')}`,
      );
    }

    if (notification.status !== 'success' || !notification.orderId) {
      return;
    }

    const order = await prisma.order.findUnique({
      where: { id: notification.orderId },
      select: { id: true, status: true, paymentTradeNo: true },
    });
    if (!order) {
      return;
    }

    const notifyTradeNo = notification.tradeNo || '';
    const storedTradeNo = order.paymentTradeNo || '';
    let reason: 'missing_trade_no' | 'trade_no_mismatch' | null = null;
    if (!notifyTradeNo) {
      reason = 'missing_trade_no';
    } else if (storedTradeNo && storedTradeNo !== notifyTradeNo) {
      reason = 'trade_no_mismatch';
    }
    if (!reason) {
      return;
    }

    console.error(`${providerName}:notify anomaly for order ${order.id}: ${reason}`, {
      notifyTradeNo,
      storedTradeNo,
    });
    await prisma.auditLog.create({
      data: {
        orderId: order.id,
        action: PAYMENT_NOTIFY_ANOMALY,
        detail: JSON.stringify({
          reason,
          notifyTradeNo,
          storedTradeNo,
          orderStatus: order.status,
          amount: notification.amount,
        }),
        operator: `${providerName}:notify`,
      },
    });
  } catch (error) {
    console.warn(`${providerName}:notify anomaly check failed for order ${JSON.stringify(notification.orderId)}:`, error);
  }
}
