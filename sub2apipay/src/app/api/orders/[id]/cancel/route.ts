import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { cancelOrder } from '@/lib/order/service';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { handleApiError } from '@/lib/utils/api';

const cancelSchema = z.object({
  token: z.string().min(1),
});

export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const body = await request.json();
    const parsed = cancelSchema.safeParse(body);

    if (!parsed.success) {
      return NextResponse.json({ error: '缺少 token 参数' }, { status: 400 });
    }

    let userId: number;
    try {
      const user = await getCurrentUserByToken(parsed.data.token);
      userId = user.id;
    } catch {
      return NextResponse.json({ error: '登录态已失效，无法取消订单' }, { status: 401 });
    }

    const outcome = await cancelOrder(id, userId);
    if (outcome === 'already_paid') {
      return NextResponse.json({ success: true, status: 'PAID', message: '订单已支付完成' });
    }
    if (outcome === 'paid_unconfirmed') {
      // 平台已收款但本地没能入账：订单原样保留，提示用户联系管理员而不是让其重复下单
      return NextResponse.json(
        {
          success: false,
          code: 'PAYMENT_CONFIRM_FAILED',
          error: '支付平台显示该订单已付款，但入账尚未完成，订单未取消；请联系管理员处理，不要重复支付。',
        },
        { status: 409 },
      );
    }
    return NextResponse.json({ success: true });
  } catch (error) {
    return handleApiError(error, '取消订单失败');
  }
}
