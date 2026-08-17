/**
 * 内嵌页面 → 宿主页面的消息桥。
 *
 * 支付页通常以 iframe 形式嵌在主站（Sub2API 前端）里，宿主顶栏展示的余额来自主站自己的
 * 用户接口。订单完成只发生在这个 iframe 内部，宿主无从感知，余额会一直停在支付前的数值，
 * 直到用户手动刷新。这里在订单完成时通知宿主，让它重新拉取用户信息。
 */

export const PAYMENT_SUCCESS_MESSAGE_TYPE = 'SUB2API_PAYMENT_SUCCESS' as const

export interface PaymentSuccessMessage {
  type: typeof PAYMENT_SUCCESS_MESSAGE_TYPE
  orderId?: string
  orderType?: string
}

/**
 * 向宿主页面广播一次订单完成事件。
 *
 * targetOrigin 优先取 src_host（宿主通过 query 告知自己的 origin）；缺失时退回 '*'：
 * 消息体只有订单号与订单类型，没有 token 或金额，宽松投递不会泄露凭据。
 */
export function notifyParentPaymentSuccess(
  message: Omit<PaymentSuccessMessage, 'type'>,
  srcHost?: string,
): void {
  if (typeof window === 'undefined') return
  // 不在 iframe 内（独立标签页打开）时没有宿主可通知。
  if (window.parent === window) return

  let targetOrigin = '*'
  if (srcHost) {
    try {
      targetOrigin = new URL(srcHost).origin
    } catch {
      targetOrigin = '*'
    }
  }

  try {
    window.parent.postMessage(
      { type: PAYMENT_SUCCESS_MESSAGE_TYPE, ...message } satisfies PaymentSuccessMessage,
      targetOrigin,
    )
  } catch {
    // 宿主已卸载或跨域策略拒绝时静默失败：这只是刷新提示，不影响支付结果。
  }
}
