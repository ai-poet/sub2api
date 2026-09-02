import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { prisma } from '@/lib/db';
import { ORDER_STATUS } from '@/lib/constants';
import { resolveLocale } from '@/lib/locale';
import { getPromotionUsageStats } from '@/lib/promotion/service';
import {
  promotionCreateSchema,
  promotionPatchSchema,
  recordToInput,
  serializePromotion,
  toPromotionData,
  validationErrorBody,
} from '@/lib/promotion/admin';

export async function PUT(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const { id } = await params;
    const existing = await prisma.rechargePromotion.findUnique({ where: { id } });
    if (!existing) {
      return NextResponse.json({ error: locale === 'en' ? 'Promotion not found' : '充值活动不存在' }, { status: 404 });
    }

    const body = await request.json().catch(() => ({}));
    const patch = promotionPatchSchema.safeParse(body);
    if (!patch.success) {
      return NextResponse.json(validationErrorBody(locale, patch.error), { status: 400 });
    }

    // 补丁与现有记录合并后整体校验，保证 percent/时间窗等跨字段规则始终成立
    const merged = promotionCreateSchema.safeParse({ ...recordToInput(existing), ...patch.data });
    if (!merged.success) {
      return NextResponse.json(validationErrorBody(locale, merged.error), { status: 400 });
    }

    const promotion = await prisma.rechargePromotion.update({
      where: { id },
      data: toPromotionData(merged.data),
    });
    const stats = await getPromotionUsageStats([id]);
    return NextResponse.json(serializePromotion(promotion, stats.get(id)));
  } catch (error) {
    console.error('Failed to update recharge promotion:', error);
    return NextResponse.json(
      { error: locale === 'en' ? 'Failed to update promotion' : '更新充值活动失败' },
      { status: 500 },
    );
  }
}

export async function DELETE(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const { id } = await params;
    const existing = await prisma.rechargePromotion.findUnique({ where: { id } });
    if (!existing) {
      return NextResponse.json({ error: locale === 'en' ? 'Promotion not found' : '充值活动不存在' }, { status: 404 });
    }

    // 进行中的订单仍依赖活动记录（履约/展示），先等它们结束或取消
    const activeOrderCount = await prisma.order.count({
      where: {
        promotionId: id,
        status: { in: [ORDER_STATUS.PENDING, ORDER_STATUS.PAID, ORDER_STATUS.RECHARGING] },
      },
    });
    if (activeOrderCount > 0) {
      return NextResponse.json(
        {
          error:
            locale === 'en'
              ? `${activeOrderCount} active order(s) still reference this promotion. Disable it instead.`
              : `该活动仍有 ${activeOrderCount} 个进行中的订单，无法删除，请先停用`,
        },
        { status: 409 },
      );
    }

    await prisma.rechargePromotion.delete({ where: { id } });
    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Failed to delete recharge promotion:', error);
    return NextResponse.json(
      { error: locale === 'en' ? 'Failed to delete promotion' : '删除充值活动失败' },
      { status: 500 },
    );
  }
}
