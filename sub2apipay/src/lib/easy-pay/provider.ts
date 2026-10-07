import type {
  PaymentProvider,
  PaymentType,
  CreatePaymentRequest,
  CreatePaymentResponse,
  QueryOrderResponse,
  PaymentNotification,
  RefundRequest,
  RefundResponse,
} from '@/lib/payment/types';
import { createPayment, queryOrder, refund } from './client';
import { verifySign } from './sign';
import { EASY_PAY_ROUTING_PARAMS, isAcceptableNotifyParamName } from './notify-params';
import { EasyPayNotifyRejectedError } from './notify-errors';
import { getEnv } from '@/lib/config';

export class EasyPayProvider implements PaymentProvider {
  readonly name: string;
  readonly providerKey = 'easypay';
  readonly supportedTypes: PaymentType[] = ['alipay', 'wxpay', 'usdt.plasma', 'usdt.polygon', 'usdc.solana', 'bank'];
  readonly defaultLimits = {
    alipay: { singleMax: 1000, dailyMax: 10000 },
    wxpay: { singleMax: 1000, dailyMax: 10000 },
    'usdt.plasma': { singleMax: 1000, dailyMax: 10000 },
    'usdt.polygon': { singleMax: 1000, dailyMax: 10000 },
    'usdc.solana': { singleMax: 1000, dailyMax: 10000 },
    bank: { singleMax: 1000, dailyMax: 10000 },
  };
  readonly instanceId?: string;
  private instanceConfig?: Record<string, string>;

  constructor(instanceId?: string, instanceConfig?: Record<string, string>) {
    this.instanceId = instanceId;
    this.instanceConfig = instanceConfig;
    this.name = instanceId ? `easy-pay:${instanceId}` : 'easy-pay';
  }

  async createPayment(request: CreatePaymentRequest): Promise<CreatePaymentResponse> {
    const result = await createPayment(
      {
        outTradeNo: request.orderId,
        amount: request.amount.toFixed(2),
        paymentType: request.paymentType,
        clientIp: request.clientIp || '127.0.0.1',
        productName: request.subject,
        notifyUrl: request.notifyUrl,
        returnUrl: request.returnUrl,
        isMobile: request.isMobile,
      },
      this.instanceConfig,
    );

    return {
      tradeNo: result.trade_no,
      payUrl: (request.isMobile && result.payurl2) || result.payurl,
      qrCode: result.qrcode,
    };
  }

  async queryOrder(tradeNo: string): Promise<QueryOrderResponse> {
    const result = await queryOrder(tradeNo, this.instanceConfig);
    return {
      tradeNo: result.trade_no,
      status: String(result.status) === '1' ? 'paid' : 'pending',
      amount: parseFloat(result.money),
      paidAt: result.endtime ? new Date(result.endtime) : undefined,
    };
  }

  async verifyNotification(rawBody: string | Buffer, _headers: Record<string, string>): Promise<PaymentNotification> {
    let pkey: string;
    let pid: string | undefined;

    if (this.instanceConfig) {
      pkey = this.instanceConfig.pkey;
      pid = this.instanceConfig.pid;
    } else {
      const env = getEnv();
      pkey = env.EASY_PAY_PKEY || '';
      pid = env.EASY_PAY_PID;
    }

    const body = typeof rawBody === 'string' ? rawBody : rawBody.toString('utf-8');
    const searchParams = new URLSearchParams(body);

    const params: Record<string, string> = {};
    for (const [key, value] of searchParams.entries()) {
      // 验签之前拦下真实通知不可能带的参数名（非纯标识符、下单专有参数，空值也算），原因见 notify-params.ts
      if (!isAcceptableNotifyParamName(key)) {
        const claimedOrderId = searchParams.get('out_trade_no') ?? '';
        throw new EasyPayNotifyRejectedError(
          `EasyPay notification rejected: unexpected param ${JSON.stringify(key)} (out_trade_no=${JSON.stringify(claimedOrderId)})`,
          'unexpected_param',
          claimedOrderId,
          key,
        );
      }
      params[key] = value;
    }

    const sign = params.sign || '';
    const paramsForSign: Record<string, string> = {};
    for (const [key, value] of Object.entries(params)) {
      if (
        key !== 'sign' &&
        key !== 'sign_type' &&
        !EASY_PAY_ROUTING_PARAMS.has(key) &&
        value !== undefined &&
        value !== null
      ) {
        paramsForSign[key] = value;
      }
    }

    const outTradeNo = params.out_trade_no || '';
    if (!pkey || !verifySign(paramsForSign, pkey, sign)) {
      throw new EasyPayNotifyRejectedError(
        'EasyPay notification signature verification failed',
        'bad_signature',
        outTradeNo,
      );
    }

    // 校验 pid 与配置一致，防止跨商户回调注入
    if (params.pid && pid && params.pid !== pid) {
      throw new EasyPayNotifyRejectedError(
        `EasyPay notification pid mismatch: expected ${pid}, got ${params.pid}`,
        'pid_mismatch',
        outTradeNo,
      );
    }

    // 校验金额为有限正数
    const amount = parseFloat(params.money || '0');
    if (!Number.isFinite(amount) || amount <= 0) {
      throw new EasyPayNotifyRejectedError(
        `EasyPay notification invalid amount: ${params.money}`,
        'invalid_amount',
        outTradeNo,
      );
    }

    return {
      tradeNo: params.trade_no || '',
      orderId: outTradeNo,
      amount,
      status: params.trade_status === 'TRADE_SUCCESS' ? 'success' : 'failed',
      rawData: params,
    };
  }

  async refund(request: RefundRequest): Promise<RefundResponse> {
    await refund(request.tradeNo, request.orderId, request.amount.toFixed(2), this.instanceConfig);
    return {
      refundId: `${request.tradeNo}-refund`,
      status: 'success',
    };
  }

  async cancelPayment(): Promise<void> {
    // EasyPay does not support cancelling payments
  }
}
