import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { requestInvoice } from '@/lib/invoice/service';
import { invoiceInvalidParamsMessage, invoiceMessage } from '@/lib/invoice/types';
import { invoiceTitleFieldsSchema } from '@/lib/invoice/request-schema';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

const invoiceRequestSchema = invoiceTitleFieldsSchema;

export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json({ error: locale === 'en' ? 'Missing token' : '缺少 token 参数' }, { status: 401 });
  }

  try {
    const user = await getCurrentUserByToken(token);
    // body 读不出来时给独立的错误码，不与字段校验失败混用同一句「参数错误」。
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
    const parsed = invoiceRequestSchema.safeParse(body);
    if (!parsed.success) {
      const fieldErrors = parsed.error.flatten().fieldErrors;
      return NextResponse.json(
        {
          error: invoiceInvalidParamsMessage(locale, Object.keys(fieldErrors)),
          details: fieldErrors,
        },
        { status: 400 },
      );
    }

    const { id } = await params;
    const invoice = await requestInvoice({
      orderIds: [id],
      userId: user.id,
      titleName: parsed.data.title_name,
      taxNo: parsed.data.tax_no,
      remark: parsed.data.remark,
      contactEmail: parsed.data.contact_email,
      locale,
    });

    return NextResponse.json({ success: true, invoice });
  } catch (error) {
    return handleApiError(error, '开票申请失败', request);
  }
}
