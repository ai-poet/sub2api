import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mockGetEnv } = vi.hoisted(() => ({
  mockGetEnv: vi.fn(() => ({
    EASY_PAY_PID: '1001',
    EASY_PAY_PKEY: 'test-merchant-secret-key',
    EASY_PAY_API_BASE: 'https://pay.example.com',
    EASY_PAY_NOTIFY_URL: 'https://pay.example.com/api/easy-pay/notify',
    EASY_PAY_RETURN_URL: 'https://pay.example.com/pay/result',
    EASY_PAY_CID: undefined as string | undefined,
    EASY_PAY_CID_ALIPAY: undefined as string | undefined,
    EASY_PAY_CID_WXPAY: undefined as string | undefined,
  })),
}));
vi.mock('@/lib/config', () => ({
  getEnv: mockGetEnv,
}));

const { mockGenerateSign, signCallSnapshots } = vi.hoisted(() => {
  const snapshots: Record<string, string>[][] = [];
  return {
    signCallSnapshots: snapshots,
    mockGenerateSign: vi.fn((...args: unknown[]) => {
      // Snapshot params at call time (before caller mutates the object)
      snapshots.push(args.map((a) => (typeof a === 'object' && a ? { ...a } : a)) as Record<string, string>[]);
      return 'mocked-sign-value';
    }),
  };
});
vi.mock('@/lib/easy-pay/sign', () => ({
  generateSign: mockGenerateSign,
}));

import { createPayment, queryOrder } from '@/lib/easy-pay/client';
import { EASY_PAY_CREATE_ONLY_PARAMS } from '@/lib/easy-pay/notify-params';

