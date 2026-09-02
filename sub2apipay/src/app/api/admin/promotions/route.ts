import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { prisma } from '@/lib/db';
import { resolveLocale } from '@/lib/locale';
import { getPromotionUsageStats } from '@/lib/promotion/service';
import { promotionCreateSchema, serializePromotion, toPromotionData, validationErrorBody } from '@/lib/promotion/admin';

export async function GET(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const promotions = await prisma.rechargePromotion.findMany({
      orderBy: [{ sortOrder: 'asc' }, { createdAt: 'desc' }],
    });
    const stats = await getPromotionUsageStats(promotions.map((p) => p.id));
    const now = new Date();
    return NextResponse.json({
      promotions: promotions.map((p) => serializePromotion(p, stats.get(p.id), now)),
    });
  } catch (error) {
    console.error('Failed to list recharge promotions:', error);
    return NextResponse.json(
      { error: locale === 'en' ? 'Failed to load promotions' : '获取充值活动列表失败' },
      { status: 500 },
    );
  }
}

export async function POST(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const body = await request.json().catch(() => ({}));
    const parsed = promotionCreateSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json(validationErrorBody(locale, parsed.error), { status: 400 });
    }

    const promotion = await prisma.rechargePromotion.create({ data: toPromotionData(parsed.data) });
    return NextResponse.json(serializePromotion(promotion), { status: 201 });
  } catch (error) {
    console.error('Failed to create recharge promotion:', error);
    return NextResponse.json(
      { error: locale === 'en' ? 'Failed to create promotion' : '创建充值活动失败' },
      { status: 500 },
    );
  }
}
