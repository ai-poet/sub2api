import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/lib/db';
import { getCurrentUserByToken, getGroup, getGroupModels } from '@/lib/sub2api/client';
import { syncPayTranslationSourcesIfStale } from '@/lib/sub2api/content-translations';
import { readUserToken } from '@/lib/utils/request-token';

export async function GET(request: NextRequest) {
  const token = readUserToken(request);
  if (!token) {
    return NextResponse.json({ error: '缺少 token' }, { status: 401 });
  }

  try {
    await getCurrentUserByToken(token);
  } catch {
    return NextResponse.json({ error: '无效的 token' }, { status: 401 });
  }

  // 内容自动翻译：顺手把套餐 / 活动 / 渠道文案登记给后端（每 10 分钟最多一次，不等待）
  syncPayTranslationSourcesIfStale();

  try {
    const plans = await prisma.subscriptionPlan.findMany({
      where: { forSale: true },
      orderBy: { sortOrder: 'asc' },
    });

    // Several durations of one plan share a group. Load its catalog once.
    const modelCatalogs = new Map<number, ReturnType<typeof getGroupModels>>();
    const loadModels = (groupId: number) => {
      if (!modelCatalogs.has(groupId)) modelCatalogs.set(groupId, getGroupModels(groupId));
      return modelCatalogs.get(groupId)!;
    };
    // 并发校验每个套餐对应的 Sub2API 分组是否存在
    const results = await Promise.all(
      plans.map(async (plan) => {
        if (plan.groupId === null) return null;

        let groupActive = false;
        let group: Awaited<ReturnType<typeof getGroup>> = null;
        let groupInfo: {
          daily_limit_usd: number | null;
          weekly_limit_usd: number | null;
          monthly_limit_usd: number | null;
        } | null = null;
        try {
          group = await getGroup(plan.groupId);
          groupActive = group !== null && group.status === 'active';
          if (group) {
            groupInfo = {
              daily_limit_usd: group.daily_limit_usd,
              weekly_limit_usd: group.weekly_limit_usd,
              monthly_limit_usd: group.monthly_limit_usd,
            };
          }
        } catch {
          groupActive = false;
        }

        if (!groupActive) return null;

        return {
          id: plan.id,
          groupId: plan.groupId,
          groupName: group?.name ?? null,
          name: plan.name,
          description: plan.description,
          price: Number(plan.price),
          originalPrice: plan.originalPrice ? Number(plan.originalPrice) : null,
          validityDays: plan.validityDays,
          validityUnit: plan.validityUnit,
          features: plan.features ? JSON.parse(plan.features) : [],
          productName: plan.productName ?? null,
          platform: group?.platform ?? null,
          rateMultiplier: group?.rate_multiplier ?? null,
          limits: groupInfo,
          // Failure stays distinct from an empty catalog and never blocks checkout.
          models: await loadModels(plan.groupId).catch(() => null),
          allowMessagesDispatch: group?.allow_messages_dispatch ?? false,
          defaultMappedModel: group?.default_mapped_model ?? null,
          supportedModelScopes: group?.supported_model_scopes ?? [],
        };
      }),
    );

    return NextResponse.json({ plans: results.filter(Boolean) });
  } catch (error) {
    console.error('Failed to list subscription plans:', error);
    return NextResponse.json({ error: '获取订阅套餐失败' }, { status: 500 });
  }
}
