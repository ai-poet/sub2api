import { NextRequest, NextResponse } from 'next/server';
import { prisma } from '@/lib/db';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminGetInvoice } from '@/lib/invoice/service';
import { handleApiError } from '@/lib/utils/api';
import { resolveLocale } from '@/lib/locale';

export async function GET(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  try {
    const { id } = await params;
    const invoice = await adminGetInvoice(id);
    if (!invoice) {
      return NextResponse.json({ error: locale === 'en' ? 'Invoice not found' : '发票申请不存在' }, { status: 404 });
    }

    // 审计日志挂在订单上，因此开票历史与退款历史在同一条时间线里。
    const auditLogs = await prisma.auditLog.findMany({
      where: { orderId: invoice.orderId },
      orderBy: { createdAt: 'desc' },
      take: 50,
    });

    return NextResponse.json({ invoice, audit_logs: auditLogs });
  } catch (error) {
    return handleApiError(error, '获取发票详情失败', request);
  }
}
