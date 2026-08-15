import { NextRequest, NextResponse } from 'next/server';
import { InvoiceStatus } from '@prisma/client';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { adminListInvoices } from '@/lib/invoice/service';
import { handleApiError } from '@/lib/utils/api';

function parseDate(raw: string | null): Date | undefined {
  if (!raw) return undefined;
  const parsed = new Date(raw);
  return Number.isNaN(parsed.getTime()) ? undefined : parsed;
}

export async function GET(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  const searchParams = request.nextUrl.searchParams;
  const page = Math.max(1, Number(searchParams.get('page') || '1'));
  const pageSize = Math.min(100, Math.max(1, Number(searchParams.get('page_size') || '20')));

  const status = searchParams.get('status');
  const userId = Number(searchParams.get('user_id'));
  const keyword = searchParams.get('q')?.trim();

  try {
    const { invoices, total, statusCounts } = await adminListInvoices(
      {
        status: status && status in InvoiceStatus ? (status as InvoiceStatus) : undefined,
        userId: Number.isFinite(userId) && userId > 0 ? userId : undefined,
        dateFrom: parseDate(searchParams.get('date_from')),
        dateTo: parseDate(searchParams.get('date_to')),
        keyword: keyword || undefined,
      },
      page,
      pageSize,
    );

    return NextResponse.json({
      invoices,
      status_counts: statusCounts,
      total,
      page,
      page_size: pageSize,
      total_pages: Math.ceil(total / pageSize),
    });
  } catch (error) {
    return handleApiError(error, '获取发票列表失败', request);
  }
}
