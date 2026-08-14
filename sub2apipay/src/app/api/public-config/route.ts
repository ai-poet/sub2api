import { NextResponse } from 'next/server';
import { getEnv } from '@/lib/config';
import { getSystemConfig } from '@/lib/system-config';
import {
  DEFAULT_USD_EXCHANGE_RATE,
  legacyBalanceCreditUsdPerCnyToCnyPerUsd,
  normalizeBalanceCreditCnyPerUsd,
  normalizeUsdExchangeRate,
} from '@/lib/currency';

/**
 * 匿名可访问的汇率端点。
 *
 * 存在的原因：网关落地页的公开定价目录要给**未登录访客**展示人民币价格，
 * 而 /api/user 需要 user_id + token（它同时返回用户余额，不能放开鉴权）。
 * 这里只暴露与用户无关的汇率配置，不含任何用户数据、密钥或支付渠道信息。
 *
 * 汇率解析逻辑与 /api/user 保持一致，避免落地页与支付页显示的汇率产生分歧。
 */
export async function GET() {
  try {
    const env = getEnv();

    const [usdExchangeRateVal, balanceCreditCnyPerUsdVal, legacyBalanceCreditUsdPerCnyVal] =
      await Promise.all([
        getSystemConfig('USD_EXCHANGE_RATE'),
        getSystemConfig('BALANCE_CREDIT_CNY_PER_USD'),
        getSystemConfig('BALANCE_CREDIT_USD_PER_CNY'),
      ]);

    // 刻意不用 resolveBalanceCreditCnyPerUsd：它在未配置时会兜底成 1，
    // 而 1 对公开定价页是有害的默认值——会把「¥ 价」渲染成和「$ 价」完全相同的数字，
    // 看上去像坏了。这里只在**确实配置过**时返回该汇率，否则返回 null，
    // 让落地页自然回退到 usdExchangeRate（默认 7.2）这个真正的汇率。
    const configuredCnyPerUsd =
      normalizeBalanceCreditCnyPerUsd(balanceCreditCnyPerUsdVal ?? env.BALANCE_CREDIT_CNY_PER_USD) ??
      legacyBalanceCreditUsdPerCnyToCnyPerUsd(
        legacyBalanceCreditUsdPerCnyVal ?? env.BALANCE_CREDIT_USD_PER_CNY,
      );

    return NextResponse.json({
      usdExchangeRate: normalizeUsdExchangeRate(usdExchangeRateVal) ?? DEFAULT_USD_EXCHANGE_RATE,
      balanceCreditCnyPerUsd: configuredCnyPerUsd,
    });
  } catch {
    // 落地页拿不到汇率时会回退到美元展示，这里不把内部错误暴露成 5xx。
    return NextResponse.json({ usdExchangeRate: null, balanceCreditCnyPerUsd: null });
  }
}
