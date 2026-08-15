import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminOpenInvoiceFile } from '@/lib/invoice/service';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

/** 管理员预览已上传的发票文件（无归属限制，仅凭管理员令牌）。 */
export async function GET(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const { id } = await params;
    const file = await adminOpenInvoiceFile({ invoiceId: id, locale });

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
