import { describe, expect, it } from 'vitest';
import { describeHttpFailure, describeNetworkFailure, readJsonBody } from '@/lib/api-response';

describe('readJsonBody', () => {
  it('returns the parsed object of a JSON error response', async () => {
    const res = new Response(JSON.stringify({ error: '支付渠道下单失败：签名错误', code: 'PAYMENT_GATEWAY_ERROR' }), {
      status: 424,
    });
    expect(await readJsonBody(res)).toEqual({ error: '支付渠道下单失败：签名错误', code: 'PAYMENT_GATEWAY_ERROR' });
  });

  it('returns null instead of throwing for a CDN HTML page, an empty body or a non-object', async () => {
    expect(await readJsonBody(new Response('<html>502 Bad Gateway</html>', { status: 502 }))).toBeNull();
    expect(await readJsonBody(new Response('', { status: 502 }))).toBeNull();
    expect(await readJsonBody(new Response('[1,2]', { status: 200 }))).toBeNull();
  });
});

describe('describeHttpFailure', () => {
  it('names gateway-class statuses as the service being unreachable', () => {
    expect(describeHttpFailure(502, 'zh')).toBe('支付服务暂时无法访问（HTTP 502），请稍后重试');
    expect(describeHttpFailure(524, 'en')).toBe(
      'The payment service is temporarily unreachable (HTTP 524). Please try again later',
    );
  });

  it('shows the status for anything else', () => {
    expect(describeHttpFailure(429, 'zh')).toBe('请求过于频繁（HTTP 429），请稍后再试');
    expect(describeHttpFailure(413, 'zh')).toBe('请求失败（HTTP 413），请稍后重试');
  });
});

describe('describeNetworkFailure', () => {
  it('keeps the browser’s reason', () => {
    expect(describeNetworkFailure(new TypeError('Failed to fetch'), 'zh')).toBe(
      '网络错误，无法连接到支付服务（Failed to fetch），请检查网络后重试',
    );
    expect(describeNetworkFailure(undefined, 'en')).toBe(
      'Network error: could not reach the payment service. Please check your connection and try again',
    );
  });
});
