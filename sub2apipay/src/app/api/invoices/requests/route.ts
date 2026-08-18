import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { INVOICE_MAX_MERGED_ORDERS, requestInvoice } from '@/lib/invoice/service';
import { invoiceInvalidParamsMessage } from '@/lib/invoice/types';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

/**
 * 合并开票入口：一次提交多张订单，开出一张发票。
 *
 * 单订单仍可走 /api/orders/[id]/invoice-request（同一个 service，行为一致），
 * 这里只是把「选哪些单」交给调用方。
 */
const mergedInvoiceRequestSchema = z.object({
  order_ids: z.array(z.string().trim().min(1)).min(1).max(INVOICE_MAX_MERGED_ORDERS),
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

export async function POST(request: NextRequest) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json({ error: locale === 'en' ? 'Missing token' : '缺少 token 参数' }, { status: 401 });
  }

  try {
    const user = await getCurrentUserByToken(token);
    const body = await request.json().catch(() => ({}));
    const parsed = mergedInvoiceRequestSchema.safeParse(body);
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

    const invoice = await requestInvoice({
      orderIds: parsed.data.order_ids,
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
