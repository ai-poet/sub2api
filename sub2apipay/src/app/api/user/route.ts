import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { getEnv } from '@/lib/config';
import { queryMethodLimits } from '@/lib/order/limits';
import { paymentRegistry } from '@/lib/payment';
import { getPaymentDisplayInfo } from '@/lib/pay-utils';
import { resolveLocale } from '@/lib/locale';
import { getSystemConfig } from '@/lib/system-config';
import { getVisiblePaymentTypes } from '@/lib/payment/visibility';
import {
  DEFAULT_BALANCE_CREDIT_CNY_PER_USD,
  DEFAULT_USD_EXCHANGE_RATE,
  resolveBalanceCreditCnyPerUsd,
  normalizeUsdExchangeRate,
} from '@/lib/currency';

export async function GET(request: NextRequest) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang') || request.headers.get('accept-language'));
  const userId = Number(request.nextUrl.searchParams.get('user_id'));
  if (!userId || isNaN(userId) || userId <= 0) {
    return NextResponse.json({ error: locale === 'en' ? 'Invalid user ID' : '无效的用户 ID' }, { status: 400 });
  }

  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json(
      { error: locale === 'en' ? 'Missing token parameter' : '缺少 token 参数' },
      { status: 401 },
    );
  }

  try {
    // 验证 token 并确保请求的 user_id 与 token 对应的用户匹配
    let tokenUser;
    try {
      tokenUser = await getCurrentUserByToken(token);
    } catch {
      return NextResponse.json({ error: locale === 'en' ? 'Invalid token' : '无效的 token' }, { status: 401 });
    }

    if (tokenUser.id !== userId) {
      return NextResponse.json(
        { error: locale === 'en' ? 'Forbidden to access this user' : '无权访问该用户信息' },
        { status: 403 },
      );
    }

    const env = getEnv();

    const configPromise = Promise.all([
      getVisiblePaymentTypes(locale),
      getSystemConfig('BALANCE_PAYMENT_DISABLED'),
      getSystemConfig('MAX_PENDING_ORDERS'),
      getSystemConfig('RECHARGE_MIN_AMOUNT'),
      getSystemConfig('RECHARGE_MAX_AMOUNT'),
      getSystemConfig('DAILY_RECHARGE_LIMIT'),
      getSystemConfig('USD_EXCHANGE_RATE'),
      getSystemConfig('BALANCE_CREDIT_CNY_PER_USD'),
      getSystemConfig('BALANCE_CREDIT_USD_PER_CNY'),
    ]).then(
      async ([
        visibleTypes,
        balanceDisabledVal,
        maxPendingVal,
        minAmountVal,
        maxAmountVal,
        dailyLimitVal,
        usdExchangeRateVal,
        balanceCreditCnyPerUsdVal,
        legacyBalanceCreditUsdPerCnyVal,
      ]) => {
        const methodLimits = await queryMethodLimits(visibleTypes);
        return {
          enabledTypes: visibleTypes,
          methodLimits,
          balanceDisabled: balanceDisabledVal === 'true',
          maxPendingOrders: maxPendingVal ? parseInt(maxPendingVal, 10) || 3 : 3,
          minAmount: minAmountVal ? parseFloat(minAmountVal) || env.MIN_RECHARGE_AMOUNT : env.MIN_RECHARGE_AMOUNT,
          maxAmount: maxAmountVal ? parseFloat(maxAmountVal) || env.MAX_RECHARGE_AMOUNT : env.MAX_RECHARGE_AMOUNT,
          maxDailyAmount: dailyLimitVal ? parseFloat(dailyLimitVal) : env.MAX_DAILY_RECHARGE_AMOUNT,
          usdExchangeRate: normalizeUsdExchangeRate(usdExchangeRateVal) ?? DEFAULT_USD_EXCHANGE_RATE,
          balanceCreditCnyPerUsd:
            resolveBalanceCreditCnyPerUsd(
              balanceCreditCnyPerUsdVal ?? env.BALANCE_CREDIT_CNY_PER_USD,
              legacyBalanceCreditUsdPerCnyVal ?? env.BALANCE_CREDIT_USD_PER_CNY,
            ) ?? DEFAULT_BALANCE_CREDIT_CNY_PER_USD,
        };
      },
    );

    const {
      enabledTypes,
      methodLimits,
      balanceDisabled,
      maxPendingOrders,
      minAmount,
      maxAmount,
      maxDailyAmount,
      usdExchangeRate,
      balanceCreditCnyPerUsd,
    } = await configPromise;

    // 收集 sublabel 覆盖
    const sublabelOverrides: Record<string, string> = {};

    // 1. 检测同 label 冲突：多个启用渠道有相同的显示名，自动标记默认 sublabel（provider 名）
    const labelCount = new Map<string, string[]>();
    for (const type of enabledTypes) {
      const { channel } = getPaymentDisplayInfo(type, locale);
      const types = labelCount.get(channel) || [];
      types.push(type);
      labelCount.set(channel, types);
    }
    for (const [, types] of labelCount) {
      if (types.length > 1) {
        for (const type of types) {
          const { provider, sublabel } = getPaymentDisplayInfo(type, locale);
          const preferredSublabel = sublabel || provider;
          if (preferredSublabel) sublabelOverrides[type] = preferredSublabel;
        }
      }
    }

    // 2. 用户手动配置的 PAYMENT_SUBLABEL_* 优先级最高，覆盖自动生成的
    if (env.PAYMENT_SUBLABEL_ALIPAY) sublabelOverrides.alipay = env.PAYMENT_SUBLABEL_ALIPAY;
    if (env.PAYMENT_SUBLABEL_ALIPAY_DIRECT) sublabelOverrides.alipay_direct = env.PAYMENT_SUBLABEL_ALIPAY_DIRECT;
    if (env.PAYMENT_SUBLABEL_WXPAY) sublabelOverrides.wxpay = env.PAYMENT_SUBLABEL_WXPAY;
    if (env.PAYMENT_SUBLABEL_WXPAY_DIRECT) sublabelOverrides.wxpay_direct = env.PAYMENT_SUBLABEL_WXPAY_DIRECT;
    if (env.PAYMENT_SUBLABEL_STRIPE) sublabelOverrides.stripe = env.PAYMENT_SUBLABEL_STRIPE;

    return NextResponse.json({
      user: {
        id: tokenUser.id,
        status: tokenUser.status,
      },
      config: {
        enabledPaymentTypes: enabledTypes,
        minAmount,
        maxAmount,
        maxDailyAmount,
        usdExchangeRate,
        balanceCreditCnyPerUsd,
        methodLimits,
        helpImageUrl: env.PAY_HELP_IMAGE_URL ?? null,
        helpText: env.PAY_HELP_TEXT ?? null,
        stripePublishableKey: (() => {
          if (!enabledTypes.includes('stripe')) return null;
          try {
            const sp = paymentRegistry.getProvider('stripe' as import('@/lib/payment').PaymentType);
            const pk =
              'getPublishableKey' in sp ? (sp as { getPublishableKey(): string | undefined }).getPublishableKey() : undefined;
            if (pk) return pk;
          } catch { /* not registered */ }
          return env.STRIPE_PUBLISHABLE_KEY || null;
        })(),
        balanceDisabled,
        maxPendingOrders,
        sublabelOverrides: Object.keys(sublabelOverrides).length > 0 ? sublabelOverrides : null,
      },
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    if (message === 'USER_NOT_FOUND') {
      return NextResponse.json({ error: locale === 'en' ? 'User not found' : '用户不存在' }, { status: 404 });
    }
    console.error('Get user error:', error);
    return NextResponse.json(
      { error: locale === 'en' ? 'Failed to fetch user info' : '获取用户信息失败' },
      { status: 500 },
    );
  }
}
