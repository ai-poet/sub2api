import { NextRequest } from 'next/server';
import { handlePaymentNotify } from '@/lib/order/service';
import { ensureDBProviders, paymentRegistry } from '@/lib/payment';
import type { PaymentType, PaymentProvider, PaymentNotification } from '@/lib/payment';
import { EasyPayProvider } from '@/lib/easy-pay/provider';
import { getInstanceConfig } from '@/lib/payment/load-balancer';
import { extractHeaders } from '@/lib/utils/api';

async function getProvider(request: NextRequest): Promise<PaymentProvider> {
  const instId = request.nextUrl.searchParams.get('inst');

  if (instId) {
    // 多实例模式：根据实例 ID 获取配置
    const config = await getInstanceConfig(instId);
    if (!config) {
      throw new Error(`EasyPay instance not found: ${instId}`);
    }
    return new EasyPayProvider(instId, config);
  }

  // 回退到环境变量单实例模式
  await ensureDBProviders();
  return paymentRegistry.getProvider('easypay' as PaymentType);
}

function textResponse(body: 'success' | 'fail', status: number) {
  return new Response(body, { status, headers: { 'Content-Type': 'text/plain' } });
}

/**
 * 易支付异步通知。
 *
 * 处理失败一律回 HTTP 500 + 正文 fail：按正文判定的平台照旧看到 fail，按 HTTP 状态判定的平台
 * 也会重试——以前失败也回 200，这类平台会把通知记成"成功"且不再重试，订单就卡在未支付。
 * 所有失败日志都带 out_trade_no，排查时直接按订单号搜。
 */
async function processNotification(request: NextRequest, rawBody: string) {
  let notification: PaymentNotification | null = null;
  try {
    const provider = await getProvider(request);
    const headers = extractHeaders(request);

    notification = await provider.verifyNotification(rawBody, headers);
    if (!notification) {
      return textResponse('success', 200);
    }
    const success = await handlePaymentNotify(notification, provider.name);
    if (!success) {
      console.error(
        `EasyPay notify rejected: order=${notification.orderId} trade=${notification.tradeNo} amount=${notification.amount}`,
      );
      return textResponse('fail', 500);
    }
    return textResponse('success', 200);
  } catch (error) {
    const scope = notification ? ` (order=${notification.orderId} trade=${notification.tradeNo})` : '';
    console.error(`EasyPay notify error${scope}:`, error);
    return textResponse('fail', 500);
  }
}

export async function GET(request: NextRequest) {
  return processNotification(request, request.nextUrl.searchParams.toString());
}

export async function POST(request: NextRequest) {
  const rawBody = await request.text();
  return processNotification(request, rawBody || request.nextUrl.searchParams.toString());
}
