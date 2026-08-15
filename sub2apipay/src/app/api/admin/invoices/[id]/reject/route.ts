import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminRejectInvoice } from '@/lib/invoice/service';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

const rejectSchema = z.object({ reason: z.string().trim().min(1).max(200) });

export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const body = await request.json().catch(() => ({}));
    const parsed = rejectSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json({ error: locale === 'en' ? 'A reason is required' : '请填写驳回原因' }, { status: 400 });
    }

    const { id } = await params;
    const invoice = await adminRejectInvoice({
      invoiceId: id,
      reason: parsed.data.reason,
      operator: 'admin',
      locale,
    });
    return NextResponse.json({ success: true, invoice });
  } catch (error) {
    return handleApiError(error, '驳回开票申请失败', request);
  }
}
