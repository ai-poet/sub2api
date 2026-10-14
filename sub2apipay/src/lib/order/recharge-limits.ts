import { getEnv } from '@/lib/config';
import { getSystemConfig } from '@/lib/system-config';
import type { Locale } from '@/lib/locale';

/**
 * 全局到账限额（USD），0 = 不限。
 *
 * 取值顺序：后台「支付配置」→ 环境变量（MIN_RECHARGE_AMOUNT / MAX_RECHARGE_AMOUNT /
 * MAX_DAILY_RECHARGE_AMOUNT）。都没设置、为空或无效时不限制，代码里不写死默认值。
 */
export interface RechargeLimits {
  /** 单笔最小到账金额 */
  minAmount: number;
  /** 单笔最大到账金额 */
  maxAmount: number;
  /** 每位用户每日累计到账上限 */
  maxDailyAmount: number;
}

export function parseRechargeLimit(configured: string | undefined, fallback: number | undefined): number {
  if (configured !== undefined && configured.trim() !== '') {
    const num = Number(configured);
    if (Number.isFinite(num) && num >= 0) return num;
  }
  return fallback !== undefined && Number.isFinite(fallback) && fallback >= 0 ? fallback : 0;
}

export async function getRechargeLimits(): Promise<RechargeLimits> {
  const env = getEnv();
  const [min, max, daily] = await Promise.all([
    getSystemConfig('RECHARGE_MIN_AMOUNT'),
    getSystemConfig('RECHARGE_MAX_AMOUNT'),
    getSystemConfig('DAILY_RECHARGE_LIMIT'),
  ]);
  return {
    minAmount: parseRechargeLimit(min, env.MIN_RECHARGE_AMOUNT),
    maxAmount: parseRechargeLimit(max, env.MAX_RECHARGE_AMOUNT),
    maxDailyAmount: parseRechargeLimit(daily, env.MAX_DAILY_RECHARGE_AMOUNT),
  };
}

/** 到账金额是否落在全局单笔范围内（任一侧为 0 时该侧不限）。 */
export function isWithinRechargeRange(amount: number, limits: Pick<RechargeLimits, 'minAmount' | 'maxAmount'>): boolean {
  if (limits.minAmount > 0 && amount < limits.minAmount) return false;
  if (limits.maxAmount > 0 && amount > limits.maxAmount) return false;
  return true;
}

/** 超出范围时的提示：只列出实际设置了的那一侧。 */
export function rechargeRangeMessage(
  locale: Locale,
  limits: Pick<RechargeLimits, 'minAmount' | 'maxAmount'>,
): string {
  const { minAmount: min, maxAmount: max } = limits;
  if (min > 0 && max > 0) {
    return locale === 'en'
      ? `Credited balance must be between $${min} and $${max}`
      : `到账金额需在 $${min} - $${max} 之间`;
  }
  if (min > 0) {
    return locale === 'en' ? `Credited balance must be at least $${min}` : `到账金额不能低于 $${min}`;
  }
  return locale === 'en' ? `Credited balance must not exceed $${max}` : `到账金额不能超过 $${max}`;
}
