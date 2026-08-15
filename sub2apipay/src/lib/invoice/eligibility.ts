import { ORDER_STATUS } from '@/lib/constants';
import { isStablecoinPaymentType } from '@/lib/currency';
import { INVOICE_REREQUESTABLE_STATUSES, type InvoiceEligibility, type InvoiceStatus } from './types';

/** 开票资格判定所需的订单字段（Decimal 可能以 string 传入，故放宽类型）。 */
export interface InvoiceEligibilityOrder {
  status: string;
  paymentType: string;
  paidAt: Date | null;
  payAmount: unknown;
  refundAmount?: unknown;
}

export interface InvoiceEligibilityOptions {
  featureEnabled: boolean;
  /** 0 表示不限制 */
  maxAgeDays: number;
  existingStatus?: InvoiceStatus | null;
  now?: Date;
}

function toNumber(value: unknown): number | null {
  if (value == null) return null;
  const parsed = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

/**
 * 纯函数，无 IO：/api/orders/my 用它算 canRequestInvoice，POST handler 用它做鉴权，
 * 两边不会漂移。
 */
export function evaluateInvoiceEligibility(
  order: InvoiceEligibilityOrder,
  options: InvoiceEligibilityOptions,
): InvoiceEligibility {
  if (!options.featureEnabled) {
    return { eligible: false, reason: 'FEATURE_DISABLED' };
  }

  if (order.status !== ORDER_STATUS.COMPLETED || !order.paidAt) {
    return { eligible: false, reason: 'NOT_COMPLETED' };
  }

  // USDT/USDC 按美元结算，开不了人民币发票。
  if (isStablecoinPaymentType(order.paymentType)) {
    return { eligible: false, reason: 'STABLECOIN_NOT_SUPPORTED' };
  }

  // payAmount 是实付人民币，也是唯一合法的开票金额。老订单（早于手续费字段上线）
  // 没有这个值——绝不用汇率反推，宁可拒绝并让用户找客服。
  const payAmount = toNumber(order.payAmount);
  if (payAmount == null || payAmount <= 0) {
    return { eligible: false, reason: 'MISSING_PAY_AMOUNT' };
  }

  const refundAmount = toNumber(order.refundAmount);
  if (refundAmount != null && refundAmount > 0) {
    return { eligible: false, reason: 'ORDER_REFUNDED' };
  }

  if (options.maxAgeDays > 0) {
    const now = options.now ?? new Date();
    const ageMs = now.getTime() - order.paidAt.getTime();
    if (ageMs > options.maxAgeDays * 24 * 60 * 60 * 1000) {
      return { eligible: false, reason: 'TOO_OLD' };
    }
  }

  const existing = options.existingStatus;
  if (existing && !INVOICE_REREQUESTABLE_STATUSES.includes(existing)) {
    return { eligible: false, reason: 'ALREADY_REQUESTED' };
  }

  return { eligible: true };
}
