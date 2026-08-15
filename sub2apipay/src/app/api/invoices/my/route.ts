import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { getInvoiceSettings, listMyInvoices, listSavedTitles } from '@/lib/invoice/service';
import { handleApiError } from '@/lib/utils/api';
import { resolveLocale } from '@/lib/locale';

const VALID_PAGE_SIZES = [20, 50, 100];

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams;
  const locale = resolveLocale(searchParams.get('lang'));
  const token = searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json({ error: locale === 'en' ? 'Missing token' : '缺少 token 参数' }, { status: 401 });
  }

  const page = Math.max(1, Number(searchParams.get('page') || '1'));
  const rawPageSize = Number(searchParams.get('page_size') || '20');
  const pageSize = VALID_PAGE_SIZES.includes(rawPageSize) ? rawPageSize : 20;

  let user;
  try {
    user = await getCurrentUserByToken(token);
  } catch {
    return NextResponse.json({ error: 'unauthorized' }, { status: 401 });
  }

  try {
    const [settings, { invoices, total }, titles] = await Promise.all([
      getInvoiceSettings(),
      listMyInvoices(user.id, page, pageSize),
      listSavedTitles(user.id),
    ]);

    return NextResponse.json({
      enabled: settings.enabled,
      invoices,
      // 抬头记忆：前端用最近一次自动回填，其余作为可选项展示。
      titles,
      page,
      page_size: pageSize,
      total,
      total_pages: Math.ceil(total / pageSize),
    });
  } catch (error) {
    return handleApiError(error, '获取发票列表失败', request);
  }
}
