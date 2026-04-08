import { PAYMENT_PREFIX } from './constants';

export const DEFAULT_USD_EXCHANGE_RATE = 7.2;
export const DEFAULT_BALANCE_CREDIT_CNY_PER_USD = 1;

export interface SettlementDisplay {
  currency: 'CNY' | 'USD';
  symbol: '¥' | '$';
  amount: number;
  exchangeRate: number | null;
}

export function normalizeUsdExchangeRate(value: number | string | null | undefined): number | null {
  const parsed = typeof value === 'string' ? parseFloat(value) : value;
  if (parsed == null || !Number.isFinite(parsed) || parsed <= 0) {
    return null;
  }
  return Math.round(parsed * 10000) / 10000;
}

export function normalizeBalanceCreditCnyPerUsd(value: number | string | null | undefined): number | null {
  const parsed = typeof value === 'string' ? parseFloat(value) : value;
  if (parsed == null || !Number.isFinite(parsed) || parsed <= 0) {
    return null;
  }
  return Math.round(parsed * 10000) / 10000;
}

export function normalizeLegacyBalanceCreditUsdPerCny(value: number | string | null | undefined): number | null {
  const parsed = typeof value === 'string' ? parseFloat(value) : value;
  if (parsed == null || !Number.isFinite(parsed) || parsed <= 0) {
    return null;
  }
  return Math.round(parsed * 10000) / 10000;
}

export function legacyBalanceCreditUsdPerCnyToCnyPerUsd(
  legacyValue: number | string | null | undefined,
): number | null {
  const normalizedLegacy = normalizeLegacyBalanceCreditUsdPerCny(legacyValue);
  if (!normalizedLegacy) {
    return null;
  }
  return Math.round((1 / normalizedLegacy) * 10000) / 10000;
}

export function resolveBalanceCreditCnyPerUsd(
  cnyPerUsdValue: number | string | null | undefined,
  legacyUsdPerCnyValue?: number | string | null | undefined,
): number {
  return (
    normalizeBalanceCreditCnyPerUsd(cnyPerUsdValue) ??
    legacyBalanceCreditUsdPerCnyToCnyPerUsd(legacyUsdPerCnyValue) ??
    DEFAULT_BALANCE_CREDIT_CNY_PER_USD
  );
}

export function isStablecoinPaymentType(type: string | null | undefined): boolean {
  return !!type && (type.startsWith(PAYMENT_PREFIX.USDT) || type.startsWith(PAYMENT_PREFIX.USDC));
}

export function convertCnyToUsd(amountCny: number, usdExchangeRate: number | string | null | undefined): number | null {
  const normalizedRate = normalizeUsdExchangeRate(usdExchangeRate);
  if (!normalizedRate || !Number.isFinite(amountCny)) {
    return null;
  }
  return Math.round((amountCny / normalizedRate) * 100) / 100;
}

export function getSettlementDisplay(
  amountCny: number,
  paymentType: string | null | undefined,
  usdExchangeRate: number | string | null | undefined,
): SettlementDisplay {
  const normalizedAmount = Number.isFinite(amountCny) ? Math.round(amountCny * 100) / 100 : 0;
  if (!isStablecoinPaymentType(paymentType)) {
    return { currency: 'CNY', symbol: '¥', amount: normalizedAmount, exchangeRate: null };
  }

  const normalizedRate = normalizeUsdExchangeRate(usdExchangeRate);
  if (!normalizedRate) {
    return { currency: 'CNY', symbol: '¥', amount: normalizedAmount, exchangeRate: null };
  }

  return {
    currency: 'USD',
    symbol: '$',
    amount: Math.round((normalizedAmount / normalizedRate) * 100) / 100,
    exchangeRate: normalizedRate,
  };
}

export function convertUsdBalanceToCnyPayment(
  amountUsd: number,
  balanceCreditCnyPerUsd: number | string | null | undefined,
): number | null {
  const normalizedCost = normalizeBalanceCreditCnyPerUsd(balanceCreditCnyPerUsd);
  if (!normalizedCost || !Number.isFinite(amountUsd)) {
    return null;
  }
  return Math.round(amountUsd * normalizedCost * 100) / 100;
}

export function convertCnySettlementToUsdBalance(
  amountCny: number,
  balanceCreditCnyPerUsd: number | string | null | undefined,
): number | null {
  const normalizedCost = normalizeBalanceCreditCnyPerUsd(balanceCreditCnyPerUsd);
  if (!normalizedCost || !Number.isFinite(amountCny)) {
    return null;
  }
  return Math.round((amountCny / normalizedCost) * 100) / 100;
}

export function getBalanceRechargeSettlementDisplay(
  creditedAmountUsd: number,
  paymentType: string | null | undefined,
  balanceCreditCnyPerUsd: number | string | null | undefined,
): SettlementDisplay {
  const normalizedAmount = Number.isFinite(creditedAmountUsd) ? Math.round(creditedAmountUsd * 100) / 100 : 0;
  if (isStablecoinPaymentType(paymentType)) {
    return { currency: 'USD', symbol: '$', amount: normalizedAmount, exchangeRate: null };
  }

  const amountCny = convertUsdBalanceToCnyPayment(normalizedAmount, balanceCreditCnyPerUsd);
  return {
    currency: 'CNY',
    symbol: '¥',
    amount: amountCny ?? normalizedAmount,
    exchangeRate: null,
  };
}
