import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest, NextResponse } from 'next/server';

const mockVerifyAdminToken = vi.fn();
const mockGetPaymentRiskOverview = vi.fn();

vi.mock('@/lib/admin-auth', () => ({
  verifyAdminToken: (...args: unknown[]) => mockVerifyAdminToken(...args),
  unauthorizedResponse: () => NextResponse.json({ error: '未授权' }, { status: 401 }),
}));

// 真实的 overview 模块会 import 数据库客户端，这里给一个空壳，只用它的参数解析
vi.mock('@/lib/db', () => ({
  prisma: {},
  getConfiguredDatabaseSchema: () => 'public',
}));

vi.mock('@/lib/payment-risk/overview', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/payment-risk/overview')>()),
  getPaymentRiskOverview: (...args: unknown[]) => mockGetPaymentRiskOverview(...args),
}));

vi.mock('@/lib/utils/api', () => ({
  handleApiError: (error: Error, msg: string) =>
    NextResponse.json({ error: msg, detail: error.message }, { status: 500 }),
}));

import { GET } from '@/app/api/admin/risk/route';

function riskRequest(query = '') {
  return new NextRequest(`https://pay.example.com/api/admin/risk${query}`, {
    headers: { Authorization: 'Bearer test-admin-token' },
  });
}

describe('GET /api/admin/risk', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockVerifyAdminToken.mockResolvedValue(true);
    mockGetPaymentRiskOverview.mockResolvedValue({ summary: { paidWithoutTradeNo: 0 } });
  });

  it('未登录管理员：401，不查数据', async () => {
    mockVerifyAdminToken.mockResolvedValue(false);

    const res = await GET(riskRequest());

    expect(res.status).toBe(401);
    expect(mockGetPaymentRiskOverview).not.toHaveBeenCalled();
  });

  it('按查询参数取数并原样返回', async () => {
    const res = await GET(riskRequest('?days=7&action=PAYMENT_NOTIFY_BLOCKED&page=2&page_size=50'));

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ summary: { paidWithoutTradeNo: 0 } });
    expect(mockGetPaymentRiskOverview).toHaveBeenCalledWith({
      days: 7,
      action: 'PAYMENT_NOTIFY_BLOCKED',
      page: 2,
      pageSize: 50,
    });
  });

  it('查询出错：500', async () => {
    mockGetPaymentRiskOverview.mockRejectedValue(new Error('db down'));

    const res = await GET(riskRequest());

    expect(res.status).toBe(500);
    expect((await res.json()).error).toBe('加载支付风控数据失败');
  });
});
