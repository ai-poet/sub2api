import { initPaymentProviders, paymentRegistry } from '@/lib/payment';
import { getSystemConfig } from '@/lib/system-config';
import {
  amountToCents,
  calculatePaymentFeeQuote,
  centsToAmount,
  feeConfigFrom,
  legacyFeeRateConfig,
  quoteFromCents,
  validatePaymentFeeConfig,
  type PaymentFeeConfig,
} from './fee-calc';

export { calculatePaymentFeeQuote, hasPaymentFee, validatePaymentFeeConfig } from './fee-calc';
export type { PaymentFeeConfig, PaymentFeeQuote } from './fee-calc';

export const PAYMENT_METHOD_FEES_CONFIG_KEY = 'PAYMENT_METHOD_FEES';

function getProviderKeyForType(paymentType: string): string | undefined {
  const registry = paymentRegistry as {
    getProviderKey?: (type: string) => string | undefined;
    getProvider?: (type: string) => { providerKey?: string } | undefined;
  };

  if (typeof registry.getProviderKey === 'function') {
    return registry.getProviderKey(paymentType);
  }

  if (typeof registry.getProvider === 'function') {
    try {
      return registry.getProvider(paymentType)?.providerKey;
    } catch {
      return undefined;
    }
  }

  return undefined;
}

/**
 * 旧环境变量费率（百分比）：FEE_RATE_{TYPE} > FEE_RATE_PROVIDER_{KEY} > 0。
 * 新部署应在支付管理中配置 PAYMENT_METHOD_FEES。
 */
export function getMethodFeeRate(paymentType: string): number {
  const methodRaw = process.env[`FEE_RATE_${paymentType.toUpperCase()}`];
  if (methodRaw !== undefined && methodRaw !== '') {
    const num = Number(methodRaw);
    if (Number.isFinite(num) && num >= 0) return num;
  }

  initPaymentProviders();
  const providerKey = getProviderKeyForType(paymentType);
  if (providerKey) {
    const providerRaw = process.env[`FEE_RATE_PROVIDER_${providerKey.toUpperCase()}`];
    if (providerRaw !== undefined && providerRaw !== '') {
      const num = Number(providerRaw);
      if (Number.isFinite(num) && num >= 0) return num;
    }
  }

  return 0;
}

export function parsePaymentMethodFees(raw: string): Record<string, PaymentFeeConfig> {
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    throw new Error('PAYMENT_METHOD_FEES 必须是合法 JSON');
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('PAYMENT_METHOD_FEES 必须是按支付方式 keyed 的 JSON 对象');
  }

  const result: Record<string, PaymentFeeConfig> = {};
  for (const [paymentType, entry] of Object.entries(value)) {
    if (!/^[a-z0-9._-]+$/i.test(paymentType)) throw new Error(`无效的支付方式: ${paymentType}`);
    if (!entry || typeof entry !== 'object' || Array.isArray(entry)) {
      throw new Error(`支付方式 ${paymentType} 的手续费配置必须是对象`);
    }
    const candidate = entry as Record<string, unknown>;
    const config: PaymentFeeConfig = {
      enabled: candidate.enabled as boolean,
      feeRate: candidate.feeRate as number,
      fixedFee: candidate.fixedFee as number,
    };
    const error = validatePaymentFeeConfig(config);
    if (error) throw new Error(`支付方式 ${paymentType}: ${error}`);
    result[paymentType] = config;
  }
  return result;
}

function getLegacyMethodFeeConfig(paymentType: string): PaymentFeeConfig {
  return legacyFeeRateConfig(getMethodFeeRate(paymentType));
}

export async function getMethodFeeConfigs(paymentTypes: string[]): Promise<Record<string, PaymentFeeConfig>> {
  const raw = await getSystemConfig(PAYMENT_METHOD_FEES_CONFIG_KEY);
  const configured = raw ? parsePaymentMethodFees(raw) : {};
  return Object.fromEntries(
    paymentTypes.map((paymentType) => [paymentType, configured[paymentType] ?? getLegacyMethodFeeConfig(paymentType)]),
  );
}

export async function getMethodFeeConfig(paymentType: string): Promise<PaymentFeeConfig> {
  return (await getMethodFeeConfigs([paymentType]))[paymentType];
}

/** 兼容旧调用方：仅传费率时也表示启用百分比手续费。 */
export function calculatePayAmount(rechargeAmount: number, fee: number | PaymentFeeConfig): string {
  return calculatePaymentFeeQuote(rechargeAmount, feeConfigFrom(fee)).payAmount.toFixed(2);
}

export function calculatePayAmountNumber(rechargeAmount: number, fee: number | PaymentFeeConfig): number {
  return Number(calculatePayAmount(rechargeAmount, fee));
}

/** 在网关实付上限（CNY）下反推最大结算金额；0 表示没有可支付的正金额。 */
export function getMaxSettlementAmountForPayLimit(payLimit: number, fee: number | PaymentFeeConfig): number {
  if (!Number.isFinite(payLimit) || payLimit <= 0) return 0;
  const config = feeConfigFrom(fee);
  const limitCents = amountToCents(payLimit);
  let left = 1;
  let right = limitCents;
  let best = 0;

  while (left <= right) {
    const mid = Math.floor((left + right) / 2);
    const actualPayCents = amountToCents(quoteFromCents(mid, config).payAmount);
    if (actualPayCents <= limitCents) {
      best = mid;
      left = mid + 1;
    } else {
      right = mid - 1;
    }
  }

  return centsToAmount(best);
}

/** 在网关实付下限（CNY）下反推最小正结算金额。 */
export function getMinSettlementAmountForPayLimit(payLimit: number, fee: number | PaymentFeeConfig): number {
  if (!Number.isFinite(payLimit) || payLimit <= 0) return 0;
  const config = feeConfigFrom(fee);
  const limitCents = amountToCents(payLimit);
  let left = 1;
  let right = limitCents;
  let best = limitCents;

  while (left <= right) {
    const mid = Math.floor((left + right) / 2);
    const actualPayCents = amountToCents(quoteFromCents(mid, config).payAmount);
    if (actualPayCents >= limitCents) {
      best = mid;
      right = mid - 1;
    } else {
      left = mid + 1;
    }
  }

  return centsToAmount(best);
}
