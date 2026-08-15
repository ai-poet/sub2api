import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminIssueInvoice, INVOICE_MAX_FILE_BYTES } from '@/lib/invoice/service';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

/**
 * 上传发票文件并将申请置为「已开具」，随后触发通知邮件。
 *
 * 上传与状态流转合并为一个动作：拆成两个接口会产生「有文件但仍待处理」的半状态。
 * 浏览器这一跳用 multipart（FormData 原生），到网关那一跳换成裸二进制。
 */
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

    const adminNote = form.get('admin_note');
    const { id } = await params;

    const result = await adminIssueInvoice({
      invoiceId: id,
      file: {
        fileName: file.name,
        contentType: file.type || 'application/octet-stream',
        data: await file.arrayBuffer(),
      },
      adminNote: typeof adminNote === 'string' ? adminNote : null,
      operator: 'admin',
      locale,
    });

    // 邮件失败不回滚已开具的发票，只把 warning 带回给管理员。
    return NextResponse.json({ success: true, invoice: result.invoice, warning: result.warning });
  } catch (error) {
    return handleApiError(error, '开票失败', request);
  }
}
