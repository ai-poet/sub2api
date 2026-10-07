/** 易支付异步通知在验签阶段被拒绝的原因。 */
export type EasyPayNotifyRejectReason = 'unexpected_param' | 'bad_signature' | 'pid_mismatch' | 'invalid_amount';

/**
 * 易支付异步通知在验签阶段的拒绝。message 与以前完全相同（日志排查和既有测试都按它匹配），
 * 另外带上结构化的原因与订单号，供"支付风控"面板记录被拦截的回调。
 */
export class EasyPayNotifyRejectedError extends Error {
  readonly reason: EasyPayNotifyRejectReason;
  /** 回调里声称的商户订单号，未经验证，只用来把记录挂到对应订单上。 */
  readonly outTradeNo: string;
  /** reason 为 unexpected_param 时被拒的参数名。 */
  readonly param?: string;

  constructor(message: string, reason: EasyPayNotifyRejectReason, outTradeNo: string, param?: string) {
    super(message);
    this.name = 'EasyPayNotifyRejectedError';
    this.reason = reason;
    this.outTradeNo = outTradeNo;
    this.param = param;
  }
}
