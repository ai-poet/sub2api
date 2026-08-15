import { NextRequest, NextResponse } from 'next/server';
import { z } from 'zod';
import { getCurrentUserByToken } from '@/lib/sub2api/client';
import { requestInvoice } from '@/lib/invoice/service';
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
  contact_email: z.email().max(200).optional(),
});

export async function POST(request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const locale = resolveLocale(request.nextUrl.searchParams.get('lang'));
  const token = request.nextUrl.searchParams.get('token')?.trim();
  if (!token) {
    return NextResponse.json({ error: locale === 'en' ? 'Missing token' : '缺少 token 参数' }, { status: 401 });
  }

  try {
    const user = await getCurrentUserByToken(token);
    const body = await request.json().catch(() => ({}));
    const parsed = invoiceRequestSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json(
        {
          error: locale === 'en' ? 'Invalid parameters' : '参数错误',
          details: parsed.error.flatten().fieldErrors,
        },
        { status: 400 },
      );
    }

    const { id } = await params;
    const invoice = await requestInvoice({
      orderId: id,
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
