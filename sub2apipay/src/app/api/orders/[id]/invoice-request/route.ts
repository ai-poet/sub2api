import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { requestInvoice } from '@/lib/invoice/service';
import { invoiceInvalidParamsMessage, invoiceMessage } from '@/lib/invoice/types';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

const invoiceRequestSchema = z.object({
  title_name: z.string().trim().min(2).max(100),
  // 统一社会信用代码 18 位；兼容旧的 15 位纳税人识别号及部分 20 位号段。
  tax_no: z
    .string()
    .trim()
    .regex(/^[0-9A-Za-z]{15,20}$/),
  remark: z.string().trim().max(200).optional(),
  // 选填字段对空串宽容：调用方漏掉「空则省略」的逻辑不该变成 400。
  contact_email: z
    .union([z.literal(''), z.email().max(200)])
    .optional()
    .transform((value) => (value === '' ? undefined : value)),
});

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
