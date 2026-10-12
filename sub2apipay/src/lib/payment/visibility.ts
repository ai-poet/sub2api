import { prisma } from '@/lib/db';
import { ensureDBProviders, paymentRegistry } from '@/lib/payment';
import { getSystemConfig } from '@/lib/system-config';
import { resolveEnabledPaymentTypes } from '@/lib/payment/resolve-enabled-types';
import { instanceSupportsPaymentType, isOptInPaymentType, parsePaymentTypeList } from '@/lib/payment/opt-in-types';

function getProviderKeyForType(type: string): string | undefined {
  const registry = paymentRegistry as {
    getProviderKey?: (paymentType: string) => string | undefined;
    getProvider?: (paymentType: string) => { providerKey?: string } | undefined;
  };

  if (typeof registry.getProviderKey === 'function') {
    return registry.getProviderKey(type);
  }

  if (typeof registry.getProvider === 'function') {
    try {
      return registry.getProvider(type)?.providerKey;
    } catch {
      return undefined;
    }
  }

  return type || undefined;
}

export async function getVisiblePaymentTypes(): Promise<string[]> {
  await ensureDBProviders();

  const supportedTypes = paymentRegistry.getSupportedTypes();
  const configuredTypes = await getSystemConfig('ENABLED_PAYMENT_TYPES');
  const enabledTypes = resolveEnabledPaymentTypes(supportedTypes, configuredTypes);
  if (enabledTypes.length === 0) return [];

  const providerKeys = [...new Set(enabledTypes.map((type) => getProviderKeyForType(type)).filter(Boolean))] as string[];
  const activeInstances = providerKeys.length
    ? await prisma.paymentProviderInstance.findMany({
        where: { providerKey: { in: providerKeys }, enabled: true },
        select: { providerKey: true, supportedTypes: true },
      })
    : [];

  const visible = enabledTypes.filter((type) => {
    const providerKey = getProviderKeyForType(type);
    if (!providerKey) return false;

    const providerInstances = activeInstances.filter((instance) => instance.providerKey === providerKey);
    if (providerInstances.length === 0) {
      // 没有数据库实例（仅环境变量配置）：必须显式启用的方式只认全局列表里明确写出的
      return !isOptInPaymentType(type) || parsePaymentTypeList(configuredTypes).includes(type);
    }

    return providerInstances.some((instance) => instanceSupportsPaymentType(instance.supportedTypes, type));
  });

  return visible;
}
