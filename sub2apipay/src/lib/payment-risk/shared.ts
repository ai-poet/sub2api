/**
 * 支付风控：前后端共用的事件类型、文案与接口返回结构。
 * 只放常量与类型，不引入服务端依赖，客户端页面可以直接 import。
 */

export const PAYMENT_UPSTREAM_MISMATCH = 'PAYMENT_UPSTREAM_MISMATCH';
export const PAYMENT_NOTIFY_BLOCKED = 'PAYMENT_NOTIFY_BLOCKED';
export const PAYMENT_NOTIFY_ANOMALY = 'PAYMENT_NOTIFY_ANOMALY';
export const PAYMENT_AMOUNT_MISMATCH = 'PAYMENT_AMOUNT_MISMATCH';
export const PAYMENT_NOTIFY_REJECTED = 'PAYMENT_NOTIFY_REJECTED';
export const PAYMENT_CONFIRM_FAILED = 'PAYMENT_CONFIRM_FAILED';

/** 计入支付风控的审计动作，按严重程度从高到低排列。 */
export const PAYMENT_RISK_ACTIONS = [
  PAYMENT_UPSTREAM_MISMATCH,
  PAYMENT_NOTIFY_BLOCKED,
  PAYMENT_NOTIFY_ANOMALY,
  PAYMENT_AMOUNT_MISMATCH,
  PAYMENT_NOTIFY_REJECTED,
  PAYMENT_CONFIRM_FAILED,
] as const;

export type PaymentRiskAction = (typeof PAYMENT_RISK_ACTIONS)[number];

export type RiskSeverity = 'high' | 'medium' | 'low';

export interface LocalizedText {
  zh: string;
  en: string;
}

export interface PaymentRiskMeta {
  severity: RiskSeverity;
  /** 图表配色 */
  color: string;
  label: LocalizedText;
  description: LocalizedText;
}

export const PAYMENT_RISK_ACTION_META: Record<PaymentRiskAction, PaymentRiskMeta> = {
  PAYMENT_UPSTREAM_MISMATCH: {
    severity: 'high',
    color: '#dc2626',
    label: { zh: '平台复核不符', en: 'Platform check mismatch' },
    description: {
      zh: '已经入账，但向支付平台查单时，平台显示未付款，或金额、交易号对不上，疑似伪造入账。',
      en: 'Credited, but the payment platform reports the order unpaid, or with a different amount or trade number. Possibly a forged credit.',
    },
  },
  PAYMENT_NOTIFY_BLOCKED: {
    severity: 'high',
    color: '#f97316',
    label: { zh: '拦截的可疑回调', en: 'Blocked callback' },
    description: {
      zh: '回调在验签阶段被拒：参数异常、签名错误、商户号不符或金额无效。可能是伪造尝试，也可能是密钥配置有误。',
      en: 'A callback was rejected during verification: unexpected parameter, bad signature, merchant ID mismatch or invalid amount. Either a forgery attempt or a key misconfiguration.',
    },
  },
  PAYMENT_NOTIFY_ANOMALY: {
    severity: 'medium',
    color: '#eab308',
    label: { zh: '回调交易号异常', en: 'Trade No. anomaly' },
    description: {
      zh: '成功回调没带平台交易号，或与下单时平台返回的交易号不一致。',
      en: 'A success callback carried no platform trade number, or one different from the number returned at order creation.',
    },
  },
  PAYMENT_AMOUNT_MISMATCH: {
    severity: 'medium',
    color: '#a855f7',
    label: { zh: '金额不符', en: 'Amount mismatch' },
    description: {
      zh: '回调或查单返回的金额与订单应付金额不符，没有入账。',
      en: 'The paid amount from a callback or query differs from the amount due. Not credited.',
    },
  },
  PAYMENT_NOTIFY_REJECTED: {
    severity: 'low',
    color: '#3b82f6',
    label: { zh: '入账被拒', en: 'Credit rejected' },
    description: {
      zh: '付款确认因金额无效等原因被拒绝入账。',
      en: 'A payment confirmation was refused, for example because of an invalid amount.',
    },
  },
  PAYMENT_CONFIRM_FAILED: {
    severity: 'low',
    color: '#64748b',
    label: { zh: '已付未入账', en: 'Paid, not credited' },
    description: {
      zh: '平台显示已付款，但本地入账失败，订单已转为失败，需要人工处理。',
      en: 'The platform reports the order paid but local crediting failed. The order was marked failed for manual handling.',
    },
  },
};

/** 不来自审计、而是直接从订单数据算出来的风险项。 */
export const PAID_WITHOUT_TRADE_NO_META: Omit<PaymentRiskMeta, 'color'> = {
  severity: 'medium',
  label: { zh: '无交易号入账', en: 'Credited without trade No.' },
  description: {
    zh: '已入账但没有平台交易号。真实平台的回调都带交易号，这类订单要和商户后台逐笔核对。若所有订单都在这里，说明平台本来就不发交易号，此项不能当作伪造信号。',
    en: 'Credited without a platform trade number. Genuine platforms send one, so check these against the merchant dashboard. If every order shows up here, the platform never sends one and this is not a forgery signal.',
  },
};

