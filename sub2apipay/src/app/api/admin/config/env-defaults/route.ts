import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { getEnv } from '@/lib/config';

const ALL_PROVIDERS = [
  { key: 'easypay', types: ['alipay', 'wxpay', 'usdt.plasma', 'usdt.polygon', 'usdc.solana'] },
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
        RECHARGE_MIN_AMOUNT: String(env.MIN_RECHARGE_AMOUNT),
        RECHARGE_MAX_AMOUNT: String(env.MAX_RECHARGE_AMOUNT),
        DAILY_RECHARGE_LIMIT: String(env.MAX_DAILY_RECHARGE_AMOUNT),
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
