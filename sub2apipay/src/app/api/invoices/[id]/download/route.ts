import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { openUserInvoiceFile } from '@/lib/invoice/service';
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
    // 归属校验在 service 层写进查询条件；这里拿到流即表示已通过。
    const file = await openUserInvoiceFile({ invoiceId: id, userId: user.id, locale });

    // 同源回传而不是 302 到对象存储：下载页在 iframe 里，跳外部源会被父页面 CSP 的
    // frame-src 拦下，HTTPS 页面跳 http:// 还会再撞一次混合内容。
    const headers = new Headers({
      'Content-Type': file.contentType,
      'Content-Disposition': file.contentDisposition,
      'Cache-Control': 'no-store, private',
      'X-Content-Type-Options': 'nosniff',
    });
    if (file.contentLength) headers.set('Content-Length', file.contentLength);

    return new NextResponse(file.body, { status: 200, headers });
  } catch (error) {
    return handleApiError(error, '发票下载失败', request);
  }
}
