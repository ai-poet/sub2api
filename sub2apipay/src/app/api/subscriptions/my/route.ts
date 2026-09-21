import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUserByToken, getUserSubscriptions, getAllGroups } from '@/lib/sub2api/client';

export async function GET(request: NextRequest) {
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json({ error: '缺少 token' }, { status: 401 });
  }

  let userId: number;
  try {
    const user = await getCurrentUserByToken(token);
    userId = user.id;
  } catch {
    return NextResponse.json({ error: '无效的 token' }, { status: 401 });
  }

  try {
    const [subscriptions, groups] = await Promise.all([getUserSubscriptions(userId), getAllGroups().catch(() => [])]);

    const groupMap = new Map(groups.map((g) => [g.id, g]));

    // 三个额度上限随订阅一起下发：卡片要显示「已用 / 上限」，只给用量会让用户把上限当成用量、
    // 或以为额度没生效。优先用分组列表里的分组（管理端可能刚改过限额），拿不到时退回订阅自带的 group。
    const enriched = subscriptions.map((sub) => {
      const group = groupMap.get(sub.group_id) ?? sub.group ?? null;
      return {
        ...sub,
        group_name: group?.name ?? null,
        platform: group?.platform ?? null,
        daily_limit_usd: group?.daily_limit_usd ?? null,
        weekly_limit_usd: group?.weekly_limit_usd ?? null,
        monthly_limit_usd: group?.monthly_limit_usd ?? null,
      };
    });

    return NextResponse.json({ subscriptions: enriched });
  } catch (error) {
    console.error('Failed to get user subscriptions:', error);
    return NextResponse.json({ error: '获取订阅信息失败' }, { status: 500 });
  }
}
