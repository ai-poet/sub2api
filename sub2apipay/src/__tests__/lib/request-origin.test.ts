import { describe, expect, it } from 'vitest';
import type { NextRequest } from 'next/server';
import { resolveRequestOrigin } from '@/lib/request-origin';

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
