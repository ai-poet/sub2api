import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminResendNotification } from '@/lib/invoice/service';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const { id } = await params;
    const result = await adminResendNotification({ invoiceId: id, locale });
    return NextResponse.json({ success: true, sent: result.sent, warning: result.warning });
  } catch (error) {
    return handleApiError(error, '重新发送通知失败', request);
  }
}
