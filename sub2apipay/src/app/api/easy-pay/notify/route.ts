import { NextRequest } from 'next/server';
import { handlePaymentNotify } from '@/lib/order/service';
import { ensureDBProviders, paymentRegistry } from '@/lib/payment';
import type { PaymentType, PaymentProvider } from '@/lib/payment';
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

async function processNotification(request: NextRequest, rawBody: string) {
  try {
    const provider = await getProvider(request);
    const headers = extractHeaders(request);

    const notification = await provider.verifyNotification(rawBody, headers);
    if (!notification) {
      return new Response('success', { headers: { 'Content-Type': 'text/plain' } });
    }
    const success = await handlePaymentNotify(notification, provider.name);
    return new Response(success ? 'success' : 'fail', {
      headers: { 'Content-Type': 'text/plain' },
    });
  } catch (error) {
    console.error('EasyPay notify error:', error);
    return new Response('fail', {
      headers: { 'Content-Type': 'text/plain' },
    });
  }
}

export async function GET(request: NextRequest) {
  return processNotification(request, request.nextUrl.searchParams.toString());
}

export async function POST(request: NextRequest) {
  const rawBody = await request.text();
  return processNotification(request, rawBody || request.nextUrl.searchParams.toString());
}
