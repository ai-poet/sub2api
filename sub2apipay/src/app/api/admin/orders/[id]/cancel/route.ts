import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { resolveLocale } from '@/lib/locale';
import { adminCancelOrder } from '@/lib/order/service';
import { handleApiError } from '@/lib/utils/api';

export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const { id } = await params;
    const outcome = await adminCancelOrder(id, locale);
    if (outcome === 'already_paid') {
      return NextResponse.json({
        success: true,
        status: 'PAID',
        message: locale === 'en' ? 'Order has already been paid' : '订单已支付完成',
      });
    }
    if (outcome === 'paid_unconfirmed') {
      // 平台已收款但本地没能入账：订单原样保留，明确告诉管理员而不是假装"已支付完成"
      return NextResponse.json(
        {
          success: false,
          code: 'PAYMENT_CONFIRM_FAILED',
          error:
            locale === 'en'
              ? 'The payment platform reports this order as paid, but crediting did not complete. The order was left unchanged; check its audit log and handle it manually.'
              : '支付平台显示该订单已付款，但入账未完成；订单未取消，请查看该订单的审计日志后人工处理。',
        },
        { status: 409 },
      );
    }
    return NextResponse.json({ success: true });
  } catch (error) {
    return handleApiError(error, locale === 'en' ? 'Cancel order failed' : '取消订单失败', request);
  }
}
