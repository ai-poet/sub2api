import { NextRequest, NextResponse } from 'next/server';
import { verifyAdminToken, unauthorizedResponse } from '@/lib/admin-auth';
import { getPaymentRiskOverview, parsePaymentRiskQuery } from '@/lib/payment-risk/overview';
import { handleApiError } from '@/lib/utils/api';

/**
 * 支付风控面板数据：可疑回调、复核不符、无交易号入账、重复交易号与可疑用户。只读。
 *
 * 查询参数：days（1-365，默认 30）、action（只看某一类风险事件）、page、page_size。
 */
export async function GET(request: NextRequest) {
  if (!(await verifyAdminToken(request))) return unauthorizedResponse(request);

  try {
    const overview = await getPaymentRiskOverview(parsePaymentRiskQuery(request.nextUrl.searchParams));
    return NextResponse.json(overview);
  } catch (error) {
    return handleApiError(error, '加载支付风控数据失败', request);
  }
}
