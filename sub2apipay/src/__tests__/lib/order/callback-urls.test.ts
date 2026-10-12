import { describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/config', () => ({
  getEnv: () => ({ JWT_SECRET: 'test-jwt-secret-123456', ADMIN_TOKEN: 'test-admin-token' }),
}));

import { resolveEasyPayCallbackUrls } from '@/lib/order/callback-urls';
import { buildOrderResultUrl } from '@/lib/order/status-access';

const APP_URL = 'http://cheaprouter.cc/pay';
/** 跳回地址去掉带时间戳的访问令牌后再比较。 */
function withoutToken(url: string): string {
  const parsed = new URL(url);
  expect(parsed.searchParams.get('access_token')).toBeTruthy();
  parsed.searchParams.delete('access_token');
  return parsed.toString();
}

const base = { appUrl: APP_URL, instanceId: 'inst-1', orderId: 'order-1', userId: 42 };

describe('resolveEasyPayCallbackUrls', () => {
  it('keeps the request-origin URLs when the instance has no notify URL', () => {
    const urls = resolveEasyPayCallbackUrls({ ...base, instanceConfig: { pid: '1001', pkey: 'k' } });
    expect(urls.notifyUrl).toBe('http://cheaprouter.cc/pay/api/easy-pay/notify?inst=inst-1');
    expect(withoutToken(urls.returnUrl)).toBe(withoutToken(buildOrderResultUrl(APP_URL, 'order-1', 42)));
    expect(urls.fromInstanceConfig).toBe(false);
  });

  it('keeps the request-origin URLs without any instance (env-configured provider)', () => {
    const urls = resolveEasyPayCallbackUrls({ appUrl: APP_URL, orderId: 'order-1', userId: 42 });
    expect(urls.notifyUrl).toBe('http://cheaprouter.cc/pay/api/easy-pay/notify');
    expect(urls.fromInstanceConfig).toBe(false);
  });

  it('ignores an http notify URL, so existing instances are unaffected', () => {
    const urls = resolveEasyPayCallbackUrls({
      ...base,
      instanceConfig: { notifyUrl: 'http://cheaprouter.cc/pay/api/easy-pay/notify' },
    });
    expect(urls.fromInstanceConfig).toBe(false);
    expect(urls.notifyUrl).toBe('http://cheaprouter.cc/pay/api/easy-pay/notify?inst=inst-1');
  });

  it('uses a full https notify URL, replacing its query with the routing param', () => {
    const urls = resolveEasyPayCallbackUrls({
      ...base,
      instanceConfig: { notifyUrl: 'https://cheaprouter.cc/pay/api/easy-pay/notify/?inst=old&foo=1' },
    });
    expect(urls.fromInstanceConfig).toBe(true);
    expect(urls.notifyUrl).toBe('https://cheaprouter.cc/pay/api/easy-pay/notify?inst=inst-1');
  });

  it('completes a bare https domain with the request base path and callback path', () => {
    const urls = resolveEasyPayCallbackUrls({ ...base, instanceConfig: { notifyUrl: 'https://cheaprouter.cc' } });
    expect(urls.notifyUrl).toBe('https://cheaprouter.cc/pay/api/easy-pay/notify?inst=inst-1');
    expect(new URL(urls.returnUrl).origin).toBe('https://cheaprouter.cc');
    expect(new URL(urls.returnUrl).pathname).toBe('/pay/result');
    expect(new URL(urls.returnUrl).searchParams.get('order_id')).toBe('order-1');
  });

  it('takes the return origin from the instance return URL when it is https', () => {
    const urls = resolveEasyPayCallbackUrls({
      ...base,
      instanceConfig: {
        notifyUrl: 'https://cheaprouter.cc/pay/api/easy-pay/notify',
        returnUrl: 'https://cdn.cheaprouter.cc/anything',
      },
    });
    expect(new URL(urls.returnUrl).origin).toBe('https://cdn.cheaprouter.cc');
    expect(withoutToken(urls.returnUrl)).toBe('https://cdn.cheaprouter.cc/pay/result?order_id=order-1');
  });

  it('ignores a notify URL that points somewhere other than the EasyPay callback', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const urls = resolveEasyPayCallbackUrls({
      ...base,
      instanceConfig: { notifyUrl: 'https://cheaprouter.cc/pay/result' },
    });
    expect(urls.fromInstanceConfig).toBe(false);
    expect(urls.notifyUrl).toBe('http://cheaprouter.cc/pay/api/easy-pay/notify?inst=inst-1');
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });
});
