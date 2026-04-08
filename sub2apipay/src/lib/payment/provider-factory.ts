import type { PaymentProvider } from '@/lib/payment/types';

export async function createProviderFromInstance(
  providerKey: string,
  instanceId: string,
  instanceConfig: Record<string, string>,
): Promise<PaymentProvider | null> {
  switch (providerKey) {
    case 'easypay': {
      const { EasyPayProvider } = await import('@/lib/easy-pay/provider');
      return new EasyPayProvider(instanceId, instanceConfig);
    }
    case 'alipay': {
      const { AlipayProvider } = await import('@/lib/alipay/provider');
      return new AlipayProvider(instanceId, instanceConfig);
    }
    case 'wxpay': {
      const { WxpayProvider } = await import('@/lib/wxpay/provider');
      return new WxpayProvider(instanceId, instanceConfig);
    }
    case 'stripe': {
      const { StripeProvider } = await import('@/lib/stripe/provider');
      return new StripeProvider(instanceId, instanceConfig);
    }
    default:
      return null;
  }
}
