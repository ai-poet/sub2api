import type { Locale } from '@/lib/locale';
import { prisma } from '@/lib/db';
import { ensureDBProviders, paymentRegistry } from '@/lib/payment';
import { getSystemConfig } from '@/lib/system-config';
import { resolveEnabledPaymentTypes } from '@/lib/payment/resolve-enabled-types';

const EASY_PAY_FIAT_TYPES = new Set(['alipay', 'wxpay']);
const EASY_PAY_CRYPTO_TYPES = new Set(['usdt.plasma', 'usdt.polygon', 'usdc.solana']);

function filterEasyPayTypesByLocale(types: string[], locale: Locale): string[] {
  const allowFiat = locale === 'zh';
  return types.filter((type) => {
    if (EASY_PAY_FIAT_TYPES.has(type)) return allowFiat;
    if (EASY_PAY_CRYPTO_TYPES.has(type)) return !allowFiat;
    return true;
  });
}

export async function getVisiblePaymentTypes(locale?: Locale): Promise<string[]> {
  await ensureDBProviders();

  const supportedTypes = paymentRegistry.getSupportedTypes();
  const configuredTypes = await getSystemConfig('ENABLED_PAYMENT_TYPES');
  const enabledTypes = resolveEnabledPaymentTypes(supportedTypes, configuredTypes);
  if (enabledTypes.length === 0) return [];

  const providerKeys = [...new Set(enabledTypes.map((type) => paymentRegistry.getProviderKey(type)).filter(Boolean))] as string[];
  const activeInstances = providerKeys.length
    ? await prisma.paymentProviderInstance.findMany({
        where: { providerKey: { in: providerKeys }, enabled: true },
        select: { providerKey: true, supportedTypes: true },
      })
    : [];

  const visible = enabledTypes.filter((type) => {
    const providerKey = paymentRegistry.getProviderKey(type);
    if (!providerKey) return false;

    const providerInstances = activeInstances.filter((instance) => instance.providerKey === providerKey);
    if (providerInstances.length === 0) {
      return true;
    }

    return providerInstances.some((instance) => {
      const supportedTypes = instance.supportedTypes
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean);
      return supportedTypes.length === 0 || supportedTypes.includes(type);
    });
  });

  return locale ? filterEasyPayTypesByLocale(visible, locale) : visible;
}
