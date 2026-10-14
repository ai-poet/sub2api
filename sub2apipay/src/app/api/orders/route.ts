import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { createOrder } from '@/lib/order/service';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { handleApiError } from '@/lib/utils/api';
import { getRechargeLimits, isWithinRechargeRange, rechargeRangeMessage } from '@/lib/order/recharge-limits';
import { getVisiblePaymentTypes } from '@/lib/payment/visibility';
import { resolveLocale } from '@/lib/locale';
import { resolveRequestOrigin } from '@/lib/request-origin';
import { readUserToken } from '@/lib/utils/request-token';

const createOrderSchema = z.object({
  // Optional here: the desktop client sends it as `Authorization: Bearer`.
  token: z.string().optional(),
  amount: z.number().positive().max(99999999.99),
  payment_type: z.string().min(1),
  src_host: z.string().max(253).optional(),
  src_url: z
    .string()
    .max(2048)
    .refine((url) => {
      try {
        const protocol = new URL(url).protocol;
        return protocol === 'http:' || protocol === 'https:';
      } catch {
        return false;
      }
    }, 'src_url must be a valid HTTP/HTTPS URL')
    .optional(),
  is_mobile: z.boolean().optional(),
  order_type: z.enum(['balance', 'subscription']).optional(),
  plan_id: z.string().optional(),
});

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const parsed = createOrderSchema.safeParse(body);
    const locale = resolveLocale(request.nextUrl.searchParams.get('lang') || request.headers.get('accept-language'));

    if (!parsed.success) {
      return NextResponse.json({ error: '参数错误', details: parsed.error.flatten().fieldErrors }, { status: 400 });
    }

    const { amount, payment_type, src_host, src_url, is_mobile, order_type, plan_id } = parsed.data;
    const token = readUserToken(request, parsed.data.token);
    if (!token) {
      return NextResponse.json({ error: '无效的 token，请重新登录', code: 'INVALID_TOKEN' }, { status: 401 });
    }

    // 通过 token 解析用户身份
    let userId: number;
    try {
      const user = await getCurrentUserByToken(token);
      userId = user.id;
    } catch {
      return NextResponse.json({ error: '无效的 token，请重新登录', code: 'INVALID_TOKEN' }, { status: 401 });
    }

    // 订阅订单跳过金额范围校验（价格由服务端套餐决定）
    if (order_type !== 'subscription') {
      // 后台配置 → 环境变量；没设置的一侧不限制
      const rechargeLimits = await getRechargeLimits();
      if (!isWithinRechargeRange(amount, rechargeLimits)) {
        return NextResponse.json({ error: rechargeRangeMessage(locale, rechargeLimits) }, { status: 400 });
      }
    }

    const visiblePaymentTypes = await getVisiblePaymentTypes();
    if (!visiblePaymentTypes.includes(payment_type)) {
      return NextResponse.json({ error: `不支持的支付方式: ${payment_type}` }, { status: 400 });
    }

    const clientIp =
      request.headers.get('x-forwarded-for')?.split(',')[0]?.trim() || request.headers.get('x-real-ip') || '127.0.0.1';

    const result = await createOrder({
      userId,
      amount,
      paymentType: payment_type,
      appUrl: resolveRequestOrigin(request),
      clientIp,
      isMobile: is_mobile,
      srcHost: src_host,
      srcUrl: src_url,
      locale,
      orderType: order_type,
      planId: plan_id,
    });

    // 不向客户端暴露 userName / userBalance 等隐私字段
    const { userName: _u, userBalance: _b, ...safeResult } = result;
    return NextResponse.json(safeResult);
  } catch (error) {
    return handleApiError(error, '创建订单失败，请稍后重试');
  }
}
