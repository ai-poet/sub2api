import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { requestInvoice } from '@/lib/invoice/service';
import { invoiceInvalidParamsMessage, invoiceMessage } from '@/lib/invoice/types';
import { invoiceTitleFieldsSchema } from '@/lib/invoice/request-schema';
import { resolveLocale } from '@/lib/locale';
import { handleApiError } from '@/lib/utils/api';

/**
 * 合并开票入口：一次提交多张订单，开出一张发票。
 *
 * 单订单仍可走 /api/orders/[id]/invoice-request（同一个 service，行为一致），
 * 这里只是把「选哪些单」交给调用方。
 */
const mergedInvoiceRequestSchema = invoiceTitleFieldsSchema.extend({
  // 这里只做形状校验，上限（INVOICE_MAX_MERGED_ORDERS）由 service 检查——
  // 那边的报错是「单次最多合并 N 张订单」，比 zod 的「订单格式不正确」能让用户
  // 知道该怎么办。500 只是防滥用的荷载兜底。
  order_ids: z.array(z.string().trim().min(1)).min(1).max(500),
});

export async function POST(request: NextRequest) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json({ error: locale === 'en' ? 'Missing token' : '缺少 token 参数' }, { status: 401 });
  }

  try {
    const user = await getCurrentUserByToken(token);
    // body 读不出来时给独立的错误码，不与字段校验失败混用同一句「参数错误」——
    // 否则请求体在代理链路上被吞时，用户会以为是自己填错了什么。
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
