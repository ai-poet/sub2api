import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { getAllGroups } from '@/lib/sub2api/client';

export async function GET(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  try {
    const groups = await getAllGroups();
    return NextResponse.json({ groups });
  } catch (error) {
    console.error('Failed to fetch main-system groups:', error);
    return NextResponse.json({ error: '获取主系统分组列表失败' }, { status: 500 });
  }
}
