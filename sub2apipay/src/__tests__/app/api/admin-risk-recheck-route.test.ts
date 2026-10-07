import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest, NextResponse } from 'next/server';

const mockVerifyAdminToken = vi.fn();
const mockRecheckEasyPayOrders = vi.fn();

vi.mock('@/lib/admin-auth', () => ({
  verifyAdminToken: (...args: unknown[]) => mockVerifyAdminToken(...args),
  unauthorizedResponse: () => NextResponse.json({ error: '未授权' }, { status: 401 }),
}));

// 真实的 recheck 模块会 import 数据库客户端与支付实例，这里给空壳，只用它的参数解析
vi.mock('@/lib/db', () => ({
  prisma: {},
  getConfiguredDatabaseSchema: () => 'public',
}));
vi.mock('@/lib/payment/load-balancer', () => ({
  getInstanceConfig: vi.fn(),
}));

vi.mock('@/lib/payment-risk/recheck', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/payment-risk/recheck')>()),
  recheckEasyPayOrders: (...args: unknown[]) => mockRecheckEasyPayOrders(...args),
}));

vi.mock('@/lib/utils/api', () => ({
  handleApiError: (error: Error, msg: string) =>
    NextResponse.json({ error: msg, detail: error.message }, { status: 500 }),
}));

import { POST } from '@/app/api/admin/risk/recheck/route';

function recheckRequest(body: unknown) {
  return new NextRequest('https://pay.example.com/api/admin/risk/recheck', {
    method: 'POST',
    headers: { Authorization: 'Bearer test-admin-token', 'Content-Type': 'application/json' },
    body: typeof body === 'string' ? body : JSON.stringify(body),
  });
}

describe('POST /api/admin/risk/recheck', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockVerifyAdminToken.mockResolvedValue(true);
    mockRecheckEasyPayOrders.mockResolvedValue({ done: true, results: [] });
  });

  it('未登录管理员：401，不查单', async () => {
    mockVerifyAdminToken.mockResolvedValue(false);

    const res = await POST(recheckRequest({ days: 30 }));

    expect(res.status).toBe(401);
    expect(mockRecheckEasyPayOrders).not.toHaveBeenCalled();
  });

  it('参数不合法：400，不查单', async () => {
    const res = await POST(recheckRequest({ days: 9999 }));

    expect(res.status).toBe(400);
    expect(mockRecheckEasyPayOrders).not.toHaveBeenCalled();
  });

  it('请求体不是 JSON 时按默认参数执行', async () => {
    const res = await POST(recheckRequest('not json'));

    expect(res.status).toBe(200);
    expect(mockRecheckEasyPayOrders).toHaveBeenCalledWith(
      expect.objectContaining({ cursor: null, batchSize: 10, since: expect.any(Date) }),
    );
  });

  it('续跑：沿用 since 与 cursor，原样返回结果', async () => {
    const since = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString();

    const res = await POST(recheckRequest({ since, cursor: 'a10', batch_size: 5 }));

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ done: true, results: [] });
    expect(mockRecheckEasyPayOrders).toHaveBeenCalledWith({ since: new Date(since), cursor: 'a10', batchSize: 5 });
  });

  it('复核出错：500', async () => {
    mockRecheckEasyPayOrders.mockRejectedValue(new Error('db down'));

    const res = await POST(recheckRequest({ days: 30 }));

    expect(res.status).toBe(500);
    expect((await res.json()).error).toBe('复核历史订单失败');
  });
});
