import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { previewQuickInvoice, requestQuickInvoice } from '@/lib/invoice/service';
import { invoiceInvalidParamsMessage, invoiceMessage } from '@/lib/invoice/types';
import { invoiceTitleFieldsSchema } from '@/lib/invoice/request-schema';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

function missingToken(locale: ReturnType<typeof resolveLocale>) {
  return NextResponse.json({ error: locale === 'en' ? 'Missing token' : '缺少 token 参数' }, { status: 401 });
}

/**
 * GET：一键开票预览。订单由服务端挑选（跨全部分页），前端只展示金额与单数。
 */
export async function GET(request: NextRequest) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) return missingToken(locale);

  try {
    const user = await getCurrentUserByToken(token);
    const preview = await previewQuickInvoice(user.id);
    return NextResponse.json({
      enabled: preview.enabled,
      count: preview.count,
      amount: preview.amount,
      eligible_count: preview.eligibleCount,
      capped: preview.capped,
      min_amount: preview.minAmount,
    });
  } catch (error) {
    return handleApiError(error, '获取可开票信息失败', request);
  }
}

/**
 * POST：一键开票。请求体只带抬头信息，覆盖哪些订单由服务端按与预览一致的规则挑选。
 */
export async function POST(request: NextRequest) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) return missingToken(locale);

  try {
    const user = await getCurrentUserByToken(token);
    let body: unknown;
    try {
      body = await request.json();
    } catch {
      return NextResponse.json(
        {
          error: invoiceMessage(locale, '请求体读取失败，请刷新页面后重试', 'Failed to read request body, refresh and retry'),
          code: 'INVALID_JSON_BODY',
        },
        { status: 400 },
      );
    }
    const parsed = invoiceTitleFieldsSchema.safeParse(body);
    if (!parsed.success) {
      const fieldErrors = parsed.error.flatten().fieldErrors;
      return NextResponse.json(
        { error: invoiceInvalidParamsMessage(locale, Object.keys(fieldErrors)), details: fieldErrors },
        { status: 400 },
      );
    }

    const result = await requestQuickInvoice({
      userId: user.id,
      titleName: parsed.data.title_name,
      taxNo: parsed.data.tax_no,
      remark: parsed.data.remark,
      contactEmail: parsed.data.contact_email,
      locale,
    });

    return NextResponse.json({ success: true, invoice: result.invoice, count: result.count, amount: result.amount });
  } catch (error) {
    return handleApiError(error, '开票申请失败', request);
  }
}
