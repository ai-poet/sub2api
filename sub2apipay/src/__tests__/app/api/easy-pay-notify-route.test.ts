import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';

const mockHandlePaymentNotify = vi.fn();
const mockVerifyNotification = vi.fn();

vi.mock('@/lib/order/service', () => ({
  handlePaymentNotify: (...args: unknown[]) => mockHandlePaymentNotify(...args),
}));

vi.mock('@/lib/payment', () => ({
  ensureDBProviders: vi.fn().mockResolvedValue(undefined),
  paymentRegistry: {
    getProvider: () => ({
      name: 'easy-pay',
      providerKey: 'easypay',
      verifyNotification: (...args: unknown[]) => mockVerifyNotification(...args),
    }),
  },
}));

vi.mock('@/lib/payment/load-balancer', () => ({
  getInstanceConfig: vi.fn().mockResolvedValue(null),
}));

vi.mock('@/lib/easy-pay/provider', () => ({
  EasyPayProvider: class {},
}));

import { GET, POST } from '@/app/api/easy-pay/notify/route';

const notification = { orderId: 'order-001', tradeNo: 'T-1', amount: 100, status: 'success', rawData: {} };

function notifyRequest(method: 'GET' | 'POST') {
  const url = new URL('https://pay.example.com/api/easy-pay/notify?out_trade_no=order-001&trade_status=TRADE_SUCCESS');
  return new NextRequest(url, { method, body: method === 'POST' ? 'out_trade_no=order-001' : undefined });
}

describe('easy-pay notify route', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    mockVerifyNotification.mockResolvedValue(notification);
  });

  it('处理成功：200 success', async () => {
    mockHandlePaymentNotify.mockResolvedValue(true);

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(200);
    expect(await res.text()).toBe('success');
    expect(mockHandlePaymentNotify).toHaveBeenCalledWith(notification, 'easy-pay');
  });

  it('处理失败：500 fail，日志带订单号，让按状态判定的平台也重试', async () => {
    mockHandlePaymentNotify.mockResolvedValue(false);

    const res = await POST(notifyRequest('POST'));

    expect(res.status).toBe(500);
    expect(await res.text()).toBe('fail');
    expect(console.error).toHaveBeenCalledWith(expect.stringContaining('order=order-001'));
  });

  it('验签异常：500 fail', async () => {
    mockVerifyNotification.mockRejectedValue(new Error('signature verification failed'));

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(500);
    expect(await res.text()).toBe('fail');
    expect(mockHandlePaymentNotify).not.toHaveBeenCalled();
  });

  it('入账阶段抛异常：500 fail 且日志带订单号', async () => {
    mockHandlePaymentNotify.mockRejectedValue(new Error('db down'));

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(500);
    expect(console.error).toHaveBeenCalledWith(expect.stringContaining('order=order-001'), expect.any(Error));
  });
});