describe('EasyPay client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    signCallSnapshots.length = 0;
  });

  describe('createPayment', () => {
    it('should build correct params and POST to mapi.php', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 1,
            trade_no: 'EP20260313000001',
            payurl: 'https://pay.example.com/pay/EP20260313000001',
          }),
          { headers: { 'content-type': 'application/json' } },
        ),
      ) as typeof fetch;

      const result = await createPayment({
        outTradeNo: 'order-001',
        amount: '10.00',
        paymentType: 'alipay',
        clientIp: '127.0.0.1',
        productName: 'Test Product',
      });

      expect(result.code).toBe(1);
      expect(result.trade_no).toBe('EP20260313000001');

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(url).toBe('https://pay.example.com/mapi.php');
      expect(init.method).toBe('POST');
      expect(init.headers).toEqual({ 'Content-Type': 'application/x-www-form-urlencoded' });

      const body = new URLSearchParams(init.body as string);
      expect(body.get('pid')).toBe('1001');
      expect(body.get('type')).toBe('alipay');
      expect(body.get('out_trade_no')).toBe('order-001');
      expect(body.get('money')).toBe('10.00');
      expect(body.get('name')).toBe('Test Product');
      expect(body.get('clientip')).toBe('127.0.0.1');
      expect(body.get('notify_url')).toBe('https://pay.example.com/api/easy-pay/notify');
      expect(body.get('return_url')).toBe('https://pay.example.com/pay/result');
      expect(body.get('sign')).toBe('mocked-sign-value');
      expect(body.get('sign_type')).toBe('MD5');
    });

    it('should call generateSign with correct params (without sign/sign_type)', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 1, trade_no: 'EP001' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await createPayment({
        outTradeNo: 'order-002',
        amount: '20.00',
        paymentType: 'wxpay',
        clientIp: '10.0.0.1',
        productName: 'Another Product',
      });

      expect(mockGenerateSign).toHaveBeenCalledTimes(1);
      const [signParams, pkey] = signCallSnapshots[signCallSnapshots.length - 1] as [Record<string, string>, string];
      expect(pkey).toBe('test-merchant-secret-key');
      // sign and sign_type should not be in the params passed to generateSign
      expect(signParams).not.toHaveProperty('sign');
      expect(signParams).not.toHaveProperty('sign_type');
      expect(signParams.type).toBe('wxpay');
    });

    it('should throw when API returns code !== 1', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: -1, msg: 'Invalid parameter' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await expect(
        createPayment({
          outTradeNo: 'order-003',
          amount: '10.00',
          paymentType: 'alipay',
          clientIp: '127.0.0.1',
          productName: 'Product',
        }),
      ).rejects.toThrow('EasyPay create payment failed: Invalid parameter');
    });

    it('should throw with "unknown error" when msg is absent', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 0 }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await expect(
        createPayment({
          outTradeNo: 'order-004',
          amount: '10.00',
          paymentType: 'alipay',
          clientIp: '127.0.0.1',
          productName: 'Product',
        }),
      ).rejects.toThrow('EasyPay create payment failed: unknown error');
    });

    it('should not include cid when no CID env vars are set', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 1, trade_no: 'EP001' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await createPayment({
        outTradeNo: 'order-005',
        amount: '10.00',
        paymentType: 'alipay',
        clientIp: '127.0.0.1',
        productName: 'Product',
      });

      const [, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const body = new URLSearchParams(init.body as string);
      expect(body.has('cid')).toBe(false);
    });
  });

  describe('createPayment CID routing', () => {
    it('should use EASY_PAY_CID_ALIPAY for alipay payment type', async () => {
      mockGetEnv.mockReturnValue({
        EASY_PAY_PID: '1001',
        EASY_PAY_PKEY: 'test-merchant-secret-key',
        EASY_PAY_API_BASE: 'https://pay.example.com',
        EASY_PAY_NOTIFY_URL: 'https://pay.example.com/api/easy-pay/notify',
        EASY_PAY_RETURN_URL: 'https://pay.example.com/pay/result',
        EASY_PAY_CID: '100',
        EASY_PAY_CID_ALIPAY: '200',
        EASY_PAY_CID_WXPAY: '300',
      });

      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 1, trade_no: 'EP001' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await createPayment({
        outTradeNo: 'order-cid-1',
        amount: '10.00',
        paymentType: 'alipay',
        clientIp: '127.0.0.1',
        productName: 'Product',
      });

      const [, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const body = new URLSearchParams(init.body as string);
      expect(body.get('cid')).toBe('200');
    });

    it('should use EASY_PAY_CID_WXPAY for wxpay payment type', async () => {
      mockGetEnv.mockReturnValue({
        EASY_PAY_PID: '1001',
        EASY_PAY_PKEY: 'test-merchant-secret-key',
        EASY_PAY_API_BASE: 'https://pay.example.com',
        EASY_PAY_NOTIFY_URL: 'https://pay.example.com/api/easy-pay/notify',
        EASY_PAY_RETURN_URL: 'https://pay.example.com/pay/result',
        EASY_PAY_CID: '100',
        EASY_PAY_CID_ALIPAY: '200',
        EASY_PAY_CID_WXPAY: '300',
      });

      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 1, trade_no: 'EP001' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await createPayment({
        outTradeNo: 'order-cid-2',
        amount: '10.00',
        paymentType: 'wxpay',
        clientIp: '127.0.0.1',
        productName: 'Product',
      });

      const [, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const body = new URLSearchParams(init.body as string);
      expect(body.get('cid')).toBe('300');
    });

    it('should fall back to EASY_PAY_CID when channel-specific CID is not set', async () => {
      mockGetEnv.mockReturnValue({
        EASY_PAY_PID: '1001',
        EASY_PAY_PKEY: 'test-merchant-secret-key',
        EASY_PAY_API_BASE: 'https://pay.example.com',
        EASY_PAY_NOTIFY_URL: 'https://pay.example.com/api/easy-pay/notify',
        EASY_PAY_RETURN_URL: 'https://pay.example.com/pay/result',
        EASY_PAY_CID: '100',
        EASY_PAY_CID_ALIPAY: undefined,
        EASY_PAY_CID_WXPAY: undefined,
      });

      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 1, trade_no: 'EP001' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await createPayment({
        outTradeNo: 'order-cid-3',
        amount: '10.00',
        paymentType: 'alipay',
        clientIp: '127.0.0.1',
        productName: 'Product',
      });

      const [, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const body = new URLSearchParams(init.body as string);
      expect(body.get('cid')).toBe('100');
    });
  });

  // 回调验签拒绝下单专有参数（notify-params.ts），前提是下单签名串排序后总以其中之一开头
  describe('createPayment invariants relied on by the notify guard', () => {
    function mockEnv(cid: string | undefined) {
      mockGetEnv.mockReturnValue({
        EASY_PAY_PID: '1001',
        EASY_PAY_PKEY: 'test-merchant-secret-key',
        EASY_PAY_API_BASE: 'https://pay.example.com',
        EASY_PAY_NOTIFY_URL: 'https://pay.example.com/api/easy-pay/notify',
        EASY_PAY_RETURN_URL: 'https://pay.example.com/pay/result',
        EASY_PAY_CID: cid,
        EASY_PAY_CID_ALIPAY: undefined,
        EASY_PAY_CID_WXPAY: undefined,
      });
      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 1, trade_no: 'EP001' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;
    }

    it('sends a non-empty clientip even when the caller passes none', async () => {
      mockEnv(undefined);

      await createPayment({
        outTradeNo: 'order-inv-1',
        amount: '10.00',
        paymentType: 'alipay',
        clientIp: '',
        productName: 'Product',
      });

      const [, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(new URLSearchParams(init.body as string).get('clientip')).toBe('127.0.0.1');
    });

    it.each([
      ['without cid, empty client ip', undefined, ''],
      ['with cid, empty client ip', '200', ''],
      ['without cid, real client ip', undefined, '203.0.113.7'],
      ['with cid, real client ip', '200', '203.0.113.7'],
    ])('the sorted signed params start with a param the notify guard rejects (%s)', async (_label, cid, clientIp) => {
      mockEnv(cid);

      await createPayment({
        outTradeNo: 'order-inv-2',
        amount: '10.00',
        paymentType: 'alipay',
        clientIp,
        productName: 'Product',
        isMobile: true,
      });

      const [signParams] = signCallSnapshots[signCallSnapshots.length - 1] as [Record<string, string>, string];
      // 与 generateSign 相同的过滤与排序
      const [firstKey] = Object.entries(signParams)
        .filter(([key, value]) => key !== 'sign' && key !== 'sign_type' && value !== '' && value != null)
        .map(([key]) => key)
        .sort((a, b) => a.localeCompare(b));
      expect(EASY_PAY_CREATE_ONLY_PARAMS.has(firstKey)).toBe(true);
    });
  });

  describe('queryOrder', () => {
    it('should call GET api.php with correct query parameters', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 1,
            trade_no: 'EP20260313000001',
            out_trade_no: 'order-001',
            type: 'alipay',
            pid: '1001',
            addtime: '2026-03-13 10:00:00',
            endtime: '2026-03-13 10:01:00',
            name: 'Test Product',
            money: '10.00',
            status: 1,
          }),
          { headers: { 'content-type': 'application/json' } },
        ),
      ) as typeof fetch;

      const result = await queryOrder('order-001');

      expect(result.code).toBe(1);
      expect(result.trade_no).toBe('EP20260313000001');
      expect(result.status).toBe(1);

      expect(global.fetch).toHaveBeenCalledTimes(1);
      const [url, init] = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const requestUrl = new URL(url as string);
      expect(requestUrl.origin + requestUrl.pathname).toBe('https://pay.example.com/api.php');
      expect(init.method).toBe('GET');
      expect(requestUrl.searchParams.get('act')).toBe('order');
      expect(requestUrl.searchParams.get('pid')).toBe('1001');
      expect(requestUrl.searchParams.get('key')).toBe('test-merchant-secret-key');
      expect(requestUrl.searchParams.get('out_trade_no')).toBe('order-001');
    });

    it('should throw when API returns code !== 1', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: -1, msg: 'Order not found' }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await expect(queryOrder('nonexistent-order')).rejects.toThrow('EasyPay query order failed: Order not found');
    });

    it('should throw with "unknown error" when msg is absent', async () => {
      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 0 }), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      await expect(queryOrder('order-err')).rejects.toThrow('EasyPay query order failed: unknown error');
    });

    it('should parse all response fields correctly', async () => {
      const mockResponse = {
        code: 1,
        trade_no: 'EP20260313000002',
        out_trade_no: 'order-010',
        type: 'wxpay',
        pid: '1001',
        addtime: '2026-03-13 12:00:00',
        endtime: '2026-03-13 12:05:00',
        name: 'Premium Plan',
        money: '99.00',
        status: 1,
        param: 'custom-param',
        buyer: 'buyer@example.com',
      };

      global.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify(mockResponse), {
          headers: { 'content-type': 'application/json' },
        }),
      ) as typeof fetch;

      const result = await queryOrder('order-010');

      expect(result).toEqual(mockResponse);
    });
  });
});
