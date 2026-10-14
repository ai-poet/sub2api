import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { getEnv } from '@/lib/config';
import { DEFAULT_BALANCE_CREDIT_CNY_PER_USD, DEFAULT_USD_EXCHANGE_RATE, resolveBalanceCreditCnyPerUsd } from '@/lib/currency';

const ALL_PROVIDERS = [
  { key: 'easypay', types: ['alipay', 'wxpay', 'usdt.plasma', 'usdt.polygon', 'usdc.solana', 'bank', 'paynow'] },
  { key: 'alipay', types: ['alipay_direct'] },
  { key: 'wxpay', types: ['wxpay_direct'] },
  { key: 'stripe', types: ['stripe'] },
];

export async function GET(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  try {
    const env = getEnv();

    return NextResponse.json({
      availablePaymentTypes: ALL_PROVIDERS.flatMap((provider) => provider.types),
      providers: ALL_PROVIDERS.map((provider) => ({
        key: provider.key,
        configured: false,
        types: provider.types,
      })),
      instanceDefaults: {},
      defaults: {
        ENABLED_PAYMENT_TYPES: '',
        // 环境变量没设置时为空，表示不限
        RECHARGE_MIN_AMOUNT: env.MIN_RECHARGE_AMOUNT !== undefined ? String(env.MIN_RECHARGE_AMOUNT) : '',
        RECHARGE_MAX_AMOUNT: env.MAX_RECHARGE_AMOUNT !== undefined ? String(env.MAX_RECHARGE_AMOUNT) : '',
        DAILY_RECHARGE_LIMIT: env.MAX_DAILY_RECHARGE_AMOUNT !== undefined ? String(env.MAX_DAILY_RECHARGE_AMOUNT) : '',
        USD_EXCHANGE_RATE: String(DEFAULT_USD_EXCHANGE_RATE),
        BALANCE_CREDIT_CNY_PER_USD: String(
          resolveBalanceCreditCnyPerUsd(env.BALANCE_CREDIT_CNY_PER_USD, env.BALANCE_CREDIT_USD_PER_CNY) ??
            DEFAULT_BALANCE_CREDIT_CNY_PER_USD,
        ),
        ORDER_TIMEOUT_MINUTES: String(env.ORDER_TIMEOUT_MINUTES),
        MAX_PENDING_ORDERS: '3',
        LOAD_BALANCE_STRATEGY: 'round-robin',
      },
    });
  } catch (error) {
    console.error('Failed to get env defaults:', error instanceof Error ? error.message : String(error));
    return NextResponse.json({ error: 'Failed to get env defaults' }, { status: 500 });
  }
}
