import { describe, expect, it } from 'vitest';
import {
  PaymentGatewayResponseError,
  describePaymentGatewayError,
  sanitizeGatewayMessage,
} from '@/lib/order/gateway-error';

describe('describePaymentGatewayError', () => {
  it('passes the channel’s own reason through without the client prefix', () => {
    const error = new Error('EasyPay create payment failed: 该商户未开通 PayNow 支付');
    expect(describePaymentGatewayError(error, 'zh')).toBe('该商户未开通 PayNow 支付');
  });

  it('classifies timeouts and connection failures', () => {
    const timeout = new Error('The operation was aborted due to timeout');
    timeout.name = 'TimeoutError';
    expect(describePaymentGatewayError(timeout, 'zh')).toBe('支付渠道响应超时');
    expect(describePaymentGatewayError(timeout, 'en')).toBe('The payment channel timed out');

    const refused = new TypeError('fetch failed', { cause: { code: 'ECONNREFUSED' } });
    expect(describePaymentGatewayError(refused, 'zh')).toBe('无法连接支付渠道');
  });

  it('names the HTTP status of an unreadable channel answer', () => {
    expect(describePaymentGatewayError(new PaymentGatewayResponseError(502), 'zh')).toBe(
      '支付渠道返回了无法识别的响应（HTTP 502）',
    );
    expect(describePaymentGatewayError(new SyntaxError('Unexpected token <'), 'en')).toBe(
      'The payment channel returned an unreadable response',
    );
  });

  it('falls back to a generic reason for an empty message', () => {
    expect(describePaymentGatewayError(new Error(''), 'zh')).toBe('支付渠道返回未知错误');
  });
});

describe('sanitizeGatewayMessage', () => {
  it('removes addresses, IPs and key-like strings', () => {
    const cleaned = sanitizeGatewayMessage(
      'EasyPay create payment failed: request to https://pay.internal.example/mapi.php from 10.0.0.12:8080 rejected key kyren_live_abcdefghijklmnopqrstuvwxyz0123',
    );
    expect(cleaned).toBe('request to [url] from [ip] rejected key [redacted]');
  });

  it('collapses whitespace and truncates long messages', () => {
    const cleaned = sanitizeGatewayMessage(`签名  错误\n${'很长的说明'.repeat(60)}`);
    expect(cleaned.startsWith('签名 错误 ')).toBe(true);
    expect(cleaned.length).toBeLessThanOrEqual(161);
    expect(cleaned.endsWith('…')).toBe(true);
  });
});
