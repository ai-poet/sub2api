import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminReplaceInvoiceFile, INVOICE_MAX_FILE_BYTES } from '@/lib/invoice/service';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

/** 替换已开具发票的文件（如开错抬头重开）。状态保持 ISSUED，需另行重发通知。 */
export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const form = await request.formData();
    const file = form.get('file');
    if (!(file instanceof File)) {
      return NextResponse.json({ error: locale === 'en' ? 'Missing file' : '缺少发票文件' }, { status: 400 });
    }
    if (file.size > INVOICE_MAX_FILE_BYTES) {
      return NextResponse.json(
        { error: locale === 'en' ? 'Invoice file must not exceed 10MB' : '发票文件不能超过 10MB' },
        { status: 413 },
      );
    }

    const { id } = await params;
    const invoice = await adminReplaceInvoiceFile({
      invoiceId: id,
      file: {
        fileName: file.name,
        contentType: file.type || 'application/octet-stream',
        data: await file.arrayBuffer(),
      },
      operator: 'admin',
      locale,
    });
    return NextResponse.json({ success: true, invoice });
  } catch (error) {
    return handleApiError(error, '替换发票文件失败', request);
  }
}
