import { describe, expect, it } from 'vitest';
import { getStablecoinPaymentNotice } from '@/lib/stablecoin-notice';

describe('getStablecoinPaymentNotice', () => {
  it.each([
    ['usdt.plasma', 'USDT', 'Plasma'],
    ['usdt.polygon', 'USDT', 'Polygon'],
    ['usdc.solana', 'USDC', 'Solana'],
  ])('%s names the token, direct transfer and the network', (type, token, network) => {
    const zh = getStablecoinPaymentNotice(type, 'zh');
    expect(zh?.title).toBe(`${token} 支付须知`);
    expect(zh?.points[0].text).toContain('Regular Transfer');
    expect(zh?.points[0].text).toContain(token);
    expect(zh?.points[1].text).toContain(`${network} 网络`);
    expect(zh?.warning).toBe('否则可能不会自动到账。');

    const en = getStablecoinPaymentNotice(type, 'en');
    expect(en?.title).toBe(`Before you pay with ${token}`);
    expect(en?.points[0].text).toContain('regular transfer');
    expect(en?.points[1].text).toContain(`${network} network`);
    expect(en?.warning).toContain('may not be credited automatically');
  });

  it.each(['alipay', 'wxpay', 'alipay_direct', 'wxpay_direct', 'stripe', 'bank', ''])(
    'returns null for %s so no dialog is shown',
    (type) => {
      expect(getStablecoinPaymentNotice(type, 'zh')).toBeNull();
      expect(getStablecoinPaymentNotice(type, 'en')).toBeNull();
    },
  );
});
