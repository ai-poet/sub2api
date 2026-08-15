import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminGetInvoiceDownloadUrl } from '@/lib/invoice/service';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

/** 管理员预览已上传的发票文件（无归属限制，仅凭管理员令牌）。 */
export async function GET(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const { id } = await params;
    const url = await adminGetInvoiceDownloadUrl({ invoiceId: id, locale });
    return NextResponse.redirect(url, { status: 302, headers: { 'Cache-Control': 'no-store, private' } });
  } catch (error) {
    return handleApiError(error, '获取发票下载链接失败', request);
  }
}
