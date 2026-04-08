import { initPaymentProviders, paymentRegistry } from '@/lib/payment';
import { Prisma } from '@prisma/client';

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
 * 获取指定支付渠道的手续费率（百分比）。
 * 优先级：FEE_RATE_{TYPE} > FEE_RATE_PROVIDER_{KEY} > 0
 */
export function getMethodFeeRate(paymentType: string): number {
  // 渠道级别：FEE_RATE_ALIPAY / FEE_RATE_WXPAY / FEE_RATE_STRIPE
  const methodRaw = process.env[`FEE_RATE_${paymentType.toUpperCase()}`];
  if (methodRaw !== undefined && methodRaw !== '') {
    const num = Number(methodRaw);
    if (Number.isFinite(num) && num >= 0) return num;
  }

  // 提供商级别：FEE_RATE_PROVIDER_EASYPAY / FEE_RATE_PROVIDER_STRIPE
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

/** decimal.js ROUND_UP = 0（远离零方向取整） */
const ROUND_UP = 0;

/**
 * 根据到账金额和手续费率计算实付金额（使用 Decimal 精确计算，避免浮点误差）。
 * feeAmount = ceil(rechargeAmount * feeRate / 100, 保留2位小数)
 * payAmount = rechargeAmount + feeAmount
 */
export function calculatePayAmount(rechargeAmount: number, feeRate: number): string {
  if (feeRate <= 0) return rechargeAmount.toFixed(2);
  const amount = new Prisma.Decimal(rechargeAmount);
  const rate = new Prisma.Decimal(feeRate.toString());
  const feeAmount = amount.mul(rate).div(100).toDecimalPlaces(2, ROUND_UP);
  return amount.plus(feeAmount).toFixed(2);
}

export function calculatePayAmountNumber(rechargeAmount: number, feeRate: number): number {
  return Number(calculatePayAmount(rechargeAmount, feeRate));
}

function toCents(amount: number): number {
  return Math.max(0, Math.round(amount * 100));
}

/**
 * 在网关实付限额（CNY）下，反推最多允许的结算金额（CNY）。
 * 结算金额经过手续费后不会超过给定的实付上限。
 */
export function getMaxSettlementAmountForPayLimit(payLimit: number, feeRate: number): number {
  if (!Number.isFinite(payLimit) || payLimit <= 0) return 0;
  if (feeRate <= 0) return Number(payLimit.toFixed(2));

  const limitCents = toCents(payLimit);
  let left = 0;
  let right = limitCents;
  let best = 0;

  while (left <= right) {
    const mid = Math.floor((left + right) / 2);
    const settlementAmount = mid / 100;
    const actualPayCents = toCents(calculatePayAmountNumber(settlementAmount, feeRate));
    if (actualPayCents <= limitCents) {
      best = mid;
      left = mid + 1;
    } else {
      right = mid - 1;
    }
  }

  return best / 100;
}

/**
 * 在网关实付下限（CNY）下，反推至少需要的结算金额（CNY）。
 * 结算金额经过手续费后不会低于给定的实付下限。
 */
export function getMinSettlementAmountForPayLimit(payLimit: number, feeRate: number): number {
  if (!Number.isFinite(payLimit) || payLimit <= 0) return 0;
  if (feeRate <= 0) return Number(payLimit.toFixed(2));

  const limitCents = toCents(payLimit);
  let left = 0;
  let right = limitCents;
  let best = limitCents;

  while (left <= right) {
    const mid = Math.floor((left + right) / 2);
    const settlementAmount = mid / 100;
    const actualPayCents = toCents(calculatePayAmountNumber(settlementAmount, feeRate));
    if (actualPayCents >= limitCents) {
      best = mid;
      right = mid - 1;
    } else {
      left = mid + 1;
    }
  }

  return best / 100;
}
