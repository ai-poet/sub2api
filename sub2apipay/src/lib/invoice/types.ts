import type { Locale } from '@/lib/locale';
import { pickLocaleText } from '@/lib/locale';

/** 与 OrderError 同形，由 handleApiError 统一映射为 HTTP 响应。 */
export class InvoiceError extends Error {
  code: string;
  statusCode: number;
  data?: Record<string, unknown>;

  constructor(code: string, message: string, statusCode: number = 400, data?: Record<string, unknown>) {
    super(message);
    this.name = 'InvoiceError';
    this.code = code;
    this.statusCode = statusCode;
    this.data = data;
  }
}

export function invoiceMessage(locale: Locale, zh: string, en: string): string {
  return pickLocaleText(locale, zh, en);
}

/** 开票请求各字段的展示名，用于把 zod 校验失败翻译成用户能定位的提示。 */
const INVOICE_FIELD_LABELS: Record<string, { zh: string; en: string }> = {
  order_ids: { zh: '订单', en: 'orders' },
  title_name: { zh: '单位名称', en: 'company name' },
  tax_no: { zh: '税号', en: 'tax ID' },
  remark: { zh: '备注', en: 'remark' },
  contact_email: { zh: '收票邮箱', en: 'contact email' },
};

/**
 * 把校验失败的字段列表拼成可读提示。
 *
 * 一句裸的「参数错误」等于让用户逐个字段猜——收票邮箱这类选填项出错时尤其致命，
 * 用户根本想不到问题出在一个可以留空的输入框上。
 */
export function invoiceInvalidParamsMessage(locale: Locale, badFields: string[]): string {
  const labels = badFields
    .map((field) => INVOICE_FIELD_LABELS[field] ?? { zh: field, en: field })
    .map((label) => pickLocaleText(locale, label.zh, label.en));
  if (labels.length === 0) {
    return invoiceMessage(locale, '参数错误', 'Invalid parameters');
  }
  return pickLocaleText(
    locale,
    `参数错误：${labels.join('、')}格式不正确`,
    `Invalid ${labels.join(', ')}`,
  );
}

/**
 * 单次合并开票的订单数上限。
 *
 * 与订单页每页最大条数（100）对齐：合并提交只作用于当前页的勾选，所以 UI 上不可能
 * 超过它；服务端仍然强制，作为直接调 API 的兜底。放在 types 里是因为前端也要引用
 * （service 引入了 prisma，不能进客户端包）。
 */
export const INVOICE_MAX_MERGED_ORDERS = 100;

export const INVOICE_STATUS = {
  PENDING: 'PENDING',
  ISSUED: 'ISSUED',
  REJECTED: 'REJECTED',
  CANCELLED: 'CANCELLED',
} as const;

export type InvoiceStatus = (typeof INVOICE_STATUS)[keyof typeof INVOICE_STATUS];

/** 重新申请只允许从这两个终态发起（复用同一行改回 PENDING）。 */
export const INVOICE_REREQUESTABLE_STATUSES: InvoiceStatus[] = [INVOICE_STATUS.REJECTED, INVOICE_STATUS.CANCELLED];

/** 开票不可用的原因码，供前端解释按钮为什么禁用。 */
export type InvoiceIneligibleReason =
  | 'FEATURE_DISABLED'
  | 'NOT_COMPLETED'
  | 'STABLECOIN_NOT_SUPPORTED'
  | 'MISSING_PAY_AMOUNT'
  | 'ORDER_REFUNDED'
  | 'TOO_OLD'
  | 'ALREADY_REQUESTED';

export interface InvoiceEligibility {
  eligible: boolean;
  reason?: InvoiceIneligibleReason;
}

export interface SavedInvoiceTitle {
  id: string;
  titleName: string;
  taxNo: string;
  remark: string | null;
  contactEmail: string | null;
  lastUsedAt: Date;
}

export interface InvoiceRequestView {
  id: string;
  /** 主订单号（合并开票时为最早付款的那张）。 */
  orderId: string;
  /** 该发票覆盖的全部订单号，单订单开票时长度为 1。 */
  orderIds: string[];
  status: InvoiceStatus;
  titleName: string;
  taxNo: string;
  remark: string | null;
  contactEmail: string | null;
  amount: number;
  fileName: string | null;
  hasFile: boolean;
  rejectReason: string | null;
  issuedAt: Date | null;
  createdAt: Date;
}

export interface AdminInvoiceRequestView extends InvoiceRequestView {
  userId: number;
  adminNote: string | null;
  notifiedAt: Date | null;
  issuedBy: string | null;
  downloadCount: number;
  order?: {
    id: string;
    payAmount: number | null;
    paymentType: string;
    orderType: string;
    paidAt: Date | null;
    status: string;
    userEmail: string | null;
    userName: string | null;
  } | null;
}

export function invoiceIneligibleMessage(locale: Locale, reason: InvoiceIneligibleReason): string {
  switch (reason) {
    case 'FEATURE_DISABLED':
      return invoiceMessage(locale, '当前未开放在线开票', 'Online invoicing is currently unavailable');
    case 'NOT_COMPLETED':
      return invoiceMessage(locale, '仅已完成的订单可以开票', 'Only completed orders can be invoiced');
    case 'STABLECOIN_NOT_SUPPORTED':
      return invoiceMessage(
        locale,
        '该支付方式不支持开具人民币发票',
        'This payment method does not support CNY invoices',
      );
    case 'MISSING_PAY_AMOUNT':
      return invoiceMessage(
        locale,
        '该订单缺少人民币支付金额，请联系客服处理',
        'This order has no CNY payment amount on record; please contact support',
      );
    case 'ORDER_REFUNDED':
      return invoiceMessage(locale, '已退款的订单不可开票', 'Refunded orders cannot be invoiced');
    case 'TOO_OLD':
      return invoiceMessage(locale, '该订单已超过可开票期限', 'This order is past the invoicing window');
    case 'ALREADY_REQUESTED':
      return invoiceMessage(locale, '该订单已申请开票', 'An invoice has already been requested for this order');
  }
}
