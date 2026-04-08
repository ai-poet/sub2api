import { NextRequest } from 'next/server';
import { handlePaymentNotify } from '@/lib/order/service';
import { ensureDBProviders, paymentRegistry } from '@/lib/payment';
import type { PaymentType } from '@/lib/payment';
import { getEnv } from '@/lib/config';
import { extractHeaders } from '@/lib/utils/api';
import { getInstanceConfig } from '@/lib/payment/load-balancer';
import { createProviderFromInstance } from '@/lib/payment/provider-factory';

export async function POST(request: NextRequest) {
  try {
    const instanceId = request.nextUrl.searchParams.get('inst')?.trim();
    let provider = null;

    if (instanceId) {
      const instanceConfig = await getInstanceConfig(instanceId);
      if (!instanceConfig) {
        throw new Error(`Alipay instance not found: ${instanceId}`);
      }
      provider = await createProviderFromInstance('alipay', instanceId, instanceConfig);
    } else {
      const env = getEnv();
      if (!env.ALIPAY_APP_ID || !env.ALIPAY_PRIVATE_KEY || !env.ALIPAY_PUBLIC_KEY) {
        return new Response('success', { headers: { 'Content-Type': 'text/plain' } });
      }

      await ensureDBProviders();
      provider = paymentRegistry.getProvider('alipay_direct' as PaymentType);
    }

    if (!provider) {
      throw new Error('Alipay provider unavailable');
    }

    const rawBody = await request.text();
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
    console.error('Alipay notify error:', error);
    return new Response('fail', {
      headers: { 'Content-Type': 'text/plain' },
    });
  }
}
