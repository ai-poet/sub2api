import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { getUserInvoiceDownloadUrl } from '@/lib/invoice/service';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

export async function GET(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json({ error: locale === 'en' ? 'Missing token' : '缺少 token 参数' }, { status: 401 });
  }

  try {
    const user = await getCurrentUserByToken(token);
    const { id } = await params;
    // 归属校验在 service 层写进查询条件；这里拿到 URL 即表示已通过。
    const url = await getUserInvoiceDownloadUrl({ invoiceId: id, userId: user.id, locale });

    // 预签名链接是短期凭证，任何一层缓存都不应留存。
    return NextResponse.redirect(url, {
      status: 302,
      headers: { 'Cache-Control': 'no-store, private' },
    });
  } catch (error) {
    return handleApiError(error, '获取发票下载链接失败', request);
  }
}
