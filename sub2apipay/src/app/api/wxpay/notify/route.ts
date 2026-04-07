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
        throw new Error(`Wxpay instance not found: ${instanceId}`);
      }
      provider = await createProviderFromInstance('wxpay', instanceId, instanceConfig);
    } else {
      const env = getEnv();
      if (!env.WXPAY_PUBLIC_KEY || !env.WXPAY_MCH_ID || !env.WXPAY_PRIVATE_KEY || !env.WXPAY_API_V3_KEY) {
        return Response.json({ code: 'SUCCESS', message: '成功' });
      }

      await ensureDBProviders();
      provider = paymentRegistry.getProvider('wxpay_direct' as PaymentType);
    }

    if (!provider) {
      throw new Error('Wxpay provider unavailable');
    }

    const rawBody = await request.text();
    const headers = extractHeaders(request);

    const notification = await provider.verifyNotification(rawBody, headers);
    if (!notification) {
      return Response.json({ code: 'SUCCESS', message: '成功' });
    }
    const success = await handlePaymentNotify(notification, provider.name);
    return Response.json(success ? { code: 'SUCCESS', message: '成功' } : { code: 'FAIL', message: '处理失败' }, {
      status: success ? 200 : 500,
    });
  } catch (error) {
    console.error('Wxpay notify error:', error);
    return Response.json({ code: 'FAIL', message: '处理失败' }, { status: 500 });
  }
}
