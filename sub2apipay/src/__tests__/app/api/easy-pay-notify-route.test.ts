import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';
import { EasyPayNotifyRejectedError } from '@/lib/easy-pay/notify-errors';

const { mockAfter, afterCallbacks } = vi.hoisted(() => {
  const afterCallbacks: Array<() => unknown> = [];
  return {
    afterCallbacks,
    mockAfter: vi.fn((task: () => unknown) => {
      afterCallbacks.push(task);
    }),
  };
});

vi.mock('next/server', async (importOriginal) => ({
  ...(await importOriginal<typeof import('next/server')>()),
  after: mockAfter,
}));

const mockHandlePaymentNotify = vi.fn();
const mockVerifyNotification = vi.fn();
const mockObserveEasyPayNotify = vi.fn();
const mockRecordBlockedEasyPayNotify = vi.fn();
const mockVerifyEasyPayCreditUpstream = vi.fn();

vi.mock('@/lib/easy-pay/notify-audit', () => ({
  observeEasyPayNotify: (...args: unknown[]) => mockObserveEasyPayNotify(...args),
  recordBlockedEasyPayNotify: (...args: unknown[]) => mockRecordBlockedEasyPayNotify(...args),
}));

vi.mock('@/lib/easy-pay/upstream-verify', () => ({
  verifyEasyPayCreditUpstream: (...args: unknown[]) => mockVerifyEasyPayCreditUpstream(...args),
}));

vi.mock('@/lib/order/service', () => ({
  handlePaymentNotify: (...args: unknown[]) => mockHandlePaymentNotify(...args),
}));

const provider = {
  name: 'easy-pay',
  providerKey: 'easypay',
  verifyNotification: (...args: unknown[]) => mockVerifyNotification(...args),
};

vi.mock('@/lib/payment', () => ({
  ensureDBProviders: vi.fn().mockResolvedValue(undefined),
  paymentRegistry: {
    getProvider: () => provider,
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

async function runAfterCallbacks() {
  await Promise.all(afterCallbacks.map((task) => task()));
}

describe('easy-pay notify route', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    afterCallbacks.length = 0;
    vi.spyOn(console, 'error').mockImplementation(() => {});
    mockVerifyNotification.mockResolvedValue(notification);
    mockObserveEasyPayNotify.mockResolvedValue(undefined);
    mockRecordBlockedEasyPayNotify.mockResolvedValue(undefined);
    mockVerifyEasyPayCreditUpstream.mockResolvedValue(undefined);
  });

  it('验签通过后、入账之前做旁路观测', async () => {
    mockHandlePaymentNotify.mockResolvedValue(true);

    await GET(notifyRequest('GET'));

    expect(mockObserveEasyPayNotify).toHaveBeenCalledWith(notification, 'easy-pay');
    expect(mockObserveEasyPayNotify.mock.invocationCallOrder[0]).toBeLessThan(
      mockHandlePaymentNotify.mock.invocationCallOrder[0],
    );
  });

  it('旁路观测出错不影响入账与响应', async () => {
    mockObserveEasyPayNotify.mockRejectedValue(new Error('db down'));
    mockHandlePaymentNotify.mockResolvedValue(true);

    const res = await POST(notifyRequest('POST'));

    expect(res.status).toBe(200);
    expect(await res.text()).toBe('success');
    expect(mockHandlePaymentNotify).toHaveBeenCalledWith(notification, 'easy-pay');
  });

  it('处理成功：200 success', async () => {
    mockHandlePaymentNotify.mockResolvedValue(true);

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(200);
    expect(await res.text()).toBe('success');
    expect(mockHandlePaymentNotify).toHaveBeenCalledWith(notification, 'easy-pay');
  });

  it('入账成功后在响应发出之后向平台复核', async () => {
    mockHandlePaymentNotify.mockResolvedValue(true);

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(200);
    expect(mockAfter).toHaveBeenCalledTimes(1);
    expect(mockVerifyEasyPayCreditUpstream).not.toHaveBeenCalled();

    await runAfterCallbacks();

    expect(mockVerifyEasyPayCreditUpstream).toHaveBeenCalledWith(notification, provider);
    expect(mockRecordBlockedEasyPayNotify).not.toHaveBeenCalled();
  });

  it('复核出错被吞掉，不会冒泡', async () => {
    mockHandlePaymentNotify.mockResolvedValue(true);
    mockVerifyEasyPayCreditUpstream.mockRejectedValue(new Error('boom'));

    await GET(notifyRequest('GET'));

    await expect(runAfterCallbacks()).resolves.toBeUndefined();
    expect(mockVerifyEasyPayCreditUpstream).toHaveBeenCalledTimes(1);
  });

  it('不在请求上下文里（after 抛错）时退回为直接在后台复核', async () => {
    mockHandlePaymentNotify.mockResolvedValue(true);
    mockAfter.mockImplementationOnce(() => {
      throw new Error('`after` was called outside a request scope');
    });

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(200);
    expect(mockVerifyEasyPayCreditUpstream).toHaveBeenCalledWith(notification, provider);
  });

  it('入账失败或非成功通知：不复核', async () => {
    mockHandlePaymentNotify.mockResolvedValue(false);
    await POST(notifyRequest('POST'));

    mockHandlePaymentNotify.mockResolvedValue(true);
    mockVerifyNotification.mockResolvedValue({ ...notification, status: 'failed' });
    await GET(notifyRequest('GET'));

    await runAfterCallbacks();
    expect(mockVerifyEasyPayCreditUpstream).not.toHaveBeenCalled();
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
    expect(mockObserveEasyPayNotify).not.toHaveBeenCalled();
  });

  it('验签被拒：响应之后把拦截记到风控面板', async () => {
    const rejected = new EasyPayNotifyRejectedError('rejected', 'unexpected_param', 'order-001', 'clientip');
    mockVerifyNotification.mockRejectedValue(rejected);

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(500);
    expect(mockRecordBlockedEasyPayNotify).not.toHaveBeenCalled();

    await runAfterCallbacks();

    expect(mockRecordBlockedEasyPayNotify).toHaveBeenCalledWith(rejected, 'easy-pay');
    expect(mockVerifyEasyPayCreditUpstream).not.toHaveBeenCalled();
  });

  it('入账阶段抛异常：500 fail 且日志带订单号，不当作可疑回调', async () => {
    mockHandlePaymentNotify.mockRejectedValue(new Error('db down'));

    const res = await GET(notifyRequest('GET'));

    expect(res.status).toBe(500);
    expect(console.error).toHaveBeenCalledWith(expect.stringContaining('order=order-001'), expect.any(Error));
    await runAfterCallbacks();
    expect(mockRecordBlockedEasyPayNotify).not.toHaveBeenCalled();
  });
});
