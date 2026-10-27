export interface PaymentFeeConfig {
  enabled: boolean;
  /** Percentage, for example 3.9 means 3.9%. */
  feeRate: number;
  /** Fixed fee in CNY. */
  fixedFee: number;
}

export interface PaymentFeeQuote {
  settlementAmount: number;
  payAmount: number;
  feeAmount: number;
}

export const NO_PAYMENT_FEE: PaymentFeeConfig = { enabled: false, feeRate: 0, fixedFee: 0 };

const RATE_SCALE = BigInt(1000000);
const ONE = BigInt(1);

function hasDecimalPrecision(value: number, decimals: number): boolean {
  const scale = 10 ** decimals;
  return Math.abs(value * scale - Math.round(value * scale)) < 1e-6;
}

export function validatePaymentFeeConfig(config: PaymentFeeConfig): string | null {
  if (typeof config.enabled !== 'boolean') return 'enabled 必须是布尔值';
  if (!Number.isFinite(config.feeRate) || config.feeRate < 0 || config.feeRate >= 100) {
    return '手续费率必须大于等于 0 且小于 100';
  }
  if (!hasDecimalPrecision(config.feeRate, 4)) return '手续费率最多支持 4 位小数';
  if (!Number.isFinite(config.fixedFee) || config.fixedFee < 0) return '固定手续费必须是非负数';
  if (!hasDecimalPrecision(config.fixedFee, 2)) return '固定手续费最多支持 2 位小数';
  if (config.enabled && config.feeRate === 0 && config.fixedFee === 0) {
    return '启用用户承担手续费时，费率或固定手续费至少一项必须大于 0';
  }
  return null;
}

export function assertValidPaymentFeeConfig(config: PaymentFeeConfig): void {
  const error = validatePaymentFeeConfig(config);
  if (error) throw new Error(`支付方式手续费配置无效: ${error}`);
}

function toCents(value: number, field: string): bigint {
  if (!Number.isFinite(value) || value < 0 || !hasDecimalPrecision(value, 2)) {
    throw new Error(`${field} 必须是非负数且最多保留 2 位小数`);
  }
  return BigInt(Math.round(value * 100));
}

function fromCents(value: bigint): number {
  return Number(value) / 100;
}

/**
 * Gross up the settlement amount so the configured fee is borne by the payer.
 * The total is rounded up to cents; the fee is the total minus the settlement.
 */
export function calculatePaymentFeeQuote(settlementAmount: number, config: PaymentFeeConfig): PaymentFeeQuote {
  if (!Number.isFinite(settlementAmount) || settlementAmount < 0 || !hasDecimalPrecision(settlementAmount, 2)) {
    throw new Error('结算金额必须是非负数且最多保留 2 位小数');
  }
  assertValidPaymentFeeConfig(config);

  const settlementCents = toCents(settlementAmount, '结算金额');
  if (!config.enabled) {
    return {
      settlementAmount: fromCents(settlementCents),
      payAmount: fromCents(settlementCents),
      feeAmount: 0,
    };
  }

  const fixedFeeCents = toCents(config.fixedFee, '固定手续费');
  const rateParts = BigInt(Math.round(config.feeRate * 10000));
  const denominator = RATE_SCALE - rateParts;
  const numerator = (settlementCents + fixedFeeCents) * RATE_SCALE;
  const payCents = (numerator + denominator - ONE) / denominator;

  return {
    settlementAmount: fromCents(settlementCents),
    payAmount: fromCents(payCents),
    feeAmount: fromCents(payCents - settlementCents),
  };
}

export function legacyFeeRateConfig(feeRate: number): PaymentFeeConfig {
  if (!Number.isFinite(feeRate) || feeRate < 0) return { ...NO_PAYMENT_FEE };
  return { enabled: feeRate > 0, feeRate, fixedFee: 0 };
}

export function feeConfigFrom(value: number | PaymentFeeConfig): PaymentFeeConfig {
  return typeof value === 'number' ? legacyFeeRateConfig(value) : value;
}

export function hasPaymentFee(config: PaymentFeeConfig): boolean {
  return config.enabled && (config.feeRate > 0 || config.fixedFee > 0);
}

export function centsToAmount(value: number): number {
  return value / 100;
}

export function amountToCents(value: number): number {
  if (!Number.isFinite(value) || value < 0) return 0;
  return Math.max(0, Math.round(value * 100));
}

export function quoteFromCents(settlementCents: number, config: PaymentFeeConfig): PaymentFeeQuote {
  return calculatePaymentFeeQuote(centsToAmount(settlementCents), config);
}
