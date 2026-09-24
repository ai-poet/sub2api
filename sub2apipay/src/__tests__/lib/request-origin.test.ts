import { describe, expect, it } from 'vitest';
import type { NextRequest } from 'next/server';
import { resolveAbsoluteUrl, resolveRequestOrigin } from '@/lib/request-origin';

function makeRequest(input: {
  url: string;
  pathname: string;
  headers?: Record<string, string>;
}): NextRequest {
  const headers = new Headers(input.headers);
  return {
    headers,
    nextUrl: {
      origin: input.url,
      protocol: new URL(input.url).protocol,
      host: new URL(input.url).host,
      pathname: input.pathname,
    },
  } as unknown as NextRequest;
}

describe('request-origin', () => {
  it('infers /pay base path from proxied pathname when forwarded prefix is absent', () => {
    const request = makeRequest({
      url: 'https://ai-coding.cyberspirit.io',
      pathname: '/pay/api/orders',
      headers: {
        host: 'ai-coding.cyberspirit.io',
        'x-forwarded-proto': 'https',
      },
    });

    expect(resolveRequestOrigin(request)).toBe('https://ai-coding.cyberspirit.io/pay');
  });

  it('prefers x-forwarded-prefix when provided', () => {
    const request = makeRequest({
      url: 'https://ai-coding.cyberspirit.io',
      pathname: '/api/orders',
      headers: {
        host: 'ai-coding.cyberspirit.io',
        'x-forwarded-proto': 'https',
        'x-forwarded-prefix': '/pay',
      },
    });

    expect(resolveRequestOrigin(request)).toBe('https://ai-coding.cyberspirit.io/pay');
  });
});

describe('resolveAbsoluteUrl', () => {
  const appUrl = 'https://gw.example.com/pay';

  it('resolves the root-relative alipay short link against the app origin', () => {
    expect(resolveAbsoluteUrl('/pay/order_1', appUrl)).toBe('https://gw.example.com/pay/order_1');
  });

  it('keeps absolute URLs, app schemes and protocol-relative values unchanged', () => {
    expect(resolveAbsoluteUrl('https://openapi.alipay.com/x?y=1', appUrl)).toBe('https://openapi.alipay.com/x?y=1');
    expect(resolveAbsoluteUrl('weixin://wxpay/bizpayurl?pr=abc', appUrl)).toBe('weixin://wxpay/bizpayurl?pr=abc');
    expect(resolveAbsoluteUrl('//cdn.example.com/a', appUrl)).toBe('//cdn.example.com/a');
  });

  it('passes empty values through', () => {
    expect(resolveAbsoluteUrl(undefined, appUrl)).toBeUndefined();
    expect(resolveAbsoluteUrl(null, appUrl)).toBeNull();
    expect(resolveAbsoluteUrl('', appUrl)).toBe('');
  });
});