export const DUPLICATE_TRADE_NO_META: Omit<PaymentRiskMeta, 'color'> = {
  severity: 'medium',
  label: { zh: '交易号重复入账', en: 'Duplicate trade No.' },
  description: {
    zh: '同一个平台交易号给多笔订单入了账。不同支付平台偶尔会生成相同的交易号，需要人工核对。',
    en: 'One platform trade number credited several orders. Different platforms occasionally generate the same number, so review manually.',
  },
};

/** 审计 detail 里 reason 字段的可读文案。 */
export const PAYMENT_RISK_REASON_LABELS: Record<string, LocalizedText> = {
  unexpected_param: { zh: '含异常参数', en: 'Unexpected parameter' },
  bad_signature: { zh: '签名错误', en: 'Bad signature' },
  pid_mismatch: { zh: '商户号不符', en: 'Merchant ID mismatch' },
  invalid_amount: { zh: '金额无效', en: 'Invalid amount' },
  non_positive_amount: { zh: '金额非正数', en: 'Non-positive amount' },
  missing_trade_no: { zh: '缺少交易号', en: 'Missing trade No.' },
  trade_no_mismatch: { zh: '交易号不一致', en: 'Trade No. mismatch' },
  upstream_not_paid: { zh: '平台显示未付款', en: 'Platform reports unpaid' },
  upstream_order_not_found: { zh: '平台查无此单', en: 'Order not found on platform' },
  upstream_amount_mismatch: { zh: '平台金额不符', en: 'Platform amount differs' },
  upstream_trade_no_mismatch: { zh: '平台交易号不符', en: 'Platform trade No. differs' },
  provider_unavailable: { zh: '支付实例已不存在，无法查单', en: 'Payment instance no longer exists' },
};

/** 批量复核历史订单时单笔订单的结果。 */
export type PaymentRecheckOutcome = 'ok' | 'mismatch' | 'failed' | 'skipped';

export const PAYMENT_RECHECK_OUTCOMES: readonly PaymentRecheckOutcome[] = ['ok', 'mismatch', 'failed', 'skipped'];

export const PAYMENT_RECHECK_OUTCOME_LABELS: Record<PaymentRecheckOutcome, LocalizedText> = {
  ok: { zh: '一致', en: 'Matched' },
  mismatch: { zh: '不符', en: 'Mismatch' },
  failed: { zh: '查询失败', en: 'Query failed' },
  skipped: { zh: '跳过', en: 'Skipped' },
};

export interface PaymentRecheckItem {
  orderId: string;
  userId: number;
  userName: string | null;
  userEmail: string | null;
  /** 入账时记下的支付金额 */
  amount: number;
  paymentType: string;
  orderStatus: string;
  paidAt: string | null;
  outcome: PaymentRecheckOutcome;
  /** mismatch / skipped 的原因，文案见 PAYMENT_RISK_REASON_LABELS */
  reason?: string;
  /** 平台返回的错误或查单失败的原因 */
  message?: string;
  /** mismatch 时是否新写了一条"平台复核不符"（同一订单同一原因只记一次） */
  recorded?: boolean;
}

export interface PaymentRecheckBatch {
  /** 复核窗口的起点，后续批次原样带回 */
  since: string;
  /** 窗口内待复核的订单总数 */
  total: number;
  processed: number;
  /** 下一批从这里继续；done 时为 null */
  nextCursor: string | null;
  done: boolean;
  results: PaymentRecheckItem[];
}

export function isPaymentRiskAction(value: string): value is PaymentRiskAction {
  return (PAYMENT_RISK_ACTIONS as readonly string[]).includes(value);
}

export interface PaymentRiskOrderBrief {
  id: string;
  userId: number;
  userName: string | null;
  userEmail: string | null;
  amount: number;
  payAmount: number | null;
  status: string;
  paymentType: string;
  orderType: string;
  paymentTradeNo: string | null;
  createdAt: string;
  paidAt: string | null;
}

export interface PaymentRiskEvent {
  id: string;
  action: PaymentRiskAction;
  detail: string | null;
  operator: string | null;
  createdAt: string;
  order: PaymentRiskOrderBrief;
}

export type PaymentRiskDailyPoint = { date: string } & Record<PaymentRiskAction, number>;

export interface PaymentRiskOverview {
  meta: {
    days: number;
    since: string;
    generatedAt: string;
    limits: { paidWithoutTradeNo: number; duplicateTradeNoGroups: number; topUsers: number };
  };
  summary: {
    events: Record<PaymentRiskAction, number>;
    paidWithoutTradeNo: number;
    duplicateTradeNoGroups: number;
  };
  daily: PaymentRiskDailyPoint[];
  events: {
    action: PaymentRiskAction | null;
    items: PaymentRiskEvent[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  };
  paidWithoutTradeNo: PaymentRiskOrderBrief[];
  duplicateTradeNos: { tradeNo: string; count: number; orders: PaymentRiskOrderBrief[] }[];
  topUsers: {
    userId: number;
    userName: string | null;
    userEmail: string | null;
    eventCount: number;
    orderCount: number;
    lastEventAt: string;
  }[];
}
