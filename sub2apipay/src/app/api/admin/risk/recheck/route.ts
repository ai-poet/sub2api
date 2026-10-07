import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { resolveLocale } from '@/lib/locale';
import { parseRecheckRequest, recheckEasyPayOrders } from '@/lib/payment-risk/recheck';
import { handleApiError } from '@/lib/utils/api';

/**
 * 批量复核历史订单的一批：把窗口内由易支付回调入账的订单逐笔拿到平台上查单。
 *
 * 只读订单，对不上的记为"平台复核不符"。请求体：第一批 { days }，之后 { since, cursor }，
 * 可选 batch_size。返回 nextCursor 与 done，前端循环调用直到 done。
 */
export async function POST(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const body = await request.json().catch(() => ({}));
  const recheckRequest = parseRecheckRequest(body);
  if (!recheckRequest) {
    return NextResponse.json({ error: locale === 'en' ? 'Invalid parameters' : '参数错误' }, { status: 400 });
  }

  try {
    return NextResponse.json(await recheckEasyPayOrders(recheckRequest));
  } catch (error) {
    return handleApiError(error, '复核历史订单失败', request);
  }
}
